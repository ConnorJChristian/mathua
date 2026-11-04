// Package review holds the one rule for what is due for review (fork
// change, FSRS plan Phase 0).
//
// Two rules used to disagree. scheduler.effectiveState served any mastered
// topic 14 days after its last review, with or without a scheduled date,
// so the review session and the mixed session offered it. The due count,
// the Study banner, the Up next review card and the "review due N days
// ago" reasons counted only rows with a past next_review_due. A credited
// course topic (mastered through the topics above it, so it has no date)
// was served by /review from its fifteenth day while the count showed 0.
//
// Every surface now reads IsDue. The rule, written in FORK.md ("What is
// due"):
//
//   - Only a stored MASTERED row can be due. LEARNING and PRACTICING rows
//     carry a next_review_due too (practice sets one), but that is a
//     practice date, not a review.
//   - With a scheduled date, the date decides: due once it has passed,
//     not before. A long SM-2 interval is honoured; decay no longer caps it
//     at DecayDays in the review session.
//   - Without a date (a credited row, or a legacy one), the row is due
//     DecayDays after its last review (its last attempt when it has no
//     review; at once when it has neither).
//
// What a learner is not reviewed on at all (topics below their floor,
// credit-only foundations no followed course lists, unless pinned) is the
// engine's notReviewed set, applied by every surface beside IsDue. It is
// not part of the rule here: this package knows nothing about courses.
//
// Decay for pacing, plan order and the progress displays
// (mastery.EffectiveStatus) is a separate notion and is unchanged: a
// mastered topic reads DECAYING DecayDays after its last review whatever
// its date says. With a date longer than DecayDays, a topic can be decayed
// for pacing before it is due for review.
package review

import (
	"time"

	"github.com/chuma-beep/mathua/internal/mastery"
)

// Row is the part of a learner's progress on one topic that the rule
// reads. Zero times mean never.
type Row struct {
	Status        mastery.Status
	LastReviewed  time.Time
	LastAttempted time.Time
	NextReviewDue *time.Time
}

// DueAt is when the row is or was due: its scheduled date, else DecayDays
// after its last review (or last attempt). ok is false for a row that is
// not mastered, or a mastered row with no date and no history, which is
// due at once (IsDue) but has no date to show.
func DueAt(r Row) (at time.Time, ok bool) {
	if r.Status != mastery.StatusMastered {
		return time.Time{}, false
	}
	if r.NextReviewDue != nil {
		return *r.NextReviewDue, true
	}
	anchor := r.LastReviewed
	if anchor.IsZero() {
		anchor = r.LastAttempted
	}
	if anchor.IsZero() {
		return time.Time{}, false
	}
	return anchor.Add(mastery.DecayDays * 24 * time.Hour), true
}

// IsDue reports whether the row is due for review at now.
func IsDue(r Row, now time.Time) bool {
	if r.Status != mastery.StatusMastered {
		return false
	}
	at, ok := DueAt(r)
	if !ok {
		// Mastered, undated and with no history: nothing says when it was
		// last seen, so it is due (the rule the scheduler always applied to
		// a mastered row with no last review).
		return true
	}
	return !at.After(now)
}

// Overdue is how many days past its due date the row is at now: 0 when it
// is not due. It orders the review queue (most overdue first) and words the
// reasons ("review due 3 days ago").
func Overdue(r Row, now time.Time) float64 {
	if !IsDue(r, now) {
		return 0
	}
	at, ok := DueAt(r)
	if !ok {
		return 0
	}
	d := now.Sub(at).Hours() / 24
	if d < 0 {
		return 0
	}
	return d
}
