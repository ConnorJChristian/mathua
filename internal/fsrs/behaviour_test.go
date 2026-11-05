package fsrs

import (
	"math"
	"testing"
	"time"
)

func TestRetrievability(t *testing.T) {
	p := DefaultParameters()
	cases := []struct {
		stability float64
		elapsed   float64
		want      float64
	}{
		{1, 0, 1},
		{1, 1, 0.90000000000000002},
		{1, 3, 0.80938810357317081},
		{1, 10, 0.69282663457262172},
		{3, 1, 0.95733579556100856},
		{3, 3, 0.90000000000000002},
		{3, 10, 0.79950685987043069},
		{10, 1, 0.98568240877751456},
		{10, 3, 0.96102426904738336},
		{10, 10, 0.90000000000000002},
	}
	for _, c := range cases {
		closeEnough(t, "R", p.Retrievability(c.elapsed, c.stability), c.want)
	}
}

// TestRetrievabilityDecreases keeps the curve honest even where no expected
// value is pinned: more elapsed time must never mean more recall.
func TestRetrievabilityDecreases(t *testing.T) {
	p := DefaultParameters()
	for _, s := range []float64{0.5, 1, 10, 365} {
		previous := 1.0
		for day := 1.0; day <= 400; day *= 2 {
			got := p.Retrievability(day, s)
			if got > previous {
				t.Fatalf("S=%v: R rose from %v to %v at day %v", s, previous, got, day)
			}
			previous = got
		}
	}
}

func TestIntervalIsTheInverseOfRetrievability(t *testing.T) {
	p := DefaultParameters()
	for _, s := range []float64{0.5, 1, 3, 10, 30, 100, 365, 1000, 3650} {
		got := p.Interval(s)
		want := int(s)
		if s < 1 {
			want = 1
		}
		if got != want {
			t.Errorf("Interval(%v) = %d, want %d", s, got, want)
		}
		// R(I(S),S) = 0.9 only while the interval was not clamped. Below one
		// day the floor of 1 applies in both Mathua and the reference, so the
		// round trip is deliberately not exact there.
		if float64(got) == s {
			closeEnough(t, "R at the interval", p.Retrievability(float64(got), s), DefaultRetention)
		}
	}
}

func TestIntervalRoundingAndBounds(t *testing.T) {
	p := DefaultParameters()
	if got := p.IntervalForRetention(10, 0.8); got != 33 {
		t.Errorf("interval at 0.8 retention = %d, want 33", got)
	}
	// Below one day the reference also clamps the interval to one day in
	// fsrs==6.3.2. A due date that is already past is not a schedule, so
	// both implementations floor it to one day.
	if got := p.IntervalForRetention(1, 0.95); got != 1 {
		t.Errorf("sub-day interval = %d, want the 1-day floor", got)
	}
	if got := p.Interval(1e9); got != MaxIntervalDays {
		t.Errorf("huge stability interval = %d, want %d", got, MaxIntervalDays)
	}
	if got := p.Interval(0); got != 1 {
		t.Errorf("zero stability interval = %d, want 1", got)
	}
	// A retention outside (0, 1) is meaningless and must not produce a
	// negative or infinite interval.
	for _, r := range []float64{0, 1, -0.5, 1.5} {
		if got := p.IntervalForRetention(10, r); got != p.Interval(10) {
			t.Errorf("retention %v gave %d, want the default-retention interval", r, got)
		}
	}
}

func TestSeedFromInterval(t *testing.T) {
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)

	t.Run("keeps the existing due date", func(t *testing.T) {
		due := now.AddDate(0, 0, 6)
		got, err := SeedFromInterval(6, due, now)
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		if got.Stability != 6 {
			t.Errorf("stability = %v, want 6 (the interval is the stability)", got.Stability)
		}
		if !got.Due.Equal(due) {
			t.Errorf("due = %v, want %v", got.Due, due)
		}
		if got.AlreadyDue {
			t.Error("a future due date was reported as already due")
		}
	})

	t.Run("marking an overdue review rather than moving it", func(t *testing.T) {
		due := now.AddDate(0, 0, -2)
		got, err := SeedFromInterval(10, due, now)
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		if !got.AlreadyDue {
			t.Error("an overdue due date was not reported as already due")
		}
		if !got.Due.Equal(due) {
			t.Errorf("seeding moved an overdue due date to %v", got.Due)
		}
	})

	t.Run("a due date exactly now counts as due", func(t *testing.T) {
		got, err := SeedFromInterval(1, now, now)
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		if !got.AlreadyDue {
			t.Error("a due date equal to now was not treated as due")
		}
	})

	t.Run("rejects input it cannot convert", func(t *testing.T) {
		cases := []struct {
			name     string
			interval int
			due      time.Time
		}{
			{"zero interval", 0, now},
			{"negative interval", -3, now},
			{"beyond the maximum", MaxIntervalDays + 1, now},
			{"no due date", 6, time.Time{}},
		}
		for _, c := range cases {
			if _, err := SeedFromInterval(c.interval, c.due, now); err == nil {
				t.Errorf("%s was accepted", c.name)
			}
		}
	})

	t.Run("accepts the maximum interval", func(t *testing.T) {
		if _, err := SeedFromInterval(MaxIntervalDays, now.AddDate(0, 0, 1), now); err != nil {
			t.Errorf("the maximum interval was rejected: %v", err)
		}
	})
}

