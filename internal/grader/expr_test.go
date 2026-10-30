package grader

import "testing"

// A learner may type the arithmetic they did rather than its result. These pin
// the strings that were reported from production as silently wrong, and the
// shapes that must stay unreadable so a list or a formula is not mistaken for a
// number.
func TestNumericReadsAnArithmeticAnswer(t *testing.T) {
	r := NewRouter()
	cases := []struct{ expected, answer string }{
		{"-1", "3-4"},
		{"-1", "3 - 4"},
		{"6", "2*3"},
		{"6", "2\u00d73"},
		{"4", "8/2"},
		{"4", "8\u00f72"},
		{"13", "3+2*5"},
		{"16", "(3+1)^2"},
		{"-5", "-5"},
		{"5", "+5"},
	}
	for _, c := range cases {
		if got := r.Grade(GradingNumeric, c.expected, c.answer); !got.Correct {
			t.Errorf("expected %q accepted %q, feedback %q", c.expected, c.answer, got.Feedback)
		}
	}
}

func TestNumericRefusesAnExpressionItCannotRead(t *testing.T) {
	r := NewRouter()
	cases := []struct{ expected, answer string }{
		{"5", "6,8"},
		{"3", "x+2"},
		{"3", "3+"},
		{"4", "8/0"},
		{"5", "(3+2"},
	}
	for _, c := range cases {
		got := r.Grade(GradingNumeric, c.expected, c.answer)
		if got.Correct {
			t.Errorf("expected %q accepted unreadable answer %q", c.expected, c.answer)
		}
	}
}
