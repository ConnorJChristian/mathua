package calculus

// The calculus concepts still missing after the multivariable and vector blocks: the average value
// of a function, volumes from known cross-sections, changing the order of integration, the tangent
// plane as a linear approximation, extrema on a closed region, and the two polar-coordinate
// computations.
//
// Every answer is an integer by construction: the average value uses a length that cancels, the
// cross-section volume uses a side whose cube is divisible by three, the double integrals use even
// widths, and the polar slope is evaluated at angles whose cotangent is whole.
import (
	"fmt"
	"math/rand"

	"github.com/chuma-beep/mathua/internal/generator"
)

func registerRemaining(reg *generator.Registry) {
	reg.Register("calc.integral.average_value", &averageValueGen{})
	reg.Register("calc.integral.cross_section_volume", &crossSectionGen{})
	reg.Register("calc.multi.change_order", &changeOrderGen{})
	reg.Register("calc.multi.tangent_plane", &tangentPlaneGen{})
	reg.Register("calc.multi.closed_extrema", &closedExtremaGen{})
	reg.Register("calc.polar.area", &polarAreaGen{})
	reg.Register("calc.polar.slope", &polarSlopeGen{})
}

// averageValueGen divides the integral by the interval's length, with an even coefficient so the
// result is whole.
type averageValueGen struct{}

func (g *averageValueGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	b := 2 * (1 + rand.Intn(3+int(ctx.Difficulty*4))) // the interval [0, b]
	slope := 2 * (1 + rand.Intn(3+int(ctx.Difficulty*4)))
	intercept := 1 + rand.Intn(5+int(ctx.Difficulty*6))
	avg := slope*b/2 + intercept
	return generator.Problem{
		Question: fmt.Sprintf("Find the average value of \\(f(x) = %dx + %d\\) on the interval \\([0, %d]\\).", slope, intercept, b),
		Answer:   fmt.Sprintf("%d", avg),
		Explanation: fmt.Sprintf("The average value is the integral divided by the length of the interval: (1/%d) times the area under the line, which is %d. That gives %d.",
			b, b*(slope*b/2+intercept), avg),
		Facts: map[string]string{"average": fmt.Sprintf("%d", avg), "width": fmt.Sprintf("%d", b)},
	}
}

// crossSectionGen integrates the area of a known cross-section, with a side that makes the cube
// divisible by three.
type crossSectionGen struct{}

func (g *crossSectionGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	n := 3 * (1 + rand.Intn(3+int(ctx.Difficulty*3))) // the side is x, over [0, n]
	v := n * n * n / 3
	return generator.Problem{
		Question: fmt.Sprintf("A solid has, for each \\(x\\) between 0 and %d, a square cross-section perpendicular to the x-axis with side \\(s(x) = x\\). What is the volume of the solid?", n),
		Answer:   fmt.Sprintf("%d", v),
		Explanation: fmt.Sprintf("Each cross-section has area s(x)^2 = x^2, and the volume is the integral of that area along the axis: the integral of x^2 from 0 to %d is %d^3/3 = %d.",
			n, n, v),
		Facts: map[string]string{"volume": fmt.Sprintf("%d", v)},
	}
}

// changeOrderGen integrates over a triangle, where swapping the order is the point of the exercise.
type changeOrderGen struct{}

func (g *changeOrderGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	n := 2 * (1 + rand.Intn(3+int(ctx.Difficulty*4)))
	v := n * n * n / 2
	return generator.Problem{
		Question: fmt.Sprintf("Evaluate \\(\\iint_R (x + y)\\,dA\\) where \\(R\\) is the triangle \\(0 \\le y \\le x \\le %d\\).", n),
		Answer:   fmt.Sprintf("%d", v),
		Explanation: fmt.Sprintf("Integrating in y first runs from 0 to x, giving %dx^2/2; integrating that in x from 0 to %d gives %d^3/2 = %d. The other order gives the same value, which is Fubini's theorem, but needs the region written as type II limits.",
			n, n, n, v),
		Facts: map[string]string{"value": fmt.Sprintf("%d", v)},
	}
}

// tangentPlaneGen evaluates the linear approximation, which is what a tangent plane is for.
type tangentPlaneGen struct{}

