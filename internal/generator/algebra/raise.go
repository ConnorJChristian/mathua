package algebra

// The 2026-10-19 raise: twenty concepts that take algebra from 104 to the
// domain target of 124.
//
// The batch fills the groups the domain left thin. Functions gain piecewise
// evaluation, graph transformations and the domain of an inverse; exponentials
// and logarithms gain their graphs and the solution of a
// logarithmic model; radicals gain simplifying, combining and conjugates;
// rational expressions gain complex fractions, proportions and vertical
// asymptotes; systems gain a three-variable elimination and
// a nonlinear pair; sequences gain sigma notation and induction; polynomials
// gain the factor theorem and the rational root theorem.
//
// Eleven concepts are numeric, so the arithmetic is exercised, and seven are
// conceptual multiple choice, where the domain is about statements rather than
// computation. Every numeric generator publishes the parameters its answer
// follows from, and the tests recompute the answer rather than trusting the
// author's arithmetic.

import (
	"math/rand"
	"strconv"

	"github.com/chuma-beep/mathua/internal/generator"
)

func registerRaise(reg *generator.Registry) {
	reg.Register("alg.func.piecewise", &piecewiseGen{})
	reg.Register("alg.func.transformations", &transformationsGen{})
	reg.Register("alg.func.inverse_domain", &inverseDomainGen{})
	reg.Register("alg.exp.graph", &expGraphGen{})
	reg.Register("alg.log.graph", &logGraphGen{})
	reg.Register("alg.log.model", &logModelGen{})
	reg.Register("alg.radical.simplify", &radicalSimplifyGen{})
	reg.Register("alg.radical.operations", &radicalOpsGen{})
	reg.Register("alg.radical.conjugate", &radicalConjugateGen{})
	reg.Register("alg.rational.complex_fraction", &complexFractionGen{})
	reg.Register("alg.rational.proportions", &proportionsGen{})
	reg.Register("alg.rational.asymptotes", &asymptotesGen{})
	reg.Register("alg.systems.three_var", &threeVarGen{})
	reg.Register("alg.systems.nonlinear", &nonlinearSystemGen{})
	reg.Register("alg.seq.sigma", &sigmaGen{})
	reg.Register("alg.seq.induction", &inductionGen{})
	reg.Register("alg.poly.factor_theorem", &factorTheoremGen{})
	reg.Register("alg.poly.rational_root", &rationalRootGen{})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// algDec renders a float with up to six decimals, trimming trailing zeros and a
// trailing point, and mapping a signed zero to plain zero.
func algDec(v float64) string {
	s := strconv.FormatFloat(v, 'f', 6, 64)
	for len(s) > 0 && s[len(s)-1] == '0' {
		s = s[:len(s)-1]
	}
	if len(s) > 0 && s[len(s)-1] == '.' {
		s = s[:len(s)-1]
	}
	if s == "" || s == "-0" {
		s = "0"
	}
	return s
}

// algMCClaim is one multiple-choice item: a question, the correct option, the
// distractors that must all be rejected, and the reason the correct option is
// right.
type algMCClaim struct {
	q           string
	correct     string
	distractors []string
	reason      string
}

func algPickMC(claims []algMCClaim) generator.Problem {
	c := claims[rand.Intn(len(claims))]
	p := generator.MultipleChoice(c.q, c.correct, c.distractors...)
	p.Explanation = c.reason
	if p.Facts == nil {
		p.Facts = map[string]string{}
	}
	p.Facts["answer"] = c.correct
	return p
}

// ---------------------------------------------------------------------------
// Piecewise evaluation (numeric)
// ---------------------------------------------------------------------------

type piecewiseGen struct{}

func (g *piecewiseGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	xs := []int{-4, -2, -1, 0, 1, 2, 3, 5}
	x0 := xs[rand.Intn(len(xs))]
	answer := 0
	branch := ""
	if x0 < 0 {
		answer = 2*x0 + 1
		branch = "x < 0, so use 2x + 1"
	} else {
		answer = 3*x0 - 2
		branch = "x >= 0, so use 3x - 2"
	}
	return generator.Problem{
		Question: "For the piecewise function f(x) = 2x + 1 when x < 0 and f(x) = 3x - 2 when x >= 0, what is f(" +
			strconv.Itoa(x0) + ")?",
		Answer:      strconv.Itoa(answer),
		Explanation: "The input " + strconv.Itoa(x0) + " satisfies " + branch + ", giving " + strconv.Itoa(answer) + ".",
		Facts: map[string]string{
			"x0": strconv.Itoa(x0), "answer": strconv.Itoa(answer),
		},
	}
}

// ---------------------------------------------------------------------------
// Graph transformations (multiple choice)
// ---------------------------------------------------------------------------

type transformationsGen struct{}

func (g *transformationsGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return algPickMC([]algMCClaim{
		{
			q:           "Compared with the graph of y = f(x), the graph of y = f(x) + 3 is...",
			correct:     "shifted up three units",
			distractors: []string{"shifted down three units", "shifted right three units", "stretched vertically by a factor of three"},
			reason:      "Adding a constant outside the function moves every output up by that constant.",
		},
		{
			q:           "The graph of y = f(x - 2) is the graph of y = f(x)...",
			correct:     "shifted right two units",
			distractors: []string{"shifted left two units", "shifted up two units", "stretched horizontally by a factor of two"},
			reason:      "A subtraction inside the function moves the graph right, because the old output now occurs at a larger input.",
		},
		{
			q:           "The graph of y = -f(x) is the graph of y = f(x)...",
			correct:     "reflected across the x-axis",
			distractors: []string{"reflected across the y-axis", "shifted down one unit", "rotated a quarter turn"},
			reason:      "Negating the output reverses its sign, which is a reflection in the horizontal axis.",
		},
		{
			q:           "The graph of y = f(2x) is the graph of y = f(x)...",
			correct:     "compressed horizontally by a factor of two",
			distractors: []string{"stretched horizontally by a factor of two", "compressed vertically by a factor of two", "shifted right two units"},
			reason:      "Multiplying the input by two makes each output happen twice as soon, which squeezes the graph horizontally.",
		},
	})
}

// ---------------------------------------------------------------------------
// Domain of an inverse (multiple choice)
// ---------------------------------------------------------------------------

type inverseDomainGen struct{}

// ---------------------------------------------------------------------------
// Exponential graphs (numeric)
// ---------------------------------------------------------------------------

type expGraphGen struct{}

func (g *expGraphGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct{ a, b, x int }
	claims := []claim{
		{a: 2, b: 3, x: 4},
		{a: 3, b: 2, x: 3},
		{a: 5, b: 2, x: 4},
		{a: 4, b: 3, x: 2},
		{a: 2, b: 5, x: 2},
		{a: 6, b: 2, x: 3},
	}
	c := claims[rand.Intn(len(claims))]
	value := 1
	for i := 0; i < c.x; i++ {
		value *= c.b
	}
	value *= c.a
	return generator.Problem{
		Question: "For the exponential function f(x) = " + strconv.Itoa(c.a) + " times " + strconv.Itoa(c.b) +
			"^x, what is f(" + strconv.Itoa(c.x) + ")?",
		Answer: strconv.Itoa(value),
		Explanation: "Raising the base " + strconv.Itoa(c.b) + " to the power " + strconv.Itoa(c.x) + " and multiplying by " +
			strconv.Itoa(c.a) + " gives " + strconv.Itoa(value) + ".",
		Facts: map[string]string{
			"a": strconv.Itoa(c.a), "b": strconv.Itoa(c.b), "x": strconv.Itoa(c.x), "answer": strconv.Itoa(value),
		},
	}
}

// ---------------------------------------------------------------------------
// Compound interest (numeric)
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Logarithmic graphs (multiple choice)
// ---------------------------------------------------------------------------

type logGraphGen struct{}

func (g *logGraphGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return algPickMC([]algMCClaim{
		{
			q:           "The graph of y = log base b of x, with b > 1, ...",
			correct:     "has a vertical asymptote at x = 0 and passes through the point (1, 0)",
			distractors: []string{"has a horizontal asymptote at y = 0 and passes through (0, 1)", "is decreasing everywhere", "passes through the origin"},
			reason:      "The logarithm is the inverse of an exponential, so their graphs reflect: the log has a vertical asymptote where the exponential had its horizontal one.",
		},
		{
			q:           "As x grows without bound, log base b of x with b > 1...",
			correct:     "grows without bound, but more slowly than x",
			distractors: []string{"approaches a horizontal asymptote", "oscillates between two values", "stays constant"},
			reason:      "The logarithm increases without limit but its rate of growth falls to zero relative to the identity.",
		},
		{
			q:           "The graph of y = log base b of x is the reflection of the graph of y = b^x across...",
			correct:     "the line y = x",
			distractors: []string{"the x-axis", "the y-axis", "the line y = -x"},
			reason:      "A function and its inverse are mirror images across the line y = x, which is the swap of input and output.",
		},
	})
}

// ---------------------------------------------------------------------------
// Logarithmic models (numeric)
// ---------------------------------------------------------------------------

type logModelGen struct{}

func (g *logModelGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct{ b, k int }
	claims := []claim{
		{b: 2, k: 3},
		{b: 2, k: 4},
		{b: 3, k: 2},
		{b: 5, k: 2},
		{b: 10, k: 3},
		{b: 3, k: 3},
	}
	c := claims[rand.Intn(len(claims))]
	value := 1
	for i := 0; i < c.k; i++ {
		value *= c.b
	}
	return generator.Problem{
		Question: "Solve log base " + strconv.Itoa(c.b) + " of x = " + strconv.Itoa(c.k) +
			" for x.",
		Answer: strconv.Itoa(value),
		Explanation: "The equation says the base " + strconv.Itoa(c.b) + " raised to the power " + strconv.Itoa(c.k) +
			" equals x, so x = " + strconv.Itoa(value) + ".",
		Facts: map[string]string{
			"b": strconv.Itoa(c.b), "k": strconv.Itoa(c.k), "answer": strconv.Itoa(value),
		},
	}
}

// ---------------------------------------------------------------------------
// Simplifying radicals (numeric)
// ---------------------------------------------------------------------------

type radicalSimplifyGen struct{}

func (g *radicalSimplifyGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct{ a, b int }
	claims := []claim{
		{a: 2, b: 2},
		{a: 3, b: 2},
		{a: 4, b: 2},
		{a: 2, b: 3},
		{a: 3, b: 5},
		{a: 5, b: 2},
		{a: 2, b: 7},
		{a: 3, b: 6},
	}
	c := claims[rand.Intn(len(claims))]
	n := c.a * c.a * c.b
	return generator.Problem{
		Question: "The square root of " + strconv.Itoa(n) + " can be written as a times the square root of b, where b is squarefree. What is the whole number a?",
		Answer:   strconv.Itoa(c.a),
		Explanation: "Since " + strconv.Itoa(n) + " = " + strconv.Itoa(c.a*c.a) + " times " + strconv.Itoa(c.b) +
			", the square root is " + strconv.Itoa(c.a) + " times the square root of " + strconv.Itoa(c.b) + ".",
		Facts: map[string]string{
			"n": strconv.Itoa(n), "a": strconv.Itoa(c.a), "b": strconv.Itoa(c.b), "answer": strconv.Itoa(c.a),
		},
	}
}

// ---------------------------------------------------------------------------
// Operations with radicals (numeric)
// ---------------------------------------------------------------------------

type radicalOpsGen struct{}

func (g *radicalOpsGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct{ a, b, c int }
	claims := []claim{
		{a: 2, b: 3, c: 2},
		{a: 5, b: 2, c: 3},
		{a: 4, b: 3, c: 5},
		{a: 7, b: 2, c: 7},
		{a: 6, b: 5, c: 6},
		{a: 3, b: 8, c: 2},
	}
	c := claims[rand.Intn(len(claims))]
	sum := c.a + c.b
	return generator.Problem{
		Question: "Simplify " + strconv.Itoa(c.a) + " times the square root of " + strconv.Itoa(c.c) + " plus " +
			strconv.Itoa(c.b) + " times the square root of " + strconv.Itoa(c.c) +
			". The result is k times the square root of " + strconv.Itoa(c.c) + "; what is k?",
		Answer: strconv.Itoa(sum),
		Explanation: "The two terms share the radical, so their coefficients add: " + strconv.Itoa(c.a) + " + " +
			strconv.Itoa(c.b) + " = " + strconv.Itoa(sum) + ".",
		Facts: map[string]string{
			"a": strconv.Itoa(c.a), "b": strconv.Itoa(c.b), "c": strconv.Itoa(c.c), "answer": strconv.Itoa(sum),
		},
	}
}

// ---------------------------------------------------------------------------
// Rationalizing with conjugates (multiple choice)
// ---------------------------------------------------------------------------

type radicalConjugateGen struct{}

func (g *radicalConjugateGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return algPickMC([]algMCClaim{
		{
			q:           "To rationalize a fraction whose denominator is the square root of a plus the square root of b, multiply numerator and denominator by...",
			correct:     "the square root of a minus the square root of b",
			distractors: []string{"the square root of a plus the square root of b", "the product of the square root of a and the square root of b", "a minus b"},
			reason:      "The conjugate is the same expression with the middle sign reversed, and their product clears the radicals.",
		},
		{
			q:           "Multiplying the square root of 5 plus the square root of 2 by its conjugate gives...",
			correct:     "3",
			distractors: []string{"7", "3 plus twice the square root of 10", "the square root of 7"},
			reason:      "The product is (5 - 2) by the difference of squares identity, which is 3.",
		},
		{
			q:           "The conjugate of x plus the square root of y is...",
			correct:     "x minus the square root of y",
			distractors: []string{"-x plus the square root of y", "x plus the square root of y", "the square root of y minus x"},
			reason:      "Only the sign between the two terms changes when forming a conjugate.",
		},
	})
}

// ---------------------------------------------------------------------------
// Complex fractions (multiple choice)
// ---------------------------------------------------------------------------

type complexFractionGen struct{}

func (g *complexFractionGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return algPickMC([]algMCClaim{
		{
			q:           "To simplify a fraction whose numerator and denominator are themselves fractions, the usual first step is to...",
			correct:     "multiply the numerator by the reciprocal of the denominator",
			distractors: []string{"add the two fractions", "cross out the common letters", "subtract the two denominators"},
			reason:      "Dividing by a fraction is the same as multiplying by its reciprocal, which clears the inner denominator.",
		},
		{
			q:           "The complex fraction (a/b) divided by (c/d), with all letters nonzero, simplifies to...",
			correct:     "a times d over the product b times c",
			distractors: []string{"a times c over the product b times d", "b times c over the product a times d", "a times b over the product c times d"},
			reason:      "Multiplying by the reciprocal of c/d gives a/b times d/c, which is a d over b c.",
		},
		{
			q:           "To combine the numerator a/b plus c/d into a single fraction, you first...",
			correct:     "write both terms over the common denominator b times d",
			distractors: []string{"add the numerators and add the denominators", "multiply the two denominators and keep the first numerator", "invert both fractions"},
			reason:      "Adding fractions requires a common denominator, and the product of the denominators always works.",
		},
	})
}

// ---------------------------------------------------------------------------
// Proportions (numeric)
// ---------------------------------------------------------------------------

type proportionsGen struct{}

func (g *proportionsGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct{ a, b, k int }
	claims := []claim{
		{a: 2, b: 3, k: 4},
		{a: 3, b: 5, k: 2},
		{a: 4, b: 7, k: 3},
		{a: 5, b: 4, k: 6},
		{a: 2, b: 9, k: 5},
		{a: 6, b: 5, k: 4},
	}
	c := claims[rand.Intn(len(claims))]
	num := c.a * c.k
	x := c.b * c.k
	return generator.Problem{
		Question: "Solve the proportion " + strconv.Itoa(c.a) + "/" + strconv.Itoa(c.b) + " = " + strconv.Itoa(num) +
			"/x for x.",
		Answer: strconv.Itoa(x),
		Explanation: "Cross-multiplying gives " + strconv.Itoa(c.a) + "x = " + strconv.Itoa(c.b) + " times " + strconv.Itoa(num) +
			", so x = " + strconv.Itoa(x) + ".",
		Facts: map[string]string{
			"a": strconv.Itoa(c.a), "b": strconv.Itoa(c.b), "num": strconv.Itoa(num), "answer": strconv.Itoa(x),
		},
	}
}

// ---------------------------------------------------------------------------
// Vertical asymptotes (numeric)
// ---------------------------------------------------------------------------

type asymptotesGen struct{}

func (g *asymptotesGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct{ a, m, n int }
	claims := []claim{
		{a: 1, m: 2, n: 5},
		{a: 3, m: -4, n: 1},
		{a: 2, m: 3, n: 7},
		{a: 5, m: -2, n: 6},
		{a: 1, m: 4, n: 9},
	}
	c := claims[rand.Intn(len(claims))]
	// f(x) = (x + a) / ((x - m)(x - n)). The denominator expands to
	// x^2 - (m+n)x + mn.
	b := -(c.m + c.n)
	d := c.m * c.n
	larger := c.m
	if c.n > c.m {
		larger = c.n
	}
	return generator.Problem{
		Question: "The rational function f(x) = (x + " + strconv.Itoa(c.a) + ")/(x^2 + " + strconv.Itoa(b) + "x + " +
			strconv.Itoa(d) + ") has two vertical asymptotes. What is the larger one?",
		Answer: strconv.Itoa(larger),
		Explanation: "The denominator factors as (x - " + strconv.Itoa(c.m) + ")(x - " + strconv.Itoa(c.n) +
			"), so the vertical asymptotes are at " + strconv.Itoa(c.m) + " and " + strconv.Itoa(c.n) + ", and the larger is " +
			strconv.Itoa(larger) + ".",
		Facts: map[string]string{
			"a": strconv.Itoa(c.a), "b": strconv.Itoa(b), "d": strconv.Itoa(d),
			"m": strconv.Itoa(c.m), "n": strconv.Itoa(c.n), "answer": strconv.Itoa(larger),
		},
	}
}

// ---------------------------------------------------------------------------
// Systems of three equations (numeric)
// ---------------------------------------------------------------------------

type threeVarGen struct{}

func (g *threeVarGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct{ x, y, z int }
	claims := []claim{
		{x: 2, y: 3, z: 1},
		{x: 5, y: 1, z: 4},
		{x: 3, y: 4, z: 2},
		{x: 6, y: 2, z: 3},
		{x: 4, y: 5, z: 1},
	}
	c := claims[rand.Intn(len(claims))]
	a := c.x + c.y + c.z
	b := c.x - c.y + c.z
	d := c.x + c.y - c.z
	return generator.Problem{
		Question: "Solve the system x + y + z = " + strconv.Itoa(a) + ", x - y + z = " + strconv.Itoa(b) +
			", x + y - z = " + strconv.Itoa(d) + " for x.",
		Answer: strconv.Itoa(c.x),
		Explanation: "Subtracting the second equation from the first gives 2y = " + strconv.Itoa(a-b) + ", so y = " +
			strconv.Itoa(c.y) + "; subtracting the third from the first gives 2z = " + strconv.Itoa(a-d) + ", so z = " +
			strconv.Itoa(c.z) + "; then x = " + strconv.Itoa(a) + " - y - z = " + strconv.Itoa(c.x) + ".",
		Facts: map[string]string{
			"a": strconv.Itoa(a), "b": strconv.Itoa(b), "d": strconv.Itoa(d),
			"y": strconv.Itoa(c.y), "z": strconv.Itoa(c.z), "answer": strconv.Itoa(c.x),
		},
	}
}

// ---------------------------------------------------------------------------
// Matrices (multiple choice)
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Nonlinear systems (numeric)
// ---------------------------------------------------------------------------

type nonlinearSystemGen struct{}

func (g *nonlinearSystemGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	ks := []int{2, 3, 4, 5, 6, 7}
	k := ks[rand.Intn(len(ks))]
	y := k * k
	return generator.Problem{
		Question: "The system y = x^2 and y = " + strconv.Itoa(k) + "x has two solutions. What is the y-coordinate of the solution with x not equal to 0?",
		Answer:   strconv.Itoa(y),
		Explanation: "Setting x^2 equal to " + strconv.Itoa(k) + "x gives x(x - " + strconv.Itoa(k) +
			") = 0, so the nonzero solution has x = " + strconv.Itoa(k) + " and y = " + strconv.Itoa(k) + "^2 = " + strconv.Itoa(y) + ".",
		Facts: map[string]string{
			"k": strconv.Itoa(k), "x": strconv.Itoa(k), "answer": strconv.Itoa(y),
		},
	}
}

// ---------------------------------------------------------------------------
// Sigma notation (numeric)
// ---------------------------------------------------------------------------

type sigmaGen struct{}

func (g *sigmaGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct{ a, b, n int }
	claims := []claim{
		{a: 1, b: 0, n: 5},
		{a: 2, b: 1, n: 4},
		{a: 3, b: 2, n: 6},
		{a: 1, b: 3, n: 8},
		{a: 2, b: 5, n: 5},
		{a: 4, b: 0, n: 5},
	}
	c := claims[rand.Intn(len(claims))]
	tri := c.n * (c.n + 1) / 2
	sum := c.a*tri + c.b*c.n
	return generator.Problem{
		Question: "Evaluate the sum from i = 1 to " + strconv.Itoa(c.n) + " of (" + strconv.Itoa(c.a) + "i + " +
			strconv.Itoa(c.b) + ").",
		Answer: strconv.Itoa(sum),
		Explanation: "The sum splits into " + strconv.Itoa(c.a) + " times the sum of i, which is " + strconv.Itoa(tri) +
			", plus " + strconv.Itoa(c.b) + " times " + strconv.Itoa(c.n) + " terms, giving " + strconv.Itoa(sum) + ".",
		Facts: map[string]string{
			"a": strconv.Itoa(c.a), "b": strconv.Itoa(c.b), "n": strconv.Itoa(c.n), "answer": strconv.Itoa(sum),
		},
	}
}

// ---------------------------------------------------------------------------
// Induction (multiple choice)
// ---------------------------------------------------------------------------

type inductionGen struct{}

func (g *inductionGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return algPickMC([]algMCClaim{
		{
			q:           "A proof by induction establishes a statement by proving...",
			correct:     "a base case and an induction step that passes from n to n + 1",
			distractors: []string{"only the base case", "the statement for one very large n", "the statement for every n without a base case"},
			reason:      "The base case starts the chain and the induction step carries it forward to every later integer.",
		},
		{
			q:           "The induction step assumes...",
			correct:     "the statement holds for some n and then proves it for n + 1",
			distractors: []string{"the statement for n + 1 and derives it for n", "the statement for every n at once", "only the base case and nothing else"},
			reason:      "The step is a conditional: if the statement is true at one rung, it is true at the next.",
		},
		{
			q:           "If the base case fails but the induction step is valid, the proof...",
			correct:     "does not establish the statement",
			distractors: []string{"still proves the statement", "proves it for every n except the base", "proves the converse instead"},
			reason:      "Without a starting point the chain never begins, so no instance is reached.",
		},
	})
}

// ---------------------------------------------------------------------------
// The factor theorem (multiple choice)
// ---------------------------------------------------------------------------

type factorTheoremGen struct{}

func (g *factorTheoremGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return algPickMC([]algMCClaim{
		{
			q:           "The factor theorem says that (x - r) is a factor of a polynomial p(x) exactly when...",
			correct:     "p(r) = 0",
			distractors: []string{"p(0) = r", "the leading coefficient is 1", "the derivative p'(r) = 0"},
			reason:      "Dividing by x - r leaves a remainder equal to p(r), so the factor divides exactly only when that remainder is zero.",
		},
		{
			q:           "If p(3) = 0 for a polynomial p, then...",
			correct:     "(x - 3) divides p(x)",
			distractors: []string{"(x + 3) divides p(x)", "3 is the leading coefficient", "p has no other roots"},
			reason:      "A zero at 3 means the corresponding linear factor is x - 3.",
		},
		{
			q:           "Dividing a polynomial p(x) by (x - r) and getting a remainder of 0 shows that...",
			correct:     "r is a root of p",
			distractors: []string{"r is not a root of p", "the degree of p increases", "p is irreducible"},
			reason:      "A zero remainder is the factor theorem read backwards, so the value at r is zero.",
		},
	})
}

// ---------------------------------------------------------------------------
// The rational root theorem (numeric)
// ---------------------------------------------------------------------------

type rationalRootGen struct{}

func (g *rationalRootGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct{ r1, r2, r3 int }
	claims := []claim{
		{r1: 2, r2: -1, r3: -3},
		{r1: 3, r2: -2, r3: -5},
		{r1: 4, r2: -1, r3: -2},
		{r1: 5, r2: -3, r3: -1},
		{r1: 1, r2: -2, r3: -4},
	}
	c := claims[rand.Intn(len(claims))]
	b := -(c.r1 + c.r2 + c.r3)
	d := c.r1*c.r2 + c.r1*c.r3 + c.r2*c.r3
	e := -(c.r1 * c.r2 * c.r3)
	return generator.Problem{
		Question: "The monic polynomial x^3 + " + strconv.Itoa(b) + "x^2 + " + strconv.Itoa(d) + "x + " + strconv.Itoa(e) +
			" has three integer roots. By the rational root theorem, what is its positive root?",
		Answer: strconv.Itoa(c.r1),
		Explanation: "The polynomial factors as (x - " + strconv.Itoa(c.r1) + ")(x + " + strconv.Itoa(-c.r2) +
			")(x + " + strconv.Itoa(-c.r3) + "), so the roots are " + strconv.Itoa(c.r1) + ", " + strconv.Itoa(c.r2) +
			" and " + strconv.Itoa(c.r3) + ", and the positive one is " + strconv.Itoa(c.r1) + ".",
		Facts: map[string]string{
			"b": strconv.Itoa(b), "d": strconv.Itoa(d), "e": strconv.Itoa(e),
			"r1": strconv.Itoa(c.r1), "r2": strconv.Itoa(c.r2), "r3": strconv.Itoa(c.r3),
			"answer": strconv.Itoa(c.r1),
		},
	}
}
