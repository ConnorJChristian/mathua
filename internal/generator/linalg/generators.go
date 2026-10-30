package linalg

import (
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
)

var matrixRe = regexp.MustCompile(`\[\[([^\]]+)\],\[([^\]]+)\]\]`)

func Register(reg *generator.Registry) {
	reg.Register("linalg.vector.concept", &vectorConceptGen{})
	reg.Register("linalg.vector.add", &vectorAddGen{})
	reg.Register("linalg.vector.dot", &vectorDotGen{})
	reg.Register("linalg.matrix.concept", &matrixConceptGen{})
	reg.Register("linalg.matrix.add", &matrixAddGen{})
	reg.Register("linalg.matrix.mult", &matrixMultGen{})
	reg.Register("linalg.matrix.identity", &matrixIdentityGen{})
	reg.Register("linalg.det.2x2", &det2x2Gen{})
	reg.Register("linalg.det.3x3", &det3x3Gen{})
	reg.Register("linalg.systems.matrix", &systemsMatrixGen{})
	reg.Register("linalg.sys.cramer", &cramerGen{})
	reg.Register("linalg.eigen.concept", &eigenConceptGen{})
	reg.Register("linalg.eigen.compute", &eigenComputeGen{})
	reg.Register("linalg.lintrans.concept", &transformationsGen{})
	reg.Register("linalg.vec.span", &spanGen{})
	reg.Register("linalg.vec.basis", &basisGen{})

	reg.Register("linalg.eigen.diagonalization", &diagonalizationGen{})
	reg.Register("linalg.matrix.rank", &rankGen{})
	reg.Register("linalg.vector.cosine_similarity", &cosineSimilarityGen{})
	reg.Register("linalg.vector.parametric", &parametricGen{})
	reg.Register("linalg.inner_product", &innerProductGen{})
	reg.Register("linalg.decomp.lu", &luGen{})
	reg.Register("linalg.decomp.qr", &qrGen{})
	reg.Register("linalg.decomp.svd", &svdGen{})
	reg.Register("linalg.decomp.jordan", &jordanGen{})
	reg.Register("linalg.subspace", &subspaceGen{})
	registerFoundations(reg)
	registerBatch(reg)
}

type vectorConceptGen struct{}

// The concept covers three things: the magnitude of a vector, its components
// along a direction, and vector operations. Upstream chooses among them by
// the knowledge-point ordinal its practice passes (ctx.KPOrdinal); the fork
// has no knowledge points, so each draw picks one of the three (fork
// adaptation of upstream eabb8e1a). Before, every draw was the magnitude
// prompt.
func (g *vectorConceptGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	switch rand.Intn(3) {
	case 1:
		return vectorComponentProblem(ctx)
	case 2:
		return vectorOperationProblem(ctx)
	default:
		return vectorMagnitudeProblem(ctx)
	}
}

// vectorTier is 0, 1 or 2 by difficulty, the bands upstream's
// mathutil.DifficultyTier uses.
func vectorTier(d float64) int {
	switch {
	case d < 0.34:
		return 0
	case d < 0.67:
		return 1
	default:
		return 2
	}
}

// vectorDrawAttempts bounds the redraws that keep the answer out of the
// question, so a guard that cannot be met costs a weaker problem, never a
// request that does not return.
const vectorDrawAttempts = 20

// containsWord reports whether word appears in q as a whole number or word,
// so an answer of 6 is found in "(6,8)" but not in "16".
func containsWord(q, word string) bool {
	fields := strings.FieldsFunc(q, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-'
	})
	for _, f := range fields {
		if f == word {
			return true
		}
	}
	return false
}

// vectorComponentProblem asks for one component after the direction vector is
// scaled to a given magnitude. The direction is a Pythagorean triple times k,
// so the answer is an integer.
func vectorComponentProblem(ctx generator.GeneratorContext) generator.Problem {
	tier := vectorTier(ctx.Difficulty)
	for attempt := 0; ; attempt++ {
		m := smallInt(2, 4+tier*2)
		n := smallInt(1, m-1)
		k := smallInt(1, 2+tier)
		x := k * (m*m - n*n)
		y := k * (2 * m * n)
		mag := k * (m*m + n*n)
		if rand.Intn(2) == 0 {
			x, y = y, x
		}
		if tier > 0 && rand.Intn(2) == 0 {
			x = -x
		}
		if tier > 0 && rand.Intn(2) == 0 {
			y = -y
		}
		scale := smallInt(2, 4+tier)
		wanted := x * scale
		question := fmt.Sprintf("A vector of magnitude %d points in the direction of \\((%d,%d)\\). What is its \\(x\\)-component?", mag*scale, x, y)
		if containsWord(question, strconv.Itoa(wanted)) && attempt < vectorDrawAttempts {
			continue
		}
		return generator.Problem{
			Question: question,
			Answer:   strconv.Itoa(wanted),
			Explanation: fmt.Sprintf("The direction vector \\((%d,%d)\\) has magnitude %d. Scaling it to magnitude %d multiplies each component by \\(%d/%d = %d\\), so the \\(x\\)-component is \\(%d \\cdot %d = %d\\).",
				x, y, mag, mag*scale, mag*scale, mag, scale, x, scale, wanted),
			Facts: map[string]string{
				"x":         strconv.Itoa(x),
				"y":         strconv.Itoa(y),
				"magnitude": strconv.Itoa(mag),
				"scale":     strconv.Itoa(scale),
				"answer":    strconv.Itoa(wanted),
			},
		}
	}
}

// vectorOperationProblem asks for one coordinate of a sum. The concept is
// graded as a number, so the question targets a single component rather than
// the tuple that the separate vector-addition concept uses.
func vectorOperationProblem(ctx generator.GeneratorContext) generator.Problem {
	tier := vectorTier(ctx.Difficulty)
	bound := 3 + tier*3
	for attempt := 0; ; attempt++ {
		a := smallInt(1, bound)
		b := smallInt(1, bound)
		c := smallInt(1, bound)
		d := smallInt(1, bound)
		if rand.Intn(2) == 0 {
			a = -a
		}
		if rand.Intn(2) == 0 {
			b = -b
		}
		if rand.Intn(2) == 0 {
			c = -c
		}
		if rand.Intn(2) == 0 {
			d = -d
		}
		answer := a + c
		question := fmt.Sprintf("Given \\(\\mathbf{u}=(%d,%d)\\) and \\(\\mathbf{v}=(%d,%d)\\), what is the \\(x\\)-component of \\(\\mathbf{u}+\\mathbf{v}\\)?", a, b, c, d)
		if containsWord(question, strconv.Itoa(answer)) && attempt < vectorDrawAttempts {
			continue
		}
		return generator.Problem{
			Question:    question,
			Answer:      strconv.Itoa(answer),
			Explanation: fmt.Sprintf("Add the \\(x\\)-components: \\(%d + (%d) = %d\\). The \\(y\\)-components add independently: \\(%d + (%d) = %d\\).", a, c, answer, b, d, b+d),
			Facts: map[string]string{
				"u1":     strconv.Itoa(a),
				"u2":     strconv.Itoa(b),
				"v1":     strconv.Itoa(c),
				"v2":     strconv.Itoa(d),
				"answer": strconv.Itoa(answer),
			},
		}
	}
}

