package review

import (
	"fmt"
	"math"
	"time"

	"github.com/chuma-beep/mathua/internal/fsrs"
)

// Event is one review_log row as Replay reads it (storage.ReviewLogEntry
// without the storage import, so this package stays pure).
type Event struct {
	At          time.Time
	Rating      fsrs.Rating
	ElapsedDays float64
	// Source is informational here; every source moves the state the same
	// way (the rating and the elapsed time decide), except "seed", which
	// sets the state the event records (SAfter, DAfter) without a review.
	Source string
	SAfter float64
	DAfter float64
}

// Replay rebuilds a topic's memory from its log, oldest first: the
// event-sourcing check (plan 6.1). Each event is applied as Review would
// have applied it, with the elapsed whole days the event recorded (so a
// clock shift in the daily QA, which moves both the log's At and the
// stored fsrs_last_review, replays the same). A "seed" event sets the
// state outright. The result must equal the stored row exactly; fsrs_doctor
// reports a row that differs.
func Replay(p Params, log []Event) Memory {
	var m Memory
	for _, e := range log {
		if e.Source == "seed" {
			m = Memory{Stability: e.SAfter, Difficulty: e.DAfter, Reps: m.Reps, Lapses: m.Lapses, Phase: PhaseReview}
			if m.Stability < fsrs.StabilityMin {
				m.Stability = fsrs.StabilityMin
			}
			continue
		}
		switch {
		case m.IsZero():
			m = p.Initial(e.Rating)
		case e.ElapsedDays < 1:
			m = p.NextSameDay(m, e.Rating)
		default:
			m = p.Next(m, e.Rating, e.ElapsedDays)
		}
	}
	return m
}

// Diff reports how a stored memory differs from a replayed one, or "" when
// they agree. Floats compare at a relative 1e-9: the stored columns are
// REAL/double precision and the replay recomputes the same float64 path, so
// anything larger is a real divergence (a dropped event, a changed formula
// or a write path that lost state), not rounding.
func Diff(stored, replayed Memory) string {
	rel := func(a, b float64) bool {
		if a == b {
			return true
		}
		s := math.Max(math.Abs(a), math.Abs(b))
		return s > 0 && math.Abs(a-b)/s <= 1e-9
	}
	if stored.IsZero() != replayed.IsZero() {
		return fmt.Sprintf("state: stored %q, replay %q", stored.Phase, replayed.Phase)
	}
	if stored.IsZero() {
		return ""
	}
	switch {
	case !rel(stored.Stability, replayed.Stability):
		return fmt.Sprintf("stability: stored %.12g, replay %.12g", stored.Stability, replayed.Stability)
	case !rel(stored.Difficulty, replayed.Difficulty):
		return fmt.Sprintf("difficulty: stored %.12g, replay %.12g", stored.Difficulty, replayed.Difficulty)
	case stored.Reps != replayed.Reps:
		return fmt.Sprintf("reps: stored %d, replay %d", stored.Reps, replayed.Reps)
	case stored.Lapses != replayed.Lapses:
		return fmt.Sprintf("lapses: stored %d, replay %d", stored.Lapses, replayed.Lapses)
	case stored.Phase != replayed.Phase:
		return fmt.Sprintf("phase: stored %q, replay %q", stored.Phase, replayed.Phase)
	}
	return ""
}
