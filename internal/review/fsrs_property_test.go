package review

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/fsrs"
)

// Properties of the wrapper over random histories (plan 6.1): S stays above
// 0 and finite, D stays in [1, 10], Again never raises S, the interval is
// monotone in S, R decreases in t. 20,000 random reviews on a fixed seed.
func TestFSRS_Properties(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	p := DefaultParams()
	ratings := []fsrs.Rating{fsrs.Again, fsrs.Hard, fsrs.Good, fsrs.Easy}
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	reviews := 0
	for hist := 0; hist < 500; hist++ {
		var m Memory
		var last time.Time
		at := base
		for i := 0; i < 40; i++ {
			r := ratings[rng.Intn(4)]
			// Gaps from minutes to years, including the same day.
			switch rng.Intn(4) {
			case 0:
				at = at.Add(time.Duration(rng.Intn(23*60)) * time.Minute)
			case 1:
				at = at.Add(time.Duration(1+rng.Intn(30)) * 24 * time.Hour)
			case 2:
				at = at.Add(time.Duration(1+rng.Intn(400)) * 24 * time.Hour)
			default:
				at = at.Add(time.Duration(rng.Float64()*24*800) * time.Minute)
			}
			next, elapsed, rb := p.Review(m, r, last, at)
			reviews++
			if !(next.Stability > 0) || math.IsInf(next.Stability, 0) || math.IsNaN(next.Stability) {
				t.Fatalf("S = %v after %+v rating %d", next.Stability, m, r)
			}
			if next.Difficulty < 1 || next.Difficulty > 10 || math.IsNaN(next.Difficulty) {
				t.Fatalf("D = %v after %+v rating %d", next.Difficulty, m, r)
			}
			if !m.IsZero() && r == fsrs.Again && next.Stability > m.Stability {
				t.Fatalf("Again raised S from %v to %v (elapsed %v)", m.Stability, next.Stability, elapsed)
			}
			if !m.IsZero() && r != fsrs.Again && elapsed < 1 && next.Stability < m.Stability {
				t.Fatalf("a same-day success lowered S from %v to %v", m.Stability, next.Stability)
			}
			if rb < 0 || rb > 1 {
				t.Fatalf("R before = %v", rb)
			}
			if next.Reps != m.Reps+1 || (r == fsrs.Again) != (next.Lapses == m.Lapses+1) {
				t.Fatalf("counters: %+v -> %+v on %d", m, next, r)
			}
			iv := p.IntervalFor(next)
			if iv < 1 || iv > DefaultMaxInterval {
				t.Fatalf("interval %d", iv)
			}
			m, last = next, at
		}
	}
	t.Logf("%d random reviews held the invariants", reviews)

	// The interval is monotone in S, and R decreases in t.
	prev := 0
	for s := 0.001; s < 10000; s *= 1.07 {
		iv := p.IntervalFor(Memory{Stability: s, Phase: PhaseReview})
		if iv < prev {
			t.Fatalf("interval fell from %d to %d as S rose to %v", prev, iv, s)
		}
		prev = iv
	}
	for _, s := range []float64{0.001, 0.5, 2.3065, 30, 365, 5000} {
		m := Memory{Stability: s, Phase: PhaseReview}
		last := 1.0
		for d := 0.0; d <= 2000; d += 0.5 {
			r := p.Retrievability(m, d)
			if r > last || r < 0 || r > 1 {
				t.Fatalf("R(S=%v) rose from %v to %v at t=%v", s, last, r, d)
			}
			last = r
		}
		// And R(S) = retention: due exactly at the interval the reference gives.
		if r := p.Retrievability(m, s); math.Abs(r-0.9) > 1e-12 {
			t.Fatalf("R(S, S) = %v, want 0.9", r)
		}
	}
	// Corrupt inputs fail closed rather than propagating.
	bad := Memory{Stability: math.NaN(), Difficulty: math.Inf(1), Reps: 1, Phase: PhaseReview}
	for _, r := range ratings {
		for _, n := range []Memory{p.Next(bad, r, 3), p.NextSameDay(bad, r)} {
			if !(n.Stability > 0) || math.IsNaN(n.Stability) || n.Difficulty < 1 || n.Difficulty > 10 {
				t.Fatalf("corrupt input gave %+v", n)
			}
		}
	}
	if p.Next(Initial(fsrs.Good), fsrs.Rating(9), 3) != Initial(fsrs.Good) {
		t.Fatalf("an invalid rating changed the state")
	}
}
