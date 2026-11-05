package engine

// FSRS Phase 2: fixes for the independent review's findings (M2, L1, L5,
// I2, I3, L2). Each test failed before its fix.

import (
	"errors"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/review"
	"github.com/chuma-beep/mathua/internal/storage"
)

// logFailRepo fails ReviewLogFor, so the shadow event cannot be computed.
type logFailRepo struct{ storage.Repository }

func (r *logFailRepo) ReviewLogFor(string, string) ([]storage.ReviewLogEntry, error) {
	return nil, errors.New("injected: review_log read failed")
}

// M2: a dropped shadow event leaves an fsrs_errors row, on both failure
// paths (not computed, not written), naming the learner, the topic and the
// committed attempt; the answer itself is committed.
func TestFix2_DroppedEventIsRecorded(t *testing.T) {
	for _, tc := range []struct {
		stage string
		wrap  func(storage.Repository) storage.Repository
	}{
		{"written", func(r storage.Repository) storage.Repository { return &commitFailRepo{Repository: r} }},
		{"computed", func(r storage.Repository) storage.Repository { return &logFailRepo{Repository: r} }},
	} {
		e, sid, sim := rvEngine(t, review.ModeShadow, rvMonday)
		answerCorrectly(t, e, sid, 6)
		sim.Advance(3 * 24 * time.Hour)
		n0 := rvAttempts(t, e, sid)
		inner := e.repo
		e.repo = tc.wrap(inner)
		res, _, err := rvReview(t, e, sid, "a", "42", "7", "r1")
		e.repo = inner
		if err != nil || res == nil {
			t.Fatalf("%s: the answer failed: %v", tc.stage, err)
		}
		atts, _ := e.repo.GetAttemptsForStudent(sid)
		if len(atts) != n0+1 {
			t.Fatalf("%s: attempts %d -> %d", tc.stage, n0, len(atts))
		}
		rows, err := e.repo.FSRSErrors(sid)
		if err != nil {
			t.Fatal(err)
		}
		last := atts[len(atts)-1]
		for _, a := range atts {
			if a.ID > last.ID {
				last = a
			}
		}
		if len(rows) != 1 || rows[0].Stage != tc.stage || rows[0].ConceptID != "a" || rows[0].StudentID != sid || rows[0].AttemptID != last.ID || rows[0].Error == "" {
			t.Errorf("%s: fsrs_errors %+v (attempt %d)", tc.stage, rows, last.ID)
		}
		if n := len(rvLog(t, e, sid, "a")); n != 1 {
			t.Errorf("%s: %d log rows, want the first mastery only", tc.stage, n)
		}
	}
	// A clean answer records nothing.
	e, sid, sim := rvEngine(t, review.ModeShadow, rvMonday)
	answerCorrectly(t, e, sid, 6)
	sim.Advance(3 * 24 * time.Hour)
	mustOK3(t)(rvReview(t, e, sid, "a", "42", "7", "r1"))
	if rows, _ := e.repo.FSRSErrors(""); len(rows) != 0 {
		t.Errorf("a clean answer wrote fsrs_errors %+v", rows)
	}
}
