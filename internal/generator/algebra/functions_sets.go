package algebra

// Functions and the sets they live on: interval notation, absolute value as
// distance, domains, inverses (including a restricted domain), composites
// and monotonicity.
//
// Each generator asks a different kind of question as difficulty rises
// rather than the same question with bigger numbers, and answers a set of
// reals in interval notation wherever the mathematics is a set. Those
// answers are graded as intervals (internal/grader/sets.go), so (1, oo),
// (1, inf) and (1, ∞) are the same answer and [1, ∞) is not.

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
	"github.com/chuma-beep/mathua/internal/mathutil"
)

// ---------------------------------------------------------------- helpers

func tierOf(d float64) int {
	switch {
	case d >= 0.7:
		return 2
	case d >= 0.45:
		return 1
	}
	return 0
}

// frac renders n/d reduced: "3", "-3/2".
func fracStr(n, d int) string {
	if d < 0 {
		n, d = -n, -d
	}
	g := mathutil.GCD(mathutil.Abs(n), d)
	if g == 0 {
		g = 1
	}
	n, d = n/g, d/g
	if d == 1 {
		return strconv.Itoa(n)
	}
	return fmt.Sprintf("%d/%d", n, d)
}

// ivl writes an interval in the stored-answer form: "[2, 5)", "(-∞, 3]".
func ivl(lo string, loIn bool, hi string, hiIn bool) string {
	l, r := "(", ")"
	if loIn {
		l = "["
	}
	if hiIn {
		r = "]"
	}
	return l + lo + ", " + hi + r
}

const inf, ninf = "∞", "-∞"

// texIvl is the same interval in LaTeX.
func texIvl(s string) string {
	return strings.NewReplacer("-∞", "-\\infty", "∞", "\\infty", "∪", "\\cup").Replace(s)
}

// lin writes a·x + b as a learner would: "3x - 2", "-x + 4", "x".
func lin(a, b int, v string) string {
	var s string
	switch a {
	case 0:
		return strconv.Itoa(b)
	case 1:
		s = v
	case -1:
		s = "-" + v
	default:
		s = strconv.Itoa(a) + v
	}
	switch {
	case b > 0:
		s += fmt.Sprintf(" + %d", b)
	case b < 0:
		s += fmt.Sprintf(" - %d", -b)
	}
	return s
}

// shift writes x - h as "x - 3", "x + 3" or "x".
func shift(v string, h int) string { return lin(1, -h, v) }

// plusc writes a constant term: " + 3", " - 3", or nothing for 0.
func plusc(n int) string {
	switch {
	case n > 0:
		return fmt.Sprintf(" + %d", n)
	case n < 0:
		return fmt.Sprintf(" - %d", -n)
	}
	return ""
}

// coef writes a positive coefficient, leaving out a 1: "3", "".
func coef(a int) string {
	if a == 1 {
		return ""
	}
	return strconv.Itoa(a)
}

func nonzero(lo, hi int) int {
	for {
		if v := lo + rand.Intn(hi-lo+1); v != 0 {
			return v
		}
	}
}

// gradeSetOrInterval grades an interval key as an interval, a roster key as
// a set, and anything else with the router under the concept's own type.
func gradeSetOrInterval(gt grader.GradingType, expected, answer string) grader.Result {
	switch {
	case grader.LooksLikeSet(expected):
		return grader.GradeSet(expected, answer)
	case grader.LooksLikeInterval(expected, false) && strings.ContainsAny(expected, "[]∞∪("):
		return grader.GradeInterval(expected, answer)
	}
	return grader.NewRouter().Grade(gt, expected, answer)
}

const intervalHint = "Answer in interval notation, e.g. (2, 5] or [0, ∞); join pieces with ∪ or U"

// ---------------------------------------------------------------- interval notation

