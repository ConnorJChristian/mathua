package grader

import (
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

var (
	commaRe      = regexp.MustCompile(`,`)
	plusZeroRe   = regexp.MustCompile(`^\+`)
	andRe        = regexp.MustCompile(`(?i)(\d)\s*and\s*(\d)`)
	mixedRe      = regexp.MustCompile(`^(-?\d+)\s+(\d+/\d+)$`)
	fractionRe   = regexp.MustCompile(`^(-?\d+)/(\d+)$`)
	leadingDotRe = regexp.MustCompile(`^(-?)\.`)
	sciRe        = regexp.MustCompile(`^(-?\d+\.?\d*)[eE](-?\d+)$`)
)

const floatTolerance = 1e-9

type numericGrader struct{}

func (g *numericGrader) grade(expected, answer string) Result {
	e := normalise(expected)
	a := normalise(answer)
	if e == "" || a == "" {
		return emptyResult()
	}
	eVal, eIsFloat, eOK := parseNumeric(e)
	aVal, aIsFloat, aOK := parseNumeric(a)
	if !eOK || !aOK {
		return Result{Correct: false, Score: 0, Feedback: "Could not parse numeric values"}
	}
	if eIsFloat || aIsFloat {
		if decimalsAgree(e, eVal, a, aVal) {
			return Result{Correct: true, Score: 1}
		}
	} else {
		if eVal == aVal {
			return Result{Correct: true, Score: 1}
		}
	}
	return Result{Correct: false, Score: 0, Feedback: "Incorrect"}
}

func normalise(s string) string {
	s = commaRe.ReplaceAllString(s, "")
	s = plusZeroRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

// numericEpsilon is the window inside which an answer counts as the same number
// as the expected one. For an expected value printed with a decimal point it is
// half a unit in the last printed place: "12.61" accepts 12.610001, because
// that is the same answer rounded differently, and rejects 12.62, which is a
// different number at that precision. Expected values without a decimal point —
// integers, fractions, mixed numbers and scientific notation — keep the flat
// tolerance, because widening those would accept a different value of the same
// shape. The relative term keeps large magnitudes comparable without making the
// window unlimited.
func numericEpsilon(expected string, value float64) float64 {
	eps := floatTolerance
	if i := strings.IndexByte(expected, '.'); i >= 0 && !strings.ContainsAny(expected, "/eE") {
		if places := len(expected) - i - 1; places > 0 {
			if half := 0.5 / math.Pow(10, float64(places)); half > eps {
				eps = half
			}
		}
	}
	if relative := floatTolerance * math.Abs(value); relative > eps {
		eps = relative
	}
	return eps
}

// decimalsAgree compares an answer with a key by the places each was written
// to (Q50). The old window was half a unit in the key's last place either way,
// which let 0.25 through for a key of 0.2 and refused 2.67 for 2.666667.
//
//   - The answer has at least the key's places: it must round to the key
//     ("12.610001" and "12.61" for 12.61; not "12.62"; not "0.25" for 0.2).
//   - The key is a long rounding (4+ places, e.g. 2.666667) and the answer
//     has 2+ places: the key must round to the answer ("2.67", "2.667").
//   - A fraction against a decimal key: it must round to the key ("8/3").
//   - Otherwise (integers, fractions, scientific notation): equal values.
func decimalsAgree(e string, eVal float64, a string, aVal float64) bool {
	if math.Abs(eVal-aVal) <= numericEpsilon("", eVal) {
		return true
	}
	p, pOK := decimalPlaces(e)
	q, qOK := decimalPlaces(a)
	if pOK && qOK && q >= p && p > 0 {
		return roundsTo(aVal, p, eVal)
	}
	if pOK && qOK && p >= 4 && q >= 2 && q < p {
		return roundsTo(eVal, q, aVal)
	}
	// An exact answer ("8/3") against a rounded decimal key: it must round to it.
	if pOK && p > 0 && !qOK {
		return roundsTo(aVal, p, eVal)
	}
	return false
}

// decimalPlaces counts the digits after the point of a plain decimal; ok is
// false for fractions and scientific notation.
func decimalPlaces(s string) (int, bool) {
	if strings.ContainsAny(s, "/eE ") {
		return 0, false
	}
	i := strings.IndexByte(s, '.')
	if i < 0 {
		return 0, true
	}
	return len(s) - i - 1, true
}

// roundsTo reports whether v rounded half away from zero to n places is want.
func roundsTo(v float64, n int, want float64) bool {
	f := math.Pow(10, float64(n))
	r := math.Round(v*f+math.Copysign(1e-9, v)) / f
	return math.Abs(r-want) <= 1e-9*math.Max(1, math.Abs(want))
}

func parseNumeric(s string) (float64, bool, bool) {
	if s == "" {
		return 0, false, false
	}

	// Accept the spoken form of a mixed number: "4 and 1/10" / "2and3/10"
	// normalize to "4 1/10" before matching. (The English reading is how
	// students naturally type it, and matches the worked explanation.)
	s = andRe.ReplaceAllString(s, "$1 $2")

	// Mixed number: "1 1/2" — check before stripping whitespace. A negative
	// whole number subtracts its fraction ("-1 1/2" = -(1 + 1/2) = -1.5), not
	// adds it ("-1 1/2" must not parse as -1 + 0.5).
	if m := mixedRe.FindStringSubmatch(s); m != nil {
		whole := new(big.Rat)
		whole.SetString(m[1])
		frac := new(big.Rat)
		if _, ok := frac.SetString(m[2]); !ok {
			return 0, false, false
		}
		total := new(big.Rat)
		if whole.Sign() < 0 {
			total.Sub(whole, frac)
		} else {
			total.Add(whole, frac)
		}
		val, _ := total.Float64() // aislop-ignore-line ai-slop/swallowed-exception -- Float64's 2nd result is an exactness flag, not an error
		return val, true, true
	}

	// Strip whitespace for all other formats
	s = strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, s)

	if strings.HasPrefix(s, "-.") {
		s = "-0." + s[2:]
	}
	if s == "." || strings.HasPrefix(s, ".") {
		s = "0" + s
	}

	// Scientific notation
	if m := sciRe.FindStringSubmatch(s); m != nil {
		coeff := new(big.Rat)
		if _, ok := coeff.SetString(m[1]); !ok {
			return 0, false, false
		}
		exp := new(big.Int)
		if _, ok := exp.SetString(m[2], 10); !ok {
			return 0, false, false
		}
		ten := big.NewInt(10)
		pow := new(big.Int).Exp(ten, new(big.Int).Abs(exp), nil)
		if exp.Sign() >= 0 {
			num := new(big.Int).Mul(coeff.Num(), pow)
			val, _ := new(big.Rat).SetFrac(num, coeff.Denom()).Float64() // aislop-ignore-line ai-slop/swallowed-exception -- Float64's 2nd result is an exactness flag, not an error
			return val, true, true
		}
		denom := new(big.Int).Mul(coeff.Denom(), pow)
		val, _ := new(big.Rat).SetFrac(coeff.Num(), denom).Float64() // aislop-ignore-line ai-slop/swallowed-exception -- Float64's 2nd result is an exactness flag, not an error
		return val, true, true
	}

	// Fraction
	if m := fractionRe.FindStringSubmatch(s); m != nil {
		frac := new(big.Rat)
		if _, ok := frac.SetString(m[1] + "/" + m[2]); !ok {
			return 0, false, false
		}
		val, _ := frac.Float64()
		return val, true, true
	}

	// Integer or float (including big.Rat for exact parsing)
	r := new(big.Rat)
	if _, ok := r.SetString(s); ok {
		val, _ := r.Float64()
		if r.IsInt() {
			return val, false, true
		}
		return val, true, true
	}

	return 0, false, false
}

// AskedPlaces reads a prompt's "to two decimal places" / "to 3 decimals":
// the number of places the question asks for, or 0.
func AskedPlaces(prompt string) int {
	m := askedPlacesRe.FindStringSubmatch(strings.ToLower(prompt))
	if m == nil {
		return 0
	}
	if n, ok := map[string]int{"one": 1, "two": 2, "three": 3, "four": 4, "five": 5, "six": 6}[m[1]]; ok {
		return n
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

var askedPlacesRe = regexp.MustCompile(`(?:to|with|correct to) (one|two|three|four|five|six|\d) (?:decimal places?|decimals|d\.p\.|dp|places)`)

// RoundedAsAsked reports an answer that is the expected value rounded to the
// places the prompt asked for, written with exactly that many places: 0.89
// for a key of 400/448 asked "to two decimal places" (fork fix: a fraction
// key took only a decimal of three or more places, so the answer the
// question asked for was refused).
func RoundedAsAsked(expected, answer string, places int) bool {
	if places <= 0 {
		return false
	}
	ev, ok := constEvalNumeric(normalise(expected))
	if !ok {
		// A mixed number ("133 1/3") is a value too.
		if v, ok2, _ := parseNumeric(normalise(expected)); ok2 {
			ev, ok = v, true
		}
	}
	if !ok {
		return false
	}
	a := normalise(strings.ReplaceAll(strings.TrimSpace(answer), "−", "-"))
	if p, ok := decimalPlaces(a); !ok || p != places {
		return false
	}
	av, err := strconv.ParseFloat(a, 64)
	if err != nil {
		return false
	}
	return roundsTo(ev, places, av)
}
