package engine

import (
	"log"
	"sort"

	"github.com/chuma-beep/mathua/internal/mastery"
	"github.com/chuma-beep/mathua/internal/storage"
)

// Fork change: partial prerequisite credit, in the spirit of Math Academy's
// fractional implicit repetition.
//
// Working on an advanced concept exercises the concepts below it, but not all
// of them equally. Each prerequisite link carries a weight: 1 when the concept
// explicitly encompasses the prerequisite (its "encompasses" list: doing it
// means doing all of the prerequisite), and creditPrereqWeight for an ordinary
// prerequisite link. Credit flowing down a chain multiplies the weights, so a
// direct prerequisite receives 1/2, one two links down 1/4, and anything whose
// strongest path falls below creditMinWeight receives nothing.
//
// When a learner masters a concept, each concept below it gains that weight
// in ImplicitCredit. Credit accumulates across the concepts above it: once it
// reaches 1 the concept is credited, written as MASTERED with PlacementSeeded
// set and no MasteredAt. That combination is the "credited" marker: the
// diagnostic's placement rows are never MASTERED, and demonstrated mastery
// always stamps MasteredAt, so credited rows stay distinguishable from both. A
// credited concept counts as done for planning and unlocks its successors.
//
// Credit is kept fresh the same way, in proportion: a correct answer on a
// concept refreshes LastReviewed on credited concepts below it only when the
// link is strong enough (weight at least creditReviewWeight). Weakly linked
// credit decays into review unless the learner works on something that
// exercises it properly.
//
// Demonstrated progress is never touched: a concept the learner mastered
// themselves keeps its own review schedule.
const (
	creditPrereqWeight = 0.5
	creditMinWeight    = 0.1
	creditReviewWeight = 0.5
	creditFull         = 1.0
)

func (e *Engine) creditPrerequisites(studentID, conceptID string, mastered bool) {
	weights := e.prereqCreditWeights(conceptID)
	if len(weights) == 0 {
		return
	}
	progress, err := e.repo.GetAllProgress(studentID)
	if err != nil {
		log.Printf("warning: prerequisite credit for %s/%s: %v", studentID, conceptID, err)
		return
	}
	ids := make([]string, 0, len(weights))
	for id := range weights {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	now := nowUTC()
	batch := make([]*storage.ConceptProgress, 0, len(ids))
	for _, id := range ids {
		w := weights[id]
		p := progress[id]
		if p != nil {
			// GetAllProgress keys rows by concept; make the identity explicit for the upsert.
			p.StudentID, p.ConceptID = studentID, id
		}
		switch {
		case isCredited(p):
			// Implicit repetition: strong links keep credited concepts fresh.
			if w >= creditReviewWeight {
				p.LastReviewed = &now
				batch = append(batch, p)
			}
		case !mastered:
			// Only mastery grants new credit; an ordinary correct answer just refreshes.
		case p != nil && p.Status == string(mastery.StatusMastered):
			// Demonstrated (or placement) mastery: leave its schedule alone.
		default:
			if p == nil {
				p = &storage.ConceptProgress{
					StudentID:  studentID,
					ConceptID:  id,
					Status:     string(mastery.StatusUnseen),
					SM2EFactor: 2.5,
				}
			}
			p.ImplicitCredit += w
			if p.ImplicitCredit >= creditFull-1e-9 {
				p.ImplicitCredit = creditFull
				p.Status = string(mastery.StatusMastered)
				p.WeaknessScore = 0
				p.LastReviewed = &now
				p.NextReviewDue = nil
				p.PlacementSeeded = true
			}
			batch = append(batch, p)
		}
	}
	if len(batch) == 0 {
		return
	}
	if err := e.repo.UpsertProgressBatch(batch); err != nil {
		log.Printf("warning: save prerequisite credit for %s/%s: %v", studentID, conceptID, err)
	}
}

// isCredited reports whether a row is prerequisite credit rather than
// demonstrated mastery or diagnostic placement (see creditPrerequisites).
func isCredited(p *storage.ConceptProgress) bool {
	return p != nil && p.PlacementSeeded && p.MasteredAt == nil &&
		p.Status == string(mastery.StatusMastered)
}

// prereqLinkWeight is the share of `prereq` that working on `c` exercises.
func (e *Engine) prereqLinkWeight(cID, prereqID string) float64 {
	if c := e.dag.Concept(cID); c != nil {
		for _, enc := range c.Encompasses {
			if enc == prereqID {
				return 1
			}
		}
	}
	return creditPrereqWeight
}

// prereqCreditWeights returns, for every concept below id, the weight of its
// strongest path from id (the product of link weights along the path),
// dropping anything below creditMinWeight.
func (e *Engine) prereqCreditWeights(id string) map[string]float64 {
	best := map[string]float64{}
	type item struct {
		id string
		w  float64
	}
	stack := []item{{id, 1}}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, p := range e.dag.PrereqsOf(cur.id) {
			w := cur.w * e.prereqLinkWeight(cur.id, p.ID)
			if w < creditMinWeight || p.ID == id || w <= best[p.ID] {
				continue
			}
			best[p.ID] = w
			stack = append(stack, item{p.ID, w})
		}
	}
	return best
}
