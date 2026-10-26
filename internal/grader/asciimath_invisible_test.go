package grader

import "testing"

// The math input box exports invisible brackets as AsciiMath "{:" and ":}".
// A right total differential typed in the box arrived as
// "(6x y^3{:)d x+ (9x^2y^2-9y^2 )d y:}" and was rejected as disallowed.
func TestAsciiMathInvisibleBracketsAreDropped(t *testing.T) {
	r := NewRouter()
	key := "dz = (6*x*y^3) dx + (9*x^2*y^2 - 9*y^2) dy"
	if res := r.Grade(GradingType("symbolic"), key, "(6x y^3{:)d x+ (9x^2y^2-9y^2 )d y:}"); !res.Correct {
		t.Errorf("right answer from the math box marked wrong: %s", res.Feedback)
	}
	if res := r.Grade(GradingType("symbolic"), key, "(6x y^3{:)d x+ (9x^2y^2-9y )d y:}"); res.Correct {
		t.Error("a wrong dy coefficient was accepted")
	}
	if res := r.Grade(GradingType("numeric"), "12", "{:12:}"); !res.Correct {
		t.Errorf("{:12:} not read as 12: %s", res.Feedback)
	}
}

// Every type, not only the typed-math ones: a multiple-choice concept with a
// number or a word for its key is answered in the same math box.
func TestInvisibleBracketsDroppedForEveryType(t *testing.T) {
	r := NewRouter()
	for _, c := range []struct {
		gt       GradingType
		key, ans string
	}{
		{GradingMultipleChoice, "14", "{:14:}"},
		{GradingMultipleChoice, "increasing", "{:increasing:}"},
		{GradingComparison, "<", "{:<:}"},
	} {
		if res := r.Grade(c.gt, c.key, c.ans); !res.Correct {
			t.Errorf("%s: %q not read as %q: %s", c.gt, c.ans, c.key, res.Feedback)
		}
	}
	if r.Grade(GradingMultipleChoice, "14", "{:15:}").Correct {
		t.Error("{:15:} accepted for 14")
	}
}
