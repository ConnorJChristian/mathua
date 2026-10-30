package engine

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/diagnostic"
	"github.com/chuma-beep/mathua/internal/generator/all"
	"github.com/chuma-beep/mathua/internal/planning"
	"github.com/chuma-beep/mathua/internal/storage"
)

// floorDomainsForTest is the grade-school floor a graduate learner is past.
var floorDomainsForTest = map[string]bool{"arithmetic": true, "fractions": true, "prealgebra": true, "geometry": true}

// gradCoursePlan is a graduate course's plan on the base graph. It lists one
// floor topic of its own (geo.coord.distance): a course may ask for one.
var gradCoursePlan = []string{
	"calc.deriv.chain_rule", "calc.deriv.exp_log", "calc.deriv.implicit", "calc.deriv.partial",
	"calc.deriv.marginal_analysis", "calc.multi.tangent_plane", "calc.multi.lagrange",
	"calc.seq.monotone", "calc.integral.ftc", "linalg.matrix.inverse_2x2", "alg.log.concept",
	"stat.prob.basic", "geo.coord.distance",
}

// The pinned floor topic: a pin brings one back.
const pinnedFloorTopic = "arith.factor.gcf"

// gradLearner builds an engine over the whole base graph with every
// generator, and a learner following a graduate course after a placement
// long ago: every topic off the plan is held, by placement or by practice,
// and has decayed; the floor domains hold practised mastered topics due for
// review, practised topics still being learned and untouched ones.
func gradLearner(t *testing.T) (*Engine, string, *concepts.DAG) {
	t.Helper()
	dag, err := concepts.LoadDirs(filepath.Join("..", "..", "data", "concepts"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	cp := filepath.Join(dir, "courses.json")
	plan := `"` + gradCoursePlan[0] + `"`
	for _, id := range gradCoursePlan[1:] {
		plan += `,"` + id + `"`
	}
	if err := os.WriteFile(cp, []byte(`[{"id":"grad","name":"Mathematics for economists","grade":"MA","description":"","plan":[`+plan+`]}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	pl, err := planning.Load(cp, dag)
	if err != nil {
		t.Fatal(err)
	}
	store, err := storage.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	e := New(store, dag, all.Registry(), nil, pl)
	st, err := e.CreateStudent("graduate")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetActiveCourses(st.ID, []string{"grad"}); err != nil {
		t.Fatal(err)
	}
	if err := store.SetDiagnosticCompleted(st.ID); err != nil {
		t.Fatal(err)
	}
	inPlan := map[string]bool{}
	for _, id := range gradCoursePlan {
		inPlan[id] = true
	}
	long := time.Now().Add(-90 * 24 * time.Hour)
	due := time.Now().Add(-60 * 24 * time.Hour)
	var batch []*storage.ConceptProgress
	for i, c := range dag.Order() {
		if inPlan[c.ID] {
			continue // the course's own topics are new work
		}
		p := &storage.ConceptProgress{StudentID: st.ID, ConceptID: c.ID, SM2EFactor: 2.5,
			LastAttempted: &long, LastReviewed: &long}
		switch {
		case c.ID == pinnedFloorTopic:
			p.Status, p.Attempts, p.WeaknessScore = "LEARNING", 2, 0.6
		case floorDomainsForTest[c.Domain] && i%3 == 0:
			p.Status, p.Attempts, p.NextReviewDue = "MASTERED", 4, &due
		case floorDomainsForTest[c.Domain] && i%3 == 1:
			p.Status, p.Attempts, p.WeaknessScore = "LEARNING", 2, 0.7
		case floorDomainsForTest[c.Domain]:
			continue // untouched
		case i%2 == 0:
			p.Status, p.PlacementSeeded, p.NextReviewDue = "MASTERED", true, &due
		default:
			p.Status, p.Attempts, p.NextReviewDue = "MASTERED", 3, &due
		}
		batch = append(batch, p)
	}
	if err := store.UpsertProgressBatch(batch); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetPins(st.ID, []string{pinnedFloorTopic}); err != nil {
		t.Fatal(err)
	}
	return e, st.ID, dag
}

// TestNoFloorTopicsForCourseLearner is the guard against grade-school topics
// coming back to a graduate learner (C's report: "repeated regressions on
// the grade school math showing up on the reviews, diagnostic test, etc").
// With no MATHUA_REVIEW_FLOOR_DOMAINS set, a learner on a graduate course is
// never served an arithmetic, fractions, prealgebra or geometry topic as a
// review, in Up next, in the mixed session, as remediation, on the mastery
// check or in a placement test (course or onboarding), unless the course
// lists it or the learner pinned it.
func TestNoFloorTopicsForCourseLearner(t *testing.T) {
	t.Setenv("MATHUA_REVIEW_FLOOR_DOMAINS", "")
	e, sid, dag := gradLearner(t)
	allowed := map[string]bool{pinnedFloorTopic: true, "geo.coord.distance": true}
	check := func(surface, id string) {
		t.Helper()
		c := dag.Concept(id)
		if c != nil && floorDomainsForTest[c.Domain] && !allowed[id] {
			t.Errorf("%s served floor topic %s (%s, %s)", surface, id, c.Label, c.Domain)
		}
	}

	if f := e.Floor(sid); f.Source != "courses" || f.Scope != FloorAll || len(f.Domains) != 4 {
		t.Fatalf("floor = %+v, want the four grade-school domains from the course", f)
	}

	// Reviews: the whole due queue, one session.
	rs, _ := e.repo.CreateSession(sid)
	for i := 0; i < 400; i++ {
		q, err := e.NextReviewQuestion(rs.ID, sid)
		if err != nil {
			t.Fatal(err)
		}
		if q == nil {
			break
		}
		check("review", q.ConceptID)
	}
	if n, _ := e.ReviewsDue(sid); n == 0 {
		t.Error("reviews above the floor should still be due")
	}

	// Up next, paging through every recommendation.
	var shown []string
	pinSeen := false
	for i := 0; i < 15; i++ {
		res, err := e.RecommendNext(sid, shown)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range recs(res) {
			if r.ConceptID == "" {
				continue
			}
			check("up next", r.ConceptID)
			pinSeen = pinSeen || r.ConceptID == pinnedFloorTopic
			shown = append(shown, r.ConceptID)
		}
	}
	if !pinSeen {
		t.Error("the pinned floor topic should be offered: a pin lifts the floor")
	}

	// The mixed session, missing often enough to halt and remediate.
	ms, _ := e.repo.CreateSession(sid)
	for i := 0; i < 40; i++ {
		q, err := e.NextQuestion(ms.ID, sid)
		if err != nil {
			t.Fatal(err)
		}
		if q == nil {
			break
		}
		check("mixed session", q.ConceptID)
		ans := "-9999"
		if key, ok := e.ExpectedAnswerForReplay(ms.ID); ok && i%3 == 0 {
			ans = key
		}
		if _, err := e.SubmitAnswer(ms.ID, sid, q.AttemptID, ans, 15); err != nil {
			t.Fatal(err)
		}
	}

	// Remediation for every plan topic, and the mastery check's queue after
	// missing each.
	for _, id := range gradCoursePlan {
		for _, p := range e.remedialPrerequisites(sid, id) {
			check("remediation of "+id, p)
		}
		d := 0.8
		if _, err := e.SubmitQuizAnswer(sid, id, "-9999", "1", 20, "q", &d); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range e.QuizRemedial(sid) {
		check("mastery check remediation", id)
	}
	picked, _, ok := e.CourseQuizConcepts(sid)
	if !ok {
		t.Fatal("no mastery check for a course learner")
	}
	for _, c := range picked {
		check("mastery check", c.ID)
	}

	// Placement: the course placement and the onboarding placement over the
	// course plan, answered right throughout.
	// Placement also stays within CoursePlacementDepth links of the plan.
	depth := map[string]int{}
	queue := append([]string(nil), gradCoursePlan...)
	for _, id := range queue {
		depth[id] = 0
	}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		for _, p := range dag.PrereqsOf(id) {
			if _, ok := depth[p.ID]; !ok {
				depth[p.ID] = depth[id] + 1
				queue = append(queue, p.ID)
			}
		}
	}
	placed := func(surface string) func(string) {
		return func(id string) {
			check(surface, id)
			if d := depth[id]; d > CoursePlacementDepth {
				t.Errorf("%s asked %s, %d links below the course (limit %d)", surface, id, d, CoursePlacementDepth)
			}
		}
	}
	sess, q, err := e.StartCourseDiagnostic(sid, "grad")
	if err != nil {
		t.Fatal(err)
	}
	walkDiagnostic(t, e, sess, q, placed("course placement"))
	sess, q, err = e.StartGoalDiagnostic(sid, gradCoursePlan)
	if err != nil {
		t.Fatal(err)
	}
	walkDiagnostic(t, e, sess, q, placed("onboarding placement"))
}

// walkDiagnostic answers a placement right until it ends, reporting every
// topic it asks.
func walkDiagnostic(t *testing.T, e *Engine, sess *diagnostic.Session, q *Question, seen func(string)) {
	t.Helper()
	if q == nil {
		return
	}
	seen(q.ConceptID)
	cid := q.ConceptID
	for i := 0; i < 80; i++ {
		e.SubmitDiagnosticAnswerTimed(sess, cid, true, 5, 30)
		if e.IsDiagnosticComplete(sess) {
			return
		}
		prob, next, err := e.NextDiagnosticQuestion(sess)
		if err != nil {
			t.Fatal(err)
		}
		if prob == nil || next == "" {
			return
		}
		seen(next)
		cid = next
	}
}