func (g *ineqIntervalGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	facts := map[string]string{"rule": "yes"}
	var q, ans, exp string
	switch tierOf(ctx.Difficulty) {
	case 0:
		a := rand.Intn(9) - 4
		b := a + 2 + rand.Intn(6)
		type shape struct {
			ineq, ans string
		}
		shapes := []shape{
			{fmt.Sprintf("%d \\le x < %d", a, b), ivl(strconv.Itoa(a), true, strconv.Itoa(b), false)},
			{fmt.Sprintf("%d < x \\le %d", a, b), ivl(strconv.Itoa(a), false, strconv.Itoa(b), true)},
			{fmt.Sprintf("%d < x < %d", a, b), ivl(strconv.Itoa(a), false, strconv.Itoa(b), false)},
			{fmt.Sprintf("%d \\le x \\le %d", a, b), ivl(strconv.Itoa(a), true, strconv.Itoa(b), true)},
			{fmt.Sprintf("x > %d", a), ivl(strconv.Itoa(a), false, inf, false)},
			{fmt.Sprintf("x \\ge %d", a), ivl(strconv.Itoa(a), true, inf, false)},
			{fmt.Sprintf("x < %d", b), ivl(ninf, false, strconv.Itoa(b), false)},
			{fmt.Sprintf("x \\le %d", b), ivl(ninf, false, strconv.Itoa(b), true)},
		}
		s := shapes[rand.Intn(len(shapes))]
		q = fmt.Sprintf("Write the set \\(\\{x \\in \\mathbb{R} : %s\\}\\) in interval notation.", s.ineq)
		ans = s.ans
		exp = fmt.Sprintf("Read each end of \\(%s\\): ≤ or ≥ includes the endpoint (square bracket), < or > leaves it out (round bracket), and an end with no bound runs to ±∞. The set is %s.", s.ineq, ans)
	case 1:
		if rand.Intn(4) == 0 {
			if rand.Intn(2) == 0 {
				q = "Write \\(\\mathbb{R}_{+}\\), the set of nonnegative real numbers, in interval notation."
				ans = ivl("0", true, inf, false)
				exp = "Nonnegative means x ≥ 0, so 0 is included: ℝ₊ = [0, ∞). The strictly positive reals ℝ₊₊ = (0, ∞) leave 0 out."
			} else {
				q = "Write \\(\\mathbb{R}_{++}\\), the set of strictly positive real numbers, in interval notation."
				ans = ivl("0", false, inf, false)
				exp = "Strictly positive means x > 0, so 0 is left out: ℝ₊₊ = (0, ∞). The nonnegative reals ℝ₊ = [0, ∞) include it."
			}
			break
		}
		if rand.Intn(3) == 0 {
			// A double inequality lo < c·x + d ≤ hi with c > 0.
			c := 2 + rand.Intn(3)
			d := rand.Intn(11) - 5
			x1 := rand.Intn(7) - 4
			x2 := x1 + 1 + rand.Intn(5)
			lo, hi := c*x1+d, c*x2+d
			q = fmt.Sprintf("Solve \\(%d < %s \\le %d\\) and write the solution set in interval notation.", lo, lin(c, d, "x"), hi)
			ans = ivl(strconv.Itoa(x1), false, strconv.Itoa(x2), true)
			facts["solved"] = fmt.Sprintf("%d < x \\le %d", x1, x2)
			exp = fmt.Sprintf("Subtract %d from all three parts and divide by %d (positive, so the signs stay): %d < x ≤ %d, which is %s.", d, c, x1, x2, ans)
			break
		}
		c := nonzero(-5, 5)
		for c == 1 || c == -1 {
			c = nonzero(-5, 5)
		}
		d := rand.Intn(15) - 7
		r := rand.Intn(11) - 5
		e := c*r + d
		ops := []string{"<", "\\le", ">", "\\ge"}
		op := rand.Intn(4)
		// After dividing by c the direction flips when c < 0.
		res := op
		if c < 0 {
			res = op ^ 2
		}
		rs := strconv.Itoa(r)
		switch res {
		case 0:
			ans = ivl(ninf, false, rs, false)
		case 1:
			ans = ivl(ninf, false, rs, true)
		case 2:
			ans = ivl(rs, false, inf, false)
		default:
			ans = ivl(rs, true, inf, false)
		}
		q = fmt.Sprintf("Solve \\(%s %s %d\\) and write the solution set in interval notation.", lin(c, d, "x"), ops[op], e)
		facts["solved"] = fmt.Sprintf("x %s %d", ops[res], r)
		if c < 0 {
			facts["flip"] = strconv.Itoa(c)
			exp = fmt.Sprintf("Move %d across, then divide by %d. Dividing by a negative number reverses the inequality, so x %s %d, which is %s.", d, c, strings.TrimPrefix(ops[res], "\\"), r, ans)
		} else {
			exp = fmt.Sprintf("Move %d across, then divide by %d: x %s %d, which is %s.", d, c, strings.TrimPrefix(ops[res], "\\"), r, ans)
		}
	default:
		a := rand.Intn(9) - 6
		b := a + 3 + rand.Intn(5)
		c := a + 1 + rand.Intn(b-a-1)
		dd := b + 1 + rand.Intn(4)
		aIn, bIn, cIn, dIn := rand.Intn(2) == 0, rand.Intn(2) == 0, rand.Intn(2) == 0, rand.Intn(2) == 0
		I := ivl(strconv.Itoa(a), aIn, strconv.Itoa(b), bIn)
		J := ivl(strconv.Itoa(c), cIn, strconv.Itoa(dd), dIn)
		switch rand.Intn(3) {
		case 0:
			ans = ivl(strconv.Itoa(c), cIn, strconv.Itoa(b), bIn)
			q = fmt.Sprintf("Write \\(%s \\cap %s\\) in interval notation.", texIvl(I), texIvl(J))
			facts["intersect"] = "yes"
			exp = fmt.Sprintf("An intersection keeps what is in both: it starts at the larger left end, %d (bracket from %s), and stops at the smaller right end, %d (bracket from %s). So the set is %s.", c, J, b, I, ans)
		case 1:
			ans = ivl(strconv.Itoa(a), aIn, strconv.Itoa(dd), dIn)
			q = fmt.Sprintf("Write \\(%s \\cup %s\\) in interval notation.", texIvl(I), texIvl(J))
			facts["union"] = "yes"
			exp = fmt.Sprintf("The intervals overlap between %d and %d, so the union is one interval from the smaller left end, %d, to the larger right end, %d, each keeping its own bracket: %s.", c, b, a, dd, ans)
		default:
			left := ivl(ninf, false, strconv.Itoa(a), !aIn)
			right := ivl(strconv.Itoa(b), !bIn, inf, false)
			ans = left + " ∪ " + right
			q = fmt.Sprintf("Write the complement of \\(%s\\) in \\(\\mathbb{R}\\) in interval notation.", texIvl(I))
			facts["complement"] = "yes"
			exp = fmt.Sprintf("The complement is everything outside %s: below %d and above %d. An endpoint the interval included is now excluded, and the reverse, so the set is %s.", I, a, b, ans)
		}
	}
	facts["why"] = exp
	return generator.Problem{Question: q, Answer: ans, Explanation: exp, Facts: facts}
}

func (g *ineqIntervalGen) Grade(expected, answer string) grader.Result {
	return gradeSetOrInterval(grader.GradingMultipleChoice, expected, answer)
}

