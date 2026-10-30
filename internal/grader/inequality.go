package grader

import (
	"math"
	"regexp"
	"strings"
)

// Inequality answers (Q55). "Solve: 6x + 5 > 39" has the solution set
// x > 17/3; it was keyed 6 (the smallest whole number in the set), so the
// right answer "x > 17/3" could not even be parsed. A key written as a
// one-sided inequality in x is graded as one: the learner may write it as
// an inequality either way round ("x > 17/3", "17/3 < x"), with \gt, ≥ or
// \geq, or as an interval ("(17/3, ∞)"). The bound is compared as a number:
// exactly, or as a decimal rounded to two or more places ("x > 5.67").

var (
	ineqOpRe  = `(>=|<=|=>|=<|>|<|≥|≤)`
	ineqVarRe = regexp.MustCompile(`^([a-z])` + ineqOpRe + `(.+)$`)
	ineqRevRe = regexp.MustCompile(`^(.+?)` + ineqOpRe + `([a-z])$`)
	decimalRe = regexp.MustCompile(`^-?\d*\.(\d+)$`)
)

type halfLine struct {
	variable string
	upper    bool // x < b (or ≤): the set is bounded above
	closed   bool
	bound    float64
	text     string // the bound as written
}

// ineqText rewrites LaTeX and Unicode spellings of the relation signs.
func ineqText(s string) string {
	s = strings.NewReplacer(
		`\geq`, "≥", `\leq`, "≤", `\ge`, "≥", `\le`, "≤", `\gt`, ">", `\lt`, "<",
		"&gt;", ">", "&lt;", "<", "−", "-", "$", "", `\(`, "", `\)`, "",
		"{:", "", ":}", "",
	).Replace(s)
	return strings.Join(strings.Fields(strings.ToLower(s)), "")
}

func normaliseOp(op string) (upper, closed bool) {
	switch op {
	case ">":
		return false, false
	case ">=", "=>", "≥":
		return false, true
	case "<":
		return true, false
	default: // "<=", "=<", "≤"
		return true, true
	}
}

// parseHalfLine reads "x > b" or "b < x". ok is false for anything else,
// including a two-sided inequality.
func parseHalfLine(s string) (halfLine, bool) {
	s = ineqText(s)
	if strings.Contains(s, `\`) {
		// \frac{17}{3} and friends, read the way typed mathematics is.
		s = strings.ReplaceAll(NormalizeTypedMath(s, ""), " ", "")
	}
	if m := ineqVarRe.FindStringSubmatch(s); m != nil && !strings.ContainsAny(m[3], "<>≤≥") {
		v, ok := constEvalNumeric(m[3])
		if !ok {
			return halfLine{}, false
		}
		upper, closed := normaliseOp(m[2])
		return halfLine{m[1], upper, closed, v, m[3]}, true
	}
	if m := ineqRevRe.FindStringSubmatch(s); m != nil && !strings.ContainsAny(m[1], "<>≤≥") {
		v, ok := constEvalNumeric(m[1])
		if !ok {
			return halfLine{}, false
		}
		upper, closed := normaliseOp(m[2])
		// "b < x" is "x > b".
		return halfLine{m[3], !upper, closed, v, m[1]}, true
	}
	return halfLine{}, false
}

// IsInequalityKey reports a key written as a one-sided inequality in a
// variable, such as "x > 17/3".
func IsInequalityKey(expected string) bool {
	_, ok := parseHalfLine(expected)
	return ok
}

// boundMatches compares the learner's bound with the key's: equal as
// numbers, or a decimal of two or more places that is the key rounded.
func boundMatches(key float64, ans halfLine) bool {
	if sameEnd(key, ans.bound) {
		return true
	}
	m := decimalRe.FindStringSubmatch(ans.text)
	if m == nil || len(m[1]) < 2 {
		return false
	}
	half := 0.5 * math.Pow(10, -float64(len(m[1])))
	return math.Abs(key-ans.bound) <= half+1e-12
}

// GradeInequality grades an answer against a one-sided inequality key.
func GradeInequality(expected, answer string) Result {
	key, ok := parseHalfLine(expected)
	if !ok {
		return Result{Feedback: "Incorrect"}
	}
	if strings.TrimSpace(answer) == "" {
		return emptyResult()
	}
	ans, ok := parseHalfLine(answer)
	if !ok {
		// An interval is the same set written another way.
		if ivs, isIv := parseIntervals(setText(answer)); isIv && len(ivs) == 1 {
			iv := ivs[0]
			switch {
			case math.IsInf(iv.hi, 1) && !math.IsInf(iv.lo, 0):
				ans = halfLine{key.variable, false, iv.loIn, iv.lo, ""}
			case math.IsInf(iv.lo, -1) && !math.IsInf(iv.hi, 0):
				ans = halfLine{key.variable, true, iv.hiIn, iv.hi, ""}
			default:
				return Result{Feedback: "Incorrect"}
			}
			ok = true
		}
	}
	if !ok {
		if _, _, isNum := parseNumeric(strings.TrimSpace(answer)); isNum {
			return Result{Feedback: "Write the whole solution set as an inequality, e.g. " + key.variable + " > 7/2"}
		}
		return Result{Feedback: "Write the solution as an inequality, e.g. " + key.variable + " > 7/2"}
	}
	if ans.variable != key.variable {
		return Result{Feedback: "Solve for " + key.variable}
	}
	if ans.upper != key.upper || ans.closed != key.closed || !boundMatches(key.bound, ans) {
		return Result{Feedback: "Incorrect"}
	}
	return Result{Correct: true, Score: 1}
}
