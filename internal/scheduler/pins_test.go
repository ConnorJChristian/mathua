package scheduler

import (
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/mastery"
)

// Fork change: this week's pins (miniDAG: a -> b, a -> c, b+c -> d).

func TestRecommend_PinnedTopicLeadsNonReviewWork(t *testing.T) {
	s := New(miniDAG(t))
	now := time.Now()
	got := s.Recommend(RecommendInput{
		Now: now, AttemptedDays: 10, ConceptsMastered: 1, DiagnosticCompleted: true,
		Snapshots: map[string]*ConceptSnapshot{"a": masteredAt(now)},
		Weakness:  map[string]float64{},
		Pinned:    map[string]bool{"c": true},
	})
	if got.Primary == nil || got.Primary.ConceptID != "c" {
		t.Fatalf("primary = %+v, want the pinned topic c", got.Primary)
	}
	if !got.Primary.Pinned || got.Primary.Badge != "This week" {
		t.Errorf("pinned topic not marked: %+v", got.Primary)
	}
}

func TestRecommend_LockedPinIsRepresentedByItsOpenPrerequisites(t *testing.T) {
	s := New(miniDAG(t))
	now := time.Now()
	got := s.Recommend(RecommendInput{
		Now: now, AttemptedDays: 10, ConceptsMastered: 1, DiagnosticCompleted: true,
		Snapshots: map[string]*ConceptSnapshot{"a": masteredAt(now)},
		Weakness:  map[string]float64{},
		Pinned:    map[string]bool{"d": true}, // b and c are open; d is not
	})
	rs := all(got)
	if len(rs) < 2 || !rs[0].Pinned || !rs[1].Pinned {
		t.Fatalf("want b and c offered as this week's work for d, got %+v", rs)
	}
	for _, r := range rs[:2] {
		if r.ConceptID != "b" && r.ConceptID != "c" {
			t.Errorf("unexpected pinned stand-in %q", r.ConceptID)
		}
		if r.PinnedFor == "" {
			t.Errorf("%q should say which pinned topic it is for", r.ConceptID)
		}
	}
}

func TestRecommend_DueReviewStillOutranksPins(t *testing.T) {
	s := New(miniDAG(t))
	now := time.Now()
	past := now.Add(-time.Hour)
	got := s.Recommend(RecommendInput{
		Now: now, AttemptedDays: 10, ConceptsMastered: 1, DiagnosticCompleted: true, DueReviews: 1,
		Snapshots: map[string]*ConceptSnapshot{
			"a": {Status: masteredAt(now).Status, LastAttempted: past, LastReviewed: past, NextReviewDue: &past},
		},
		Weakness: map[string]float64{},
		Pinned:   map[string]bool{"b": true},
	})
	if got.Primary == nil || got.Primary.Kind != KindReview {
		t.Fatalf("primary = %+v, want the due review ahead of the pin", got.Primary)
	}
	if len(got.Alternatives) == 0 || got.Alternatives[0].ConceptID != "b" {
		t.Errorf("pinned b should follow the review, got %+v", got.Alternatives)
	}
}

func TestRecommend_PlacementSeededRowsAreNotOfferedAsWork(t *testing.T) {
	s := New(miniDAG(t))
	now := time.Now()
	got := s.Recommend(RecommendInput{
		Now: now, AttemptedDays: 1, DiagnosticCompleted: true,
		Snapshots: map[string]*ConceptSnapshot{
			// a: answered right in the diagnostic (seeded, never practised); b, c open because of it.
			"a": {Status: mastery.StatusLearning, LastAttempted: now, Placement: true, Attempts: 0},
		},
		Weakness: map[string]float64{},
	})
	for _, r := range all(got) {
		if r.ConceptID == "a" {
			t.Fatalf("placement-seeded, never-practised a was offered: %+v", r)
		}
	}
	if got.Primary == nil || (got.Primary.ConceptID != "b" && got.Primary.ConceptID != "c") {
		t.Fatalf("want b or c (opened by the placement), got %+v", got.Primary)
	}
}

func TestRecommend_MasteredPinDoesNotPullInItsPrerequisites(t *testing.T) {
	s := New(miniDAG(t))
	now := time.Now()
	got := s.Recommend(RecommendInput{
		Now: now, AttemptedDays: 5, ConceptsMastered: 1, DiagnosticCompleted: true,
		Snapshots: map[string]*ConceptSnapshot{
			"a": {Status: mastery.StatusLearning, LastAttempted: now, Placement: true}, // seeded, open
			"b": masteredAt(now),                                                       // the pin, done
		},
		Weakness: map[string]float64{},
		Pinned:   map[string]bool{"b": true},
	})
	for _, r := range all(got) {
		if r.Pinned {
			t.Errorf("a mastered pin produced a 'This week' item: %+v", r)
		}
	}
}

// Q45: a locked pin is represented by its open prerequisites, but not by ones
// the placement already put the learner above, nor anything below them.
// miniDAG: a -> b, a -> c, b+c -> d. a is placed (seeded, never practised),
// b is open and unstarted, c is mastered; pinning d offers b, never a.
func TestRecommend_LockedPinSkipsPlacedPrerequisites(t *testing.T) {
	s := New(miniDAG(t))
	now := time.Now()
	got := s.Recommend(RecommendInput{
		Now: now, AttemptedDays: 5, ConceptsMastered: 1, DiagnosticCompleted: true,
		Snapshots: map[string]*ConceptSnapshot{
			"a": {Status: mastery.StatusLearning, LastAttempted: now, Placement: true},
			"c": masteredAt(now),
		},
		Weakness: map[string]float64{},
		Pinned:   map[string]bool{"d": true},
	})
	for _, r := range all(got) {
		if r.ConceptID == "a" {
			t.Errorf("placed prerequisite a was offered: %+v", r)
		}
	}
	found := false
	for _, r := range all(got) {
		found = found || (r.ConceptID == "b" && r.Pinned)
	}
	if !found {
		t.Errorf("b, the open step toward d, should be this week's work: %+v", all(got))
	}
}
