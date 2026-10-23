package calculus

// Surface integrals, Stokes' theorem, and the two applications that make them concrete.
//
// These are the pieces the vector-calculus block was missing: the surface integral of a scalar over
// a graph (where the area element carries the factor sqrt(1 + f_x^2 + f_y^2)), Stokes' theorem (the
// circulation around a boundary as the flux of the curl through any surface spanning it), the curl
// of a field in three dimensions, and the work done by a field along a path.
//
// Every answer is an integer by construction. The surface area uses directions whose normal is a
// Pythagorean triple, so the square root in the area element comes out whole, and the work and the
// curl components are linear forms in the parameters.
import (
	"fmt"
	"math/rand"

	"github.com/chuma-beep/mathua/internal/generator"
)

func registerSurface(reg *generator.Registry) {
	reg.Register("calc.vec.surface_area", &surfaceAreaGen{})
	reg.Register("calc.vec.stokes", &stokesGen{})
	reg.Register("calc.vec.curl_3d", &curl3DGen{})
	reg.Register("calc.vec.work", &workGen{})
}

// surfaceAreaGen integrates the area element of a graph over a rectangle. The direction is one whose
// normal has a whole length, so the answer stays an integer.
type surfaceAreaGen struct{}

func (g *surfaceAreaGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	// (a, b, c) with 1 + a^2 + b^2 = c^2.
	type dir struct{ a, b, c int }
	easy := []dir{{0, 0, 1}, {2, 2, 3}}
	hard := []dir{{4, 8, 9}, {12, 12, 17}, {6, 18, 19}}
	dirs := hard
	if ctx.Difficulty < 0.45 {
		dirs = easy
	}
	d := dirs[pickMV(len(dirs))]
	p := small(1, 2+int(ctx.Difficulty*3))
	q := small(1, 2+int(ctx.Difficulty*3))
	return generator.Problem{
		Question: fmt.Sprintf("Find the surface area of the part of the plane \\(z = %dx + %dy\\) that lies above the rectangle \\(0 \\le x \\le %d\\), \\(0 \\le y \\le %d\\).", d.a, d.b, p, q),
		Answer:   fmt.Sprintf("%d", p*q*d.c),
		Explanation: fmt.Sprintf("The area element for a graph is sqrt(1 + f_x^2 + f_y^2) dx dy, and here that factor is sqrt(1 + %d + %d) = %d, a constant, so the area is %d times the rectangle's area %d * %d = %d.",
			d.a*d.a, d.b*d.b, d.c, d.c, p, q, p*q*d.c),
		Facts: map[string]string{"area": fmt.Sprintf("%d", p*q*d.c)},
	}
}

// stokesGen uses Stokes' theorem on a flat surface, where the flux of the curl is a constant times
// the area.
type stokesGen struct{}

func (g *stokesGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	a := small(1, 2+int(ctx.Difficulty*3))
	b := small(1, 2+int(ctx.Difficulty*3))
	p := small(1, 3+int(ctx.Difficulty*3))
	q := small(1, 3+int(ctx.Difficulty*3))
	return generator.Problem{
		Question: fmt.Sprintf("Use Stokes' theorem to evaluate the circulation of \\\\(\\mathbf{F} = (%dy,\\\\ %dx,\\\\ 0)\\\\) around the boundary of the rectangle \\\\(0 \\le x \\le %d\\\\), \\\\(0 \\le y \\le %d\\\\) in the plane \\\\(z = 2\\\\), oriented counterclockwise seen from above.", a, b, p, q),
		Answer:   fmt.Sprintf("%d", (b-a)*p*q),
		Explanation: fmt.Sprintf("The surface is flat with unit normal in the z direction, and the z component of the curl is dQ/dx - dP/dy = %d - %d = %d. Stokes' theorem turns the circulation into that constant times the area: %d * %d * %d = %d.",
			b, a, b-a, b-a, p, q, (b-a)*p*q),
		Facts: map[string]string{"circulation": fmt.Sprintf("%d", (b-a)*p*q)},
	}
}

// curl3DGen asks for one component of the curl of a field in three dimensions, chosen at random so
// the three formulas each appear.
type curl3DGen struct{}

func (g *curl3DGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	a := small(1, 4+int(ctx.Difficulty*4))
	b := small(1, 4+int(ctx.Difficulty*4))
	c := small(1, 4+int(ctx.Difficulty*4))
	// F = (a y, b z, c x) has curl (-b, -c, -a).
	components := []struct {
		name, answer, why string
	}{
		{"x", fmt.Sprintf("%d", -b), fmt.Sprintf("dR/dy - dQ/dz = 0 - %d", b)},
		{"y", fmt.Sprintf("%d", -c), fmt.Sprintf("dP/dz - dR/dx = 0 - %d", c)},
		{"z", fmt.Sprintf("%d", -a), fmt.Sprintf("dQ/dx - dP/dy = 0 - %d", a)},
	}
	chosen := components[rand.Intn(3)]
	return generator.Problem{
		Question: fmt.Sprintf("For \\\\(\\mathbf{F} = (%dy,\\\\ %dz,\\\\ %dx)\\\\), find the %s-component of \\\\(\\nabla \\times \\mathbf{F}\\\\).", a, b, c, chosen.name),
		Answer:   chosen.answer,
		Explanation: fmt.Sprintf("The components of the curl are the three cross-partial differences, and the %s one is %s = %s.",
			chosen.name, chosen.why, chosen.answer),
		Facts: map[string]string{"component": chosen.answer},
	}
}

// workGen is the physical reading of a line integral: the work a constant field does along a
// straight path.
type workGen struct{}

func (g *workGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	a := small(1, 4+int(ctx.Difficulty*4))
	b := small(1, 4+int(ctx.Difficulty*4))
	p := small(1, 4+int(ctx.Difficulty*4))
	q := small(1, 4+int(ctx.Difficulty*4))
	return generator.Problem{
		Question: fmt.Sprintf("A constant force \\\\(\\mathbf{F} = (%d, %d)\\\\) acts on an object that moves in a straight line from \\\\((0,0)\\\\) to \\\\((%d,%d)\\\\). Find the work done.", a, b, p, q),
		Answer:   fmt.Sprintf("%d", a*p+b*q),
		Explanation: fmt.Sprintf("Work is the integral of F along the path, and along a straight displacement it is the dot product of the force with it: %d*%d + %d*%d = %d.",
			a, p, b, q, a*p+b*q),
		Facts: map[string]string{"work": fmt.Sprintf("%d", a*p+b*q)},
	}
}