func (g *ineqIntervalGen) AnswerFormat() generator.AnswerFormat {
	return generator.AnswerFormat{Hint: intervalHint, InputMode: "text"}
}

// ---------------------------------------------------------------- absolute value

func (g *funcAbsValGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	p := absValProblem(ctx)
	p.Facts["why"] = p.Explanation
	return p
}

func absValProblem(ctx generator.GeneratorContext) generator.Problem {
	switch tierOf(ctx.Difficulty) {
	case 0:
		a := nonzero(-4, 4)
		b := rand.Intn(13) - 6
		x := rand.Intn(11) - 5
		in := a*x + b
		return generator.Problem{
			Question:    fmt.Sprintf("If \\(f(x) = |%s|\\), what is \\(f(%d)\\)?", lin(a, b, "x"), x),
			Answer:      strconv.Itoa(mathutil.Abs(in)),
			Explanation: fmt.Sprintf("Inside the bars first: %s at x = %d is %d. The absolute value is its distance from 0, %d.", lin(a, b, "x"), x, in, mathutil.Abs(in)),
			Facts:       map[string]string{"x": strconv.Itoa(x), "inner": strconv.Itoa(in), "abs": strconv.Itoa(mathutil.Abs(in))},
		}
	case 1:
		c := rand.Intn(13) - 6
		r := 1 + rand.Intn(6)
		strict := rand.Intn(2) == 0
		op := "<"
		if !strict {
			op = "\\le"
		}
		ans := ivl(strconv.Itoa(c-r), !strict, strconv.Itoa(c+r), !strict)
		return generator.Problem{
			Question: fmt.Sprintf("Write \\(\\{x \\in \\mathbb{R} : |%s| %s %d\\}\\) in interval notation.", shift("x", c), op, r),
			Answer:   ans,
			Explanation: fmt.Sprintf("|%s| is the distance from x to %d, so the set is every x within %d of %d: from %d to %d, %s.",
				shift("x", c), c, r, c, c-r, c+r, ans),
			Facts: map[string]string{"center": strconv.Itoa(c), "radius": strconv.Itoa(r), "lo": strconv.Itoa(c - r), "hi": strconv.Itoa(c + r)},
		}
	default:
		switch rand.Intn(3) {
		case 0:
			c := rand.Intn(13) - 6
			r := 1 + rand.Intn(6)
			ans := ivl(ninf, false, strconv.Itoa(c-r), false) + " ∪ " + ivl(strconv.Itoa(c+r), false, inf, false)
			return generator.Problem{
				Question: fmt.Sprintf("Write \\(\\{x \\in \\mathbb{R} : |%s| > %d\\}\\) in interval notation.", shift("x", c), r),
				Answer:   ans,
				Explanation: fmt.Sprintf("The x more than %d away from %d lie outside [%d, %d], on either side, with the endpoints excluded: %s.",
					r, c, c-r, c+r, ans),
				Facts: map[string]string{"center": strconv.Itoa(c), "radius": strconv.Itoa(r), "outside": "yes", "lo": strconv.Itoa(c - r), "hi": strconv.Itoa(c + r)},
			}
		case 1:
			// |a·x + b| ≤ e, with a > 1: divide through by a.
			a := 2 + rand.Intn(3)
			b := rand.Intn(13) - 6
			e := 1 + rand.Intn(9)
			lo, hi := fracStr(-e-b, a), fracStr(e-b, a)
			ans := ivl(lo, true, hi, true)
			return generator.Problem{
				Question: fmt.Sprintf("Write \\(\\{x \\in \\mathbb{R} : |%s| \\le %d\\}\\) in interval notation.", lin(a, b, "x"), e),
				Answer:   ans,
				Explanation: fmt.Sprintf("|%s| ≤ %d means %d ≤ %s ≤ %d. Subtract %d and divide by %d: %s ≤ x ≤ %s, so the set is %s.",
					lin(a, b, "x"), e, -e, lin(a, b, "x"), e, b, a, lo, hi, ans),
				Facts: map[string]string{"scaled": "yes", "lo": lo, "hi": hi},
			}
		default:
			c := rand.Intn(13) - 6
			r := 1 + rand.Intn(6)
			lo, hi := c-r, c+r
			return generator.Problem{
				Question: fmt.Sprintf("Find every solution of \\(|%s| = %d\\). Give the solution set.", shift("x", c), r),
				Answer:   fmt.Sprintf("{%d,%d}", lo, hi),
				Explanation: fmt.Sprintf("The points at distance exactly %d from %d are %d − %d = %d and %d + %d = %d, so the solution set is {%d, %d}.",
					r, c, c, r, lo, c, r, hi, lo, hi),
				Facts: map[string]string{"center": strconv.Itoa(c), "radius": strconv.Itoa(r), "equation": "yes", "lo": strconv.Itoa(lo), "hi": strconv.Itoa(hi)},
			}
		}
	}
}

func (g *funcAbsValGen) Grade(expected, answer string) grader.Result {
	return gradeSetOrInterval(grader.GradingSymbolic, expected, answer)
}

func (g *funcAbsValGen) AnswerFormat() generator.AnswerFormat {
	return generator.AnswerFormat{Hint: "Answer with a number, an interval such as (2, 5], or a set such as {-1, 3}", InputMode: "text"}
}

// ---------------------------------------------------------------- domain

