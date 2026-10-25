package linalg

// The linear-algebra concepts the domain was missing: independence, the rank-nullity theorem, the
// inverse of a two-by-two matrix, the two eigenvalue invariants, projection onto a vector, the norm,
// and the determinant of a rotation-scaling.
//
// The domain had vectors, matrices, determinants, eigenvalues, systems and decompositions, but
// nothing on when a set of vectors is independent, nothing on dimension or nullity, no inverse, and
// no orthogonality at all -- which is most of what a first course is about.
//
// Every answer is an integer by construction: the vectors are built from a known rank, the matrices
// have determinant one or a diagonal form, and the projections use a coefficient that divides.
import (
	"fmt"
	"math/rand"

	"github.com/chuma-beep/mathua/internal/generator"
)

func registerFoundations(reg *generator.Registry) {
	reg.Register("linalg.independence.count", &independenceCountGen{})
	reg.Register("linalg.rank.nullity", &rankNullityGen{})
	reg.Register("linalg.matrix.inverse_2x2", &inverse2x2Gen{})
	reg.Register("linalg.eigen.trace", &eigenTraceGen{})
	reg.Register("linalg.eigen.det_product", &eigenDetProductGen{})
	reg.Register("linalg.orth.projection_coeff", &projectionCoeffGen{})
	reg.Register("linalg.vec.norm_squared", &normSquaredGen{})
	reg.Register("linalg.transform.det_scale", &detScaleGen{})
}

func smallInt(lo, hi int) int {
	if hi < lo {
		hi = lo
	}
	return lo + rand.Intn(hi-lo+1)
}

// independenceCountGen builds three vectors with a known rank, so the count of independent ones is
// exact.
type independenceCountGen struct{}

func (g *independenceCountGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	rank := 1 + rand.Intn(3)
	a := smallInt(1, 3+int(ctx.Difficulty*3))
	b := smallInt(1, 3+int(ctx.Difficulty*3))
	c := smallInt(1, 3+int(ctx.Difficulty*3))
	var vs [3][3]int
	switch rank {
	case 1:
		vs = [3][3]int{{a, b, c}, {2 * a, 2 * b, 2 * c}, {3 * a, 3 * b, 3 * c}}
	case 2:
		vs = [3][3]int{{a, b, c}, {1, 0, 0}, {a + 1, b, c}}
	default:
		vs = [3][3]int{{a, b, c}, {1, 0, 0}, {0, 1, 0}}
	}
	return generator.Problem{
		Question: fmt.Sprintf("How many of the vectors \\((%d,%d,%d)\\), \\((%d,%d,%d)\\) and \\((%d,%d,%d)\\) are linearly independent?",
			vs[0][0], vs[0][1], vs[0][2], vs[1][0], vs[1][1], vs[1][2], vs[2][0], vs[2][1], vs[2][2]),
		Answer: fmt.Sprintf("%d", rank),
		Explanation: fmt.Sprintf("The number of independent vectors is the dimension of what they span, which is the rank of the matrix they form as rows. Here that rank is %d, so %d of them are independent and the rest are combinations of those.",
			rank, rank),
		Facts: map[string]string{"independent": fmt.Sprintf("%d", rank)},
	}
}

// rankNullityGen is the rank-nullity theorem read as an arithmetic fact.
type rankNullityGen struct{}

func (g *rankNullityGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	cols := 3 + rand.Intn(3+int(ctx.Difficulty*3))
	rows := 2 + rand.Intn(3+int(ctx.Difficulty*3))
	rank := 1 + rand.Intn(cols)
	return generator.Problem{
		Question: fmt.Sprintf("A matrix has %d rows and %d columns, and its rank is %d. What is the dimension of its null space?", rows, cols, rank),
		Answer:   fmt.Sprintf("%d", cols-rank),
		Explanation: fmt.Sprintf("The rank-nullity theorem says rank plus nullity equals the number of columns: %d + nullity = %d, so the null space has dimension %d. The number of rows does not enter, which is the part worth noticing.",
			rank, cols, cols-rank),
		Facts: map[string]string{"nullity": fmt.Sprintf("%d", cols-rank), "rank": fmt.Sprintf("%d", rank)},
	}
}

