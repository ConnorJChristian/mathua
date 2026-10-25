package linalg

// Regression tests for the 2026-10-25 QA run (run 5).

import (
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
)

func run5Samples(t *testing.T, id string, n int) []generator.Problem {
	t.Helper()
	reg := generator.NewRegistry()
	Register(reg)
	var out []generator.Problem
	for i := 0; i < n; i++ {
		p, err := reg.GenerateContext(id, generator.GeneratorContext{Difficulty: []float64{0.3, 0.6, 0.9}[i%3], Seed: int64(7000 + i)})
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		out = append(out, p)
	}
	return out
}

var run5Pair = regexp.MustCompile(`v=\((-?\d+),(-?\d+)\)\$ and \$w=\((-?\d+),(-?\d+)\)`)

// "the vectors yes span R²" (fails on the old text), and 5 of 6 items keyed
// "no": the key must follow the determinant and be roughly balanced.
func TestRun5SpanExplanationAndBalance(t *testing.T) {
	yes := 0
	ps := run5Samples(t, "linalg.vec.span", 1200)
	for _, p := range ps {
		if strings.Contains(p.Explanation, "yes span") || strings.Contains(p.Explanation, "no span") {
			t.Fatalf("template word in explanation: %q", p.Explanation)
		}
		m := run5Pair.FindStringSubmatch(p.Question)
		if m == nil {
			t.Fatalf("cannot read %q", p.Question)
		}
		v := make([]int, 4)
		for i := range v {
			v[i], _ = strconv.Atoi(m[i+1])
		}
		want := "no"
		if v[0]*v[3]-v[1]*v[2] != 0 {
			want = "yes"
			yes++
		}
		if p.Answer != want {
			t.Fatalf("%q keyed %s, determinant says %s", p.Question, p.Answer, want)
		}
	}
	if yes < len(ps)*35/100 || yes > len(ps)*65/100 {
		t.Fatalf("%d of %d items keyed yes", yes, len(ps))
	}
}

// "(1/1)×", "(1/-1)×", "- -2*(...)", "--2*-2" and "-4+-6" (fails on the old text).
func TestRun5NoSignOrUnitArtefacts(t *testing.T) {
	checks := []struct {
		id  string
		bad *regexp.Regexp
	}{
		{"linalg.matrix.identity", regexp.MustCompile(`\(1/-?1\)`)},
		{"linalg.det.3x3", regexp.MustCompile(`-\s*-|\*-|\+\s*-\d`)},
		{"linalg.vector.dot", regexp.MustCompile(`\+\s*-\d|\*-`)},
	}
	for _, c := range checks {
		for _, p := range run5Samples(t, c.id, 1500) {
			if c.bad.MatchString(p.Explanation) {
				t.Fatalf("%s: artefact in %q", c.id, p.Explanation)
			}
		}
	}
}

var run5Mat = regexp.MustCompile(`\\begin\{pmatrix\} (-?\d+) & (-?\d+) \\\\ (-?\d+) & (-?\d+) \\end\{pmatrix\}`)

// The computed spectral items: the keyed eigenvalues have the matrix's trace
// and determinant, and the keyed eigenvector is one.
func TestRun5SpectralVariantKeys(t *testing.T) {
	for i := 0; i < 2000; i++ {
		rand.Seed(int64(i))
		p := spectralVariant()
		m := run5Mat.FindStringSubmatch(p.Question)
		if m == nil {
			t.Fatalf("cannot read %q", p.Question)
		}
		a, _ := strconv.Atoi(m[1])
		b, _ := strconv.Atoi(m[2])
		d, _ := strconv.Atoi(m[4])
		switch {
		case strings.Contains(p.Question, "has which eigenvalues"):
			var x, y int
			parts := strings.Split(p.Answer, " and ")
			x, _ = strconv.Atoi(parts[0])
			y, _ = strconv.Atoi(parts[1])
			if x+y != a+d || x*y != a*d-b*b {
				t.Fatalf("eigenvalues %q do not match %v", p.Answer, m[1:])
			}
		case strings.Contains(p.Question, "Which vector"):
			nums := regexp.MustCompile(`-?\d+`).FindAllString(p.Answer, -1)
			x, _ := strconv.Atoi(nums[0])
			y, _ := strconv.Atoi(nums[1])
			ax, ay := a*x+b*y, b*x+d*y
			if ax*y != ay*x {
				t.Fatalf("%q is not an eigenvector of %v", p.Answer, m[1:])
			}
		default:
			if p.Answer != strconv.Itoa(a*d-b*b) {
				t.Fatalf("product of eigenvalues keyed %s, det is %d", p.Answer, a*d-b*b)
			}
		}
	}
}
