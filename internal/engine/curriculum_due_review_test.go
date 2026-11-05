package engine

// Phase 1 independent reviewer (F1 follow-up): the Domains topic list
// (CurriculumDomain, rendered by app/domains/page.tsx as "Due for review")
// and the readiness payload (EligibilityFor, rendered by the Study page and
// PrerequisitePanel as "Due a check") take their `due_for_review` state
// from scheduler.EligibilityOf, which reads IsDue without the notReviewed
// exclusions. A practised foundation below the review floor is excluded
// from every count and from the review session, yet these two surfaces
// still call it due.

import (
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/storage"
)

func TestReviewer_CurriculumStateAgreesWithDueRule(t *testing.T) {
	e, sid := creditOnlyFixture(t)
	past := time.Now().Add(-time.Hour)
	p := storage.ConceptProgress{StudentID: sid, ConceptID: "f2", Status: "MASTERED", Attempts: 9, SM2EFactor: 2.5,
		LastReviewed: &past, LastAttempted: &past, NextReviewDue: &past}
	if err := e.repo.UpsertProgress(&p); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MATHUA_REVIEW_FLOOR_DOMAINS", "arithmetic, fractions")
	if n, _ := e.ReviewsDue(sid); n != 0 {
		t.Fatalf("setup: ReviewsDue = %d, want 0 (f2 is below the floor)", n)
	}
	progress, _ := e.repo.GetAllProgress(sid)
	if e.DueReviewSet(sid, progress, time.Now())["f2"] {
		t.Fatal("setup: f2 in the due set")
	}
	dom := e.dag.Concept("f2").Domain
	cur, err := e.CurriculumDomain(sid, dom)
	if err != nil {
		t.Fatal(err)
	}
	for _, tp := range cur.Topics {
		if tp.ConceptID == "f2" && tp.State == "due_for_review" {
			t.Errorf("Domains topic list: f2 state %q, but the due rule (count, session) excludes it", tp.State)
		}
	}
	if el, err := e.EligibilityFor(sid, "f2"); err == nil && el != nil && el.State == "due_for_review" {
		t.Errorf("readiness payload: f2 state %q, but the due rule excludes it", el.State)
	}
}
