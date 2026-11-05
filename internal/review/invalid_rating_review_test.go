package review

// Phase 1 independent reviewer (finding, left uncommitted): an invalid
// rating on a topic's first review is silently treated as Good, while the
// same rating on an existing state changes nothing. Replay does the same,
// so a log row with rating 0 replays "consistently" and fsrs_doctor cannot
// see it. Phase 2's event-to-rating mapping would fail open.

import (
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/fsrs"
)

func TestReviewer_InvalidRatingOnFirstReviewChangesNothing(t *testing.T) {
	for _, bad := range []fsrs.Rating{0, 5, -1} {
		m, _, _ := DefaultParams().Review(Memory{}, bad, time.Time{}, oracleBase)
		if !m.IsZero() {
			t.Errorf("Review(no state, rating %d) created state %+v (a Good)", bad, m)
		}
		if r := Replay(DefaultParams(), []Event{{At: oracleBase, Rating: bad}}); !r.IsZero() {
			t.Errorf("Replay of one rating-%d event gives %+v", bad, r)
		}
	}
}
