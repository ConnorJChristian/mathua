package pytrack

import (
	"regexp"
	"strings"
)

// normalizeTypedMath rewrites the spellings a learner naturally types into
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
func normalizeTypedMath(s, expected string) string {
	s = strings.TrimSpace(s)
	// "βx" is beta times x: a typed Greek letter runs into the next symbol,
	// so it is spelled out with a product sign before a following letter.
	s = greekRunRe.ReplaceAllString(s, "$1*$2")
	for _, g := range greek {
		s = strings.ReplaceAll(s, g[0], g[1])
		s = strings.ReplaceAll(s, "\\"+g[1], g[1])
	}
	s = strings.NewReplacer("\\cdot", "*", "\\times", "*", "×", "*", "·", "*", "\\left", "", "\\right", "", "−", "-").Replace(s)
	for i := 0; i < 4; i++ {
		next := fracRe.ReplaceAllString(s, "($1)/($2)")
		if next == s {
			break
		}
		s = next
	}
	s = supBraceRe.ReplaceAllString(s, "^($1)")
	s = subBraceRe.ReplaceAllString(s, "_$1")
	if subscriptRe.MatchString(expected) {
		s = bareSubscriptRe.ReplaceAllString(s, "${1}${2}_$3")
	}
	return s
}

var (
	greekRunRe      = regexp.MustCompile(`([αβγδεθλμπρστφω])([A-Za-z\\\\])`)
	fracRe          = regexp.MustCompile(`\\d?frac\{([^{}]*)\}\{([^{}]*)\}`)
	supBraceRe      = regexp.MustCompile(`\^\{([^{}]*)\}`)
	subBraceRe      = regexp.MustCompile(`_\{([^{}]*)\}`)
	subscriptRe     = regexp.MustCompile(`[A-Za-z]_\d`)
	bareSubscriptRe = regexp.MustCompile(`(^|[^A-Za-z_])([A-Za-z])(\d+)`)
	greek           = [][2]string{
		{"α", "alpha"}, {"β", "beta"}, {"γ", "gamma"}, {"δ", "delta"}, {"ε", "epsilon"},
		{"θ", "theta"}, {"λ", "lambda"}, {"μ", "mu"}, {"π", "pi"}, {"ρ", "rho"},
		{"σ", "sigma"}, {"τ", "tau"}, {"φ", "phi"}, {"ω", "omega"},
	}
)
