package engine

import (
	"testing"

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
