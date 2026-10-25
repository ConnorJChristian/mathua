package calculus

import (
	"math/big"
	"regexp"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
)

// Q46: the f/g limit is the exact quotient, not integer division.
func TestLimitPropertiesQuotientIsExact(t *testing.T) {
	reg := generator.NewRegistry()
	Register(reg)
	re := regexp.MustCompile(`lim f\(x\)\s*=\s*(-?\d+).*?lim g\(x\)\s*=\s*(-?\d+)`)
	seen := 0
	for i := 0; i < 2000 && seen < 100; i++ {
		p, _ := reg.GenerateContext("calc.limit.properties", generator.GeneratorContext{Difficulty: float64(i%3)*0.3 + 0.3, Seed: int64(i + 1)})
		m := re.FindStringSubmatch(p.Question)
		if m == nil || !regexp.MustCompile(`f\(x\)/g\(x\)`).MatchString(p.Question) {
			continue
		}
		seen++
		f, _ := new(big.Rat).SetString(m[1])
		g, _ := new(big.Rat).SetString(m[2])
		want := new(big.Rat).Quo(f, g)
		got, ok := new(big.Rat).SetString(p.Answer)
		if !ok || got.Cmp(want) != 0 {
			t.Fatalf("%s keyed %q, want %s", p.Question, p.Answer, want.RatString())
		}
	}
	if seen == 0 {
		p, _ := reg.GenerateContext("calc.limit.properties", generator.GeneratorContext{Difficulty: 0.5, Seed: 7})
		t.Log(p.Question)
		t.Fatal("no quotient items sampled")
	}
}
