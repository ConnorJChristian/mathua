package scheduler

import (
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/mastery"
)

// Independent review of FSRS Phase 0: the scheduler (the review session's
// and the mixed session's candidate builder) reads the one rule and nothing
// else.

func tp(t time.Time) *time.Time { return &t }

// A mastered topic reviewed 20 days ago with a 37-day SM-2 date is decayed
// for pacing but NOT due: the review session must not serve it, and the
// mixed session must not offer it as new work either (it is mastered).
func TestSchedulerReview_DecayedWithFutureDateNotServed(t *testing.T) {
	d := miniDAG(t)
	s := New(d)
	now := time.Now()
	old := now.AddDate(0, 0, -20)
	snaps := map[string]*ConceptSnapshot{
		"a": {Status: mastery.StatusMastered, LastReviewed: old, LastAttempted: old, NextReviewDue: tp(now.AddDate(0, 0, 37)), Attempts: 9},
	}
	if st, isReview := effectiveState(snaps["a"], now); st != "DECAYING" || isReview {
		t.Fatalf("effectiveState = %v, %v; want DECAYING for pacing and not due", st, isReview)
	}
	if next := s.NextReviewSkipping(snaps, "", nil); next != nil {
		t.Fatalf("review session served %s, which is not due", next.Concept.ID)
	}
	for _, c := range buildCandidates(d, snaps, now, "") {
		if c.Concept.ID == "a" {
			t.Fatalf("a is a candidate (review=%v) while mastered and not due", c.IsReview)
		}
	}
	// Its dependents are still unlocked: DECAYING satisfies a prerequisite.
	el := EligibilityOf(snaps, now, d.Concept("b"))
	if !el.Eligible {
		t.Errorf("b locked behind a decayed-but-not-due prerequisite: %+v", el)
	}
	// And its own state is "mastered", not "due_for_review".
	if st := EligibilityOf(snaps, now, d.Concept("a")).State; st != StateMastered {
		t.Errorf("a's state = %s, want %s (decayed is not due)", st, StateMastered)
	}
}

// A LEARNING row with a past next_review_due (a practice date) is never a
// review candidate, whatever its date says.
func TestSchedulerReview_LearningWithPastDateIsNotAReview(t *testing.T) {
	d := miniDAG(t)
	s := New(d)
	now := time.Now()
	snaps := map[string]*ConceptSnapshot{
		"a": {Status: mastery.StatusLearning, LastReviewed: now.AddDate(0, 0, -3), LastAttempted: now.AddDate(0, 0, -3), NextReviewDue: tp(now.AddDate(0, 0, -2)), Attempts: 2},
	}
	if next := s.NextReviewSkipping(snaps, "", nil); next != nil {
		t.Fatalf("review session served learning topic %s", next.Concept.ID)
	}
	for _, c := range buildCandidates(d, snaps, now, "") {
		if c.Concept.ID == "a" && c.IsReview {
			t.Fatalf("learning topic a is a review candidate")
		}
	}
	if st := EligibilityOf(snaps, now, d.Concept("a")).State; st != StateInProgress {
		t.Errorf("a's state = %s, want in_progress", st)
	}
}

// A due review is served even when a prerequisite reads unmet (the count
// does not look at prerequisites, so neither may the session), and the
// most overdue comes first across the set.
func TestSchedulerReview_DueReviewServedWhateverPrereqsRead(t *testing.T) {
	d := miniDAG(t)
	s := New(d)
	now := time.Now()
	snaps := map[string]*ConceptSnapshot{
		"a": {Status: mastery.StatusUnseen},
		"b": {Status: mastery.StatusLearning, LastAttempted: now.AddDate(0, 0, -1), Attempts: 1},
		// d rests on b and c; both unmet, yet d is mastered and 3 days overdue.
		"d": {Status: mastery.StatusMastered, LastReviewed: now.AddDate(0, 0, -9), LastAttempted: now.AddDate(0, 0, -9), NextReviewDue: tp(now.AddDate(0, 0, -3)), Attempts: 5},
		// c: credited, undated, 20 days: 6 days past its horizon, most overdue.
		"c": {Status: mastery.StatusMastered, LastReviewed: now.AddDate(0, 0, -20), Placement: true},
	}
	seen := map[string]bool{}
	prev := ""
	var order []string
	for i := 0; i < 4; i++ {
		next := s.NextReviewSkipping(snaps, prev, seen)
		if next == nil {
			break
		}
		order = append(order, next.Concept.ID)
		seen[next.Concept.ID] = true
		prev = next.Concept.ID
	}
	if len(order) != 2 || order[0] != "c" || order[1] != "d" {
		t.Fatalf("review order = %v, want [c d] (most overdue first, prerequisites ignored)", order)
	}
}

// The mixed session's Next: a due review outranks new work, and a mastered
// topic that is not due is not offered at all.
func TestSchedulerReview_MixedSessionPrefersDueReview(t *testing.T) {
	d := miniDAG(t)
	s := New(d)
	now := time.Now()
	snaps := map[string]*ConceptSnapshot{
		"a": {Status: mastery.StatusMastered, LastReviewed: now.AddDate(0, 0, -30), LastAttempted: now.AddDate(0, 0, -30), NextReviewDue: tp(now.AddDate(0, 0, -1)), Attempts: 6},
	}
	next := s.Next(snaps, "", 0, 0)
	if next == nil || next.Concept.ID != "a" || !next.IsReview {
		t.Fatalf("mixed session next = %+v, want a as a review", next)
	}
	snaps["a"].NextReviewDue = tp(now.AddDate(0, 0, 10))
	next = s.Next(snaps, "", 0, 0)
	if next != nil && next.Concept.ID == "a" {
		t.Fatalf("mixed session offered a (mastered, decayed, date in 10 days) as review=%v", next.IsReview)
	}
}
