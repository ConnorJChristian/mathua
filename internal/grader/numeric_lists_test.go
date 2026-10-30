package grader

import "testing"

// A comma used to be deleted before anything was parsed, on the assumption that
// it only ever grouped thousands. That turned `6,8` into 68, merged the parts of
// an expanded number into one, and made a decimal comma unanswerable. Learners
// reported all three, so these pin the readings that must survive.

func TestNumericListsIgnoreOrder(t *testing.T) {
	r := NewRouter()
	for _, tc := range []struct{ expected, answer string }{
		{"6,8", "8,6"},
		{"7,10", "10,7"},
		{"(4,-3)", "-3,4"},
		{"-3,4", "-3, 4"},
		{"6;8", "8,6"},
		{"[4,-3]", "( -3 , 4 )"},
		{"2,-2", "-2,2"},
		// Report 249: x^2 - 1 = 0 was answered -1,1 while the key was 1,-1.
		{"1,-1", "-1,1"},
		{"2,-2", "x = -2, x = 2"},
		{"1,2,3", "3,1,2"},
		{"1/2,-1/2", "-1/2,1/2"},
	} {
		if res := r.Grade(GradingNumeric, tc.expected, tc.answer); !res.Correct {
			t.Errorf("expected %q and %q to match, got %q", tc.expected, tc.answer, res.Feedback)
		}
	}
}

func TestNumericListLengthMustMatch(t *testing.T) {
	r := NewRouter()
	for _, tc := range []struct{ expected, answer string }{
		{"1,2", "1,2,3"},
		{"6,8", "6,9"},
		// One value is not a list, so a learner cannot hit a pair by typing the
		// digits of a single number.
		{"12", "1,2"},
	} {
		if res := r.Grade(GradingNumeric, tc.expected, tc.answer); res.Correct {
			t.Errorf("expected %q and %q to differ", tc.expected, tc.answer)
		}
	}
}

func TestNumericThousandsAndExpandedForm(t *testing.T) {
	r := NewRouter()
	for _, tc := range []struct{ expected, answer string }{
		{"1,000", "1000"},
		{"1000", "1,000"},
		{"1000,200,0,3", "1000, 200, 0, 3"},
	} {
		if res := r.Grade(GradingNumeric, tc.expected, tc.answer); !res.Correct {
			t.Errorf("expected %q and %q to match, got %q", tc.expected, tc.answer, res.Feedback)
		}
	}
}

func TestNumericDecimalCommaAnswersADot(t *testing.T) {
	r := NewRouter()
	for _, tc := range []struct{ expected, answer string }{
		{"3.5", "3,5"},
		{"75.36", "75,36"},
		{"-0.25", "-0,25"},
	} {
		if res := r.Grade(GradingNumeric, tc.expected, tc.answer); !res.Correct {
			t.Errorf("expected %q and %q to match, got %q", tc.expected, tc.answer, res.Feedback)
		}
	}
}

func TestNumericAcceptsAVariablePrefix(t *testing.T) {
	r := NewRouter()
	for _, tc := range []struct{ expected, answer string }{
		{"5", "x=5"},
		{"5", "x = 5"},
		{"-3", "y=-3"},
	} {
		if res := r.Grade(GradingNumeric, tc.expected, tc.answer); !res.Correct {
			t.Errorf("expected %q and %q to match, got %q", tc.expected, tc.answer, res.Feedback)
		}
	}
}

// A value that cannot be read at all is a different problem from a wrong one,
// and the message says so.
func TestNumericUnreadableAnswerIsNotCalledWrong(t *testing.T) {
	r := NewRouter()
	if res := r.Grade(GradingNumeric, "4", `12\times4`); res.Feedback != "Could not parse numeric values" {
		t.Errorf("feedback = %q, want the parse failure", res.Feedback)
	}
}
