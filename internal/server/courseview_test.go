package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/auth"
	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/engine"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/planning"
	"github.com/chuma-beep/mathua/internal/storage"
)

// The course view lists a course's modules with each topic's state, keeps the
// path's prerequisite fill-ins apart from the plan, and names what to do next.
func TestCourseView(t *testing.T) {
	th := concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 60}
	d, _ := concepts.Build([]concepts.Concept{
		{ID: "a", Label: "A", Domain: "d", GradingType: "numeric", MasteryThreshold: th},
		{ID: "b", Label: "B", Domain: "d", GradingType: "numeric", Prerequisites: []string{"a"}, MasteryThreshold: th},
		{ID: "c", Label: "C", Domain: "d", GradingType: "numeric", Prerequisites: []string{"b"}, MasteryThreshold: th},
	})
	store, _ := storage.NewSQLiteStore(":memory:")
	t.Cleanup(func() { store.Close() })
	reg := generator.NewRegistry()
	for _, id := range []string{"a", "b", "c"} {
		reg.Register(id, &testGen{})
	}
	cp := filepath.Join(t.TempDir(), "courses.json")
	os.WriteFile(cp, []byte(`[{"id":"plan.t","name":"T: study plan","term":"Fall","plan":["b","c"],
		"modules":[{"title":"Unit 1","topics":["b"]},{"title":"Unit 2","topics":["c"]}]}]`), 0o644)
	pl, err := planning.Load(cp, d)
	if err != nil {
		t.Fatal(err)
	}
	s := New(engine.New(store, d, reg, nil, pl), store, auth.New(store))
	mux := http.NewServeMux()
	s.Register(mux)
	token, sid := guestTokenAndID(t, mux)

	rec := authedGet(t, mux, "/api/courses/view?id=plan.t", token)
	if rec.Code != 200 {
		t.Fatalf("view: %d %s", rec.Code, rec.Body.String())
	}
	var v courseView
	json.Unmarshal(rec.Body.Bytes(), &v)
	if v.Term != "Fall" || len(v.Modules) != 2 || v.Total != 2 || v.Modules[0].Topics[0].ID != "b" {
		t.Fatalf("modules: %+v", v)
	}
	if len(v.Prerequisites) != 1 || v.Prerequisites[0].ID != "a" {
		t.Fatalf("prerequisite fill-in should be listed apart: %+v", v.Prerequisites)
	}
	if !v.Modules[0].Topics[0].Locked || v.NextTopic == nil || v.NextTopic.ID != "a" {
		t.Fatalf("b is locked behind a, so a is next: %+v", v.NextTopic)
	}
	if !v.NeedsPlacement {
		t.Fatal("nothing started and the plan locked: placement should be offered")
	}
	if v.Followed {
		t.Fatal("not followed yet")
	}
	// Master a: b opens and is next; placement is no longer the offer.
	ts := time.Now()
	now := &ts
	store.UpsertProgress(&storage.ConceptProgress{StudentID: sid, ConceptID: "a", Status: "MASTERED", Attempts: 5, Streak: 5, MasteredAt: now, LastAttempted: now})
	rec = authedGet(t, mux, "/api/courses/view?id=plan.t", token)
	json.Unmarshal(rec.Body.Bytes(), &v)
	if len(v.Prerequisites) != 0 || v.NextTopic == nil || v.NextTopic.ID != "b" || v.NeedsPlacement {
		t.Fatalf("after mastering a: prereqs=%d next=%v placement=%v", len(v.Prerequisites), v.NextTopic, v.NeedsPlacement)
	}
	if rec := authedGet(t, mux, "/api/courses/view?id=nope", token); rec.Code != 404 {
		t.Fatalf("unknown course: %d", rec.Code)
	}
}