func (g *domainConvertedGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	a := rand.Intn(13) - 6
	as := strconv.Itoa(a)
	facts := map[string]string{}
	var fn, ans, why string
	switch tierOf(ctx.Difficulty) {
	case 0:
		switch rand.Intn(4) {
		case 0:
			fn, ans = fmt.Sprintf("\\sqrt{%s}", shift("x", a)), ivl(as, true, inf, false)
			why = fmt.Sprintf("A square root needs a radicand that is not negative: %s ≥ 0, so x ≥ %d.", shift("x", a), a)
			facts["root"] = "yes"
		case 1:
			fn, ans = fmt.Sprintf("\\sqrt{%s}", lin(-1, a, "x")), ivl(ninf, false, as, true)
			why = fmt.Sprintf("The radicand must not be negative: %s ≥ 0, so x ≤ %d.", lin(-1, a, "x"), a)
			facts["root"] = "yes"
		case 2:
			fn, ans = fmt.Sprintf("\\frac{1}{%s}", shift("x", a)), ivl(ninf, false, as, false)+" ∪ "+ivl(as, false, inf, false)
			why = fmt.Sprintf("Division by zero is undefined, so x = %d is left out and every other real is allowed.", a)
			facts["pole"] = "yes"
		default:
			fn, ans = fmt.Sprintf("\\ln(%s)", shift("x", a)), ivl(as, false, inf, false)
			why = fmt.Sprintf("A logarithm needs a strictly positive argument: %s > 0, so x > %d.", shift("x", a), a)
			facts["log"] = "yes"
		}
	case 1:
		switch rand.Intn(3) {
		case 0:
			c := nonzero(-4, 4)
			for c == 1 || c == -1 {
				c = nonzero(-4, 4)
			}
			d := rand.Intn(13) - 6
			e := fracStr(-d, c)
			fn = fmt.Sprintf("\\sqrt{%s}", lin(c, d, "x"))
			if c > 0 {
				ans = ivl(e, true, inf, false)
				why = fmt.Sprintf("%s ≥ 0 gives x ≥ %s (dividing by a positive number).", lin(c, d, "x"), e)
			} else {
				ans = ivl(ninf, false, e, true)
				why = fmt.Sprintf("%s ≥ 0 gives x ≤ %s: dividing by the negative %d reverses the inequality.", lin(c, d, "x"), e, c)
			}
			facts["root"] = "yes"
		case 1:
			k := 1 + rand.Intn(5)
			ks := strconv.Itoa(k)
			fn = fmt.Sprintf("\\frac{1}{x^2 - %d}", k*k)
			ans = ivl(ninf, false, "-"+ks, false) + " ∪ " + ivl("-"+ks, false, ks, false) + " ∪ " + ivl(ks, false, inf, false)
			why = fmt.Sprintf("x² − %d = 0 at x = ±%d, and both points are removed; three pieces remain.", k*k, k)
			facts["pole"] = "yes"
		default:
			fn, ans = fmt.Sprintf("\\ln(%s)", lin(-1, a, "x")), ivl(ninf, false, as, false)
			why = fmt.Sprintf("The argument must be strictly positive: %s > 0, so x < %d.", lin(-1, a, "x"), a)
			facts["log"] = "yes"
		}
	default:
		b := a + 1 + rand.Intn(5)
		bs := strconv.Itoa(b)
		switch rand.Intn(4) {
		case 0:
			fn = fmt.Sprintf("\\frac{\\sqrt{%s}}{%s}", shift("x", a), shift("x", b))
			ans = ivl(as, true, bs, false) + " ∪ " + ivl(bs, false, inf, false)
			why = fmt.Sprintf("Two conditions: the radicand needs x ≥ %d, and the denominator needs x ≠ %d. Since %d > %d, removing it splits [%d, ∞) in two.", a, b, b, a, a)
			facts["root"], facts["pole"] = "yes", "yes"
		case 1:
			fn = fmt.Sprintf("\\sqrt{(%s)(%s)}", shift("x", a), shift("x", b))
			ans = ivl(ninf, false, as, true) + " ∪ " + ivl(bs, true, inf, false)
			why = fmt.Sprintf("The product (%s)(%s) is ≥ 0 when both factors have the same sign: x ≤ %d or x ≥ %d.", shift("x", a), shift("x", b), a, b)
			facts["root"] = "yes"
		case 2:
			fn = fmt.Sprintf("\\sqrt{%s} + \\ln(%s)", shift("x", a), lin(-1, b, "x"))
			ans = ivl(as, true, bs, false)
			why = fmt.Sprintf("Both parts must be defined: x ≥ %d for the root and x < %d for the logarithm. The domain is where both hold.", a, b)
			facts["root"], facts["log"] = "yes", "yes"
		default:
			k := 1 + rand.Intn(5)
			ks := strconv.Itoa(k)
			fn = fmt.Sprintf("\\ln(x^2 - %d)", k*k)
			ans = ivl(ninf, false, "-"+ks, false) + " ∪ " + ivl(ks, false, inf, false)
			why = fmt.Sprintf("x² − %d > 0 when |x| > %d: x < −%d or x > %d.", k*k, k, k, k)
			facts["log"] = "yes"
		}
	}
	facts["fn"] = fn
	facts["domain"] = ans
	facts["why"] = why
	return generator.Problem{
		Question:    fmt.Sprintf("Find the largest set of real numbers on which \\(f(x) = %s\\) is defined (its natural domain). Write it in interval notation.", fn),
		Answer:      ans,
		Explanation: why + " So the domain is " + ans + ".",
		Facts:       facts,
	}
}

func (g *domainConvertedGen) Grade(expected, answer string) grader.Result {
	return gradeSetOrInterval(grader.GradingNumeric, expected, answer)
}

func (g *domainConvertedGen) AnswerFormat() generator.AnswerFormat {
	return generator.AnswerFormat{Hint: intervalHint, InputMode: "text"}
}

