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
