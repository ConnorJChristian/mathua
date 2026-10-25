package calculus

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
)

// distinctQuestions mirrors all.distinctQuestionLowerBound: the smallest number of distinct
// question texts seen across a difficulty sweep.
func distinctQuestions(gen generator.Generator) int {
	best := -1
	for _, d := range []float64{0.3, 0.5, 0.6, 0.8, 0.9, 1.0} {
		seen := map[string]bool{}
		for i := 0; i < 300; i++ {
			seed := int64(i)*2654435761 + int64(d*1000)
			rand.Seed(seed)
			p := gen.Generate(generator.GeneratorContext{Difficulty: d, Seed: seed})
			seen[p.Question] = true
		}
		if n := len(seen); best < 0 || n < best {
			best = n
		}
	}
	return best
}

// TestWidenedPoolsHaveVariety guards the concepts whose pools were widened so a learner does
// not see the same handful of items over and over.
func TestWidenedPoolsHaveVariety(t *testing.T) {
	reg := generator.NewRegistry()
	Register(reg)
	for _, id := range []string{
		"calc.deriv.concept", "calc.multi.taylor2", "calc.deriv.diff_continuity",
		"calc.deriv.directional", "calc.deriv.clairaut", "calc.deriv.gradient",
		"calc.deriv.non_differentiability", "calc.series.taylor",
	} {
		gen, err := reg.Get(id)
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		n := distinctQuestions(gen)
		t.Logf("%s: %d distinct questions", id, n)
		if n < 12 {
			t.Errorf("%s: only %d distinct questions, want >= 12", id, n)
		}
	}
}

// The question reads "f(x,y) = a x^2 + b y^2 ... at (p,q) to estimate f(x,y)", so its integers
// in order are a, 2, b, 2, p, q, x, y.
var questionInts = regexp.MustCompile(`-?\d+`)

// TestTangentPlaneExplanationIsTrue recomputes the true value of f at the
// estimated point and checks the explanation: it must never call the estimate
// exact when it is not, and must state the true value it is compared with.
func TestTangentPlaneExplanationIsTrue(t *testing.T) {
	gen := &tangentPlaneGen{}
	for i := 0; i < 300; i++ {
		p := gen.Generate(generator.GeneratorContext{Difficulty: float64(i%10) / 9})
		m := questionInts.FindAllString(p.Question, -1)
		if len(m) != 8 {
			t.Fatalf("cannot parse question %q", p.Question)
		}
		v := make([]int, 8)
		for k := range v {
			v[k], _ = strconv.Atoi(m[k])
		}
		n := []int{v[0], v[2], v[4], v[5], v[6], v[7]}
		a, b, x, y := n[0], n[1], n[4], n[5]
		truth := a*x*x + b*y*y
		estimate, err := strconv.Atoi(p.Answer)
		if err != nil {
			t.Fatalf("answer %q is not an integer", p.Answer)
		}
		// The key is the linear approximation, not the true value.
		if want := a*n[2]*n[2] + b*n[3]*n[3] + 2*a*n[2] + 2*b*n[3]; estimate != want {
			t.Fatalf("%q: answer %d, want tangent-plane estimate %d", p.Question, estimate, want)
		}
		if estimate != truth && strings.Contains(strings.ToLower(p.Explanation), "exact") {
			t.Fatalf("estimate %d differs from f = %d but the explanation claims exactness: %q", estimate, truth, p.Explanation)
		}
		if !strings.Contains(p.Explanation, fmt.Sprintf("= %d,", truth)) {
			t.Fatalf("explanation does not state the true value %d: %q", truth, p.Explanation)
		}
	}
}