// ---------------------------------------------------------------- inverse

func (g *funcInverseGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	facts := map[string]string{}
	var fn, ans, shown, steps string
	switch tierOf(ctx.Difficulty) {
	case 0:
		a := nonzero(-6, 6)
		for a == 1 {
			a = nonzero(-6, 6)
		}
		b := rand.Intn(15) - 7
		fn = lin(a, b, "x")
		switch {
		case b == 0 && a > 0:
			shown = fmt.Sprintf("x/%d", a)
		case b == 0:
			shown = fmt.Sprintf("-x/%d", -a)
		case a > 0:
			shown = fmt.Sprintf("(%s)/%d", lin(1, -b, "x"), a)
		default:
			shown = fmt.Sprintf("(%s)/%d", lin(-1, b, "x"), -a)
		}
		ans = shown
		steps = fmt.Sprintf("Write y = %s and solve for x: x = (y − (%d))/%d. Then rename y as x.", fn, b, a)
		facts["linear"] = "yes"
	case 1:
		h := nonzero(-5, 5)
		switch rand.Intn(3) {
		case 0:
			fn = "e^{x}" + plusc(h)
			ans = fmt.Sprintf("ln(%s)", shift("x", h))
			shown = ans
			steps = fmt.Sprintf("y = %s gives eˣ = %s, so x = ln(%s). The inverse is defined for x > %d, the range of f.", fn, shift("y", h), shift("y", h), h)
			facts["explog"] = "yes"
		case 1:
			fn = fmt.Sprintf("\\ln(%s)", shift("x", h))
			shown = lin(1, h, "e^x")
			ans = shown
			steps = fmt.Sprintf("y = ln(%s) gives %s = eʸ, so x = eʸ%s. The logarithm and the exponential undo each other.", shift("x", h), shift("x", h), plusc(h))
			facts["explog"] = "yes"
		default:
			k := nonzero(-4, 4)
			fn = fmt.Sprintf("\\frac{1}{%s}%s", shift("x", h), plusc(k))
			shown = fmt.Sprintf("1/(%s)%s", shift("x", k), plusc(h))
			ans = shown
			steps = fmt.Sprintf("y = 1/(%s)%s gives %s = 1/(%s), so %s = 1/(%s) and x = 1/(%s)%s. Then rename y as x.", shift("x", h), plusc(k), shift("y", k), shift("x", h), shift("x", h), shift("y", k), shift("y", k), plusc(h))
			facts["reciprocal"] = "yes"
		}
	default:
		h := rand.Intn(9) - 4
		k := rand.Intn(11) - 5
		sq := fmt.Sprintf("(%s)^2", shift("x", h))
		if h == 0 {
			sq = "x^2"
		}
		body := sq + plusc(k)
		if rand.Intn(2) == 0 {
			fn = fmt.Sprintf("%s, \\; x \\ge %d", body, h)
			shown = fmt.Sprintf("sqrt(%s)%s", shift("x", k), plusc(h))
			ans = shown
			steps = fmt.Sprintf("Write y = %s, so %s = (%s)². On x ≥ %d, %s ≥ 0, so take the positive square root: %s = √(%s) and x = %d + √(%s). Then rename y as x.", body, shift("y", k), shift("x", h), h, shift("x", h), shift("x", h), shift("y", k), h, shift("y", k))
		} else {
			fn = fmt.Sprintf("%s, \\; x \\le %d", body, h)
			shown = fmt.Sprintf("%d - sqrt(%s)", h, shift("x", k))
			if h == 0 {
				shown = fmt.Sprintf("-sqrt(%s)", shift("x", k))
			}
			ans = shown
			steps = fmt.Sprintf("Write y = %s, so %s = (%s)². On x ≤ %d, %s ≤ 0, so it is the negative square root: %s = −√(%s) and x = %d − √(%s). Without the restriction f is not one-to-one and has no inverse.", body, shift("y", k), shift("x", h), h, shift("x", h), shift("x", h), shift("y", k), h, shift("y", k))
		}
		facts["restricted"] = "yes"
		facts["k"] = strconv.Itoa(k)
	}
	facts["fn"] = fn
	facts["inverse"] = shown
	facts["why"] = steps
	return generator.Problem{
		Question:    fmt.Sprintf("Find the inverse of \\(f(x) = %s\\). Give \\(f^{-1}(x)\\).", fn),
		Answer:      ans,
		Explanation: steps + " So f⁻¹(x) = " + shown + ".",
		Facts:       facts,
	}
}

// ---------------------------------------------------------------- inverse domain

