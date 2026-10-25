package grader

import (
	"regexp"
	"strings"
)

// NormalizeTypedMath rewrites the spellings a learner naturally types into
// the plain form the math grader reads (Q41). The shown answers use some of
// them ("3.6x^{2}"), so a learner who copied the style was marked wrong:
//
//	x^{2}        -> x^(2)
//	x_{2}        -> x_2
//	\cdot, ×     -> *
//	\frac{a}{b}  -> (a)/(b)
//	β, \beta     -> beta (and the other Greek letters)
//	x2           -> x_2, when the expected answer writes subscripts
//
// It is applied to the expected answer too, so both sides are read alike.
func NormalizeTypedMath(s, expected string) string {
	s = strings.TrimSpace(s)
	// "βx" is beta times x: a typed Greek letter runs into the next symbol,
	// so it is spelled out with a product sign before a following letter.
	s = greekRunRe.ReplaceAllString(s, "$1*$2")
	for _, g := range greek {
		s = strings.ReplaceAll(s, g[0], g[1])
		s = strings.ReplaceAll(s, "\\"+g[1], g[1])
	}
	s = strings.NewReplacer("\\cdot", "*", "\\times", "*", "×", "*", "·", "*", "\\left", "", "\\right", "", "−", "-").Replace(s)
	// Innermost braces first, until nothing changes: "\\sqrt{x^{2}+5}"
	// needs the power rewritten before the root can be.
	for i := 0; i < 8; i++ {
		next := supBraceRe.ReplaceAllString(s, "^($1)")
		next = subBraceRe.ReplaceAllString(next, "_$1")
		next = fracRe.ReplaceAllString(next, "($1)/($2)")
		next = sqrtRe.ReplaceAllString(next, "sqrt($1)")
		if next == s {
			break
		}
		s = next
	}
	// "(1)/(3)": a bracketed plain number is the number.
	s = unwrapNumbers(s)
	if subscriptRe.MatchString(expected) {
		s = bareSubscriptRe.ReplaceAllString(s, "${1}${2}_$3")
	}
	return s
}

var (
	greekRunRe        = regexp.MustCompile(`([αβγδεθλμπρστφω])([A-Za-z\\\\])`)
	bareNumberParenRe = regexp.MustCompile(`\((-?\d+(?:\.\d+)?)\)`)
	sqrtRe            = regexp.MustCompile(`\\sqrt\{([^{}]*)\}`)
	fracRe            = regexp.MustCompile(`\\d?frac\{([^{}]*)\}\{([^{}]*)\}`)
	supBraceRe        = regexp.MustCompile(`\^\{([^{}]*)\}`)
	subBraceRe        = regexp.MustCompile(`_\{([^{}]*)\}`)
	subscriptRe       = regexp.MustCompile(`[A-Za-z]_\d`)
	bareSubscriptRe   = regexp.MustCompile(`(^|[^A-Za-z_])([A-Za-z])(\d+)`)
	greek             = [][2]string{
		{"α", "alpha"}, {"β", "beta"}, {"γ", "gamma"}, {"δ", "delta"}, {"ε", "epsilon"},
		{"θ", "theta"}, {"λ", "lambda"}, {"μ", "mu"}, {"π", "pi"}, {"ρ", "rho"},
		{"σ", "sigma"}, {"τ", "tau"}, {"φ", "phi"}, {"ω", "omega"},
	}
)

// unwrapNumbers removes brackets round a plain number where they group
// nothing: "(1)/(3)" and "x^(2)", but not "3(2)" (a product) or "f(2)".
func unwrapNumbers(s string) string {
	idx := bareNumberParenRe.FindAllStringSubmatchIndex(s, -1)
	if idx == nil {
		return s
	}
	var b strings.Builder
	last := 0
	for _, m := range idx {
		before := byte(0)
		if m[0] > 0 {
			before = s[m[0]-1]
		}
		after := byte(0)
		if m[1] < len(s) {
			after = s[m[1]]
		}
		okBefore := before == 0 || strings.IndexByte(" +-*/^(", before) >= 0
		okAfter := after == 0 || strings.IndexByte(" +-*/)", after) >= 0
		if !okBefore || !okAfter {
			continue
		}
		b.WriteString(s[last:m[0]])
		b.WriteString(s[m[2]:m[3]])
		last = m[1]
	}
	b.WriteString(s[last:])
	return b.String()
}