// inverse2x2Gen uses matrices of determinant one, whose inverses have integer entries.
type inverse2x2Gen struct{}

func (g *inverse2x2Gen) Generate(ctx generator.GeneratorContext) generator.Problem {
	// [[a, 1], [a^2 - 1, a]] has determinant a^2 - (a^2 - 1) = 1 for every a, so the inverse's
	// entries are the cofactors themselves and stay whole.
	a := 2 + rand.Intn(3+int(ctx.Difficulty*4))
	b, c, d := 1, a*a-1, a
	det := a*d - b*c
	return generator.Problem{
		Question: fmt.Sprintf("The matrix \\(A = \\begin{pmatrix} %d & %d \\\\ %d & %d \\end{pmatrix}\\) has determinant %d, so it is invertible. What is the entry in the first row and first column of \\(A^{-1}\\)?", a, b, c, d, det),
		Answer:   fmt.Sprintf("%d", d),
		Explanation: fmt.Sprintf("For a two-by-two matrix the inverse is the transpose of the cofactor matrix divided by the determinant: the swap of the diagonal entries and the sign change of the other two, all over %d. The first-row first-column entry is therefore %d.",
			det, d),
		Facts: map[string]string{"entry": fmt.Sprintf("%d", d), "determinant": fmt.Sprintf("%d", det)},
	}
}

// eigenTraceGen uses the trace as the sum of the eigenvalues.
type eigenTraceGen struct{}

func (g *eigenTraceGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	l1 := smallInt(-5, 5)
	l2 := smallInt(-5, 5)
	if l2 == l1 {
		l2 = l1 + 1
	}
	trace := l1 + l2
	return generator.Problem{
		Question: fmt.Sprintf("A two-by-two matrix has trace %d, and one of its eigenvalues is %d. What is the other eigenvalue?", trace, l1),
		Answer:   fmt.Sprintf("%d", l2),
		Explanation: fmt.Sprintf("The trace is the sum of the eigenvalues, and it is also the sum of the diagonal entries. So the other eigenvalue is %d - %d = %d. The trace is one of the two invariants that determine a two-by-two matrix's eigenvalues, the determinant being the other.",
			trace, l1, l2),
		Facts: map[string]string{"eigenvalue": fmt.Sprintf("%d", l2), "trace": fmt.Sprintf("%d", trace)},
	}
}

// eigenDetProductGen uses the determinant as the product of the eigenvalues.
type eigenDetProductGen struct{}

func (g *eigenDetProductGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	l1 := 1 + rand.Intn(4+int(ctx.Difficulty*4))
	l2 := smallInt(-4, 4)
	if l2 == 0 {
		l2 = 1
	}
	return generator.Problem{
		Question: fmt.Sprintf("A two-by-two matrix has determinant %d, and one of its eigenvalues is %d. What is the other eigenvalue?", l1*l2, l1),
		Answer:   fmt.Sprintf("%d", l2),
		Explanation: fmt.Sprintf("The determinant is the product of the eigenvalues, and it is also the signed area scaling the matrix applies. So the other eigenvalue is %d / %d = %d.",
			l1*l2, l1, l2),
		Facts: map[string]string{"eigenvalue": fmt.Sprintf("%d", l2), "determinant": fmt.Sprintf("%d", l1*l2)},
	}
}

// projectionCoeffGen asks for the scalar whose multiple of b is the projection of a onto b.
type projectionCoeffGen struct{}

