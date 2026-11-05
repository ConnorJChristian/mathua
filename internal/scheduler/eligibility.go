package scheduler

import (
	"time"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/mastery"
)

// Concept states, as the learner sees them. These are derived from the DAG and
// the learner's snapshots; nothing stores them, so they cannot drift from the
// scheduler that actually decides what is reachable.
const (
	StateMastered     = "mastered"       // demonstrated
	StateDueForReview = "due_for_review" // demonstrated, retrieval may have weakened
	StateInProgress   = "in_progress"    // started, not yet demonstrated
	StateUnlocked     = "unlocked"       // not started, everything it needs is done
	StateUpcoming     = "upcoming"       // prerequisites started but not yet demonstrated
	StateLocked       = "locked"         // at least one prerequisite is untouched
)

// Eligibility is the single server-side answer to "are this concept's
// prerequisites satisfied, and if not, which are missing". It reuses
// prereqsMet/effectiveState rather than restating them: a second copy of the
// prerequisite rule is exactly how the browser and the engine came to disagree
// before (ADR-042).
//
// Eligible is about prerequisites only. Whether to *offer* the learning loop is
// the State's job: a mastered concept is eligible but should route to review,
// not teaching (ADR-037, ADR-047).
type Eligibility struct {
	State    string
	Eligible bool
	Reason   string
	// Missing lists prerequisite concept ids that are not satisfied. Empty when
	// Eligible.
	Missing []string
}

// EligibilityOf decides whether c's prerequisites are met and renders the
// learner-facing state. Placement-seeded prerequisites count as reachability,
// not mastery.
//
// Its "due_for_review" reads the snapshot's IsDue alone, without the
// engine's exclusions (credit-only off-course, below the review floor), so a
// learner-facing surface must use EligibilityWithDue with the engine's one
// due rule (engine.DueReviewSet) instead. EligibilityOf is kept for callers
// that read only Eligible/Missing and for the scheduler's own tests.
func EligibilityOf(snapshots map[string]*ConceptSnapshot, now time.Time, c *concepts.Concept) Eligibility {
	return EligibilityWithDue(snapshots, now, c, nil)
}

// EligibilityWithDue is EligibilityOf with the review question answered by
// due: a mastered concept is "due_for_review" exactly when due[c.ID] is set.
// The engine passes DueReviewSet (the one due rule, with its exclusions), so
// the Domains topic list, the readiness payload and the review count agree
// (FSRS Phase 1 review, M1). A nil map falls back to the snapshot's IsDue.
func EligibilityWithDue(snapshots map[string]*ConceptSnapshot, now time.Time, c *concepts.Concept, due map[string]bool) Eligibility {
	snap := snapshots[c.ID]

	met := true
	allStarted := true
	missing := make([]string, 0, len(c.Prerequisites))
	for _, pid := range c.Prerequisites {
		ps := snapshots[pid]
		if ps == nil {
			met = false
			allStarted = false
			missing = append(missing, pid)
			continue
		}
		if !ps.Placement {
			pstatus := mastery.EffectiveStatus(ps.Status, daysSince(ps.LastReviewed), mastery.DecayDays)
			if pstatus != mastery.StatusMastered && pstatus != "DECAYING" {
				met = false
				missing = append(missing, pid)
			}
		}
		if ps.Status == mastery.StatusUnseen {
			allStarted = false
		}
	}

	state := StateLocked
	switch {
	case snap != nil && snap.Status == mastery.StatusMastered:
		// Decay is a review question, not a teaching one (ADR-037).
		isReview := due[c.ID]
		if due == nil {
			_, isReview = effectiveState(snap, now)
		}
		if isReview {
			state = StateDueForReview
		} else {
			state = StateMastered
		}
	case snap != nil && (snap.Status == mastery.StatusLearning || snap.Status == mastery.StatusPracticing):
		state = StateInProgress
	case met:
		state = StateUnlocked
	case allStarted:
		state = StateUpcoming
	}

	reason := ""
	if !met {
		reason = "missing_prerequisites"
	}
	return Eligibility{State: state, Eligible: met, Reason: reason, Missing: missing}
}
