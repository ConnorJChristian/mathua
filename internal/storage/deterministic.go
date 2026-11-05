package storage

import (
	mathrand "math/rand"
	"sync"
)

// Deterministic ids, for scripts/replay_audit and tests only (FSRS Phase 2,
// step 1). Session ids are random UUIDs, and a session id is part of a
// review_log idempotency key (sessionID:attemptID), so two same-seed replays
// wrote different keys. SetDeterministicIDs makes newUUID a seeded sequence;
// seed 0 restores crypto/rand. The server never calls it.
var (
	idMu  sync.Mutex
	idRng *mathrand.Rand
)

// SetDeterministicIDs fixes the UUID sequence (seed 0: crypto/rand again).
func SetDeterministicIDs(seed int64) {
	idMu.Lock()
	defer idMu.Unlock()
	if seed == 0 {
		idRng = nil
		return
	}
	idRng = mathrand.New(mathrand.NewSource(seed))
}

// deterministicIDBytes fills b from the seeded sequence when one is set.
func deterministicIDBytes(b []byte) bool {
	idMu.Lock()
	defer idMu.Unlock()
	if idRng == nil {
		return false
	}
	idRng.Read(b)
	return true
}
