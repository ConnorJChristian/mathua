package fsrs

import "math"

// InitialDifficulty is the difficulty FSRS assigns a concept the first time it
// is answered, given the rating. It is clamped to [1, 10].
func (p Parameters) InitialDifficulty(rating Rating) float64 {
	return clampDifficulty(p.initialDifficultyUnclamped(rating))
}

// initialDifficultyUnclamped is D0 before the [1, 10] clamp.
//
// It has to be reachable separately because the difficulty update's mean
// reversion uses D0(Easy) unclamped, which is about -4.77 rather than 1.0.
// Using the clamped value there is a real and quiet error: it moves every
// updated difficulty by a few thousandths and makes the implementation
// disagree with the reference while still looking plausible. The regression
// table pins the ordering, so a future reader cannot "tidy" the clamp back in.
func (p Parameters) initialDifficultyUnclamped(rating Rating) float64 {
	return p[4] - math.Exp(p[5]*(float64(rating)-1)) + 1
}

// NextDifficulty updates difficulty after a review of the given rating.
//
// The update is damped, then pulled slightly back towards D0(Easy), and only
// then clamped. Clamping before the mean reversion, or reverting towards the
// clamped D0, is the mistake the tests in difficulty_test.go exist to catch.
func (p Parameters) NextDifficulty(difficulty float64, rating Rating) float64 {
	if !rating.Valid() {
		return clampDifficulty(difficulty)
	}
	d := clampDifficulty(difficulty)
	delta := -p[6] * (float64(rating) - 3)
	damped := d + (10-d)*delta/9
	reverted := p[7]*p.initialDifficultyUnclamped(Easy) + (1-p[7])*damped
	return clampDifficulty(reverted)
}

func clampDifficulty(d float64) float64 {
	if math.IsNaN(d) {
		return 1
	}
	if d < 1 {
		return 1
	}
	if d > 10 {
		return 10
	}
	return d
}