// Q27/Q2/Q33: the next step is the open prerequisite nearest the plan, not the
// first root in topological order; prerequisites the placement already put
// the learner above are not listed; and only a mastered topic can be due.
func TestCourseView_NearestNextAndHonestDue(t *testing.T) {
	th := concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 60}
	d, _ := concepts.Build([]concepts.Concept{
		{ID: "a", Label: "A (root, far)", Domain: "d", GradingType: "numeric", MasteryThreshold: th},
		{ID: "w", Label: "W (root, near)", Domain: "d", GradingType: "numeric", MasteryThreshold: th},
		{ID: "x", Label: "X", Domain: "d", GradingType: "numeric", Prerequisites: []string{"a"}, MasteryThreshold: th},
		{ID: "p", Label: "P", Domain: "d", GradingType: "numeric", Prerequisites: []string{"x", "w"}, MasteryThreshold: th},
		{ID: "q", Label: "Q", Domain: "d", GradingType: "numeric", MasteryThreshold: th},
		{ID: "s", Label: "S", Domain: "d", GradingType: "numeric", Prerequisites: []string{"z"}, MasteryThreshold: th},
		{ID: "z", Label: "Z", Domain: "d", GradingType: "numeric", MasteryThreshold: th},
	})
	store, _ := storage.NewSQLiteStore(":memory:")
	t.Cleanup(func() { store.Close() })
	reg := generator.NewRegistry()
	for _, id := range []string{"a", "w", "x", "p", "q", "s", "z"} {
		reg.Register(id, &testGen{})
	}
	cp := filepath.Join(t.TempDir(), "courses.json")
	os.WriteFile(cp, []byte(`[{"id":"plan.n","name":"N","plan":["p","q","s"]}]`), 0o644)
	pl, err := planning.Load(cp, d)
	if err != nil {
		t.Fatal(err)
	}
	s := New(engine.New(store, d, reg, nil, pl), store, auth.New(store))
	mux := http.NewServeMux()
	s.Register(mux)
	token, sid := guestTokenAndID(t, mux)

	past := time.Now().Add(-48 * time.Hour)
	// q: a plan topic in progress with a past practice date: not a review.
	store.UpsertProgress(&storage.ConceptProgress{StudentID: sid, ConceptID: "q", Status: "LEARNING", Attempts: 2, NextReviewDue: &past, LastAttempted: &past})
	// z: placed by the placement test (reachable, never studied).
	store.UpsertProgress(&storage.ConceptProgress{StudentID: sid, ConceptID: "z", Status: "LEARNING", PlacementSeeded: true, LastAttempted: &past})

	var v courseView
	json.Unmarshal(authedGet(t, mux, "/api/courses/view?id=plan.n", token).Body.Bytes(), &v)
	if v.ReviewsDue != 0 {
		t.Errorf("reviews_due = %d; a learning topic's practice date is not a review", v.ReviewsDue)
	}
	for _, m := range v.Modules {
		for _, tp := range m.Topics {
			if tp.Due {
				t.Errorf("%s marked due while not mastered", tp.ID)
			}
		}
	}
	for _, pr := range v.Prerequisites {
		if pr.ID == "z" {
			t.Error("z was placed by the placement test and should not be listed as work")
		}
	}
	// q is open and unfinished, so it is next; mastering it leaves p (locked)
	// and the nearest open prerequisite of p is w, not the root a.
	ts := time.Now()
	store.UpsertProgress(&storage.ConceptProgress{StudentID: sid, ConceptID: "q", Status: "MASTERED", Attempts: 5, Streak: 5, MasteredAt: &ts, LastAttempted: &ts, LastReviewed: &ts})
	store.UpsertProgress(&storage.ConceptProgress{StudentID: sid, ConceptID: "s", Status: "MASTERED", Attempts: 5, Streak: 5, MasteredAt: &ts, LastAttempted: &ts, LastReviewed: &ts})
	json.Unmarshal(authedGet(t, mux, "/api/courses/view?id=plan.n", token).Body.Bytes(), &v)
	if v.NextTopic == nil || v.NextTopic.ID != "w" {
		t.Fatalf("next = %+v, want w (one step from p), not the root a", v.NextTopic)
	}
	// A mastered topic past its review date is due.
	store.UpsertProgress(&storage.ConceptProgress{StudentID: sid, ConceptID: "q", Status: "MASTERED", Attempts: 5, Streak: 5, MasteredAt: &past, LastAttempted: &past, LastReviewed: &past, NextReviewDue: &past})
	json.Unmarshal(authedGet(t, mux, "/api/courses/view?id=plan.n", token).Body.Bytes(), &v)
	if v.ReviewsDue != 1 {
		t.Errorf("reviews_due = %d, want 1 (q mastered and past due)", v.ReviewsDue)
	}
}

// A course that ranks its lecture notes among its texts ("notes:<module>")
// had the raw key listed on its page as a book: "notes:ma-program (online)".
func TestCourseTextNotesKey(t *testing.T) {
	titles := map[string]string{"m/book.pdf": "A Book"}
	libIDs := map[string]string{"m/book.pdf": "m-book"}
	if got := resolveCourseText("notes:ma-program", titles, libIDs); got.Title != "Lecture notes" || got.Kind != "notes" || got.URL != "" {
		t.Errorf("notes key resolved to %+v", got)
	}
	if got := resolveCourseText("m/book.pdf", titles, libIDs); got.Title != "A Book" || got.Kind != "pdf" || got.LibraryID != "m-book" {
		t.Errorf("file key resolved to %+v", got)
	}
	if got := resolveCourseText("web:Statistical Learning", titles, libIDs); got.Title != "Statistical Learning" || got.Kind != "web" {
		t.Errorf("web key resolved to %+v", got)
	}
}