func TestLegacyStreakUnmet(t *testing.T) {
	prev := LegacySM2{Repetitions: 5, EFactor: 2.5, IntervalDays: 30}
	got := LegacyStreakUnmet(prev)
	if got.Repetitions != 0 {
		t.Errorf("repetitions = %d, want 0", got.Repetitions)
	}
	if got.IntervalDays != 1 {
		t.Errorf("interval = %d, want 1", got.IntervalDays)
	}
	if got.EFactor != prev.EFactor {
		t.Errorf("easiness changed from %v to %v", prev.EFactor, got.EFactor)
	}
	if prev.Repetitions != 5 || prev.IntervalDays != 30 {
		t.Error("LegacyStreakUnmet modified its input")
	}
	// Applying it twice is the same as applying it once: the transition is a
	// reset, not a decrement.
	again := LegacyStreakUnmet(got)
	if again != got {
		t.Errorf("second application changed the state: %+v then %+v", got, again)
	}
}

func TestNativeRatingOnlyProducesAgainOrGood(t *testing.T) {
	cases := map[bool]Rating{true: Good, false: Again}
	for correct, want := range cases {
		got := NativeRating(correct)
		if got != want {
			t.Errorf("NativeRating(%v) = %v, want %v", correct, got, want)
		}
	}
	// Hard and Easy exist because the formulas need all four values, but no
	// mapping may produce them: there is no calibrated signal for "recalled
	// with difficulty" or "effortless" in a graded answer.
	for _, correct := range []bool{true, false} {
		if r := NativeRating(correct); r == Hard || r == Easy {
			t.Errorf("NativeRating(%v) produced %v", correct, r)
		}
	}
}

// TestInvalidRatingsDoNotPanic keeps a corrupt stored rating from turning into a
// NaN that propagates into a due date.
func TestInvalidRatingsDoNotPanic(t *testing.T) {
	p := DefaultParameters()
	for _, r := range []Rating{-1, 0, 5, 100} {
		if got := p.InitialStability(r); got != StabilityMin {
			t.Errorf("InitialStability(%d) = %v, want the minimum", r, got)
		}
		if got := p.NextStability(10, 5, 0.9, r); got != 10 {
			t.Errorf("NextStability with rating %d = %v, want it unchanged", r, got)
		}
		if got := p.NextDifficulty(5, r); got != 5 {
			t.Errorf("NextDifficulty with rating %d = %v, want it unchanged", r, got)
		}
	}
}

