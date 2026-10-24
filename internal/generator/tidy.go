package generator

import (
	"regexp"
	"strings"
)

// Tidy rewrites the arithmetic-by-template leftovers that generators print
// when they splice random coefficients into a fixed pattern:
//
//	1x^{2} + -6x + 0  →  x^{2} - 6x
//	(x - -1)          →  (x + 1)
//	5-1i              →  5-i
//
// A generator that writes "%dx + %d" with a = 1 and b = -6 is correct but
// reads as broken, and learners reported it as such. Fixing every format
// string by hand (there are a few hundred) would leave the next generator to
// repeat it, so the registry applies this to every problem it serves.
//
// It only touches algebra: a rule fires inside an expression that has a
// variable in it, so arithmetic drills that mean "5 + -3" keep it (see
// keepsSignedOperands for the concepts that drill exactly that).
func Tidy(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	for _, seg := range splitMath(s) {
		if seg.math {
			b.WriteString(seg.open + tidyRun(seg.text, true) + seg.close)
			continue
		}
		b.WriteString(exprRunRe.ReplaceAllStringFunc(seg.text, func(run string) string {
			return tidyRun(run, false)
		}))
	}
	return b.String()
}

// keepsSignedOperands names the concepts whose point is the signed operand
// itself: adding and subtracting negatives. Tidying "4 - -3" into "4 + 3"
// would answer the question for the learner.
func keepsSignedOperands(conceptID string) bool {
	return strings.HasPrefix(conceptID, "arith.") ||
		strings.Contains(conceptID, "integer") ||
		strings.Contains(conceptID, ".neg")
}

type segment struct {
	text        string
	math        bool
	open, close string
}

// splitMath cuts s into prose and $...$ / $$...$$ math segments. An escaped
// \$ is prose (a price). An unclosed opener leaves the rest as prose.
func splitMath(s string) []segment {
	var out []segment
	i, start := 0, 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) && s[i+1] == '$' {
			i += 2
			continue
		}
		if s[i] != '$' {
			i++
			continue
		}
		delim := "$"
		if strings.HasPrefix(s[i:], "$$") {
			delim = "$$"
		}
		end := findClose(s, i+len(delim), delim)
		if end < 0 {
			break
		}
		if i > start {
			out = append(out, segment{text: s[start:i]})
		}
		out = append(out, segment{text: s[i+len(delim) : end], math: true, open: delim, close: delim})
		i = end + len(delim)
		start = i
	}
	if start < len(s) {
		out = append(out, segment{text: s[start:]})
	}
	return out
}

func findClose(s string, from int, delim string) int {
	for j := from; j < len(s); j++ {
		if s[j] == '\\' && j+1 < len(s) && s[j+1] == '$' {
			j++
			continue
		}
		if strings.HasPrefix(s[j:], delim) {
			if delim == "$" && strings.HasPrefix(s[j:], "$$") {
				return -1
			}
			return j
		}
	}
	return -1
}

var (
	// A run of plain-text algebra: "x^3 + -1x^2 + -10x + -8".
	exprRunRe = regexp.MustCompile(`[0-9a-zA-Z^²³.+\-*/()=<>'' ]+`)
	// A run counts as algebra when a single-letter variable sits next to a
	// number, a power or an operator.
	algebraRe = regexp.MustCompile(`\d[a-z]|[a-z]\^|[a-z][²³]|[+\-=]\s*[a-z]\b|\b[a-z]\s*[+\-=<>]|\(\d*[a-z]`)

	plusNegRe  = regexp.MustCompile(`([\w)}²³'])(\s*)\+\s*-\s*(\d)`)
	minusNegRe = regexp.MustCompile(`([\w)}²³'])(\s*)-\s*-\s*(\d)`)
	// " + 0x", " - 0", "+0y'" — a zero term after another term. Not 0i:
	// "20 + 0i" is a complex number written in a + bi form on purpose.
	zeroTermRe = regexp.MustCompile(`([\w)}²³'])\s*[+\-]\s*0(?:[a-hj-z](?:\^\{?\d+\}?)?'*)?([\s=$),<>&}\]]|\\|$)`)
)

