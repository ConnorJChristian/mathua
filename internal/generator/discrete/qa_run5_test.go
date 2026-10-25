package discrete

// Regression tests for the 2026-10-25 QA run (run 5).

import (
	"math/big"
	"regexp"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
)

var run5Option = regexp.MustCompile(`\n\n([A-E])\) ([^\n]*)`)

// "If P(n): ..., what is P(1)?" keyed 1 (P(1) is a statement), and "(one
// word)" keyed "base case" (fails on the old text). Every item is now a
// multiple-choice question whose key is exactly one option.
func TestRun5InductionItemsAreWellPosed(t *testing.T) {
	reg := generator.NewRegistry()
	Register(reg)
	for i := 0; i < 1500; i++ {
		p, err := reg.GenerateContext("discrete.proof.induction", generator.GeneratorContext{Difficulty: []float64{0.3, 0.6, 0.9}[i%3], Seed: int64(300 + i)})
		if err != nil {
			t.Fatal(err)
		}
		if regexp.MustCompile(`what is P\(\d+\)\?`).MatchString(p.Question) {
			t.Fatalf("asks for the value of a statement: %q", p.Question)
		}
		if strings.Contains(p.Question, "(one word)") && strings.Contains(strings.TrimSpace(p.Answer), " ") {
			t.Fatalf("one-word item keyed %q", p.Answer)
		}
		hits := 0
		opts := run5Option.FindAllStringSubmatch(p.Question, -1)
		for _, m := range opts {
			if m[2] == p.Answer {
				hits++
			}
		}
		if len(opts) != 4 || hits != 1 {
			t.Fatalf("%d options, key %q found %d times: %q", len(opts), p.Answer, hits, p.Question)
		}
	}
}

// The common value of both sides of P(m) is computed by summing terms; check
// it against the closed forms written independently here.
func TestRun5InductionValuesMatchClosedForms(t *testing.T) {
	closed := []func(m int64) *big.Rat{
		func(m int64) *big.Rat { return big.NewRat(m*(m+1)/2, 1) },
		func(m int64) *big.Rat { return big.NewRat(m*m, 1) },
		func(m int64) *big.Rat { return big.NewRat(m*(m+1)*(2*m+1)/6, 1) },
		func(m int64) *big.Rat { return big.NewRat(m*m*(m+1)*(m+1)/4, 1) },
		func(m int64) *big.Rat { return big.NewRat(pow64(2, m)-1, 1) },
		func(m int64) *big.Rat { return big.NewRat((pow64(3, m)-1)/2, 1) },
		func(m int64) *big.Rat { return big.NewRat(m*(m+1)*(m+2)/3, 1) },
		func(m int64) *big.Rat { return big.NewRat(m, m+1) },
	}
	if len(closed) != len(inductionSums) {
		t.Fatalf("%d formulas, %d closed forms", len(inductionSums), len(closed))
	}
	for i, f := range inductionSums {
		for m := int64(1); m <= 10; m++ {
			if got, want := inductionValue(f, m), closed[i](m); got.Cmp(want) != 0 {
				t.Fatalf("formula %d at m=%d: sum %s, closed form %s", i, m, got, want)
			}
		}
	}
}

// The contradiction item about log2 3 keyed its longest option by a wide
// margin (fails on the old text).
func TestRun5ContradictionKeyIsNotTheLongestByFar(t *testing.T) {
	reg := generator.NewRegistry()
	Register(reg)
	for i := 0; i < 3000; i++ {
		p, _ := reg.GenerateContext("discrete.proof.contradiction", generator.GeneratorContext{Difficulty: 0.9, Seed: int64(i + 1)})
		opts := run5Option.FindAllStringSubmatch(p.Question, -1)
		if len(opts) < 2 {
			continue
		}
		key, longest := 0, 0
		for _, m := range opts {
			n := len([]rune(m[2]))
			if m[2] == p.Answer {
				key = n
			} else if n > longest {
				longest = n
			}
		}
		if float64(key) > 1.4*float64(longest) {
			t.Fatalf("key %d characters against at most %d for the others: %q", key, longest, p.Question)
		}
	}
}
