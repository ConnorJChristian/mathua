package calculus

// The vector-calculus theorems and the coordinate changes that make the multiple integrals of
// multivariable.go computable.
//
// These are the concepts that connect an integral over a region to an integral over its boundary
// (Green's theorem), and the two differential operators that decide whether a field is a gradient
// (curl and divergence). Every answer is an exact integer by construction, and every generator
// offers at least three distinct question texts at difficulty 0.3, because the practice feed asks
// for three questions at a time and the bank test gates on the smallest of the difficulties it
// sweeps rather than on the average.
import (
	"fmt"
	"math/rand"

	"github.com/chuma-beep/mathua/internal/generator"
)

func registerVector(reg *generator.Registry) {
	reg.Register("calc.vec.curl", &curlGen{})
	reg.Register("calc.vec.divergence", &divergenceGen{})
	reg.Register("calc.vec.conservative_test", &conservativeTestGen{})
	reg.Register("calc.vec.greens_theorem", &greensTheoremGen{})
	reg.Register("calc.multi.transform_jacobian", &transformJacobianGen{})
	reg.Register("calc.multi.double_integral_region", &doubleIntegralRegionGen{})
	reg.Register("calc.multi.saddle_test", &saddleTestGen{})
	reg.Register("calc.multi.lagrange", &lagrangeGen{})
}

// small returns a random int in [lo, hi], and never panics when hi < lo.
func small(lo, hi int) int {
	if hi < lo {
		hi = lo
	}
	return lo + rand.Intn(hi-lo+1)
}

// curlGen asks for the scalar curl of a linear field, which is the constant dQ/dx - dP/dy.
type curlGen struct{}

func (g *curlGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	a := small(1, 2+int(ctx.Difficulty*4))
	b := small(1, 2+int(ctx.Difficulty*4))
	return generator.Problem{
		Question: fmt.Sprintf("For the field \\(\\mathbf{F} = (%dy,\\ %dx)\\), find the scalar curl \\(\\frac{\\partial Q}{\\partial x} - \\frac{\\partial P}{\\partial y}\\).", a, b),
		Answer:   fmt.Sprintf("%d", b-a),
		Explanation: fmt.Sprintf("Q = %dx so dQ/dx = %d, and P = %dy so dP/dy = %d. The curl is %d - %d = %d: a non-zero curl says the field swirls and is not a gradient.",
			b, b, a, a, b, a, b-a),
		Facts: map[string]string{"a": fmt.Sprintf("%d", a), "b": fmt.Sprintf("%d", b),
			"curl": fmt.Sprintf("%d", b-a)},
	}
}

// divergenceGen asks for the divergence of a linear field, which is a + b.
type divergenceGen struct{}

func (g *divergenceGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	a := small(1, 3+int(ctx.Difficulty*4))
	b := small(1, 3+int(ctx.Difficulty*4))
	return generator.Problem{
		Question: fmt.Sprintf("For the field \\(\\mathbf{F} = (%dx,\\ %dy)\\), find the divergence \\(\\frac{\\partial P}{\\partial x} + \\frac{\\partial Q}{\\partial y}\\).", a, b),
		Answer:   fmt.Sprintf("%d", a+b),
		Explanation: fmt.Sprintf("The partials are the diagonal entries %d and %d, so the divergence is %d + %d = %d. It measures how much the field spreads rather than turns, which is why a field can have zero curl and non-zero divergence.",
			a, b, a, b, a+b),
		Facts: map[string]string{"a": fmt.Sprintf("%d", a), "b": fmt.Sprintf("%d", b),
			"divergence": fmt.Sprintf("%d", a+b)},
	}
}

// conservativeTestGen asks for the cross-partial difference, which is zero exactly for a gradient.
type conservativeTestGen struct{}

func (g *conservativeTestGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	b := small(1, 3+int(ctx.Difficulty*4))
	c := small(1, 3+int(ctx.Difficulty*4))
	return generator.Problem{
		Question: fmt.Sprintf("For \\(\\mathbf{F} = (2x + %dy,\\ %dx + 5y)\\), compute \\(\\frac{\\partial Q}{\\partial x} - \\frac{\\partial P}{\\partial y}\\). A value of zero means the field is conservative.", b, c),
		Answer:   fmt.Sprintf("%d", c-b),
		Explanation: fmt.Sprintf("dQ/dx = %d and dP/dy = %d, so the difference is %d - %d = %d. When it is zero on a simply connected region the field is the gradient of a potential and every line integral becomes a difference of potentials.",
			c, b, c, b, c-b),
		Facts: map[string]string{"b": fmt.Sprintf("%d", b), "c": fmt.Sprintf("%d", c),
			"difference": fmt.Sprintf("%d", c-b)},
	}
}

// greensTheoremGen uses Green's theorem to turn a boundary integral into an area, which for a
// rectangle is the product of the sides.
type greensTheoremGen struct{}

