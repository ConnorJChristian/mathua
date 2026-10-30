package engine

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/planning"
	"github.com/chuma-beep/mathua/internal/storage"
)

// constGen serves one fixed question whose answer is "1".
type constGen struct{ id string }

func (g constGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return generator.Problem{Question: "What is 1 for " + g.id + "?", Answer: "1"}
}

// seqFixture is a learner on one course with a study plan. The plan's first
// module (due1..due3) is due by the date, its second (late1) is taught after
// it. Around the plan sit a topic off every path (off), a credit-only
// prerequisite (cred) and an untouched grade-school prerequisite (fl).
func seqFixture(t *testing.T, courses string, extra ...concepts.Concept) (*Engine, string) {
	t.Helper()
	th := concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 30}
	cs := []concepts.Concept{
		{ID: "r", Label: "Root", Domain: "calculus", Subdomain: "s.r", GradingType: "numeric", MasteryThreshold: th},
		{ID: "fl", Label: "Two-digit addition", Domain: "arithmetic", Subdomain: "s.fl", GradingType: "numeric", MasteryThreshold: th},
		{ID: "cred", Label: "Credited", Domain: "algebra", Subdomain: "s.cred", GradingType: "numeric", Prerequisites: []string{"r"}, MasteryThreshold: th},
		{ID: "due1", Label: "Due 1", Domain: "calculus", Subdomain: "s.d1", GradingType: "numeric", Prerequisites: []string{"r", "fl"}, MasteryThreshold: th},
		{ID: "due2", Label: "Due 2", Domain: "calculus", Subdomain: "s.d2", GradingType: "numeric", Prerequisites: []string{"r", "cred"}, MasteryThreshold: th},
		{ID: "due3", Label: "Due 3", Domain: "calculus", Subdomain: "s.d3", GradingType: "numeric", Prerequisites: []string{"r"}, MasteryThreshold: th},
		{ID: "late1", Label: "After the final", Domain: "calculus", Subdomain: "s.l1", GradingType: "numeric", Prerequisites: []string{"r"}, MasteryThreshold: th},
		{ID: "off", Label: "Off every path", Domain: "calculus", Subdomain: "s.off", GradingType: "numeric", Prerequisites: []string{"r"}, MasteryThreshold: th},
	}
	cs = append(cs, extra...)
	d, err := concepts.Build(cs)
	if err != nil {
		t.Fatal(err)
	}
	cp := filepath.Join(t.TempDir(), "courses.json")
	if err := os.WriteFile(cp, []byte(courses), 0o644); err != nil {
		t.Fatal(err)
	}
	pl, err := planning.Load(cp, d)
	if err != nil {
		t.Fatal(err)
	}
	reg := generator.NewRegistry()
	for _, c := range cs {
		reg.Register(c.ID, constGen{c.ID})
	}
	store, _ := storage.NewSQLiteStore(":memory:")
	t.Cleanup(func() { store.Close() })
	e := New(store, d, reg, nil, pl)
	st, _ := e.CreateStudent("seq")
	store.SetDiagnosticCompleted(st.ID)
	long := time.Now().Add(-3 * 24 * time.Hour)
	later := time.Now().Add(30 * 24 * time.Hour)
	store.UpsertProgressBatch([]*storage.ConceptProgress{
		{StudentID: st.ID, ConceptID: "r", Status: "MASTERED", Attempts: 5, SM2EFactor: 2.5, LastReviewed: &long, LastAttempted: &long, NextReviewDue: &later},
		{StudentID: st.ID, ConceptID: "cred", Status: "LEARNING", PlacementSeeded: true, SM2EFactor: 2.5, LastAttempted: &long},
	})
	return e, st.ID
}

func seqCourse(due time.Time) string {
	return `[{"id":"c","name":"Math for economists: study plan","grade":"MA","description":"",
		"plan":["due1","due2","due3","late1"],"due":"` + due.Format("2006-01-02") + `",
		"modules":[{"title":"1 Due","topics":["due1","due2","due3"]},{"title":"2 After the final","topics":["late1"],"after_due":true}]}]`
}

// The mixed session (/session) follows the same rules as Up next: the
// course's plan in paced order, never a topic taught after the due date
// while one due by it is ready, never a topic off every course path, a
// credit-only foundation or a topic below the floor. The untouched floor
// topic still counts as met for the plan topic above it.
func TestMixedSession_FollowsThePlanAndItsLimits(t *testing.T) {
	t.Setenv("MATHUA_REVIEW_FLOOR_DOMAINS", "")
	e, sid := seqFixture(t, seqCourse(time.Now().Add(40*24*time.Hour)))
	if _, err := e.SetActiveCourses(sid, []string{"c"}); err != nil {
		t.Fatal(err)
	}
	sess, _ := e.repo.CreateSession(sid)
	served := map[string]int{}
	var order []string
	for i := 0; i < 9; i++ {
		q, err := e.NextQuestion(sess.ID, sid)
		if err != nil {
			t.Fatal(err)
		}
		if q == nil {
			break
		}
		served[q.ConceptID]++
		order = append(order, q.ConceptID)
		if q.Reason == "" {
			t.Errorf("question %d (%s) carries no reason", i, q.ConceptID)
		}
		if _, err := e.SubmitAnswer(sess.ID, sid, q.AttemptID, "1", 10); err != nil {
			t.Fatal(err)
		}
	}
	for _, bad := range []string{"off", "cred", "fl"} {
		if served[bad] > 0 {
			t.Errorf("mixed session served %s: %v", bad, order)
		}
	}
	if served["due1"] == 0 {
		t.Errorf("due1 rests on an untouched floor topic and must still be served: %v", order)
	}
	// late1 may come only once due1..due3 are mastered (streak 3 each: 9
	// correct answers at least).
	for i, id := range order {
		if id == "late1" && i < 9 {
			t.Errorf("after-due topic served at %d before the due ones were done: %v", i, order)
		}
	}
}

