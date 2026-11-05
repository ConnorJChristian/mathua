package review

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/fsrs"
)

// The wrapper is pinned against the reference scheduler end to end:
// tools/fsrs-reference/review_vectors.json holds what py-fsrs 6.3.2's
// Scheduler.review_card produced for each sequence (generate_review_vectors.py),
// and Review must reproduce every stability, difficulty, interval, elapsed
// whole-day count and pre-review retrievability. These cover what upstream's
// 115 formula vectors do not: the first review, the same-day path, lapses,
// gaps over a year, R at t = 0, and the weights at the reference's bounds.

type refVectors struct {
	Reference struct {
		Version           string    `json:"version"`
		DefaultParameters []float64 `json:"default_parameters"`
	} `json:"reference"`
	Sequences []struct {
		Name             string      `json:"name"`
		Parameters       interface{} `json:"parameters"`
		DesiredRetention float64     `json:"desired_retention"`
		MaximumInterval  int         `json:"maximum_interval"`
		Events           []struct {
			Rating  int     `json:"rating"`
			GapDays float64 `json:"gap_days"`
		} `json:"events"`
		Steps []struct {
			Rating               int      `json:"rating"`
			ElapsedWholeDays     *float64 `json:"elapsed_whole_days"`
			RetrievabilityBefore *float64 `json:"retrievability_before"`
			Stability            float64  `json:"stability"`
			Difficulty           float64  `json:"difficulty"`
			IntervalDays         int      `json:"interval_days"`
			State                string   `json:"state"`
		} `json:"steps"`
	} `json:"sequences"`
	Retrievability []struct {
		Stability   float64 `json:"stability"`
		ElapsedDays float64 `json:"elapsed_days"`
		Value       float64 `json:"value"`
	} `json:"retrievability"`
}

func loadRefVectors(t *testing.T) refVectors {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "tools", "fsrs-reference", "review_vectors.json"))
	if err != nil {
		t.Fatalf("read review_vectors.json: %v", err)
	}
	var v refVectors
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("parse review_vectors.json: %v", err)
	}
	if v.Reference.Version != "6.3.2" {
		t.Fatalf("vectors come from fsrs %s; the wrapper is pinned to 6.3.2", v.Reference.Version)
	}
	if len(v.Sequences) < 20 || len(v.Retrievability) < 30 {
		t.Fatalf("vectors look truncated: %d sequences, %d retrievability values", len(v.Sequences), len(v.Retrievability))
	}
	return v
}

func closeRel(a, b, tol float64) bool {
	if a == b {
		return true
	}
	scale := math.Max(math.Abs(a), math.Abs(b))
	if scale < 1e-9 {
		return math.Abs(a-b) < 1e-12
	}
	return math.Abs(a-b)/scale <= tol
}

func TestReview_MatchesReferenceSequences(t *testing.T) {
	v := loadRefVectors(t)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	total := 0
	for _, seq := range v.Sequences {
		seq := seq
		t.Run(seq.Name, func(t *testing.T) {
			p := DefaultParams()
			p.Retention = seq.DesiredRetention
			p.MaxInterval = seq.MaximumInterval
			if ws, ok := seq.Parameters.([]interface{}); ok {
				if len(ws) != 21 {
					t.Fatalf("%d weights", len(ws))
				}
				for i, w := range ws {
					p.Weights[i] = w.(float64)
				}
			}
			var m Memory
			at := base
			var last time.Time
			for i, ev := range seq.Events {
				want := seq.Steps[i]
				if i > 0 {
					at = at.Add(time.Duration(ev.GapDays * 24 * float64(time.Hour)))
				}
				next, elapsed, r := p.Review(m, fsrs.Rating(ev.Rating), last, at)
				total++
				if want.ElapsedWholeDays != nil && elapsed != *want.ElapsedWholeDays {
					t.Errorf("step %d: elapsed whole days %v, reference %v", i, elapsed, *want.ElapsedWholeDays)
				}
				if want.RetrievabilityBefore != nil && !closeRel(r, *want.RetrievabilityBefore, 1e-12) {
					t.Errorf("step %d: R before %.15g, reference %.15g", i, r, *want.RetrievabilityBefore)
				}
				if !closeRel(next.Stability, want.Stability, 1e-9) {
					t.Errorf("step %d (rating %d, gap %v): S %.12g, reference %.12g", i, ev.Rating, ev.GapDays, next.Stability, want.Stability)
				}
				if !closeRel(next.Difficulty, want.Difficulty, 1e-9) {
					t.Errorf("step %d (rating %d): D %.12g, reference %.12g", i, ev.Rating, next.Difficulty, want.Difficulty)
				}
				if got := p.IntervalFor(next); got != want.IntervalDays {
					t.Errorf("step %d: interval %d, reference %d (S %.6g)", i, got, want.IntervalDays, next.Stability)
				}
				// The reference never leaves "review" without steps; the fork
				// adds "relearning" after a lapse, as information.
				if want.State != "review" {
					t.Errorf("step %d: reference state %q, expected review", i, want.State)
				}
				wantPhase := PhaseReview
				if ev.Rating == int(fsrs.Again) {
					wantPhase = PhaseRelearning
				}
				if next.Phase != wantPhase {
					t.Errorf("step %d: phase %q, want %q", i, next.Phase, wantPhase)
				}
				if next.Reps != i+1 {
					t.Errorf("step %d: reps %d", i, next.Reps)
				}
				m, last = next, at
			}
		})
	}
	t.Logf("%d reviews across %d sequences agree with the reference", total, len(v.Sequences))
}