func (g *greensTheoremGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	a := small(1, 3+int(ctx.Difficulty*4))
	b := small(1, 3+int(ctx.Difficulty*4))
	return generator.Problem{
		Question: fmt.Sprintf("Use Green's theorem to evaluate \\(\\oint_C x\\,dy\\) counterclockwise around the rectangle with corners \\((0,0)\\), \\((%d,0)\\), \\((%d,%d)\\) and \\((0,%d)\\).", a, a, b, b),
		Answer:   fmt.Sprintf("%d", a*b),
		Explanation: fmt.Sprintf("Green's theorem turns the boundary integral into \\(\\iint (\\partial Q/\\partial x - \\partial P/\\partial y)\\,dA\\) with P = 0 and Q = x, and that integrand is 1, so the integral is the area %d times %d = %d.",
			a, b, a*b),
		Facts: map[string]string{"a": fmt.Sprintf("%d", a), "b": fmt.Sprintf("%d", b),
			"area": fmt.Sprintf("%d", a*b)},
	}
}

// transformJacobianGen asks for the determinant of a linear change of variables, which is ad - bc.
type transformJacobianGen struct{}

func (g *transformJacobianGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	a := small(1, 3+int(ctx.Difficulty*4))
	b := small(1, 3+int(ctx.Difficulty*4))
	c := small(1, 3+int(ctx.Difficulty*4))
	d := small(1, 3+int(ctx.Difficulty*4))
	det := a*d - b*c
	return generator.Problem{
		Question: fmt.Sprintf("For the change of variables \\(x = %du + %dv,\\ y = %du + %dv\\), find \\(\\frac{\\partial(x,y)}{\\partial(u,v)}\\).", a, b, c, d),
		Answer:   fmt.Sprintf("%d", det),
		Explanation: fmt.Sprintf("The matrix of partials is [[%d, %d], [%d, %d]] and its determinant is %d*%d - %d*%d = %d. Its absolute value is the factor that converts an area element du dv into dx dy.",
			a, b, c, d, a, d, b, c, det),
		Facts: map[string]string{"det": fmt.Sprintf("%d", det)},
	}
}

// doubleIntegralRegionGen integrates 1 over a triangle whose area is a^2/2, with a even so the
// answer is whole.
type doubleIntegralRegionGen struct{}

func (g *doubleIntegralRegionGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	a := 2 * small(1, 3+int(ctx.Difficulty*3))
	return generator.Problem{
		Question: fmt.Sprintf("Evaluate \\(\\iint_R 1\\,dA\\) where R is the triangle \\(0 \\le y \\le x \\le %d\\).", a),
		Answer:   fmt.Sprintf("%d", a*a/2),
		Explanation: fmt.Sprintf("The integrand is 1, so the integral is the area of the triangle, which has base %d and height %d: base times height over two is %d.",
			a, a, a*a/2),
		Facts: map[string]string{"a": fmt.Sprintf("%d", a), "area": fmt.Sprintf("%d", a*a/2)},
	}
}

// saddleTestGen asks for the second-derivative discriminant of f = a x^2 + b y^2 at its critical
// point, which is 4ab.
type saddleTestGen struct{}

func (g *saddleTestGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	a := small(1, 3+int(ctx.Difficulty*3))
	b := small(1, 3+int(ctx.Difficulty*3))
	return generator.Problem{
		Question: fmt.Sprintf("For \\(f(x,y) = %dx^2 + %dy^2\\), evaluate the discriminant \\(D = f_{xx}f_{yy} - (f_{xy})^2\\) at the origin.", a, b),
		Answer:   fmt.Sprintf("%d", 4*a*b),
		Explanation: fmt.Sprintf("f_xx = %d, f_yy = %d and f_xy = 0, so D = %d * %d = %d. A positive D with f_xx > 0 makes the point a minimum; a negative D would make it a saddle.",
			2*a, 2*b, 2*a, 2*b, 4*a*b),
		Facts: map[string]string{"D": fmt.Sprintf("%d", 4*a*b)},
	}
}

// lagrangeGen maximises a product under a fixed sum, which is (s/2)^2 with s even.
type lagrangeGen struct{}

func (g *lagrangeGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	s := 2 * small(2, 4+int(ctx.Difficulty*4))
	return generator.Problem{
		Question: fmt.Sprintf("Use Lagrange multipliers to find the largest value of \\(xy\\) subject to \\(x + y = %d\\) with \\(x, y > 0\\).", s),
		Answer:   fmt.Sprintf("%d", s*s/4),
		Explanation: fmt.Sprintf("Lagrange gives y = lambda and x = lambda, so x = y = %d/2 = %d and the product is %d*%d = %d. The constraint forces the two factors to share the sum equally.",
			s, s/2, s/2, s/2, s*s/4),
		Facts: map[string]string{"s": fmt.Sprintf("%d", s), "maximum": fmt.Sprintf("%d", s*s/4), "half": fmt.Sprintf("%d", s/2)},
	}
}