func (g *tangentPlaneGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	a := 1 + rand.Intn(3+int(ctx.Difficulty*3))
	b := 1 + rand.Intn(3+int(ctx.Difficulty*3))
	p := 1 + rand.Intn(3+int(ctx.Difficulty*3))
	q := 1 + rand.Intn(3+int(ctx.Difficulty*3))
	// f(x,y) = a x^2 + b y^2, its tangent plane at (p,q), evaluated at (p+1, q+1)
	f := a*p*p + b*q*q
	v := f + 2*a*p + 2*b*q
	// The true value is a(p+1)^2 + b(q+1)^2 = v + a + b: the plane leaves out the
	// second-order terms a·1^2 + b·1^2, so it always underestimates this bowl.
	exact := a*(p+1)*(p+1) + b*(q+1)*(q+1)
	return generator.Problem{
		Question: fmt.Sprintf("For \\(f(x,y) = %dx^2 + %dy^2\\), use the tangent plane at \\((%d,%d)\\) to estimate \\(f(%d,%d)\\).", a, b, p, q, p+1, q+1),
		Answer:   fmt.Sprintf("%d", v),
		Explanation: fmt.Sprintf("At (%d,%d) the value is %d, and the partial derivatives are %d and %d, so a step of one in each direction adds those: %d + %d + %d = %d. That is the tangent-plane estimate. The true value is f(%d,%d) = %d, so the estimate is %d too low: the plane drops the second-order terms %d(1)^2 + %d(1)^2, and a bowl-shaped function always lies above its tangent plane.",
			p, q, f, 2*a*p, 2*b*q, f, 2*a*p, 2*b*q, v, p+1, q+1, exact, exact-v, a, b),
		Facts: map[string]string{"estimate": fmt.Sprintf("%d", v), "value": fmt.Sprintf("%d", f)},
	}
}

// closedExtremaGen uses a downward bowl on a disc, whose maximum is at the centre.
type closedExtremaGen struct{}

func (g *closedExtremaGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	c := 5 + rand.Intn(20+int(ctx.Difficulty*20))
	a := 1 + rand.Intn(3+int(ctx.Difficulty*3))
	b := 1 + rand.Intn(3+int(ctx.Difficulty*3))
	return generator.Problem{
		Question:    fmt.Sprintf("Find the maximum value of \\(f(x,y) = %d - %dx^2 - %dy^2\\) on the closed disc \\(x^2 + y^2 \\le 4\\).", c, a, b),
		Answer:      fmt.Sprintf("%d", c),
		Explanation: fmt.Sprintf("A closed and bounded region always has a maximum, and here the function decreases as either variable moves away from the origin, so the maximum is at the interior critical point (0,0), where the value is %d. The boundary circle gives smaller values, which is why it need not be checked further once the interior point wins.", c),
		Facts:       map[string]string{"maximum": fmt.Sprintf("%d", c)},
	}
}

// polarAreaGen uses the sector area integral, with an angular width that keeps it whole.
type polarAreaGen struct{}

func (g *polarAreaGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	r := 2 + rand.Intn(4+int(ctx.Difficulty*6))
	k := 2 * (1 + rand.Intn(3+int(ctx.Difficulty*3))) // the angular width, in radians
	v := r * r * k / 2
	return generator.Problem{
		Question: fmt.Sprintf("Find the area of the region swept by \\(r = %d\\) as \\(\\theta\\) runs from 0 to %d radians.", r, k),
		Answer:   fmt.Sprintf("%d", v),
		Explanation: fmt.Sprintf("The area in polar coordinates is (1/2) times the integral of r^2 dtheta, and here r is constant: (1/2)(%d^2)(%d) = %d. Half the square of the radius times the angle is also the fraction of the circle's area that the angle covers.",
			r, k, v),
		Facts: map[string]string{"area": fmt.Sprintf("%d", v)},
	}
}

// polarSlopeGen evaluates dy/dx for a circle given in polar form, at angles whose cotangent is
// whole.
type polarSlopeGen struct{}

func (g *polarSlopeGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	// r = a is a circle: dy/dx = -cot(theta), which is -1, 1 or 0 at these angles.
	type entry struct {
		label string
		value int
	}
	angles := []entry{{"\\pi/4", -1}, {"3\\pi/4", 1}, {"\\pi/2", 0}, {"5\\pi/4", -1}}
	e := angles[rand.Intn(len(angles))]
	return generator.Problem{
		Question:    fmt.Sprintf("The curve \\(r = 5\\) is a circle in polar coordinates. Find \\(\\frac{dy}{dx}\\) at \\(\\theta = %s\\).", e.label),
		Answer:      fmt.Sprintf("%d", e.value),
		Explanation: fmt.Sprintf("For a polar curve, dy/dx is (r' sin t + r cos t)/(r' cos t - r sin t). With r constant its derivative is zero, leaving -cot t, and at this angle that is %d. That agrees with the tangent to the circle at that point.", e.value),
		Facts:       map[string]string{"slope": fmt.Sprintf("%d", e.value)},
	}
}
