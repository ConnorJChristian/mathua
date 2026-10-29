package grader

import (
	"math"
	"sort"
	"strconv"
	"strings"
)

// Set and interval answers.
//
// A learner answers "A ∪ B" with a roster such as {1, 2, 3} and an inequality
// with an interval such as (2, 5] or [0, ∞). The math box exports these as
// AsciiMath-ish text: infinity as "oo", the empty set as "O/", ℝ as "RR",
// invisible brackets as "{:" ... ":}". The other graders compare text, so a
// roster in another order, {} for ∅, or "inf" for \infty was marked wrong.
// These graders compare the mathematical objects: a finite set ignores order
// and repeats, and an interval (or a union of intervals) compares its
// endpoints as numbers and its brackets as included or not.

// setText rewrites the spellings of set and interval notation to one form.
func setText(s string) string {
	// Spaces go first: the math box exports a typed "oo" (or "RR") as two
	// variables, "o o", so the spellings below must match across them.
	s = strings.Join(strings.Fields(s), "")
	s = strings.NewReplacer(
		"{:", "", ":}", "",
		"\\left", "", "\\right", "",
		"\\{", "{", "\\}", "}", "\\lbrace", "{", "\\rbrace", "}",
		"\\lbrack", "[", "\\rbrack", "]",
		"−", "-", "\\,", "", "\\;", "", "\\!", "",
		"\\mathbb{R}", "ℝ", "\\mathbbR", "ℝ", "\\R", "ℝ", "RR", "ℝ",
		"\\emptyset", "∅", "\\varnothing", "∅", "O/", "∅", "ø", "∅", "Ø", "∅",
		"\\cup", "∪", "uu", "∪",
		"\\infty", "∞", "infinity", "∞", "infty", "∞", "inf", "∞", "oo", "∞",
	).Replace(s)
	return s
}

var emptyWords = map[string]bool{
	"emptyset": true, "theemptyset": true, "empty": true, "none": true, "noelements": true,
}

// splitTop splits s on sep where no bracket is open.
func splitTop(s string, sep rune) []string {
	var parts []string
	depth, start := 0, 0
	for i, r := range s {
		switch r {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		}
		if r == sep && depth == 0 {
			parts = append(parts, s[start:i])
			start = i + len(string(r))
		}
	}
	return append(parts, s[start:])
}

// setElements parses a roster. ok is false when s is not a set.
func setElements(s string) (map[string]bool, bool) {
	if s == "∅" || s == "{}" || emptyWords[strings.ToLower(s)] {
		return map[string]bool{}, true
	}
	if len(s) < 2 || s[0] != '{' || s[len(s)-1] != '}' {
		return nil, false
	}
	// The braces must enclose the whole roster: "{1}∪{2}" is not one set.
	if !enclosed(s) {
		return nil, false
	}
	inner := s[1 : len(s)-1]
	out := map[string]bool{}
	for _, el := range splitTop(inner, ',') {
		if el == "" {
			return nil, false
		}
		out[canonElement(el)] = true
	}
	return out, true
}

// enclosed reports that the bracket opening s closes at its last character.
func enclosed(s string) bool {
	depth := 0
	for i, r := range s {
		switch r {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
			if depth == 0 && i != len(s)-1 {
				return false
			}
		}
	}
	return depth == 0
}

