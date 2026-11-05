package engine

// FSRS Phase 2 independent reviewer (finding L-seedrace). A stateless
// mastered row is seeded at its first shadow event with keys
// "seed:a<attempt id>", the same keys scripts/fsrs_seed -apply writes. If
// the script commits those rows between the engine reading the row and
// committing the answer (the script run on the live database while the
// app serves), the answer's commit meets a stored key, is taken for a
// retry, and the learner's answer is dropped: no attempt, no progress, an
// error. Off mode records it. A seed row's key colliding is not a retry of
// this answer.

import (
	"fmt"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/review"
	"github.com/chuma-beep/mathua/internal/storage"
)

// seedRaceRepo commits the script's seed for the topic just before the
// engine's own commit, as a concurrent fsrs_seed -apply would.
type seedRaceRepo struct {
	storage.Repository
	e    *Engine
	sid  string
	done bool
}

func (r *seedRaceRepo) CommitAnswer(c storage.AnswerCommit) (storage.CommitResult, error) {
	if !r.done && c.FSRS != nil && c.Attempt != nil {
		r.done = true
		seeds, _, err := r.e.FSRSSeedPlan(r.sid, c.Attempt.Timestamp.Add(-time.Nanosecond))
		if err != nil {
			return storage.CommitResult{}, err
		}
		r.e.repo = r.Repository
		for _, s := range seeds {
			if _, err := r.e.ApplyFSRSSeed(r.sid, s); err != nil {
				return storage.CommitResult{}, fmt.Errorf("concurrent seed: %w", err)
			}
		}
		r.e.repo = r
	}
	return r.Repository.CommitAnswer(c)
}

func TestReview2_ConcurrentSeedDoesNotDropTheAnswer(t *testing.T) {
	e, sid, sim := rvEngine(t, review.ModeOff, rvMonday)
	answerCorrectly(t, e, sid, 6) // mastered in off mode: no state
	e.SetFSRSMode(review.ModeShadow)
	sim.Advance(3 * 24 * time.Hour)
	n0 := rvAttempts(t, e, sid)
	race := &seedRaceRepo{Repository: e.repo, e: e, sid: sid}
	e.repo = race
	res, _, err := rvReview(t, e, sid, "a", "42", "42", "r1")
	e.repo = race.Repository
	if !race.done {
		t.Fatal("the race did not run")
	}
	if err != nil || res == nil {
		t.Errorf("the answer failed: %v", err)
	}
	if n := rvAttempts(t, e, sid); n != n0+1 {
		t.Errorf("attempts %d -> %d: the review answer was dropped", n0, n)
	}
}
