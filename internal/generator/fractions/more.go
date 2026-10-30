package fractions

// The fraction concepts the domain was missing: the reciprocal, a fraction of a whole number, a
// fraction divided by a fraction, order of operations, comparison by cross-multiplication, and the
// count of how many of one fraction fit into a whole.
//
// The domain had the four operations, equivalence, mixed numbers and the word problems, but nothing
// on reciprocals (so division was a rule to memorise), nothing on the order of operations with
// fractions, and no way to compare two fractions except by converting them.
//
// Answers follow the package's convention: a reduced fraction written as `num/den`, even when the
// denominator is one, which is what the existing generators do and what the grader reads.
import (
	"fmt"
	"math/rand"
	"strconv"

	"github.com/chuma-beep/mathua/internal/generator"
)

func registerFoundations(reg *generator.Registry) {
	reg.Register("frac.reciprocal", &reciprocalGen{})
	reg.Register("frac.of_number", &ofNumberGen{})
	reg.Register("frac.complex", &complexFractionGen{})
	reg.Register("frac.order_ops", &orderOpsGen{})
	reg.Register("frac.compare.cross", &crossCompareGen{})
	reg.Register("frac.how_many", &howManyGen{})
}

func smallInt(lo, hi int) int {
	if hi < lo {
		hi = lo
	}
	return lo + rand.Intn(hi-lo+1)
}

// outFrac renders a reduced fraction the way the rest of the package does.
func outFrac(num, den int) string {
	n, d := reduce(num, den)
	return fmt.Sprintf("%d/%d", n, d)
}

// reciprocalGen asks for the reciprocal, which is the rule division by a fraction is built on.
type reciprocalGen struct{}

func (g *reciprocalGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	a := smallInt(2, 4+int(ctx.Difficulty*6))
	b := smallInt(2, 4+int(ctx.Difficulty*6))
	for a == b {
		b = smallInt(2, 4+int(ctx.Difficulty*6))
	}
	return generator.Problem{
		Question: fmt.Sprintf("What is the reciprocal of \\(\\frac{%d}{%d}\\)?", a, b),
		Answer:   outFrac(b, a),
		Explanation: fmt.Sprintf("The reciprocal turns the fraction over: %d/%d becomes %d/%d. Multiplying a fraction by its reciprocal gives 1, which is why dividing by a fraction is the same as multiplying by its reciprocal.",
			a, b, b, a),
		Facts: map[string]string{"in_num": strconv.Itoa(a), "in_den": strconv.Itoa(b),
			"out_num": strconv.Itoa(b), "out_den": strconv.Itoa(a)},
	}
}

// ofNumberGen takes a fraction of a whole number, with the denominator dividing it so the answer is
// a whole number.
type ofNumberGen struct{}

func (g *ofNumberGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	b := smallInt(2, 4+int(ctx.Difficulty*5))
	a := smallInt(1, b-1)
	n := b * smallInt(2, 5+int(ctx.Difficulty*8))
	v := a * n / b
	return generator.Problem{
		Question: fmt.Sprintf("What is \\(\\frac{%d}{%d}\\) of %d?", a, b, n),
		Answer:   outFrac(v, 1),
		Explanation: fmt.Sprintf("A fraction of a number is the fraction times the number: (%d/%d) * %d = %d/%d, which simplifies to %d. Splitting %d into %d equal parts and taking %d of them gives the same answer.",
			a, b, n, a*n, b, v, n, b, a),
		Facts: map[string]string{"in_num": strconv.Itoa(a), "in_den": strconv.Itoa(b),
			"of": strconv.Itoa(n), "out_num": strconv.Itoa(v), "out_den": "1"},
	}
}

// complexFractionGen divides a fraction by a fraction, which is what a complex fraction is.
type complexFractionGen struct{}

func (g *complexFractionGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	a := smallInt(1, 3+int(ctx.Difficulty*4))
	b := smallInt(2, 4+int(ctx.Difficulty*5))
	c := smallInt(1, 3+int(ctx.Difficulty*4))
	d := smallInt(2, 4+int(ctx.Difficulty*5))
	return generator.Problem{
		Question: fmt.Sprintf("Simplify the complex fraction \\(\\dfrac{\\frac{%d}{%d}}{\\frac{%d}{%d}}\\).", a, b, c, d),
		Answer:   outFrac(a*d, b*c),
		Explanation: fmt.Sprintf("A fraction over a fraction is the numerator fraction divided by the denominator one, and dividing by a fraction means multiplying by its reciprocal: (%d/%d) * (%d/%d) = %d/%d.",
			a, b, d, c, a*d, b*c),
		Facts: map[string]string{"out_num": strconv.Itoa(a * d), "out_den": strconv.Itoa(b * c),
			"left_num": strconv.Itoa(a), "left_den": strconv.Itoa(b),
			"right_num": strconv.Itoa(c), "right_den": strconv.Itoa(d)},
	}
}

