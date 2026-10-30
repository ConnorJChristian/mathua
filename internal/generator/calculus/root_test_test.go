package calculus

// Tests for the root-test concept after the 2026-10-19 content pass.
//
// The generator is a fixed table, so the test asserts the whole table rather
// than a random sample: every question and its answer must appear, the
// self-revealing question that used to give away its own limit must be gone,
// and the three limits must agree with an independent numerical evaluation at
// a large n. The answers are then put through the real symbolic grader, which
// is the path a learner's answer takes.

import (
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
)

type rootTestPair struct {
	question string
	answer   string
}

var rootTestPairs = []rootTestPair{
	{"Does the root test use the nth root of \\(|a_n|\\)? (yes/no)", "yes"},
	{"Does \\(\\sum (n/(n+1))^{n^2}\\) converge? (yes/no)", "yes"},
	{"Does \\(\\sum 1/2^n\\) converge? (yes/no)", "yes"},
	{"Is the root test useful when \\(a_n\\) involves nth powers? (yes/no)", "yes"},
	{"For \\(a_n = (2n/(3n+1))^n\\), what is the root-test limit \\(L = \\lim |a_n|^{1/n}\\)? (enter an exact fraction)", "2/3"},
	{"For \\(a_n = ((3n+1)/n)^n\\), what is the root-test limit \\(L = \\lim |a_n|^{1/n}\\)? (enter a number)", "3"},
	{"For \\(a_n = ((n+1)/n)^n\\), what is the root-test limit \\(L = \\lim |a_n|^{1/n}\\)? (enter a number)", "1"},
	{"A series has root-test limit \\(L = 1\\). Does the root test decide convergence? (yes/no)", "no"},
}

func TestRootTestTableIsComplete(t *testing.T) {
	// The table is package-level, so the count and uniqueness are asserted
	// directly. Sampling alone cannot tell the eight intended items from nine
	// entries with one question repeated.
	if len(rootTestTable) != len(rootTestPairs) {
		t.Fatalf("rootTestTable has %d entries, want %d", len(rootTestTable), len(rootTestPairs))
	}
	tableQuestions := map[string]bool{}
	for _, e := range rootTestTable {
		if strings.TrimSpace(e.q) == "" || strings.TrimSpace(e.a) == "" || strings.TrimSpace(e.e) == "" {
			t.Errorf("entry with an empty field: %+v", e)
		}
		if strings.Contains(e.q, "a_n") || strings.Contains(e.q, "^") || strings.Contains(e.q, "sum") {
			if !strings.Contains(e.q, "\\(") {
				t.Errorf("question contains math without delimiters: %q", e.q)
			}
		}
		if tableQuestions[e.q] {
			t.Errorf("duplicate question in the table: %q", e.q)
		}
		tableQuestions[e.q] = true
	}
	if len(tableQuestions) != len(rootTestTable) {
		t.Fatalf("%d unique questions for %d entries", len(tableQuestions), len(rootTestTable))
	}

	gen := &rootTestGen{}
	seen := map[string]string{}
	for i := 0; i < 500; i++ {
		p := gen.Generate(generator.GeneratorContext{Difficulty: 0.5})
		if strings.TrimSpace(p.Explanation) == "" {
			t.Fatalf("empty explanation for %q", p.Question)
		}
		seen[p.Question] = p.Answer
	}
	if len(seen) != len(rootTestPairs) {
		t.Fatalf("saw %d distinct questions, want %d", len(seen), len(rootTestPairs))
	}
	for _, want := range rootTestPairs {
		got, ok := seen[want.question]
		if !ok {
			t.Errorf("question missing: %q", want.question)
			continue
		}
		if got != want.answer {
			t.Errorf("%q answered %q, want %q", want.question, got, want.answer)
		}
	}
	for q := range seen {
		if strings.Contains(q, "L = 1/2. What is L?") {
			t.Errorf("the self-revealing question is still present: %q", q)
		}
	}
}

func parseFraction(s string) (float64, bool) {
	parts := strings.Split(strings.TrimSpace(s), "/")
	if len(parts) == 1 {
		v, err := strconv.ParseFloat(parts[0], 64)
		return v, err == nil
	}
	if len(parts) != 2 {
		return 0, false
	}
	num, err1 := strconv.ParseFloat(parts[0], 64)
	den, err2 := strconv.ParseFloat(parts[1], 64)
	if err1 != nil || err2 != nil || den == 0 {
		return 0, false
	}
	return num / den, true
}

func TestRootTestLimitsAreIndependentlyCorrect(t *testing.T) {
	// Each limit is the limit of |a_n|^(1/n); for these three geometric-power
	// families that is the base itself, evaluated here at n = 1e6 so the test
	// does not trust the table's arithmetic.
	const n = 1e6
	limits := []struct {
		question string
		value    float64
	}{
		{rootTestPairs[4].question, 2 * n / (3*n + 1)},
		{rootTestPairs[5].question, (3*n + 1) / n},
		{rootTestPairs[6].question, (n + 1) / n},
	}
	for _, c := range limits {
		answer := ""
		for _, p := range rootTestPairs {
			if p.question == c.question {
				answer = p.answer
			}
		}
		got, ok := parseFraction(answer)
		if !ok {
			t.Fatalf("answer %q is not a number or fraction", answer)
		}
		if math.Abs(got-c.value) > 1e-4 {
			t.Errorf("%q answered %s (%v), independent limit is %v", c.question, answer, got, c.value)
		}
	}
}

func TestRootTestAnswersGradeAsSymbolic(t *testing.T) {
	router := grader.NewRouter()
	defer router.Close()

	wrong := func(answer string) string {
		switch answer {
		case "yes":
			return "no"
		case "no":
			return "yes"
		case "2/3":
			return "3/2"
		default:
			return "2"
		}
	}

	for _, p := range rootTestPairs {
		res := router.Grade(grader.GradingSymbolic, p.answer, p.answer)
		if res.Unavailable {
			t.Skip("symbolic grader unavailable; table checks still ran")
		}
		if !res.Correct {
			t.Errorf("%q: answer %q does not grade as correct: %s", p.question, p.answer, res.Feedback)
		}
		if res := router.Grade(grader.GradingSymbolic, p.answer, wrong(p.answer)); res.Correct {
			t.Errorf("%q: wrong answer %q accepted", p.question, wrong(p.answer))
		}
	}

	// The grader strips whitespace before the exact-match fast path.
	if res := router.Grade(grader.GradingSymbolic, "2/3", " 2/3 "); !res.Correct {
		t.Errorf("whitespace-equivalent fraction rejected: %s", res.Feedback)
	}
}
