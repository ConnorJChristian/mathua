package grader

import (
	"fmt"
	"regexp"
	"strings"
)

var tupleRe = regexp.MustCompile(`[();,]+`)

type tupleGrader struct{}

func (g *tupleGrader) grade(expected, answer string) Result {
	e := normaliseTuple(expected)
	a := normaliseTuple(answer)
	if len(e) == 0 || len(a) == 0 {
		return emptyResult()
	}
	if len(e) != len(a) {
		return Result{Correct: false, Score: 0, Feedback: fmt.Sprintf("Expected %s, got %s", strings.Join(e, ","), strings.Join(a, ","))}
	}
	for i := range e {
		if !tupleElementMatches(e[i], a[i]) {
			return Result{Correct: false, Score: 0, Feedback: fmt.Sprintf("Mismatch at position %d", i+1)}
		}
	}
	return Result{Correct: true, Score: 1}
}

// tupleElementMatches compares one component by what it is, not as text: a
// number as a number ("0.5" for "1/2"), an expression as mathematics
// ("2*x" for "2x"; the gradient's "(2*x, 2*y)" was refused), a word as words.
func tupleElementMatches(e, a string) bool {
	if strings.EqualFold(e, a) {
		return true
	}
	if _, _, ok := parseNumeric(normalise(e)); ok {
		if _, _, ok2 := parseNumeric(normalise(a)); ok2 {
			return (&numericGrader{}).grade(e, a).Correct
		}
		return false
	}
	if isWordKey(e) {
		return wordAnswerMatches(e, a)
	}
	if strings.ContainsAny(e, "abcdefghijklmnopqrstuvwxyz") {
		return gradeSymPy(e, a).Correct
	}
	return false
}

func normaliseTuple(s string) []string {
	s = strings.TrimSpace(s)
	s = tupleRe.ReplaceAllString(s, ",")
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
