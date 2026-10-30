package calculus

// The numeric calculus concepts the domain was still missing: the value of a convergent geometric
// series, the limit of a rational sequence, a recursive sequence's term, the mass of a lamina, a
// definite integral by substitution, the mass from a variable density, a higher-order derivative at a
// point, and a directional derivative in a given direction.
//
// The domain's series, sequences and multivariable concepts are almost all `symbolic`, and the
// corpus records what that means in production: the grading runtime has no SymPy, so a symbolic
// answer always grades incorrect. These eight are `numeric` instead.
//
// Every answer is an integer by construction: the geometric ratio is chosen so the sum is whole, the
// sequence's leading coefficients divide, the substitution integrand is an exact derivative, the
// derivative is a falling factorial times a whole power, and the direction is given unnormalised.
import (
	"fmt"
	"math/rand"

	"github.com/chuma-beep/mathua/internal/generator"
)

func registerNumeric(reg *generator.Registry) {
	reg.Register("calc.seq.limit_value", &sequenceLimitGen{})
	reg.Register("calc.seq.recursive_value", &recursiveValueGen{})
	reg.Register("calc.seq.arithmetic_sum", &arithmeticSumGen{})
	reg.Register("calc.multi.mass_lamina", &massLaminaGen{})
	reg.Register("calc.integral.substitution_value", &substitutionValueGen{})
	reg.Register("calc.integral.mass_density", &massDensityGen{})
	reg.Register("calc.deriv.higher_order_value", &higherOrderValueGen{})
}

// sequenceLimitGen compares the leading coefficients of a rational sequence.
type sequenceLimitGen struct{}

func (g *sequenceLimitGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	ratio := 1 + rand.Intn(3+int(ctx.Difficulty*3))
	d := 1 + rand.Intn(2+int(ctx.Difficulty*3))
	a := ratio * d
	b, c := rand.Intn(9), rand.Intn(9)
	e, f := rand.Intn(9), rand.Intn(9)
	return generator.Problem{
		Question: fmt.Sprintf("Find \\(\\lim_{n \\to \\infty} \\frac{%dn^2 + %dn + %d}{%dn^2 + %dn + %d}\\).", a, b, c, d, e, f),
		Answer:   fmt.Sprintf("%d", ratio),
		Explanation: fmt.Sprintf("For a ratio of polynomials at infinity only the highest powers matter, so the limit is the ratio of the leading coefficients: %d/%d = %d. The lower terms are negligible because the squared terms grow fastest.",
			a, d, ratio),
		Facts: map[string]string{"limit": fmt.Sprintf("%d", ratio)},
	}
}

// recursiveValueGen iterates an affine recurrence a fixed number of steps.
type recursiveValueGen struct{}

func (g *recursiveValueGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	start := 1 + rand.Intn(6+int(ctx.Difficulty*6))
	r := 2 + rand.Intn(2+int(ctx.Difficulty*2))
	c := 1 + rand.Intn(5+int(ctx.Difficulty*5))
	steps := 2 + rand.Intn(3+int(ctx.Difficulty*3))
	v := start
	for i := 0; i < steps; i++ {
		v = r*v + c
	}
	return generator.Problem{
		Question: fmt.Sprintf("A sequence is defined by \\(a_1 = %d\\) and \\(a_{n+1} = %da_n + %d\\). Find \\(a_{%d}\\).", start, r, c, steps+1),
		Answer:   fmt.Sprintf("%d", v),
		Explanation: fmt.Sprintf("Apply the recurrence %d times: each term is %d times the one before plus %d, which gives %d. A recursive sequence is defined by what each term does to its predecessor, so the only way to reach a term is through the ones before it.",
			steps, r, c, v),
		Facts: map[string]string{"value": fmt.Sprintf("%d", v)},
	}
}

// massLaminaGen multiplies a constant density by an area.
type massLaminaGen struct{}

func (g *massLaminaGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	rho := 1 + rand.Intn(4+int(ctx.Difficulty*6))
	w := 2 + rand.Intn(5+int(ctx.Difficulty*6))
	h := 2 + rand.Intn(5+int(ctx.Difficulty*6))
	return generator.Problem{
		Question: fmt.Sprintf("A rectangular lamina is %d units by %d units and has uniform density %d per unit area. Find its mass.", w, h, rho),
		Answer:   fmt.Sprintf("%d", rho*w*h),
		Explanation: fmt.Sprintf("With a uniform density the mass is the density times the area: %d * (%d * %d) = %d. When the density varies, the same idea becomes an integral of the density over the region, which is what the double integral computes.",
			rho, w, h, rho*w*h),
		Facts: map[string]string{"mass": fmt.Sprintf("%d", rho*w*h)},
	}
}

