package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/planning"
	"github.com/chuma-beep/mathua/internal/pytrack"
	"github.com/chuma-beep/mathua/internal/quiz"
	"github.com/chuma-beep/mathua/internal/storage"
)

// numGen is a generated topic: every question is new and grades as "1".
type numGen struct {
	id string
	n  atomic.Int64
}

func (g *numGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return generator.Problem{Question: fmt.Sprintf("%s question %d", g.id, g.n.Add(1)), Answer: "1"}
}

// testsFixture is a small corpus with four courses:
//
//   - plan.mixed: two units of own topics resting on a generated chain
//     g1..g4. Unit A is question-bank topics (b1..b4 multiple choice mixed
//     with recall cards and a code task; b5 recall cards only), Unit B is
//     generated topics n1..n4.
//   - plan.recall: nine chained topics with recall cards only.
//   - plan.flat: eight unconnected multiple-choice bank topics.
//   - plan.other: one generated topic elsewhere.
func testsFixture(t *testing.T) (*Engine, *storage.SQLiteStore) {
	t.Helper()
	th := concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 30}
	var raw []concepts.Concept
	gen := func(id, sub string, pre ...string) {
		raw = append(raw, concepts.Concept{ID: id, Label: "Topic " + id, Domain: "algebra", Subdomain: sub, GradingType: "numeric", Prerequisites: pre, MasteryThreshold: th})
	}
	bank := func(id, sub string, pre ...string) {
		raw = append(raw, concepts.Concept{ID: id, Label: "Topic " + id, Domain: "reading", Subdomain: sub, GradingType: "bank", Prerequisites: pre, MasteryThreshold: th})
	}
	gen("g1", "s-g")
	gen("g2", "s-g", "g1")
	gen("g3", "s-g", "g2")
	gen("g4", "s-g", "g3")
	bank("b1", "s-b1", "g4")
	bank("b2", "s-b2", "b1")
	bank("b3", "s-b3", "b2")
	bank("b4", "s-b4", "b3")
	bank("b5", "s-b5", "b4")
	gen("n1", "s-n1", "b4")
	gen("n2", "s-n2", "n1")
	gen("n3", "s-n3", "n2")
	gen("n4", "s-n4", "n3")
	for i := 1; i <= 9; i++ {
		if i == 1 {
			bank("r1", "s-r")
		} else {
			bank(fmt.Sprintf("r%d", i), "s-r", fmt.Sprintf("r%d", i-1))
		}
	}
	for i := 1; i <= 8; i++ {
		bank(fmt.Sprintf("f%d", i), fmt.Sprintf("s-f%d", i))
	}
	gen("o1", "s-o")
	d, err := concepts.Build(raw)
	if err != nil {
		t.Fatal(err)
	}

	// The bank: real pytrack generators over a real bank file.
	type item map[string]interface{}
	mc := func(id string) item {
		return item{"id": id, "kind": "mc", "level": 1, "prompt": "Pick " + id, "correct": "yes " + id, "wrong": []string{"no", "maybe"}, "explain": "Because."}
	}
	recall := func(id string) item {
		return item{"id": id, "kind": "recall", "level": 1, "prompt": "Recall " + id, "back": "The back.", "explain": ""}
	}
	code := func(id string) item {
		return item{"id": id, "kind": "code", "level": 1, "prompt": "Write " + id, "solution": "x = 1", "tests": []item{{"name": "t", "code": "assert x == 1"}}, "explain": "Set x."}
	}
	var bc []item
	for _, c := range raw {
		if c.GradingType != "bank" {
			continue
		}
		var items []item
		switch {
		case c.ID == "b5" || strings.HasPrefix(c.ID, "r"):
			items = []item{recall(c.ID + "a"), recall(c.ID + "b")}
		case strings.HasPrefix(c.ID, "b"):
			items = []item{mc(c.ID + "m"), recall(c.ID + "r1"), recall(c.ID + "r2"), recall(c.ID + "r3"), code(c.ID + "c")}
		default:
			items = []item{mc(c.ID + "m")}
		}
		bc = append(bc, item{"id": c.ID, "book": item{"title": "Handbook"}, "intro": "", "items": items})
	}
	dir := t.TempDir()
	bankDir := filepath.Join(dir, "bank")
	os.MkdirAll(bankDir, 0o755)
	js, _ := json.Marshal(item{"chapter": "1", "title": "One", "concepts": bc})
	if err := os.WriteFile(filepath.Join(bankDir, "one.json"), js, 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := pytrack.LoadBank(bankDir)
	if err != nil {
		t.Fatal(err)
	}
	reg := generator.NewRegistry()
	if err := (&pytrack.Track{Bank: b, Runner: pytrack.NewRunner()}).Register(reg); err != nil {
		t.Fatal(err)
	}
	for _, c := range raw {
		if c.GradingType != "bank" {
			reg.Register(c.ID, &numGen{id: c.ID})
		}
	}

	courses := `[
	 {"id":"plan.mixed","name":"Mixed course","plan":["b1","b2","b3","b4","b5","n1","n2","n3","n4"],
	  "modules":[{"title":"Unit A","topics":["b1","b2","b3","b4","b5"]},{"title":"Unit B","topics":["n1","n2","n3","n4"]}]},
	 {"id":"plan.recall","name":"Recall course","plan":["r1","r2","r3","r4","r5","r6","r7","r8","r9"]},
	 {"id":"plan.flat","name":"Flat course","plan":["f1","f2","f3","f4","f5","f6","f7","f8"]},
	 {"id":"plan.other","name":"Other course","plan":["o1"]}
	]`
	cp := filepath.Join(dir, "courses.json")
	if err := os.WriteFile(cp, []byte(courses), 0o644); err != nil {
		t.Fatal(err)
	}
	pl, err := planning.Load(cp, d)
	if err != nil {
		t.Fatal(err)
	}
	store, err := storage.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	return New(store, d, reg, nil, pl), store
}

