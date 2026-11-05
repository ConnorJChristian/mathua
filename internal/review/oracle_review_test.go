package review

// Phase 1 independent reviewer: the wrapper against fresh py-fsrs 6.3.2
// cases (oracle_data_review_test.go, generated outside the author's
// tools/fsrs-reference). Gaps are in seconds, so sub-day and fractional-day
// elapsed times are exercised, and the wrapper is chained on its own output:
// any drift accumulates instead of being reset to the oracle each step.
//
// Tolerance: S and D relative 1e-9 (absolute 1e-12 near 0), R absolute
// 1e-12, intervals exact. The task asked for at least 1e-6.

import (
	"math"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/fsrs"
)

func mustUTC(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t.UTC()
}

func relClose(a, b, tol float64) bool {
	scale := math.Max(math.Abs(a), math.Abs(b))
	if scale < 1e-9 {
		return math.Abs(a-b) < 1e-12
	}
	return math.Abs(a-b)/scale <= tol
}

func TestReviewer_WrapperMatchesFreshOracle(t *testing.T) {
	p := DefaultParams()
	if len(oracleCases) < 30 {
		t.Fatalf("oracle table has %d cases; generator output missing?", len(oracleCases))
	}
	steps := 0
	for _, c := range oracleCases {
		t.Run(c.Name, func(t *testing.T) {
			var m Memory
			var last time.Time
			at := oracleBase
			for i, want := range c.Steps {
				if i > 0 {
					at = at.Add(time.Duration(want.GapSec) * time.Second)
				}
				next, _, r := p.Review(m, fsrs.Rating(want.Rating), last, at)
				if want.HasR && math.Abs(r-want.R) > 1e-12 {
					t.Errorf("step %d: R %.17g, oracle %.17g", i, r, want.R)
				}
				if !relClose(next.Stability, want.S, 1e-9) {
					t.Errorf("step %d: S %.17g, oracle %.17g", i, next.Stability, want.S)
				}
				if !relClose(next.Difficulty, want.D, 1e-9) {
					t.Errorf("step %d: D %.17g, oracle %.17g", i, next.Difficulty, want.D)
				}
				if got := p.IntervalFor(next); got != want.Interval {
					t.Errorf("step %d: interval %d, oracle %d", i, got, want.Interval)
				}
				if next.Reps != i+1 {
					t.Errorf("step %d: reps %d", i, next.Reps)
				}
				m, last = next, at
				steps++
			}
		})
	}
	if steps < 100 {
		t.Fatalf("only %d steps checked", steps)
	}
}

// The first review is the reference's, not upstream's one-step-behind seed
// (S from the SM-2 interval, then an update: S 1, interval 1).
func TestReviewer_FirstGoodIsNotUpstreamsOneStepBehind(t *testing.T) {
	w := fsrs.DefaultParameters()
	m, el, r := DefaultParams().Review(Memory{}, fsrs.Good, time.Time{}, oracleBase)
	if m.Stability != w[2] || el != 0 || r != 0 {
		t.Fatalf("first Good: S %v (want w[2]=%v), elapsed %v, R %v", m.Stability, w[2], el, r)
	}
	if got := IntervalFor(m, 0.9, 365); got != 2 {
		t.Fatalf("first Good interval %d, want 2 (upstream's integration gives 1)", got)
	}
	for g := fsrs.Again; g <= fsrs.Easy; g++ {
		if got := Initial(g).Stability; got != w[int(g)-1] {
			t.Errorf("Initial(%d).S = %v, want w[%d] = %v", g, got, int(g)-1, w[int(g)-1])
		}
	}
	// Upstream's path: S seeded as 1 (the SM-2 interval) then a zero-elapsed
	// long-term Good leaves S at 1. Must not be what Review produces.
	upstream := Next(Memory{Stability: 1, Difficulty: w.InitialDifficulty(fsrs.Good), Phase: PhaseReview}, fsrs.Good, 0)
	if upstream.Stability == m.Stability {
		t.Fatalf("first review equals upstream's seeded path")
	}
}

// Under 24 hours, Review takes the short-term path; at exactly 24 hours the
// long-term one; the two differ (so the dispatch is observable).
func TestReviewer_SameDayUsesShortTermFormula(t *testing.T) {
	p := DefaultParams()
	m0, _, _ := p.Review(Memory{}, fsrs.Good, time.Time{}, oracleBase)
	for _, g := range []fsrs.Rating{fsrs.Again, fsrs.Hard, fsrs.Good, fsrs.Easy} {
		for _, d := range []time.Duration{0, time.Minute, 23*time.Hour + 59*time.Minute + 59*time.Second} {
			got, el, r := p.Review(m0, g, oracleBase, oracleBase.Add(d))
			want := p.NextSameDay(m0, g)
			if got != want || el != 0 || r != 1 {
				t.Errorf("rating %d after %v: %+v el %v r %v; want short-term %+v", g, d, got, el, r, want)
			}
			// At S = 2.31 Hard/Good floor to x1 and coincide with the
			// long-term formula at R = 1; from a low S (first Again, 0.212)
			// every rating separates the two formulas.
			low := Initial(fsrs.Again)
			st, _, _ := p.Review(low, g, oracleBase, oracleBase.Add(d))
			// (Hard's short-term factor floors to x1 at default weights for
			// any S above ~0.0004, so it coincides by construction.)
			if long := p.Next(low, g, 0); g != fsrs.Hard && long.Stability == st.Stability {
				t.Errorf("rating %d after %v: short-term equals long-term at 0 from S %v", g, d, low.Stability)
			}
		}
		got, el, _ := p.Review(m0, g, oracleBase, oracleBase.Add(24*time.Hour))
		if el != 1 || got != p.Next(m0, g, 1) {
			t.Errorf("rating %d at 24h: elapsed %v, not the long-term path", g, el)
		}
	}
}

// The 365-day cap binds (a 400-day Easy gap pushes S well past 365) and
// retention is 0.9 (interval == round(S) below the cap).
func TestReviewer_MaxIntervalAndRetention(t *testing.T) {
	p := DefaultParams()
	if p.Retention != 0.9 || p.MaxInterval != 365 {
		t.Fatalf("defaults %v / %v", p.Retention, p.MaxInterval)
	}
	m := Initial(fsrs.Easy)
	m = Next(m, fsrs.Easy, 400)
	m = Next(m, fsrs.Easy, 400)
	if m.Stability <= 365 {
		t.Fatalf("setup: S %v not above the cap", m.Stability)
	}
	if got := p.IntervalFor(m); got != 365 {
		t.Fatalf("interval %d, want the 365 cap", got)
	}
	for _, s := range []float64{0.6, 2.3065, 10.49, 100.5, 101.5, 364.4} {
		want := int(math.RoundToEven(s))
		if want < 1 {
			want = 1
		}
		if got := p.IntervalFor(Memory{Stability: s, Difficulty: 5, Phase: PhaseReview}); got != want {
			t.Errorf("S %v: interval %d, want %d", s, got, want)
		}
	}
	// No fuzz: the same input twice gives the same interval.
	a := p.IntervalFor(Memory{Stability: 47.3, Difficulty: 5, Phase: PhaseReview})
	b := p.IntervalFor(Memory{Stability: 47.3, Difficulty: 5, Phase: PhaseReview})
	if a != b {
		t.Fatal("interval not deterministic")
	}
}
