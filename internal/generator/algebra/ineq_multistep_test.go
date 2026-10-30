package algebra

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
)

// Q55: "Solve: 6x + 5 > 39" was keyed 6 with the explanation "So the
// solution is x = 6". The key is now the solution set, the prompt asks for
// an inequality, and the generator grades the forms the prompt allows.
func TestMultiStepInequalityKeyIsTheSolutionSet(t *testing.T) {
	g := &multiStepIneqGen{}
	for i := 0; i < 300; i++ {
		rand.Seed(int64(i + 1))
		p := g.Generate(generator.GeneratorContext{Difficulty: 0.5})
		a, _ := strconv.Atoi(p.Facts["a"])
		rhs, _ := strconv.Atoi(p.Facts["rhs"])
		v := float64(rhs) / float64(a)
		if !strings.HasPrefix(p.Answer, "x > ") {
			t.Fatalf("%s: key %q is not an inequality", p.Question, p.Answer)
		}
		if !strings.Contains(p.Question, "inequality") {
			t.Fatalf("prompt does not ask for an inequality: %q", p.Question)
		}
		if strings.Contains(p.Explanation, "x = ") {
			t.Fatalf("explanation names one number: %q", p.Explanation)
		}
		for _, ans := range []string{p.Answer, fmt.Sprintf("x>%.2f", v), fmt.Sprintf("%d/%d < x", rhs, a)} {
			if !g.Grade(p.Answer, ans).Correct {
				t.Fatalf("%s: %q refused for key %q", p.Question, ans, p.Answer)
			}
		}
		lo := int(v) + 1 // the old key: the smallest whole number in the set
		for _, ans := range []string{strconv.Itoa(lo), fmt.Sprintf("x >= %d/%d", rhs, a), fmt.Sprintf("x > %.2f", v+0.02)} {
			if g.Grade(p.Answer, ans).Correct {
				t.Fatalf("%s: %q accepted for key %q", p.Question, ans, p.Answer)
			}
		}
	}
}