// The first review is the reference's: S = w[G-1], D = D0(G), and the
// interval that follows (2 days for Good). Upstream's integration gives
// S = 1 and 1 day here; that path is not reproduced.
func TestInitial_IsTheReferenceFirstReview(t *testing.T) {
	w := fsrs.DefaultParameters()
	for _, r := range []fsrs.Rating{fsrs.Again, fsrs.Hard, fsrs.Good, fsrs.Easy} {
		m := Initial(r)
		if m.Stability != w[int(r)-1] {
			t.Errorf("Initial(%d).Stability = %v, want w[%d] = %v", r, m.Stability, int(r)-1, w[int(r)-1])
		}
		if m.Difficulty != w.InitialDifficulty(r) {
			t.Errorf("Initial(%d).Difficulty = %v, want %v", r, m.Difficulty, w.InitialDifficulty(r))
		}
		if m.Reps != 1 || m.IsZero() {
			t.Errorf("Initial(%d) = %+v: one repetition, with state", r, m)
		}
	}
	good := Initial(fsrs.Good)
	if got := IntervalFor(good, DefaultRetention, DefaultMaxInterval); got != 2 {
		t.Errorf("first Good interval = %d, want 2 (the reference); upstream's one-step-behind path gives 1", got)
	}
	if again := Initial(fsrs.Again); again.Lapses != 1 || again.Phase != PhaseRelearning {
		t.Errorf("Initial(Again) = %+v: a lapse", again)
	}
	// A Review with no state is Initial, whatever the clock says.
	next, elapsed, r := DefaultParams().Review(Memory{}, fsrs.Good, time.Time{}, time.Now())
	if next != good || elapsed != 0 || r != 0 {
		t.Errorf("Review of an empty memory = %+v, %v, %v; want Initial(Good), 0, 0", next, elapsed, r)
	}
}

func TestRetrievability_MatchesReference(t *testing.T) {
	v := loadRefVectors(t)
	p := DefaultParams()
	for _, c := range v.Retrievability {
		m := Memory{Stability: c.Stability, Difficulty: 5, Reps: 1, Phase: PhaseReview}
		// The reference floors elapsed time to whole days; the wrapper's
		// Retrievability takes the days it is given, so the floor is applied
		// here as Review applies it.
		got := p.Retrievability(m, math.Floor(c.ElapsedDays))
		if !closeRel(got, c.Value, 1e-12) {
			t.Errorf("R(S=%v, t=%v) = %.15g, reference %.15g", c.Stability, c.ElapsedDays, got, c.Value)
		}
	}
	// R at t = 0 is exactly 1, for any stability; no state is 0.
	for _, s := range []float64{0.001, 0.212, 2.3065, 365, 1e6} {
		if got := Retrievability(Memory{Stability: s, Phase: PhaseReview}, 0); got != 1 {
			t.Errorf("R(S=%v, 0) = %v, want 1", s, got)
		}
	}
	if got := Retrievability(Memory{}, 10); got != 0 {
		t.Errorf("R of no state = %v, want 0", got)
	}
}

func TestElapsedWholeDays_IsTheReferenceFloor(t *testing.T) {
	base := time.Date(2026, 11, 1, 6, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		h    float64
		want float64
	}{
		{0, 0}, {0.01, 0}, {23, 0}, {23.99, 0}, {24, 1}, {36, 1}, {47.9, 1}, {48, 2}, {24 * 400, 400},
		// A clock that went backwards: the reference's timedelta.days floors
		// toward negative infinity; Review reads anything under 1 as same-day
		// and clamps the elapsed time to 0.
		{-0.5, -1}, {-30, -2},
	} {
		got := ElapsedWholeDays(base, base.Add(time.Duration(c.h*float64(time.Hour))))
		if got != c.want {
			t.Errorf("ElapsedWholeDays(+%vh) = %v, want %v", c.h, got, c.want)
		}
	}
	p := DefaultParams()
	m := Initial(fsrs.Good)
	next, elapsed, r := p.Review(m, fsrs.Good, base, base.Add(-time.Hour))
	if elapsed != 0 || r != 1 || next != p.NextSameDay(m, fsrs.Good) {
		t.Errorf("a review before its last review: elapsed %v, R %v, %+v", elapsed, r, next)
	}
}

// The fork's cap: 365 days, however stable the memory, and the reference's
// own [1, 36500] clamp underneath it.
func TestIntervalFor_CapAndRetention(t *testing.T) {
	p := DefaultParams()
	big := Memory{Stability: 5000, Difficulty: 2, Reps: 10, Phase: PhaseReview}
	if got := p.IntervalFor(big); got != 365 {
		t.Errorf("interval at S=5000 = %d, want the 365-day cap", got)
	}
	if got := IntervalFor(big, 0.9, 36500); got != 5000 {
		t.Errorf("interval at S=5000 with the reference cap = %d, want 5000", got)
	}
	if got := IntervalFor(Memory{Stability: 0.001, Phase: PhaseReview}, 0.9, 365); got != 1 {
		t.Errorf("interval at S=0.001 = %d, want 1", got)
	}
	if got := IntervalFor(Memory{}, 0.9, 365); got != 0 {
		t.Errorf("interval of no state = %d, want 0", got)
	}
	// Retention trades interval for recall: lower retention, longer interval.
	s := Memory{Stability: 100, Phase: PhaseReview}
	lo, mid, hi := IntervalFor(s, 0.85, 365), IntervalFor(s, 0.9, 365), IntervalFor(s, 0.95, 365)
	if !(lo > mid && mid > hi) || mid != 100 {
		t.Errorf("intervals at R=0.85/0.9/0.95 = %d/%d/%d, want decreasing with 100 in the middle", lo, mid, hi)
	}
	// An out-of-range retention falls back to the default, as internal/fsrs does.
	if got := IntervalFor(s, 0, 365); got != 100 {
		t.Errorf("interval with retention 0 = %d, want the default's 100", got)
	}
}
