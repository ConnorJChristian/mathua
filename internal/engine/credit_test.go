package engine

import (
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/mastery"
	"github.com/chuma-beep/mathua/internal/storage"
)

// Fork change tests: prerequisite credit (credit.go). testEngine's graph is a <- b.

func TestCreditPrerequisites_MasteryCreditsUnseenPrereq(t *testing.T) {
	e := testEngine(t)
	e.creditPrerequisites("s1", "b", true)

	p, err := e.repo.GetProgress("s1", "a")
	if err != nil || p == nil {
		t.Fatalf("prereq row missing: %v", err)
	}
	if p.Status != string(mastery.StatusMastered) || !p.PlacementSeeded || p.MasteredAt != nil {
		t.Fatalf("want credited (MASTERED, placement, no MasteredAt), got %+v", p)
	}
	if p.LastReviewed == nil {
		t.Fatal("credited prereq should start its decay clock")
	}
	if !isCredited(p) {
		t.Fatal("isCredited should recognise the row")
	}
	if el, err := e.EligibilityFor("s1", "b"); err != nil || !el.Eligible {
		t.Fatalf("b should be eligible once a is credited: %+v %v", el, err)
	}
}

func TestCreditPrerequisites_CorrectAnswerWithoutMasteryGrantsNothing(t *testing.T) {
	e := testEngine(t)
	e.creditPrerequisites("s1", "b", false)
	if p, _ := e.repo.GetProgress("s1", "a"); p != nil && p.Status == string(mastery.StatusMastered) {
		t.Fatalf("no credit expected before mastery, got %+v", p)
	}
}

func TestCreditPrerequisites_LeavesDemonstratedMasteryAlone(t *testing.T) {
	e := testEngine(t)
	then := time.Now().UTC().Add(-48 * time.Hour).Truncate(time.Second)
	due := then.AddDate(0, 0, 6)
	if err := e.repo.UpsertProgress(&storage.ConceptProgress{
		StudentID: "s1", ConceptID: "a", Status: string(mastery.StatusMastered),
		MasteredAt: &then, LastReviewed: &then, NextReviewDue: &due, SM2EFactor: 2.5,
	}); err != nil {
		t.Fatal(err)
	}
	e.creditPrerequisites("s1", "b", true)
	p, _ := e.repo.GetProgress("s1", "a")
	if p.PlacementSeeded || p.MasteredAt == nil || !p.LastReviewed.Equal(then) {
		t.Fatalf("demonstrated mastery must be untouched, got %+v", p)
	}
}

func TestCreditPrerequisites_CorrectAnswerRefreshesCredit(t *testing.T) {
	e := testEngine(t)
	old := time.Now().UTC().Add(-10 * 24 * time.Hour).Truncate(time.Second)
	if err := e.repo.UpsertProgress(&storage.ConceptProgress{
		StudentID: "s1", ConceptID: "a", Status: string(mastery.StatusMastered),
		PlacementSeeded: true, LastReviewed: &old, SM2EFactor: 2.5,
	}); err != nil {
		t.Fatal(err)
	}
	e.creditPrerequisites("s1", "b", false)
	p, _ := e.repo.GetProgress("s1", "a")
	if p.LastReviewed == nil || !p.LastReviewed.After(old) {
		t.Fatalf("credited prereq should be refreshed, got %+v", p)
	}
}
