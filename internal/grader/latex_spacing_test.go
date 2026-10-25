package grader

import "testing"

// A learner who copies the shown LaTeX style writes thin spaces before
// differentials ("3y^{2}\,dx"). The spacing commands carry no meaning and must
// not make a right answer "disallowed".
func TestLatexSpacingCommandsAreIgnored(t *testing.T) {
	r := NewRouter()
	key := "dz = (3*y^2) dx + (6*x*y + 2*y) dy"
	for _, a := range []string{`3y^{2}\,dx+(6xy+2y)\,dy`, `3y^2\;dx + (6xy+2y)\!dy`, `3y^2\ dx+(6xy+2y)\:dy`} {
		if res := r.Grade(GradingType("symbolic"), key, a); !res.Correct {
			t.Errorf("%q marked wrong: %s", a, res.Feedback)
		}
	}
	if res := r.Grade(GradingType("symbolic"), key, `3y^{2}\,dx+(6x+2y)\,dy`); res.Correct {
		t.Errorf("a wrong dy coefficient was accepted")
	}
}
