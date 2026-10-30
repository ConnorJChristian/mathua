package engine

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/planning"
	"github.com/chuma-beep/mathua/internal/scheduler"
	"github.com/chuma-beep/mathua/internal/storage"
)

func recs(r scheduler.RecommendationResponse) []scheduler.Recommendation {
	var out []scheduler.Recommendation
	if r.Primary != nil {
		out = append(out, *r.Primary)
	}
	return append(out, r.Alternatives...)
}

// A graduate course rests on a long foundation that an early diagnostic
// credits without the learner ever answering a question on it. Those topics
// were coming back as reviews ("Two-digit addition" as Up next) and as new
// topics. The fixture:
//
//	f1 -> f2 -> p1 <- f4      p1, p2: the course's own topics (plan.c)
//	f1 -> p2                  q: off the course, practised by the learner
//
// f1 placement-seeded and decayed, f2 held on partial credit and due, f4
// untouched but credited a little: all credit only, all off the plan.
func creditOnlyFixture(t *testing.T) (*Engine, string) {
	t.Helper()
	th := concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 60}
	d, err := concepts.Build([]concepts.Concept{
		{ID: "f1", Label: "Foundation 1", Domain: "arithmetic", GradingType: "numeric", Prerequisites: []string{}, MasteryThreshold: th},
		{ID: "f2", Label: "Foundation 2", Domain: "arithmetic", GradingType: "numeric", Prerequisites: []string{"f1"}, MasteryThreshold: th},
		{ID: "f4", Label: "Foundation 4", Domain: "algebra", GradingType: "numeric", Prerequisites: []string{"f1"}, MasteryThreshold: th},
		{ID: "p1", Label: "Plan 1", Domain: "calculus", GradingType: "numeric", Prerequisites: []string{"f2", "f4"}, MasteryThreshold: th},
		{ID: "p2", Label: "Plan 2", Domain: "calculus", GradingType: "numeric", Prerequisites: []string{"f1"}, MasteryThreshold: th},
		{ID: "q", Label: "Off course", Domain: "statistics", GradingType: "numeric", Prerequisites: []string{}, MasteryThreshold: th},
	})
	if err != nil {
		t.Fatal(err)
	}
	cp := filepath.Join(t.TempDir(), "courses.json")
	if err := os.WriteFile(cp, []byte(`[{"id":"plan.c","name":"C","plan":["p1","p2"]}]`), 0o644); err != nil {
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
	reg := generator.NewRegistry()
	for _, id := range []string{"f1", "f2", "f4", "p1", "p2", "q"} {
		reg.Register(id, &testGen{answer: "1"})
	}
	e := New(store, d, reg, nil, pl)
	st, err := e.CreateStudent("grad")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetActiveCourses(st.ID, []string{"plan.c"}); err != nil {
		t.Fatal(err)
	}
	if err := store.SetDiagnosticCompleted(st.ID); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-20 * 24 * time.Hour)
	past := time.Now().Add(-time.Hour)
	for _, p := range []storage.ConceptProgress{
		{ConceptID: "f1", Status: "MASTERED", PlacementSeeded: true, LastReviewed: &old, LastAttempted: &old},
		{ConceptID: "f2", Status: "MASTERED", ImplicitCredit: 1, LastReviewed: &past, NextReviewDue: &past},
		{ConceptID: "f4", Status: "UNSEEN", ImplicitCredit: 0.25},
	} {
		p.StudentID = st.ID
		p.SM2EFactor = 2.5
		if err := store.UpsertProgress(&p); err != nil {
			t.Fatal(err)
		}
	}
	return e, st.ID
}

func reviewServed(t *testing.T, e *Engine, sid string, n int) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for i := 0; i < n; i++ {
		sess, err := e.repo.CreateSession(sid)
		if err != nil {
			t.Fatal(err)
		}
		q, err := e.NextReviewQuestion(sess.ID, sid)
		if err != nil {
			t.Fatal(err)
		}
		if q != nil {
			out[q.ConceptID] = true
		}
	}
	return out
}