// vectorMagnitudeProblem is the magnitude of a vector drawn from a
// Pythagorean triple, so the answer is an integer.
func vectorMagnitudeProblem(ctx generator.GeneratorContext) generator.Problem {
	triples := [][3]int{
		{3, 4, 5}, {4, 3, 5},
		{6, 8, 10}, {8, 6, 10},
		{5, 12, 13}, {12, 5, 13},
		{8, 15, 17}, {15, 8, 17},
		{9, 12, 15}, {12, 9, 15},
		{7, 24, 25}, {24, 7, 25},
	}
	if ctx.Difficulty > 0.6 {
		triples = append(triples,
			[][3]int{{20, 21, 29}, {9, 40, 41}, {11, 60, 61}}...)
	}
	t := triples[rand.Intn(len(triples))]
	x, y, mag := t[0], t[1], t[2]
	if rand.Intn(2) == 0 {
		x = -x
	}
	if rand.Intn(2) == 0 {
		y = -y
	}
	return generator.Problem{
		Question:    fmt.Sprintf("What is the magnitude of vector \\((%d,%d)\\)?", x, y),
		Answer:      fmt.Sprintf("%d", mag),
		Explanation: fmt.Sprintf("|(%d,%d)| = sqrt(%d²+%d²) = sqrt(%d+%d) = sqrt(%d) = %d", x, y, x, y, x*x, y*y, x*x+y*y, mag),
	}
}

type vectorAddGen struct{}

func (g *vectorAddGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	scale := int(1 + ctx.Difficulty*5)
	a, b := rand.Intn(max(1, scale*2))-scale, rand.Intn(max(1, scale*2))-scale
	c, d := rand.Intn(max(1, scale*2))-scale, rand.Intn(max(1, scale*2))-scale
	if rand.Intn(2) == 0 {
		sumX, sumY := a+c, b+d
		return generator.Problem{
			Question:    fmt.Sprintf("Add vectors: \\((%d,%d) + (%d,%d) =\\) ?", a, b, c, d),
			Answer:      fmt.Sprintf("(%d,%d)", sumX, sumY),
			Explanation: fmt.Sprintf("(%d,%d) + (%d,%d) = (%d,%d)", a, b, c, d, sumX, sumY),
		}
	}
	scalar := rand.Intn(max(1, scale)) + 2
	if rand.Intn(2) == 0 {
		scalar = -scalar
	}
	return generator.Problem{
		Question:    fmt.Sprintf("Compute scalar multiplication: \\(%d \\cdot (%d,%d) =\\) ?", scalar, a, b),
		Answer:      fmt.Sprintf("(%d,%d)", scalar*a, scalar*b),
		Explanation: fmt.Sprintf("%d * (%d,%d) = (%d,%d)", scalar, a, b, scalar*a, scalar*b),
	}
}

type vectorDotGen struct{}

func (g *vectorDotGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	scale := int(1 + ctx.Difficulty*5)
	a, b := rand.Intn(max(1, scale*2))-scale, rand.Intn(max(1, scale*2))-scale
	c, d := rand.Intn(max(1, scale*2))-scale, rand.Intn(max(1, scale*2))-scale
	dot := a*c + b*d
	return generator.Problem{
		Question:    fmt.Sprintf("Compute the dot product: \\((%d,%d) \\cdot (%d,%d) =\\) ?", a, b, c, d),
		Answer:      fmt.Sprintf("%d", dot),
		Explanation: fmt.Sprintf("(%d,%d) · (%d,%d) = %s*%s + %s*%s = %d + %s = %d", a, b, c, d, parenInt(a), parenInt(c), parenInt(b), parenInt(d), a*c, parenInt(b*d), dot),
	}
}

type matrixConceptGen struct{}

func (g *matrixConceptGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	scale := int(1 + ctx.Difficulty*5)
	var rows, cols int
	if rand.Intn(2) == 0 {
		rows, cols = 2, 2
	} else {
		rows, cols = 3, 3
	}
	m := make([][]int, rows)
	for i := range m {
		m[i] = make([]int, cols)
		for j := range m[i] {
			m[i][j] = rand.Intn(max(1, scale*3)) + 1
		}
	}
	r := rand.Intn(rows) + 1
	c := rand.Intn(cols) + 1
	answer := m[r-1][c-1]
	return generator.Problem{
		Question:    fmt.Sprintf("In the matrix \\(%s\\), what is the \\((%d,%d)\\) entry?", formatMatrix(m), r, c),
		Answer:      fmt.Sprintf("%d", answer),
		Explanation: fmt.Sprintf("The entry at row %d, column %d is %d.", r, c, answer),
	}
}

type matrixAddGen struct{}

// A matrix answer is written with brackets and commas. The concept is numeric,
// so the field offered digits only: the syntax existed in the accepted answer
// and nowhere in the interface.
func (g *matrixAddGen) AnswerFormat() generator.AnswerFormat {
	return generator.AnswerFormat{Hint: "Matrix with rows in brackets, e.g. [[1,2],[3,4]]", InputMode: "text"}
}

func (g *matrixAddGen) Grade(expected, userAnswer string) grader.Result {
	return gradeMatrix(expected, userAnswer)
}

func (g *matrixAddGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	scale := int(1 + ctx.Difficulty*5)
	a := make([][]int, 2)
	b := make([][]int, 2)
	for i := range a {
		a[i] = make([]int, 2)
		b[i] = make([]int, 2)
		for j := range a[i] {
			a[i][j] = rand.Intn(max(1, scale*2)) - scale
			b[i][j] = rand.Intn(max(1, scale*2)) - scale
		}
	}
	if rand.Intn(2) == 0 {
		sum := make([][]int, 2)
		for i := range sum {
			sum[i] = make([]int, 2)
			for j := range sum[i] {
				sum[i][j] = a[i][j] + b[i][j]
			}
		}
		return generator.Problem{
			Question:    fmt.Sprintf("Add matrices: \\(%s + %s =\\) ?", formatMatrix(a), formatMatrix(b)),
			Answer:      formatMatrix(sum),
			Explanation: fmt.Sprintf("%s + %s = %s (add corresponding entries)", formatMatrix(a), formatMatrix(b), formatMatrix(sum)),
		}
	}
	scalar := rand.Intn(max(1, scale)) + 2
	if rand.Intn(2) == 0 {
		scalar = -scalar
	}
	prod := make([][]int, 2)
	for i := range prod {
		prod[i] = make([]int, 2)
		for j := range prod[i] {
			prod[i][j] = scalar * a[i][j]
		}
	}
	return generator.Problem{
		Question:    fmt.Sprintf("Compute scalar multiplication: \\(%d \\cdot %s =\\) ?", scalar, formatMatrix(a)),
		Answer:      formatMatrix(prod),
		Explanation: fmt.Sprintf("%d * %s = %s", scalar, formatMatrix(a), formatMatrix(prod)),
	}
}

type matrixMultGen struct{}

func (g *matrixMultGen) AnswerFormat() generator.AnswerFormat {
	return generator.AnswerFormat{Hint: "Matrix with rows in brackets, e.g. [[1,2],[3,4]]", InputMode: "text"}
}

func (g *matrixMultGen) Grade(expected, userAnswer string) grader.Result {
	return gradeMatrix(expected, userAnswer)
}

func (g *matrixMultGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	scale := int(1 + ctx.Difficulty*5)
	a := make([][]int, 2)
	b := make([][]int, 2)
	for i := range a {
		a[i] = make([]int, 2)
		b[i] = make([]int, 2)
		for j := range a[i] {
			a[i][j] = rand.Intn(max(1, scale*2)) - scale
			b[i][j] = rand.Intn(max(1, scale*2)) - scale
		}
	}
	c := make([][]int, 2)
	for i := range c {
		c[i] = make([]int, 2)
	}
	c[0][0] = a[0][0]*b[0][0] + a[0][1]*b[1][0]
	c[0][1] = a[0][0]*b[0][1] + a[0][1]*b[1][1]
	c[1][0] = a[1][0]*b[0][0] + a[1][1]*b[1][0]
	c[1][1] = a[1][0]*b[0][1] + a[1][1]*b[1][1]
	return generator.Problem{
		Question:    fmt.Sprintf("Multiply matrices: \\(%s \\times %s =\\) ?", formatMatrix(a), formatMatrix(b)),
		Answer:      formatMatrix(c),
		Explanation: fmt.Sprintf("%s × %s = %s", formatMatrix(a), formatMatrix(b), formatMatrix(c)),
	}
}

