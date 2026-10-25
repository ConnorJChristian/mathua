package grader

import "testing"

func TestWordAnswers(t *testing.T) {
	r := NewRouter()
	cases := []struct {
		gt         GradingType
		key, typed string
		want       bool
	}{
		{GradingSymbolic, "corner", "a corner", true},
		{GradingSymbolic, "corner", "Corner.", true},
		{GradingSymbolic, "corner", "cusp", false},
		{GradingSymbolic, "convex (concave up)", "convex", true},
		{GradingSymbolic, "convex (concave up)", "concave up", true},
		{GradingSymbolic, "convex (concave up)", "concave", false},
		{GradingSymbolic, "neither (inflection point)", "neither", true},
		{GradingSymbolic, "no", "no, it has a cusp", true},
		{GradingSymbolic, "no", "yes", false},
		{GradingMultipleChoice, "base case", "the base case", true},
		{GradingSymbolic, "infinity", "∞", true},
		{GradingSymbolic, "infinity", "oo", true},
	}
	for _, c := range cases {
		res := r.Grade(c.gt, c.key, c.typed)
		if res.Unavailable {
			t.Fatalf("%q for %q: grader unavailable (%s)", c.typed, c.key, res.Feedback)
		}
		if res.Correct != c.want {
			t.Errorf("%q for %q: correct=%v, want %v", c.typed, c.key, res.Correct, c.want)
		}
	}
}

func TestWordKeyExcludesFunctionsOfAVariable(t *testing.T) {
	for _, k := range []string{"cos(x)", "sinh(x)", "degrees of freedom (k)", "x", "e"} {
		if isWordKey(k) {
			t.Errorf("%q is mathematics, not a word key", k)
		}
	}
	for _, k := range []string{"corner", "convex (concave up)", "no", "probability density function (PDF)"} {
		if !isWordKey(k) {
			t.Errorf("%q should be a word key", k)
		}
	}
}

func TestTupleComponentsCompareByKind(t *testing.T) {
	r := NewRouter()
	cases := []struct {
		key, typed string
		want       bool
	}{
		{"(2x, 2y)", "(2*x, 2*y)", true},
		{"(2x, 2y)", "(2y, 2x)", false},
		{"(1/2, 3)", "(0.5, 3)", true},
		{"(1, 2)", "(1, 3)", false},
	}
	for _, c := range cases {
		res := r.Grade(GradingTuple, c.key, c.typed)
		if res.Unavailable {
			t.Skip("grading service unavailable")
		}
		if res.Correct != c.want {
			t.Errorf("%q for %q: correct=%v, want %v", c.typed, c.key, res.Correct, c.want)
		}
	}
}
