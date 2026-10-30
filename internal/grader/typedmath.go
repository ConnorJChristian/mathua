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
//	\div, ÷      -> /
//	\, \; \!     -> spacing dropped
//	{: and :}    -> dropped (the math box's AsciiMath for invisible brackets)
//	\frac{a}{b}  -> (a)/(b)
//	x², x³       -> x^(2), x^(3)
//	\frac34      -> (3)/(4)
//	\sqrt[3]{L}, root(3)(L) -> (L)^(1/3)
//	\ln x, ∗     -> ln x, *
//	f_(L K)      -> f_LK
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
	// A mixed number, "2\frac{1}{4}", is "2 1/4" to the numeric grader.
	s = mixedFracRe.ReplaceAllString(s, "$1$2 $3/$4")
	// \frac34 and \frac1{12}: the digit arguments TeX allows without braces.
	s = fracDigitsRe.ReplaceAllString(s, "\\frac{$1}{$2}")
	s = fracDigitBraceRe.ReplaceAllString(s, "\\frac{$1}{")
	s = fracBraceDigitRe.ReplaceAllString(s, "\\frac{$1}{$2}")
	// Function names keep their names without the backslash: \ln x is ln x,
	// and x\ln 2 is x ln 2, not the name "xln".
	s = funcNameRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := funcNameRe.FindStringSubmatch(m)
		if sub[1] != "" {
			return sub[1] + " " + sub[2]
		}
		return sub[2]
	})
	s = strings.NewReplacer("\\div", "/", "÷", "/", "\\cdot", "*", "\\times", "*", "×", "*", "·", "*", "∗", "*", "\\ast", "*", "{:", "", ":}", "", "\\left", "", "\\right", "", "\\,", " ", "\\;", " ", "\\:", " ", "\\!", "", "\\ ", " ", "−", "-",
		// Superscript digits typed from a keyboard ("3x²"), and the inverse
		// sign of f⁻¹ (fork fix).
		"⁻¹", "^(-1)", "¹", "^(1)", "²", "^(2)", "³", "^(3)", "⁴", "^(4)").Replace(s)
	// Innermost braces first, until nothing changes: "\\sqrt{x^{2}+5}"
	// needs the power rewritten before the root can be.
	for i := 0; i < 8; i++ {
		next := supBraceRe.ReplaceAllString(s, "^($1)")
		next = subBraceRe.ReplaceAllString(next, "_$1")
		next = fracRe.ReplaceAllString(next, "($1)/($2)")
		// An nth root, as LaTeX writes it and as the math box exports it
		// (root(3)(L)), is a fractional power.
		next = nthRootRe.ReplaceAllString(next, "($2)^(1/$1)")
		next = rootExportRe.ReplaceAllString(next, "($2)^(1/$1)")
		next = sqrtRe.ReplaceAllString(next, "sqrt($1)")
		if next == s {
			break
		}
		s = next
	}
	// "(1)/(3)": a bracketed plain number is the number.
	s = unwrapNumbers(s)
	// SymPy's LaTeX writes a function's argument as \log{\left(x \right)}:
	// braces round a bracketed group are only grouping.
	s = braceParenRe.ReplaceAllString(s, "$1")
	// A multi-letter subscript the math box exports with spaces, "f_(L K)",
	// is the one name f_LK.
	s = subGroupRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := subGroupRe.FindStringSubmatch(m)
		out := "_" + strings.ReplaceAll(sub[1], " ", "")
		if sub[2] != "" {
			out += "*" + sub[2] // f_(K K)f_(L L): a product of two names
		}
		return out
	})
	if subscriptRe.MatchString(expected) {
		s = bareSubscriptRe.ReplaceAllString(s, "${1}${2}_$3")
	}
	return s
}

var (
	mixedFracRe       = regexp.MustCompile(`(^|[^\d.])(\d+)\\d?frac\{(\d+)\}\{(\d+)\}`)
	fracDigitsRe      = regexp.MustCompile(`\\d?frac(\d)(\d)`)
	fracDigitBraceRe  = regexp.MustCompile(`\\d?frac(\d)\{`)
	fracBraceDigitRe  = regexp.MustCompile(`\\d?frac\{([^{}]*)\}(\d)`)
	funcNameRe        = regexp.MustCompile(`([A-Za-z0-9)])?\\(ln|log|exp|sin|cos|tan|sec|csc|cot|arcsin|arccos|arctan|sinh|cosh|tanh|max|min|det)\b`)
	nthRootRe         = regexp.MustCompile(`\\sqrt\[(\d+)\]\{([^{}]*)\}`)
	rootExportRe      = regexp.MustCompile(`root\((\d+)\)\(([^()]*)\)`)
	braceParenRe      = regexp.MustCompile(`\{(\([^{}]*\))\}`)
	subGroupRe        = regexp.MustCompile(`_\(([A-Za-z0-9]+(?: [A-Za-z0-9]+)+)\)([A-Za-z]?)`)
	greekRunRe        = regexp.MustCompile(`([αβγδεθλμπρστφωηκνξψχζ])([A-Za-z\\\\])`)
	bareNumberParenRe = regexp.MustCompile(`\((-?\d+(?:\.\d+)?)\)`)
	sqrtRe            = regexp.MustCompile(`\\sqrt\{([^{}]*)\}`)
	fracRe            = regexp.MustCompile(`\\d?frac\s*\{([^{}]*)\}\s*\{([^{}]*)\}`)
	supBraceRe        = regexp.MustCompile(`\^\{([^{}]*)\}`)
	subBraceRe        = regexp.MustCompile(`_\{([^{}]*)\}`)
	subscriptRe       = regexp.MustCompile(`[A-Za-z]_\d`)
	bareSubscriptRe   = regexp.MustCompile(`(^|[^A-Za-z_])([A-Za-z])(\d+)`)
	greek             = [][2]string{
		{"α", "alpha"}, {"β", "beta"}, {"γ", "gamma"}, {"δ", "delta"}, {"ε", "epsilon"},
		{"θ", "theta"}, {"λ", "lambda"}, {"μ", "mu"}, {"π", "pi"}, {"ρ", "rho"},
		{"σ", "sigma"}, {"τ", "tau"}, {"φ", "phi"}, {"ω", "omega"},
		{"η", "eta"}, {"κ", "kappa"}, {"ν", "nu"}, {"ξ", "xi"}, {"ψ", "psi"}, {"χ", "chi"}, {"ζ", "zeta"},
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
		// A negative exponent keeps its brackets: "e^(-2)" is not "e^-2",
		// which the SymPy service cannot read (fork fix).
		if before == '^' && s[m[2]] == '-' {
			continue
		}
		// Nor a negative base: "(-1)^n" is not "-1^n".
		if s[m[2]] == '-' && (after == '^' || after == '*') {
			continue
		}
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