// substitutionValueGen integrates an exact derivative over a symmetric interval.
type substitutionValueGen struct{}

func (g *substitutionValueGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	a := 2 * (1 + rand.Intn(3+int(ctx.Difficulty*3))) // even, so the endpoint substitution is whole
	// integral from 0 to a of 2x (x^2 + 1) dx = ((a^2+1)^2 - 1)/2
	v := ((a*a+1)*(a*a+1) - 1) / 2
	return generator.Problem{
		Question: fmt.Sprintf("Evaluate \\(\\int_0^{%d} 2x(x^2 + 1)\\,dx\\).", a),
		Answer:   fmt.Sprintf("%d", v),
		Explanation: fmt.Sprintf("Let u = x^2 + 1, so du = 2x dx, which is exactly what multiplies the bracket: the integral becomes the integral of u du from 1 to %d. That evaluates to (u^2)/2 between those limits, giving %d. Substitution is the chain rule read backwards, which is why the 2x has to be present for it to work.",
			a*a+1, v),
		Facts: map[string]string{"value": fmt.Sprintf("%d", v)},
	}
}

// massDensityGen integrates a linear density along a rod.
type massDensityGen struct{}

func (g *massDensityGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	k := 2 * (1 + rand.Intn(3+int(ctx.Difficulty*4))) // even, so kL^2/2 is whole for any L
	l := 1 + rand.Intn(5+int(ctx.Difficulty*6))
	v := k * l * l / 2
	return generator.Problem{
		Question: fmt.Sprintf("A rod of length %d has linear density \\(\\rho(x) = %dx\\) at a distance \\(x\\) from one end. Find its mass.", l, k),
		Answer:   fmt.Sprintf("%d", v),
		Explanation: fmt.Sprintf("Mass is the integral of density over the length: the integral of %dx from 0 to %d is %d*%d^2/2 = %d. A varying density is why the mass is an integral rather than a multiplication, even though the units are the same.",
			k, l, k, l, v),
		Facts: map[string]string{"mass": fmt.Sprintf("%d", v)},
	}
}

// higherOrderValueGen evaluates a falling factorial times a whole power.
type higherOrderValueGen struct{}

func (g *higherOrderValueGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	n := 2 + rand.Intn(3+int(ctx.Difficulty*3))
	k := 1 + rand.Intn(n-1)
	a := 1 + rand.Intn(2+int(ctx.Difficulty*2))
	// n!/(n-k)! * a^(n-k)
	coef := 1
	for i := 0; i < k; i++ {
		coef *= n - i
	}
	pw := 1
	for i := 0; i < n-k; i++ {
		pw *= a
	}
	v := coef * pw
	return generator.Problem{
		Question: fmt.Sprintf("For \\(f(x) = x^{%d}\\), find \\(f^{(%d)}(%d)\\).", n, k, a),
		Answer:   fmt.Sprintf("%d", v),
		Explanation: fmt.Sprintf("Each derivative brings down the current power and reduces it: the k-th derivative of x^n is n!/(n-k)! times x^(n-k), so here it is %d times %d^%d = %d. The power rule applied %d times is the whole computation.",
			coef, a, n-k, v, k),
		Facts: map[string]string{"value": fmt.Sprintf("%d", v)},
	}
}

// arithmeticSumGen adds an arithmetic sequence, with an even number of terms so the half is whole.
type arithmeticSumGen struct{}

func (g *arithmeticSumGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	a := -3 + rand.Intn(8+int(ctx.Difficulty*6))      // the first term
	d := 1 + rand.Intn(4+int(ctx.Difficulty*5))       // the common difference
	n := 2 * (1 + rand.Intn(3+int(ctx.Difficulty*5))) // an even number of terms
	sum := n * (2*a + (n-1)*d) / 2
	return generator.Problem{
		Question: fmt.Sprintf("Find the sum of the arithmetic sequence with first term %d, common difference %d, and %d terms.", a, d, n),
		Answer:   fmt.Sprintf("%d", sum),
		Explanation: fmt.Sprintf("An arithmetic sequence sums by pairing the first term with the last: the last term is %d, the average of the pair is (%d + %d)/2 = %d, and %d terms at that average give %d. The pairing is Gauss's method, and it needs the number of terms for the average to be well defined.",
			a+(n-1)*d, a, a+(n-1)*d, (2*a+(n-1)*d)/2, n, sum),
		Facts: map[string]string{"sum": fmt.Sprintf("%d", sum), "terms": fmt.Sprintf("%d", n)},
	}
}
