package engine

import (
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/clock"
	"github.com/chuma-beep/mathua/internal/mastery"
	"github.com/chuma-beep/mathua/internal/storage"
)

// The engine reads the settable clock (internal/clock), not time.Now, on the
// scheduling paths (FSRS Phase 2, step 1): the due count, the review
// session's candidates and the timestamps an answer writes all follow a
// simulated clock.
func TestEngineReadsTheClock(t *testing.T) {
	e := testEngine(t)
	sid := creditStudent(t, e)
	due := time.Date(2030, 3, 1, 12, 0, 0, 0, time.UTC)
	last := due.AddDate(0, 0, -3)
	if err := e.repo.UpsertProgress(&storage.ConceptProgress{
		StudentID: sid, ConceptID: "a", Status: string(mastery.StatusMastered), Attempts: 4, Streak: 3,
		LastAttempted: &last, LastReviewed: &last, MasteredAt: &last, NextReviewDue: &due, SM2EFactor: 2.5, SM2Interval: 3,
	}); err != nil {
		t.Fatal(err)
	}
	sim := clock.NewSim(due.Add(-time.Hour))
	defer sim.Install()()
	if n, err := e.ReviewsDue(sid); err != nil || n != 0 {
		t.Fatalf("an hour before the date: %d due (%v), want 0", n, err)
	}
	progress, _ := e.repo.GetAllProgress(sid)
	if c := e.ReviewCandidates(sid, progress, clock.Now()); len(c) != 0 {
		t.Fatalf("candidates before the date: %v", c)
	}
	sim.Set(due.Add(time.Hour))
	if n, err := e.ReviewsDue(sid); err != nil || n != 1 {
		t.Fatalf("an hour after the date: %d due (%v), want 1", n, err)
	}
	// An answer is stamped with the clock's time.
	e.SetStudyAnchor(sid, "b", "99")
	if _, err := e.SubmitStudyAnswer(sid, "b", "99", 2.0, ""); err != nil {
		t.Fatal(err)
	}
	p, err := e.repo.GetProgress(sid, "b")
	if err != nil || p == nil || p.LastAttempted == nil {
		t.Fatalf("progress b: %+v %v", p, err)
	}
	if !p.LastAttempted.Equal(clock.Now().Truncate(time.Second)) {
		t.Errorf("last_attempted %v, want the clock's %v", p.LastAttempted, clock.Now())
	}
	atts, err := e.repo.GetAttemptsForStudent(sid)
	if err != nil || len(atts) != 1 || !atts[0].Timestamp.Equal(clock.Now().Truncate(time.Second)) {
		t.Errorf("attempts %+v (%v), want one stamped %v", atts, err, clock.Now())
	}
}
