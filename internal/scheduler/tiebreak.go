package scheduler

import (
	"math/rand"
	"sync"
	"time"
)

// Ties among equal-priority candidates are broken at random. By default the
// draw is seeded from the clock per call, as it always was. SetTieBreakSeed
// makes it a seeded sequence instead, so a replay with a fixed seed serves
// the same topics every run (FSRS Phase 1 review, I3: replay_audit was not
// deterministic, so "replay identical" could not be a gate). Production
// never calls it.
var (
	tieMu  sync.Mutex
	tieRng *rand.Rand
)

// SetTieBreakSeed fixes the tie-break sequence (seed 0 restores the clock).
func SetTieBreakSeed(seed int64) {
	tieMu.Lock()
	defer tieMu.Unlock()
	if seed == 0 {
		tieRng = nil
		return
	}
	tieRng = rand.New(rand.NewSource(seed))
}

// tieBreak picks an index in [0, n).
func tieBreak(salt int64, n int) int {
	tieMu.Lock()
	r := tieRng
	if r != nil {
		i := r.Intn(n)
		tieMu.Unlock()
		return i
	}
	tieMu.Unlock()
	return rand.New(rand.NewSource(time.Now().UnixNano() ^ salt)).Intn(n)
}