func TestFSRSEdgeCasesAndClamping(t *testing.T) {
	p := DefaultParameters()

	// Retrievability with NaN or negative stability and negative elapsed days
	if got := p.Retrievability(-5, 10); got != 1.0 {
		t.Errorf("Retrievability(-5, 10) = %v, want 1.0", got)
	}
	if got := p.Retrievability(1, math.NaN()); got <= 0 || got > 1 {
		t.Errorf("Retrievability with NaN stability = %v, want valid probability", got)
	}
	if got := p.Retrievability(1, -2); got <= 0 || got > 1 {
		t.Errorf("Retrievability with negative stability = %v, want valid probability", got)
	}

	// IntervalForRetention with NaN stability
	if got := p.IntervalForRetention(math.NaN(), 0.9); got != 1 {
		t.Errorf("IntervalForRetention with NaN stability = %d, want 1", got)
	}

	// NextDifficulty boundary clamping
	if got := p.NextDifficulty(0.5, Good); got < 1 {
		t.Errorf("NextDifficulty with sub-1 difficulty = %v, want >= 1", got)
	}
	if got := p.NextDifficulty(15, Good); got > 10 {
		t.Errorf("NextDifficulty with over-10 difficulty = %v, want <= 10", got)
	}
	if got := p.NextDifficulty(math.NaN(), Good); got < 1 || got > 10 {
		t.Errorf("NextDifficulty with NaN difficulty = %v, want clamped [1, 10]", got)
	}

	// NextStability with Hard and Easy ratings
	sHard := p.NextStability(10, 5, 0.9, Hard)
	sGood := p.NextStability(10, 5, 0.9, Good)
	sEasy := p.NextStability(10, 5, 0.9, Easy)
	if sHard >= sGood {
		t.Errorf("NextStability(Hard)=%v should be less than Good=%v", sHard, sGood)
	}
	if sEasy <= sGood {
		t.Errorf("NextStability(Easy)=%v should be greater than Good=%v", sEasy, sGood)
	}

	// NextStability retrievability clamping (<0, >1 and NaN)
	sNegR := p.NextStability(10, 5, -0.5, Good)
	sZeroR := p.NextStability(10, 5, 0.0, Good)
	if sNegR != sZeroR {
		t.Errorf("NextStability with negative R = %v, want %v", sNegR, sZeroR)
	}
	sOverR := p.NextStability(10, 5, 1.5, Good)
	sOneR := p.NextStability(10, 5, 1.0, Good)
	if sOverR != sOneR {
		t.Errorf("NextStability with R > 1 = %v, want %v", sOverR, sOneR)
	}
	sNaNR := p.NextStability(10, 5, math.NaN(), Good)
	sDefR := p.NextStability(10, 5, DefaultRetention, Good)
	if sNaNR != sDefR {
		t.Errorf("NextStability with NaN R = %v, want %v", sNaNR, sDefR)
	}

	// Stability itself is not capped at MaxStabilityDays. The reference lets it
	// grow past the maximum interval and clamps only the interval. This is the
	// regression that the Phase 3 conformance work corrected.
	hugeS := p.NextStability(1e9, 1, 1.0, Easy)
	closeEnough(t, "NextStability huge S", hugeS, 1e9)
	if got := p.Interval(hugeS); got != MaxIntervalDays {
		t.Errorf("Interval(1e9) = %d, want %d", got, MaxIntervalDays)
	}
	nanS := p.NextStability(math.NaN(), 5, 0.9, Good)
	if nanS < StabilityMin {
		t.Errorf("NextStability with NaN S = %v, want >= StabilityMin", nanS)
	}
}

func TestNonFiniteInputsFailClosed(t *testing.T) {
	p := DefaultParameters()
	inf := math.Inf(1)
	negInf := math.Inf(-1)
	nan := math.NaN()

	finite := func(t *testing.T, label string, got, lo, hi float64) {
		t.Helper()
		if math.IsNaN(got) || math.IsInf(got, 0) {
			t.Fatalf("%s = %v, want a finite value", label, got)
		}
		if got < lo || got > hi {
			t.Fatalf("%s = %v, want [%v, %v]", label, got, lo, hi)
		}
	}

	t.Run("next stability", func(t *testing.T) {
		cases := []struct {
			name           string
			stability      float64
			difficulty     float64
			retrievability float64
		}{
			{"positive infinite stability", inf, 5, 0.9},
			{"negative infinite stability", negInf, 5, 0.9},
			{"positive infinite difficulty", 10, inf, 0.9},
			{"negative infinite difficulty", 10, negInf, 0.9},
			{"NaN difficulty", 10, nan, 0.9},
			{"positive infinite retrievability", 10, 5, inf},
			{"negative infinite retrievability", 10, 5, negInf},
			{"NaN retrievability", 10, 5, nan},
		}
		for _, tc := range cases {
			got := p.NextStability(tc.stability, tc.difficulty, tc.retrievability, Good)
			finite(t, tc.name, got, StabilityMin, math.Inf(1))
		}
	})

	t.Run("retrievability", func(t *testing.T) {
		for _, elapsed := range []float64{nan, inf, negInf} {
			if got := p.Retrievability(elapsed, 10); got != 0 {
				t.Errorf("Retrievability with elapsed %v = %v, want 0", elapsed, got)
			}
		}
		for _, stability := range []float64{nan, inf, negInf, 0, -1} {
			finite(t, "Retrievability non-finite stability", p.Retrievability(1, stability), 0, 1)
		}
	})

	t.Run("interval", func(t *testing.T) {
		for _, stability := range []float64{nan, inf, negInf} {
			if got := p.IntervalForRetention(stability, 0.9); got != 1 {
				t.Errorf("IntervalForRetention with stability %v = %d, want 1", stability, got)
			}
		}
		// A value this large used to overflow the int conversion before the
		// clamp could apply.
		if got := p.Interval(1e300); got != MaxIntervalDays {
			t.Errorf("Interval(1e300) = %d, want %d", got, MaxIntervalDays)
		}
	})

	t.Run("difficulty", func(t *testing.T) {
		for _, difficulty := range []float64{nan, inf, negInf} {
			finite(t, "NextDifficulty", p.NextDifficulty(difficulty, Good), 1, 10)
		}
	})

	t.Run("initial stability remains finite", func(t *testing.T) {
		for _, rating := range []Rating{Again, Hard, Good, Easy} {
			finite(t, ratingName(rating), p.InitialStability(rating), StabilityMin, math.Inf(1))
		}
	})
}
