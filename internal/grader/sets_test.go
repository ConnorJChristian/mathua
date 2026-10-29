package grader

import "testing"

// Set and interval answers as the math box exports them. Before sets.go the
// multiple-choice grader compared text, so every "should accept" row below
// except the verbatim ones was marked wrong.
func TestSetAnswersCompareAsSets(t *testing.T) {
	r := NewRouter()
	cases := []struct {
		expected, answer string
		want             bool
	}{
		{"{1,2,3}", "{1, 2, 3}", true},
		{"{1,2,3}", "{3,2,1}", true},
		{"{1,2,3}", "\\{1,2,3\\}", true},
		{"{1,2,3}", "{:{1,2,3}:}", true},
		{"{1,2,3}", "1, 2, 3", true},
		{"{1,2,3}", "{1,2,3,3}", true},
		{"{1,2,3}", "{1,2}", false},
		{"{1,2,3}", "{1,2,3,4}", false},
		{"{1,2,3}", "∅", false},
		{"∅", "{}", true},
		{"∅", "O/", true},
		{"∅", "\\emptyset", true},
		{"∅", "empty set", true},
		{"∅", "{0}", false},
		{"∅", "{∅}", false},
		{"{∅,{1},{2},{1,2}}", "{{2,1},{},{1},{2}}", true},
		{"{∅,{1},{2},{1,2}}", "{{1},{2},{1,2}}", false},
		{"{(1,a),(2,a)}", "{(2, a), (1, a)}", true},
		{"{(1,a),(2,a)}", "{(a,2),(1,a)}", false},
		{"{-1/2,3}", "{3, -0.5}", true},
	}
	for _, c := range cases {
		for _, gt := range []GradingType{GradingMultipleChoice, GradingSet} {
			if got := r.Grade(gt, c.expected, c.answer).Correct; got != c.want {
				t.Errorf("%s: key %q, answer %q: got %v, want %v", gt, c.expected, c.answer, got, c.want)
			}
		}
	}
}

func TestIntervalAnswersCompareAsSetsOfReals(t *testing.T) {
	r := NewRouter()
	cases := []struct {
		expected, answer string
		want             bool
	}{
		{"(1,\\infty)", "(1, inf)", true},
		{"(1,\\infty)", "(1, ∞)", true},
		{"(1,\\infty)", "(1, oo)", true},
		{"(1,\\infty)", "{:(1, oo):}", true},
		{"(1,\\infty)", "(1,infinity)", true},
		{"(1,\\infty)", "[1, oo)", false},
		{"(1,\\infty)", "(1, oo]", false},
		{"(1,\\infty)", "x > 1", false},
		{"(-\\infty,6]", "(-oo, 6]", true},
		{"(-\\infty,6]", "(-∞,6)", false},
		{"[-3,2)", "[-3, 2)", true},
		{"[-3,2)", "(-3, 2)", false},
		{"[-3,2)", "[-3, 2]", false},
		{"(1,7)", "(1, 7)", true},
		{"(1,7)", "(7, 1)", false},
		{"[0,1/2]", "[0, (1)/(2)]", true},
		{"[0,1/2]", "[0, 0.5]", true},
		{"(-\\infty,1)∪(3,\\infty)", "(3,oo)uu(-oo,1)", true},
		{"(-\\infty,1)∪(3,\\infty)", "(-inf,1) U (3,inf)", true},
		{"(-\\infty,1)∪(3,\\infty)", "(-oo,1]uu(3,oo)", false},
		{"(-\\infty,\\infty)", "RR", true},
		{"(-\\infty,\\infty)", "ℝ", true},
		{"[0,2]∪[2,5)", "[0,5)", true},
	}
	for _, c := range cases {
		for _, gt := range []GradingType{GradingMultipleChoice, GradingInterval} {
			if got := r.Grade(gt, c.expected, c.answer).Correct; got != c.want {
				t.Errorf("%s: key %q, answer %q: got %v, want %v", gt, c.expected, c.answer, got, c.want)
			}
		}
	}
}

// An ordered pair or a vector key is not an interval.
func TestPairsAndVectorsAreNotIntervals(t *testing.T) {
	r := NewRouter()
	if r.Grade(GradingTuple, "[1, 2]", "[1, 2)").Correct {
		t.Error("a tuple key [1, 2] accepted the interval [1, 2)")
	}
	if !r.Grade(GradingTuple, "(2, 3)", "2, 3").Correct {
		t.Error("the tuple (2, 3) no longer accepts 2, 3")
	}
	if LooksLikeInterval("(2, 3)", true) {
		t.Error("(2, 3) read as an interval in strict mode")
	}
}

// The math box reads a typed "oo" (AsciiMath for ∞) as two variables and
// exports "o o", with a space. setText matched "oo" before it removed spaces,
// so (-∞, 3) ∪ (3, ∞) typed the AsciiMath way was marked wrong, with the
// "write an interval" feedback.
func TestIntervalAcceptsMathBoxInfinity(t *testing.T) {
	r := NewRouter()
	cases := []struct {
		expected, answer string
		want             bool
	}{
		{"(-∞, 3) ∪ (3, ∞)", "(-o o,3)U(3,o o)", true},
		{"(-∞, 3) ∪ (3, ∞)", "(-oo,3)U(3,oo)", true},
		{"[0, ∞)", "[0,o o)", true},
		{"[0, ∞)", "[0, i n f)", true},
		{"ℝ", "R R", true},
		{"(-∞, 3) ∪ (3, ∞)", "(-o o,3)U(4,o o)", false},
		{"[0, ∞)", "[0,o o]", false},
	}
	for _, c := range cases {
		for _, gt := range []GradingType{GradingInterval, GradingMultipleChoice} {
			if got := r.Grade(gt, c.expected, c.answer).Correct; got != c.want {
				t.Errorf("%s: key %q, answer %q: got %v, want %v", gt, c.expected, c.answer, got, c.want)
			}
		}
	}
}
