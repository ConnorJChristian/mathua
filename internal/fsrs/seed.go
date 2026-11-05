package fsrs

import (
	"fmt"
	"math"
	"time"
)

// Seeded is the FSRS state that corresponds to an existing SM-2 schedule.
type Seeded struct {
	// Stability is the seeded stability, in days.
	Stability float64
	// Due is the due date the SM-2 schedule already holds.
	Due time.Time
	// AlreadyDue is true when Due is at or before the seeding moment.
	AlreadyDue bool
}

// SeedFromInterval converts an existing SM-2 interval into FSRS state.
//
// # Why the interval is the stability
//
// FSRS defines stability as the interval at which recall is 90 percent, so
// taking S = interval places the concept's existing due date exactly at
// R = 0.9. That is what makes the conversion agree with the due date the
// learner already has, instead of silently moving it.
//
// It is an approximation, and it must be described as one. SM-2's interval is
// not a measured 90 percent recall probability: it is a ladder (1 day, 6 days,
// then interval x easiness) that was never calibrated. Seeding therefore says
// "start the FSRS clock where the expected review is", not "this learner has
// 90 percent recall at this date". The difference matters most on the first
// review after seeding, where one Good multiplies the interval by roughly 2.5
// to 4 times; a shadow measurement must report that first answer separately.
//
// # Eligibility
//
// A row is eligible to be seeded when its interval is a whole number of days in
// [1, MaxIntervalDays] and its due date is set. An interval of zero means the
// concept has never been scheduled and seeding it would invent a due date;
// a negative or non-finite interval is corrupt data; an interval beyond the
// maximum is clamped by the caller's own policy, not silently accepted here.
//
// A due date at or before now is not an error. It is the normal state of an
// overdue review, and it is recorded rather than corrected: the backfill must
// not move a learner's overdue concept into the future.
//
// # Backfill shape (not implemented in this stage)
//
// A backfill over existing rows should be idempotent and resumable:
//
//   - select only rows whose FSRS state is still empty, ordered by
//     (student_id, concept_id), so a run that stops halfway resumes at the
//     first unseeded row;
//   - compute one row at a time and write the whole seed for that row in one
//     statement, so a crash cannot leave a stability without its due date;
//   - never touch sm2_*, next_review_due, xp_total, streak or mastery;
//   - skip rows that fail validation and count them, rather than guessing a
//     plausible interval;
//   - record the run under the existing schema_migrations marker so a second
//     run does not repeat work, and make the row-level update conditional on
//     the FSRS state still being empty so a concurrent run cannot overwrite a
//     row that has already been reviewed.
func SeedFromInterval(intervalDays int, due, now time.Time) (Seeded, error) {
	if intervalDays < 1 {
		return Seeded{}, fmt.Errorf("fsrs: cannot seed from interval %d days: no schedule to convert", intervalDays)
	}
	if intervalDays > MaxIntervalDays {
		return Seeded{}, fmt.Errorf("fsrs: cannot seed from interval %d days: beyond the %d-day maximum", intervalDays, MaxIntervalDays)
	}
	if due.IsZero() {
		return Seeded{}, fmt.Errorf("fsrs: cannot seed without a due date")
	}
	stability := clampStabilityMin(float64(intervalDays))
	if math.IsNaN(stability) {
		return Seeded{}, fmt.Errorf("fsrs: cannot seed from a non-finite interval")
	}
	return Seeded{
		Stability:  stability,
		Due:        due,
		AlreadyDue: !due.After(now),
	}, nil
}
