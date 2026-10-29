package precalculus

import "testing"

// A numeric answer to a precalculus generator takes the spellings the shared
// numeric grader takes (fork fix): a thousands separator, a typeset minus
// and a fraction were "not a number".
func TestGradeNumReadsSharedSpellings(t *testing.T) {
	for _, tc := range []struct{ key, answer string }{
		{"4372", "4,372"}, {"-10", "−10"}, {"1.88", "47/25"}, {"-1", "x = -1"}, {"7", "7.0"},
	} {
		if !gradeNum(tc.key, tc.answer, 0.01).Correct {
			t.Errorf("%q refused for %q", tc.answer, tc.key)
		}
	}
	for _, tc := range []struct{ key, answer string }{{"4372", "4,382"}, {"-10", "10"}, {"1.88", "1.9"}, {"7", ""}} {
		if gradeNum(tc.key, tc.answer, 0.001).Correct {
			t.Errorf("%q accepted for %q", tc.answer, tc.key)
		}
	}
}
