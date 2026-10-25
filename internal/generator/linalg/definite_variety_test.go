package linalg

import (
	"math"
	"regexp"
	"strconv"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
)

var definiteMatrix = regexp.MustCompile(`\\begin\{pmatrix\} (-?\d+) & (-?\d+) \\\\ (-?\d+) & (-?\d+) \\end\{pmatrix\}`)

// definiteByEigenvalues classifies [[a, b], [b, c]] from its eigenvalues.
func definiteByEigenvalues(a, b, c float64) string {
	m, r := (a+c)/2, math.Hypot((a-c)/2, b)
	lo, hi := m-r, m+r
	const eps = 1e-9
	switch {
	case lo > eps:
		return "positive definite"
	case hi < -eps:
		return "negative definite"
	case lo < -eps && hi > eps:
		return "indefinite"
	case hi > eps:
		return "positive semidefinite but not definite"
	}
	return "negative semidefinite but not definite"
}

func TestDefiniteVariantsClassifyCorrectly(t *testing.T) {
	gen := &definiteGen{}
	seen := map[string]bool{}
	verdict := map[string]string{
		"positive definite": "a local minimum", "negative definite": "a local maximum", "indefinite": "a saddle point",
		"positive semidefinite but not definite": "not determined by the second-order test",
		"negative semidefinite but not definite": "not determined by the second-order test",
	}
	checked := 0
	for i := 0; i < 3000; i++ {
		p := gen.Generate(generator.GeneratorContext{Difficulty: []float64{0.3, 0.6, 0.9}[i%3]})
		seen[p.Question] = true
		m := definiteMatrix.FindStringSubmatch(p.Question)
		if m == nil {
			continue
		}
		var v [4]float64
		for k := range v {
			n, _ := strconv.Atoi(m[k+1])
			v[k] = float64(n)
		}
		if v[1] != v[2] {
			t.Fatalf("%q: matrix is not symmetric", p.Question)
		}
		want := definiteByEigenvalues(v[0], v[1], v[3])
		if regexp.MustCompile(`Hessian`).MatchString(p.Question) {
			want = verdict[want]
		}
		if p.Answer != want {
			t.Errorf("%q: answer %q, eigenvalues say %q", p.Question, p.Answer, want)
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
