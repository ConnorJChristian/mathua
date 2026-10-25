package complex

// The 2026-10-17 batch: sixteen concepts that take complex_numbers from 26 to the domain target of 42.
//
// Every answer is computed from the instance. Seven concepts answer with the `complex` grading type,
// whose path rewrites `i` to SymPy's `I` and compares expressions; the rest are exact integers or
// recognitions. Answers avoid polar form except where the angle is a whole number of degrees, because
// the grader's polar conversion only accepts integers.

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/chuma-beep/mathua/internal/generator"
)

func registerBatch(reg *generator.Registry) {
	reg.Register("complex.basics.parts", &partsGen{})
	reg.Register("complex.basics.powers", &powersGen{})
	reg.Register("complex.basics.reciprocal", &reciprocalGen{})
	reg.Register("complex.basics.geometry", &planeDistanceGen{})
	reg.Register("complex.basics.loci", &lociGen{})
	reg.Register("complex.basics.arg_arithmetic", &argArithmeticGen{})
	reg.Register("complex.basics.equations", &complexEquationsGen{})
	reg.Register("complex.adv.conjugate_roots", &conjugateRootsGen{})
	reg.Register("complex.adv.trig_exponential", &trigExponentialGen{})
	reg.Register("complex.adv.hyperbolic", &hyperbolicGen{})
	reg.Register("complex.adv.geometric_series", &complexSeriesGen{})
	reg.Register("complex.adv.mobius_fixed", &mobiusFixedGen{})
	reg.Register("complex.adv.residue_simple", &simpleResidueGen{})
	reg.Register("complex.adv.cross_ratio", &crossRatioGen{})
	reg.Register("complex.topology.open_sets", &openSetGen{})
	reg.Register("complex.topology.domain", &domainGen{})
}

// ----- helpers -----