func TestCourseTests_Viability(t *testing.T) {
	e, _ := testsFixture(t)
	for _, tc := range []struct {
		course            string
		placement, review bool
		gradable          int
	}{
		// b5 has recall cards only, so the course has eight gradable topics
		// of its own, not nine; with its generated chain it offers both.
		{"plan.mixed", true, true, 8},
		// Recall cards are graded by the learner: nothing to test.
		{"plan.recall", false, false, 0},
		// A multiple-choice bank can be reviewed, but with no prerequisite
		// edges every topic is open already, so placement has no purpose.
		{"plan.flat", false, true, 8},
		// Too small for either.
		{"plan.other", false, false, 1},
	} {
		info, err := e.CourseTests("", tc.course)
		if err != nil {
			t.Fatal(err)
		}
		if info.Placement != tc.placement || info.Review != tc.review || info.GradableTopics != tc.gradable {
			t.Errorf("%s: got placement=%v review=%v gradable=%d, want %v %v %d", tc.course, info.Placement, info.Review, info.GradableTopics, tc.placement, tc.review, tc.gradable)
		}
	}
}

// placementRun takes a course placement test, answering with knows, and
// returns every served problem by topic.
func placementRun(t *testing.T, e *Engine, studentID, course string, knows func(string) bool) (served map[string][]generator.Problem, asked int) {
	t.Helper()
	session, q, err := e.StartCourseDiagnostic(studentID, course)
	if err != nil {
		t.Fatal(err)
	}
	session.StudentID = studentID
	served = map[string][]generator.Problem{}
	for q != nil && asked < 100 {
		cid := session.LastConceptID
		served[cid] = append(served[cid], *session.LastProblem)
		asked++
		e.SubmitDiagnosticAnswerTimed(session, cid, knows(cid), 5, 30)
		if e.IsDiagnosticComplete(session) {
			break
		}
		prob, next, err := e.NextDiagnosticQuestion(session)
		if err != nil {
			t.Fatal(err)
		}
		if prob == nil || next == "" {
			break
		}
	}
	if _, err := e.ApplyCoursePlacement(studentID, session); err != nil {
		t.Fatal(err)
	}
	return served, asked
}

