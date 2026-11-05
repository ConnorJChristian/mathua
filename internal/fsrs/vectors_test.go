package fsrs

import (
	"math"
	"strconv"
	"testing"
)

// These are not recomputed values. They are the numbers fsrs==6.3.2 produced
// with fuzzing disabled, written down literally so that changing a formula
// fails a test instead of quietly changing what the test expects.
//
// Tolerance is 1e-12 relative: the arithmetic is the same sequence of double
// operations in both languages, so agreement should be at the last bit, and a
// looser tolerance would hide a wrong coefficient.

func closeEnough(t *testing.T, label string, got, want float64) {
	t.Helper()
	const rel = 1e-12
	if math.Abs(got-want) > rel*math.Max(1, math.Abs(want)) {
		t.Errorf("%s = %.17g, want %.17g", label, got, want)
	}
}

func TestValues(t *testing.T) {
	p := DefaultParameters()

	t.Run("constants", func(t *testing.T) {
		closeEnough(t, "decay", p.Decay(), -0.1542)
		closeEnough(t, "factor", p.Factor(), 0.9803464944134797)
	})

	t.Run("initial stability", func(t *testing.T) {
		want := map[Rating]float64{Again: 0.212, Hard: 1.2931, Good: 2.3065, Easy: 8.2956}
		for rating, w := range want {
			closeEnough(t, ratingName(rating), p.InitialStability(rating), w)
		}
	})

	t.Run("initial difficulty clamped", func(t *testing.T) {
		want := map[Rating]float64{
			Again: 6.4133,
			Hard:  5.112170705601056,
			Good:  2.118103970459016,
			Easy:  1.0,
		}
		for rating, w := range want {
			closeEnough(t, ratingName(rating), p.InitialDifficulty(rating), w)
		}
	})

	t.Run("initial difficulty unclamped", func(t *testing.T) {
		// The mean reversion in NextDifficulty uses this value, not 1.0.
		closeEnough(t, "D0(Easy) unclamped", p.initialDifficultyUnclamped(Easy), -4.771630703161737)
	})

	t.Run("next difficulty", func(t *testing.T) {
		cases := []struct {
			difficulty float64
			rating     Rating
			want       float64
		}{
			{1, Again, 7.0269895692968385}, {1, Hard, 4.010608969296839},
			{1, Good, 1.0}, {1, Easy, 1.0},
			{5, Again, 8.341762369296838}, {5, Hard, 6.665995369296838},
			{5, Good, 4.9902283692968386}, {5, Easy, 3.3144613692968385},
			{9, Again, 9.656535169296836}, {9, Hard, 9.321381769296838},
			{9, Good, 8.986228369296837}, {9, Easy, 8.651074969296838},
		}
		for _, c := range cases {
			got := p.NextDifficulty(c.difficulty, c.rating)
			closeEnough(t, ratingName(c.rating)+" at D="+floatName(c.difficulty), got, c.want)
		}
	})
}

// TestNextStability pins both formulas. Again is the forgetting formula capped
// at the previous stability; Hard, Good and Easy are the recall formula with
// w15 and w16 as multipliers.
func TestNextStability(t *testing.T) {
	p := DefaultParameters()
	cases := []struct {
		rating     Rating
		stability  float64
		difficulty float64
		retrieval  float64
		want       float64
	}{
		{Again, 1, 1, 0.9, 0.34967089737408596},
		{Again, 1, 1, 0.7, 0.48621573224326564},
		{Again, 10, 5, 0.9, 1.3919869729546932},
		{Again, 100, 9, 0.7, 5.0258878445733748},

		{Hard, 1, 1, 0.9, 4.2401281813369227},
		{Hard, 1, 1, 0.7, 11.547998423077729},
		{Hard, 10, 5, 0.9, 23.246875110466814},
		{Hard, 10, 5, 0.7, 53.124225325634441},
		{Hard, 100, 1, 0.9, 250.43960743681629},
		{Hard, 100, 9, 0.7, 197.94901023682462},

		{Good, 1, 1, 0.9, 6.3876424697986742},
		{Good, 1, 5, 0.7, 11.523443721061916},
		{Good, 10, 1, 0.9, 46.711215803311212},
		{Good, 10, 5, 0.9, 32.02672948198672},
		{Good, 10, 9, 0.7, 33.902131318941599},
		{Good, 100, 5, 0.9, 250.08939883952408},
		{Good, 100, 1, 0.7, 914.34162152331714},

		{Easy, 1, 1, 0.9, 11.090515581685937},
		{Easy, 1, 9, 0.7, 7.5697859150589553},
		{Easy, 10, 5, 0.9, 51.253861646812936},
		{Easy, 10, 9, 0.9, 23.751287215604311},
		{Easy, 100, 1, 0.7, 1625.1804229510205},
		// 150.0298 is the Good value for this state; the Easy value is higher
		// because w16 scales the gain. Verified against the oracle directly
		// after a transcription slip put it in the Easy row.
		{Good, 100, 9, 0.9, 150.0297996131747},
		{Easy, 100, 9, 0.9, 193.70081169551489},
	}
	for _, c := range cases {
		got := p.NextStability(c.stability, c.difficulty, c.retrieval, c.rating)
		closeEnough(t, ratingName(c.rating)+" S="+floatName(c.stability), got, c.want)
	}
}

// TestLapseNeverIncreasesStability states the invariant the cap exists for. A
// failure cannot make a concept more stable; otherwise a learner who forgets
// would be rewarded with a longer gap.
func TestLapseNeverIncreasesStability(t *testing.T) {
	p := DefaultParameters()
	for _, s := range []float64{0.5, 1, 3, 10, 30, 100, 365, 1000} {
		for _, d := range []float64{1, 5, 10} {
			for _, r := range []float64{0, 0.5, 0.9, 1} {
				got := p.NextStability(s, d, r, Again)
				if got > s+1e-12 {
					t.Errorf("Again at S=%v D=%v R=%v gave S'=%v, larger than the previous stability", s, d, r, got)
				}
				if got < StabilityMin {
					t.Errorf("Again at S=%v gave S'=%v below the minimum", s, got)
				}
			}
		}
	}
}

func ratingName(r Rating) string {
	switch r {
	case Again:
		return "Again"
	case Hard:
		return "Hard"
	case Good:
		return "Good"
	case Easy:
		return "Easy"
	}
	return "Rating(?)"
}

func floatName(v float64) string {
	return strconv.FormatFloat(v, 'g', -1, 64)
}
