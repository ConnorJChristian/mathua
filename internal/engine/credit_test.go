package engine

import (
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/mastery"
	"github.com/chuma-beep/mathua/internal/storage"
)

// Fork change tests: prerequisite credit (credit.go). testEngine's graph is a <- b.

// creditStudent creates the learner the progress rows belong to: the progress
// table has a foreign key to students, so rows for an id that was never
// created are rejected.
func creditStudent(t *testing.T, e *Engine) string {
	t.Helper()
	st, err := e.CreateStudent("credit-learner")
	if err != nil {
		t.Fatal(err)
	}
	return st.ID
}

func TestCreditPrerequisites_MasteryGivesPartialCredit(t *testing.T) {
	e := testEngine(t)
	s1 := creditStudent(t, e)
	e.creditPrerequisites(s1, "b", true)

	p, err := e.repo.GetProgress(s1, "a")
	if err != nil || p == nil {
		t.Fatalf("prereq row missing: %v", err)
	}
	if p.ImplicitCredit != creditPrereqWeight || p.Status == string(mastery.StatusMastered) || isCredited(p) {
		t.Fatalf("one mastery above a direct prerequisite should give half credit, not mastery: %+v", p)
	}
}

func TestCreditPrerequisites_CreditAccumulatesToMastery(t *testing.T) {
	e := testEngine(t)
	s1 := creditStudent(t, e)
	e.creditPrerequisites(s1, "b", true)
	e.creditPrerequisites(s1, "b", true)

	p, err := e.repo.GetProgress(s1, "a")
	if err != nil || p == nil {
		t.Fatalf("prereq row missing: %v", err)
	}
	if p.Status != string(mastery.StatusMastered) || !p.PlacementSeeded || p.MasteredAt != nil || p.ImplicitCredit != creditFull {
		t.Fatalf("two half credits should credit a (MASTERED, placement, no MasteredAt), got %+v", p)
	}
	if p.LastReviewed == nil {
		t.Fatal("credited prereq should start its decay clock")
	}
	if !isCredited(p) {
		t.Fatal("isCredited should recognise the row")
	}
	if el, err := e.EligibilityFor(s1, "b"); err != nil || !el.Eligible {
		t.Fatalf("b should be eligible once a is credited: %+v %v", el, err)
	}
}

func TestCreditPrerequisites_CorrectAnswerWithoutMasteryGrantsNothing(t *testing.T) {
	e := testEngine(t)
	s1 := creditStudent(t, e)
	e.creditPrerequisites(s1, "b", false)
	if p, _ := e.repo.GetProgress(s1, "a"); p != nil && p.Status == string(mastery.StatusMastered) {
		t.Fatalf("no credit expected before mastery, got %+v", p)
	}
}

func TestCreditPrerequisites_LeavesDemonstratedMasteryAlone(t *testing.T) {
	e := testEngine(t)
	s1 := creditStudent(t, e)
	then := time.Now().UTC().Add(-48 * time.Hour).Truncate(time.Second)
	due := then.AddDate(0, 0, 6)
	if err := e.repo.UpsertProgress(&storage.ConceptProgress{
		StudentID: s1, ConceptID: "a", Status: string(mastery.StatusMastered),
		MasteredAt: &then, LastReviewed: &then, NextReviewDue: &due, SM2EFactor: 2.5,
	}); err != nil {
		t.Fatal(err)
	}
	e.creditPrerequisites(s1, "b", true)
	p, _ := e.repo.GetProgress(s1, "a")
	if p.PlacementSeeded || p.MasteredAt == nil || !p.LastReviewed.Equal(then) {
		t.Fatalf("demonstrated mastery must be untouched, got %+v", p)
	}
}

func TestCreditPrerequisites_CorrectAnswerRefreshesCredit(t *testing.T) {
	e := testEngine(t)
	s1 := creditStudent(t, e)
	old := time.Now().UTC().Add(-10 * 24 * time.Hour).Truncate(time.Second)
	if err := e.repo.UpsertProgress(&storage.ConceptProgress{
		StudentID: s1, ConceptID: "a", Status: string(mastery.StatusMastered),
		PlacementSeeded: true, LastReviewed: &old, SM2EFactor: 2.5,
	}); err != nil {
		t.Fatal(err)
	}
	e.creditPrerequisites(s1, "b", false)
	p, _ := e.repo.GetProgress(s1, "a")
	if p.LastReviewed == nil || !p.LastReviewed.After(old) {
		t.Fatalf("credited prereq should be refreshed, got %+v", p)
	}
}

func TestPrereqCreditWeights_DecayEncompassAndCutoff(t *testing.T) {
	th := concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 60}
	d, err := concepts.Build([]concepts.Concept{
		{ID: "p1", Label: "P1", Domain: "d", GradingType: "numeric", Prerequisites: []string{}, MasteryThreshold: th},
		{ID: "p2", Label: "P2", Domain: "d", GradingType: "numeric", Prerequisites: []string{"p1"}, MasteryThreshold: th},
		{ID: "p3", Label: "P3", Domain: "d", GradingType: "numeric", Prerequisites: []string{"p2"}, MasteryThreshold: th},
		{ID: "p4", Label: "P4", Domain: "d", GradingType: "numeric", Prerequisites: []string{"p3"}, MasteryThreshold: th},
		{ID: "top", Label: "Top", Domain: "d", GradingType: "numeric", Prerequisites: []string{"p4", "p2"}, Encompasses: []string{"p2"}, MasteryThreshold: th},
	})
	if err != nil {
		t.Fatal(err)
	}
	e := &Engine{dag: d}
	w := e.prereqCreditWeights("top")
	// p2 is encompassed (weight 1), so p1 below it gets 1/2 via that path,
	// stronger than via p4 -> p3 -> p2 -> p1.
	want := map[string]float64{"p4": 0.5, "p3": 0.25, "p2": 1, "p1": 0.5}
	for id, v := range want {
		if w[id] != v {
			t.Errorf("%s weight = %v, want %v (all: %v)", id, w[id], v, w)
		}
	}
	// A plain chain falls below the cutoff after three links.
	w = e.prereqCreditWeights("p4")
	if w["p3"] != 0.5 || w["p2"] != 0.25 || w["p1"] != 0.125 {
		t.Errorf("chain weights wrong: %v", w)
	}
	if _, ok := e.prereqCreditWeights("top")["missing"]; ok {
		t.Error("unexpected entry")
	}
}
