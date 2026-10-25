package realanalysis

// The first real-analysis batch: twenty-five concepts that open the domain with
// the completeness of the real line, the theory of sequences and series, and the
// first two-thirds of limits and continuity.
//
// Real analysis is where the informal calculus is made rigorous. This batch
// starts from the least upper bound property, draws out the Archimedean
// property, density, countability and the uncountability of the reals, and then
// proves the theorems about sequences, series, limits and continuity that the
// calculus course used. Nineteen concepts are multiple choice, because the
// domain is largely about statements and their proofs, and six are numeric, so
// that the definitions can be exercised as well as quoted.
//
// Every numeric generator publishes the parameters its answer follows from, so
// the tests recompute the answer rather than trusting the author's arithmetic.

import (
	"math"
	"math/rand"
	"strconv"

	"github.com/chuma-beep/mathua/internal/generator"
)

func decimal6(v float64) string {
	s := strconv.FormatFloat(v, 'f', 6, 64)
	for len(s) > 0 && s[len(s)-1] == '0' {
		s = s[:len(s)-1]
	}
	if len(s) > 0 && s[len(s)-1] == '.' {
		s = s[:len(s)-1]
	}
	if s == "" || s == "-0" {
		s = "0"
	}
	return s
}

// mcClaim is one multiple-choice item: a question, the correct option, the
// distractors that must all be rejected, and the reason the correct option is
// right.
type mcClaim struct {
	q           string
	correct     string
	distractors []string
	reason      string
}

func pickMC(claims []mcClaim) generator.Problem {
	c := claims[rand.Intn(len(claims))]
	p := generator.MultipleChoice(c.q, c.correct, c.distractors...)
	p.Explanation = c.reason
	return p
}

// ---------------------------------------------------------------------------
// Supremum and infimum (numeric)
// ---------------------------------------------------------------------------

type supremumGen struct{}