func (g *inverseDomainGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type item struct {
		q, correct, reason string
		wrong              []string
		facts              map[string]string
	}
	var it item
	switch tierOf(ctx.Difficulty) {
	case 0:
		m := nonzero(-4, 4)
		c := rand.Intn(11) - 5
		p := rand.Intn(5) - 2
		q := p + 1 + rand.Intn(4)
		fp, fq := m*p+c, m*q+c
		lo, hi := fp, fq
		if lo > hi {
			lo, hi = hi, lo
		}
		it = item{
			q:       fmt.Sprintf("\\(f(x) = %s\\) on the domain \\([%d, %d]\\). What is the domain of \\(f^{-1}\\)?", lin(m, c, "x"), p, q),
			correct: fmt.Sprintf("[%d, %d]", lo, hi),
			wrong:   []string{fmt.Sprintf("[%d, %d]", p, q), fmt.Sprintf("(%d, %d)", lo, hi), "(-∞, ∞)"},
			reason: fmt.Sprintf("The inverse takes the values of f as its inputs, so its domain is the range of f. A line is monotone, so the range runs between f(%d) = %d and f(%d) = %d: [%d, %d].",
				p, fp, q, fq, lo, hi),
			facts: map[string]string{"range": fmt.Sprintf("[%d, %d]", lo, hi)},
		}
		if lo == p && hi == q {
			it.wrong[0] = fmt.Sprintf("[%d, %d]", lo, hi+1)
		}
	case 1:
		b := nonzero(-5, 5)
		a := rand.Intn(9) - 4
		switch rand.Intn(3) {
		case 0:
			it = item{
				q:       fmt.Sprintf("\\(f(x) = \\sqrt{%s}%s\\) on its natural domain. What is the domain of \\(f^{-1}\\)?", shift("x", a), plusc(b)),
				correct: fmt.Sprintf("[%d, ∞)", b),
				wrong:   []string{fmt.Sprintf("[%d, ∞)", a), fmt.Sprintf("(%d, ∞)", b), "(-∞, ∞)"},
				reason:  fmt.Sprintf("√(%s) takes every value ≥ 0, so f takes every value ≥ %d. The domain of f⁻¹ is that range, [%d, ∞); the domain of f, [%d, ∞), becomes the range of f⁻¹.", shift("x", a), b, b, a),
				facts:   map[string]string{"range": fmt.Sprintf("[%d, ∞)", b)},
			}
		case 1:
			it = item{
				q:       fmt.Sprintf("\\(f(x) = e^{x}%s\\) for every real \\(x\\). What is the domain of \\(f^{-1}\\)?", plusc(b)),
				correct: fmt.Sprintf("(%d, ∞)", b),
				wrong:   []string{fmt.Sprintf("[%d, ∞)", b), "(0, ∞)", "(-∞, ∞)"},
				reason:  fmt.Sprintf("eˣ takes every positive value and never 0, so f takes every value above %d and never %d itself: the range, and so the domain of f⁻¹(x) = ln(x − (%d)), is (%d, ∞).", b, b, b, b),
				facts:   map[string]string{"range": fmt.Sprintf("(%d, ∞)", b)},
			}
		default:
			it = item{
				q:       fmt.Sprintf("\\(f(x) = \\frac{1}{%s}%s\\) for \\(x \\ne %d\\). What is the domain of \\(f^{-1}\\)?", shift("x", a), plusc(b), a),
				correct: fmt.Sprintf("(-∞, %d) ∪ (%d, ∞)", b, b),
				wrong:   []string{fmt.Sprintf("(-∞, %d) ∪ (%d, ∞)", a, a), fmt.Sprintf("(%d, ∞)", b), "(-∞, ∞)"},
				reason:  fmt.Sprintf("1/(%s) takes every value except 0, so f takes every value except %d. That range is the domain of f⁻¹.", shift("x", a), b),
				facts:   map[string]string{"range": fmt.Sprintf("(-∞, %d) ∪ (%d, ∞)", b, b)},
			}
		}
	default:
		h := nonzero(-4, 4)
		k := rand.Intn(11) - 5
		sq := fmt.Sprintf("(%s)^2%s", shift("x", h), plusc(k))
		if rand.Intn(2) == 0 {
			it = item{
				q:       fmt.Sprintf("\\(f(x) = %s\\) is not one-to-one on \\(\\mathbb{R}\\). Which restriction of its domain makes it one-to-one, with the largest such domain containing \\(x = %d\\)?", sq, h+1),
				correct: fmt.Sprintf("[%d, ∞)", h),
				wrong:   []string{"[0, ∞)", fmt.Sprintf("[%d, ∞)", k), fmt.Sprintf("[%d, ∞)", h-1)},
				reason:  fmt.Sprintf("The parabola turns at x = %d. On either side of the vertex it is monotone, so it is one-to-one on [%d, ∞) (the side containing %d) or on (−∞, %d]. Any interval that crosses %d contains two inputs with the same value.", h, h, h+1, h, h),
				facts:   map[string]string{"vertex": strconv.Itoa(h)},
			}
			if k == 0 || k == h-1 {
				it.wrong[1] = fmt.Sprintf("(%d, ∞)", h)
			}
			if h == 0 {
				it.wrong[0] = "(-∞, ∞)"
			}
		} else {
			it = item{
				q:       fmt.Sprintf("\\(f(x) = %s\\) restricted to \\(x \\ge %d\\). What is the domain of \\(f^{-1}\\)?", sq, h),
				correct: fmt.Sprintf("[%d, ∞)", k),
				wrong:   []string{fmt.Sprintf("[%d, ∞)", h), fmt.Sprintf("(%d, ∞)", k), "(-∞, ∞)"},
				reason:  fmt.Sprintf("On x ≥ %d the square (x − (%d))² takes every value ≥ 0, so f takes every value ≥ %d. That range, [%d, ∞), is the domain of f⁻¹(x) = %d + √(x − (%d)).", h, h, k, k, h, k),
				facts:   map[string]string{"range": fmt.Sprintf("[%d, ∞)", k)},
			}
			if h == k {
				it.wrong[0] = fmt.Sprintf("[%d, ∞)", k+1)
			}
		}
	}
	p := generator.MultipleChoice(it.q, it.correct, distinctWrong(it.correct, it.wrong)...)
	p.Explanation = it.reason
	p.Facts = it.facts
	p.Facts["answer"] = it.correct
	p.Facts["why"] = it.reason
	return p
}

