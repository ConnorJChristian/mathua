// Package fsrs implements the parts of the FSRS-6 spaced-repetition model that
// Mathua needs, and nothing else.
//
// It is deliberately a pure package: no database, no clock of its own, no
// concept graph, no side effects. Every function takes the values it needs and
// returns a value. That is what lets the numbers be pinned against the
// reference implementation in a checked-in test instead of being trusted.
//
// # What is implemented
//
// The long-term memory model of FSRS-6: initial stability and difficulty per
// rating, the difficulty update with mean reversion, the success and failure
// stability updates, the forgetting curve, and the inverse interval. The
// parameter set is FSRS-6's published default; no fitting is attempted.
//
// # What is deliberately absent
//
// FSRS's learning and relearning steps (1 minute, 10 minutes) are not
// implemented. Mathua schedules in days, and its practice attempt is not a
// scheduled review, so there is no sub-day queue for those steps to describe.
//
// The same-day stability update is also absent. In FSRS it applies when a card
// is reviewed less than a day after its last review, and it is a different
// formula whose result is far smaller than the long-term one. Mathua's second
// attempt at a concept on the same day is practice, not a review, and treating
// it as a review would both record a repetition the learner did not earn and
// shorten the interval that repetition earned. If a future stage needs the
// same-day update it must be added as its own function with its own tests, not
// by relaxing the condition inside NextStability.
//
// # Expected values
//
// The numbers in vectors_test.go were produced by fsrs==6.3.2, the reference
// implementation, with fuzzing disabled. They are recorded literally rather
// than recomputed, so that changing a formula fails a test instead of silently
// changing what the test expects. Intervals are clamped to [1, 36500] because a
// due date in the past is not a schedule. The reference clamps the interval to
// the same range in 6.3.2, so that clamp is not a difference. Stability itself
// is not capped, because the reference lets it exceed the maximum interval.
package fsrs

import "math"

const (
	// StabilityMin is the smallest stability FSRS allows, in days. A stability
	// of zero would make the forgetting curve a division by zero.
	StabilityMin = 0.001

	// MaxStabilityDays is the reference maximum interval, in days. It is used
	// when seeding stability from an SM-2 interval. It does not bound
	// NextStability: the reference lets stability grow past this value and
	// clamps only the interval it derives from it.
	MaxStabilityDays = 36500.0

	// MaxIntervalDays is the longest interval this package will return.
	MaxIntervalDays = 36500

	// DefaultRetention is the probability of recall an interval is chosen for.
	// FSRS's default, and the value the identity Interval(S) == S assumes.
	DefaultRetention = 0.9
)

// Parameters is the FSRS-6 parameter vector. It is a fixed-size array so a
// caller cannot pass a shorter or longer slice and get NaN halfway through a
// calculation.
type Parameters [21]float64

// DefaultParameters returns FSRS-6's published default parameters.
//
// They are the output of fitting on a large review log from another product.
// They are used unchanged because Mathua has no review history in FSRS's shape
// to fit against, and a hand-tuned set would be a claim of evidence we do not
// have. A later stage may fit them; until then they are a starting point.
func DefaultParameters() Parameters {
	return Parameters{
		0.212, 1.2931, 2.3065, 8.2956, 6.4133, 0.8334, 3.0194, 0.001,
		1.8722, 0.1666, 0.796, 1.4835, 0.0614, 0.2629, 1.6483, 0.6014,
		1.8729, 0.5425, 0.0912, 0.0658, 0.1542,
	}
}

// Decay is the exponent of the forgetting curve, -w20.
func (p Parameters) Decay() float64 { return -p[20] }

// Factor is the scale of the forgetting curve, 0.9^(1/decay) - 1.
//
// The subtraction matters. The reference vault port of FSRS omitted it, which
// made every interval roughly half as long as it should be; see the package
// documentation on expected values.
func (p Parameters) Factor() float64 {
	return math.Pow(0.9, 1/p.Decay()) - 1
}
