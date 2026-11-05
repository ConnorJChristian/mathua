// Package clock is the one source of "now" for scheduling and the records it
// reads (FSRS plan Phase 2, step 1).
//
// The engine, the scheduler, the planner, the scorer and the progress and
// attempt writers of the store read the time through Now instead of
// time.Now. In the server Now is the wall clock. A replay or a test sets a
// simulated clock (Sim), so a run is a pure function of its inputs: the same
// snapshot, seed and clock give the same serves, the same rows and the same
// review_log, byte for byte. That is what makes "flag off = identical" and
// "shadow = off on every surface" checkable gates (scripts/replay_audit).
//
// The clock is process-wide, like the seeded random sources of
// engine.SetDeterministicSeed: a replay is one learner in one process, and
// tests that set it restore it (Set returns the restore function). Security
// expiries (auth tokens, password resets) keep the wall clock.
package clock

import (
	"sync"
	"time"
)

var (
	mu  sync.RWMutex
	now func() time.Time = time.Now
)

// Now is the current time: the wall clock unless Set installed another.
func Now() time.Time {
	mu.RLock()
	f := now
	mu.RUnlock()
	return f()
}

// Since is Now().Sub(t), the clock's time.Since.
func Since(t time.Time) time.Duration { return Now().Sub(t) }

// Set installs f as the clock (nil restores the wall clock) and returns a
// function that puts back the clock it replaced.
func Set(f func() time.Time) (restore func()) {
	mu.Lock()
	prev := now
	if f == nil {
		f = time.Now
	}
	now = f
	mu.Unlock()
	return func() {
		mu.Lock()
		now = prev
		mu.Unlock()
	}
}

// Sim is a simulated clock: it stands still until it is set or advanced, so
// the number of times code reads it changes nothing (a shadow write that
// reads the clock once more than off mode cannot shift a later timestamp).
type Sim struct {
	mu sync.Mutex
	t  time.Time
}

// NewSim is a simulated clock at t.
func NewSim(t time.Time) *Sim { return &Sim{t: t} }

// Now is the simulated time.
func (s *Sim) Now() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.t
}

// Set moves the clock to t.
func (s *Sim) Set(t time.Time) {
	s.mu.Lock()
	s.t = t
	s.mu.Unlock()
}

// Advance moves the clock forward by d.
func (s *Sim) Advance(d time.Duration) {
	s.mu.Lock()
	s.t = s.t.Add(d)
	s.mu.Unlock()
}

// Install makes s the process clock and returns the restore function.
func (s *Sim) Install() (restore func()) { return Set(s.Now) }
