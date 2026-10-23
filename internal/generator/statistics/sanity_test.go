package statistics

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
)

// TestGeneratorsProduceCleanProblems draws from every registered statistics
// generator and rejects what a format verb leaves behind when it is given the
// wrong argument: "%!d(float64=0.5)" in a stem, "NaN" or "+Inf" from a divisor
// that reached zero, an empty answer, an empty question, and an unbalanced
// LaTeX delimiter pair. A learner cannot answer a question that prints one of
// those, and no per-generator test had swept the package this way.
func TestGeneratorsProduceCleanProblems(t *testing.T) {
	reg := generator.NewRegistry()
	Register(reg)

	ids := reg.Concepts()
	if len(ids) < 20 {
		t.Fatalf("swept only %d statistics generators, expected the whole package", len(ids))
	}

	const draws = 150
	problems := 0
	for _, id := range ids {
		for i := 0; i < draws; i++ {
			ctx := generator.GeneratorContext{
				Difficulty: rand.Float64(),
				Seed:       rand.Int63(),
			}
			p, err := reg.GenerateContext(id, ctx)
			if err != nil {
				t.Errorf("%s: generate: %v", id, err)
				problems++
				break
			}
			if reason := unclean(p.Question, p.Answer, p.Explanation); reason != "" {
				t.Errorf("%s: %s\n  question: %q\n  answer: %q", id, reason, p.Question, p.Answer)
				problems++
				break
			}
		}
		if problems >= 10 {
			t.Fatalf("stopping after %d unclean generators", problems)
		}
	}
	if problems > 0 {
		t.Fatalf("%d of %d statistics generators produced an unusable problem", problems, len(ids))
	}
}

// unclean returns a short reason when a problem cannot be shown to a learner,
// or "" when it is fine.
func unclean(question, answer, explanation string) string {
	fields := []struct {
		name  string
		value string
	}{
		{"question", question},
		{"answer", answer},
		{"explanation", explanation},
	}
	for _, f := range fields {
		if strings.TrimSpace(f.value) == "" {
			return f.name + " is empty"
		}
		for _, bad := range []string{"%!", "NaN", "+Inf", "-Inf"} {
			if strings.Contains(f.value, bad) {
				return f.name + " contains " + bad
			}
		}
		if strings.Count(f.value, `\(`) != strings.Count(f.value, `\)`) {
			return f.name + " has an unbalanced \\() pair"
		}
		if strings.Count(f.value, `\[`) != strings.Count(f.value, `\]`) {
			return f.name + " has an unbalanced \\[] pair"
		}
	}
	return ""
}
