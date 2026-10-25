package linalg

import (
	"regexp"
	"strconv"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
)

func TestDiagonalization_AnyValidPairIsRight(t *testing.T) {
	ref := "P=[[2,2],[2,-2]] D=[[3,0],[0,-1]]" // A = [[1,2],[2,1]]
	for _, ok := range []string{
		"P=[[1,1],[1,-1]] D=[[3,0],[0,-1]]",
		"P=[[1,-1],[1,1]] D=[[3,0],[0,-1]]",
		"P=[[1,1],[-1,1]] D=[[-1,0],[0,3]]", // swapped order
		"P = [[2, 2], [2, -2]]  D = [[3, 0], [0, -1]]",
	} {
		if r := gradeDiagonalization(ref, ok); !r.Correct {
			t.Errorf("%q should be right: %s", ok, r.Feedback)
		}
	}
	for _, bad := range []string{
		"P=[[1,2],[2,1]] D=[[3,0],[0,-1]]",  // P = A, the old generator's "answer"
		"P=[[1,1],[1,-1]] D=[[-1,0],[0,3]]", // columns and eigenvalues mismatched
		"P=[[1,1],[1,1]] D=[[3,0],[0,-1]]",  // singular P
		"P=[[1,1],[1,-1]] D=[[3,1],[0,-1]]", // not diagonal
	} {
		if r := gradeDiagonalization(ref, bad); r.Correct {
			t.Errorf("%q should be wrong", bad)
		}
	}
	g := &diagonalizationGen{}
	for i := 0; i < 30; i++ {
		p := g.Generate(generator.GeneratorContext{})
		if r := gradeDiagonalization(p.Answer, p.Answer); !r.Correct {
			t.Fatalf("generated answer %q does not grade as right: %s", p.Answer, r.Feedback)
		}
	}
}

// Q40: every served inverse, multiplied by its matrix, gives the identity.
func TestMatrixInverseKeysAreInverses(t *testing.T) {
	g := &matrixIdentityGen{}
	re := regexp.MustCompile(`-?\d+`)
	for i := 0; i < 400; i++ {
		p := g.Generate(generator.GeneratorContext{Difficulty: 0.5})
		q := re.FindAllString(p.Question, -1)
		k := re.FindAllString(p.Answer, -1)
		if len(q) < 4 || len(k) != 4 {
			t.Fatalf("parse %q / %q", p.Question, p.Answer)
		}
		n := func(s string) int { v, _ := strconv.Atoi(s); return v }
		a, b, c, d := n(q[len(q)-4]), n(q[len(q)-3]), n(q[len(q)-2]), n(q[len(q)-1])
		w, x, y, z := n(k[0]), n(k[1]), n(k[2]), n(k[3])
		if a*w+b*y != 1 || a*x+b*z != 0 || c*w+d*y != 0 || c*x+d*z != 1 {
			t.Fatalf("%s keyed %s, which is not its inverse", p.Question, p.Answer)
		}
	}
}
