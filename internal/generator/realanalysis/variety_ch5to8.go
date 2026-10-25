package realanalysis

// Concrete instances for Abbott chapters 5 to 8: derivatives, sequences of
// functions, the integral, and metric spaces. The numeric variants publish a
// kind fact and the parameters, and batch2_test.go recomputes each kind
// independently.

import (
	"math"
	"math/big"
	"math/rand"
	"strconv"

	"github.com/chuma-beep/mathua/internal/generator"
)

// ---------------------------------------------------------------------------
// ra.deriv.mvt (numeric)
// ---------------------------------------------------------------------------

func mvtVariant() generator.Problem {
	switch rand.Intn(3) {
	case 0:
		a := randIn(-4, 4)
		b := a + randIn(1, 6)
		p := randNonzero(-3, 4)
		q := randIn(-5, 5)
		c := rq(int64(a+b), 2)
		answer := ratAnswer(c)
		f := coefX(p, "x²")
		if q != 0 {
			f += signedTerm(q) + "x"
		}
		return generator.Problem{
			Question: "For f(x) = " + minus(f) + " on [" + itoa(a) + ", " + itoa(b) +
				"], the mean value theorem gives a point c with f'(c) equal to the average slope. What is c? (enter a decimal)",
			Answer: answer,
			Explanation: "The average slope is (f(b) − f(a))/(b − a) = " + itoa(p) + "(a + b)" + signedTerm(q) + ", and f'(c) = " + itoa(2*p) + "c" + signedTerm(q) +
				". Setting them equal gives c = (a + b)/2 = " + answer + ".",
			Facts: map[string]string{"kind": "quad", "a": strconv.Itoa(a), "b": strconv.Itoa(b), "answer": answer},
		}
	case 1:
		pr := pick([][2]int{{1, 4}, {1, 9}, {2, 8}, {4, 9}, {1, 16}, {3, 12}, {4, 16}, {2, 18}, {9, 16}, {1, 25}})
		a, b := pr[0], pr[1]
		c := int(math.Round(math.Sqrt(float64(a * b))))
		answer := strconv.Itoa(c)
		return generator.Problem{
			Question: "For f(x) = 1/x on [" + strconv.Itoa(a) + ", " + strconv.Itoa(b) +
				"], the mean value theorem gives a point c with f'(c) equal to the average slope. What is c? (enter a decimal)",
			Answer: answer,
			Explanation: "The average slope is (1/" + strconv.Itoa(b) + " − 1/" + strconv.Itoa(a) + ")/(" + strconv.Itoa(b) + " − " + strconv.Itoa(a) + ") = −1/" + strconv.Itoa(a*b) +
				", and f'(c) = −1/c², so c² = " + strconv.Itoa(a*b) + " and c = " + answer + ", which lies in the interval.",
			Facts: map[string]string{"kind": "recip", "a": strconv.Itoa(a), "b": strconv.Itoa(b), "answer": answer},
		}
	default:
		s := randIn(0, 4)
		t := s + randIn(1, 4)
		a, b := s*s, t*t
		c := rq(int64((s+t)*(s+t)), 4)
		answer := ratAnswer(c)
		return generator.Problem{
			Question: "For f(x) = √x on [" + strconv.Itoa(a) + ", " + strconv.Itoa(b) +
				"], the mean value theorem gives a point c with f'(c) equal to the average slope. What is c? (enter a decimal)",
			Answer: answer,
			Explanation: "The average slope is (" + strconv.Itoa(t) + " − " + strconv.Itoa(s) + ")/(" + strconv.Itoa(b) + " − " + strconv.Itoa(a) + ") = 1/" + strconv.Itoa(s+t) +
				", and f'(c) = 1/(2√c), so √c = " + ratDec(rq(int64(s+t), 2)) + " and c = " + answer + ".",
			Facts: map[string]string{"kind": "sqrt", "a": strconv.Itoa(a), "b": strconv.Itoa(b), "answer": answer},
		}
	}
}

// ---------------------------------------------------------------------------
// ra.deriv.chain (numeric)
// ---------------------------------------------------------------------------

