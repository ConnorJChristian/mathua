package algebra

import (
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
)

// The concept draws one of three questions: the definition, an identity or
// the logarithmic function (upstream eabb8e1a picks them by knowledge point;
// the fork has none, so it draws). Every kind must come up, and the two
// multiple-choice kinds must carry their answer exactly once.
func TestLogConceptDrawsAllThreeQuestions(t *testing.T) {
	g := &logConceptGen{}
	definition, choice := 0, 0
	for i := 0; i < 200; i++ {
		p := g.Generate(generator.GeneratorContext{Difficulty: 0.5})
		if strings.Contains(p.Question, "Write as a logarithm") {
			definition++
			continue
		}
		choice++
		assertOneCorrectOption(t, p, optionsOf(p.Question))
	}
	if definition == 0 || choice == 0 {
		t.Fatalf("200 draws gave %d definition and %d identity/function questions", definition, choice)
	}
}

func assertOneCorrectOption(t *testing.T, p generator.Problem, options []string) {
	t.Helper()
	if len(options) != 4 {
		t.Fatalf("%q has %d options, want 4", p.Question, len(options))
	}
	seen := 0
	for _, option := range options {
		if strings.TrimSpace(option) == strings.TrimSpace(p.Answer) {
			seen++
		}
	}
	if seen != 1 {
		t.Fatalf("%q has answer %q in %d options: %v", p.Question, p.Answer, seen, options)
	}
}
