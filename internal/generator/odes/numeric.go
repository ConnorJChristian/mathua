package odes

// The numeric differential-equations concepts the domain was missing: an initial value problem
// evaluated at a point, the sum of a characteristic equation's roots, a non-zero equilibrium, the
// Wronskian of two powers, the number of Euler steps between two points, the Laplace transform of a
// power evaluated at a point, and the discriminant that classifies a second-order PDE.
//
// The domain had twenty-five concepts and twenty-four of them `multiple_choice`, which is the most
// recognition-shaped surface in the corpus and the one section 4.2 of docs/content-expansion.md
// measures. These seven are computed.
//
// Every answer is an integer by construction: the forcing is chosen so the antiderivative is whole,
// the characteristic sum is a ratio that divides, the equilibrium is the root the form makes visible,
// the Wronskian is evaluated at one, the step size divides the interval, the transform is evaluated
// at one, and the discriminant is a difference of products.
import (
	"fmt"
	"math/rand"

	"github.com/chuma-beep/mathua/internal/generator"
)

func registerNumeric(reg *generator.Registry) {
	reg.Register("ode.ivp.evaluate", &ivpEvaluateGen{})
	reg.Register("ode.characteristic.sum", &characteristicSumGen{})
	reg.Register("ode.equilibrium.nonzero", &equilibriumGen{})
	reg.Register("ode.wronskian", &wronskianGen{})
	reg.Register("ode.numerical.steps", &eulerStepsGen{})
	reg.Register("ode.laplace.value", &laplaceValueGen{})
	reg.Register("ode.pde.discriminant", &pdeDiscriminantGen{})
}

func smallInt(lo, hi int) int {
	if hi < lo {
		hi = lo
	}
	return lo + rand.Intn(hi-lo+1)
}

// ivpEvaluateGen integrates a linear forcing and evaluates the solution at a point.
type ivpEvaluateGen struct{}

func (g *ivpEvaluateGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	a := 2 * (1 + rand.Intn(2+int(ctx.Difficulty*3))) // even, so a t^2 / 2 is whole
	b := smallInt(-5, 5)
	c := smallInt(-5, 5+int(ctx.Difficulty*6))
	t := smallInt(1, 3+int(ctx.Difficulty*4))
	// y = a x^2/2 + b x + c
	v := a*t*t/2 + b*t + c
	return generator.Problem{
		Question: fmt.Sprintf("Solve \\(y' = %dx + %d\\) with \\(y(0) = %d\\), and find \\(y(%d)\\).", a, b, c, t),
		Answer:   fmt.Sprintf("%d", v),
		Explanation: fmt.Sprintf("Integrate the right side: y = %dx^2/2 + %dx + C, and the initial condition makes C = %d. Substituting x = %d gives %d. The constant is the only unknown, which is why one initial condition fixes one solution.",
			a, b, c, t, v),
		Facts: map[string]string{"value": fmt.Sprintf("%d", v), "constant": fmt.Sprintf("%d", c)},
	}
}

// characteristicSumGen uses the fact that a monic quadratic's roots sum to the negation of its linear
// coefficient.
type characteristicSumGen struct{}

func (g *characteristicSumGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	a := smallInt(1, 3+int(ctx.Difficulty*3))
	// a y'' + b y' + c y = 0, with a dividing b so the root sum -b/a is whole
	b := a * smallInt(-4, 4)
	c := smallInt(1, 9)
	sum := -b / a
	return generator.Problem{
		Question: fmt.Sprintf("For the equation \\(%dy'' %+dy' %+dy = 0\\), what is the sum of the roots of the characteristic equation?", a, b, c),
		Answer:   fmt.Sprintf("%d", sum),
		Explanation: fmt.Sprintf("The characteristic equation is %dr^2 + %dr + %d = 0. Its roots sum to the negation of the linear coefficient over the leading one: %d/%d = %d. That is the same coefficient reading that makes the characteristic equation the reason a constant-coefficient equation solves by exponentials.",
			a, b, c, -b, a, sum),
		Facts: map[string]string{"sum": fmt.Sprintf("%d", sum)},
	}
}

// equilibriumGen asks for the non-zero equilibrium of the logistic form, where the roots are visible.
type equilibriumGen struct{}