func tidyRun(run string, math bool) string {
	if !math && !algebraRe.MatchString(run) {
		return run
	}
	if math && !hasVariable(run) {
		return run
	}
	run = plusNegRe.ReplaceAllString(run, "$1$2- $3")
	run = minusNegRe.ReplaceAllString(run, "$1$2+ $3")
	run = dropUnitCoefficients(run, math)
	for {
		next := zeroTermRe.ReplaceAllString(run, "$1$2")
		if next == run {
			break
		}
		run = next
	}
	return unwrapLoneVariable(run)
}

var loneVarRe = regexp.MustCompile(`\(([a-z])\)\^`)

// unwrapLoneVariable turns "(y)^{2}", left by dropping "+0" from "(y+0)^{2}",
// into "y^{2}". Not "f(x)^2": a letter before the bracket is a function name.
func unwrapLoneVariable(s string) string {
	idx := loneVarRe.FindAllStringSubmatchIndex(s, -1)
	if idx == nil {
		return s
	}
	var b strings.Builder
	last := 0
	for _, m := range idx {
		if m[0] > 0 && (isLetter(s[m[0]-1]) || s[m[0]-1] == '\\') {
			continue
		}
		b.WriteString(s[last:m[0]])
		b.WriteString(s[m[2]:m[3]] + "^")
		last = m[1]
	}
	b.WriteString(s[last:])
	return b.String()
}

// hasVariable reports whether math text names a variable: a letter that is
// not part of a \command.
func hasVariable(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' {
			for i+1 < len(s) && isLetter(s[i+1]) {
				i++
			}
			continue
		}
		if isLetter(c) {
			return true
		}
	}
	return false
}

func isLetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }
func isDigit(c byte) bool  { return c >= '0' && c <= '9' }

// dropUnitCoefficients turns "1x" into "x" (and "-1x" into "-x") where the 1
// is a coefficient: it starts a term and a variable follows.
func dropUnitCoefficients(s string, math bool) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '1' || !unitCoefficientAt(s, i, math) {
			b.WriteByte(s[i])
			continue
		}
		// Skip the 1 and, in math, the one space between it and the variable.
		if i+1 < len(s) && s[i+1] == ' ' {
			i++
		}
	}
	return b.String()
}

func unitCoefficientAt(s string, i int, math bool) bool {
	// What precedes: the start of a term.
	if i > 0 {
		p := s[i-1]
		if isDigit(p) || p == '.' || isLetter(p) || p == '^' || p == '_' {
			return false
		}
		q := p
		if p == ' ' {
			j := i - 1
			for j > 0 && s[j-1] == ' ' {
				j--
			}
			if j == 0 {
				q = 0
			} else {
				q = s[j-1]
			}
		}
		if !strings.ContainsRune("(+-={,[", rune(q)) {
			if !math || !(q == 0 || q == ' ' || q == '}' || q == '\\' || q == '$') {
				return false
			}
		}
	} else if !math {
		return false
	}
	// What follows: a variable, directly or (in math) after one space.
	j := i + 1
	spaced := false
	if math && j < len(s) && s[j] == ' ' {
		j++
		spaced = true
	}
	if j >= len(s) || !(s[j] >= 'a' && s[j] <= 'z') {
		return false
	}
	var after byte
	if j+1 < len(s) {
		after = s[j+1]
	}
	// 1e-5 and 1e3 are numbers.
	if s[j] == 'e' && (isDigit(after) || after == '-' || after == '+') {
		return false
	}
	if isLetter(after) && (!math || spaced) {
		// "1st", "1 cm": a word, not a variable.
		return false
	}
	return true
}