func (g *supremumGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	if rand.Intn(4) != 0 {
		return supremumVariant()
	}
	type claim struct {
		lo, hi int
		set    string
	}
	claims := []claim{
		{lo: 0, hi: 1, set: "the open interval of all x with 0 < x < 1"},
		{lo: -2, hi: 3, set: "the half-open interval of all x with −2 < x ≤ 3"},
		{lo: 1, hi: 5, set: "the half-open interval of all x with 1 ≤ x < 5"},
		{lo: 2, hi: 4, set: "the open interval of all x with 2 < x < 4"},
		{lo: 0, hi: 10, set: "the closed interval of all x with 0 ≤ x ≤ 10"},
		{lo: -5, hi: -1, set: "the open interval of all x with −5 < x < −1"},
	}
	c := claims[rand.Intn(len(claims))]
	answer := strconv.Itoa(c.hi)
	return generator.Problem{
		Question: "What is the supremum of " + c.set + "? (enter a decimal)",
		Answer:   answer,
		Explanation: "The supremum is the least upper bound: it is an upper bound, and no smaller number is. Every element of " +
			c.set + " is at most " + strconv.Itoa(c.hi) + ", and numbers just below it are exceeded by elements of the set, so the least upper bound is " +
			strconv.Itoa(c.hi) + ".",
		Facts: map[string]string{
			"lo": strconv.Itoa(c.lo), "hi": strconv.Itoa(c.hi), "ask": "sup", "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// The Archimedean property (numeric)
// ---------------------------------------------------------------------------

type archimedeanGen struct{}

func (g *archimedeanGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	if rand.Intn(4) != 0 {
		return archimedeanVariant()
	}
	type claim struct {
		pn, pd int
	}
	claims := []claim{
		{pn: 1, pd: 100},
		{pn: 1, pd: 10},
		{pn: 2, pd: 5},
		{pn: 1, pd: 1000},
		{pn: 3, pd: 4},
		{pn: 1, pd: 7},
	}
	c := claims[rand.Intn(len(claims))]
	n := c.pd/c.pn + 1
	if c.pd%c.pn == 0 {
		n = c.pd/c.pn + 1
	} else {
		n = int(math.Floor(float64(c.pd)/float64(c.pn))) + 1
	}
	answer := strconv.Itoa(n)
	return generator.Problem{
		Question: "By the Archimedean property there is a positive integer N with 1/N < ε. For ε = " +
			strconv.Itoa(c.pn) + "/" + strconv.Itoa(c.pd) + ", what is the smallest such N? (enter a whole number)",
		Answer:      answer,
		Explanation: "The inequality 1/N < ε is the same as N > 1/ε, so the smallest integer N is the floor of 1/ε plus one; here 1/ε = " + strconv.Itoa(c.pd) + "/" + strconv.Itoa(c.pn) + ", giving N = " + answer + ".",
		Facts: map[string]string{
			"pn": strconv.Itoa(c.pn), "pd": strconv.Itoa(c.pd), "xn": "1", "xd": "1", "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// Convergence of sequences (numeric)
// ---------------------------------------------------------------------------

type sequenceGen struct{}

func (g *sequenceGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	if rand.Intn(4) != 0 {
		return sequenceVariant()
	}
	type claim struct {
		a, b, c, d int
	}
	claims := []claim{
		{a: 2, b: 1, c: 1, d: 3},
		{a: 3, b: 0, c: 2, d: 1},
		{a: 1, b: 0, c: 4, d: 7},
		{a: 5, b: 2, c: 1, d: 1},
		{a: 6, b: 0, c: 3, d: 1},
		{a: 7, b: 1, c: 2, d: 3},
	}
	c := claims[rand.Intn(len(claims))]
	answer := decimal6(float64(c.a) / float64(c.c))
	return generator.Problem{
		Question:    "The sequence with terms " + ratSeq{c.a, c.b, c.c, c.d}.String() + " converges as n tends to infinity. What is its limit? (enter a decimal)",
		Answer:      answer,
		Explanation: "Dividing the numerator and denominator by n, the ratio of the leading coefficients is the limit, because the constant terms become negligible; here that ratio is " + strconv.Itoa(c.a) + "/" + strconv.Itoa(c.c) + " = " + answer + ".",
		Facts: map[string]string{
			"a": strconv.Itoa(c.a), "b": strconv.Itoa(c.b), "c": strconv.Itoa(c.c), "d": strconv.Itoa(c.d), "kind": "limit", "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// Monotone convergence (numeric)
// ---------------------------------------------------------------------------

type monotoneGen struct{}

func (g *monotoneGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	if rand.Intn(4) != 0 {
		return monotoneVariant()
	}
	// a = L(L-1) makes the fixed point of L = sqrt(a + L) equal to L.
	claims := []int{2, 6, 12, 20, 30, 42}
	a := claims[rand.Intn(len(claims))]
	l := int((1 + math.Sqrt(1+4*float64(a))) / 2)
	answer := strconv.Itoa(l)
	return generator.Problem{
		Question: "The increasing sequence x_1 = 0, x_{n+1} = sqrt(a + x_n) is bounded above and converges. Its limit L satisfies L = sqrt(a + L). For a = " +
			strconv.Itoa(a) + ", what is L? (enter a decimal)",
		Answer:      answer,
		Explanation: "Squaring the fixed-point equation gives L² − L − a = 0, whose positive root is (1 + sqrt(1 + 4a))/2; the increasing sequence starting at zero converges to that root.",
		Facts: map[string]string{
			"kind": "sqrt", "a": strconv.Itoa(a), "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// Series and partial sums (numeric)
// ---------------------------------------------------------------------------

type seriesGen struct{}

func (g *seriesGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	if rand.Intn(4) != 0 {
		return seriesVariant()
	}
	type claim struct {
		pn, pd int
	}
	claims := []claim{
		{pn: 1, pd: 2},
		{pn: 1, pd: 3},
		{pn: 1, pd: 4},
		{pn: 2, pd: 3},
		{pn: 1, pd: 5},
		{pn: 3, pd: 5},
	}
	c := claims[rand.Intn(len(claims))]
	r := float64(c.pn) / float64(c.pd)
	answer := decimal6(1 / (1 - r))
	return generator.Problem{
		Question: "The geometric series with first term one and ratio r = " + strconv.Itoa(c.pn) + "/" + strconv.Itoa(c.pd) +
			" converges to 1/(1 − r). What is its sum? (enter a decimal)",
		Answer:      answer,
		Explanation: "A geometric series with ratio of absolute value less than one converges to the first term divided by one minus the ratio, which is the limit of its partial sums.",
		Facts: map[string]string{
			"pn": strconv.Itoa(c.pn), "pd": strconv.Itoa(c.pd), "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// The limit of a function (numeric)
// ---------------------------------------------------------------------------

type functionLimitGen struct{}

func (g *functionLimitGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	if rand.Intn(4) != 0 {
		return functionLimitVariant()
	}
	type claim struct {
		m, en, ed int
	}
	claims := []claim{
		{m: 3, en: 1, ed: 1},
		{m: 2, en: 1, ed: 1},
		{m: 4, en: 1, ed: 1},
		{m: 5, en: 2, ed: 1},
		{m: 3, en: 2, ed: 1},
		{m: 2, en: 3, ed: 1},
	}
	c := claims[rand.Intn(len(claims))]
	eps := float64(c.en) / float64(c.ed)
	answer := decimal6(eps / float64(c.m))
	return generator.Problem{
		Question: "For the linear function f(x) = " + strconv.Itoa(c.m) + "x − 1 and its limit at x = 2, the largest δ that makes |f(x) − f(2)| < ε whenever |x − 2| < δ is ε divided by the slope. For ε = " +
			ratDec(rq(int64(c.en), int64(c.ed))) + ", what is δ? (enter a decimal)",
		Answer:      answer,
		Explanation: "Because the function is linear, |f(x) − f(2)| = m|x − 2|, so the condition is m|x − 2| < ε, that is |x − 2| < ε/m; the largest such δ is exactly ε/m.",
		Facts: map[string]string{
			"m": strconv.Itoa(c.m), "en": strconv.Itoa(c.en), "ed": strconv.Itoa(c.ed), "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// The completeness axiom (multiple choice)
// ---------------------------------------------------------------------------

type completenessGen struct{}

func (g *completenessGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return mixMC([]mcClaim{
		{
			q:           "The completeness axiom says every nonempty set of reals that is bounded above has...",
			correct:     "a least upper bound in the reals",
			distractors: []string{"a greatest element", "only rational upper bounds", "no upper bound"},
			reason:      "Completeness is exactly the existence of a supremum in the reals.",
		},
		{
			q:           "The rationals are not complete because...",
			correct:     "the rationals with square less than two are bounded above but have no rational supremum",
			distractors: []string{"they are countable", "they are infinite", "they have no ordering"},
			reason:      "The missing supremum is the square root of two, which is irrational.",
		},
		{
			q:           "The supremum of a set is...",
			correct:     "its least upper bound",
			distractors: []string{"its greatest element", "its infimum", "any upper bound"},
			reason:      "The supremum is the smallest number that no element of the set exceeds.",
		},
		{
			q:           "A bounded nonempty set of reals always has...",
			correct:     "a supremum and an infimum",
			distractors: []string{"a maximum", "a minimum", "a rational element"},
			reason:      "Completeness gives the bounds even when the set does not attain them.",
		},
		{
			q:           "The supremum differs from the maximum exactly when...",
			correct:     "the set does not contain its least upper bound",
			distractors: []string{"the set is bounded", "the set is finite", "the set is empty"},
			reason:      "If the supremum belongs to the set it is the maximum; otherwise the set has no maximum.",
		},
	}, completenessBoundClaim, completenessBoundClaim, completenessApproxClaim, completenessScaleClaim)
}

// ---------------------------------------------------------------------------
// Nested intervals (multiple choice)
// ---------------------------------------------------------------------------

type nestedIntervalsGen struct{}

func (g *nestedIntervalsGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return mixMC([]mcClaim{
		{
			q:           "The nested interval property says a decreasing sequence of nonempty closed bounded intervals has...",
			correct:     "a nonempty intersection",
			distractors: []string{"an empty intersection", "exactly one point always", "infinitely many points always"},
			reason:      "The intersection is nonempty, and it can be a single point when the lengths shrink to zero.",
		},
		{
			q:           "The nested interval property fails for open intervals because...",
			correct:     "open intervals can shrink to a point that none of them contains",
			distractors: []string{"open intervals are unbounded", "open intervals are empty", "points are not real numbers"},
			reason:      "Closedness is what keeps the limiting point inside every interval.",
		},
		{
			q:           "The nested interval property is equivalent to...",
			correct:     "the completeness of the real line",
			distractors: []string{"the uncountability of the reals", "the countability of the rationals", "the Archimedean property alone"},
			reason:      "Each can be derived from the other, so they express the same property of the reals.",
		},
		{
			q:           "The nested interval property is used to prove...",
			correct:     "the intermediate value theorem and the Bolzano-Weierstrass theorem",
			distractors: []string{"the countability of the rationals", "the Archimedean property", "the density of the integers"},
			reason:      "Both classical proofs bisect an interval and use the limiting point.",
		},
		{
			q:           "When the nested intervals are used to locate a single point, their lengths tend to...",
			correct:     "zero",
			distractors: []string{"infinity", "one", "a negative number"},
			reason:      "The intersection collapses to one point exactly when the lengths shrink to zero.",
		},
	}, nestedIntersectionClaim, nestedIntersectionClaim, nestedBisectionClaim)
}

// ---------------------------------------------------------------------------
// Density of the rationals (multiple choice)
// ---------------------------------------------------------------------------

type densityGen struct{}

func (g *densityGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return mixMC([]mcClaim{
		{
			q:           "The density of the rationals says that between any two distinct reals there is...",
			correct:     "a rational number",
			distractors: []string{"an integer", "no real number", "only an irrational"},
			reason:      "Scaling a short interval by a large integer produces a rational point inside it.",
		},
		{
			q:           "The density of the irrationals says that between any two distinct reals there is...",
			correct:     "an irrational number",
			distractors: []string{"an integer", "no real number", "only a rational"},
			reason:      "Adding an irrational multiple small enough to stay inside the interval to a rational point gives an irrational inside it.",
		},
		{
			q:           "A consequence of density is that every open interval contains...",
			correct:     "infinitely many rationals and infinitely many irrationals",
			distractors: []string{"exactly one rational", "no rationals", "only integers"},
			reason:      "Density can be applied again inside any subinterval, so the count is infinite.",
		},
		{
			q:           "Density follows from the Archimedean property together with...",
			correct:     "the ability to subdivide an interval by a large enough integer",
			distractors: []string{"completeness of the rationals", "the extended real line", "countability"},
			reason:      "Choosing an integer larger than the reciprocal of the interval length produces a grid point inside it.",
		},
		{
			q:           "Density shows that the rationals are...",
			correct:     "spread throughout the real line",
			distractors: []string{"finite", "complete", "discrete"},
			reason:      "No open interval is free of rationals, so they are not separated from one another.",
		},
	}, densityProofClaim, densityIrrationalClaim, densityCountClaim)
}

// ---------------------------------------------------------------------------
// Countable sets (multiple choice)
// ---------------------------------------------------------------------------

type countableGen struct{}

func (g *countableGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return mixMC([]mcClaim{
		{
			q:           "A set is countable when...",
			correct:     "it is finite or can be listed as a sequence",
			distractors: []string{"it is infinite", "it has no order", "it is a subset of the reals"},
			reason:      "Countability is the existence of an enumeration, possibly finite.",
		},
		{
			q:           "The set of rational numbers is...",
			correct:     "countable",
			distractors: []string{"uncountable", "finite", "empty"},
			reason:      "The rationals can be listed by pairing each numerator with each denominator.",
		},
		{
			q:           "A countable union of countable sets is...",
			correct:     "countable",
			distractors: []string{"uncountable", "finite", "empty"},
			reason:      "The lists can be interleaved along the diagonals of a grid.",
		},
		{
			q:           "The set of algebraic numbers is...",
			correct:     "countable",
			distractors: []string{"uncountable", "finite", "empty"},
			reason:      "There are countably many polynomials with integer coefficients and each has finitely many roots.",
		},
		{
			q:           "The set of integers is...",
			correct:     "countable",
			distractors: []string{"uncountable", "finite", "a field"},
			reason:      "The integers can be listed, for instance as zero, one, minus one, two, minus two, and so on.",
		},
	}, func() mcClaim { return countabilityClaim(true) }, countableZListingClaim, countablePairClaim)
}

// ---------------------------------------------------------------------------
// The uncountability of the reals (multiple choice)
// ---------------------------------------------------------------------------

type uncountableGen struct{}

func (g *uncountableGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return mixMC([]mcClaim{
		{
			q:           "The real numbers are...",
			correct:     "uncountable",
			distractors: []string{"countable", "finite", "empty"},
			reason:      "Cantor's diagonal argument shows no sequence can list them all.",
		},
		{
			q:           "Cantor's diagonal argument shows that...",
			correct:     "the reals cannot be listed as a sequence",
			distractors: []string{"the rationals are uncountable", "the integers are uncountable", "every set is countable"},
			reason:      "A number is built that differs from the n-th listed number in its n-th digit, so it is missing from the list.",
		},
		{
			q:           "The set of transcendental numbers is...",
			correct:     "uncountable",
			distractors: []string{"countable", "finite", "empty"},
			reason:      "The algebraic numbers are countable and the reals are not, so the transcendental complement is uncountable.",
		},
		{
			q:           "A countable subset of the reals has...",
			correct:     "measure zero",
			distractors: []string{"positive measure always", "no elements", "finitely many elements"},
			reason:      "A sequence of points can be covered by intervals of arbitrarily small total length.",
		},
		{
			q:           "The closed interval from zero to one is...",
			correct:     "uncountable",
			distractors: []string{"countable", "finite", "empty"},
			reason:      "A diagonal argument applies to the decimals of that interval just as it does to the whole line.",
		},
	}, func() mcClaim { return countabilityClaim(false) }, uncountableDiagonalClaim, uncountablePowerSetClaim)
}

// ---------------------------------------------------------------------------
// The extended real line (multiple choice)
// ---------------------------------------------------------------------------

type extendedGen struct{}

func (g *extendedGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return mixMC([]mcClaim{
		{
			q:           "The extended real line adds to the reals the two symbols...",
			correct:     "plus infinity and minus infinity",
			distractors: []string{"zero and one", "the rationals", "a largest finite real number"},
			reason:      "The two symbols act as endpoints beyond every real number.",
		},
		{
			q:           "In the extended reals, the supremum of a set that is unbounded above is...",
			correct:     "plus infinity",
			distractors: []string{"zero", "undefined", "a finite real number"},
			reason:      "The convention lets every set have a supremum, finite or infinite.",
		},
		{
			q:           "The extended real line is...",
			correct:     "order complete but not a field",
			distractors: []string{"a field", "countable", "compact"},
			reason:      "The order extends, but the arithmetic of the infinities is not a field.",
		},
		{
			q:           "Arithmetic in the extended reals leaves undefined...",
			correct:     "infinity minus infinity, and zero times infinity",
			distractors: []string{"infinity plus one", "infinity times two", "one over a positive number"},
			reason:      "These are the indeterminate forms that the conventions cannot give a value.",
		},
		{
			q:           "Suprema and infima are stated in the extended reals because then...",
			correct:     "every set has a supremum and an infimum",
			distractors: []string{"every set has a maximum", "the reals become countable", "every limit exists"},
			reason:      "The unbounded cases are absorbed by the two infinite endpoints.",
		},
	}, extendedArithmeticClaim, extendedSupInfClaim)
}

// ---------------------------------------------------------------------------
// Subsequences (multiple choice)
// ---------------------------------------------------------------------------

type subsequenceGen struct{}

func (g *subsequenceGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return mixMC([]mcClaim{
		{
			q:           "A subsequence of a convergent sequence...",
			correct:     "converges to the same limit",
			distractors: []string{"converges to a different limit always", "diverges", "oscillates"},
			reason:      "Dropping terms cannot change where the remaining terms settle.",
		},
		{
			q:           "A sequence converges to L exactly when...",
			correct:     "every subsequence converges to L",
			distractors: []string{"some subsequence converges to L", "no subsequence converges", "every subsequence is bounded"},
			reason:      "A sequence is its own subsequence, and every subsequence inherits the limit.",
		},
		{
			q:           "The limit points of a sequence are...",
			correct:     "the limits of its convergent subsequences",
			distractors: []string{"its first terms", "its upper and lower bounds", "its maxima"},
			reason:      "That is the definition of a limit point, or cluster point.",
		},
		{
			q:           "The sequence alternating between minus one and one has subsequences converging...",
			correct:     "to minus one and to one",
			distractors: []string{"only to zero", "only to one", "nowhere"},
			reason:      "Taking the even or the odd terms gives constant subsequences at each value.",
		},
		{
			q:           "If every subsequence of a bounded sequence has a further subsequence converging to L, then the original sequence...",
			correct:     "converges to L",
			distractors: []string{"diverges", "is unbounded", "has no limit points"},
			reason:      "If it did not converge to L, a subsequence would avoid a neighbourhood of L and have no further subsequence converging there.",
		},
	}, subsequenceLimitsClaim, subsequenceSameLimitClaim, subsequenceOddEvenClaim)
}

// ---------------------------------------------------------------------------
// The Bolzano-Weierstrass theorem (multiple choice)
// ---------------------------------------------------------------------------

type bolzanoWeierstrassGen struct{}

func (g *bolzanoWeierstrassGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return mixMC([]mcClaim{
		{
			q:           "The Bolzano-Weierstrass theorem says every bounded sequence of reals has...",
			correct:     "a convergent subsequence",
			distractors: []string{"a limit", "a maximum", "a monotone tail"},
			reason:      "The theorem gives a subsequential limit, not necessarily a limit of the whole sequence.",
		},
		{
			q:           "The theorem fails if the sequence is...",
			correct:     "unbounded",
			distractors: []string{"bounded", "monotone", "Cauchy"},
			reason:      "An unbounded sequence can escape every compact set, so no subsequence need converge.",
		},
		{
			q:           "A proof of the Bolzano-Weierstrass theorem uses...",
			correct:     "the nested interval property or repeated bisection",
			distractors: []string{"the ratio test", "the Archimedean property alone", "countability"},
			reason:      "Bisecting a bounded interval that contains infinitely many terms produces a nested sequence of intervals.",
		},
		{
			q:           "A bounded sequence that is not convergent has...",
			correct:     "at least two subsequential limits",
			distractors: []string{"no subsequential limit", "exactly one subsequential limit", "a maximum"},
			reason:      "Bolzano-Weierstrass gives one limit point, and non-convergence forces another.",
		},
		{
			q:           "In a metric space, the Bolzano-Weierstrass property is equivalent to...",
			correct:     "sequential compactness of the space",
			distractors: []string{"completeness alone", "connectedness", "countability"},
			reason:      "A space in which every sequence has a convergent subsequence is exactly a sequentially compact space.",
		},
	}, bwWhichClaim, bwBisectionClaim, bwCosineClaim)
}

// ---------------------------------------------------------------------------
// Cauchy sequences (multiple choice)
// ---------------------------------------------------------------------------

type cauchyGen struct{}

func (g *cauchyGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return mixMC([]mcClaim{
		{
			q:           "A sequence is Cauchy when...",
			correct:     "its terms eventually get and stay arbitrarily close to one another",
			distractors: []string{"its terms increase", "it is bounded", "it has a limit point"},
			reason:      "The definition compares two later terms directly, without naming a limit.",
		},
		{
			q:           "In the real numbers a sequence converges exactly when...",
			correct:     "it is Cauchy",
			distractors: []string{"it is bounded", "it is monotone", "it is bounded and Cauchy"},
			reason:      "This equivalence is the Cauchy completeness of the reals.",
		},
		{
			q:           "The rationals are not complete because...",
			correct:     "a Cauchy sequence of rationals can fail to converge to a rational",
			distractors: []string{"they are countable", "they are ordered", "they are infinite"},
			reason:      "The truncated decimal expansions of the square root of two are a counterexample.",
		},
		{
			q:           "Every Cauchy sequence is...",
			correct:     "bounded",
			distractors: []string{"monotone", "convergent in every ordered field", "eventually constant"},
			reason:      "The later terms are close together and only finitely many earlier terms remain.",
		},
		{
			q:           "A Cauchy sequence that has a convergent subsequence...",
			correct:     "converges to the same limit",
			distractors: []string{"diverges", "is constant", "is unbounded"},
			reason:      "Once the tail is Cauchy and one subsequence is near the limit, the whole tail is near it.",
		},
	}, cauchyWhichClaim, cauchyTrapClaim, cauchyThresholdClaim)
}

// ---------------------------------------------------------------------------
// Comparison and ratio tests (multiple choice)
// ---------------------------------------------------------------------------

type ratioTestGen struct{}

func (g *ratioTestGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return mixMC([]mcClaim{
		{
			q:           "The comparison test concludes convergence when...",
			correct:     "the terms are bounded by those of a convergent series",
			distractors: []string{"the terms decrease", "the terms alternate", "the terms are positive"},
			reason:      "A nonnegative series bounded termwise by a convergent one has bounded partial sums and so converges.",
		},
		{
			q:           "The ratio test is inconclusive when the ratio of successive terms tends to...",
			correct:     "one",
			distractors: []string{"zero", "a number less than one", "a number greater than one"},
			reason:      "At exactly one the test gives no information, as the harmonic series and the convergent reciprocal-square series both show.",
		},
		{
			q:           "If the ratio of successive terms tends to a limit less than one, the series...",
			correct:     "converges absolutely",
			distractors: []string{"diverges", "converges conditionally only", "oscillates"},
			reason:      "Beyond some point the terms shrink at least geometrically.",
		},
		{
			q:           "If the ratio of successive terms tends to a limit greater than one, the series...",
			correct:     "diverges",
			distractors: []string{"converges", "converges conditionally", "is inconclusive"},
			reason:      "The terms eventually grow, so they cannot tend to zero.",
		},
		{
			q:           "The root test is most useful when...",
			correct:     "the terms contain n-th powers",
			distractors: []string{"the terms alternate", "the terms are constant", "the series has finitely many terms"},
			reason:      "Taking the n-th root of a power removes the exponent and exposes the limiting factor.",
		},
	}, testsRatioClaim, testsRatioClaim, testsComparisonClaim, testsRootClaim)
}

// ---------------------------------------------------------------------------
// Absolute and conditional convergence (multiple choice)
// ---------------------------------------------------------------------------

type absoluteGen struct{}

func (g *absoluteGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return mixMC([]mcClaim{
		{
			q:           "A series converges absolutely when...",
			correct:     "the series of the absolute values of its terms converges",
			distractors: []string{"its terms tend to zero", "its terms alternate", "its partial sums are bounded"},
			reason:      "Absolute convergence is convergence of the majorant series of absolute values.",
		},
		{
			q:           "Absolute convergence implies...",
			correct:     "convergence, but not conversely",
			distractors: []string{"divergence", "conditional convergence only", "bounded partial sums only"},
			reason:      "The absolute series controls the original, while the alternating harmonic series converges without its absolute series.",
		},
		{
			q:           "The alternating harmonic series is...",
			correct:     "conditionally convergent",
			distractors: []string{"absolutely convergent", "divergent", "finite"},
			reason:      "It converges, but the harmonic series of absolute values diverges.",
		},
		{
			q:           "A conditionally convergent series...",
			correct:     "converges, while its absolute series diverges",
			distractors: []string{"converges absolutely", "diverges", "has finitely many terms"},
			reason:      "That is the definition of conditional convergence.",
		},
		{
			q:           "The terms of any convergent series tend to...",
			correct:     "zero",
			distractors: []string{"one", "infinity", "a nonzero constant"},
			reason:      "A convergent sequence of partial sums has differences, the terms, tending to zero.",
		},
	}, absoluteClassClaim, absoluteClassClaim, absoluteErrorClaim)
}

// ---------------------------------------------------------------------------
// Rearrangement of series (multiple choice)
// ---------------------------------------------------------------------------

type rearrangementGen struct{}

func (g *rearrangementGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return mixMC([]mcClaim{
		{
			q:           "Riemann's rearrangement theorem says a conditionally convergent series can...",
			correct:     "be rearranged to converge to any real number, or to diverge",
			distractors: []string{"never be rearranged", "only be rearranged finitely many times", "keep its sum under every rearrangement"},
			reason:      "The positive and negative parts both diverge, so their partial sums can be steered anywhere.",
		},
		{
			q:           "An absolutely convergent series...",
			correct:     "has the same sum under every rearrangement",
			distractors: []string{"changes sum under rearrangement", "diverges under rearrangement", "cannot be rearranged"},
			reason:      "Absolute convergence makes the sum independent of the order of the terms.",
		},
		{
			q:           "Rearranging the terms of a conditionally convergent series changes...",
			correct:     "the sum in general",
			distractors: []string{"nothing", "only the first term", "only whether it converges"},
			reason:      "Riemann's theorem shows the sum is not an invariant of a conditionally convergent series.",
		},
		{
			q:           "The rearrangement theorem shows that...",
			correct:     "the commutative law fails for conditionally convergent series",
			distractors: []string{"all series commute", "convergence is impossible", "absolute convergence is unnecessary"},
			reason:      "Reordering is a commutative operation, and the sum changes, so commutativity does not hold here.",
		},
		{
			q:           "A rearrangement of a series...",
			correct:     "uses every term exactly once in a new order",
			distractors: []string{"drops some terms", "repeats some terms", "changes the values of the terms"},
			reason:      "A rearrangement is a bijection from the natural numbers to themselves applied to the order.",
		},
	}, rearrangeGreedyClaim, rearrangeClassClaim)
}

// ---------------------------------------------------------------------------
// The Cauchy criterion for series (multiple choice)
// ---------------------------------------------------------------------------

type cauchyCriterionGen struct{}

func (g *cauchyCriterionGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return mixMC([]mcClaim{
		{
			q:           "The Cauchy criterion for series says a series converges exactly when...",
			correct:     "its tail sums can be made arbitrarily small",
			distractors: []string{"its terms tend to zero", "its terms are positive", "its partial sums are bounded"},
			reason:      "Convergence of the partial sums is their being Cauchy, which is a condition on blocks of terms.",
		},
		{
			q:           "The terms of a convergent series tend to zero, but...",
			correct:     "that condition alone does not imply convergence",
			distractors: []string{"it is sufficient", "it is necessary and sufficient", "it is irrelevant"},
			reason:      "The harmonic series has terms tending to zero yet diverges.",
		},
		{
			q:           "The harmonic series is...",
			correct:     "divergent even though its terms tend to zero",
			distractors: []string{"convergent", "absolutely convergent", "finite"},
			reason:      "Its blocks of equal length each contribute a fixed amount, so the partial sums grow without bound.",
		},
		{
			q:           "The Cauchy criterion is phrased in terms of...",
			correct:     "the difference of two partial sums",
			distractors: []string{"the first term", "the ratio of consecutive terms", "the maximum term"},
			reason:      "It requires that any block of consecutive terms have arbitrarily small sum.",
		},
		{
			q:           "Completeness is what makes the Cauchy criterion equivalent to...",
			correct:     "convergence",
			distractors: []string{"boundedness", "monotonicity", "divergence"},
			reason:      "In a complete space a Cauchy sequence converges, so the block condition guarantees a limit.",
		},
	}, cauchyCritGeometricClaim, cauchyCritHarmonicClaim, cauchyCritTermClaim)
}

// ---------------------------------------------------------------------------
// The sequential criterion (multiple choice)
// ---------------------------------------------------------------------------

type sequentialGen struct{}

func (g *sequentialGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return mixMC([]mcClaim{
		{
			q:           "The sequential criterion for a limit says the limit of f at a is L exactly when...",
			correct:     "for every sequence tending to a within the domain, the images tend to L",
			distractors: []string{"for one particular sequence", "the function is bounded", "the function is continuous"},
			reason:      "The limit is characterised by the behaviour of every admissible sequence.",
		},
		{
			q:           "The sequential criterion is used to...",
			correct:     "show a limit does not exist by finding two sequences with different image limits",
			distractors: []string{"compute derivatives", "sum series", "prove countability"},
			reason:      "If two sequences approaching a give different image limits, no single limit can exist.",
		},
		{
			q:           "The limit of f at a depends on...",
			correct:     "the behaviour of f near a, not at a",
			distractors: []string{"the value at a only", "the value at infinity", "the derivative at a"},
			reason:      "The definition excludes the point a itself from the requirement.",
		},
		{
			q:           "A function has a limit at a only if...",
			correct:     "all sequences in the domain approaching a give the same image limit",
			distractors: []string{"it is defined at a", "it is continuous at a", "it is bounded"},
			reason:      "Uniqueness of the image limit across all sequences is exactly the criterion.",
		},
		{
			q:           "The sequential criterion links limits of functions to...",
			correct:     "limits of sequences",
			distractors: []string{"continuity of derivatives", "countability", "series"},
			reason:      "It reduces every statement about a function limit to a statement about sequences.",
		},
	}, sequentialSinClaim, sequentialStepClaim, sequentialDirichletClaim, sequentialRemovableClaim)
}

// ---------------------------------------------------------------------------
// Continuity and its algebra (multiple choice)
// ---------------------------------------------------------------------------

type continuityGen struct{}

func (g *continuityGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return mixMC([]mcClaim{
		{
			q:           "A function is continuous at a when...",
			correct:     "the limit at a equals the value at a",
			distractors: []string{"it is bounded near a", "it is differentiable at a", "it has a limit at infinity"},
			reason:      "Continuity is the meeting of the limit and the value.",
		},
		{
			q:           "Sums, products and quotients of continuous functions are continuous...",
			correct:     "wherever the quotient is defined",
			distractors: []string{"always", "never", "only for polynomials"},
			reason:      "The algebra of limits gives the result immediately; a quotient needs its denominator nonzero.",
		},
		{
			q:           "A composition of continuous functions is...",
			correct:     "continuous",
			distractors: []string{"discontinuous", "differentiable", "bounded"},
			reason:      "A small change in the input gives a small change in the intermediate value and hence in the output.",
		},
		{
			q:           "Continuity at a point can be phrased with...",
			correct:     "epsilon and delta",
			distractors: []string{"only sequences", "only derivatives", "countability"},
			reason:      "For every epsilon there is a delta making the images close whenever the inputs are close.",
		},
		{
			q:           "A function continuous on a closed bounded interval is...",
			correct:     "bounded and attains its bounds",
			distractors: []string{"unbounded", "constant", "differentiable"},
			reason:      "This is the extreme value theorem, a consequence of compactness.",
		},
	}, continuityPiecewiseClaim, continuityFillClaim, continuityWhichClaim)
}

// ---------------------------------------------------------------------------
// The intermediate value theorem (multiple choice)
// ---------------------------------------------------------------------------

type ivtGen struct{}

func (g *ivtGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return mixMC([]mcClaim{
		{
			q:           "The intermediate value theorem says a continuous function on an interval...",
			correct:     "takes every value between its endpoint values",
			distractors: []string{"is increasing", "attains a maximum", "is differentiable"},
			reason:      "Continuity forces the graph to pass through every intermediate height.",
		},
		{
			q:           "The intermediate value theorem is used to...",
			correct:     "show that an equation has a solution",
			distractors: []string{"compute a derivative", "sum a series", "prove countability"},
			reason:      "A sign change at the endpoints gives a zero inside the interval.",
		},
		{
			q:           "The intermediate value theorem needs...",
			correct:     "continuity on an interval",
			distractors: []string{"differentiability", "monotonicity", "boundedness alone"},
			reason:      "Differentiability is stronger than needed, and mere boundedness is far too weak.",
		},
		{
			q:           "A continuous function on a closed interval that changes sign...",
			correct:     "has a zero inside the interval",
			distractors: []string{"is increasing", "is constant", "has no zero"},
			reason:      "The value zero lies between the two endpoint values of opposite sign.",
		},
		{
			q:           "The intermediate value theorem ultimately follows from...",
			correct:     "the completeness of the reals",
			distractors: []string{"the Archimedean property alone", "countability", "the ratio test"},
			reason:      "The proof locates the intermediate value as a supremum, which needs completeness.",
		},
	}, ivtCubicClaim, ivtValueClaim, ivtBisectClaim)
}

// ---------------------------------------------------------------------------
// The extreme value theorem (multiple choice)
// ---------------------------------------------------------------------------

type evtGen struct{}

func (g *evtGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return mixMC([]mcClaim{
		{
			q:           "The extreme value theorem says a continuous function on a closed bounded interval...",
			correct:     "attains a maximum and a minimum",
			distractors: []string{"is constant", "is differentiable", "is unbounded"},
			reason:      "The image of a compact interval is compact, hence closed and bounded, so the bounds are attained.",
		},
		{
			q:           "The extreme value theorem fails on an open interval because...",
			correct:     "the endpoints can be approached without being reached",
			distractors: []string{"the function is unbounded", "the function is discontinuous", "the interval is too long"},
			reason:      "The function may climb toward a value it never equals, as x does on the open interval from zero to one.",
		},
		{
			q:           "The extreme value theorem requires...",
			correct:     "compactness of the interval",
			distractors: []string{"differentiability", "monotonicity", "countability"},
			reason:      "Closed and bounded is exactly what the theorem needs.",
		},
		{
			q:           "A continuous function on a compact set is...",
			correct:     "bounded and attains its bounds",
			distractors: []string{"increasing", "constant", "linear"},
			reason:      "This is the general form of the extreme value theorem.",
		},
		{
			q:           "Together the intermediate and extreme value theorems describe...",
			correct:     "the global behaviour of a continuous function on a closed interval",
			distractors: []string{"its derivative", "its integral", "its countability"},
			reason:      "One gives the range of values, the other the attainment of the extreme ones.",
		},
	}, evtQuadraticClaim, evtQuadraticClaim, evtDomainClaim)
}

// ---------------------------------------------------------------------------
// Uniform continuity (multiple choice)
// ---------------------------------------------------------------------------

type uniformGen struct{}

func (g *uniformGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return mixMC([]mcClaim{
		{
			q:           "A function is uniformly continuous when...",
			correct:     "one delta works for every point of the domain",
			distractors: []string{"it is continuous", "it is bounded", "it is differentiable"},
			reason:      "Uniformity is the independence of delta from the point, depending only on epsilon.",
		},
		{
			q:           "Continuity on a compact set implies...",
			correct:     "uniform continuity",
			distractors: []string{"differentiability", "boundedness of the derivative", "monotonicity"},
			reason:      "This is one of the main theorems about compactness and continuity.",
		},
		{
			q:           "The function x squared is continuous but not uniformly continuous on...",
			correct:     "the whole real line",
			distractors: []string{"any closed interval", "the interval from zero to one", "a compact set"},
			reason:      "On an unbounded domain the same delta cannot serve points that are far apart.",
		},
		{
			q:           "Uniform continuity means delta depends on...",
			correct:     "epsilon alone",
			distractors: []string{"the point and epsilon", "nothing at all", "the derivative"},
			reason:      "The choice of delta must be uniform over the domain.",
		},
		{
			q:           "A uniformly continuous function maps Cauchy sequences to...",
			correct:     "Cauchy sequences",
			distractors: []string{"divergent sequences", "bounded sequences only", "constant sequences"},
			reason:      "Close terms stay close after the mapping, uniformly, which is exactly what is needed.",
		},
	}, uniformLinearDeltaClaim, uniformSquareDeltaClaim, uniformWhichClaim)
}

// ---------------------------------------------------------------------------
// Compactness and Heine-Borel (multiple choice)
// ---------------------------------------------------------------------------

type compactGen struct{}

func (g *compactGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return mixMC([]mcClaim{
		{
			q:           "The Heine-Borel theorem says a subset of the reals is compact exactly when it is...",
			correct:     "closed and bounded",
			distractors: []string{"open and bounded", "closed and unbounded", "connected"},
			reason:      "For the real line, closed and bounded is equivalent to every open cover having a finite subcover.",
		},
		{
			q:           "A continuous function on a compact set is...",
			correct:     "uniformly continuous",
			distractors: []string{"discontinuous", "unbounded", "constant"},
			reason:      "Compactness upgrades continuity to uniform continuity.",
		},
		{
			q:           "Compactness for the reals is equivalent to...",
			correct:     "every sequence having a convergent subsequence inside the set",
			distractors: []string{"being open", "being countable", "being disconnected"},
			reason:      "This is sequential compactness, equivalent to compactness in metric spaces.",
		},
		{
			q:           "The open interval from zero to one is not compact because...",
			correct:     "it is not closed",
			distractors: []string{"it is unbounded", "it is empty", "it is countable"},
			reason:      "The sequence of points one over n has no subsequence converging to a point of the set.",
		},
		{
			q:           "A finite union of compact sets is...",
			correct:     "compact",
			distractors: []string{"open", "unbounded", "connected"},
			reason:      "A finite subcover can be taken on each compact piece and combined.",
		},
	}, compactWhichClaim, compactWhichClaim, compactWitnessClaim)
}