type matrixIdentityGen struct{}

func (g *matrixIdentityGen) AnswerFormat() generator.AnswerFormat {
	return generator.AnswerFormat{Hint: "Matrix with rows in brackets, e.g. [[1,2],[3,4]]", InputMode: "text"}
}

func (g *matrixIdentityGen) Grade(expected, userAnswer string) grader.Result {
	return gradeMatrix(expected, userAnswer)
}

func (g *matrixIdentityGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	candidates := [][4]int{
		{2, 1, 1, 1}, // det=1
		{3, 2, 2, 1}, // det=-1
		{3, 4, 2, 3}, // det=1
		{4, 3, 3, 2}, // det=-1
		{5, 3, 3, 2}, // det=1
		{5, 7, 2, 3}, // det=1
		{4, 5, 3, 4}, // det=1
		{3, 1, 4, 1}, // det=-1
		{2, 5, 1, 3}, // det=1
		{1, 2, 3, 5}, // det=-1
		{2, 3, 5, 7}, // det=-1
		{7, 4, 5, 3}, // det=1
		{6, 5, 5, 4}, // det=-1
		{3, 5, 2, 3}, // det=-1
		{4, 7, 1, 2}, // det=1
		{7, 5, 4, 3}, // det=1
	}
	c := candidates[rand.Intn(len(candidates))]
	a, b, cVal, d := c[0], c[1], c[2], c[3]
	// det is ±1 for every candidate, so the inverse is integral. Its sign
	// matters: flipping it to +1 keyed six of sixteen matrices to minus their
	// true inverse, and graded the right answer wrong (Q40).
	det := a*d - b*cVal
	inv := make([][]int, 2)
	for i := range inv {
		inv[i] = make([]int, 2)
	}
	inv[0][0] = d / det
	inv[0][1] = -b / det
	inv[1][0] = -cVal / det
	inv[1][1] = a / det
	return generator.Problem{
		Question:    fmt.Sprintf("Find the inverse of the matrix \\(%s\\).", formatMatrix2([][]int{{a, b}, {cVal, d}})),
		Answer:      formatMatrix(inv),
		Explanation: identityInverseExplanation(a, b, cVal, d, det, inv),
	}
}

// identityInverseExplanation writes the 2x2 inverse as (1/det) times the
// adjugate, without the "(1/1)×" and "(1/-1)×" a template prints when det = ±1.
func identityInverseExplanation(a, b, c, d, det int, inv [][]int) string {
	adj := formatMatrix2([][]int{{d, -b}, {-c, a}})
	head := fmt.Sprintf("det = %s*%s - %s*%s = %d", parenInt(a), parenInt(d), parenInt(b), parenInt(c), det)
	switch det {
	case 1:
		return fmt.Sprintf("%s, so the inverse is the adjugate [[d,-b],[-c,a]] = %s.", head, adj)
	case -1:
		return fmt.Sprintf("%s, so the inverse is the adjugate [[d,-b],[-c,a]] = %s with every sign flipped: %s.", head, adj, formatMatrix(inv))
	}
	return fmt.Sprintf("%s, so the inverse is (1/%d) times the adjugate %s, which is %s.", head, det, adj, formatMatrix(inv))
}

type det2x2Gen struct{}

func (g *det2x2Gen) Generate(ctx generator.GeneratorContext) generator.Problem {
	scale := int(1 + ctx.Difficulty*5)
	a, b := rand.Intn(max(1, scale*2))-scale, rand.Intn(max(1, scale*2))-scale
	c, d := rand.Intn(max(1, scale*2))-scale, rand.Intn(max(1, scale*2))-scale
	det := a*d - b*c
	return generator.Problem{
		Question:    fmt.Sprintf("Find the determinant of \\(%s\\).", formatMatrix2([][]int{{a, b}, {c, d}})),
		Answer:      fmt.Sprintf("%d", det),
		Explanation: fmt.Sprintf("det = %d*%d - %d*%d = %d-%d = %d", a, d, b, c, a*d, b*c, det),
	}
}

type det3x3Gen struct{}

func (g *det3x3Gen) Generate(ctx generator.GeneratorContext) generator.Problem {
	scale := int(1 + ctx.Difficulty*5)
	m := make([][]int, 3)
	for i := range m {
		m[i] = make([]int, 3)
		for j := range m[i] {
			m[i][j] = rand.Intn(max(1, scale*2)) - scale
		}
	}
	a, b, c := m[0][0], m[0][1], m[0][2]
	d, e, f := m[1][0], m[1][1], m[1][2]
	gVal, h, i := m[2][0], m[2][1], m[2][2]
	det := a*(e*i-f*h) - b*(d*i-f*gVal) + c*(d*h-e*gVal)
	return generator.Problem{
		Question: fmt.Sprintf("Find the determinant of \\(%s\\).", formatMatrix(m)),
		Answer:   fmt.Sprintf("%d", det),
		// Negative entries are parenthesised, so no "- -2*(...)" or "--2*-2" (Q35).
		Explanation: fmt.Sprintf("Expand along the first row: det = %s*(%s*%s - %s*%s) - %s*(%s*%s - %s*%s) + %s*(%s*%s - %s*%s) = %s*%s - %s*%s + %s*%s = %d",
			parenInt(a), parenInt(e), parenInt(i), parenInt(f), parenInt(h),
			parenInt(b), parenInt(d), parenInt(i), parenInt(f), parenInt(gVal),
			parenInt(c), parenInt(d), parenInt(h), parenInt(e), parenInt(gVal),
			parenInt(a), parenInt(e*i-f*h), parenInt(b), parenInt(d*i-f*gVal), parenInt(c), parenInt(d*h-e*gVal), det),
	}
}

type systemsMatrixGen struct{}

func (g *systemsMatrixGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	scale := int(1 + ctx.Difficulty*5)
	x := rand.Intn(max(1, scale*2)) - scale
	y := rand.Intn(max(1, scale*2)) - scale
	a, b := rand.Intn(max(1, scale*2))-scale, rand.Intn(max(1, scale*2))-scale
	c, d := rand.Intn(max(1, scale*2))-scale, rand.Intn(max(1, scale*2))-scale
	if a == 0 && c == 0 {
		a = 1
		c = 2
	}
	if b == 0 && d == 0 {
		b = 1
		d = 3
	}
	e1 := a*x + b*y
	e2 := c*x + d*y
	return generator.Problem{
		Question:    fmt.Sprintf("Solve the linear system using matrices: \\(%dx + %dy = %d\\), \\(%dx + %dy = %d\\)", a, b, e1, c, d, e2),
		Answer:      fmt.Sprintf("(%d,%d)", x, y),
		Explanation: fmt.Sprintf("The solution is x=%d, y=%d.", x, y),
	}
}

type cramerGen struct{}

