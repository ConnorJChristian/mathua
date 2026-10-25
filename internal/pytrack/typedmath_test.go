package pytrack

import (
	"testing"

	"github.com/chuma-beep/mathua/internal/grader"
)

func TestNormalizeTypedMath(t *testing.T) {
	cases := []struct{ in, expected, want string }{
		{"3.6x^{2}", "3.6x^2", "3.6x^(2)"},
		{"2x2/(3x1)", "2x_2/(3x_1)", "2x_2/(3x_1)"},
		{"βx_3/(γx_2)", "beta*x_3/(gamma*x_2)", "beta*x_3/(gamma*x_2)"},
		{"\\frac{1}{28}", "1/28", "(1)/(28)"},
		{"x2 + 1", "x^2 + 1", "x2 + 1"}, // no subscripts expected: x2 left alone
	}
	for _, c := range cases {
		if got := normalizeTypedMath(c.in, c.expected); got != c.want {
			t.Errorf("normalizeTypedMath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// The spellings from the QA report grade right once normalised (needs the
// SymPy service; skipped where it is unavailable).
func TestTypedMathSpellingsGradeRight(t *testing.T) {
	r := grader.NewRouter()
	for _, c := range []struct{ expected, typed string }{
		{"3.6x^2", "3.6x^{2}"},
		{"2x_2/(3x_1)", "2x2/(3x1)"},
		{"1/28", "\\frac{1}{28}"},
		{"beta*x_3/(gamma*x_2)", "βx3/(γx2)"},
	} {
		res := r.Grade(grader.GradingSymbolic, normalizeTypedMath(c.expected, c.expected), normalizeTypedMath(c.typed, c.expected))
		if res.Unavailable {
			t.Skip("grading service unavailable")
		}
		if !res.Correct {
			t.Errorf("%q for %q graded wrong: %s", c.typed, c.expected, res.Feedback)
		}
	}
}
