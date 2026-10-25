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
