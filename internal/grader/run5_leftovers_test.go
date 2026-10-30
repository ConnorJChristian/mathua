package grader

import "testing"

// Leftovers from the 2026-10-25 run-5 review that never became issues:
// right answers refused for their spelling.
func TestRun5SpellingLeftovers(t *testing.T) {
	r := NewRouter()
	cases := []struct {
		gt         GradingType
		key, typed string
		want       bool
	}{
		{GradingMultipleChoice, "no", "not continuous", true},
		{GradingMultipleChoice, "no", "it is not continuous", true},
		{GradingMultipleChoice, "yes", "not continuous", false},
		{GradingMultipleChoice, "no", "not", false},
		{GradingSymbolic, "decreasing", "strictly decreasing", true},
		{GradingSymbolic, "decreasing", "strictly increasing", false},
		{GradingSymbolic, "decreasing", "not decreasing", false},
		{GradingMultipleChoice, "0", "0.0", true},
		{GradingMultipleChoice, "0", "0.01", false},
		{GradingTuple, "(-2, 6)", "<-2, 6>", true},
		{GradingTuple, "(-2, 6)", "[-2, 6]", true},
		{GradingTuple, "(-2, 6)", "[6, -2]", false},
	}
	for _, c := range cases {
		res := r.Grade(c.gt, c.key, c.typed)
		if res.Unavailable {
			t.Skipf("grader unavailable: %s", res.Feedback)
		}
		if res.Correct != c.want {
			t.Errorf("%q for %q (%s): correct=%v, want %v (%s)", c.typed, c.key, c.gt, res.Correct, c.want, res.Feedback)
		}
	}
	if !sympyAvailable(t) {
		t.Skip("SymPy not available")
	}
	for _, c := range []struct {
		typed string
		want  bool
	}{{`\frac{38}{3}`, true}, {`\dfrac{38}{3}`, true}, {`\frac{37}{3}`, false}} {
		if res := r.Grade(GradingMultipleChoice, "38/3", c.typed); res.Correct != c.want {
			t.Errorf("%q for the option 38/3: correct=%v, want %v (%s)", c.typed, res.Correct, c.want, res.Feedback)
		}
	}
}