func gcdInt(a, b int) int {
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

// fracStr renders n/d in lowest terms with a positive denominator.
func fracStr(n, d int) string {
	if d < 0 {
		n, d = -n, -d
	}
	g := gcdInt(n, d)
	if g > 1 {
		n, d = n/g, d/g
	}
	if d == 1 {
		return strconv.Itoa(n)
	}
	return fmt.Sprintf("%d/%d", n, d)
}

// complexFrac renders (reN/reD) + (imN/imD)i, which is what a conjugate division produces.
func complexFrac(reN, reD, imN, imD int) string {
	re := fracStr(reN, reD)
	im := fracStr(imN, imD)
	im = strings.TrimPrefix(im, "-")
	if imN >= 0 {
		return re + "+" + im + "i"
	}
	return re + "-" + im + "i"
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// signed renders a leading sign for use inside a sum.
func signed(n int) string {
	if n >= 0 {
		return "+" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

// ----- 1. parts -----

type partsGen struct{}

func (g *partsGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	a := rand.Intn(17) - 8 // imaginary part of the left side
	b := rand.Intn(17) - 8 // real part of the right side
	if a == 0 {
		a = 3
	}
	if b == 0 {
		b = 5
	}
	left := fmt.Sprintf("x %s %di", map[bool]string{true: "+", false: "-"}[a >= 0], absInt(a))
	q := fmt.Sprintf("The complex numbers \\(%s\\) and \\(%d + yi\\) are equal. What is \\(x + y\\)?", left, b)
	p := generator.Problem{
		Question: q,
		Answer:   strconv.Itoa(b + a),
		Facts: map[string]string{
			"a": strconv.Itoa(a), "b": strconv.Itoa(b),
			"statement": fmt.Sprintf("%s = %d + yi", left, b),
			"x":         strconv.Itoa(b), "y": strconv.Itoa(a), "value": strconv.Itoa(b + a),
		},
	}
	p.Explanation = fmt.Sprintf("Two complex numbers are equal exactly when their real parts agree and their imaginary parts agree, so x = %d and y = %d. Then x + y = %d.", b, a, b+a)
	return p
}

// ----- 2. powers -----

type powersGen struct{}

func (g *powersGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	if rand.Intn(2) == 0 {
		k := 10 + rand.Intn(90)
		answers := []string{"1", "i", "-1", "-i"}
		ans := answers[k%4]
		p := generator.Problem{
			Question: fmt.Sprintf("What is \\(i^{%d}\\)?", k),
			Answer:   ans,
			Facts: map[string]string{
				"base": "i", "exponent": strconv.Itoa(k), "value": ans,
				"method": fmt.Sprintf("The powers of i repeat with period 4, and %d leaves remainder %d on division by 4.", k, k%4),
			},
		}
		p.Explanation = fmt.Sprintf("The powers of i repeat with period 4: i, -1, -i, 1. Since %d leaves remainder %d on division by 4, \\(i^{%d} = %s\\).", k, k%4, k, ans)
		return p
	}
	a := 1 + rand.Intn(4)
	b := 1 + rand.Intn(4)
	re := a*a - b*b
	im := 2 * a * b
	ans := fmtComplex(re, im)
	p := generator.Problem{
		Question: fmt.Sprintf("What is \\((%s)^2\\)?", fmtComplex(a, b)),
		Answer:   ans,
		Facts: map[string]string{
			"base": fmtComplex(a, b), "exponent": "2", "value": ans,
			"method": fmt.Sprintf("Expand the square: (a+bi)^2 = a^2 - b^2 + 2abi, so with a = %d and b = %d the real part is %d - %d and the imaginary part is %d.", a, b, a*a, b*b, im),
		},
	}
	p.Explanation = fmt.Sprintf("Square by expanding: (a+bi)^2 = a^2 - b^2 + 2abi. With a = %d and b = %d that is %d - %d + %d i = %s.", a, b, a*a, b*b, im, ans)
	return p
}

// ----- 3. reciprocal -----

type reciprocalGen struct{}

func (g *reciprocalGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	a := 1 + rand.Intn(4)
	b := 1 + rand.Intn(4)
	den := a*a + b*b
	ans := complexFrac(a, den, -b, den)
	p := generator.Problem{
		Question: fmt.Sprintf("Write \\(\\dfrac{1}{%s}\\) in the form \\(a + bi\\).", fmtComplex(a, b)),
		Answer:   ans,
		Facts: map[string]string{
			"a": strconv.Itoa(a), "b": strconv.Itoa(b), "den": strconv.Itoa(den), "z": fmtComplex(a, b),
			"conjugate": fmtComplex(a, -b),
			"real_num":  strconv.Itoa(a), "imag_num": strconv.Itoa(-b), "value": ans,
		},
	}
	p.Explanation = fmt.Sprintf("Multiply above and below by the conjugate %s: the denominator becomes %s + %s = %d, and the numerator is %s. So \\(1/(%s) = %s\\).",
		fmtComplex(a, -b), generator.Sq(a), generator.Sq(b), den, fmtComplex(a, -b), fmtComplex(a, b), ans)
	return p
}

// ----- 4. distance in the plane -----

type planeDistanceGen struct{}

func (g *planeDistanceGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type triple struct{ a, b, c int }
	triples := []triple{{3, 4, 5}, {4, 3, 5}, {6, 8, 10}, {8, 6, 10}, {5, 12, 13}, {12, 5, 13}}
	t := triples[rand.Intn(len(triples))]
	if rand.Intn(2) == 0 {
		t.a = -t.a
	}
	if rand.Intn(2) == 0 {
		t.b = -t.b
	}
	r1 := rand.Intn(9) - 4
	i1 := rand.Intn(9) - 4
	r2, i2 := r1+t.a, i1+t.b
	q := fmt.Sprintf("Find the distance in the complex plane between \\(%s\\) and \\(%s\\).",
		fmtComplex(r1, i1), fmtComplex(r2, i2))
	p := generator.Problem{
		Question: q,
		Answer:   strconv.Itoa(t.c),
		Facts: map[string]string{
			"z1": fmtComplex(r1, i1), "z2": fmtComplex(r2, i2),
			"dx": strconv.Itoa(t.a), "dy": strconv.Itoa(t.b), "value": strconv.Itoa(t.c),
		},
	}
	p.Explanation = fmt.Sprintf("The distance is the modulus of the difference: |(%s) - (%s)| = |%s| = sqrt(%s + %s) = %d.",
		fmtComplex(r2, i2), fmtComplex(r1, i1), fmtComplex(t.a, t.b), generator.Sq(t.a), generator.Sq(t.b), t.c)
	return p
}

// ----- 5. loci -----

type lociGen struct{}

func (g *lociGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	p := 1 + rand.Intn(4)
	q := 1 + rand.Intn(4)
	if rand.Intn(2) == 0 {
		q = -q
	}
	r := 2 + rand.Intn(4)
	z0 := fmtComplex(p, q)

	if rand.Intn(2) == 0 {
		statement := fmt.Sprintf("|z - (%s)| = %d", z0, r)
		correct := fmt.Sprintf("a circle centred at %s with radius %d", z0, r)
		wrongCentre := fmt.Sprintf("a circle centred at %d with radius %d", r, p+q)
		wrongRadius := fmt.Sprintf("a circle centred at %s with radius %d", z0, r+1)
		wrongType := fmt.Sprintf("the perpendicular bisector of the segment from 0 to %s", z0)
		pr := generator.MultipleChoice(
			fmt.Sprintf("Describe the set of complex numbers z with \\(|z - (%s)| = %d\\).", z0, r),
			correct, wrongCentre, wrongRadius, wrongType)
		pr.Facts = map[string]string{
			"statement": statement, "kind": "circle", "value": correct,
			"reason": fmt.Sprintf("|z - z0| is the distance from z to %s, so the points at the fixed distance %d from it form a circle centred there.", z0, r),
		}
		pr.Explanation = fmt.Sprintf("|z - z0| is the distance from z to z0, so the set of points at a fixed distance %d from %s is a circle centred there.", r, z0)
		return pr
	}

	p2 := p + 1 + rand.Intn(3)
	q2 := q + 1 + rand.Intn(3)
	z1 := fmtComplex(p, q)
	z2 := fmtComplex(p2, q2)
	statement := fmt.Sprintf("|z - (%s)| = |z - (%s)|", z1, z2)
	correct := fmt.Sprintf("the perpendicular bisector of the segment from %s to %s", z1, z2)
	wrongType := fmt.Sprintf("a circle centred at %s with radius %d", z1, p2)
	wrongPoint := fmt.Sprintf("the single point %s", z1)
	wrongHalf := fmt.Sprintf("the perpendicular bisector of the segment from 0 to %s", z2)
	pr := generator.MultipleChoice(
		fmt.Sprintf("Describe the set of complex numbers z with \\(|z - (%s)| = |z - (%s)|\\).", z1, z2),
		correct, wrongType, wrongPoint, wrongHalf)
	pr.Facts = map[string]string{
		"statement": statement, "kind": "bisector", "value": correct,
		"reason": fmt.Sprintf("Both distances are equal, so z is equidistant from %s and %s, and the points equidistant from two points form their perpendicular bisector.", z1, z2),
	}
	pr.Explanation = fmt.Sprintf("Both distances are equal, so z is equidistant from %s and %s. The locus of points equidistant from two points is their perpendicular bisector.", z1, z2)
	return pr
}

// ----- 6. argument arithmetic -----

type argArithmeticGen struct{}

func (g *argArithmeticGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	a1 := 10 * rand.Intn(36)
	a2 := 10 * rand.Intn(36)
	sum := (a1 + a2) % 360
	q := fmt.Sprintf("The complex number z has argument %d\u00b0 and w has argument %d\u00b0. Give the argument of \\(zw\\) in degrees, in the interval \\([0, 360)\\).", a1, a2)
	p := generator.Problem{
		Question: q,
		Answer:   strconv.Itoa(sum),
		Facts: map[string]string{
			"a1": strconv.Itoa(a1), "a2": strconv.Itoa(a2),
			"sum": strconv.Itoa(a1 + a2), "value": strconv.Itoa(sum),
		},
	}
	p.Explanation = fmt.Sprintf("Multiplying multiplies the moduli and adds the arguments: arg(zw) = arg z + arg w = %d\u00b0 + %d\u00b0 = %d\u00b0, which reduces to %d\u00b0 modulo 360.", a1, a2, a1+a2, sum)
	return p
}

// ----- 7. linear equations -----

type complexEquationsGen struct{}

func (g *complexEquationsGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	a := 1 + rand.Intn(3)
	b := 1 + rand.Intn(3)
	if rand.Intn(2) == 0 {
		b = -b
	}
	// Choose the answer first so the coefficient equation is exact.
	pr := rand.Intn(5) - 2
	pi := rand.Intn(5) - 2
	c := a*pr - b*pi
	d := a*pi + b*pr
	ans := fmtComplex(pr, pi)
	q := fmt.Sprintf("Solve \\((%s)z = %s\\) for the complex number z.", fmtComplex(a, b), fmtComplex(c, d))
	p := generator.Problem{
		Question: q,
		Answer:   ans,
		Facts: map[string]string{
			"a": strconv.Itoa(a), "b": strconv.Itoa(b), "c": strconv.Itoa(c), "d": strconv.Itoa(d),
			"lhs": fmtComplex(a, b), "rhs": fmtComplex(c, d),
			"re": strconv.Itoa(pr), "im": strconv.Itoa(pi), "value": ans,
		},
	}
	p.Explanation = fmt.Sprintf("Divide both sides by %s, which means multiplying by its conjugate %s over %d: z = %s.", fmtComplex(a, b), fmtComplex(a, -b), a*a+b*b, ans)
	return p
}

// ----- 8. conjugate roots -----

type conjugateRootsGen struct{}

func (g *conjugateRootsGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	p := 1 + rand.Intn(5)
	q := 1 + rand.Intn(5)
	if rand.Intn(2) == 0 {
		q = -q
	}
	root := fmtComplex(p, q)
	correct := fmtComplex(p, -q)
	wrong1 := fmtComplex(-p, q)
	wrong2 := fmtComplex(-p, -q)
	wrong3 := fmtComplex(q, -p)
	pr := generator.MultipleChoice(
		fmt.Sprintf("A polynomial with real coefficients has the complex root %s. Which number must also be a root?", root),
		correct, wrong1, wrong2, wrong3)
	pr.Facts = map[string]string{
		"root": root, "conjugate": correct, "value": correct,
	}
	pr.Explanation = fmt.Sprintf("Real coefficients make the polynomial's values at conjugate points conjugate, so if %s is a root then %s is too: the pair appears together.", root, correct)
	return pr
}

// ----- 9. trig from exponentials -----

type trigExponentialGen struct{}

func (g *trigExponentialGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	// The angle is θ, 2θ or 3θ, so a learner meets six questions, not two
	// (two stems were all this had once option order stopped varying).
	arg := []string{"\u03b8", "2\u03b8", "3\u03b8"}[rand.Intn(3)]
	e := func(sign string) string { return "e^(" + sign + "i" + arg + ")" }
	cosIdent := "(" + e("") + " + " + e("-") + ")/2"
	sinIdent := "(" + e("") + " - " + e("-") + ")/(2i)"
	cosSigned := "(" + e("") + " - " + e("-") + ")/2"
	cosOverI := "(" + e("") + " + " + e("-") + ")/(2i)"
	if rand.Intn(2) == 0 {
		pr := generator.MultipleChoice(
			fmt.Sprintf("Which identity expresses cos(%s) in terms of complex exponentials?", arg),
			cosIdent, sinIdent, cosSigned, cosOverI)
		pr.Facts = map[string]string{"asked": "cos", "cos": cosIdent, "sin": sinIdent, "value": cosIdent}
		pr.Explanation = fmt.Sprintf("Adding the two exponentials cancels the sine terms: %s + %s = 2 cos(%s), so cos(%s) = %s.", e(""), e("-"), arg, arg, cosIdent)
		return pr
	}
	pr := generator.MultipleChoice(
		fmt.Sprintf("Which identity expresses sin(%s) in terms of complex exponentials?", arg),
		sinIdent, cosIdent, cosSigned, cosOverI)
	pr.Facts = map[string]string{"asked": "sin", "cos": cosIdent, "sin": sinIdent, "value": sinIdent}
	pr.Explanation = fmt.Sprintf("Subtracting the two exponentials cancels the cosine terms: %s - %s = 2i sin(%s), so sin(%s) = %s.", e(""), e("-"), arg, arg, sinIdent)
	return pr
}

// ----- 10. hyperbolic on imaginary arguments -----

type hyperbolicGen struct{}

func (g *hyperbolicGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	// k·iθ for k = 1, 2, 3: six questions where there were two.
	k := 1 + rand.Intn(3)
	th := "\u03b8"
	if k > 1 {
		th = strconv.Itoa(k) + "\u03b8"
	}
	arg := "i" + th
	if k > 1 {
		arg = strconv.Itoa(k) + "i\u03b8"
	}
	if rand.Intn(2) == 0 {
		pr := generator.MultipleChoice(
			fmt.Sprintf("For real \u03b8, which real function equals cosh(%s)?", arg),
			"cos "+th, "sin "+th, "i sin "+th, "cosh "+th)
		pr.Facts = map[string]string{"fn": "cosh", "argument": arg, "value": "cos " + th}
		pr.Explanation = fmt.Sprintf("cosh(ix) = (e^(ix) + e^(-ix))/2, which is exactly the identity for cos x; with x = %s, cosh(%s) = cos %s.", th, arg, th)
		return pr
	}
	pr := generator.MultipleChoice(
		fmt.Sprintf("For real \u03b8, which function equals sinh(%s)?", arg),
		"i sin "+th, "sin "+th, "cos "+th, "i cosh "+th)
	pr.Facts = map[string]string{"fn": "sinh", "argument": arg, "value": "i sin " + th}
	pr.Explanation = fmt.Sprintf("sinh(ix) = (e^(ix) - e^(-ix))/2, and e^(ix) - e^(-ix) = 2i sin x; with x = %s, sinh(%s) = i sin %s.", th, arg, th)
	return pr
}

// ----- 11. complex geometric series -----

type complexSeriesGen struct{}

func (g *complexSeriesGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	k := 2 + rand.Intn(3) // ratio i/k, modulus 1/k < 1
	den := k*k + 1
	reN := k * k
	imN := k
	ans := complexFrac(reN, den, imN, den)
	r := fmt.Sprintf("i/%d", k)
	q := fmt.Sprintf("The series \\(1 + r + r^2 + \\cdots\\) has ratio \\(r = %s\\). Find its sum as a complex number.", r)
	p := generator.Problem{
		Question: q,
		Answer:   ans,
		Facts: map[string]string{
			"k": strconv.Itoa(k), "ratio": r, "den": strconv.Itoa(den),
			"real_num": strconv.Itoa(reN), "imag_num": strconv.Itoa(imN), "value": ans,
		},
	}
	p.Explanation = fmt.Sprintf("The modulus is 1/%d < 1, so the geometric series converges to 1/(1 - r) = 1/(1 - i/%d). Multiplying by the conjugate of the denominator gives %s.", k, k, ans)
	return p
}

// ----- 12. Mobius fixed points -----

type mobiusFixedGen struct{}

func (g *mobiusFixedGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	c := 1 + rand.Intn(2)
	rootA := rand.Intn(7) - 3
	rootB := rand.Intn(7) - 3
	for rootB == rootA {
		rootB = rand.Intn(7) - 3
	}
	a := rand.Intn(7) - 3
	b := -c * rootA * rootB // fixes the constant term of cz^2 + (d-a)z - b = 0
	d := a - c*(rootA+rootB)
	for a*d-b*c == 0 { // degenerate maps are not Mobius transformations
		a++
		d = a - c*(rootA+rootB)
	}
	lo, hi := rootA, rootB
	if lo > hi {
		lo, hi = hi, lo
	}
	ans := strconv.Itoa(hi)
	question := fmt.Sprintf("The Moebius transformation is \\(f(z) = \\dfrac{%dz %s}{%dz %s}\\). Its fixed points solve \\(z = f(z)\\). Give the fixed point with the larger real part.",
		a, signed(b), c, signed(d))
	pb := generator.Problem{
		Question: question,
		Answer:   ans,
		Facts: map[string]string{
			"a": strconv.Itoa(a), "b": strconv.Itoa(b), "c": strconv.Itoa(c), "d": strconv.Itoa(d),
			"fz":    fmt.Sprintf("(%dz %s)/(%dz %s)", a, signed(b), c, signed(d)),
			"root1": strconv.Itoa(lo), "root2": strconv.Itoa(hi), "value": ans,
		},
	}
	pb.Explanation = fmt.Sprintf("Clearing the denominator gives %dz^2 %sz %s = 0, whose roots are %d and %d. The larger real part is %d.",
		c, signed(d-a), signed(-b), lo, hi, hi)
	return pb
}

// ----- 13. residue at a simple pole -----

type simpleResidueGen struct{}

func (g *simpleResidueGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	a := 1 + rand.Intn(4)
	p := rand.Intn(5) - 2
	q := 1 + rand.Intn(4)
	u := rand.Intn(7) - 3
	v := rand.Intn(7) - 3
	z0 := fmtComplex(p, q)
	re := a*p + u
	im := a*q + v
	ans := fmtComplex(re, im)
	q2 := fmt.Sprintf("The function is \\(f(z) = \\dfrac{%dz %s}{z - (%s)}\\). Find the residue of f at \\(z = %s\\).",
		a, fmtSignedComplex(u, v), z0, z0)
	pb := generator.Problem{
		Question: q2,
		Answer:   ans,
		Facts: map[string]string{
			"a": strconv.Itoa(a), "z0": z0, "constant": fmtSignedComplex(u, v),
			"fz":   fmt.Sprintf("(%dz %s)/(z - (%s))", a, fmtSignedComplex(u, v), z0),
			"real": strconv.Itoa(re), "imag": strconv.Itoa(im), "value": ans,
		},
	}
	pb.Explanation = fmt.Sprintf("The pole is simple, so the residue is \\(\\lim_{z \\to %s} (z - %s) f(z)\\), which is just %d(%s) %s = %s.",
		z0, z0, a, z0, fmtSignedComplex(u, v), ans)
	return pb
}

// fmtSignedComplex renders a complex constant with an explicit leading sign, for use inside a sum.
func fmtSignedComplex(r, i int) string {
	if i >= 0 {
		return fmt.Sprintf("%s + %di", signed(r), i)
	}
	return fmt.Sprintf("%s - %di", signed(r), -i)
}

// ----- 14. cross ratio -----

type crossRatioGen struct{}

func (g *crossRatioGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	// Four distinct real points, which makes the cross-ratio real.
	z1 := rand.Intn(3)
	z2 := z1 + 2 + rand.Intn(3)
	z3 := z2 + 1 + rand.Intn(3)
	z4 := z3 + 1 + rand.Intn(4)
	num := (z3 - z1) * (z4 - z2)
	den := (z3 - z2) * (z4 - z1)
	ans := fracStr(num, den)
	q := fmt.Sprintf("For the real points \\(z_1 = %d\\), \\(z_2 = %d\\), \\(z_3 = %d\\) and \\(z_4 = %d\\), compute the cross-ratio \\(\\dfrac{(z_3 - z_1)(z_4 - z_2)}{(z_3 - z_2)(z_4 - z_1)}\\).",
		z1, z2, z3, z4)
	p := generator.Problem{
		Question: q,
		Answer:   ans,
		Facts: map[string]string{
			"z1": strconv.Itoa(z1), "z2": strconv.Itoa(z2), "z3": strconv.Itoa(z3), "z4": strconv.Itoa(z4),
			"numerator": strconv.Itoa(num), "denominator": strconv.Itoa(den), "value": ans,
		},
	}
	p.Explanation = fmt.Sprintf("Substitute the four points: (%d - %d)(%d - %d) = %d over (%d - %d)(%d - %d) = %d, which is %s. The value is real, as it must be for four points on a line.",
		z3, z1, z4, z2, num, z3, z2, z4, z1, den, ans)
	return p
}

// ----- 15. open and closed sets -----

type openSetGen struct{}

func (g *openSetGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type entry struct{ set, class, why string }
	entries := []entry{
		{"{z : |z| < 3}", "open, not closed", "the boundary |z| = 3 is excluded, and every point inside has a small disk that stays inside"},
		{"{z : |z| \u2264 3}", "closed, not open", "the boundary circle is included as well as the interior, and an interior point can never move in every direction and stay inside"},
		{"{z : Re z > 0}", "open, not closed", "the boundary line Re z = 0 is excluded, and each point has room to its left and right"},
		{"{z : Re z \u2265 0}", "closed, not open", "the boundary line is included, and a point on it cannot move left at all"},
	}
	e := entries[rand.Intn(len(entries))]
	pr := generator.MultipleChoice(
		fmt.Sprintf("Classify the set \\(%s\\) in the complex plane.", e.set),
		e.class, "open, not closed", "closed, not open", "both open and closed", "neither open nor closed")
	pr.Facts = map[string]string{"set": e.set, "class": e.class, "why": e.why, "value": e.class}
	pr.Explanation = fmt.Sprintf("The set is %s: %s. Openness is about whether the boundary is included, which is a separate question from whether the set is all in one piece.", e.class, e.why)
	return pr
}

// ----- 16. domains -----

type domainGen struct{}

func (g *domainGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type entry struct{ set, verdict, why string }
	domains := []entry{
		{"the punctured plane C \\ {0}", "a domain",
			"it is open, because every point has a small disk avoiding 0, and connected, because any two points can be joined by a path that bends around 0"},
		{"the open unit disk", "a domain",
			"it is open, because its boundary |z| = 1 is excluded, and connected, because any two interior points can be joined by a straight segment inside it"},
		{"the upper half-plane", "a domain",
			"it is open, because its boundary, the real axis Im z = 0, is excluded, and connected, because a straight segment between two points with positive imaginary part stays there"},
		{"the annulus {z : 1 < |z| < 2}", "a domain",
			"it is open, because both boundary circles are excluded, and connected, because a path can go around the hole between radius 1 and radius 2"},
	}
	notDomains := []entry{
		{"the union of two disjoint open disks", "not a domain",
			"it is open, but it is not connected: it splits into two disjoint non-empty open pieces"},
		{"the closed unit disk", "not a domain",
			"it is connected, but it is not open: a point on |z| = 1 has no disk that stays inside"},
		{"the real axis", "not a domain",
			"it is connected, but it is not open in the complex plane: a point on the line has no disk that stays on the line"},
		{"{z : |z| \u2265 1}", "not a domain",
			"it is connected, but it is not open: the circle |z| = 1 belongs to it and has no disk that stays inside"},
	}
	answer := domains[rand.Intn(len(domains))]
	rand.Shuffle(len(notDomains), func(i, j int) { notDomains[i], notDomains[j] = notDomains[j], notDomains[i] })
	pr := generator.MultipleChoice(
		"Which of these sets is a domain, meaning an open connected subset of the complex plane?",
		answer.set, notDomains[0].set, notDomains[1].set, notDomains[2].set)
	pr.Facts = map[string]string{
		"value": answer.set, "verdict": answer.verdict, "why": answer.why,
		"distractor1": notDomains[0].set, "distractor2": notDomains[1].set, "distractor3": notDomains[2].set,
	}
	pr.Explanation = fmt.Sprintf("%s is %s: %s. The two requirements are independent, so a set that fails either one is not a domain.", answer.set, answer.verdict, answer.why)
	return pr
}
