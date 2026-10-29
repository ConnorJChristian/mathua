package grader

import "testing"

// Q50: places decide, not a window around the key.
func TestNumericDecimalPlaces(t *testing.T) {
	r := NewRouter()
	cases := []struct {
		key, typed string
		want       bool
	}{
		{"12.61", "12.61", true},
		{"12.61", "12.610001", true},
		{"12.61", "12.62", false},
		{"0.2", "0.25", false},
		{"0.2", "0.20", true},
		{"0.01", "0.015", false},
		{"3.5", "3.55", false},
		{"2.666667", "2.67", true},
		{"2.666667", "2.667", true},
		{"2.666667", "8/3", true},
		{"2.666667", "2.6", false},
		{"2.666667", "2.7", false},
		{"5", "5.0", true},
		{"5", "5.1", false},
		{"1/3", "0.333", true},
	}
	for _, c := range cases {
		if got := r.Grade(GradingNumeric, c.key, c.typed).Correct; got != c.want {
			t.Errorf("key %s, typed %s: correct=%v, want %v", c.key, c.typed, got, c.want)
		}
	}
}

// A fraction key answered as a rounded decimal passes the expression grader
// as it passes the numeric one (fork fix): which grader a generator declared
// must not decide whether 0.333 is 1/3.
func TestFractionKeyAcceptsRoundedDecimalInExpressionGrading(t *testing.T) {
	r := NewRouter()
	for _, gt := range []GradingType{GradingNumeric, GradingExpression, GradingSymbolic} {
		if !r.Grade(gt, "-1/9", "-0.111").Correct {
			t.Errorf("%s: -0.111 refused for -1/9", gt)
		}
		if r.Grade(gt, "-1/9", "-0.11").Correct {
			t.Errorf("%s: -0.11 accepted for -1/9", gt)
		}
		if r.Grade(gt, "1/3", "0.334").Correct {
			t.Errorf("%s: 0.334 accepted for 1/3", gt)
		}
		if r.Grade(gt, "1/3", "0.3339").Correct {
			t.Errorf("%s: 0.3339 accepted for 1/3", gt)
		}
		// A tiny fraction: the window must scale, so the opposite sign and
		// a bare zero are refused.
		if r.Grade(gt, "1/6561", "-0.000152").Correct {
			t.Errorf("%s: -0.000152 accepted for 1/6561", gt)
		}
		if r.Grade(gt, "1/6561", "0.000").Correct {
			t.Errorf("%s: 0.000 accepted for 1/6561", gt)
		}
		if !r.Grade(gt, "1/6561", "0.000152").Correct {
			t.Errorf("%s: 0.000152 refused for 1/6561", gt)
		}
		if !r.Grade(gt, "61/3", "20.333").Correct {
			t.Errorf("%s: 20.333 refused for 61/3", gt)
		}
	}
}

// Superscript digits typed from a keyboard are powers (fork fix).
func TestSuperscriptDigitsArePowers(t *testing.T) {
	r := NewRouter()
	if !r.Grade(GradingExpression, "3x^2 + 2x", "3x² + 2x").Correct {
		t.Error("3x² refused for 3x^2")
	}
	if !r.Grade(GradingExpression, "x^3*y^2", "x³y²").Correct {
		t.Error("x³y² refused for x^3*y^2")
	}
	if r.Grade(GradingExpression, "3x^2 + 2x", "3x³ + 2x").Correct {
		t.Error("3x³ accepted for 3x^2")
	}
}

// A key written with Greek letters is read as the learner's answer is: the
// key's own text grades correct (fork fix; stat.dist.beta's tuple key).
func TestExpectedAnswerIsReadLikeTheTypedOne(t *testing.T) {
	r := NewRouter()
	key := "α (alpha) and β (beta)"
	if !r.Grade(GradingTuple, key, key).Correct {
		t.Errorf("the key %q does not grade itself", key)
	}
	if !r.Grade(GradingTuple, key, "alpha (alpha) and beta (beta)").Correct {
		t.Error("the spelled-out key refused")
	}
	if !r.Grade(GradingExpression, "x^{2}+1", "x^2+1").Correct {
		t.Error("x^2+1 refused for a LaTeX key x^{2}+1")
	}
}
