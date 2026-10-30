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
		// Q52 recurred: the live item's key is a list, not "corner".
		{GradingSymbolic, "corner/cusp/vertical tangent/discontinuity", "corner", true},
		{GradingSymbolic, "corner/cusp/vertical tangent/discontinuity", "a corner", true},
		{GradingSymbolic, "corner/cusp/vertical tangent/discontinuity", "Cusp", true},
		{GradingSymbolic, "corner/cusp/vertical tangent/discontinuity", "a vertical tangent", true},
		{GradingSymbolic, "corner/cusp/vertical tangent/discontinuity", "discontinuity", true},
		{GradingSymbolic, "corner/cusp/vertical tangent/discontinuity", "smooth", false},
		{GradingSymbolic, "corner/cusp/vertical tangent/discontinuity", "tangent", false},
		{GradingSymbolic, "stable spiral (spiral sink/stable focus)", "stable focus", true},
		{GradingSymbolic, "stable spiral (spiral sink/stable focus)", "unstable spiral", false},
		{GradingSymbolic, "convex (concave up)", "it is convex", true},
		{GradingSymbolic, "convex (concave up)", "it is concave", false},
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
	for _, k := range []string{"cos(x)", "sinh(x)", "degrees of freedom (k)", "x", "e", "x-y", "wz-y x"} {
		if isWordKey(k) {
			t.Errorf("%q is mathematics, not a word key", k)
		}
	}
	for _, k := range []string{"1/2", "x/y", "a/b/c", "dy/dx"} {
		if wordListParts(k) != nil {
			t.Errorf("%q is mathematics, not a list of word alternatives", k)
		}
	}
	for _, k := range []string{"corner", "convex (concave up)", "no", "probability density function (PDF)", "well-defined", "x-intercept", "ad-hoc"} {
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

// A difference of variables is graded as mathematics (fork fix: "wz-yx" was
// compared as words with the determinant "wz-y x" and refused).
func TestDifferenceOfVariablesIsMath(t *testing.T) {
	if !sympyAvailable(t) {
		t.Skip("SymPy not available")
	}
	r := NewRouter()
	for _, c := range []struct {
		key, typed string
		want       bool
	}{
		{"wz-y x", "wz-yx", true},
		{"wz-y x", "w*z-x*y", true},
		{"wz-y x", "wz+yx", false},
		{"x-y", "-y+x", true},
		{"x-y", "y-x", false},
	} {
		if got := r.Grade(GradingExpression, c.key, c.typed); got.Correct != c.want || got.Unavailable {
			t.Errorf("%q for %q: %+v, want correct=%v", c.typed, c.key, got, c.want)
		}
	}
}
