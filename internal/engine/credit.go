package engine

import (
	"log"

	"github.com/chuma-beep/mathua/internal/mastery"
	"github.com/chuma-beep/mathua/internal/storage"
)

// Fork change: prerequisite credit, in the spirit of Math Academy's implicit
// credit for topics an advanced topic encompasses.
//
// When a learner masters a concept, every prerequisite below it (transitively)
// that is not already MASTERED is credited: it is written as MASTERED with
// PlacementSeeded set and no MasteredAt. That combination is the "credited"
// marker — the diagnostic's placement rows are never MASTERED, and demonstrated
// mastery always stamps MasteredAt — so credited rows stay distinguishable from
// both. A credited concept counts as done for planning and unlocks its
// successors, so the learner is not sent back to drill basics they have shown
// they can use.
//
// Credit is kept fresh the same way: every correct answer on a concept refreshes
// LastReviewed on its credited prerequisites, so they only decay into review if
// the learner stops working above them.
//
// Demonstrated progress is never touched: a concept the learner mastered
// themselves keeps its own review schedule.
func (e *Engine) creditPrerequisites(studentID, conceptID string, mastered bool) {
	ancestors := e.transitivePrereqIDs(conceptID)
	if len(ancestors) == 0 {
		return
	}
	progress, err := e.repo.GetAllProgress(studentID)
	if err != nil {
		log.Printf("warning: prerequisite credit for %s/%s: %v", studentID, conceptID, err)
		return
	}
	now := nowUTC()
	batch := make([]*storage.ConceptProgress, 0, len(ancestors))
	for _, id := range ancestors {
		p := progress[id]
		if p != nil {
			// GetAllProgress keys rows by concept; make the identity explicit for the upsert.
			p.StudentID, p.ConceptID = studentID, id
		}
		switch {
		case isCredited(p):
			// Implicit repetition: working above a credited concept keeps it fresh.
			p.LastReviewed = &now
			batch = append(batch, p)
		case !mastered:
			// Only mastery grants new credit; an ordinary correct answer just refreshes.
		case p == nil:
			batch = append(batch, &storage.ConceptProgress{
				StudentID:       studentID,
				ConceptID:       id,
				Status:          string(mastery.StatusMastered),
				SM2EFactor:      2.5,
				LastReviewed:    &now,
				PlacementSeeded: true,
			})
		case p.Status != string(mastery.StatusMastered):
			p.Status = string(mastery.StatusMastered)
			p.WeaknessScore = 0
			p.LastReviewed = &now
			p.NextReviewDue = nil
			p.PlacementSeeded = true
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

// transitivePrereqIDs lists every concept below id in the graph, nearest first.
func (e *Engine) transitivePrereqIDs(id string) []string {
	seen := map[string]bool{id: true}
	var out []string
	queue := []string{id}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, p := range e.dag.PrereqsOf(cur) {
			if seen[p.ID] {
				continue
			}
			seen[p.ID] = true
			out = append(out, p.ID)
			queue = append(queue, p.ID)
		}
	}
	return out
}
