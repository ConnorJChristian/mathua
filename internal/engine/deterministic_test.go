package engine

import "testing"

// SetDeterministicSeed (replay_audit only) makes attempt ids repeat for one
// seed and differ across seeds; seed 0 restores crypto/rand ids.
func TestSetDeterministicSeed_AttemptIDs(t *testing.T) {
	defer SetDeterministicSeed(0)
	draw := func(seed int64) []string {
		SetDeterministicSeed(seed)
		return []string{newAttemptID(), newAttemptID(), newAttemptID()}
	}
	a, b, c := draw(2), draw(2), draw(3)
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("seed 2 id %d: %s then %s", i, a[i], b[i])
		}
	}
	if a[0] == c[0] {
		t.Errorf("seeds 2 and 3 gave the same id %s", a[0])
	}
	SetDeterministicSeed(0)
	if x, y := newAttemptID(), newAttemptID(); x == y || len(x) != 32 {
		t.Errorf("default ids %q %q", x, y)
	}
}
