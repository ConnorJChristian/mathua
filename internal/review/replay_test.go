package review

import (
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/fsrs"
)

// Event sourcing (plan 6.1): the log a sequence of reviews writes, replayed
// from the start, reproduces the final state exactly, not approximately.
// 300 random histories including same-day reviews, lapses and long gaps.
func TestReplay_ReproducesTheStoredStateExactly(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	p := DefaultParams()
	ratings := []fsrs.Rating{fsrs.Again, fsrs.Hard, fsrs.Good, fsrs.Easy}
	for h := 0; h < 300; h++ {
		var m Memory
		var last time.Time
		at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		var log []Event
		n := 1 + rng.Intn(30)
		for i := 0; i < n; i++ {
			r := ratings[rng.Intn(4)]
			at = at.Add(time.Duration(rng.Float64()*24*60*40) * time.Minute)
			next, elapsed, rb := p.Review(m, r, last, at)
			log = append(log, Event{At: at, Rating: r, ElapsedDays: elapsed, Source: "review", SAfter: next.Stability, DAfter: next.Difficulty})
			_ = rb
			m, last = next, at
		}
		got := Replay(p, log)
		if got != m {
			t.Fatalf("history %d: replay %+v, stored %+v", h, got, m)
		}
		if d := Diff(m, got); d != "" {
			t.Fatalf("Diff of equal states: %s", d)
		}
	}
	// A dropped event, a changed rating, and a lost column each show.
	m := Initial(fsrs.Good)
	log := []Event{{Rating: fsrs.Good, ElapsedDays: 0, Source: "review"}}
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, gap := range []float64{2, 11, 0.5, 46} {
		at = at.Add(time.Duration(gap * 24 * float64(time.Hour)))
		next, elapsed, _ := p.Review(m, fsrs.Good, at.Add(-time.Duration(gap*24*float64(time.Hour))), at)
		log = append(log, Event{At: at, Rating: fsrs.Good, ElapsedDays: elapsed, Source: "review"})
		m = next
	}
	if Diff(m, Replay(p, log)) != "" {
		t.Fatal("the straight replay differs")
	}
	if Diff(m, Replay(p, log[:len(log)-1])) == "" {
		t.Error("a dropped event replays to the same state")
	}
	changed := append([]Event(nil), log...)
	changed[2].Rating = fsrs.Again
	if Diff(m, Replay(p, changed)) == "" {
		t.Error("a changed rating replays to the same state")
	}
	lost := m
	lost.Stability = 0
	lost.Phase = PhaseNone
	if Diff(lost, Replay(p, log)) == "" {
		t.Error("a row that lost its state matches its replay")
	}
	// A seed event sets the state outright; reviews continue from it.
	seeded := Replay(p, []Event{{Source: "seed", SAfter: 30, DAfter: 4}, {Rating: fsrs.Good, ElapsedDays: 30, Source: "review"}})
	want := p.Next(Memory{Stability: 30, Difficulty: 4, Phase: PhaseReview}, fsrs.Good, 30)
	if seeded != want {
		t.Errorf("seeded replay %+v, want %+v", seeded, want)
	}
	if Replay(p, nil) != (Memory{}) {
		t.Error("an empty log replays to state")
	}
}

// ReviewChecked and ReplayChecked say so on an invalid rating (FSRS Phase 1
// review, L1); the unchecked forms fail closed (no state, or unchanged).
func TestInvalidRating_Errors(t *testing.T) {
	p := DefaultParams()
	at := time.Date(2026, 11, 5, 17, 0, 0, 0, time.UTC)
	good := p.Initial(fsrs.Good)
	for _, bad := range []fsrs.Rating{0, 5, -1} {
		if m, el, r, err := p.ReviewChecked(Memory{}, bad, time.Time{}, at); !errors.Is(err, ErrInvalidRating) || !m.IsZero() || el != 0 || r != 0 {
			t.Errorf("ReviewChecked(no state, %d) = %+v %v %v %v", bad, m, el, r, err)
		}
		if m, _, _, err := p.ReviewChecked(good, bad, at, at.Add(72*time.Hour)); !errors.Is(err, ErrInvalidRating) || m != good {
			t.Errorf("ReviewChecked(state, %d) = %+v %v", bad, m, err)
		}
		if p.Initial(bad) != (Memory{}) {
			t.Errorf("Initial(%d) created state", bad)
		}
		m, err := ReplayChecked(p, []Event{{At: at, Rating: fsrs.Good}, {At: at.Add(time.Hour), Rating: bad}})
		if !errors.Is(err, ErrInvalidRating) || m != good {
			t.Errorf("ReplayChecked with a rating-%d event = %+v %v", bad, m, err)
		}
	}
	for r := fsrs.Again; r <= fsrs.Easy; r++ {
		if _, _, _, err := p.ReviewChecked(Memory{}, r, time.Time{}, at); err != nil {
			t.Errorf("ReviewChecked(%d): %v", r, err)
		}
	}
	// A seed event carries no rating of its own and is not an error.
	if _, err := ReplayChecked(p, []Event{{At: at, Source: "seed", SAfter: 3, DAfter: 5}}); err != nil {
		t.Errorf("seed event: %v", err)
	}
}
