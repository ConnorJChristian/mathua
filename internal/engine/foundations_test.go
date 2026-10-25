package engine

import (
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/diagnostic"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/storage"
)

// Q27: a learner placed into calculus is placed above the whole grade-school
// floor, not only the arithmetic that happens to sit under the questions
// asked. Fraction and prealgebra topics no correct answer covered were left
// unseen, and the queue served "What fraction of the bar is filled?" to an
// MA student.
func TestApplyGoalResults_PlacementAboveFoundationsCreditsTheFloor(t *testing.T) {
	th := concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 60}
	d, err := concepts.Build([]concepts.Concept{
		{ID: "arith.add", Label: "Add", Domain: "arithmetic", GradingType: "numeric", MasteryThreshold: th},
		{ID: "frac.basics", Label: "Fractions", Domain: "fractions", GradingType: "numeric", MasteryThreshold: th},
		{ID: "prealg.word", Label: "Word problems", Domain: "prealgebra", GradingType: "numeric", Prerequisites: []string{"frac.basics"}, MasteryThreshold: th},
		{ID: "calc.deriv", Label: "Derivatives", Domain: "calculus", GradingType: "numeric", Prerequisites: []string{"arith.add"}, MasteryThreshold: th},
		{ID: "calc.rates", Label: "Rates", Domain: "calculus", GradingType: "numeric", Prerequisites: []string{"prealg.word", "calc.deriv"}, MasteryThreshold: th},
	})
	if err != nil {
		t.Fatal(err)
	}
	store, _ := storage.NewSQLiteStore(":memory:")
	t.Cleanup(func() { store.Close() })
	e := New(store, d, generator.NewRegistry(), nil, nil)
	st, _ := e.CreateStudent("placed")

	session := &diagnostic.Session{StudentID: st.ID, Attempts: []diagnostic.Attempt{{ConceptID: "calc.deriv", Correct: true}}}
	if err := e.ApplyGoalResults(st.ID, session); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"arith.add", "frac.basics", "prealg.word"} {
		p, _ := store.GetProgress(st.ID, id)
		if p == nil || !p.PlacementSeeded {
			t.Errorf("%s: want placement-seeded after a correct calculus answer, got %+v", id, p)
		}
	}
	// Nothing above the floor is credited by the rule: rates was not answered.
	if p, _ := store.GetProgress(st.ID, "calc.rates"); p != nil && p.PlacementSeeded {
		t.Error("calc.rates must not be credited")
	}

	// A learner right only on grade-school topics keeps the old behaviour.
	st2, _ := e.CreateStudent("young")
	session2 := &diagnostic.Session{StudentID: st2.ID, Attempts: []diagnostic.Attempt{{ConceptID: "arith.add", Correct: true}}}
	if err := e.ApplyGoalResults(st2.ID, session2); err != nil {
		t.Fatal(err)
	}
	if p, _ := store.GetProgress(st2.ID, "frac.basics"); p != nil && p.PlacementSeeded {
		t.Error("a right arithmetic answer must not credit fractions")
	}
}

// Q27 (Study queue): a topic outside the learner's course path is never
// offered, even when the placement credited its prerequisite. Dropping it
// from the scheduler's view made it look never seen with its prerequisite
// met: "Addition word problems — Ready to learn" after a calculus placement.
func TestRecommend_OffPathTopicIsNotOffered(t *testing.T) {
	th := concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 60}
	d, err := concepts.Build([]concepts.Concept{
		{ID: "r", Label: "R", Domain: "d", GradingType: "numeric", MasteryThreshold: th},
		{ID: "w", Label: "W off path", Domain: "d", GradingType: "numeric", Prerequisites: []string{"r"}, MasteryThreshold: th},
		{ID: "q", Label: "Q", Domain: "d", GradingType: "numeric", Prerequisites: []string{"r"}, MasteryThreshold: th},
		{ID: "p", Label: "P", Domain: "d", GradingType: "numeric", Prerequisites: []string{"q"}, MasteryThreshold: th},
	})
	if err != nil {
		t.Fatal(err)
	}
	store, _ := storage.NewSQLiteStore(":memory:")
	t.Cleanup(func() { store.Close() })
	e := New(store, d, generator.NewRegistry(), nil, nil)
	st, _ := e.CreateStudent("pathed")
	past := time.Now().Add(-time.Hour)
	// Both r and w were credited by the placement (as the whole grade-school
	// floor is); only r is on the course path.
	for _, id := range []string{"r", "w"} {
		store.UpsertProgress(&storage.ConceptProgress{StudentID: st.ID, ConceptID: id, Status: "LEARNING", PlacementSeeded: true, WeaknessScore: 0.3, LastAttempted: &past})
	}
	e.mu.Lock()
	e.activePath[st.ID] = map[string]bool{"r": true, "q": true, "p": true}
	e.mu.Unlock()

	res, err := e.RecommendNext(st.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	if res.Primary != nil {
		ids = append(ids, res.Primary.ConceptID)
	}
	for _, a := range res.Alternatives {
		ids = append(ids, a.ConceptID)
	}
	for _, id := range ids {
		if id == "w" {
			t.Fatalf("w is off the course path but was offered: %v", ids)
		}
	}
}
