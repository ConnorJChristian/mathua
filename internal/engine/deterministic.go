package engine

import (
	mathrand "math/rand"
	"sync"

	"github.com/chuma-beep/mathua/internal/scheduler"
)

// Deterministic mode, for scripts/replay_audit and tests only (FSRS Phase 1
// review, I3). Three sources made a same-seed replay serve different topics
// run to run: the attempt id (crypto/rand) that seeds each generated
// question, the scheduler's clock-seeded tie-break, and the global
// math/rand stream the card, proof and scaffold pickers draw from.
// SetDeterministicSeed fixes all three from one seed. The server never
// calls it: attempt ids stay unguessable crypto/rand tokens there.

var (
	detMu  sync.Mutex
	detRng *mathrand.Rand
)

// SetDeterministicSeed makes attempt ids, scheduler tie-breaks and the
// global math/rand stream a seeded sequence (go.mod's godebug
// randseednop=0 keeps rand.Seed effective). Seed 0 restores the defaults
// for attempt ids and tie-breaks.
func SetDeterministicSeed(seed int64) {
	detMu.Lock()
	if seed == 0 {
		detRng = nil
	} else {
		detRng = mathrand.New(mathrand.NewSource(seed))
	}
	detMu.Unlock()
	scheduler.SetTieBreakSeed(seed)
	if seed != 0 {
		mathrand.Seed(seed) //nolint:staticcheck // effective under godebug randseednop=0
	}
}

// deterministicAttemptBytes fills b from the seeded sequence when
// deterministic mode is on, and reports whether it did.
func deterministicAttemptBytes(b []byte) bool {
	detMu.Lock()
	defer detMu.Unlock()
	if detRng == nil {
		return false
	}
	detRng.Read(b)
	return true
}