func balanced(s string) bool {
	depth := 0
	for _, r := range s {
		switch r {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return depth == 0
}

// canonElement is one spelling per element: a number by value, a set by its
// sorted elements, an ordered pair by its canonical coordinates.
func canonElement(el string) string {
	if v, ok := constEvalNumeric(el); ok {
		return strconv.FormatFloat(v, 'g', 12, 64)
	}
	if m, ok := setElements(el); ok {
		return canonSet(m)
	}
	if len(el) >= 2 && el[0] == '(' && el[len(el)-1] == ')' && balanced(el[1:len(el)-1]) {
		parts := splitTop(el[1:len(el)-1], ',')
		if len(parts) > 1 {
			for i, p := range parts {
				parts[i] = canonElement(p)
			}
			return "(" + strings.Join(parts, ",") + ")"
		}
	}
	return el
}

func canonSet(m map[string]bool) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return "{" + strings.Join(keys, ",") + "}"
}

// LooksLikeSet reports an expected answer written as a roster or ∅.
func LooksLikeSet(expected string) bool {
	e := setText(expected)
	if e == "∅" {
		return true
	}
	_, ok := setElements(e)
	return ok && strings.HasPrefix(e, "{")
}

// GradeSet compares two finite sets, ignoring order, repeats and spacing. A
// roster typed without its braces ("1, 2, 3") is read as the set.
func GradeSet(expected, answer string) Result {
	res, _ := gradeSet(expected, answer)
	return res
}

// gradeSet also reports whether the answer was a set at all.
func gradeSet(expected, answer string) (Result, bool) {
	em, ok := setElements(setText(expected))
	if !ok {
		return Result{Feedback: "Incorrect"}, false
	}
	a := setText(answer)
	if a == "" {
		return emptyResult(), false
	}
	am, ok := setElements(a)
	if !ok && !strings.HasPrefix(a, "{") {
		am, ok = setElements("{" + a + "}")
	}
	if !ok {
		return Result{Feedback: "Write the set as a list in braces, such as {1, 2, 3}, or ∅ for the empty set"}, false
	}
	if canonSet(em) == canonSet(am) {
		return Result{Correct: true, Score: 1}, true
	}
	return Result{Feedback: "Incorrect"}, true
}

// interval is one piece of a union of intervals.
type interval struct {
	lo, hi     float64
	loIn, hiIn bool
}

func endpoint(s string) (float64, bool) {
	switch s {
	case "∞", "+∞":
		return math.Inf(1), true
	case "-∞":
		return math.Inf(-1), true
	}
	return constEvalNumeric(s)
}

// parseIntervals reads an interval, a union of intervals joined by ∪ (or U),
// or ℝ. ok is false when s is not one, including a closed bracket on ∞ and an
// empty interval such as (3, 3).
func parseIntervals(s string) ([]interval, bool) {
	if s == "ℝ" || s == "R" {
		return []interval{{math.Inf(-1), math.Inf(1), false, false}}, true
	}
	pieces := splitTop(s, '∪')
	if len(pieces) == 1 {
		pieces = splitTop(s, 'U')
	}
	var out []interval
	for _, p := range pieces {
		if len(p) < 5 || !strings.ContainsRune("[(", rune(p[0])) || !strings.ContainsRune("])", rune(p[len(p)-1])) {
			return nil, false
		}
		ends := splitTop(p[1:len(p)-1], ',')
		if len(ends) != 2 || !balanced(p[1:len(p)-1]) {
			return nil, false
		}
		lo, ok1 := endpoint(ends[0])
		hi, ok2 := endpoint(ends[1])
		if !ok1 || !ok2 {
			return nil, false
		}
		iv := interval{lo, hi, p[0] == '[', p[len(p)-1] == ']'}
		if (math.IsInf(lo, 0) && iv.loIn) || (math.IsInf(hi, 0) && iv.hiIn) {
			return nil, false
		}
		if lo > hi || (lo == hi && !(iv.loIn && iv.hiIn)) {
			return nil, false
		}
		out = append(out, iv)
	}
	return mergeIntervals(out), true
}

func mergeIntervals(in []interval) []interval {
	sort.Slice(in, func(i, j int) bool {
		if in[i].lo != in[j].lo {
			return in[i].lo < in[j].lo
		}
		return in[i].loIn && !in[j].loIn
	})
	var out []interval
	for _, iv := range in {
		if n := len(out); n > 0 {
			last := &out[n-1]
			if iv.lo < last.hi || (iv.lo == last.hi && (iv.loIn || last.hiIn)) {
				if iv.hi > last.hi || (iv.hi == last.hi && iv.hiIn) {
					last.hi, last.hiIn = iv.hi, iv.hiIn
				}
				continue
			}
		}
		out = append(out, iv)
	}
	return out
}

func sameEnd(a, b float64) bool {
	if math.IsInf(a, 0) || math.IsInf(b, 0) {
		return a == b
	}
	return math.Abs(a-b) <= 1e-9*math.Max(1, math.Abs(a))
}

// LooksLikeInterval reports an expected answer in interval notation. A bare
// "(a, b)" is ambiguous with an ordered pair, so it counts only when strict
// is false; ∞, ∪, ℝ or a square bracket make it an interval either way.
func LooksLikeInterval(expected string, strict bool) bool {
	e := setText(expected)
	if _, ok := parseIntervals(e); !ok {
		return false
	}
	if !strict {
		return true
	}
	return strings.ContainsAny(e, "∞∪ℝ[]") || e == "R"
}

// GradeInterval compares intervals, or unions of intervals, as sets of reals.
func GradeInterval(expected, answer string) Result {
	res, _ := gradeInterval(expected, answer)
	return res
}

// gradeInterval also reports whether the answer was in interval notation.
func gradeInterval(expected, answer string) (Result, bool) {
	ev, ok := parseIntervals(setText(expected))
	if !ok {
		return Result{Feedback: "Incorrect"}, false
	}
	a := setText(answer)
	if a == "" {
		return emptyResult(), false
	}
	av, ok := parseIntervals(a)
	if !ok {
		return Result{Feedback: "Write an interval such as (2, 5] or [0, ∞); ∞ always takes a round bracket"}, false
	}
	if len(av) != len(ev) {
		return Result{Feedback: "Incorrect"}, true
	}
	for i := range ev {
		if !sameEnd(ev[i].lo, av[i].lo) || !sameEnd(ev[i].hi, av[i].hi) ||
			ev[i].loIn != av[i].loIn || ev[i].hiIn != av[i].hiIn {
			return Result{Feedback: "Incorrect"}, true
		}
	}
	return Result{Correct: true, Score: 1}, true
}

// gradeSetLike grades an expected set or interval as one. The explicit set
// and interval types always do. Otherwise it applies only where the key is
// unmistakably a set or an interval, and only when the answer is written as
// one too; anything else (an ordered pair typed as "2, 3", say) is left to
// the concept's own grader. A tuple key such as [1, 2] is a vector, never an
// interval.
func gradeSetLike(t GradingType, expected, answer string) (Result, bool) {
	switch t {
	case GradingSet:
		return GradeSet(expected, answer), true
	case GradingInterval:
		return GradeInterval(expected, answer), true
	case GradingMultipleChoice, GradingSymbolic, GradingExpression, GradingTuple:
	default:
		return Result{}, false
	}
	if LooksLikeSet(expected) {
		if res, parsed := gradeSet(expected, answer); parsed {
			return res, true
		}
		return Result{}, false
	}
	if t != GradingTuple && LooksLikeInterval(expected, t != GradingMultipleChoice) {
		if res, parsed := gradeInterval(expected, answer); parsed {
			return res, true
		}
	}
	return Result{}, false
}