func (g *cramerGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	scale := int(1 + ctx.Difficulty*5)
	x := rand.Intn(max(1, scale*2)) - scale
	y := rand.Intn(max(1, scale*2)) - scale
	a, b := rand.Intn(max(1, scale*2))-scale, rand.Intn(max(1, scale*2))-scale
	c, d := rand.Intn(max(1, scale*2))-scale, rand.Intn(max(1, scale*2))-scale
	if a == 0 && c == 0 {
		a = 1
		c = 2
	}
	if b == 0 && d == 0 {
		b = 1
		d = 3
	}
	e1 := a*x + b*y
	e2 := c*x + d*y
	det := a*d - b*c
	if det == 0 {
		det = 1
	}
	askX := rand.Intn(2) == 0
	if askX {
		detX := e1*d - b*e2
		valX := detX / det
		return generator.Problem{
			Question:    fmt.Sprintf("Use Cramer's rule to find \\(x\\): \\(%dx + %dy = %d\\), \\(%dx + %dy = %d\\)", a, b, e1, c, d, e2),
			Answer:      fmt.Sprintf("%d", valX),
			Explanation: fmt.Sprintf("x = det([[%d,%d],[%d,%d]]) / det([[%d,%d],[%d,%d]]) = (%d*%d-%d*%d)/(%d*%d-%d*%d) = %d/%d = %d", e1, b, e2, d, a, b, c, d, e1, d, b, e2, a, d, b, c, detX, det, valX),
		}
	}
	detY := a*e2 - e1*c
	valY := detY / det
	return generator.Problem{
		Question:    fmt.Sprintf("Use Cramer's rule to find \\(y\\): \\(%dx + %dy = %d\\), \\(%dx + %dy = %d\\)", a, b, e1, c, d, e2),
		Answer:      fmt.Sprintf("%d", valY),
		Explanation: fmt.Sprintf("y = det([[%d,%d],[%d,%d]]) / det([[%d,%d],[%d,%d]]) = (%d*%d-%d*%d)/(%d*%d-%d*%d) = %d/%d = %d", a, e1, c, e2, a, b, c, d, a, e2, e1, c, a, d, b, c, detY, det, valY),
	}
}

type eigenConceptGen struct{}

func (g *eigenConceptGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	if rand.Intn(2) == 0 {
		// A definition is a choice, not free text (fork fix: it was served as
		// multiple choice with no options and string-matched).
		p := generator.MultipleChoice("Which statement defines an eigenvalue \\(\\lambda\\) of a square matrix \\(A\\)?",
			"A scalar \\(\\lambda\\) with \\(Av=\\lambda v\\) for some nonzero vector \\(v\\)",
			"A scalar \\(\\lambda\\) with \\(\\det(A)=\\lambda\\)",
			"A vector \\(v\\) with \\(Av=0\\)",
			"A scalar \\(\\lambda\\) with \\(A-\\lambda I\\) invertible")
		p.Explanation = "An eigenvalue \\(\\lambda\\) satisfies \\(Av=\\lambda v\\) for some nonzero \\(v\\), equivalently \\(\\det(A-\\lambda I)=0\\)."
		return p
	}
	scale := int(1 + ctx.Difficulty*5)
	a, d := rand.Intn(max(1, scale*2))-scale, rand.Intn(max(1, scale*2))-scale
	if a == 0 {
		a = 1
	}
	if d == 0 {
		d = 2
	}
	lambda := rand.Intn(max(1, scale*2)) - scale
	det := (a - lambda) * (d - lambda)
	answer := "no"
	if det == 0 {
		answer = "yes"
	}
	return generator.Problem{
		Question:    fmt.Sprintf("Is \\(\\lambda=%d\\) an eigenvalue of matrix \\(%s\\)?", lambda, formatMatrix2([][]int{{a, 0}, {0, d}})),
		Answer:      answer,
		Explanation: fmt.Sprintf("det(A-λI) = det([[%d,%d],[%d,%d]]) = (%d)*(%d) = %d. Since det %s 0, λ=%d %s an eigenvalue.", a-lambda, 0, 0, d-lambda, a-lambda, d-lambda, det, map[bool]string{true: "=", false: "≠"}[det == 0], lambda, map[bool]string{true: "is", false: "is not"}[det == 0]),
	}
}

type eigenComputeGen struct{}

func (g *eigenComputeGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	scale := int(1 + ctx.Difficulty*5)
	a := rand.Intn(max(1, scale*2)) - scale
	c := rand.Intn(max(1, scale*2)) - scale
	d := rand.Intn(max(1, scale*2)) - scale
	if a == 0 {
		a = 2
	}
	if d == 0 {
		d = 1
	}
	return generator.Problem{
		Question:    fmt.Sprintf("Find the eigenvalues of the triangular matrix \\(%s\\).", formatMatrix2([][]int{{a, c}, {0, d}})),
		Answer:      fmt.Sprintf("%d,%d", a, d),
		Explanation: fmt.Sprintf("For a triangular matrix, eigenvalues are the diagonal entries: λ₁=%d, λ₂=%d.", a, d),
	}
}

type transformationsGen struct{}

func (g *transformationsGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	scale := int(1 + ctx.Difficulty*5)
	a, b := rand.Intn(max(1, scale*2))+1, rand.Intn(max(1, scale*2))+1
	x, y := rand.Intn(max(1, scale*2))-scale, rand.Intn(max(1, scale*2))-scale
	return generator.Problem{
		Question:    fmt.Sprintf("Under the linear transformation \\(T(x,y) = (%dx,%dy)\\), where does the point \\((%d,%d)\\) map to?", a, b, x, y),
		Answer:      fmt.Sprintf("(%d,%d)", a*x, b*y),
		Explanation: fmt.Sprintf("T(%d,%d) = (%d*%d, %d*%d) = (%d,%d)", x, y, a, x, b, y, a*x, b*y),
	}
}

type spanGen struct{}

func (g *spanGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	scale := int(1 + ctx.Difficulty*5)
	r := func() int { return rand.Intn(max(1, scale*2)+1) - scale }
	q := func(v1, v2, w1, w2 int) string {
		return fmt.Sprintf("Do the vectors \\(v=(%d,%d)\\) and \\(w=(%d,%d)\\) span \\(\\mathbb{R}^{2}\\)?", v1, v2, w1, w2)
	}
	// Half the items span and half do not, so the key carries no bias.
	if rand.Intn(2) == 0 {
		v1, v2, w1, w2 := r(), r(), r(), r()
		for v1*w2-v2*w1 == 0 {
			v1, v2, w1, w2 = r(), r(), r(), r()
		}
		det := v1*w2 - v2*w1
		return generator.Problem{
			Question:    q(v1, v2, w1, w2),
			Answer:      "yes",
			Explanation: fmt.Sprintf("det([[%d,%d],[%d,%d]]) = %d. Since the determinant is nonzero, the vectors are linearly independent, so they do span R².", v1, v2, w1, w2, det),
		}
	}
	// Dependent pair: both are multiples of one direction u, and not always
	// with w an integer multiple of v (v=(2,4), w=(3,6) is w = (3/2)v).
	dirs := [][2]int{{1, 0}, {0, 1}, {1, 1}, {1, -1}, {1, 2}, {2, 1}, {1, -2}, {2, -1}, {1, 3}, {3, 1}, {2, 3}, {3, -2}}
	u := dirs[rand.Intn(len(dirs))]
	ms := []int{1, 2, 3, -1, -2, -3}
	m := ms[rand.Intn(len(ms))]
	n := ms[rand.Intn(len(ms))]
	for n == m {
		n = ms[rand.Intn(len(ms))]
	}
	v1, v2, w1, w2 := m*u[0], m*u[1], n*u[0], n*u[1]
	ratio := fmt.Sprintf("%d", n/m)
	if n%m != 0 {
		g := spanGCD(n, m)
		nn, mm := n/g, m/g
		if mm < 0 {
			nn, mm = -nn, -mm
		}
		ratio = fmt.Sprintf("(%d/%d)", nn, mm)
	}
	return generator.Problem{
		Question:    q(v1, v2, w1, w2),
		Answer:      "no",
		Explanation: fmt.Sprintf("det([[%d,%d],[%d,%d]]) = 0: w = %s·v, so the vectors are linearly dependent and both lie on one line through the origin. Two dependent vectors do not span R².", v1, v2, w1, w2, ratio),
	}
}