func TestCreditOnlyFoundationsAreNotReviewedOrSuggested(t *testing.T) {
	e, sid := creditOnlyFixture(t)

	// Only credit-only foundations are due: nothing to review.
	if n, _ := e.ReviewsDue(sid); n != 0 {
		t.Errorf("ReviewsDue = %d, want 0: f2 is due but held on credit alone", n)
	}
	if served := reviewServed(t, e, sid, 10); len(served) != 0 {
		t.Errorf("review session served %v; f1 and f2 were never practised and no course asks for them", served)
	}
	resp, err := e.RecommendNext(sid, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range recs(resp) {
		if r.ConceptID == "f1" || r.ConceptID == "f2" || r.ConceptID == "f4" {
			t.Errorf("recommended %s (%s, %s): a credit-only foundation off the course", r.ConceptID, r.Badge, r.Detail)
		}
	}

	// The same kind of topic stays reviewable where the learner or the
	// course asks for it: q was practised, p2 is on the plan.
	old := time.Now().Add(-20 * 24 * time.Hour)
	past := time.Now().Add(-time.Hour)
	now := time.Now()
	later := now.Add(72 * time.Hour)
	upsert := func(p storage.ConceptProgress) {
		t.Helper()
		p.StudentID = sid
		p.SM2EFactor = 2.5
		if err := e.repo.UpsertProgress(&p); err != nil {
			t.Fatal(err)
		}
	}
	upsert(storage.ConceptProgress{ConceptID: "q", Status: "MASTERED", Attempts: 6, LastReviewed: &past, LastAttempted: &past, NextReviewDue: &past})
	if n, _ := e.ReviewsDue(sid); n != 1 {
		t.Errorf("ReviewsDue = %d, want 1 (q)", n)
	}
	if served := reviewServed(t, e, sid, 10); len(served) != 1 || !served["q"] {
		t.Errorf("review session served %v, want q only", served)
	}
	upsert(storage.ConceptProgress{ConceptID: "q", Status: "MASTERED", Attempts: 7, LastReviewed: &now, LastAttempted: &now, NextReviewDue: &later})
	upsert(storage.ConceptProgress{ConceptID: "p2", Status: "MASTERED", PlacementSeeded: true, LastReviewed: &old, LastAttempted: &old})
	if served := reviewServed(t, e, sid, 10); len(served) != 1 || !served["p2"] {
		t.Errorf("review session served %v, want p2 only (placed, but on the plan)", served)
	}

	// A pin brings a foundation back.
	if _, err := e.SetPins(sid, []string{"f4"}); err != nil {
		t.Fatal(err)
	}
	resp, _ = e.RecommendNext(sid, nil)
	found := false
	for _, r := range recs(resp) {
		found = found || r.ConceptID == "f4"
	}
	if !found {
		t.Errorf("pinned f4 is not recommended: %+v", recs(resp))
	}
}

// Below the review floor, practice of one's own does not bring a mastered
// topic back as a review: f2 (arithmetic) was practised and is due, yet with
// arithmetic under the floor it is not served. q (statistics, above the floor)
// still is, and a pin lifts the floor for f2.
func TestReviewFloorStopsPractisedFoundations(t *testing.T) {
	e, sid := creditOnlyFixture(t)
	past := time.Now().Add(-time.Hour)
	upsert := func(p storage.ConceptProgress) {
		t.Helper()
		p.StudentID = sid
		p.SM2EFactor = 2.5
		if err := e.repo.UpsertProgress(&p); err != nil {
			t.Fatal(err)
		}
	}
	upsert(storage.ConceptProgress{ConceptID: "f2", Status: "MASTERED", Attempts: 9, LastReviewed: &past, LastAttempted: &past, NextReviewDue: &past})

	// With the floor switched off the practised foundation is due, as before.
	t.Setenv("MATHUA_REVIEW_FLOOR_DOMAINS", "none")
	if n, _ := e.ReviewsDue(sid); n != 1 {
		t.Fatalf("no floor: ReviewsDue = %d, want 1 (f2)", n)
	}
	// Unset, a learner on a course above school grades has the course's
	// floor (planning.GradeSchoolDomains): arithmetic is below it.
	t.Setenv("MATHUA_REVIEW_FLOOR_DOMAINS", "")
	if n, _ := e.ReviewsDue(sid); n != 0 {
		t.Errorf("course floor: ReviewsDue = %d, want 0", n)
	}
	if f := e.Floor(sid); f.Source != "courses" || f.Scope != FloorAll {
		t.Errorf("course floor: Floor = %+v", f)
	}

	t.Setenv("MATHUA_REVIEW_FLOOR_DOMAINS", "arithmetic, fractions")
	if n, _ := e.ReviewsDue(sid); n != 0 {
		t.Errorf("floor: ReviewsDue = %d, want 0", n)
	}
	if served := reviewServed(t, e, sid, 10); len(served) != 0 {
		t.Errorf("floor: review session served %v", served)
	}
	resp, err := e.RecommendNext(sid, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range recs(resp) {
		if r.ConceptID == "f2" {
			t.Errorf("floor: recommended f2 (%s)", r.Badge)
		}
	}

	upsert(storage.ConceptProgress{ConceptID: "q", Status: "MASTERED", Attempts: 6, LastReviewed: &past, LastAttempted: &past, NextReviewDue: &past})
	if served := reviewServed(t, e, sid, 10); len(served) != 1 || !served["q"] {
		t.Errorf("floor: served %v, want q (statistics is above the floor)", served)
	}

	if _, err := e.SetPins(sid, []string{"f2"}); err != nil {
		t.Fatal(err)
	}
	if n, _ := e.ReviewsDue(sid); n != 2 {
		t.Errorf("floor + pin: ReviewsDue = %d, want 2 (f2 pinned, q)", n)
	}
}
