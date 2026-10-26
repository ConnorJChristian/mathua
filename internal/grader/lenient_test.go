package grader

import "testing"

// Answer forms a learner reasonably types grade the same as the canonical
// form (fork fix, from a simulated learner's 21 false "wrong" grades).
func TestLenientForms(t *testing.T) {
	r := NewRouter()
	cases := []struct {
		gt       GradingType
		expected string
		answer   string
		want     bool
	}{
		{GradingNumeric, "6", "x = 6", true},
		{GradingNumeric, "6", "x=6", true},
		{GradingNumeric, "6", "7", false},
		{GradingNumeric, "61/3", "20.333", true},
		{GradingNumeric, "61/3", "20.3", false},
		{GradingSymbolic, "decreasing", "Decreasing", true},
		{GradingSymbolic, "concave up", "Concave Up", true},
		{GradingSymbolic, "decreasing", "increasing", false},
		{GradingTuple, "[[9,6],[0,0]]", "[[9, 6], [0, 0]]", true},
		{GradingTuple, "[[9,6],[0,0]]", "[[9, 6], [0, 1]]", false},
		{GradingMultipleChoice, "B", "b", true},
		{GradingMultipleChoice, "y=x^2/2+C", "y = x^2/2 + C", true},
		{GradingMultipleChoice, "y=x^2/2+C", "x^2/2 + c", true},
		// The inverse named before it, as a learner copies the question.
		{GradingSymbolic, "(x - 4)/3", "f^(-1)(x) = (x-4)/3", true},
		{GradingSymbolic, "(x - 4)/3", "f^{-1}(x)=(x-4)/3", true},
		{GradingSymbolic, "(x - 4)/3", "f^(-1)(x) = (x+4)/3", false},
	}
	for _, c := range cases {
		got := r.Grade(c.gt, c.expected, c.answer)
		if got.Unavailable {
			t.Logf("skip (grader unavailable): %q vs %q", c.expected, c.answer)
			continue
		}
		if got.Correct != c.want {
			t.Errorf("%s: expected %q answer %q: got %v, want %v (%s)", c.gt, c.expected, c.answer, got.Correct, c.want, got.Feedback)
		}
	}
}