// orderOpsGen evaluates a sum with a multiplication inside it, so the multiplication comes first.
type orderOpsGen struct{}

func (g *orderOpsGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	a := smallInt(1, 3+int(ctx.Difficulty*4))
	b := smallInt(2, 4+int(ctx.Difficulty*5))
	c := smallInt(1, 3+int(ctx.Difficulty*4))
	d := smallInt(2, 4+int(ctx.Difficulty*5))
	k := smallInt(2, 4+int(ctx.Difficulty*4))
	// a/b + (c/d)*k, computed exactly
	mn, md := c*k, d
	num := a*md + mn*b
	den := b * md
	n, dd := reduce(num, den)
	return generator.Problem{
		Question: fmt.Sprintf("Evaluate \\(\\frac{%d}{%d} + \\frac{%d}{%d} \\times %d\\), giving your answer as a fraction in lowest terms.", a, b, c, d, k),
		Answer:   fmt.Sprintf("%d/%d", n, dd),
		Explanation: fmt.Sprintf("Multiplication comes before addition, so the product is %d/%d first; then the two fractions need a common denominator and the sum is %d/%d. Doing the addition first would give a different, wrong answer.",
			c*k, d, n, dd),
		Facts: map[string]string{"out_num": strconv.Itoa(n), "out_den": strconv.Itoa(dd)},
	}
}

// crossCompareGen compares two fractions without converting them, by the sign of ad - bc.
type crossCompareGen struct{}

func (g *crossCompareGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	a := smallInt(1, 4+int(ctx.Difficulty*5))
	b := smallInt(2, 5+int(ctx.Difficulty*6))
	c := smallInt(1, 4+int(ctx.Difficulty*5))
	d := smallInt(2, 5+int(ctx.Difficulty*6))
	for a*d == b*c {
		c = smallInt(1, 4+int(ctx.Difficulty*5))
	}
	v := a*d - b*c
	return generator.Problem{
		Question: fmt.Sprintf("To compare \\(\\frac{%d}{%d}\\) with \\(\\frac{%d}{%d}\\) by cross-multiplication, evaluate \\(%d \\times %d - %d \\times %d\\). Enter the resulting integer. A positive value means the first fraction is larger.", a, b, c, d, a, d, b, c),
		Answer:   fmt.Sprintf("%d/%d", v, 1),
		Explanation: fmt.Sprintf("Cross-multiplying puts both fractions over the same denominator without writing it out: %d*%d = %d against %d*%d = %d, and the difference %d has the same sign as the difference of the fractions themselves.",
			a, d, a*d, b, c, b*c, v),
		Facts: map[string]string{"difference": strconv.Itoa(v), "left": strconv.Itoa(a * d),
			"right": strconv.Itoa(b * c), "out_num": strconv.Itoa(v), "out_den": "1"},
	}
}

// howManyGen asks how many copies of a unit fraction fit into a whole number, which is division read
// as a count.
type howManyGen struct{}

func (g *howManyGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	b := smallInt(2, 4+int(ctx.Difficulty*5))
	a := smallInt(1, b-1)
	// n is a multiple of a, so the count of a/b inside n is whole
	n := a * smallInt(1, 4+int(ctx.Difficulty*6))
	v := n * b / a
	return generator.Problem{
		Question: fmt.Sprintf("How many times does \\(\\frac{%d}{%d}\\) fit into %d?", a, b, n),
		Answer:   outFrac(v, 1),
		Explanation: fmt.Sprintf("Dividing %d by %d/%d multiplies it by the reciprocal: %d * %d / %d = %d. Reading division as a count is why the answer is a whole number whenever the fractions line up.",
			n, a, b, n, b, a, v),
		Facts: map[string]string{"in_num": strconv.Itoa(a), "in_den": strconv.Itoa(b),
			"of": strconv.Itoa(n), "out_num": strconv.Itoa(v), "out_den": "1"},
	}
}
