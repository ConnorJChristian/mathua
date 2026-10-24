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
