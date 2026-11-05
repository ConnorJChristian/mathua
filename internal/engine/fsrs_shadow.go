package engine

import (
	"sort"

	"github.com/chuma-beep/mathua/internal/review"
)

// SetFSRSMode sets the engine's FSRS mode (MATHUA_FSRS by default). The
// replay and tests use it to run off and shadow in one process.
func (e *Engine) SetFSRSMode(m review.Mode) { e.fsrsMode = m }

// FSRSMode is the engine's FSRS mode.
func (e *Engine) FSRSMode() review.Mode { return e.fsrsMode }

// SetFSRSRules sets which events move FSRS (MATHUA_FSRS_RULES by default).
func (e *Engine) SetFSRSRules(r review.Rules) { e.fsrsRules = r }

// FSRSRules is the rules the engine applies in shadow and active mode.
func (e *Engine) FSRSRules() review.Rules { return e.fsrsRules }

// PacedPlanOrder is the order new topics are offered in (pacedOrder), as a
// list: read-only, for the replay's surface comparison.
func (e *Engine) PacedPlanOrder(studentID string) []string {
	progress, err := e.repo.GetAllProgress(studentID)
	if err != nil {
		return nil
	}
	order := e.pacedOrder(studentID, progress)
	ids := make([]string, 0, len(order))
	for id := range order {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		if order[ids[i]] != order[ids[j]] {
			return order[ids[i]] < order[ids[j]]
		}
		return ids[i] < ids[j]
	})
	return ids
}
