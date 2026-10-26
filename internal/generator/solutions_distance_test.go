package generator_test

import (
	"regexp"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/solutions"
)

// A distance is never negative. The absolute-value schema said "-6 is -6
// units from zero" for every negative operand; the low difficulties draw
// only negatives, so every one of those explanations was wrong.
func TestAbsoluteValueExplanationGivesAPositiveDistance(t *testing.T) {
	l, err := solutions.Load(solutionsDir)
	if err != nil {
		t.Fatal(err)
	}
	reg := generator.NewRegistry()
	reg.SetSolutions(l)
	registerAllDomains(reg)
	negDistance := regexp.MustCompile(`is -\d+ units`)
	for i := 0; i < 60; i++ {
		p, err := reg.GenerateContext("arith.neg.abs_value", generator.GeneratorContext{Difficulty: 0.3, Seed: int64(100 + i)})
		if err != nil {
			t.Fatal(err)
		}
		if m := negDistance.FindString(p.Explanation); m != "" {
			t.Fatalf("negative distance %q in %q", m, p.Explanation)
		}
	}
}
