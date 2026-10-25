package odes

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
)

var phaseMatrix = regexp.MustCompile(`\\begin\{pmatrix\} (-?\d+) & (-?\d+) \\\\ (-?\d+) & (-?\d+) \\end\{pmatrix\}`)

// classifyByEigenvalues names the origin of x' = Ax from the eigenvalues of A,
// computed directly rather than from the trace-determinant rules the generator uses.
func classifyByEigenvalues(a, b, c, d float64) string {
	tr, det := a+d, a*d-b*c
	disc := tr*tr - 4*det
	if disc >= 0 {
		l1, l2 := (tr-math.Sqrt(disc))/2, (tr+math.Sqrt(disc))/2
		switch {
		case l1 < 0 && l2 > 0:
			return "saddle"
		case l2 < 0:
			return "stable node"
		case l1 > 0:
			return "unstable node"
		}
		return "degenerate"
	}
	switch re := tr / 2; {
	case re == 0:
		return "center"
	case re < 0:
		return "stable spiral"
	}
	return "unstable spiral"
}

func TestLinearPhaseVariantsClassifyCorrectly(t *testing.T) {
	gen := &linearPhaseGen{}
	seen := map[string]bool{}
	checked := 0
	for i := 0; i < 3000; i++ {
		p := gen.Generate(generator.GeneratorContext{Difficulty: []float64{0.3, 0.6, 0.9}[i%3]})
		seen[p.Question] = true
		if strings.TrimSpace(p.Explanation) == "" {
			t.Fatalf("empty explanation: %q", p.Question)
		}
		m := phaseMatrix.FindStringSubmatch(p.Question)
		if m == nil {
			continue
		}
		var v [4]float64
		for k := range v {
			n, _ := strconv.Atoi(m[k+1])
			v[k] = float64(n)
		}
		if want := classifyByEigenvalues(v[0], v[1], v[2], v[3]); p.Answer != want {
			t.Errorf("%q: answer %q, eigenvalues say %q", p.Question, p.Answer, want)
		}
		if !strings.Contains(p.Question+"\n", ") "+p.Answer+"\n") {
			t.Errorf("%q: correct answer %q is not among the options", p.Question, p.Answer)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no matrix variants generated")
	}
	if len(seen) < 12 {
		t.Errorf("only %d distinct questions, want >= 12", len(seen))
	}
}
