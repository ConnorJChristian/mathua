package grader

import (
	"os/exec"
	"testing"
)

func sympyAvailable(t *testing.T) bool {
	t.Helper()
	if err := exec.Command("python3", "-c", "import sympy").Run(); err != nil {
		t.Log("SymPy not available, skipping test")
		return false
	}
	if findSymPyService() == "" {
		t.Log("sympy_service.py not found, skipping test")
		return false
	}
	return true
}

func TestSympyGrade_EquivalentExpressions(t *testing.T) {
	if !sympyAvailable(t) {
		t.Skip("SymPy not available")
	}
	r := NewRouter()
	tests := []struct {
		expected, answer string
	}{
		{"x^2+2x+1", "(x+1)^2"},
		{"6x^2", "6*x^2"},
		{"(x+3)(x+4)", "x^2+7x+12"},
		{"e^x", "exp(x)"},
		{"2e^(2x)", "2*exp(2*x)"},
		{"cos(x)", "cos(x)"},
		{"3(2x+1)^2", "12x^2+12x+3"},
		{"y = (z - 2x)/3", "(z-2*x)/3"},
		{"xe^x-e^x+C", "e^x*(x-1)+C"},
		{"(x+5)(x-5)", "x^2-25"},
		{"-2", "-2"},
	}
	for _, tc := range tests {
		res := r.Grade(GradingExpression, tc.expected, tc.answer)
		if !res.Correct {
			t.Errorf("expected %q ≈ %q to be correct, got feedback: %q", tc.expected, tc.answer, res.Feedback)
		}
	}
}

func TestSympyGrade_NonequivalentExpressions(t *testing.T) {
	if !sympyAvailable(t) {
		t.Skip("SymPy not available")
	}
	r := NewRouter()
	tests := []struct {
		expected, answer string
	}{
		{"2x+3", "2x+4"},
		{"x^2", "x^3"},
		{"cos(x)", "sin(x)"},
	}
	for _, tc := range tests {
		res := r.Grade(GradingExpression, tc.expected, tc.answer)
		if res.Correct {
			t.Errorf("expected %q ≠ %q to be incorrect", tc.expected, tc.answer)
		}
	}
}

func TestSympyGrade_ParseError(t *testing.T) {
	if !sympyAvailable(t) {
		t.Skip("SymPy not available")
	}
	r := NewRouter()
	res := r.Grade(GradingExpression, "x^2", "invalid(((")
	if res.Correct {
		t.Error("expected parse error to be incorrect")
	}
	if res.Feedback == "" || res.Feedback == "Incorrect" {
		t.Errorf("expected specific parse error feedback, got %q", res.Feedback)
	}
}

func TestSympyGrade_ExpressionAndPolynomialBothRoute(t *testing.T) {
	if !sympyAvailable(t) {
		t.Skip("SymPy not available")
	}
	r := NewRouter()
	tests := []struct {
		gt               GradingType
		expected, answer string
	}{
		{GradingExpression, "x^2+2x+1", "(x+1)^2"},
		{GradingPolynomial, "6x^2", "6*x^2"},
	}
	for _, tc := range tests {
		res := r.Grade(tc.gt, tc.expected, tc.answer)
		if !res.Correct {
			t.Errorf("expected %s(%q ≈ %q) to be correct", tc.gt, tc.expected, tc.answer)
		}
	}
}

func TestSympyGrade_MultipleRequests(t *testing.T) {
	if !sympyAvailable(t) {
		t.Skip("SymPy not available")
	}
	r := NewRouter()
	for i := 0; i < 5; i++ {
		res := r.Grade(GradingExpression, "x^2+2x+1", "(x+1)^2")
		if !res.Correct {
			t.Errorf("request %d: expected correct, got %q", i, res.Feedback)
		}
	}
}

// A wrong answer whose values are tiny everywhere the check samples (twice
// a derivative with a fifth-power denominator) must not pass on an absolute
// tolerance (fork fix: the relative scale had a floor of 1).
func TestSympyGrade_TinyValuedWrongAnswerIsRefused(t *testing.T) {
	if !sympyAvailable(t) {
		t.Skip("SymPy not available")
	}
	r := NewRouter()
	key := "-(1464x^2)/((122x^3-49)^5)"
	if r.Grade(GradingExpression, key, "2*("+key+")").Correct {
		t.Error("twice the key graded correct")
	}
	if !r.Grade(GradingExpression, key, "-1464*x^2*(122*x^3-49)^(-5)").Correct {
		t.Error("an equivalent spelling of the key graded wrong")
	}
	if !r.Grade(GradingExpression, "(x+1)^2 - x^2 - 2x", "1").Correct {
		t.Error("a cancelling difference graded wrong")
	}
}

// An empty answer is never sent to SymPy, and never correct, whatever the
// grading type (fork fix: a late reply from a timed-out request used to be
// read as the next request's verdict).
func TestSympyGrade_EmptyAnswerIsNeverCorrect(t *testing.T) {
	r := NewRouter()
	for _, gt := range []GradingType{GradingSymbolic, GradingExpression, GradingPolynomial} {
		res := r.Grade(gt, "x^2", "")
		if res.Correct || res.Unavailable {
			t.Errorf("%s: empty answer gave correct=%v unavailable=%v", gt, res.Correct, res.Unavailable)
		}
	}
}

// A reply to an earlier request is skipped: the service is asked twice and
// the second verdict must be the second request's, even after a stale line.
func TestSympyGrade_RepliesMatchTheirRequests(t *testing.T) {
	if !sympyAvailable(t) {
		t.Skip("SymPy not available")
	}
	r := NewRouter()
	if !r.Grade(GradingSymbolic, "x^2+2x+1", "(x+1)^2").Correct {
		t.Fatal("warm-up failed")
	}
	// Push a stale reply onto the service's stdout: a request the grader
	// never waits for.
	sympyMu.Lock()
	if sympyStdin != nil {
		sympyStdin.Write([]byte(`{"id":"stale","expected":"1","answer":"1"}` + "\n"))
	}
	sympyMu.Unlock()
	if r.Grade(GradingSymbolic, "x^2", "x^3").Correct {
		t.Error("x^3 graded correct for x^2: a stale reply was read as the verdict")
	}
	if !r.Grade(GradingSymbolic, "2x", "x+x").Correct {
		t.Error("x+x refused for 2x after the stale reply")
	}
}

// Letters typed together before a subscript are a product when the key names
// each of them: the math box exports "a b p_{1}" as "abp_1" (fork fix). A
// name the key does not use is not invented by splitting.
func TestSympyGrade_JoinedNamesBeforeASubscript(t *testing.T) {
	if !sympyAvailable(t) {
		t.Skip("SymPy not available")
	}
	r := NewRouter()
	for _, c := range []struct {
		key, typed string
		want       bool
	}{
		{"a*b*p_1", "abp_1", true},
		{"(-a*b*p_1 + m)/(p_1*(b + 1))", "(-a b p_1+m)/(p_1(b+1))", true},
		{"a*b*p_1", "abp_2", false},
		{"x_1*y", "xy_1", false},
		{"a*b*p_1", "a*b", false},
	} {
		if got := r.Grade(GradingExpression, c.key, c.typed); got.Correct != c.want || got.Unavailable {
			t.Errorf("%q for %q: %+v, want correct=%v", c.typed, c.key, got, c.want)
		}
	}
}
