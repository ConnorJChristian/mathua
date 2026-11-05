package clock

import (
	"testing"
	"time"
)

func TestSimStandsStillUntilMoved(t *testing.T) {
	at := time.Date(2026, 11, 5, 17, 0, 0, 0, time.UTC)
	s := NewSim(at)
	restore := s.Install()
	defer restore()
	for i := 0; i < 3; i++ {
		if got := Now(); !got.Equal(at) {
			t.Fatalf("read %d: %v, want %v (a read must not move the clock)", i, got, at)
		}
	}
	s.Advance(15 * time.Second)
	if got := Now(); !got.Equal(at.Add(15 * time.Second)) {
		t.Fatalf("after Advance: %v", got)
	}
	if d := Since(at); d != 15*time.Second {
		t.Fatalf("Since: %v", d)
	}
	s.Set(at)
	if !Now().Equal(at) {
		t.Fatalf("after Set: %v", Now())
	}
}

func TestSetRestores(t *testing.T) {
	fixed := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	restore := Set(func() time.Time { return fixed })
	if !Now().Equal(fixed) {
		t.Fatalf("Now = %v", Now())
	}
	restore()
	if d := time.Since(Now()); d < 0 || d > time.Minute {
		t.Fatalf("restored clock is not the wall clock: %v", Now())
	}
	r2 := Set(nil)
	defer r2()
	if d := time.Since(Now()); d < 0 || d > time.Minute {
		t.Fatalf("Set(nil) is not the wall clock: %v", Now())
	}
}