func spanGCD(a, b int) int {
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

type basisGen struct{}

func (g *basisGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	scale := int(1 + ctx.Difficulty*5)
	qType := rand.Intn(3)
	switch qType {
	case 0:
		return generator.Problem{
			Question:    "What is the dimension of \\(\\mathbb{R}^{3}\\)?",
			Answer:      "3",
			Explanation: "\\(\\mathbb{R}^{3}\\) has dimension 3 because it has a basis of 3 vectors, e.g., \\(\\{(1,0,0), (0,1,0), (0,0,1)\\}\\).",
		}
	case 1:
		return generator.Problem{
			Question:    "What is the dimension of \\(\\mathbb{R}^{2}\\)?",
			Answer:      "2",
			Explanation: "\\(\\mathbb{R}^{2}\\) has dimension 2 because it has a basis of 2 vectors, e.g., \\(\\{(1,0), (0,1)\\}\\).",
		}
	default:
		v1, v2 := rand.Intn(max(1, scale*2))-scale, rand.Intn(max(1, scale*2))-scale
		w1, w2 := rand.Intn(max(1, scale*2))-scale, rand.Intn(max(1, scale*2))-scale
		if v1 == 0 && v2 == 0 {
			v1, v2 = 1, 0
		}
		if w1 == 0 && w2 == 0 {
			w1, w2 = 0, 1
		}
		det := v1*w2 - v2*w1
		answer := "no"
		if det != 0 {
			answer = "yes"
		}
		return generator.Problem{
			Question:    fmt.Sprintf("Is the set \\(\\{(%d,%d), (%d,%d)\\}\\) a basis for \\(\\mathbb{R}^{2}\\)?", v1, v2, w1, w2),
			Answer:      answer,
			Explanation: fmt.Sprintf("det = %d, so the vectors are %s. Since they %s span R² and are %s, they %s a basis.", det, map[bool]string{true: "linearly independent", false: "linearly dependent"}[det != 0], map[bool]string{true: "do", false: "do not"}[det != 0], map[bool]string{true: "independent", false: "dependent"}[det != 0], map[bool]string{true: "form", false: "do not form"}[det != 0]),
		}
	}
}

func formatMatrix(m [][]int) string {
	if len(m) == 2 && len(m[0]) == 2 {
		return formatMatrix2(m)
	}
	if len(m) == 3 && len(m[0]) == 3 {
		return formatMatrix3(m)
	}
	return fmt.Sprintf("%v", m)
}

func formatMatrix2(m [][]int) string {
	return fmt.Sprintf("[[%d,%d],[%d,%d]]", m[0][0], m[0][1], m[1][0], m[1][1])
}

func formatMatrix3(m [][]int) string {
	return fmt.Sprintf("[[%d,%d,%d],[%d,%d,%d],[%d,%d,%d]]",
		m[0][0], m[0][1], m[0][2],
		m[1][0], m[1][1], m[1][2],
		m[2][0], m[2][1], m[2][2])
}

// parseMatrix reads a whole answer as a 2x2 integer matrix (Q48). It accepts
// "[[-7, 3], [5, -2]]" with any spacing, "(" for "[", whole-number decimals
// ("-7.0") and rows written "-7 3; 5 -2". The whole answer must be the matrix:
// "[[2,2],[0,0]] + 1" used to match its prefix and grade right.
func parseMatrix(s string) ([][]int, bool) {
	t := strings.NewReplacer("(", "[", ")", "]", "−", "-").Replace(strings.TrimSpace(s))
	var rows []string
	if strings.HasPrefix(t, "[") {
		compact := strings.Join(strings.Fields(t), "")
		m := matrixFullRe.FindStringSubmatch(compact)
		if m == nil {
			return nil, false
		}
		rows = []string{m[1], m[2]}
	} else {
		rows = strings.Split(t, ";")
		if len(rows) != 2 {
			return nil, false
		}
	}
	out := make([][]int, 2)
	for i, r := range rows {
		fields := strings.FieldsFunc(r, func(c rune) bool { return c == ',' || c == ' ' || c == '\t' })
		if len(fields) != 2 {
			return nil, false
		}
		out[i] = make([]int, 2)
		for j, f := range fields {
			v, err := strconv.ParseFloat(f, 64)
			if err != nil || v != math.Trunc(v) {
				return nil, false
			}
			out[i][j] = int(v)
		}
	}
	return out, true
}

var matrixFullRe = regexp.MustCompile(`^\[\[([^\[\]]+)\],\[([^\[\]]+)\]\]$`)

func gradeMatrix(expected, userAnswer string) grader.Result {
	eMat, ok := parseMatrix(expected)
	if !ok {
		return grader.Result{Correct: false, Score: 0, Feedback: "Invalid matrix format"}
	}
	aMat, ok := parseMatrix(userAnswer)
	if !ok {
		return grader.Result{Correct: false, Score: 0, Feedback: "Invalid matrix format"}
	}
	if len(eMat) != len(aMat) || len(eMat[0]) != len(aMat[0]) {
		return grader.Result{Correct: false, Score: 0, Feedback: "Matrix dimension mismatch"}
	}
	for i := range eMat {
		for j := range eMat[i] {
			if eMat[i][j] != aMat[i][j] {
				return grader.Result{Correct: false, Score: 0, Feedback: fmt.Sprintf("Mismatch at (%d,%d)", i+1, j+1)}
			}
		}
	}
	return grader.Result{Correct: true, Score: 1}
}

// Diagonalization: find P and D such that A = PDP⁻¹ for a 2x2 matrix
type diagonalizationGen struct{}

func (g *diagonalizationGen) Grade(expected, userAnswer string) grader.Result {
	// Any valid pair is right: D diagonal with A's eigenvalues, P invertible
	// with A P = P D. A is recovered from the reference pair (fork fix: the
	// generator used to expect P = A and string-match it).
	return gradeDiagonalization(expected, userAnswer)
}

func (g *diagonalizationGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	// Diagonalizable matrices (real distinct eigenvalues)
	type entry struct {
		a, b, c, d int
		l1, l2     int
	}
	entries := []entry{
		{2, 0, 0, 3, 2, 3},  // diagonal already
		{3, 1, 0, 2, 3, 2},  // triangular
		{1, 2, 2, 1, 3, -1}, // need eigenvectors
		{5, 0, 0, 4, 5, 4},
		{2, 1, 0, 1, 2, 1},
		{3, 0, 0, 5, 3, 5},
		{2, 2, 0, 4, 2, 4},
	}
	e := entries[rand.Intn(len(entries))]
	// An eigenvector of [[a,b],[c,d]] for λ: (b, λ−a) when b≠0, (λ−d, c) when
	// c≠0, else a standard basis vector.
	vec := func(l int) (int, int) {
		switch {
		case e.b != 0:
			return e.b, l - e.a
		case e.c != 0:
			return l - e.d, e.c
		case l == e.a:
			return 1, 0
		default:
			return 0, 1
		}
	}
	x1, y1 := vec(e.l1)
	x2, y2 := vec(e.l2)
	answer := fmt.Sprintf("P=[[%d,%d],[%d,%d]] D=[[%d,0],[0,%d]]", x1, x2, y1, y2, e.l1, e.l2)
	return generator.Problem{
		Question:    fmt.Sprintf("Find matrices \\(P\\) and \\(D\\) (diagonal) such that \\(A = PDP^{-1}\\) for \\(A = %s\\). Answer as P=[[..],[..]] D=[[..],[..]]; any valid pair counts.", formatMatrix2([][]int{{e.a, e.b}, {e.c, e.d}})),
		Answer:      answer,
		Explanation: fmt.Sprintf("Eigenvalues are λ=%d and λ=%d; eigenvectors (%d,%d) and (%d,%d) form the columns of P. One valid answer: %s", e.l1, e.l2, x1, y1, x2, y2, answer),
	}
}

func parse2x2(s string) ([2][2]float64, bool) {
	var m [2][2]float64
	mm := matrixRe.FindStringSubmatch(strings.ReplaceAll(s, " ", ""))
	if mm == nil {
		return m, false
	}
	for i, row := range mm[1:3] {
		cells := strings.Split(row, ",")
		if len(cells) != 2 {
			return m, false
		}
		for j, c := range cells {
			v, err := parseRational(c)
			if err != nil {
				return m, false
			}
			m[i][j] = v
		}
	}
	return m, true
}

func parseRational(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '/'); i > 0 {
		n, err := strconv.ParseFloat(s[:i], 64)
		if err != nil {
			return 0, err
		}
		d, err := strconv.ParseFloat(s[i+1:], 64)
		if err != nil || d == 0 {
			return 0, fmt.Errorf("bad fraction")
		}
		return n / d, nil
	}
	return strconv.ParseFloat(s, 64)
}