// A course placement test asks the course's own topics, question-bank ones
// included, not only the foundations under them; it is bounded; and it
// never lowers progress: it never asks a topic the learner already has
// progress on, and never un-masters or un-places one.
func TestCoursePlacement_CoversCourseTopicsAndNeverLowersProgress(t *testing.T) {
	e, store := testsFixture(t)
	st, err := e.CreateStudent("p")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	keep := []*storage.ConceptProgress{
		{StudentID: st.ID, ConceptID: "n2", Status: "MASTERED", Attempts: 7, Streak: 5, WeaknessScore: 0.05, MasteredAt: &now, LastAttempted: &now, SM2EFactor: 2.6},
		{StudentID: st.ID, ConceptID: "b2", Status: "PRACTICING", Attempts: 3, Streak: 2, WeaknessScore: 0.2, LastAttempted: &now, SM2EFactor: 2.5},
		{StudentID: st.ID, ConceptID: "g2", Status: "LEARNING", PlacementSeeded: true, WeaknessScore: 0.3, LastAttempted: &now, SM2EFactor: 2.5},
	}
	for _, p := range keep {
		if err := store.UpsertProgress(p); err != nil {
			t.Fatal(err)
		}
	}

	// Worst case for "never lower": every answer wrong.
	served, asked := placementRun(t, e, st.ID, "plan.mixed", func(string) bool { return false })
	if asked == 0 || asked > CoursePlacementMaxQuestions {
		t.Fatalf("asked %d questions", asked)
	}
	ownAsked := 0
	for _, id := range []string{"b1", "b3", "b4", "n1", "n3", "n4"} {
		if len(served[id]) > 0 {
			ownAsked++
		}
	}
	if ownAsked == 0 {
		t.Errorf("no course topic asked; served %v", keys(served))
	}
	for _, p := range keep {
		if len(served[p.ConceptID]) > 0 {
			t.Errorf("%s has progress and was asked anyway", p.ConceptID)
		}
		got, err := store.GetProgress(st.ID, p.ConceptID)
		if err != nil || got == nil {
			t.Fatalf("%s: %v", p.ConceptID, err)
		}
		if got.Status != p.Status || got.Attempts != p.Attempts || got.Streak != p.Streak || got.PlacementSeeded != p.PlacementSeeded || got.WeaknessScore != p.WeaknessScore {
			t.Errorf("%s changed: %+v", p.ConceptID, got)
		}
	}

	// A learner who knows everything is placed into the course itself.
	st2, _ := e.CreateStudent("q")
	served, _ = placementRun(t, e, st2.ID, "plan.mixed", func(string) bool { return true })
	placedOwn := 0
	for _, id := range []string{"b1", "b2", "b3", "b4", "n1", "n2", "n3"} {
		if p, _ := store.GetProgress(st2.ID, id); p != nil && p.PlacementSeeded {
			placedOwn++
		}
	}
	if placedOwn == 0 {
		t.Errorf("a strong learner was placed on no course topic; served %v", keys(served))
	}
}

// Placement serves question-bank multiple choice, never a recall card or a
// code task, and never asks a topic that has only recall cards.
func TestCoursePlacement_BankMultipleChoiceYesRecallNo(t *testing.T) {
	e, _ := testsFixture(t)
	bankAsked := 0
	for run := 0; run < 6; run++ {
		st, _ := e.CreateStudent(fmt.Sprintf("r%d", run))
		served, _ := placementRun(t, e, st.ID, "plan.mixed", func(id string) bool { return run%2 == 0 || id[0] == 'g' })
		if len(served["b5"]) > 0 {
			t.Fatalf("b5 has recall cards only and was asked")
		}
		for id, ps := range served {
			for _, p := range ps {
				if strings.HasPrefix(p.Answer, "recall:") || strings.HasPrefix(p.Answer, "code:") {
					t.Fatalf("%s served %q", id, p.Answer)
				}
				if strings.HasPrefix(p.Answer, "mc:") {
					bankAsked++
				}
			}
		}
	}
	if bankAsked == 0 {
		t.Fatal("no bank multiple-choice item was ever asked")
	}
}

// A review test covers only topics that course has taught the learner:
// studied (not placed), its own (not prerequisites, not another course's),
// gradable (not recall-only), spread across its sections.
func TestCourseReview_OnlyStudiedTopicsOfTheCourse(t *testing.T) {
	e, store := testsFixture(t)
	st, _ := e.CreateStudent("rv")
	now := time.Now().UTC()
	old := now.Add(-48 * time.Hour)
	for _, p := range []*storage.ConceptProgress{
		{StudentID: st.ID, ConceptID: "b1", Status: "LEARNING", Attempts: 2, WeaknessScore: 0.6, LastAttempted: &now},
		{StudentID: st.ID, ConceptID: "b3", Status: "MASTERED", Attempts: 6, WeaknessScore: 0.1, LastAttempted: &old},
		{StudentID: st.ID, ConceptID: "b5", Status: "LEARNING", Attempts: 4, WeaknessScore: 0.9, LastAttempted: &now}, // recall only
		{StudentID: st.ID, ConceptID: "n1", Status: "LEARNING", Attempts: 3, WeaknessScore: 0.5, LastAttempted: &now},
		{StudentID: st.ID, ConceptID: "n2", Status: "LEARNING", PlacementSeeded: true},                                // placed only
		{StudentID: st.ID, ConceptID: "g3", Status: "LEARNING", Attempts: 5, WeaknessScore: 0.9, LastAttempted: &now}, // prerequisite
		{StudentID: st.ID, ConceptID: "o1", Status: "LEARNING", Attempts: 5, WeaknessScore: 0.9, LastAttempted: &now}, // other course
	} {
		if err := store.UpsertProgress(p); err != nil {
			t.Fatal(err)
		}
	}
	got, err := e.CourseReviewConcepts(st.ID, "plan.mixed", false)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, c := range got {
		ids[c.ID] = true
	}
	if len(got) != 3 || !ids["b1"] || !ids["b3"] || !ids["n1"] {
		t.Fatalf("want b1, b3, n1; got %v", ids)
	}
	// The weakest topic of each section leads, alternating sections.
	if got[0].ID != "b1" || got[1].ID != "n1" {
		t.Errorf("order = %s, %s, %s", got[0].ID, got[1].ID, got[2].ID)
	}

	info, _ := e.CourseTests(st.ID, "plan.mixed")
	if info.ReviewTopics != 3 || info.ReviewQuestions != 3 {
		t.Errorf("review counts = %+v", info)
	}
}