// distinctWrong drops a distractor equal to the key or to another
// distractor, then tops the list up to three with intervals that are wrong
// for a reason a learner can make: the other bracket on the left end, or
// the whole line.
func distinctWrong(correct string, wrong []string) []string {
	seen := map[string]bool{correct: true}
	var out []string
	add := func(w string) {
		if !seen[w] && len(out) < 3 {
			seen[w] = true
			out = append(out, w)
		}
	}
	for _, w := range wrong {
		add(w)
	}
	swapped := correct
	switch {
	case strings.HasPrefix(correct, "["):
		swapped = "(" + correct[1:]
	case strings.HasPrefix(correct, "("):
		swapped = "[" + correct[1:]
	}
	if !strings.HasPrefix(swapped, "[-∞") {
		add(swapped)
	}
	add("(-∞, ∞)")
	add("(0, ∞)")
	add("[0, ∞)")
	return out
}

// ---------------------------------------------------------------- composite

func (g *funcCompositeGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	p := compositeProblem(ctx)
	p.Facts["why"] = p.Explanation
	return p
}

func compositeProblem(ctx generator.GeneratorContext) generator.Problem {
	facts := map[string]string{}
	switch tierOf(ctx.Difficulty) {
	case 0:
		a := nonzero(-3, 4)
		b := rand.Intn(9) - 4
		c := rand.Intn(9) - 4
		x := rand.Intn(7) - 3
		f := lin(a, b, "x")
		gq := lin(1, c, "x^2")
		if rand.Intn(2) == 0 {
			gx := x*x + c
			v := a*gx + b
			facts["inner"], facts["value"] = strconv.Itoa(gx), strconv.Itoa(v)
			return generator.Problem{
				Question:    fmt.Sprintf("Let \\(f(x) = %s\\) and \\(g(x) = %s\\). Find \\((f \\circ g)(%d) = f(g(%d))\\).", f, gq, x, x),
				Answer:      strconv.Itoa(v),
				Explanation: fmt.Sprintf("Inside out: g(%d) = %d, then f(%d) = %d.", x, gx, gx, v),
				Facts:       facts,
			}
		}
		fx := a*x + b
		v := fx*fx + c
		facts["inner"], facts["value"] = strconv.Itoa(fx), strconv.Itoa(v)
		return generator.Problem{
			Question:    fmt.Sprintf("Let \\(f(x) = %s\\) and \\(g(x) = %s\\). Find \\((g \\circ f)(%d) = g(f(%d))\\).", f, gq, x, x),
			Answer:      strconv.Itoa(v),
			Explanation: fmt.Sprintf("Inside out: f(%d) = %d, then g(%d) = %d. The order matters: g∘f applies f first.", x, fx, fx, v),
			Facts:       facts,
		}
	case 1:
		a := nonzero(-3, 3)
		b := rand.Intn(9) - 4
		c := rand.Intn(9) - 4
		inner := lin(a, b, "x")
		if rand.Intn(2) == 0 {
			ans := fmt.Sprintf("(%s)^2%s", inner, plusc(c))
			facts["inner"], facts["value"] = inner, fmt.Sprintf("(%s)²%s", inner, plusc(c))
			return generator.Problem{
				Question:    fmt.Sprintf("Let \\(f(x) = %s\\) and \\(g(x) = %s\\). Find \\(f(g(x))\\).", lin(1, c, "x^2"), inner),
				Answer:      ans,
				Explanation: fmt.Sprintf("Replace every x in f by g(x) = %s: f(g(x)) = (%s)²%s. Expanding is optional.", inner, inner, plusc(c)),
				Facts:       facts,
			}
		}
		ans := fmt.Sprintf("%d(%s)%s", a, lin(1, c, "x^2"), plusc(b))
		if a == 1 {
			ans = lin(1, c+b, "x^2")
		} else if a == -1 {
			ans = fmt.Sprintf("-(%s)%s", lin(1, c, "x^2"), plusc(b))
		}
		facts["inner"], facts["value"] = lin(1, c, "x^2"), fmt.Sprintf("%d(x²%s)%s", a, plusc(c), plusc(b))
		return generator.Problem{
			Question:    fmt.Sprintf("Let \\(f(x) = %s\\) and \\(g(x) = %s\\). Find \\(f(g(x))\\).", inner, lin(1, c, "x^2")),
			Answer:      ans,
			Explanation: fmt.Sprintf("Replace x in f by g(x) = %s: f(g(x)) = %d(x²%s)%s = %s.", lin(1, c, "x^2"), a, plusc(c), plusc(b), lin(a, a*c+b, "x^2")),
			Facts:       facts,
		}
	default:
		a := rand.Intn(11) - 5
		as := strconv.Itoa(a)
		var q, ans, why string
		switch rand.Intn(4) {
		case 0:
			q = fmt.Sprintf("Let \\(f(x) = \\sqrt{x}\\) and \\(g(x) = %s\\). Find the domain of \\(f \\circ g\\).", shift("x", a))
			ans = ivl(as, true, inf, false)
			why = fmt.Sprintf("f∘g(x) = √(%s) needs g(x) in the domain of f, [0, ∞): %s ≥ 0, so x ≥ %d.", shift("x", a), shift("x", a), a)
		case 1:
			k := 1 + rand.Intn(5)
			ks := strconv.Itoa(k)
			q = fmt.Sprintf("Let \\(f(x) = \\sqrt{x}\\) and \\(g(x) = %d - x^2\\). Find the domain of \\(f \\circ g\\).", k*k)
			ans = ivl("-"+ks, true, ks, true)
			why = fmt.Sprintf("g(x) must land in [0, ∞): %d − x² ≥ 0, so x² ≤ %d and −%d ≤ x ≤ %d.", k*k, k*k, k, k)
		case 2:
			q = fmt.Sprintf("Let \\(f(x) = \\ln x\\) and \\(g(x) = %s\\). Find the domain of \\(f \\circ g\\).", shift("x", a))
			ans = ivl(as, false, inf, false)
			why = fmt.Sprintf("ln needs a positive input, so g(x) = %s > 0: x > %d.", shift("x", a), a)
		default:
			q = fmt.Sprintf("Let \\(f(x) = \\frac{1}{x}\\) and \\(g(x) = %s\\). Find the domain of \\(f \\circ g\\).", shift("x", a))
			ans = ivl(ninf, false, as, false) + " ∪ " + ivl(as, false, inf, false)
			why = fmt.Sprintf("f is undefined at 0, so g(x) = %s must not be 0: x ≠ %d.", shift("x", a), a)
		}
		facts["domain"] = ans
		return generator.Problem{Question: q, Answer: ans, Explanation: why + " The domain of f∘g is " + ans + ".", Facts: facts}
	}
}