func mul2(a, b [2][2]float64) [2][2]float64 {
	var r [2][2]float64
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			r[i][j] = a[i][0]*b[0][j] + a[i][1]*b[1][j]
		}
	}
	return r
}

func gradeDiagonalization(expected, userAnswer string) grader.Result {
	parsePair := func(s string) (p, d [2][2]float64, ok bool) {
		parts := strings.SplitN(strings.ReplaceAll(s, " ", ""), "D=", 2)
		if len(parts) != 2 {
			return p, d, false
		}
		p, ok1 := parse2x2(strings.TrimPrefix(parts[0], "P="))
		d, ok2 := parse2x2(parts[1])
		return p, d, ok1 && ok2
	}
	eP, eD, ok := parsePair(expected)
	if !ok {
		return grader.Result{Correct: false, Score: 0, Feedback: "Expected format: P=[[...]] D=[[...]]"}
	}
	uP, uD, ok := parsePair(userAnswer)
	if !ok {
		return grader.Result{Correct: false, Score: 0, Feedback: "Answer as P=[[a,b],[c,d]] D=[[λ1,0],[0,λ2]]"}
	}
	// A = P D P^{-1} from the reference pair.
	det := eP[0][0]*eP[1][1] - eP[0][1]*eP[1][0]
	if det == 0 {
		return grader.Result{Correct: false, Score: 0, Feedback: "Incorrect P or D"}
	}
	inv := [2][2]float64{{eP[1][1] / det, -eP[0][1] / det}, {-eP[1][0] / det, eP[0][0] / det}}
	A := mul2(mul2(eP, eD), inv)
	near := func(x, y float64) bool { return math.Abs(x-y) < 1e-6 }
	if !near(uD[0][1], 0) || !near(uD[1][0], 0) {
		return grader.Result{Correct: false, Score: 0, Feedback: "D must be diagonal"}
	}
	if udet := uP[0][0]*uP[1][1] - uP[0][1]*uP[1][0]; near(udet, 0) {
		return grader.Result{Correct: false, Score: 0, Feedback: "P must be invertible (its columns are the eigenvectors)"}
	}
	ap, pd := mul2(A, uP), mul2(uP, uD)
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			if !near(ap[i][j], pd[i][j]) {
				return grader.Result{Correct: false, Score: 0, Feedback: "A P ≠ P D: check the eigenvectors against their eigenvalues (same column order in P and D)"}
			}
		}
	}
	return grader.Result{Correct: true, Score: 1}
}

// Rank of a matrix
type rankGen struct{}

func (g *rankGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type entry struct {
		matrix [][]int
		rank   int
	}
	entries := []entry{
		{[][]int{{1, 0}, {0, 1}}, 2},
		{[][]int{{1, 2}, {2, 4}}, 1},
		{[][]int{{1, 0, 0}, {0, 0, 0}}, 1},
		{[][]int{{1, 2, 3}, {0, 0, 0}}, 1},
		{[][]int{{1, 0}, {0, 0}}, 1},
		{[][]int{{1, 2}, {3, 4}}, 2},
		{[][]int{{1, 0, 0}, {0, 1, 0}}, 2},
		{[][]int{{0, 0}, {0, 0}}, 0},
		{[][]int{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}, 3},
		{[][]int{{1, 1, 1}, {2, 2, 2}, {3, 3, 3}}, 1},
	}
	e := entries[rand.Intn(len(entries))]
	return generator.Problem{
		Question:    fmt.Sprintf("What is the rank of matrix \\(%s\\)?", formatMatrix(e.matrix)),
		Answer:      fmt.Sprintf("%d", e.rank),
		Explanation: fmt.Sprintf("The rank is %d (the number of linearly independent rows/columns).", e.rank),
	}
}

// Cosine similarity between two vectors
type cosineSimilarityGen struct{}

func (g *cosineSimilarityGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type entry struct {
		u, v [2]int
		cos  int
	}
	entries := []entry{
		{[2]int{1, 0}, [2]int{1, 0}, 1},
		{[2]int{1, 0}, [2]int{0, 1}, 0},
		{[2]int{1, 1}, [2]int{1, 1}, 1},
		{[2]int{1, 0}, [2]int{-1, 0}, -1},
		{[2]int{3, 4}, [2]int{6, 8}, 1},
		{[2]int{1, 1}, [2]int{1, -1}, 0},
		{[2]int{1, 2}, [2]int{2, 4}, 1},
		{[2]int{1, 0}, [2]int{0, -1}, 0},
		{[2]int{1, 1}, [2]int{-1, 1}, 0},
	}
	e := entries[rand.Intn(len(entries))]
	uStr := fmt.Sprintf("(%d,%d)", e.u[0], e.u[1])
	vStr := fmt.Sprintf("(%d,%d)", e.v[0], e.v[1])
	dot := e.u[0]*e.v[0] + e.u[1]*e.v[1]
	uMag := int(math.Sqrt(float64(e.u[0]*e.u[0] + e.u[1]*e.u[1])))
	vMag := int(math.Sqrt(float64(e.v[0]*e.v[0] + e.v[1]*e.v[1])))
	return generator.Problem{
		Question:    fmt.Sprintf("Compute the cosine similarity between \\(u=%s\\) and \\(v=%s\\).", uStr, vStr),
		Answer:      fmt.Sprintf("%d", e.cos),
		Explanation: fmt.Sprintf("cos(θ) = (u·v)/(|u||v|) = (%d)/(%d×%d) = %d/%d = %d", dot, uMag, vMag, dot, uMag*vMag, e.cos),
	}
}

// Parametric form of a line
type parametricGen struct{}