func (g *projectionCoeffGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	b1 := smallInt(1, 3+int(ctx.Difficulty*2))
	b2 := smallInt(-3, 3)
	if b2 == 0 && b1 == 0 {
		b1 = 1
	}
	c := smallInt(-3, 3+int(ctx.Difficulty*3))
	bb := b1*b1 + b2*b2
	// a = c*b plus a component orthogonal to b, chosen so the coefficient is exactly c
	a1, a2 := c*b1-b2, c*b2+b1
	return generator.Problem{
		Question: fmt.Sprintf("Find the scalar \\(c\\) such that \\(c(%d,%d)\\) is the projection of \\((%d,%d)\\) onto \\((%d,%d)\\).", b1, b2, a1, a2, b1, b2),
		Answer:   fmt.Sprintf("%d", c),
		Explanation: fmt.Sprintf("The projection coefficient is the dot product over the squared length of the direction: (%d)(%d) + (%d)(%d) = %d, divided by %d, which is %d. The scalar is the length of the projection in units of b, not the projection itself.",
			a1, b1, a2, b2, a1*b1+a2*b2, bb, c),
		Facts: map[string]string{"coefficient": fmt.Sprintf("%d", c), "squared_length": fmt.Sprintf("%d", bb)},
	}
}

// normSquaredGen is the dot product of a vector with itself, which avoids a square root.
type normSquaredGen struct{}

func (g *normSquaredGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	n := 2 + rand.Intn(2) // 2 or 3 dimensions
	v := make([]int, n)
	for i := range v {
		v[i] = smallInt(-6, 6)
		if v[i] == 0 {
			v[i] = 1
		}
	}
	sum := 0
	for _, x := range v {
		sum += x * x
	}
	if n == 2 {
		return generator.Problem{
			Question: fmt.Sprintf("Find the squared length \\(\\|v\\|^2\\) of the vector \\(v = (%d, %d)\\).", v[0], v[1]),
			Answer:   fmt.Sprintf("%d", sum),
			Explanation: fmt.Sprintf("The squared length is the vector's dot product with itself: %s + %s = %d. Squaring avoids the square root, and any comparison of lengths can be done with the squares because squaring is increasing for positive numbers.",
				generator.Sq(v[0]), generator.Sq(v[1]), sum),
			Facts: map[string]string{"squared_length": fmt.Sprintf("%d", sum)},
		}
	}
	return generator.Problem{
		Question: fmt.Sprintf("Find the squared length \\(\\|v\\|^2\\) of the vector \\(v = (%d, %d, %d)\\).", v[0], v[1], v[2]),
		Answer:   fmt.Sprintf("%d", sum),
		Explanation: fmt.Sprintf("The squared length is the sum of the squared components: %d + %d + %d = %d. In any number of dimensions the same sum applies, which is what makes the norm a single concept rather than one per dimension.",
			v[0]*v[0], v[1]*v[1], v[2]*v[2], sum),
		Facts: map[string]string{"squared_length": fmt.Sprintf("%d", sum)},
	}
}

// detScaleGen uses a rotation-scaling matrix, whose determinant is the squared scale plus the
// squared rotation, so it stays whole.
type detScaleGen struct{}

func (g *detScaleGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	a := smallInt(1, 4+int(ctx.Difficulty*4))
	b := smallInt(-4, 4)
	return generator.Problem{
		Question: fmt.Sprintf("Find the determinant of \\(\\begin{pmatrix} %d & %d \\\\ %d & %d \\end{pmatrix}\\).", a, -b, b, a),
		Answer:   fmt.Sprintf("%d", a*a+b*b),
		Explanation: fmt.Sprintf("The determinant is %d*%d - (-%d)*%d = %d. A matrix of this shape scales every vector by the same factor and rotates it, so its determinant is that factor squared, and a sum of squares is never negative.",
			a, a, b, b, a*a+b*b),
		Facts: map[string]string{"determinant": fmt.Sprintf("%d", a*a+b*b), "scale_squared": fmt.Sprintf("%d", a*a+b*b)},
	}
}
