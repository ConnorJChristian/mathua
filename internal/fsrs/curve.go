package fsrs

import "math"

// Retrievability is the probability that the learner still recalls a concept
// after elapsedDays, given a stability that equals the interval at which recall
// is 90 percent.
//
// R(t, S) = (1 + FACTOR * t / S) ^ DECAY, with DECAY negative, so R falls
// towards zero as t grows and R(S, S) is exactly 0.9.
func (p Parameters) Retrievability(elapsedDays, stability float64) float64 {
	if stability <= 0 || math.IsNaN(stability) || math.IsInf(stability, 0) {
		stability = StabilityMin
	}
	if math.IsNaN(elapsedDays) || math.IsInf(elapsedDays, 0) {
		// A non-finite elapsed time is corrupt state. Fail closed to zero
		// recall rather than letting NaN or either infinity reach a due date.
		return 0
	}
	if elapsedDays <= 0 {
		// A review cannot be more than perfectly recalled, and a negative
		// elapsed time is a clock anomaly rather than evidence of forgetting.
		return 1
	}
	return math.Pow(1+p.Factor()*elapsedDays/stability, p.Decay())
}

// Interval is the gap at which the learner is predicted to recall the concept
// with DefaultRetention (0.9).
func (p Parameters) Interval(stability float64) int {
	return p.IntervalForRetention(stability, DefaultRetention)
}

// IntervalForRetention is the inverse of Retrievability: the gap after which
// recall is predicted to be retention, rounded to whole days and clamped to
// [1, MaxIntervalDays].
//
// The clamp matches the reference implementation, fsrs==6.3.2, which also
// returns at least one day and at most 36,500 days. A computed gap beyond the
// maximum is clamped before the float-to-integer conversion, so a very large
// stability cannot overflow the integer. Retention outside (0, 1) is not
// meaningful and falls back to the default rather than producing a negative or
// infinite interval.
func (p Parameters) IntervalForRetention(stability, retention float64) int {
	if stability <= 0 || math.IsNaN(stability) || math.IsInf(stability, 0) {
		return 1
	}
	if !(retention > 0 && retention < 1) {
		retention = DefaultRetention
	}
	raw := (stability / p.Factor()) * (math.Pow(retention, 1/p.Decay()) - 1)
	if math.IsNaN(raw) || raw <= 0 {
		return 1
	}
	if raw >= MaxIntervalDays {
		return MaxIntervalDays
	}
	// RoundToEven matches the reference implementation's Python round, which
	// rounds halves to even. math.Round would differ on exact halves and make
	// a boundary test fight the oracle.
	days := int(math.RoundToEven(raw))
	if days < 1 {
		return 1
	}
	if days > MaxIntervalDays {
		return MaxIntervalDays
	}
	return days
}