// The full review covers more topics than the ordinary one, each capped.
func TestCourseReview_Limits(t *testing.T) {
	e, store := testsFixture(t)
	st, _ := e.CreateStudent("lim")
	now := time.Now().UTC()
	for i := 1; i <= 8; i++ {
		if err := store.UpsertProgress(&storage.ConceptProgress{StudentID: st.ID, ConceptID: fmt.Sprintf("f%d", i), Status: "LEARNING", Attempts: 1, WeaknessScore: 0.5, LastAttempted: &now}); err != nil {
			t.Fatal(err)
		}
	}
	got, _ := e.CourseReviewConcepts(st.ID, "plan.flat", false)
	full, _ := e.CourseReviewConcepts(st.ID, "plan.flat", true)
	if len(got) != 8 || len(full) != 8 {
		t.Fatalf("got %d and %d", len(got), len(full))
	}
	picked, _, _ := e.CourseQuizConceptsFor(st.ID, QuizScope{Courses: []string{"plan.flat"}, Limit: 3, BySection: true, Gradable: true})
	if len(picked) != 3 {
		t.Fatalf("limit 3 picked %d", len(picked))
	}
}

// proofStubAll files a proof under every topic.
type proofStubAll struct{ proofSrcStub }

func (s proofStubAll) ByConcept() map[string][]string {
	return map[string][]string{"g1": {"p"}, "g2": {"p"}, "g3": {"p"}, "g4": {"p"}, "n1": {"p"}, "n2": {"p"}, "n3": {"p"}, "n4": {"p"}, "b1": {"p"}}
}

// Neither test ever serves a proof, even on topics that serve one every time
// in practice.
func TestCourseTests_NeverServeProofs(t *testing.T) {
	e, _ := testsFixture(t)
	attachProofsFrom(e.dag, e.registry, proofStubAll{proofSrcStub{serving: true, share: 1}})
	if p, _ := e.registry.GenerateContext("n1", generator.GeneratorContext{Difficulty: 0.5}); !strings.HasPrefix(p.Answer, "proof:") {
		t.Fatalf("fixture: practice should serve the proof, got %q", p.Answer)
	}
	st, _ := e.CreateStudent("pf")
	for run := 0; run < 3; run++ {
		served, _ := placementRun(t, e, st.ID, "plan.mixed", func(string) bool { return run == 1 })
		for id, ps := range served {
			for _, p := range ps {
				if strings.HasPrefix(p.Answer, "proof:") {
					t.Fatalf("placement served a proof on %s", id)
				}
			}
		}
	}
	// The review test's session (server/course_tests.go): Objective set.
	var list []*concepts.Concept
	for _, id := range []string{"g1", "g2", "n1", "n2", "n3", "n4", "b1"} {
		list = append(list, e.dag.Concept(id))
	}
	qe := quiz.NewEngine(e.dag, e.registry)
	for run := 0; run < 5; run++ {
		s := qe.StartLimit(list, len(list))
		s.Objective = true
		for {
			p, cid, err := qe.NextQuestion(s)
			if err != nil {
				t.Fatal(err)
			}
			if p == nil {
				break
			}
			if strings.HasPrefix(p.Answer, "proof:") || strings.HasPrefix(p.Answer, "recall:") || strings.HasPrefix(p.Answer, "code:") {
				t.Fatalf("review served %q on %s", p.Answer, cid)
			}
			qe.RecordAnswer(s, cid, true)
		}
	}
}

func keys(m map[string][]generator.Problem) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

// A topic placed by the placement test and then practised is studied: it must
// be reviewable. One that was only placed is not.
func TestStudiedForReviewCountsPlacedThenPractised(t *testing.T) {
	if studiedForReview(&storage.ConceptProgress{PlacementSeeded: true}) {
		t.Error("placed only, never practised, counted as studied")
	}
	if !studiedForReview(&storage.ConceptProgress{PlacementSeeded: true, Attempts: 3}) {
		t.Error("placed and then practised was left out of review")
	}
}
