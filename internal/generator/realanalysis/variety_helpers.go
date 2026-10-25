package realanalysis

// Helpers for the parameterised real-analysis variants.
//
// Multiple-choice option order is fixed per stem (generator.MultipleChoice
// hashes the stem), so reshuffling an item no longer counts as a new item. The
// variants below get their variety from concrete instances instead: a set, a
// sequence, a series or a function with numbers in it, whose answer is computed
// here from those numbers. Exact values use math/big rationals so a printed
// fraction is never a rounded float.

import (
	"math"
	"math/big"
	"math/rand"
	"strconv"
	"strings"

	"github.com/chuma-beep/mathua/internal/generator"
)

// mcProblem renders one claim as a multiple-choice problem.
func mcProblem(c mcClaim) generator.Problem {
	p := generator.MultipleChoice(c.q, c.correct, c.distractors...)
	p.Explanation = c.reason
	return p
}

// mixMC draws uniformly among the fixed claims and the parameterised
// families; each family call builds a fresh concrete instance.
func mixMC(static []mcClaim, families ...func() mcClaim) generator.Problem {
	i := rand.Intn(len(static) + len(families))
	if i < len(static) {
		return mcProblem(static[i])
	}
	return mcProblem(families[i-len(static)]())
}

// distractorsFrom keeps the first three candidates that differ from the
// correct option and from each other.
func distractorsFrom(correct string, cands ...string) []string {
	seen := map[string]bool{strings.TrimSpace(correct): true}
	var out []string
	for _, c := range cands {
		k := strings.TrimSpace(c)
		if k == "" || seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, c)
		if len(out) == 3 {
			break
		}
	}
	return out
}

// minus swaps an ASCII hyphen-minus for the typographic minus used in prose.
func minus(s string) string { return strings.ReplaceAll(s, "-", "−") }

// rq is the exact rational a/b.
func rq(a, b int64) *big.Rat { return big.NewRat(a, b) }

// ri is the exact integer n as a rational.
func ri(n int64) *big.Rat { return new(big.Rat).SetInt64(n) }

func radd(a, b *big.Rat) *big.Rat { return new(big.Rat).Add(a, b) }
func rsub(a, b *big.Rat) *big.Rat { return new(big.Rat).Sub(a, b) }
func rmul(a, b *big.Rat) *big.Rat { return new(big.Rat).Mul(a, b) }
func rquo(a, b *big.Rat) *big.Rat { return new(big.Rat).Quo(a, b) }

// ratS prints a rational as a reduced fraction, or an integer, for display.
func ratS(r *big.Rat) string { return minus(r.RatString()) }

// ratDec prints a rational as a decimal for display, falling back to the
// fraction when the decimal does not terminate within six places.
func ratDec(r *big.Rat) string {
	if ratTerminates(r) {
		return minus(ratAnswer(r))
	}
	return ratS(r)
}

// ratAnswer is the numeric-answer form: decimal6 with an ASCII sign.
func ratAnswer(r *big.Rat) string {
	f, _ := r.Float64()
	return decimal6(f)
}

// ratTerminates reports whether r has a decimal expansion of at most six
// places.
func ratTerminates(r *big.Rat) bool {
	m := rmul(r, ri(1000000))
	return m.IsInt()
}

// itoa is strconv.Itoa with a typographic minus.
func itoa(n int) string { return minus(strconv.Itoa(n)) }

// pick returns a random element of xs.
func pick[T any](xs []T) T { return xs[rand.Intn(len(xs))] }

// randIn returns a uniform integer in [lo, hi].
func randIn(lo, hi int) int { return lo + rand.Intn(hi-lo+1) }

// randNonzero returns a uniform nonzero integer in [lo, hi].
func randNonzero(lo, hi int) int {
	for {
		if v := randIn(lo, hi); v != 0 {
			return v
		}
	}
}

// signedTerm prints "+ k" or "− k" for appending to an expression.
func signedTerm(k int) string {
	if k < 0 {
		return " − " + strconv.Itoa(-k)
	}
	return " + " + strconv.Itoa(k)
}

// coefX prints k·x compactly: "x", "−x", "3x".
func coefX(k int, x string) string {
	switch k {
	case 1:
		return x
	case -1:
		return "−" + x
	}
	return itoa(k) + x
}

// sqrtF is the square root of a positive integer, for approximate display.
func sqrtF(m int) float64 { return math.Sqrt(float64(m)) }