// courseViewFixture is a three-unit course over unconnected topics: Unit 1
// (a, b), Unit 2 (c, d) and a unit taught after the course's date (e, f).
func courseViewFixture(t *testing.T) (*http.ServeMux, *storage.SQLiteStore, *Server, string, string) {
	t.Helper()
	th := concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 60}
	ids := []string{"a", "b", "c", "d", "e", "f"}
	var cs []concepts.Concept
	for _, id := range ids {
		cs = append(cs, concepts.Concept{ID: id, Label: "Topic " + id, Domain: "d", GradingType: "numeric", MasteryThreshold: th})
	}
	d, _ := concepts.Build(cs)
	store, _ := storage.NewSQLiteStore(":memory:")
	t.Cleanup(func() { store.Close() })
	reg := generator.NewRegistry()
	for _, id := range ids {
		reg.Register(id, &testGen{})
	}
	cp := filepath.Join(t.TempDir(), "courses.json")
	os.WriteFile(cp, []byte(`[{"id":"plan.u","name":"U","due":"2099-12-01","plan":["a","b","c","d","e","f"],
		"modules":[{"title":"Unit 1","topics":["a","b"]},{"title":"Unit 2","topics":["c","d"]},
		           {"title":"Unit 3 (after the final)","topics":["e","f"],"after_due":true}]}]`), 0o644)
	pl, err := planning.Load(cp, d)
	if err != nil {
		t.Fatal(err)
	}
	s := New(engine.New(store, d, reg, nil, pl), store, auth.New(store))
	mux := http.NewServeMux()
	s.Register(mux)
	token, sid := guestTokenAndID(t, mux)
	return mux, store, s, token, sid
}

// Pacing counts the topics due by the course's date, not the whole plan: a
// unit marked after_due stays on the page but out of the pace.
func TestCourseView_DueTotalLeavesOutAfterDueModules(t *testing.T) {
	mux, store, _, token, sid := courseViewFixture(t)
	now := time.Now()
	store.UpsertProgress(&storage.ConceptProgress{StudentID: sid, ConceptID: "a", Status: "MASTERED", Attempts: 6, LastAttempted: &now, LastReviewed: &now})
	store.UpsertProgress(&storage.ConceptProgress{StudentID: sid, ConceptID: "e", Status: "MASTERED", Attempts: 6, LastAttempted: &now, LastReviewed: &now})
	var v courseView
	json.Unmarshal(authedGet(t, mux, "/api/courses/view?id=plan.u", token).Body.Bytes(), &v)
	if v.Total != 6 || v.Mastered != 2 {
		t.Fatalf("total %d mastered %d, want 6 and 2", v.Total, v.Mastered)
	}
	if v.DueTotal != 4 || v.DueMastered != 1 {
		t.Errorf("due_total %d due_mastered %d, want 4 and 1 (unit 3 is after the date)", v.DueTotal, v.DueMastered)
	}
	if !v.Modules[2].AfterDue || v.Modules[0].AfterDue {
		t.Errorf("after_due flags: %+v", v.Modules)
	}
}

// A course's "Up next" follows the learner's pins: the earliest pinned topic
// of the course that is not mastered, in course order, ahead of the first
// open topic. A mastered pin, or one from another unit pinned later in the
// course, does not take its place.
func TestCourseView_UpNextPrefersEarliestOpenPin(t *testing.T) {
	mux, store, s, token, sid := courseViewFixture(t)
	var v courseView
	json.Unmarshal(authedGet(t, mux, "/api/courses/view?id=plan.u", token).Body.Bytes(), &v)
	if v.NextTopic == nil || v.NextTopic.ID != "a" {
		t.Fatalf("no pins: next = %+v, want a", v.NextTopic)
	}
	now := time.Now()
	store.UpsertProgress(&storage.ConceptProgress{StudentID: sid, ConceptID: "b", Status: "MASTERED", Attempts: 6, LastAttempted: &now, LastReviewed: &now})
	if _, err := s.eng.SetPins(sid, []string{"d", "b", "c"}); err != nil {
		t.Fatal(err)
	}
	json.Unmarshal(authedGet(t, mux, "/api/courses/view?id=plan.u", token).Body.Bytes(), &v)
	if v.NextTopic == nil || v.NextTopic.ID != "c" {
		t.Fatalf("pins d, b (mastered), c: next = %+v, want c (the earliest open pin in course order)", v.NextTopic)
	}
}

type fakeWritten struct {
	enabled bool
	by      map[string][]string
}

func (f fakeWritten) Enabled() bool                  { return f.enabled }
func (f fakeWritten) ByConcept() map[string][]string { return f.by }

// Without an API key, proofs and writing problems are left out of practice,
// silently. The course view counts the ones on the course's topics, so the
// page can say they exist.
func TestCourseView_CountsWrittenTasksThatNeedAKey(t *testing.T) {
	mux, _, s, token, _ := courseViewFixture(t)
	by := map[string][]string{"a": {"p1", "p2"}, "e": {"p3"}, "elsewhere": {"p4"}}
	s.eng.SetWrittenTaskCatalog(fakeWritten{enabled: false, by: by})
	var v courseView
	json.Unmarshal(authedGet(t, mux, "/api/courses/view?id=plan.u", token).Body.Bytes(), &v)
	if v.UngradedTasks != 3 {
		t.Errorf("ungraded_tasks = %d, want 3 (two on a, one on e; p4 is not on the course)", v.UngradedTasks)
	}
	s.eng.SetWrittenTaskCatalog(fakeWritten{enabled: true, by: by})
	v = courseView{}
	json.Unmarshal(authedGet(t, mux, "/api/courses/view?id=plan.u", token).Body.Bytes(), &v)
	if v.UngradedTasks != 0 {
		t.Errorf("with a grader: ungraded_tasks = %d, want 0", v.UngradedTasks)
	}
}
