package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/modules"
	"github.com/chuma-beep/mathua/internal/scheduler"
	"github.com/chuma-beep/mathua/internal/storage"
)

// crossTrackEngine: a math concept "a" and a module track "econ" whose "x"
// needs "a" and whose "y" needs "x". Prerequisites cross the track boundary.
func crossTrackEngine(t *testing.T) *Engine {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "modules", "econ")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"id": "econ", "name": "Econ", "tracks": [{"id": "econ", "label": "Econ", "no_placement": true, "domains": ["econ"]}]}`
	if err := os.WriteFile(filepath.Join(dir, "module.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	mods, err := modules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	th := concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 60}
	d, err := concepts.Build([]concepts.Concept{
		{ID: "a", Label: "A", Domain: "algebra", GradingType: "numeric", Prerequisites: []string{}, MasteryThreshold: th},
		{ID: "x", Label: "X", Domain: "econ", GradingType: "numeric", Prerequisites: []string{"a"}, MasteryThreshold: th},
		{ID: "y", Label: "Y", Domain: "econ", GradingType: "numeric", Prerequisites: []string{"x"}, MasteryThreshold: th},
	})
	if err != nil {
		t.Fatal(err)
	}
	store, err := storage.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	reg := generator.NewRegistry()
	for _, id := range []string{"a", "x", "y"} {
		reg.Register(id, &testGen{answer: "1"})
	}
	e := New(store, d, reg, nil, nil)
	e.SetModules(mods)
	return e
}

func TestCrossTrack_ViewKeepsMembersOnly(t *testing.T) {
	e := crossTrackEngine(t)
	if n := e.TrackDAG("econ").Count(); n != 2 {
		t.Fatalf("econ track should hold its own two concepts, got %d", n)
	}
	if n := e.TrackDAG("math").Count(); n != 1 {
		t.Fatalf("math track should hold only a, got %d", n)
	}
	if !e.IsBridge("a", "econ") || e.IsBridge("x", "econ") {
		t.Fatal("a is a bridge into econ; x is a member, not a bridge")
	}
}

func TestCrossTrack_RecommendsTheOutsidePrerequisite(t *testing.T) {
	e := crossTrackEngine(t)
	st, err := e.CreateStudent("cross")
	if err != nil {
		t.Fatal(err)
	}
	res, err := e.RecommendNextTrack(st.ID, nil, "econ")
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	if res.Primary != nil {
		ids[res.Primary.ConceptID] = true
	}
	for _, r := range res.Alternatives {
		ids[r.ConceptID] = true
	}
	if !ids["a"] {
		t.Fatalf("econ needs math's a, which is ready to learn: it should be offered, got %v", ids)
	}
	if ids["y"] {
		t.Fatalf("y is two steps away and must not be offered yet, got %v", ids)
	}
	for _, r := range append([]scheduler.Recommendation{*res.Primary}, res.Alternatives...) {
		if r.ConceptID == "a" && r.Detail != "Comes before X in this track" {
			t.Fatalf("a bridge should say what it unlocks, got %q", r.Detail)
		}
	}
}

func TestCrossTrack_EligibilityAndCreditCrossTracks(t *testing.T) {
	e := crossTrackEngine(t)
	st, err := e.CreateStudent("cross-credit")
	if err != nil {
		t.Fatal(err)
	}
	if el, _ := e.EligibilityFor(st.ID, "x"); el.Eligible {
		t.Fatal("x needs a, which is untouched: it should be locked")
	}
	// Credit flows across the track boundary, weakening with distance: y is two
	// links above math's a, so it gives a quarter; x, one link above, a half.
	e.creditPrerequisites(st.ID, "y", true)
	p, err := e.repo.GetProgress(st.ID, "a")
	if err != nil || p == nil || p.ImplicitCredit != creditPrereqWeight*creditPrereqWeight || isCredited(p) {
		t.Fatalf("mastering econ's y should give math's a a quarter credit: %+v %v", p, err)
	}
	e.creditPrerequisites(st.ID, "x", true)
	e.creditPrerequisites(st.ID, "x", true)
	p, err = e.repo.GetProgress(st.ID, "a")
	if err != nil || !isCredited(p) {
		t.Fatalf("a quarter plus two halves should credit math's a: %+v %v", p, err)
	}
	if el, _ := e.EligibilityFor(st.ID, "x"); !el.Eligible {
		t.Fatalf("x should be eligible once a is credited: %+v", el)
	}
}

func TestCrossTrack_JumpInWhenEverythingWaitsOnAnotherTrack(t *testing.T) {
	e := crossTrackEngine(t)
	st, err := e.CreateStudent("jump")
	if err != nil {
		t.Fatal(err)
	}
	// a is not offerable from econ once the learner has moved past it in the
	// test below; here, exclude it to model "nothing ready in this track".
	res, err := e.RecommendNextTrack(st.ID, []string{"a"}, "econ")
	if err != nil {
		t.Fatal(err)
	}
	if res.Primary == nil || res.Primary.ConceptID != "x" || res.Primary.Badge != "Jump in" {
		t.Fatalf("with only outside prerequisites missing, econ should offer x as a jump-in, got %+v", res.Primary)
	}
}