func (g *parametricGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type entry struct {
		x0, y0, dx, dy int
	}
	directions := []entry{
		{1, 2, 3, 4},
		{0, 0, 1, 2},
		{2, 3, -1, 2},
		{-1, 1, 2, -3},
		{0, 1, 1, 0},
		{1, 0, 0, 1},
		{3, 0, 1, 1},
		{-2, 4, 3, -1},
	}
	e := directions[rand.Intn(len(directions))]
	return generator.Problem{
		Question:    fmt.Sprintf("Find the parametric equations for the line through \\((%d,%d)\\) with direction vector \\((%d,%d)\\).", e.x0, e.y0, e.dx, e.dy),
		Answer:      fmt.Sprintf("x=%d+%dt,y=%d+%dt", e.x0, e.dx, e.y0, e.dy),
		Explanation: fmt.Sprintf("The parametric form is (x,y) = (%d,%d) + t(%d,%d), so x=%d+%dt, y=%d+%dt.", e.x0, e.y0, e.dx, e.dy, e.x0, e.dx, e.y0, e.dy),
	}
}

// Grade accepts each equation in any equivalent form: "x=3+t, y=t" for the
// stored "x=3+1t,y=0+1t", in either order, with or without spaces. The tuple
// grader compared the strings, so the tidy form the explanation shows was
// marked wrong.
func (g *parametricGen) Grade(expected, userAnswer string) grader.Result {
	want, ok1 := parametricSides(expected)
	got, ok2 := parametricSides(userAnswer)
	if !ok1 || !ok2 || len(got) != len(want) {
		return grader.Result{Correct: false, Feedback: "Give both equations, e.g. x=1+2t, y=3-t"}
	}
	r := grader.NewRouter()
	for v, rhs := range want {
		u, ok := got[v]
		if !ok || !r.Grade(grader.GradingExpression, rhs, u).Correct {
			return grader.Result{Correct: false}
		}
	}
	return grader.Result{Correct: true, Score: 1}
}

func parametricSides(s string) (map[string]string, bool) {
	s = strings.NewReplacer(" ", "", "(", "", ")", "", ";", ",").Replace(strings.ToLower(s))
	out := map[string]string{}
	for _, part := range strings.Split(s, ",") {
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 || (kv[0] != "x" && kv[0] != "y") || kv[1] == "" {
			return nil, false
		}
		out[kv[0]] = kv[1]
	}
	return out, len(out) == 2
}

type innerProductGen struct{}

func (g *innerProductGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	scale := int(1 + ctx.Difficulty*4)
	type entry struct {
		question string
		answer   string
		reason   string
	}
	easy := []entry{
		{"Is inner product 〈u,v〉=u·v positive definite? (yes/no)", "yes", "u·u=|u|²≥0 and =0 iff u=0."},
		{"Are orthogonal vectors u·v=0? (yes/no)", "yes", "Definition of orthogonality."},
		{"Does Cauchy-Schwarz state |〈u,v〉|≤||u||·||v||? (yes/no)", "yes", "Cauchy-Schwarz inequality."},
		{"Is every symmetric bilinear form an inner product? (yes/no)", "no", "Needs positive definiteness too: xᵀAx with indefinite A fails."},
	}
	hard := []entry{
		{"Is 〈u,v〉=2u₁v₁+u₂v₂ an inner product on R²? (yes/no)", "yes", "Weighted dot with positive weights is inner product."},
		{"Does Gram-Schmidt use projections via inner products? (yes/no)", "yes", "Orthogonalization via subtract projections."},
		{"Is every inner product induced by a positive-definite matrix? (yes/no)", "yes", "〈u,v〉=uᵀAv with A SPD."},
	}
	pool := easy
	if scale > 3 {
		pool = append(easy, hard...)
	}
	if rand.Intn(3) == 0 {
		// production: dot products and norms (unique answers).
		type dotEntry struct {
			question string
			answer   string
			reason   string
		}
		dotTable := []dotEntry{
			{"Compute (1,2)·(3,4) (enter a number)", "11", "1·3+2·4 = 11."},
			{"What is (1,0)·(0,1)? (enter a number)", "0", "Perpendicular unit vectors: dot is 0."},
			{"What is ||(3,4)||? (enter a number)", "5", "sqrt(9+16) = 5."},
		}
		d := dotTable[rand.Intn(len(dotTable))]
		return generator.Problem{Question: d.question, Answer: d.answer, Explanation: d.reason}
	}
	e := pool[rand.Intn(len(pool))]
	return generator.Problem{Question: e.question, Answer: e.answer, Explanation: e.reason}
}

type luGen struct{}

func (g *luGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	scale := int(1 + ctx.Difficulty*4)
	type entry struct {
		question string
		answer   string
		reason   string
	}
	easy := []entry{
		{"Does LU decomposition write A=LU with L lower unit and U upper? (yes/no)", "yes", "LU factors invertible; solve via forward/back substitution."},
		{"Is LU without pivoting possible only if leading principals nonzero? (yes/no)", "yes", "Need nonzero pivots."},
		{"Does LU solve Ax=b via Ly=b then Ux=y? (yes/no)", "yes", "Two triangular solves."},
		{"Does every invertible matrix admit LU without row swaps? (yes/no)", "no", "Zero leading pivots need permutation P: PA=LU."},
	}
	hard := []entry{
		{"For A=[[2,1],[4,3]], is L=[[1,0],[2,1]] and U=[[2,1],[0,1]]? (yes/no)", "yes", "Multiply L×U = A."},
		{"Does PA=LU with permutation P handle zero pivots? (yes/no)", "yes", "Partial pivoting."},
		{"Is determinant det(A)=det(U) for LU with unit L? (yes/no)", "yes", "det(L)=1, so det(A)=product of U diagonal."},
	}
	pool := easy
	if scale > 3 {
		pool = append(easy, hard...)
	}
	if rand.Intn(3) == 0 {
		// production: triangular solves and determinants (unique answers).
		type luEntry struct {
			question string
			answer   string
			reason   string
		}
		luTable := []luEntry{
			{"Ly=b with L=[[1,0],[2,1]], b=[1,3]. What is y_2? (enter a number)", "1", "y_1=1, then 2·1+y_2=3 gives y_2=1."},
			{"LU with unit L and U diagonal (2,3): what is det(A)? (enter a number)", "6", "det(L)=1, so det(A) = 2·3 = 6."},
			{"How many triangular solves does Ax=b need via LU? (enter a number)", "2", "Forward Ly=b, then back Ux=y."},
		}
		v := luTable[rand.Intn(len(luTable))]
		return generator.Problem{Question: v.question, Answer: v.answer, Explanation: v.reason}
	}
	e := pool[rand.Intn(len(pool))]
	return generator.Problem{Question: e.question, Answer: e.answer, Explanation: e.reason}
}

type qrGen struct{}

