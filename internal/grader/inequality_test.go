package grader

import "testing"

// Q55: "Solve: 6x + 5 > 39" has the solution x > 17/3. The key was 6 and
// "x > 17/3" could not be parsed.
func TestInequalityAnswers(t *testing.T) {
	r := NewRouter()
	cases := []struct {
		key, typed string
		want       bool
	}{
		{"x > 17/3", "x > 17/3", true},
		{"x > 17/3", "x>17/3", true},
		{"x > 17/3", "17/3 < x", true},
		{"x > 17/3", "x > 34/6", true},
		{"x > 17/3", "x>5.67", true},
		{"x > 17/3", "x > 5.667", true},
		{"x > 17/3", "x \\gt \\frac{17}{3}", true},
		{"x > 17/3", "(17/3, oo)", true},
		{"x > 17/3", "(17/3, \\infty)", true},
		{"x > 17/3", "x > 5.7", false},  // one place is not the rounding asked for
		{"x > 17/3", "x > 5.66", false}, // a wrong rounding
		{"x > 17/3", "x >= 17/3", false},
		{"x > 17/3", "[17/3, oo)", false},
		{"x > 17/3", "x < 17/3", false},
		{"x > 17/3", "6", false},
		{"x > 17/3", "17/3", false},
		{"x > 17/3", "y > 17/3", false},
		{"x > 6", "x > 6", true},
		{"x > 6", "x > 6.0", true},
		{"x >= 2", "x ≥ 2", true},
		{"x >= 2", "x \\geq 2", true},
		{"x <= -3/2", "x <= -1.5", true},
		{"x <= -3/2", "-3/2 >= x", true},
		{"x <= -3/2", "x ≤ −3/2", true},
	}
	for _, c := range cases {
		res := r.Grade(GradingNumeric, c.key, c.typed)
		if res.Correct != c.want {
			t.Errorf("%q for key %q: correct=%v, want %v (%s)", c.typed, c.key, res.Correct, c.want, res.Feedback)
		}
	}
	if res := r.Grade(GradingNumeric, "x > 17/3", "17/3"); res.Feedback == "" || res.Feedback == "Incorrect" {
		t.Errorf("a bare bound should be told to write an inequality, got %q", res.Feedback)
	}
	for _, k := range []string{"6", "17/3", "(2, 3)", "x", "x^2 > 1", "1 < x < 3"} {
		if IsInequalityKey(k) {
			t.Errorf("%q is not a one-sided inequality key", k)
		}
	}
}
