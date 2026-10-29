package grader

import (
	"fmt"
	"math"
	"strings"
)

// Diagnose names the mistake, when the submitted answer makes it determinable
// from the expected one. It is a description of what happened, never a hint
// about what to do: it is only ever produced after an answer has been graded,
// and it never influences Correct.
//
// Returning "" is the normal case and means "nothing here we can state with
// certainty". Anything guessed from a fuzzy distance would be worse than
// silence — a learner told the wrong reason for a mistake learns the wrong
// lesson, which is the opposite of what an explanation is for.
func Diagnose(gradingType GradingType, expected, answer string) string {
	return DiagnoseIn(gradingType, false, expected, answer)
}

// DiagnoseIn is Diagnose for a question whose context is known. counting says
// the answer is a count (how many elements, terms, factors, edges...): only
// there does an answer one away name a mistake, a count or a boundary off by
// one. Anywhere else, an optimum at x = 4 answered as 5, two integers one
// apart are a coincidence, and calling it an off-by-one teaches the wrong
// lesson.
func DiagnoseIn(gradingType GradingType, counting bool, expected, answer string) string {
	if strings.TrimSpace(answer) == "" {
		return ""
	}
	switch gradingType {
	case GradingNumeric:
		return diagnoseNumeric(expected, answer, counting)
	case GradingComparison:
		return diagnoseComparison(expected, answer)
	case GradingTuple, GradingOrdering:
		return diagnoseSequence(expected, answer)
	default:
		return ""
	}
}

// diagnoseNumeric reports the mistakes that are visible in the numbers alone:
// a sign dropped, a count off by one, or an answer that is the right number
// rounded to fewer decimal places.
func diagnoseNumeric(expected, answer string, counting bool) string {
	e, eIsFloat, eOK := parseNumeric(normalise(expected))
	a, aIsFloat, aOK := parseNumeric(normalise(answer))
	if !eOK || !aOK {
		return ""
	}
	if e == a {
		// Equal to within tolerance — the numeric grader already called this
		// correct. Nothing to diagnose.
		return ""
	}
	switch {
	case e == -a:
		return fmt.Sprintf("You have the right number with the sign flipped — the answer is %s.", strings.TrimSpace(answer))
	case counting && !eIsFloat && !aIsFloat && e == math.Trunc(e) && a == math.Trunc(a) && math.Abs(e-a) == 1:
		return "You are exactly one away, so a count or a boundary is off by one."
	}
	// Fewer decimal places than the expected value usually means the answer was
	// rounded rather than found.
	if eIsFloat && aIsFloat {
		if decimalsOf(strings.TrimSpace(answer)) < decimalsOf(strings.TrimSpace(expected)) {
			return fmt.Sprintf("That is the answer rounded to fewer decimal places than it needs — %s keeps them.", strings.TrimSpace(expected))
		}
	}
	return ""
}

// diagnoseComparison names the direction. The three symbols are each other's
// mirror, so a reversed comparison is a distinct and common error.
func diagnoseComparison(expected, answer string) string {
	e := strings.TrimSpace(expected)
	a := strings.TrimSpace(answer)
	flipped := map[string]string{">": "<", "<": ">", "=": "="}
	if want, ok := flipped[e]; ok && a == want {
		return fmt.Sprintf("The comparison goes the other way round — it should be %s, not %s.", e, a)
	}
	return ""
}

// diagnoseSequence names the position, when exactly one of several values is
// wrong. Knowing which one is off is the difference between a fixable mistake
// and a shrug.
func diagnoseSequence(expected, answer string) string {
	e := splitValues(expected)
	a := splitValues(answer)
	if len(e) == 0 || len(e) != len(a) {
		return ""
	}
	var wrong []int
	for i := range e {
		if normalise(e[i]) != normalise(a[i]) {
			wrong = append(wrong, i)
		}
	}
	if len(wrong) != 1 {
		return ""
	}
	return fmt.Sprintf("Only the %s value is wrong; the rest match.", ordinal(wrong[0]+1))
}

func splitValues(s string) []string {
	s = strings.Trim(strings.TrimSpace(s), "()[]")
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func ordinal(n int) string {
	switch {
	case n%100 >= 11 && n%100 <= 13:
		return fmt.Sprintf("%dth", n)
	default:
		switch n % 10 {
		case 1:
			return fmt.Sprintf("%dst", n)
		case 2:
			return fmt.Sprintf("%dnd", n)
		case 3:
			return fmt.Sprintf("%drd", n)
		default:
			return fmt.Sprintf("%dth", n)
		}
	}
}

func decimalsOf(s string) int {
	s = strings.TrimSpace(s)
	if !strings.Contains(s, ".") {
		return 0
	}
	return len(s) - strings.Index(s, ".") - 1
}

// countingDomains and countingSubdomains are where a numeric answer is a count:
// arithmetic, counting and combinatorics, graphs (vertices, edges, colours) and
// number theory (divisors, residues), plus sequences, where an index or a
// number of terms is off by one.
var countingDomains = map[string]bool{
	"arithmetic":    true,
	"discrete_math": true,
	"graph_theory":  true,
	"number_theory": true,
}

// IsCountingContext reports whether a concept's numeric answers are counts, so
// an answer one away is a count or a boundary off by one.
func IsCountingContext(domain, subdomain string) bool {
	if countingDomains[domain] {
		return true
	}
	return strings.HasSuffix(subdomain, ".sequences") || strings.HasSuffix(subdomain, ".combinatorics") || strings.HasSuffix(subdomain, ".recurrence")
}