func chainVariant() generator.Problem {
	a := randIn(1, 4)
	b := randIn(1, 4)
	n := randIn(2, 5)
	x0 := randIn(-2, 2)
	inner := a*x0 + b
	answer := decimal6(float64(n * a * raIntPow(inner, n-1)))
	return generator.Problem{
		Question: "Let f(x) = (" + coefX(a, "x") + " + " + strconv.Itoa(b) + ")^" + strconv.Itoa(n) +
			". Using the chain rule, what is f'(" + itoa(x0) + ")? (enter a decimal)",
		Answer: answer,
		Explanation: "f'(x) = " + strconv.Itoa(n) + "(" + coefX(a, "x") + " + " + strconv.Itoa(b) + ")^" + strconv.Itoa(n-1) + "·" + strconv.Itoa(a) +
			". At x = " + itoa(x0) + " the inner value is " + itoa(inner) + ", so f'(" + itoa(x0) + ") = " + strconv.Itoa(n) + "·" + strconv.Itoa(a) + "·" +
			parenNeg(inner) + "^" + strconv.Itoa(n-1) + " = " + minus(answer) + ".",
		Facts: map[string]string{
			"a": strconv.Itoa(a), "b": strconv.Itoa(b), "n": strconv.Itoa(n), "x0": strconv.Itoa(x0), "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// ra.deriv.lhopital (numeric)
// ---------------------------------------------------------------------------

func lhopitalVariant() generator.Problem {
	a := randIn(1, 9)
	b := randIn(1, 6)
	as, bs := strconv.Itoa(a), strconv.Itoa(b)
	kind := pick([]string{"cos", "sin", "exp", "log"})
	var q, why string
	var v *big.Rat
	switch kind {
	case "cos":
		v = rq(int64(a*a), int64(2*b))
		q = "(1 − cos(" + coefX(a, "x") + "))/(" + coefX(b, "x²") + ")"
		why = "Both parts tend to 0. One application of L'Hôpital gives " + as + "·sin(" + coefX(a, "x") + ")/(" + coefX(2*b, "x") + "), still 0/0; a second gives " +
			strconv.Itoa(a*a) + "·cos(" + coefX(a, "x") + ")/" + strconv.Itoa(2*b) + ", which tends to " + ratS(v) + "."
	case "sin":
		v = rq(int64(a), int64(b))
		q = "sin(" + coefX(a, "x") + ")/(" + coefX(b, "x") + ")"
		why = "Both parts tend to 0, and the quotient of derivatives is " + as + "·cos(" + coefX(a, "x") + ")/" + bs + ", which tends to " + ratS(v) + "."
	case "exp":
		v = rq(int64(a), int64(b))
		q = "(e^(" + coefX(a, "x") + ") − 1)/(" + coefX(b, "x") + ")"
		why = "Both parts tend to 0, and the quotient of derivatives is " + as + "·e^(" + coefX(a, "x") + ")/" + bs + ", which tends to " + ratS(v) + "."
	default:
		v = rq(int64(a), int64(b))
		q = "ln(1 + " + coefX(a, "x") + ")/(" + coefX(b, "x") + ")"
		why = "Both parts tend to 0, and the quotient of derivatives is (" + as + "/(1 + " + coefX(a, "x") + "))/" + bs + ", which tends to " + ratS(v) + "."
	}
	answer := ratAnswer(v)
	return generator.Problem{
		Question:    "Using L'Hôpital's rule, find the limit of " + q + " as x tends to 0. (enter a decimal)",
		Answer:      answer,
		Explanation: why,
		Facts:       map[string]string{"kind": kind, "a": as, "b": bs, "answer": answer},
	}
}

// ---------------------------------------------------------------------------
// ra.deriv.riemann and ra.deriv.ftc (numeric)
// ---------------------------------------------------------------------------

func riemannVariant() generator.Problem {
	b := randIn(1, 6)
	n := pick([]int{2, 4, 5, 8, 10, 20})
	answer := decimal6(float64(b*b) * float64(n+1) / (2 * float64(n)))
	return generator.Problem{
		Question: "Use the right-endpoint Riemann sum with n = " + strconv.Itoa(n) + " equal subintervals to approximate the integral of f(x) = x on the interval from 0 to " +
			strconv.Itoa(b) + ". What is the value of the sum? (enter a decimal)",
		Answer: answer,
		Explanation: "With step " + strconv.Itoa(b) + "/" + strconv.Itoa(n) + " the sum is Σ (i·b/n)(b/n) for i from 1 to n, which equals b²(n + 1)/(2n) = " + answer +
			". It overestimates the exact value b²/2 = " + ratAnswer(rq(int64(b*b), 2)) + " because x is increasing.",
		Facts: map[string]string{"b": strconv.Itoa(b), "n": strconv.Itoa(n), "answer": answer},
	}
}

func ftcVariant() generator.Problem {
	a := randIn(1, 8)
	c := randIn(0, 7)
	b := randIn(1, 6)
	answer := decimal6(float64(a)*float64(b*b)/2 + float64(c*b))
	integrand := coefX(a, "t")
	if c != 0 {
		integrand += " + " + strconv.Itoa(c)
	}
	return generator.Problem{
		Question: "Using the fundamental theorem of calculus, evaluate the integral from 0 to " + strconv.Itoa(b) + " of (" + integrand + ") dt. (enter a decimal)",
		Answer:   answer,
		Explanation: "F(t) = " + halfCoef(a, "t²") + func() string {
			if c == 0 {
				return ""
			}
			return " + " + strconv.Itoa(c) + "t"
		}() + " has F' equal to the integrand, so the integral is F(" + strconv.Itoa(b) + ") − F(0) = " + answer + ".",
		Facts: map[string]string{"a": strconv.Itoa(a), "c": strconv.Itoa(c), "b": strconv.Itoa(b), "answer": answer},
	}
}

// ---------------------------------------------------------------------------
// ra.deriv.parts (numeric)
// ---------------------------------------------------------------------------

func partsVariant() generator.Problem {
	switch rand.Intn(3) {
	case 0:
		b := randIn(2, 5)
		v := float64(b-1)*math.Exp(float64(b)) + 1
		answer := decimal6(v)
		return generator.Problem{
			Question: "Use integration by parts to evaluate the integral from 0 to " + strconv.Itoa(b) + " of x·e^x dx. (enter a decimal)",
			Answer:   answer,
			Explanation: "With u = x and dv = e^x dx, the integral is [x·e^x] − ∫ e^x dx = [x·e^x − e^x] from 0 to " + strconv.Itoa(b) +
				", which is " + strconv.Itoa(b-1) + "e^" + strconv.Itoa(b) + " + 1 ≈ " + answer + ".",
			Facts: map[string]string{"kind": "xexp", "b": strconv.Itoa(b), "answer": answer},
		}
	case 1:
		b := randIn(2, 8)
		v := float64(b)*math.Log(float64(b)) - float64(b) + 1
		answer := decimal6(v)
		return generator.Problem{
			Question: "Use integration by parts to evaluate the integral from 1 to " + strconv.Itoa(b) + " of ln x dx. (enter a decimal)",
			Answer:   answer,
			Explanation: "With u = ln x and dv = dx, the integral is [x ln x] − ∫ 1 dx = [x ln x − x] from 1 to " + strconv.Itoa(b) +
				", which is " + strconv.Itoa(b) + " ln " + strconv.Itoa(b) + " − " + strconv.Itoa(b-1) + " ≈ " + answer + ".",
			Facts: map[string]string{"kind": "ln", "b": strconv.Itoa(b), "answer": answer},
		}
	default:
		k := randIn(2, 4)
		kf := float64(k)
		v := math.Exp(kf)*(kf-1)/(kf*kf) + 1/(kf*kf)
		answer := decimal6(v)
		ks := strconv.Itoa(k)
		return generator.Problem{
			Question: "Use integration by parts to evaluate the integral from 0 to 1 of x·e^(" + ks + "x) dx. (enter a decimal)",
			Answer:   answer,
			Explanation: "With u = x and dv = e^(" + ks + "x) dx, the integral is [x·e^(" + ks + "x)/" + ks + " − e^(" + ks + "x)/" + strconv.Itoa(k*k) +
				"] from 0 to 1, which is " + strconv.Itoa(k-1) + "e^" + ks + "/" + strconv.Itoa(k*k) + " + 1/" + strconv.Itoa(k*k) + " ≈ " + answer + ".",
			Facts: map[string]string{"kind": "xexpk", "k": ks, "answer": answer},
		}
	}
}

// ---------------------------------------------------------------------------
// ra.deriv.improper (numeric)
// ---------------------------------------------------------------------------

func improperVariant() generator.Problem {
	switch rand.Intn(3) {
	case 0:
		p := randIn(2, 12)
		v := rq(1, int64(p-1))
		answer := ratAnswer(v)
		ps := strconv.Itoa(p)
		return generator.Problem{
			Question: "Evaluate the improper integral from 1 to ∞ of x^(−" + ps + ") dx. (enter a decimal)",
			Answer:   answer,
			Explanation: "The integral from 1 to t is (1 − t^(−" + strconv.Itoa(p-1) + "))/" + strconv.Itoa(p-1) + ", which tends to 1/" + strconv.Itoa(p-1) +
				" as t → ∞ because the exponent is negative.",
			Facts: map[string]string{"kind": "p", "p": ps, "a": "1", "answer": answer},
		}
	case 1:
		p := randIn(2, 4)
		a := randIn(2, 3)
		den := int64(p - 1)
		for i := 0; i < p-1; i++ {
			den *= int64(a)
		}
		v := rq(1, den)
		answer := ratAnswer(v)
		ps, as := strconv.Itoa(p), strconv.Itoa(a)
		return generator.Problem{
			Question: "Evaluate the improper integral from " + as + " to ∞ of x^(−" + ps + ") dx. (enter a decimal)",
			Answer:   answer,
			Explanation: "An antiderivative is −x^(−" + strconv.Itoa(p-1) + ")/" + strconv.Itoa(p-1) + ", which tends to 0 at ∞, so the value is " + as + "^(−" + strconv.Itoa(p-1) +
				")/" + strconv.Itoa(p-1) + " = " + ratS(v) + ".",
			Facts: map[string]string{"kind": "p", "p": ps, "a": as, "answer": answer},
		}
	default:
		k := randIn(2, 10)
		v := rq(1, int64(k))
		answer := ratAnswer(v)
		ks := strconv.Itoa(k)
		return generator.Problem{
			Question:    "Evaluate the improper integral from 0 to ∞ of e^(−" + coefX(k, "x") + ") dx. (enter a decimal)",
			Answer:      answer,
			Explanation: "The integral from 0 to t is (1 − e^(−" + coefX(k, "t") + "))/" + ks + ", which tends to 1/" + ks + " as t → ∞.",
			Facts:       map[string]string{"kind": "exp", "k": ks, "answer": answer},
		}
	}
}

// ---------------------------------------------------------------------------
// ra.deriv.power_series (numeric)
// ---------------------------------------------------------------------------

func powerSeriesVariant() generator.Problem {
	c := randIn(2, 10)
	cs := strconv.Itoa(c)
	switch rand.Intn(4) {
	case 0:
		answer := decimal6(1 / float64(c))
		return generator.Problem{
			Question:    "What is the radius of convergence of the power series Σ " + cs + "^n x^n? (enter a decimal)",
			Answer:      answer,
			Explanation: "The ratio test gives the limit |" + cs + "x|, which is below 1 exactly when |x| < 1/" + cs + ", so the radius is " + answer + ".",
			Facts:       map[string]string{"kind": "mul", "c": cs, "answer": answer},
		}
	case 1:
		return generator.Problem{
			Question:    "What is the radius of convergence of the power series Σ x^n/" + cs + "^n? (enter a decimal)",
			Answer:      cs,
			Explanation: "The ratio test gives the limit |x|/" + cs + ", which is below 1 exactly when |x| < " + cs + ", so the radius is " + cs + ".",
			Facts:       map[string]string{"kind": "div", "c": cs, "answer": cs},
		}
	case 2:
		return generator.Problem{
			Question: "What is the radius of convergence of the power series Σ n·x^n/" + cs + "^n? (enter a decimal)",
			Answer:   cs,
			Explanation: "The ratio of consecutive terms is ((n + 1)/n)·|x|/" + cs + ", which tends to |x|/" + cs +
				"; the factor n does not change the radius, which is " + cs + ".",
			Facts: map[string]string{"kind": "div", "c": cs, "answer": cs},
		}
	default:
		a := randNonzero(-4, 4)
		return generator.Problem{
			Question: "What is the radius of convergence of the power series Σ (" + xPlus(-a) + ")^n/(n·" + cs + "^n)? (enter a decimal)",
			Answer:   cs,
			Explanation: "The ratio of consecutive terms tends to |" + xPlus(-a) + "|/" + cs + ", so the series converges when |" + xPlus(-a) + "| < " + cs +
				". The centre " + itoa(a) + " shifts the interval but not the radius, which is " + cs + ".",
			Facts: map[string]string{"kind": "div", "c": cs, "answer": cs},
		}
	}
}

// ---------------------------------------------------------------------------
// ra.deriv.taylor (numeric)
// ---------------------------------------------------------------------------

func factorial(n int) int64 {
	f := int64(1)
	for i := 2; i <= n; i++ {
		f *= int64(i)
	}
	return f
}

func taylorVariant() generator.Problem {
	a := randIn(2, 5)
	as := strconv.Itoa(a)
	pow := func(k int) int64 { return int64(raIntPow(a, k)) }
	var fn, why string
	var k int
	var v *big.Rat
	kind := pick([]string{"exp", "sin", "cos", "geom", "log"})
	switch kind {
	case "exp":
		k = randIn(2, 4)
		v = rq(pow(k), factorial(k))
		fn = "e^(" + coefX(a, "x") + ")"
		why = "The k-th derivative of e^(ax) at 0 is a^k, so the x^k coefficient is a^k/k! = " + as + "^" + strconv.Itoa(k) + "/" + strconv.Itoa(int(factorial(k))) + "."
	case "sin":
		k = 3
		v = rq(-pow(3), 6)
		fn = "sin(" + coefX(a, "x") + ")"
		why = "sin u = u − u³/6 + …, and substituting u = " + coefX(a, "x") + " gives the x³ coefficient −" + as + "³/6."
	case "cos":
		k = 2
		v = rq(-pow(2), 2)
		fn = "cos(" + coefX(a, "x") + ")"
		why = "cos u = 1 − u²/2 + …, and substituting u = " + coefX(a, "x") + " gives the x² coefficient −" + as + "²/2."
	case "geom":
		k = randIn(2, 4)
		v = rq(pow(k), 1)
		fn = "1/(1 − " + coefX(a, "x") + ")"
		why = "1/(1 − u) = 1 + u + u² + … for |u| < 1, and substituting u = " + coefX(a, "x") + " gives the x^k coefficient " + as + "^k."
	default:
		k = randIn(2, 4)
		sign := int64(1)
		if k%2 == 0 {
			sign = -1
		}
		v = rq(sign*pow(k), int64(k))
		fn = "ln(1 + " + coefX(a, "x") + ")"
		why = "ln(1 + u) = u − u²/2 + u³/3 − …, and substituting u = " + coefX(a, "x") + " gives the x^k coefficient (−1)^(k+1)·" + as + "^k/k."
	}
	answer := ratAnswer(v)
	ks := strconv.Itoa(k)
	xk := "x^" + ks
	if k == 2 {
		xk = "x²"
	} else if k == 3 {
		xk = "x³"
	}
	return generator.Problem{
		Question:    "What is the coefficient of " + xk + " in the Taylor series of " + fn + " at 0? (enter a decimal)",
		Answer:      answer,
		Explanation: why + " Here that is " + ratS(v) + ".",
		Facts:       map[string]string{"kind": kind, "a": as, "k": ks, "answer": answer},
	}
}

// ---------------------------------------------------------------------------
// ra.metric.contraction (numeric)
// ---------------------------------------------------------------------------

func contractionVariant() generator.Problem {
	pd := randIn(2, 10)
	pn := randIn(1, pd-1)
	if g := gcd(pn, pd); g > 1 {
		pn, pd = pn/g, pd/g
	}
	if rand.Intn(3) == 0 {
		pn = -pn
	}
	b := randNonzero(-6, 9)
	a := rq(int64(pn), int64(pd))
	fixed := rquo(ri(int64(b)), rsub(ri(1), a))
	answer := decimal6(float64(b) / (1 - float64(pn)/float64(pd)))
	return generator.Problem{
		Question: "T(x) = (" + ratS(a) + ")x" + minus(signedTerm(b)) + " is a contraction of R, since |T(x) − T(y)| = (" + ratS(rq(int64(absInt(pn)), int64(pd))) +
			")|x − y|. What is its unique fixed point? (enter a decimal)",
		Answer: answer,
		Explanation: "The fixed point solves x = (" + ratS(a) + ")x" + minus(signedTerm(b)) + ", so x = " + itoa(b) + "/(1 − (" + ratS(a) + ")) = " + ratS(fixed) +
			". The contraction mapping principle says the iterates from any start converge to it.",
		Facts: map[string]string{"pn": strconv.Itoa(pn), "pd": strconv.Itoa(pd), "b": strconv.Itoa(b), "answer": answer},
	}
}

// ---------------------------------------------------------------------------
// ra.deriv.darboux (multiple choice)
// ---------------------------------------------------------------------------

func darbouxValueClaim() mcClaim {
	a := randIn(-3, 3)
	b := a + randIn(1, 4)
	fa := randIn(-8, 8)
	fb := randIn(-8, 8)
	for absInt(fb-fa) < 3 {
		fb = randIn(-8, 8)
	}
	lo, hi := min(fa, fb), max(fa, fb)
	v := randIn(lo+1, hi-1)
	correct := itoa(v)
	return mcClaim{
		q: "f is differentiable on [" + itoa(a) + ", " + itoa(b) + "] with f'(" + itoa(a) + ") = " + itoa(fa) + " and f'(" + itoa(b) + ") = " + itoa(fb) +
			". f' need not be continuous. Which value must f' take somewhere strictly between " + itoa(a) + " and " + itoa(b) + "?",
		correct:     correct,
		distractors: distractorsFrom(correct, itoa(hi+1), itoa(lo-1), itoa(hi+4), itoa(lo-3)),
		reason: "By Darboux's theorem a derivative has the intermediate value property, so f' takes every value strictly between " + itoa(fa) + " and " + itoa(fb) +
			", including " + correct + ", even without continuity. Values outside that range need not be taken.",
	}
}

func darbouxNotDerivativeClaim() mcClaim {
	c := randIn(-3, 3)
	p := randIn(-4, 4)
	q := randIn(-4, 4)
	for q == p {
		q = randIn(-4, 4)
	}
	cs := itoa(c)
	correct := "g(x) = " + itoa(p) + " for x < " + cs + " and g(x) = " + itoa(q) + " for x ≥ " + cs
	return mcClaim{
		q:       "Which of these functions on R cannot be the derivative of any differentiable function?",
		correct: correct,
		distractors: []string{
			"g(x) = |" + xPlus(-c) + "|",
			"g(x) = 2x·sin(1/x) − cos(1/x) for x ≠ 0, and g(0) = 0",
			"g(x) = " + coefX(absInt(p)+1, "x²") + signedTerm(q),
		},
		reason: correct + " has a jump at " + cs + ": it skips every value strictly between " + itoa(p) + " and " + itoa(q) +
			", which Darboux's theorem forbids for a derivative. The others are derivatives: two are continuous, and the oscillating one is the derivative of x²·sin(1/x) (with value 0 at 0), discontinuous at 0 but without a jump.",
	}
}

// ---------------------------------------------------------------------------
// ra.deriv.uniform_conv (multiple choice)
// ---------------------------------------------------------------------------

func uniformConvSupClaim() mcClaim {
	if rand.Intn(2) == 0 {
		a := randIn(1, 9)
		n := randIn(2, 20)
		correct := ratS(rq(int64(a), int64(n)))
		return mcClaim{
			q:           "f_n(x) = x/n converges to 0 on [0, " + strconv.Itoa(a) + "]. What is sup |f_n(x) − 0| over the interval when n = " + strconv.Itoa(n) + "?",
			correct:     correct,
			distractors: distractorsFrom(correct, ratS(rq(1, int64(n))), strconv.Itoa(a), ratS(rq(int64(a), int64(n+1))), ratS(rq(int64(a), int64(2*n)))),
			reason: "x/n is largest at the right endpoint, so the supremum is " + strconv.Itoa(a) + "/" + strconv.Itoa(n) + " = " + correct +
				". It tends to 0, so the convergence is uniform on this bounded interval.",
		}
	}
	q := randIn(2, 5)
	n := randIn(2, 5)
	b := rq(1, int64(q))
	sup := rq(1, int64(raIntPow(q, n)))
	correct := ratS(sup)
	return mcClaim{
		q:           "f_n(x) = x^n converges to 0 on [0, " + ratS(b) + "]. What is sup |f_n(x) − 0| over the interval when n = " + strconv.Itoa(n) + "?",
		correct:     correct,
		distractors: distractorsFrom(correct, ratS(b), "1", ratS(rq(int64(n), int64(q))), ratS(rq(1, int64(raIntPow(q, n+1))))),
		reason: "x^n is increasing on the interval, so the supremum is (" + ratS(b) + ")^" + strconv.Itoa(n) + " = " + correct +
			". These suprema tend to 0, so the convergence is uniform on [0, " + ratS(b) + "], unlike on [0, 1].",
	}
}

func uniformConvNClaim() mcClaim {
	a := randIn(1, 9)
	e := pick([]int{10, 100, 1000})
	// sup over [0, a] of x/n is a/n; need a/N < 1/e, so N > a·e.
	n := a*e + 1
	correct := strconv.Itoa(n)
	return mcClaim{
		q: "f_n(x) = x/n on [0, " + strconv.Itoa(a) + "]. What is the smallest N such that |f_n(x)| < " + decimal6(1/float64(e)) +
			" for every x in the interval and every n ≥ N?",
		correct:     correct,
		distractors: distractorsFrom(correct, strconv.Itoa(n-1), strconv.Itoa(e+1), strconv.Itoa(n+1), strconv.Itoa(2*a*e)),
		reason: "The worst point is x = " + strconv.Itoa(a) + ", where |f_n| = " + strconv.Itoa(a) + "/n. That is below " + decimal6(1/float64(e)) +
			" exactly when n > " + strconv.Itoa(a*e) + ", so N = " + correct + " works for every x at once, which is what uniform convergence asks.",
	}
}

func uniformConvWhichClaim() mcClaim {
	b := pick([]string{"1/2", "1/3", "2/3", "3/4", "0.9"})
	k := randIn(1, 9)
	ks := strconv.Itoa(k)
	yes := []string{
		"f_n(x) = x^n on [0, " + b + "]",
		"f_n(x) = sin(" + coefX(k, "nx") + ")/n on R",
		"f_n(x) = x/n on [0, " + ks + "]",
		"f_n(x) = x^n/n on [0, 1]",
		"f_n(x) = " + ks + "/(n + x²) on R",
	}
	no := []string{
		"f_n(x) = x^n on [0, 1]",
		"f_n(x) = x/n on R",
		"f_n(x) = nx/(1 + n²x²) on [0, 1]",
		"f_n(x) = x^n on [0, 1)",
		"f_n(x) = " + ks + "/(1 + nx) on (0, 1)",
	}
	if rand.Intn(2) == 0 {
		return oneOfFour("Which of these sequences converges uniformly on the set given?", yes, no, func(c string) string {
			return "For " + c + ", sup |f_n − f| tends to 0. Each of the others converges pointwise, but sup |f_n − f| stays away from 0: it is infinite, or 1/2 (attained at x = 1/n), or close to the value of f_n at an endpoint the limit does not respect."
		})
	}
	return oneOfFour("Which of these sequences converges pointwise but not uniformly on the set given?", no, yes, func(c string) string {
		return "For " + c + ", sup |f_n − f| does not tend to 0, so the convergence is not uniform. Each of the others has sup |f_n − f| → 0."
	})
}

// ---------------------------------------------------------------------------
// ra.deriv.interchange (multiple choice)
// ---------------------------------------------------------------------------

func interchangeSpikeClaim() mcClaim {
	k := randIn(1, 9)
	ks := strconv.Itoa(k)
	correct := ratS(rq(int64(k), 2))
	return mcClaim{
		q:           "f_n(x) = " + coefX(k, "nx") + "·e^(−nx²) on [0, 1] converges pointwise to 0. What is the limit of the integrals ∫_0^1 f_n(x) dx?",
		correct:     correct,
		distractors: distractorsFrom(correct, "0", ks, "∞", ratS(rq(int64(k), 4))),
		reason:      "∫_0^1 f_n = " + halfCoef(k, "(1 − e^(−n))") + ", which tends to " + correct + ", while the integral of the pointwise limit is 0. The limit and the integral cannot be swapped here because the convergence is not uniform: f_n has a peak near x = 1/√(2n) whose height grows like √n.",
	}
}

func interchangeUniformClaim() mcClaim {
	a := randIn(1, 6)
	c := randIn(1, 9)
	correct := strconv.Itoa(c * a)
	return mcClaim{
		q:           "f_n(x) = x/n + " + strconv.Itoa(c) + " converges uniformly on [0, " + strconv.Itoa(a) + "]. What is the limit of ∫_0^" + strconv.Itoa(a) + " f_n(x) dx?",
		correct:     correct,
		distractors: distractorsFrom(correct, "0", ratAnswer(radd(ri(int64(c*a)), rq(int64(a*a), 2))), strconv.Itoa(c), ratAnswer(rq(int64(a*a), 2))),
		reason: "The limit function is the constant " + strconv.Itoa(c) + ", and uniform convergence on a bounded interval lets the limit pass through the integral: the answer is " +
			strconv.Itoa(c) + "·" + strconv.Itoa(a) + " = " + correct + ". Directly, ∫ f_n = " + ratS(rq(int64(a*a), 2)) + "/n + " + correct + ", which tends to the same value.",
	}
}

func interchangeDerivClaim() mcClaim {
	q := randIn(2, 6)
	p := randIn(1, q-1)
	for gcd(p, q) != 1 {
		p = randIn(1, q-1)
	}
	r := rq(int64(p), int64(q))
	one := rsub(ri(1), r)
	v := rquo(ri(1), rmul(one, one))
	correct := ratS(v)
	return mcClaim{
		q:           "On |x| < 1, Σ_{n≥0} x^n = 1/(1 − x). Differentiating term by term, what is Σ_{n≥1} n·x^(n−1) at x = " + ratS(r) + "?",
		correct:     correct,
		distractors: distractorsFrom(correct, ratS(rquo(ri(1), one)), ratS(rquo(r, rmul(one, one))), ratS(rquo(ri(2), rmul(one, one))), ratS(rmul(one, one))),
		reason: "A power series converges uniformly on every closed interval inside its radius, and so does its derived series, so term-by-term differentiation is valid: the sum is 1/(1 − x)² = " +
			correct + " at x = " + ratS(r) + ".",
	}
}

// ---------------------------------------------------------------------------
// ra.metric.spaces (multiple choice)
// ---------------------------------------------------------------------------

func metricDistanceClaim() mcClaim {
	trip := pick([][3]int{{3, 4, 5}, {5, 12, 13}, {6, 8, 10}, {8, 15, 17}, {4, 3, 5}, {12, 5, 13}, {9, 12, 15}})
	dx, dy, d2 := trip[0], trip[1], trip[2]
	x1, y1 := randIn(-4, 4), randIn(-4, 4)
	sx, sy := 1, 1
	if rand.Intn(2) == 0 {
		sx = -1
	}
	if rand.Intn(2) == 0 {
		sy = -1
	}
	x2, y2 := x1+sx*dx, y1+sy*dy
	pt := func(a, b int) string { return "(" + itoa(a) + ", " + itoa(b) + ")" }
	vals := map[string]int{"d_1, the taxicab metric |x₁ − x₂| + |y₁ − y₂|": dx + dy, "d_2, the Euclidean metric": d2, "d_∞, the maximum metric max(|x₁ − x₂|, |y₁ − y₂|)": max(dx, dy)}
	names := []string{"d_1, the taxicab metric |x₁ − x₂| + |y₁ − y₂|", "d_2, the Euclidean metric", "d_∞, the maximum metric max(|x₁ − x₂|, |y₁ − y₂|)"}
	name := pick(names)
	correct := strconv.Itoa(vals[name])
	return mcClaim{
		q:           "In R² with " + name + ", what is the distance between " + pt(x1, y1) + " and " + pt(x2, y2) + "?",
		correct:     correct,
		distractors: distractorsFrom(correct, strconv.Itoa(dx+dy), strconv.Itoa(d2), strconv.Itoa(max(dx, dy)), strconv.Itoa(dx*dx+dy*dy), strconv.Itoa(min(dx, dy))),
		reason: "The coordinate differences are " + strconv.Itoa(dx) + " and " + strconv.Itoa(dy) + ". Their sum is " + strconv.Itoa(dx+dy) + ", the Euclidean length is √(" +
			strconv.Itoa(dx*dx) + " + " + strconv.Itoa(dy*dy) + ") = " + strconv.Itoa(d2) + ", and the larger is " + strconv.Itoa(max(dx, dy)) + "; the metric asked for gives " + correct + ".",
	}
}

func metricWhichClaim() mcClaim {
	k := randIn(2, 7)
	ks := strconv.Itoa(k)
	yes := []string{
		"d(x, y) = " + ks + "|x − y|",
		"d(x, y) = √|x − y|",
		"d(x, y) = min(1, |x − y|)",
		"d(x, y) = |x − y|/(1 + |x − y|)",
		"d(x, y) = |x³ − y³|",
	}
	no := []string{
		"d(x, y) = (x − y)²",
		"d(x, y) = |x² − y²|",
		"d(x, y) = |x − " + ks + "y|",
		"d(x, y) = x − y",
		"d(x, y) = |x − y| + " + ks,
	}
	if rand.Intn(2) == 0 {
		return oneOfFour("Which of these is a metric on R?", yes, no, func(c string) string {
			return c + " is nonnegative, symmetric, zero exactly when x = y, and satisfies the triangle inequality. Each of the others fails one of these: (x − y)² breaks the triangle inequality at 0, 1, 2; |x² − y²| vanishes at x = −y; |x − ky| is not symmetric; x − y can be negative; adding a constant makes d(x, x) nonzero."
		})
	}
	return oneOfFour("Which of these is not a metric on R?", no, yes, func(c string) string {
		return c + " fails one of the axioms: nonnegativity, d(x, y) = 0 exactly when x = y, symmetry, or the triangle inequality. Each of the others satisfies all four."
	})
}

// ---------------------------------------------------------------------------
// ra.metric.open_closed and ra.metric.closure (multiple choice)
// ---------------------------------------------------------------------------

var openClosedLabels = []string{
	"open but not closed",
	"closed but not open",
	"both open and closed",
	"neither open nor closed",
}

func openClosedClaim() mcClaim {
	a := randIn(-4, 4)
	b := a + randIn(1, 4)
	c := b + randIn(1, 3)
	d := c + randIn(1, 3)
	as, bs, cs, ds := itoa(a), itoa(b), itoa(c), itoa(d)
	type item struct {
		set string
		cls int
		why string
	}
	items := []item{
		{"(" + as + ", " + bs + ")", 0, "every point has a small interval around it inside the set, but the endpoints are limit points it does not contain"},
		{"[" + as + ", " + bs + "]", 1, "it contains all its limit points, but no interval around " + as + " stays inside it"},
		{"[" + as + ", " + bs + ")", 3, "no interval around " + as + " stays inside it, and the limit point " + bs + " is missing"},
		{"R", 2, "it contains every interval and every limit point"},
		{"Z", 1, "its points are isolated, so it has no limit points to miss, but no interval around an integer stays inside it"},
		{"{" + as + " + 1/n : n ∈ N}", 3, "the limit point " + as + " is missing, and no interval around a point of the set stays inside it"},
		{"{" + as + " + 1/n : n ∈ N} ∪ {" + as + "}", 1, "its only limit point " + as + " is included, but no interval around a point stays inside it"},
		{"Q", 3, "every real is a limit point of Q, and every interval contains irrationals"},
		{"(" + as + ", ∞)", 0, "each point has an interval around it inside the set, but the limit point " + as + " is missing"},
		{"[" + as + ", ∞)", 1, "it contains all its limit points, but no interval around " + as + " stays inside it"},
		{"R with the point " + as + " removed", 0, "its complement {" + as + "} is closed, and " + as + " is a limit point it misses"},
		{"(" + as + ", " + bs + ") ∪ (" + cs + ", " + ds + ")", 0, "a union of open intervals is open, and the endpoints are missing limit points"},
	}
	it := pick(items)
	correct := openClosedLabels[it.cls]
	return mcClaim{
		q:           "In R with the usual metric, the set " + it.set + " is...",
		correct:     correct,
		distractors: distractorsFrom(correct, openClosedLabels...),
		reason:      "It is " + correct + ": " + it.why + ".",
	}
}

func closureClaim() mcClaim {
	a := randIn(-4, 4)
	b := a + randIn(1, 4)
	c := b + randIn(1, 3)
	as, bs, cs := itoa(a), itoa(b), itoa(c)
	type item struct {
		op, set, want string
		others        []string
		why           string
	}
	items := []item{
		{"closure", "(" + as + ", " + bs + ")", "[" + as + ", " + bs + "]", []string{"(" + as + ", " + bs + ")", "[" + as + ", " + bs + ")", "R"},
			"the endpoints are limits of points of the interval, and nothing outside [" + as + ", " + bs + "] is"},
		{"closure", "{" + as + " + 1/n : n ∈ N}", "{" + as + " + 1/n : n ∈ N} ∪ {" + as + "}", []string{"{" + as + " + 1/n : n ∈ N}", "[" + as + ", " + itoa(a+1) + "]", "{" + as + "}"},
			"the only limit point not already in the set is " + as + ", the limit of the sequence"},
		{"closure", "Q ∩ (" + as + ", " + bs + ")", "[" + as + ", " + bs + "]", []string{"Q ∩ [" + as + ", " + bs + "]", "(" + as + ", " + bs + ")", "Q"},
			"by density every point of [" + as + ", " + bs + "] is a limit of rationals from the interval"},
		{"closure", "(" + as + ", " + bs + ") ∪ (" + bs + ", " + cs + ")", "[" + as + ", " + cs + "]", []string{"[" + as + ", " + bs + ") ∪ (" + bs + ", " + cs + "]", "(" + as + ", " + cs + ")", "[" + as + ", " + bs + "] ∪ [" + bs + ", " + cs + "]∖{" + bs + "}"},
			"the points " + as + ", " + bs + " and " + cs + " are all limits of points in the union"},
		{"interior", "[" + as + ", " + bs + "]", "(" + as + ", " + bs + ")", []string{"[" + as + ", " + bs + "]", "{" + as + ", " + bs + "}", "the empty set"},
			"every point strictly between the endpoints has an interval inside the set, and the endpoints do not"},
		{"interior", "Q", "the empty set", []string{"Q", "R", "the irrational numbers"},
			"every interval contains irrationals, so no interval fits inside Q"},
		{"interior", "[" + as + ", " + bs + "] ∪ {" + cs + "}", "(" + as + ", " + bs + ")", []string{"[" + as + ", " + bs + "]", "(" + as + ", " + bs + ") ∪ {" + cs + "}", "(" + as + ", " + cs + ")"},
			"the isolated point " + cs + " and the endpoints have no interval inside the set"},
	}
	it := pick(items)
	return mcClaim{
		q:           "In R with the usual metric, what is the " + it.op + " of " + it.set + "?",
		correct:     it.want,
		distractors: distractorsFrom(it.want, it.others...),
		reason:      "The " + it.op + " is " + it.want + ": " + it.why + ".",
	}
}

// ---------------------------------------------------------------------------
// ra.metric.convergence (multiple choice)
// ---------------------------------------------------------------------------

func metricConvNClaim() mcClaim {
	a, b := randIn(-4, 4), randIn(-4, 4)
	p, q := randIn(1, 6), randIn(1, 6)
	e := pick([]int{10, 100})
	useMax := rand.Intn(2) == 0
	name, dist, distS := "d_1 (taxicab)", p+q, strconv.Itoa(p)+" + "+strconv.Itoa(q)
	if useMax {
		name, dist, distS = "d_∞ (maximum)", max(p, q), "max("+strconv.Itoa(p)+", "+strconv.Itoa(q)+")"
	}
	n := dist*e + 1
	correct := strconv.Itoa(n)
	return mcClaim{
		q: "In R² with the metric " + name + ", x_n = (" + itoa(a) + " + " + strconv.Itoa(p) + "/n, " + itoa(b) + " − " + strconv.Itoa(q) + "/n) converges to (" + itoa(a) + ", " + itoa(b) +
			"). What is the smallest N such that the distance to the limit is below " + decimal6(1/float64(e)) + " for every n ≥ N?",
		correct:     correct,
		distractors: distractorsFrom(correct, strconv.Itoa((p+q)*e+1), strconv.Itoa(max(p, q)*e+1), strconv.Itoa(n-1), strconv.Itoa(min(p, q)*e+1)),
		reason: "The distance is (" + distS + ")/n = " + strconv.Itoa(dist) + "/n, which decreases in n and is below " + decimal6(1/float64(e)) + " exactly when n > " + strconv.Itoa(dist*e) +
			", so N = " + correct + ".",
	}
}

func metricDiscreteClaim() mcClaim {
	k := randIn(3, 20)
	c := randIn(-5, 9)
	ks, cs := strconv.Itoa(k), itoa(c)
	correct := "x_n = n for n < " + ks + ", and x_n = " + cs + " for n ≥ " + ks
	return mcClaim{
		q:       "R carries the discrete metric: d(x, y) = 1 whenever x ≠ y. Which of these sequences converges?",
		correct: correct,
		distractors: []string{
			"x_n = " + cs + " + 1/n",
			"x_n = (−1)^n",
			"x_n = " + cs + " + 1/2^n",
		},
		reason: "In the discrete metric a ball of radius 1/2 is a single point, so x_n → x forces x_n = x for all large n. Only the sequence that is eventually constant at " + cs +
			" does that; " + cs + " + 1/n gets close in the usual sense but stays at distance 1 from every point.",
	}
}

// ---------------------------------------------------------------------------
// ra.metric.complete (multiple choice)
// ---------------------------------------------------------------------------

func completeWhichClaim() mcClaim {
	a := randIn(-4, 4)
	b := a + randIn(1, 5)
	as, bs := itoa(a), itoa(b)
	yes := []string{"[" + as + ", " + bs + "]", "Z", "[" + as + ", ∞)", "R²", "a finite set of reals", "R with the discrete metric"}
	no := []string{"(" + as + ", " + bs + ")", "Q", "Q ∩ [" + as + ", " + bs + "]", "(" + as + ", ∞)", "[" + as + ", " + bs + ")"}
	if rand.Intn(2) == 0 {
		return oneOfFour("With the usual metric unless stated, which of these spaces is complete?", yes, no, func(c string) string {
			return c + " is complete: it is a closed subset of a complete space, or its Cauchy sequences are eventually constant. Each of the others has a Cauchy sequence converging (in R) to a point it does not contain."
		})
	}
	return oneOfFour("With the usual metric unless stated, which of these spaces is not complete?", no, yes, func(c string) string {
		return c + " is not complete: some Cauchy sequence in it converges in R to a point outside it, such as a missing endpoint or an irrational. Each of the others is complete."
	})
}

func completeRationalClaim() mcClaim {
	m := pick([]int{2, 3, 5, 6, 7, 10, 11})
	ms := strconv.Itoa(m)
	c := randIn(1, 5)
	cs := strconv.Itoa(c)
	correct := "x_n = the decimal expansion of √" + ms + " cut off after n places"
	return mcClaim{
		q:       "Which of these Cauchy sequences of rationals has no limit in Q?",
		correct: correct,
		distractors: []string{
			"x_n = " + cs + " + 1/n",
			"x_n = (" + cs + "n + 1)/n",
			"x_n = " + cs + "·(−1)^n/n",
		},
		reason: "Each truncation of √" + ms + " is rational, and they are Cauchy because they agree to n places, but their limit √" + ms + " is irrational. The others converge to the rationals " + cs + ", " + cs + " and 0.",
	}
}

// ---------------------------------------------------------------------------
// ra.metric.compact (multiple choice)
// ---------------------------------------------------------------------------

func metricCompactClaim() mcClaim {
	r := randIn(1, 5)
	a := randIn(1, 4)
	m := randIn(1, 4)
	rs, as, ms := strconv.Itoa(r), strconv.Itoa(a), strconv.Itoa(m)
	yes := []string{
		"the closed disc x² + y² ≤ " + strconv.Itoa(r*r),
		"the circle x² + y² = " + strconv.Itoa(r*r),
		"the square [0, " + as + "] × [0, " + rs + "]",
		"the arc {(x, x²) : |x| ≤ " + as + "}",
		"{(1/n, 0) : n ∈ N} ∪ {(0, 0)}",
	}
	no := []string{
		"the open disc x² + y² < " + strconv.Itoa(r*r),
		"the line y = " + ms + "x",
		"the region x² + y² ≥ " + strconv.Itoa(r*r),
		"Z × Z",
		"{(1/n, 0) : n ∈ N}",
	}
	if rand.Intn(2) == 0 {
		return oneOfFour("Which of these subsets of R² (Euclidean metric) is compact?", yes, no, func(c string) string {
			return c + " is closed and bounded in R², so it is compact by Heine–Borel. Each of the others is unbounded or misses one of its own limit points."
		})
	}
	return oneOfFour("Which of these subsets of R² (Euclidean metric) is not compact?", no, yes, func(c string) string {
		return c + " is unbounded or misses one of its own limit points, so it is not closed and bounded, and a sequence in it has no subsequence converging inside it. Each of the others is closed and bounded."
	})
}

// ---------------------------------------------------------------------------
// ra.metric.continuity (multiple choice)
// ---------------------------------------------------------------------------

func metricPreimageClaim() mcClaim {
	p := randIn(0, 3)
	q := p + randIn(1, 3)
	ps, qs := strconv.Itoa(p), strconv.Itoa(q)
	p2, q2 := strconv.Itoa(p*p), strconv.Itoa(q*q)
	correct := "(−" + qs + ", −" + ps + ") ∪ (" + ps + ", " + qs + ")"
	if p == 0 {
		correct = "(−" + qs + ", 0) ∪ (0, " + qs + ")"
	}
	return mcClaim{
		q:       "f(x) = x² on R. What is the preimage f⁻¹((" + p2 + ", " + q2 + ")) of the open interval (" + p2 + ", " + q2 + ")?",
		correct: correct,
		distractors: distractorsFrom(correct, "("+ps+", "+qs+")", "("+p2+", "+q2+")", "[−"+qs+", "+qs+"]",
			"(−"+qs+", "+qs+")", "("+itoa(-q*q)+", "+q2+")"),
		reason: p2 + " < x² < " + q2 + " means " + ps + " < |x| < " + qs + ", which is " + correct +
			". It is open, as it must be: a continuous function pulls open sets back to open sets.",
	}
}

func metricLipschitzClaim() mcClaim {
	a := randNonzero(-5, 5)
	b := randNonzero(-5, 5)
	l := absInt(a) + absInt(b)
	correct := strconv.Itoa(l)
	f := coefX(a, "x") + minus(signedTerm(b)) + "y"
	if b == 1 || b == -1 {
		f = coefX(a, "x") + map[bool]string{true: " + y", false: " − y"}[b > 0]
	}
	return mcClaim{
		q:           "f(x, y) = " + f + " from (R², d_∞) to R, where d_∞ is the maximum metric. What is the smallest L with |f(p) − f(q)| ≤ L·d_∞(p, q) for all p, q?",
		correct:     correct,
		distractors: distractorsFrom(correct, strconv.Itoa(max(absInt(a), absInt(b))), strconv.Itoa(absInt(a+b)), strconv.Itoa(a*a+b*b), strconv.Itoa(l+1)),
		reason: "|f(p) − f(q)| ≤ |" + itoa(a) + "||Δx| + |" + itoa(b) + "||Δy| ≤ " + correct + "·d_∞(p, q), with equality when Δx and Δy have size d_∞ and the signs of " +
			itoa(a) + " and " + itoa(b) + ". So f is Lipschitz, hence uniformly continuous: δ = ε/" + correct + " works everywhere.",
	}
}

// ---------------------------------------------------------------------------
// ra.metric.equivalent (multiple choice)
// ---------------------------------------------------------------------------

func equivalentBoundedClaim() mcClaim {
	x := randIn(-6, 6)
	y := randIn(-6, 6)
	for y == x {
		y = randIn(-6, 6)
	}
	d := absInt(x - y)
	if rand.Intn(2) == 0 {
		v := rq(int64(d), int64(d+1))
		correct := ratS(v)
		return mcClaim{
			q:           "On R let d(x, y) = |x − y| and ρ(x, y) = d(x, y)/(1 + d(x, y)). What is ρ(" + itoa(x) + ", " + itoa(y) + ")?",
			correct:     correct,
			distractors: distractorsFrom(correct, strconv.Itoa(d), ratS(rq(1, int64(d+1))), ratS(rq(int64(d+1), int64(d))), "1"),
			reason: "d(" + itoa(x) + ", " + itoa(y) + ") = " + strconv.Itoa(d) + ", so ρ = " + strconv.Itoa(d) + "/" + strconv.Itoa(d+1) +
				". ρ is always below 1, yet it has the same open sets as d, because for small distances the two are nearly equal.",
		}
	}
	v := min(1, d)
	correct := strconv.Itoa(v)
	return mcClaim{
		q:           "On R let ρ(x, y) = min(1, |x − y|). What is ρ(" + itoa(x) + ", " + itoa(y) + ")?",
		correct:     correct,
		distractors: distractorsFrom(correct, strconv.Itoa(d), "0", ratS(rq(int64(d), int64(d+1))), strconv.Itoa(d+1)),
		reason: "|x − y| = " + strconv.Itoa(d) + ", so ρ = min(1, " + strconv.Itoa(d) + ") = " + correct +
			". Truncating at 1 does not change balls of radius below 1, so ρ and the usual metric are equivalent.",
	}
}

func equivalentConstantClaim() mcClaim {
	n := randIn(2, 9)
	ns := strconv.Itoa(n)
	rootN := "√" + ns
	if n == 4 || n == 9 {
		rootN = strconv.Itoa(int(math.Round(math.Sqrt(float64(n)))))
	}
	type item struct {
		ineq, want, why string
		others          []string
	}
	items := []item{
		{"d_1(x, y) ≤ C·d_∞(x, y)", ns, "the sum of n coordinate gaps is at most n times the largest, with equality when all gaps are equal",
			[]string{rootN, "1", strconv.Itoa(n * n)}},
		{"d_2(x, y) ≤ C·d_∞(x, y)", rootN, "the sum of n squared gaps is at most n times the largest squared gap, so d_2 ≤ √n·d_∞, with equality when all gaps are equal",
			[]string{ns, "1", strconv.Itoa(n * n)}},
		{"d_∞(x, y) ≤ C·d_1(x, y)", "1", "the largest gap is one of the terms of the sum, with equality when only one coordinate differs",
			[]string{ns, rootN, "1/" + ns}},
		{"d_1(x, y) ≤ C·d_2(x, y)", rootN, "by the Cauchy–Schwarz inequality the sum of n gaps is at most √n times the Euclidean length, with equality when all gaps are equal",
			[]string{ns, "1", strconv.Itoa(n * n)}},
	}
	it := pick(items)
	return mcClaim{
		q:           "On R^" + ns + ", what is the smallest constant C with " + it.ineq + " for all x and y?",
		correct:     it.want,
		distractors: distractorsFrom(it.want, it.others...),
		reason:      "C = " + it.want + ": " + it.why + ". Two-sided bounds like this make the metrics Lipschitz equivalent, so they have the same open sets and the same Cauchy sequences.",
	}
}

// halfCoef prints (a/2)·v: "3t²", "t²" or "(7/2)t²".
func halfCoef(a int, v string) string {
	if a%2 == 0 {
		return coefX(a/2, v)
	}
	return "(" + ratS(rq(int64(a), 2)) + ")" + v
}
