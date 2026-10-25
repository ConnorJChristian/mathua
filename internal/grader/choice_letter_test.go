package grader

import "testing"

func TestResolveChoiceLetter(t *testing.T) {
	q := "Which is a line?\n\nA) a segment\n\nB) a line through two points\n\nC) a ray"
	if got := ResolveChoiceLetter(q, "a line through two points", "b"); got != "a line through two points" {
		t.Errorf("b -> %q", got)
	}
	// A key that is itself a letter keeps the typed letter.
	if got := ResolveChoiceLetter(q, "B", "b"); got != "b" {
		t.Errorf("letter key changed the answer: %q", got)
	}
	// No such option: unchanged.
	if got := ResolveChoiceLetter(q, "x", "E"); got != "E" {
		t.Errorf("E -> %q", got)
	}
}

// The old rule accepted any letter that appeared as a word in the key.
func TestChoiceLetterIsNotMatchedAgainstTheKeysWords(t *testing.T) {
	r := NewRouter()
	if r.Grade(GradingMultipleChoice, "a line through two points", "a").Correct {
		t.Error(`"a" graded right because the key contains the word "a"`)
	}
	q := "Which?\n\nA) a segment\n\nB) a line through two points"
	if !r.Grade(GradingMultipleChoice, "a line through two points", ResolveChoiceLetter(q, "a line through two points", "B")).Correct {
		t.Error("the right letter, resolved, should grade right")
	}
	if r.Grade(GradingMultipleChoice, "a line through two points", ResolveChoiceLetter(q, "a line through two points", "A")).Correct {
		t.Error("the wrong letter, resolved, should grade wrong")
	}
}

// Q44: a one-character key is an option's text, not a letter key. The
// clicked "4" (option D) must stay "4", and typing D resolves to "4".
func TestResolveChoiceLetter_OneCharacterOptionKeys(t *testing.T) {
	r := NewRouter()
	q := "How many?\n\nA) 1\n\nB) 2\n\nC) 3\n\nD) 4"
	for _, typed := range []string{"4", "D", "d"} {
		if !r.Grade(GradingMultipleChoice, "4", ResolveChoiceLetter(q, "4", typed)).Correct {
			t.Errorf("%q should grade right for key 4", typed)
		}
	}
	for _, typed := range []string{"3", "C", "A"} {
		if r.Grade(GradingMultipleChoice, "4", ResolveChoiceLetter(q, "4", typed)).Correct {
			t.Errorf("%q should grade wrong for key 4", typed)
		}
	}
	z := "Which set is not countable?\n\nA) Z\n\nB) Q\n\nC) R\n\nD) N"
	if !r.Grade(GradingMultipleChoice, "R", ResolveChoiceLetter(z, "R", "R")).Correct {
		t.Error("clicking R (option C) for key R should grade right")
	}
	if !r.Grade(GradingMultipleChoice, "R", ResolveChoiceLetter(z, "R", "C")).Correct {
		t.Error("typing C for key R should grade right")
	}
	if r.Grade(GradingMultipleChoice, "R", ResolveChoiceLetter(z, "R", "Q")).Correct {
		t.Error("Q is option B, wrong for key R")
	}
}

// Q49: generated items accept the notation their prompts use.
func TestTypedNotationGradesRightOnGeneratedTypes(t *testing.T) {
	r := NewRouter()
	cases := []struct {
		gt              GradingType
		expected, typed string
	}{
		{GradingExpression, "8y^2+4", "8y^{2} + 4"},
		{GradingExpression, "3e^(3x)", "3e^{3x}"},
		{GradingExpression, "x/sqrt(x^2+5)", "x/\\sqrt{x^{2}+5}"},
		{GradingNumeric, "-1/12", "−1/12"},
		{GradingNumeric, "1/3", "\\frac{1}{3}"},
	}
	for _, c := range cases {
		res := r.Grade(c.gt, c.expected, c.typed)
		if res.Unavailable {
			t.Skip("grading service unavailable")
		}
		if !res.Correct {
			t.Errorf("%q for %q (%s) graded wrong: %s", c.typed, c.expected, c.gt, res.Feedback)
		}
	}
}
