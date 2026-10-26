package geometry

// The last geometry block: slope, distance in space, the trapezoid, similar solids' volumes, and the
// tangent-chord angle.
//
// These fill the four corners the domain still had open -- coordinate work beyond the distance
// formula, the trapezoid as the one quadrilateral whose area needs an average, volumes of similar
// solids as the cube of the length ratio, and the second of the two tangent theorems.
//
// Every answer is an integer by construction, and each ships an authored solution schema because
// geometry is one of the six learner domains.
import (
	"fmt"
	"math/rand"

	"github.com/chuma-beep/mathua/internal/generator"
)

func registerFinal(reg *generator.Registry) {
	reg.Register("geo.coord.slope", &slopeGen{})
	reg.Register("geo.coord.distance_3d", &distance3DGen{})
	reg.Register("geo.quad.trapezoid_area", &trapezoidAreaGen{})
	reg.Register("geo.solid.similar_volume", &similarVolumeGen{})
	reg.Register("geo.circle.tangent_chord", &tangentChordGen{})
}

// slopeGen asks for the slope through two points, with integer offsets so the answer is whole.
type slopeGen struct{}

func (g *slopeGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	run := 1 + rand.Intn(2+int(ctx.Difficulty*4))
	slope := 1 + rand.Intn(2+int(ctx.Difficulty*4))
	if rand.Intn(2) == 0 {
		slope = -slope
	}
	x1 := rand.Intn(7) - 3
	y1 := rand.Intn(7) - 3
	x2, y2 := x1+run, y1+slope*run
	return generator.Problem{
		Question: fmt.Sprintf("Find the slope of the line through \\((%d,%d)\\) and \\((%d,%d)\\).", x1, y1, x2, y2),
		Answer:   fmt.Sprintf("%d", slope),
		Explanation: fmt.Sprintf("Slope is the rise over the run: the change in y is %d and the change in x is %d, so the slope is %d.",
			slope*run, run, slope),
		Facts: map[string]string{"slope": fmt.Sprintf("%d", slope), "rise": fmt.Sprintf("%d", slope*run),
			"run": fmt.Sprintf("%d", run)},
	}
}

// distance3DGen uses a Pythagorean quadruple, so the distance in space is
// whole. Difficulty moves the first point off the origin, then permutes and
// signs the differences so both points have negative coordinates.
type distance3DGen struct{}

var pythagoreanQuads = [][4]int{
	{1, 2, 2, 3}, {2, 3, 6, 7}, {1, 4, 8, 9}, {4, 4, 7, 9}, {2, 6, 9, 11}, {6, 6, 7, 11},
	{3, 4, 12, 13}, {2, 5, 14, 15}, {2, 10, 11, 15}, {1, 12, 12, 17}, {8, 9, 12, 17}, {6, 10, 15, 19},
}

func (g *distance3DGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	pool := pythagoreanQuads[:6]
	if ctx.Difficulty >= 0.45 {
		pool = pythagoreanQuads
	}
	q := pool[rand.Intn(len(pool))]
	d := []int{q[0], q[1], q[2]}
	var p1 [3]int
	if ctx.Difficulty >= 0.45 {
		for i := range p1 {
			p1[i] = rand.Intn(11) - 5
		}
	}
	if ctx.Difficulty >= 0.7 {
		rand.Shuffle(3, func(i, j int) { d[i], d[j] = d[j], d[i] })
		for i := range d {
			if rand.Intn(2) == 0 {
				d[i] = -d[i]
			}
		}
	}
	p2 := [3]int{p1[0] + d[0], p1[1] + d[1], p1[2] + d[2]}
	sq := d[0]*d[0] + d[1]*d[1] + d[2]*d[2]
	return generator.Problem{
		Question: fmt.Sprintf("Find the distance between the points \\((%d,%d,%d)\\) and \\((%d,%d,%d)\\).", p1[0], p1[1], p1[2], p2[0], p2[1], p2[2]),
		Answer:   fmt.Sprintf("%d", q[3]),
		Explanation: fmt.Sprintf("The coordinate differences are %d, %d and %d. The distance is sqrt(%d + %d + %d) = sqrt(%d) = %d.",
			d[0], d[1], d[2], d[0]*d[0], d[1]*d[1], d[2]*d[2], sq, q[3]),
		Facts: map[string]string{
			"dx": fmt.Sprintf("%d", d[0]), "dy": fmt.Sprintf("%d", d[1]), "dz": fmt.Sprintf("%d", d[2]),
			"dx_sq": fmt.Sprintf("%d", d[0]*d[0]), "dy_sq": fmt.Sprintf("%d", d[1]*d[1]), "dz_sq": fmt.Sprintf("%d", d[2]*d[2]),
			"sum_sq": fmt.Sprintf("%d", sq), "distance": fmt.Sprintf("%d", q[3]),
		},
	}
}