func (g *equilibriumGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	k := smallInt(1, 4+int(ctx.Difficulty*4))
	a := smallInt(1, 8+int(ctx.Difficulty*8))
	return generator.Problem{
		Question: fmt.Sprintf("The population model \\(y' = %dy(%d - y)\\) has two equilibria. What is the non-zero one?", k, a),
		Answer:   fmt.Sprintf("%d", a),
		Explanation: fmt.Sprintf("An equilibrium is where the rate is zero, and the product %dy(%d - y) is zero at y = 0 and at y = %d. The first is extinction and the second is the carrying capacity, which is why the non-zero root is the interesting one.",
			k, a, a),
		Facts: map[string]string{"equilibrium": fmt.Sprintf("%d", a)},
	}
}

// wronskianGen uses two powers, whose Wronskian is a single term.
type wronskianGen struct{}

func (g *wronskianGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	m := smallInt(0, 3)
	n := m + 1 + rand.Intn(3+int(ctx.Difficulty*3))
	x := 1 + rand.Intn(2+int(ctx.Difficulty*2))
	// W(x^m, x^n) = (n - m) x^(m+n-1)
	v := (n - m) * powInt(x, m+n-1)
	return generator.Problem{
		Question: fmt.Sprintf("Find the Wronskian \\(W(x^{%d}, x^{%d})\\) evaluated at \\(x = %d\\).", m, n, x),
		Answer:   fmt.Sprintf("%d", v),
		Explanation: fmt.Sprintf("The Wronskian is f g' - f' g, which for two powers is (%d - %d) x^(%d+%d-1). At x = %d that is %d. A non-zero Wronskian is what makes the two functions independent, so this number answers a question about independence as well as being a determinant.",
			m, n, m, n, x, v),
		Facts: map[string]string{"wronskian": fmt.Sprintf("%d", v)},
	}
}

// eulerStepsGen asks how many steps a step size needs to cross an interval.
type eulerStepsGen struct{}

func (g *eulerStepsGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	h := smallInt(1, 2+int(ctx.Difficulty*2))
	steps := 2 + rand.Intn(6+int(ctx.Difficulty*8))
	a := smallInt(0, 4)
	b := a + h*steps
	return generator.Problem{
		Question: fmt.Sprintf("Euler's method runs from \\(x = %d\\) to \\(x = %d\\) with step size \\(h = %d\\). How many steps does it take?", a, b, h),
		Answer:   fmt.Sprintf("%d", steps),
		Explanation: fmt.Sprintf("The number of steps is the width of the interval divided by the step size: (%d - %d)/%d = %d. The count is exact because the step size divides the interval, which is why an interval is normally chosen to suit the step rather than the other way round.",
			b, a, h, steps),
		Facts: map[string]string{"steps": fmt.Sprintf("%d", steps)},
	}
}

// laplaceValueGen uses L{t^n}(s) = n!/s^(n+1), evaluated where the answer is whole.
type laplaceValueGen struct{}

func (g *laplaceValueGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	n := smallInt(1, 5)
	f := 1
	for i := 2; i <= n; i++ {
		f *= i
	}
	return generator.Problem{
		Question: fmt.Sprintf("Evaluate the Laplace transform \\(L\\{t^{%d}\\}(s)\\) at \\(s = 1\\).", n),
		Answer:   fmt.Sprintf("%d", f),
		Explanation: fmt.Sprintf("The transform of t^n is n! over s^(n+1), so at s = 1 it is simply %d! = %d. The factorial is the shape of the transform for a power, and the extra power of s in the denominator is what makes the transform decay faster for larger n.",
			n, f),
		Facts: map[string]string{"value": fmt.Sprintf("%d", f)},
	}
}

// pdeDiscriminantGen computes B^2 - 4AC, the sign of which classifies a second-order PDE.
type pdeDiscriminantGen struct{}

func (g *pdeDiscriminantGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	a := smallInt(1, 3+int(ctx.Difficulty*5))
	b := smallInt(0, 4+int(ctx.Difficulty*5))
	c := smallInt(1, 3+int(ctx.Difficulty*5))
	d := b*b - 4*a*c
	return generator.Problem{
		Question: fmt.Sprintf("For the equation \\(%du_{xx} %+du_{xy} %+du_{yy} = 0\\), find the discriminant \\(B^2 - 4AC\\).", a, b, c),
		Answer:   fmt.Sprintf("%d", d),
		Explanation: fmt.Sprintf("The discriminant is B^2 - 4AC = %d^2 - 4(%d)(%d) = %d. Its sign says which of the three classical types the equation is, and that sign is what the analogues of the wave, heat and Laplace equations differ in.",
			b, a, c, d),
		Facts: map[string]string{"discriminant": fmt.Sprintf("%d", d)},
	}
}

func powInt(b, e int) int {
	r := 1
	for i := 0; i < e; i++ {
		r *= b
	}
	return r
}
