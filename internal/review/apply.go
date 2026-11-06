package review

import (
	"time"

	"github.com/chuma-beep/mathua/internal/fsrs"
)

// Held reasons: an event logged without changing the state (FSRS Phase 2).
const (
	// HeldDailyCap: a second capped event (one that raises stability, or a
	// correct answer) for the topic on one learner-local day
	// (Rules.DailyCap). The whole event is held: S, D, reps and the last
	// review time all stay.
	HeldDailyCap = "daily_cap"
	// HeldLapseRHigh: a practice miss on a mastered topic while R was at or
	// above Rules.LapseBelowR: recorded, not a lapse.
	HeldLapseRHigh = "lapse_r_high"
)

// Step is one event to apply to a topic's memory.
type Step struct {
	Rating fsrs.Rating
	At     time.Time
	// LapseBelowR > 0 makes this a practice lapse: it applies only when R
	// at the event is below it, and is held otherwise. A topic with no
	// state has no R; its first event is never held for it.
	LapseBelowR float64
	// RaisedToday reports a capped event (Applied.Capped: it raised
	// stability or was a correct answer) already applied to the topic on
	// At's learner-local day; with Cap, a capped event is then held whole.
	// The day is the learner-local calendar day, not 24 hours, so across
	// the DST change an event 24 h later can be the same day or the next.
	RaisedToday bool
	Cap         bool
}

// Applied is the outcome of one Step: the memory after it (the memory
// before when Held is set), the whole days and R the update read, and
// whether it raised stability (a first review counts: it sets S from
// nothing) and whether it counts against the daily cap (Capped: a raise or
// a correct answer; a miss never does).
type Applied struct {
	Next        Memory
	ElapsedDays float64
	R           float64
	Held        string
	Raised      bool
	Capped      bool
}

// Apply runs one event the way shadow and active scheduling commit it: the
// reference review (Review), then the lapse threshold and the daily cap.
// It fails closed on an invalid rating (ErrInvalidRating, memory
// unchanged). The engine, scripts/fsrs_seed and the doctor's replay all
// call it, so the log replays to exactly what was stored.
func (p Params) Apply(m Memory, last time.Time, s Step) (Applied, error) {
	next, elapsed, r, err := p.ReviewChecked(m, s.Rating, last, s.At)
	if err != nil {
		return Applied{Next: m}, err
	}
	out := Applied{Next: next, ElapsedDays: elapsed, R: r}
	if s.LapseBelowR > 0 && !m.IsZero() && !last.IsZero() && r >= s.LapseBelowR {
		return Applied{Next: m, ElapsedDays: elapsed, R: r, Held: HeldLapseRHigh}, nil
	}
	raised := m.IsZero() || next.Stability > m.Stability
	// A second correct answer the same day is held even when the
	// short-term formula would not raise S (large S): otherwise D, reps and
	// the last review time still moved (FSRS Phase 2 review, I2).
	capped := raised || s.Rating != fsrs.Again
	if capped && s.Cap && s.RaisedToday {
		return Applied{Next: m, ElapsedDays: elapsed, R: r, Held: HeldDailyCap}, nil
	}
	out.Raised, out.Capped = raised, capped
	return out, nil
}