// trapezoidAreaGen uses the average of the two parallel sides.
type trapezoidAreaGen struct{}

func (g *trapezoidAreaGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	b1 := 2 * (1 + rand.Intn(4+int(ctx.Difficulty*5)))
	b2 := 2 * (1 + rand.Intn(4+int(ctx.Difficulty*5)))
	h := 1 + rand.Intn(5+int(ctx.Difficulty*6))
	area := (b1 + b2) / 2 * h
	return generator.Problem{
		Question: fmt.Sprintf("A trapezoid has parallel sides of %d and %d and a height of %d. What is its area?", b1, b2, h),
		Answer:   fmt.Sprintf("%d", area),
		Explanation: fmt.Sprintf("A trapezoid's area is the average of its parallel sides times the height: the average of %d and %d is %d, and %d times %d is %d.",
			b1, b2, (b1+b2)/2, (b1+b2)/2, h, area),
		Facts: map[string]string{"area": fmt.Sprintf("%d", area), "base_a": fmt.Sprintf("%d", b1),
			"base_b": fmt.Sprintf("%d", b2), "height": fmt.Sprintf("%d", h)},
	}
}

// similarVolumeGen applies the cube of the length ratio to volumes.
type similarVolumeGen struct{}

func (g *similarVolumeGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	k := 2 + rand.Intn(3+int(ctx.Difficulty*3))
	return generator.Problem{
		Question: fmt.Sprintf("Two solids are similar, and every length in the larger is %d times the corresponding length in the smaller. How many times larger is the larger solid's volume?", k),
		Answer:   fmt.Sprintf("%d", k*k*k),
		Explanation: fmt.Sprintf("Volumes scale by the cube of the length ratio, because a solid has three dimensions: %d^3 = %d. Its surface area, having two dimensions, scales by %d instead.",
			k, k*k*k, k*k),
		Facts: map[string]string{"factor": fmt.Sprintf("%d", k), "volume_ratio": fmt.Sprintf("%d", k*k*k),
			"area_ratio": fmt.Sprintf("%d", k*k)},
	}
}

// tangentChordGen uses the angle between a tangent and a chord, which is half the intercepted arc.
type tangentChordGen struct{}

func (g *tangentChordGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	arc := 2 * (3 + rand.Intn(20+int(ctx.Difficulty*20)))
	return generator.Problem{
		Question: fmt.Sprintf("A tangent touches a circle at a point, and a chord is drawn from that point. The arc intercepted by the angle between the tangent and the chord has measure %d degrees. What is that angle, in degrees?", arc),
		Answer:   fmt.Sprintf("%d", arc/2),
		Explanation: fmt.Sprintf("The angle between a tangent and a chord is half the intercepted arc, the same halving as the inscribed-angle theorem: %d / 2 = %d.",
			arc, arc/2),
		Facts: map[string]string{"angle": fmt.Sprintf("%d", arc/2), "arc": fmt.Sprintf("%d", arc)},
	}
}