// One review session serves a topic once: a review missed in it comes back
// in the next session, not two questions later.
func TestReviewSession_ServesATopicOnce(t *testing.T) {
	e, sid := seqFixture(t, seqCourse(time.Now().Add(40*24*time.Hour)))
	e.SetActiveCourses(sid, []string{"c"})
	past := time.Now().Add(-48 * time.Hour)
	for _, id := range []string{"due1", "due2"} {
		e.repo.UpsertProgress(&storage.ConceptProgress{StudentID: sid, ConceptID: id, Status: "MASTERED", Attempts: 4, SM2EFactor: 2.5, LastReviewed: &past, LastAttempted: &past, NextReviewDue: &past})
	}
	sess, _ := e.repo.CreateSession(sid)
	var order []string
	for i := 0; i < 6; i++ {
		q, err := e.NextReviewQuestion(sess.ID, sid)
		if err != nil {
			t.Fatal(err)
		}
		if q == nil {
			break
		}
		order = append(order, q.ConceptID)
		if q.Reason == "" {
			t.Errorf("review %s carries no reason", q.ConceptID)
		}
		e.SubmitAnswer(sess.ID, sid, q.AttemptID, "wrong", 10) // missed: still due
	}
	if len(order) != 2 || order[0] == order[1] {
		t.Errorf("review session served %v, want due1 and due2 once each", order)
	}
}

// A topic two followed courses share is one review: it counts once, takes
// one Up next card and is served once in a review session.
func TestReviews_TopicSharedByTwoCoursesCountsOnce(t *testing.T) {
	e, sid := seqFixture(t, `[
		{"id":"a","name":"A","grade":"MA","description":"","plan":["due1","due2"]},
		{"id":"b","name":"B","grade":"MA","description":"","plan":["due2","due3"]}]`)
	if _, err := e.SetActiveCourses(sid, []string{"a", "b"}); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-48 * time.Hour)
	e.repo.UpsertProgress(&storage.ConceptProgress{StudentID: sid, ConceptID: "due2", Status: "MASTERED", Attempts: 4, SM2EFactor: 2.5, LastReviewed: &past, LastAttempted: &past, NextReviewDue: &past})
	if n, _ := e.ReviewsDue(sid); n != 1 {
		t.Errorf("ReviewsDue = %d, want 1", n)
	}
	res, _ := e.RecommendNext(sid, nil)
	reviews := 0
	for _, r := range recs(res) {
		if r.Kind == "review" {
			reviews++
			if r.Why == "" {
				t.Error("the review card carries no reason")
			}
		}
	}
	if reviews != 1 {
		t.Errorf("Up next has %d review cards, want 1", reviews)
	}
	sess, _ := e.repo.CreateSession(sid)
	n := 0
	for i := 0; i < 4; i++ {
		q, _ := e.NextReviewQuestion(sess.ID, sid)
		if q == nil {
			break
		}
		n++
		e.SubmitAnswer(sess.ID, sid, q.AttemptID, "wrong", 10)
	}
	if n != 1 {
		t.Errorf("review session served the shared topic %d times, want 1", n)
	}
}

// Due reviews come before new topics, except when a followed course is due
// within scheduler.UrgentDays: then its next topic leads, and says why.
func TestUpNext_CourseDeadlineBeforeReviews(t *testing.T) {
	past := time.Now().Add(-48 * time.Hour)
	setup := func(due time.Time) (*Engine, string) {
		e, sid := seqFixture(t, seqCourse(due))
		e.SetActiveCourses(sid, []string{"c"})
		e.repo.UpsertProgress(&storage.ConceptProgress{StudentID: sid, ConceptID: "r", Status: "MASTERED", Attempts: 5, SM2EFactor: 2.5, LastReviewed: &past, LastAttempted: &past, NextReviewDue: &past})
		return e, sid
	}
	e, sid := setup(time.Now().Add(40 * 24 * time.Hour))
	res, _ := e.RecommendNext(sid, nil)
	if res.Primary == nil || res.Primary.Kind != "review" {
		t.Fatalf("far from the date a due review leads, got %+v", res.Primary)
	}
	e, sid = setup(time.Now().Add(3 * 24 * time.Hour))
	res, _ = e.RecommendNext(sid, nil)
	if res.Primary == nil || res.Primary.Kind == "review" {
		t.Fatalf("three days before the date the course's next topic leads, got %+v", res.Primary)
	}
	if res.Primary.ConceptID == "late1" {
		t.Error("an after-due topic is not urgent")
	}
	if res.Primary.Why == "" || res.Primary.Why[:len("Math for economists is due in")] != "Math for economists is due in" {
		t.Errorf("urgent topic reason = %q", res.Primary.Why)
	}
	found := false
	for _, r := range res.Alternatives {
		found = found || r.Kind == "review"
	}
	if !found {
		t.Error("the due review is still offered, after the urgent topic")
	}
}