func (g *qrGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	scale := int(1 + ctx.Difficulty*4)
	type entry struct {
		question string
		answer   string
		reason   string
	}
	easy := []entry{
		{"Does QR write A=QR with Q orthogonal and R upper triangular? (yes/no)", "yes", "QR via Gram-Schmidt."},
		{"Is Q's columns orthonormal? (yes/no)", "yes", "QᵀQ=I."},
		{"Does QR solve least squares via R x = Qᵀb? (yes/no)", "yes", "Normal equations stability."},
		{"Is R lower triangular? (yes/no)", "no", "R is upper triangular by construction."},
	}
	hard := []entry{
		{"For A=[[1,1],[1,0]], is Q's first column normalized (1,1)/√2? (yes/no)", "yes", "Gram-Schmidt on columns."},
		{"Does Householder reflection give stable QR? (yes/no)", "yes", "Householder is stable."},
		{"Is R's diagonal positive if using standard Gram-Schmidt? (yes/no)", "yes", "By convention."},
	}
	pool := easy
	if scale > 3 {
		pool = append(easy, hard...)
	}
	if rand.Intn(3) == 0 {
		// production: Gram-Schmidt quantities (unique answers).
		type qrEntry struct {
			question string
			answer   string
			reason   string
		}
		qrTable := []qrEntry{
			{"Gram-Schmidt on (1,1): what is ||(1,1)||? (enter like sqrt(2))", "sqrt(2)", "sqrt(1+1) = sqrt(2)."},
			{"R is 2x2 upper triangular. How many possibly-nonzero entries? (enter a number)", "3", "Two diagonal plus one above: 3."},
			{"Least squares via Rx=Qᵀb with upper-triangular R: forward or back substitution? (type one word)", "back", "Upper-triangular systems solve bottom-up: back substitution."},
		}
		v := qrTable[rand.Intn(len(qrTable))]
		return generator.Problem{Question: v.question, Answer: v.answer, Explanation: v.reason}
	}
	e := pool[rand.Intn(len(pool))]
	return generator.Problem{Question: e.question, Answer: e.answer, Explanation: e.reason}
}

type svdGen struct{}

func (g *svdGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	scale := int(1 + ctx.Difficulty*4)
	type entry struct {
		question string
		answer   string
		reason   string
	}
	easy := []entry{
		{"Does SVD write A=UΣVᵀ with U,V orthogonal and Σ diagonal? (yes/no)", "yes", "SVD exists for any real matrix."},
		{"Are singular values σ_i ≥0? (yes/no)", "yes", "By definition."},
		{"Does rank(A) equal number of nonzero singular values? (yes/no)", "yes", "Counting."},
		{"Is SVD defined only for square matrices? (yes/no)", "no", "SVD exists for any m×n real matrix."},
	}
	hard := []entry{
		{"Is σ_i = sqrt(eig(AᵀA))? (yes/no)", "yes", "Singular values via AᵀA."},
		{"Does truncated SVD give best low-rank approximation? (yes/no)", "yes", "Eckart-Young theorem."},
		{"Is U's columns left singular vectors, V's right? (yes/no)", "yes", "By construction."},
	}
	pool := easy
	if scale > 3 {
		pool = append(easy, hard...)
	}
	if rand.Intn(3) == 0 {
		// production: singular-value arithmetic (unique answers).
		type svdEntry struct {
			question string
			answer   string
			reason   string
		}
		svdTable := []svdEntry{
			{"A has singular values 5, 2, 0. What is rank(A)? (enter a number)", "2", "Two nonzero singular values: rank 2."},
			{"Best rank-1 approximation keeps how many singular values? (enter a number)", "1", "Truncate to the top one (Eckart-Young)."},
			{"AᵀA has eigenvalues 9, 4, 0. What is the largest singular value? (enter a number)", "3", "σ = sqrt(eig): sqrt(9) = 3."},
		}
		v := svdTable[rand.Intn(len(svdTable))]
		return generator.Problem{Question: v.question, Answer: v.answer, Explanation: v.reason}
	}
	e := pool[rand.Intn(len(pool))]
	return generator.Problem{Question: e.question, Answer: e.answer, Explanation: e.reason}
}

type jordanGen struct{}

func (g *jordanGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	scale := int(1 + ctx.Difficulty*4)
	type entry struct {
		question string
		answer   string
		reason   string
	}
	easy := []entry{
		{"Does Jordan form exist for every complex matrix? (yes/no)", "yes", "Jordan canonical form over C."},
		{"Is diagonalizable matrices' Jordan blocks all 1×1? (yes/no)", "yes", "No nontrivial blocks."},
		{"Does Jordan block J_2(λ) have λ on diagonal and 1 above? (yes/no)", "yes", "[[λ,1],[0,λ]]."},
		{"Is every real matrix diagonalizable over R? (yes/no)", "no", "Rotations have no real eigenvectors; Jordan blocks appear."},
	}
	hard := []entry{
		{"Is matrix [[2,1],[0,2]] already in Jordan form? (yes/no)", "yes", "Single Jordan block for eigenvalue 2."},
		{"Does minimal polynomial exponent equal largest Jordan block size? (yes/no)", "yes", "Block size gives exponent."},
		{"Is number of Jordan blocks for λ equal to geometric multiplicity? (yes/no)", "yes", "Counting blocks."},
	}
	pool := easy
	if scale > 3 {
		pool = append(easy, hard...)
	}
	if rand.Intn(3) == 0 {
		// production: block counts and shapes (unique answers).
		type jorEntry struct {
			question string
			answer   string
			reason   string
		}
		jorTable := []jorEntry{
			{"[[2,1],[0,2]] is one Jordan block. How many blocks? (enter a number)", "1", "Single block for eigenvalue 2."},
			{"A 3x3 diagonalizable matrix: how many Jordan blocks? (enter a number)", "3", "All blocks 1×1: three of them."},
			{"J_2(λ) is 2x2. What sits above the diagonal? (enter a number)", "1", "Ones on the superdiagonal by definition."},
		}
		v := jorTable[rand.Intn(len(jorTable))]
		return generator.Problem{Question: v.question, Answer: v.answer, Explanation: v.reason}
	}
	e := pool[rand.Intn(len(pool))]
	return generator.Problem{Question: e.question, Answer: e.answer, Explanation: e.reason}
}

// ----- subspace -----

type subspaceGen struct{}

func (g *subspaceGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	scale := int(1 + ctx.Difficulty*4)
	type entry struct {
		question string
		answer   string
		reason   string
	}
	tableEasy := []entry{
		{"Does the line y = x + 1 form a subspace of R^2? (yes/no)", "no", "It misses the origin: no subspace lacks 0."},
		{"Is {(0,0)} a subspace of R^2? (yes/no)", "yes", "The zero subspace always qualifies."},
		{"Is the union of the x-axis and y-axis a subspace of R^2? (yes/no)", "no", "(1,0)+(0,1) = (1,1) escapes the union."},
		{"Do the subspaces of R^2 consist exactly of {0}, lines through origin, R^2? (yes/no)", "yes", "One nonzero vector spans a line; two independent ones span all of R^2."},
	}
	tableHard := []entry{
		{"Two planes through the origin in R^3 meet in a line. Their sum has dimension? (enter a number)", "3", "Grassmann: 2+2-1 = 3."},
		{"A complement of a line in R^2 has dimension? (enter a number)", "1", "dim A + dim B = dim V: 1 + 1 = 2."},
		{"Is the sum of three lines in R^2 with pairwise trivial intersection direct? (yes/no)", "no", "(1,0)+(0,1)-(1,1) = 0 is a nontrivial relation."},
	}
	pool := tableEasy
	if scale > 3 {
		pool = append(tableEasy, tableHard...)
	}
	if rand.Intn(3) == 0 {
		// production: dimension counts (unique answers).
		type dimEntry struct {
			question string
			answer   string
			reason   string
		}
		dimTable := []dimEntry{
			{"Two planes through the origin in R^3 meet in a line. Their sum has dimension? (enter a number)", "3", "Grassmann: 2+2-1 = 3."},
			{"A complement of a line in R^2 has dimension? (enter a number)", "1", "dim A + dim B = dim V: 1 + 1 = 2."},
			{"The zero subspace of R^5 has dimension? (enter a number)", "0", "Only the zero vector: dimension 0."},
		}
		d := dimTable[rand.Intn(len(dimTable))]
		return generator.Problem{Question: d.question, Answer: d.answer, Explanation: d.reason}
	}
	e := pool[rand.Intn(len(pool))]
	return generator.Problem{Question: e.question, Answer: e.answer, Explanation: e.reason}
}