func (g *funcCompositeGen) Grade(expected, answer string) grader.Result {
	return gradeSetOrInterval(grader.GradingSymbolic, expected, answer)
}

func (g *funcCompositeGen) AnswerFormat() generator.AnswerFormat {
	return generator.AnswerFormat{Hint: "Answer with a number, an expression in x, or an interval such as [2, ∞)", InputMode: "text"}
}

// ---------------------------------------------------------------- monotonicity

func (g *monotonicityGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	word := func(up bool) string {
		if up {
			return "increasing"
		}
		return "decreasing"
	}
	a := rand.Intn(5) + 1
	b := rand.Intn(9) - 4
	var fn, domain, why, dir string
	switch tierOf(ctx.Difficulty) {
	case 0:
		if rand.Intn(2) == 0 {
			a = -a
		}
		fn, domain, dir = lin(a, b, "x"), "\\mathbb{R}", word(a > 0)
		why = fmt.Sprintf("a line with slope %d; the intercept does not change the direction", a)
	case 1:
		switch rand.Intn(4) {
		case 0:
			base := []string{"2", "3", "5", "e"}[rand.Intn(4)]
			if rand.Intn(2) == 0 {
				fn, domain, dir = base+"^{x}", "\\mathbb{R}", "increasing"
				why = fmt.Sprintf("an exponential with base %s > 1 grows", base)
			} else {
				fn, domain, dir = base+"^{-x}", "\\mathbb{R}", "decreasing"
				why = fmt.Sprintf("%s^(−x) = (1/%s)^x, an exponential with base below 1, which decays", base, base)
			}
		case 1:
			fn, domain, dir = fmt.Sprintf("\\frac{%d}{x}", a), "(0, \\infty)", "decreasing"
			why = fmt.Sprintf("for 0 < x₁ < x₂, %d/x₁ > %d/x₂: a larger denominator gives a smaller value", a, a)
		case 2:
			if rand.Intn(2) == 0 {
				fn, domain, dir = coef(a)+"\\ln x", "(0, \\infty)", "increasing"
				why = "ln x rises on (0, ∞), and a positive multiple keeps the direction"
			} else {
				fn, domain, dir = "-"+coef(a)+"\\ln x", "(0, \\infty)", "decreasing"
				why = "ln x rises on (0, ∞), and a negative multiple reverses the direction"
			}
		default:
			if rand.Intn(2) == 0 {
				fn, domain, dir = coef(a)+"x^{3}", "\\mathbb{R}", "increasing"
				why = "x³ preserves order: x₁ < x₂ gives x₁³ < x₂³, and a positive multiple keeps it"
			} else {
				fn, domain, dir = "-"+coef(a)+"x^{3}", "\\mathbb{R}", "decreasing"
				why = "x³ preserves order, and a negative multiple reverses it"
			}
		}
	default:
		h := rand.Intn(9) - 4
		sq := fmt.Sprintf("(%s)^{2}", shift("x", h))
		if h == 0 {
			sq = "x^{2}"
		}
		sign := 1
		if rand.Intn(2) == 0 {
			sign = -1
		}
		if sign < 0 {
			fn = "-" + sq
		} else {
			fn = sq
		}
		fn += plusc(b)
		switch rand.Intn(3) {
		case 0:
			domain = fmt.Sprintf("[%d, \\infty)", h)
			dir = word(sign > 0)
			why = fmt.Sprintf("the vertex is at x = %d; to its right the square %s as x moves away from %d", h, map[bool]string{true: "grows", false: "grows, and the minus sign makes f fall"}[sign > 0], h)
		case 1:
			domain = fmt.Sprintf("(-\\infty, %d]", h)
			dir = word(sign < 0)
			why = fmt.Sprintf("the vertex is at x = %d; moving right towards it the square shrinks, so f %s", h, map[bool]string{true: "falls", false: "rises (the minus sign flips it)"}[sign > 0])
		default:
			lo, hi := h-1-rand.Intn(3), h+1+rand.Intn(3)
			domain = fmt.Sprintf("[%d, %d]", lo, hi)
			dir = "neither"
			why = fmt.Sprintf("the interval contains the vertex x = %d, so f %s on one side of it and %s on the other", h, map[bool]string{true: "falls", false: "rises"}[sign > 0], map[bool]string{true: "rises", false: "falls"}[sign > 0])
		}
	}
	return generator.Problem{
		Question:    fmt.Sprintf("Is \\(f(x) = %s\\) increasing, decreasing, or neither on \\(%s\\)?", fn, domain),
		Answer:      dir,
		Explanation: fmt.Sprintf("On this domain, %s. So f is %s.", why, dir),
		Facts:       map[string]string{"fn": fn, "domain": domain, "why": why, "dir": dir},
	}
}
