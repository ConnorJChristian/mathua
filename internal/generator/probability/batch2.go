package probability

// The second probability-theory batch: twenty-five concepts that close the
// domain with joint and conditional distributions, covariance and correlation,
// the modes of convergence and their limit theorems, characteristic functions and
// stochastic processes.
//
// The first batch ended at the classical limit inequalities and the central limit
// theorem. This batch starts where those leave off: it conditions one variable on
// another, measures linear dependence, separates the modes of convergence, and
// introduces the transform that proves the limit theorems and the processes that
// run in time.
//
// Twelve concepts are multiple choice and thirteen are numeric. Every numeric
// generator publishes the parameters its answer follows from, so the tests
// recompute the answer rather than trusting the author's arithmetic.

import (
	"math/rand"
	"strconv"
	"strings"

	"github.com/chuma-beep/mathua/internal/generator"
)

// ---------------------------------------------------------------------------
// Marginal distributions (numeric)
// ---------------------------------------------------------------------------

type marginalGen struct{}

func (g *marginalGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		weights []int
		reason  string
	}
	claims := []claim{
		{weights: []int{1, 2, 3, 4}, reason: "X is one at the last two pairs, which carry 3 and 4 of the total 10, so the marginal is 7/10 = 0.7."},
		{weights: []int{2, 2, 1, 5}, reason: "(1 + 5)/10 = 0.6."},
		{weights: []int{3, 3, 1, 3}, reason: "(1 + 3)/10 = 0.4."},
		{weights: []int{4, 1, 1, 4}, reason: "(1 + 4)/10 = 0.5."},
		{weights: []int{5, 2, 2, 1}, reason: "(2 + 1)/10 = 0.3."},
		{weights: []int{1, 1, 2, 6}, reason: "(2 + 6)/10 = 0.8."},
	}
	c := claims[rand.Intn(len(claims))]
	total := 0
	for _, w := range c.weights {
		total += w
	}
	answer := decimal6(float64(c.weights[2]+c.weights[3]) / float64(total))
	parts := make([]string, len(c.weights))
	for i, w := range c.weights {
		parts[i] = strconv.Itoa(w)
	}
	return generator.Problem{
		Question: "A joint mass function on the four pairs (0,0), (0,1), (1,0) and (1,1) is proportional to the weights " +
			strings.Join(parts, ", ") + " in that order. What is the marginal P(X = 1)? (enter a decimal)",
		Answer:      answer,
		Explanation: "The marginal sums the joint masses over all values of the other variable; " + c.reason,
		Facts: map[string]string{
			"weights": strings.Join(parts, ","), "total": strconv.Itoa(total), "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// Joint densities (numeric)
// ---------------------------------------------------------------------------

type jointDensityGen struct{}

func (g *jointDensityGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		a, b, x, y int
		reason     string
	}
	claims := []claim{
		{a: 4, b: 5, x: 2, y: 3, reason: "(2/4)(3/5) = 0.3."},
		{a: 10, b: 10, x: 5, y: 4, reason: "(5/10)(4/10) = 0.2."},
		{a: 2, b: 5, x: 1, y: 4, reason: "(1/2)(4/5) = 0.4."},
		{a: 5, b: 5, x: 3, y: 4, reason: "(3/5)(4/5) = 0.48."},
		{a: 8, b: 10, x: 4, y: 5, reason: "(4/8)(5/10) = 0.25."},
		{a: 5, b: 4, x: 1, y: 1, reason: "(1/5)(1/4) = 0.05."},
	}
	c := claims[rand.Intn(len(claims))]
	answer := decimal6(float64(c.x) / float64(c.a) * float64(c.y) / float64(c.b))
	return generator.Problem{
		Question: "A pair (X, Y) is uniform on the rectangle from 0 to " + strconv.Itoa(c.a) +
			" in x and 0 to " + strconv.Itoa(c.b) + " in y. What is P(X ≤ " + strconv.Itoa(c.x) +
			" and Y ≤ " + strconv.Itoa(c.y) + ")? (enter a decimal)",
		Answer:      answer,
		Explanation: "A uniform pair has independent coordinates, so the joint probability is the product of the two marginal probabilities; " + c.reason,
		Facts: map[string]string{
			"a": strconv.Itoa(c.a), "b": strconv.Itoa(c.b), "x": strconv.Itoa(c.x),
			"y": strconv.Itoa(c.y), "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// Independent random variables (multiple choice)
// ---------------------------------------------------------------------------

type rvIndependenceGen struct{}

func (g *rvIndependenceGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		question    string
		correct     string
		distractors []string
		reason      string
	}
	claims := []claim{
		{
			question:    "Two random variables X and Y are independent when...",
			correct:     "the joint distribution factors into the product of the marginals everywhere",
			distractors: []string{"their covariance is zero", "they are uncorrelated", "they have the same distribution"},
			reason:      "Independence is the product rule for the whole joint law, which is strictly stronger than a zero covariance.",
		},
		{
			question:    "If X and Y are independent, then E[XY] equals...",
			correct:     "E[X]E[Y]",
			distractors: []string{"E[X] + E[Y]", "zero", "Var(X)Var(Y)"},
			reason:      "Independence factors the expectation of the product into the product of the expectations.",
		},
		{
			question:    "Independence of two variables with finite variances implies...",
			correct:     "zero covariance",
			distractors: []string{"equal means", "a symmetric joint density", "zero variance"},
			reason:      "E[XY] = E[X]E[Y] makes the covariance E[XY] − E[X]E[Y] vanish.",
		},
		{
			question:    "Which of these is not implied by a zero covariance?",
			correct:     "independence",
			distractors: []string{"uncorrelatedness", "finite means", "finite variances"},
			reason:      "Uncorrelated does not imply independent; a nonlinear dependence can leave the covariance at zero.",
		},
		{
			question:    "For independent continuous variables the joint density is...",
			correct:     "the product of the marginal densities",
			distractors: []string{"the sum of the marginal densities", "the average of the marginal densities", "the ratio of the marginal densities"},
			reason:      "Independence factors the joint density just as it factors the joint mass function.",
		},
	}
	c := claims[rand.Intn(len(claims))]
	p := generator.MultipleChoice(c.question, c.correct, c.distractors...)
	p.Explanation = c.reason
	return p
}

// ---------------------------------------------------------------------------
// Conditional expectation (numeric)
// ---------------------------------------------------------------------------

type conditionalGen struct{}

func (g *conditionalGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		weights []int
		y       int
		reason  string
	}
	claims := []claim{
		{weights: []int{1, 1, 3, 1}, y: 0, reason: "With Y = 0 the only pairs are (0,0) and (1,0), and X is one with chance 3/(1+3) = 0.75."},
		{weights: []int{3, 1, 1, 1}, y: 0, reason: "X is one with chance 1/(3+1) = 0.25."},
		{weights: []int{2, 2, 3, 3}, y: 0, reason: "X is one with chance 3/(2+3) = 0.6."},
		{weights: []int{4, 1, 1, 4}, y: 1, reason: "With Y = 1 the pairs are (0,1) and (1,1), and X is one with chance 4/(1+4) = 0.8."},
		{weights: []int{3, 7, 2, 8}, y: 0, reason: "X is one with chance 2/(3+2) = 0.4."},
		{weights: []int{1, 9, 9, 1}, y: 1, reason: "X is one with chance 1/(9+1) = 0.1."},
	}
	c := claims[rand.Intn(len(claims))]
	// Pairs are (0,0), (0,1), (1,0), (1,1), so X = 0 at index y and X = 1 at index 2+y.
	num := c.weights[2+c.y]
	den := c.weights[c.y] + num
	answer := decimal6(float64(num) / float64(den))
	parts := make([]string, len(c.weights))
	for i, w := range c.weights {
		parts[i] = strconv.Itoa(w)
	}
	return generator.Problem{
		Question: "A joint mass function on the four pairs (0,0), (0,1), (1,0) and (1,1) is proportional to the weights " +
			strings.Join(parts, ", ") + " in that order. What is E[X | Y = " + strconv.Itoa(c.y) + "]? (enter a decimal)",
		Answer:      answer,
		Explanation: "Conditioning on Y fixes a column of the table, and the conditional mean is the weighted average of X within it; " + c.reason,
		Facts: map[string]string{
			"weights": strings.Join(parts, ","), "y": strconv.Itoa(c.y), "answer": answer,
			"w0": strconv.Itoa(c.weights[c.y]), "w1": strconv.Itoa(num), "col": strconv.Itoa(den),
		},
	}
}

// ---------------------------------------------------------------------------
// The law of total expectation (numeric)
// ---------------------------------------------------------------------------

type totalExpectationGen struct{}

func (g *totalExpectationGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		ca, cb, pn, pd int
		reason         string
	}
	claims := []claim{
		{ca: 1, cb: 3, pn: 1, pd: 2, reason: "(1/2)(1) + (1/2)(3) = 2."},
		{ca: 2, cb: 4, pn: 1, pd: 4, reason: "(3/4)(2) + (1/4)(4) = 2.5."},
		{ca: 0, cb: 5, pn: 1, pd: 5, reason: "(4/5)(0) + (1/5)(5) = 1."},
		{ca: 3, cb: 7, pn: 1, pd: 2, reason: "(1/2)(3) + (1/2)(7) = 5."},
		{ca: 1, cb: 2, pn: 3, pd: 4, reason: "(1/4)(1) + (3/4)(2) = 1.75."},
		{ca: 4, cb: 8, pn: 1, pd: 2, reason: "(1/2)(4) + (1/2)(8) = 6."},
	}
	c := claims[rand.Intn(len(claims))]
	p := float64(c.pn) / float64(c.pd)
	answer := decimal6((1-p)*float64(c.ca) + p*float64(c.cb))
	return generator.Problem{
		Question: "A variable Y takes the values zero and one with P(Y = 1) = " + strconv.Itoa(c.pn) + "/" +
			strconv.Itoa(c.pd) + ". Given Y, the conditional mean of X is " + strconv.Itoa(c.ca) +
			" when Y is zero and " + strconv.Itoa(c.cb) + " when Y is one. What is E[X]? (enter a decimal)",
		Answer:      answer,
		Explanation: "The law of total expectation averages the conditional means with the probabilities of the conditioning variable; " + c.reason,
		Facts: map[string]string{
			"ca": strconv.Itoa(c.ca), "cb": strconv.Itoa(c.cb),
			"pn": strconv.Itoa(c.pn), "pd": strconv.Itoa(c.pd), "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// The law of total variance (numeric)
// ---------------------------------------------------------------------------

type totalVarianceGen struct{}

func (g *totalVarianceGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		m0, m1, v0, v1 int
		reason         string
	}
	claims := []claim{
		{m0: 0, m1: 2, v0: 1, v1: 3, reason: "E[Var(X|Y)] = 2 and Var(E[X|Y]) = (0 − 2)²/4 = 1, so Var(X) = 3."},
		{m0: 1, m1: 3, v0: 2, v1: 2, reason: "E[Var(X|Y)] = 2 and Var(E[X|Y]) = 1, so Var(X) = 3."},
		{m0: 0, m1: 4, v0: 1, v1: 1, reason: "E[Var(X|Y)] = 1 and Var(E[X|Y]) = 4, so Var(X) = 5."},
		{m0: 2, m1: 4, v0: 2, v1: 4, reason: "E[Var(X|Y)] = 3 and Var(E[X|Y]) = 1, so Var(X) = 4."},
		{m0: 0, m1: 0, v0: 1, v1: 2, reason: "E[Var(X|Y)] = 1.5 and Var(E[X|Y]) = 0, so Var(X) = 1.5."},
		{m0: 1, m1: 4, v0: 3, v1: 5, reason: "E[Var(X|Y)] = 4 and Var(E[X|Y]) = (1 − 4)²/4 = 2.25, so Var(X) = 6.25."},
	}
	c := claims[rand.Intn(len(claims))]
	between := float64((c.m0-c.m1)*(c.m0-c.m1)) / 4.0
	within := float64(c.v0+c.v1) / 2.0
	answer := decimal6(within + between)
	return generator.Problem{
		Question: "A variable Y takes the values zero and one with equal probability. Given Y, the conditional mean of X is " +
			strconv.Itoa(c.m0) + " or " + strconv.Itoa(c.m1) + ", and the conditional variance is " +
			strconv.Itoa(c.v0) + " or " + strconv.Itoa(c.v1) + ". What is Var(X)? (enter a decimal)",
		Answer:      answer,
		Explanation: "The law of total variance adds the mean of the conditional variances to the variance of the conditional means; " + c.reason,
		Facts: map[string]string{
			"m0": strconv.Itoa(c.m0), "m1": strconv.Itoa(c.m1),
			"v0": strconv.Itoa(c.v0), "v1": strconv.Itoa(c.v1), "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// Covariance (numeric)
// ---------------------------------------------------------------------------

type covarianceGen struct{}

func (g *covarianceGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		weights []int
		reason  string
	}
	claims := []claim{
		{weights: []int{1, 1, 1, 7}, reason: "E[XY] = 0.7, E[X] = E[Y] = 0.8, so the covariance is 0.7 − 0.64 = 0.06."},
		{weights: []int{2, 1, 1, 6}, reason: "E[XY] = 0.6 and E[X] = E[Y] = 0.7, so 0.6 − 0.49 = 0.11."},
		{weights: []int{1, 1, 2, 6}, reason: "E[XY] = 0.6, E[X] = 0.8 and E[Y] = 0.7, so 0.6 − 0.56 = 0.04."},
		{weights: []int{2, 2, 1, 5}, reason: "E[XY] = 0.5, E[X] = 0.6 and E[Y] = 0.7, so 0.5 − 0.42 = 0.08."},
		{weights: []int{3, 1, 1, 5}, reason: "E[XY] = 0.5 and E[X] = E[Y] = 0.6, so 0.5 − 0.36 = 0.14."},
		{weights: []int{4, 1, 1, 4}, reason: "E[XY] = 0.4 and E[X] = E[Y] = 0.5, so 0.4 − 0.25 = 0.15."},
	}
	c := claims[rand.Intn(len(claims))]
	total := 0
	for _, w := range c.weights {
		total += w
	}
	_, w01, w10, w11 := c.weights[0], c.weights[1], c.weights[2], c.weights[3]
	// E[XY] = w11/T; E[X] = (w10+w11)/T; E[Y] = (w01+w11)/T. Keep it in integers.
	num := w11*total - (w10+w11)*(w01+w11)
	answer := decimal6(float64(num) / float64(total*total))
	parts := make([]string, len(c.weights))
	for i, w := range c.weights {
		parts[i] = strconv.Itoa(w)
	}
	exy := decimal6(float64(w11) / float64(total))
	ex := decimal6(float64(w10+w11) / float64(total))
	ey := decimal6(float64(w01+w11) / float64(total))
	return generator.Problem{
		Question: "A joint mass function on the four pairs (0,0), (0,1), (1,0) and (1,1) is proportional to the weights " +
			strings.Join(parts, ", ") + " in that order. What is Cov(X, Y) = E[XY] − E[X]E[Y]? (enter a decimal)",
		Answer:      answer,
		Explanation: "Only the pair (1,1) contributes to E[XY], and the two means come from the margins; " + c.reason,
		Facts: map[string]string{
			"weights": strings.Join(parts, ","), "total": strconv.Itoa(total), "answer": answer,
			"w11": strconv.Itoa(w11), "wx": strconv.Itoa(w10 + w11), "wy": strconv.Itoa(w01 + w11),
			"exy": exy, "ex": ex, "ey": ey, "exey": decimal6(float64((w10+w11)*(w01+w11)) / float64(total*total)),
		},
	}
}

// ---------------------------------------------------------------------------
// Correlation (numeric)
// ---------------------------------------------------------------------------

type correlationGen struct{}

func (g *correlationGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		cov, sx, sy int
		reason      string
	}
	claims := []claim{
		{cov: 6, sx: 3, sy: 5, reason: "6/(3·5) = 6/15 = 0.4."},
		{cov: 9, sx: 3, sy: 5, reason: "9/15 = 0.6."},
		{cov: 4, sx: 2, sy: 4, reason: "4/8 = 0.5."},
		{cov: 2, sx: 4, sy: 5, reason: "2/20 = 0.1."},
		{cov: 14, sx: 4, sy: 5, reason: "14/20 = 0.7."},
		{cov: 3, sx: 2, sy: 4, reason: "3/8 = 0.375."},
	}
	c := claims[rand.Intn(len(claims))]
	answer := decimal6(float64(c.cov) / float64(c.sx*c.sy))
	return generator.Problem{
		Question: "X and Y have covariance " + strconv.Itoa(c.cov) + " and standard deviations " +
			strconv.Itoa(c.sx) + " and " + strconv.Itoa(c.sy) + ". What is their correlation? (enter a decimal)",
		Answer:      answer,
		Explanation: "Correlation is the covariance divided by the product of the standard deviations; " + c.reason,
		Facts: map[string]string{
			"cov": strconv.Itoa(c.cov), "sx": strconv.Itoa(c.sx), "sy": strconv.Itoa(c.sy), "answer": answer,
			"sxsy": strconv.Itoa(c.sx * c.sy),
		},
	}
}

// ---------------------------------------------------------------------------
// Convergence in probability (multiple choice)
// ---------------------------------------------------------------------------

type convergenceInProbabilityGen struct{}

func (g *convergenceInProbabilityGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		question    string
		correct     string
		distractors []string
		reason      string
	}
	claims := []claim{
		{
			question:    "Convergence in probability of X_n to X means that for every positive ε...",
			correct:     "P(|X_n − X| > ε) tends to zero",
			distractors: []string{"X_n equals X eventually", "the distribution functions converge pointwise", "the means converge"},
			reason:      "The defining requirement is that the probability of a departure larger than ε vanishes.",
		},
		{
			question:    "Convergence in probability implies...",
			correct:     "convergence in distribution",
			distractors: []string{"almost-sure convergence", "convergence of the densities", "convergence of every moment"},
			reason:      "The CDFs of a sequence converging in probability converge weakly; the converse fails.",
		},
		{
			question:    "The weak law of large numbers is a statement of...",
			correct:     "convergence in probability",
			distractors: []string{"almost-sure convergence", "convergence in distribution", "convergence in mean square"},
			reason:      "The sample mean converges in probability to the population mean.",
		},
		{
			question:    "Which of these does convergence in probability not imply?",
			correct:     "almost-sure convergence",
			distractors: []string{"convergence in distribution", "uniqueness of the limit in probability", "the existence of an almost-surely convergent subsequence"},
			reason:      "Almost-sure convergence is strictly stronger; a subsequence converges almost surely, but not the whole sequence.",
		},
		{
			question:    "If X_n converges in probability to a constant, then X_n also converges to it...",
			correct:     "in distribution",
			distractors: []string{"almost surely", "in mean square", "uniformly"},
			reason:      "Convergence in probability is stronger than convergence in distribution in general.",
		},
	}
	c := claims[rand.Intn(len(claims))]
	p := generator.MultipleChoice(c.question, c.correct, c.distractors...)
	p.Explanation = c.reason
	return p
}

// ---------------------------------------------------------------------------
// Convergence in distribution (multiple choice)
// ---------------------------------------------------------------------------

type convergenceInDistributionGen struct{}

func (g *convergenceInDistributionGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		question    string
		correct     string
		distractors []string
		reason      string
	}
	claims := []claim{
		{
			question:    "Convergence in distribution means that...",
			correct:     "the distribution functions converge at every point where the limit is continuous",
			distractors: []string{"the variables converge pointwise", "the probabilities of every event converge", "the densities converge pointwise"},
			reason:      "Weak convergence constrains the laws, not the variables or their densities.",
		},
		{
			question:    "Convergence in distribution is equivalent to...",
			correct:     "E[g(X_n)] tending to E[g(X)] for every bounded continuous g",
			distractors: []string{"convergence of all moments", "convergence in probability", "almost-sure convergence"},
			reason:      "The portmanteau characterisation uses bounded continuous test functions.",
		},
		{
			question:    "Convergence in distribution alone does not determine...",
			correct:     "the joint behaviour of the variables",
			distractors: []string{"the limiting distribution", "the limiting distribution function", "the limit of E[g(X_n)] for bounded continuous g"},
			reason:      "Each variable is controlled only through its own law, so joint structure is invisible.",
		},
		{
			question:    "The central limit theorem is a statement of...",
			correct:     "convergence in distribution",
			distractors: []string{"convergence in probability", "almost-sure convergence", "uniform convergence of the densities"},
			reason:      "The standardised sum converges weakly to the standard normal.",
		},
		{
			question:    "A sequence can converge in distribution but not in probability when...",
			correct:     "the variables are not all defined on the same probability space",
			distractors: []string{"the variables are independent", "the variables are identically distributed", "the limit is normal"},
			reason:      "Weak convergence is a statement about the laws and does not relate the variables to a common limit.",
		},
	}
	c := claims[rand.Intn(len(claims))]
	p := generator.MultipleChoice(c.question, c.correct, c.distractors...)
	p.Explanation = c.reason
	return p
}

// ---------------------------------------------------------------------------
// Almost-sure convergence (multiple choice)
// ---------------------------------------------------------------------------

type almostSureGen struct{}

func (g *almostSureGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		question    string
		correct     string
		distractors []string
		reason      string
	}
	claims := []claim{
		{
			question:    "Almost-sure convergence of X_n to X means that...",
			correct:     "the set of outcomes on which X_n converges to X has probability one",
			distractors: []string{"P(|X_n − X| > ε) tends to zero for each ε", "the distribution functions converge", "the means converge"},
			reason:      "The convergence happens for almost every outcome, a statement about the sample paths.",
		},
		{
			question:    "Almost-sure convergence implies...",
			correct:     "convergence in probability",
			distractors: []string{"the reverse implication", "nothing about convergence in distribution", "uniform convergence"},
			reason:      "The strong mode implies the weak one, and the converse does not hold.",
		},
		{
			question:    "The strong law of large numbers is a statement of...",
			correct:     "almost-sure convergence",
			distractors: []string{"convergence in probability", "convergence in distribution", "convergence in mean square"},
			reason:      "The sample mean converges to the population mean for almost every outcome.",
		},
		{
			question:    "A sequence that converges in probability has...",
			correct:     "an almost-surely convergent subsequence",
			distractors: []string{"a unique almost-sure limit on the whole sequence", "no convergent subsequence", "a normally distributed limit"},
			reason:      "The subsequence theorem extracts almost-sure convergence along a subsequence.",
		},
		{
			question:    "The Borel-Cantelli lemma is the tool that turns a summable tail of probabilities into...",
			correct:     "an almost-sure statement",
			distractors: []string{"a convergence-in-probability statement", "a distributional statement", "a monotonicity statement"},
			reason:      "It shows that a finite total probability leaves only finitely many of the events occurring, almost surely.",
		},
	}
	c := claims[rand.Intn(len(claims))]
	p := generator.MultipleChoice(c.question, c.correct, c.distractors...)
	p.Explanation = c.reason
	return p
}

// ---------------------------------------------------------------------------
// The strong law of large numbers (multiple choice)
// ---------------------------------------------------------------------------

type strongLawGen struct{}

func (g *strongLawGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		question    string
		correct     string
		distractors []string
		reason      string
	}
	claims := []claim{
		{
			question:    "The strong law of large numbers says that the sample mean converges to the population mean...",
			correct:     "almost surely",
			distractors: []string{"in probability", "in distribution", "in mean square"},
			reason:      "The strong law is the almost-sure version of the law of large numbers.",
		},
		{
			question:    "The strong law needs the observations to...",
			correct:     "be independent and identically distributed with a finite mean",
			distractors: []string{"have a finite variance", "be normally distributed", "be bounded"},
			reason:      "A finite mean and independence are enough; a finite variance is not required.",
		},
		{
			question:    "Compared with the weak law, the strong law draws...",
			correct:     "a stronger conclusion",
			distractors: []string{"a weaker conclusion", "an equivalent conclusion", "a conclusion only about variances"},
			reason:      "Almost-sure convergence implies convergence in probability, so the strong law is stronger.",
		},
		{
			question:    "The strong law can fail for a distribution with...",
			correct:     "an infinite mean",
			distractors: []string{"an infinite variance", "a skewed shape", "a bounded support"},
			reason:      "The Cauchy distribution has no mean, and its sample average does not settle down.",
		},
		{
			question:    "The almost-sure limit of the sample mean is...",
			correct:     "the population mean",
			distractors: []string{"the median", "the standard deviation", "the mode"},
			reason:      "The limit is the expectation of the distribution, when it exists.",
		},
	}
	c := claims[rand.Intn(len(claims))]
	p := generator.MultipleChoice(c.question, c.correct, c.distractors...)
	p.Explanation = c.reason
	return p
}

// ---------------------------------------------------------------------------
// The Borel-Cantelli lemmas (multiple choice)
// ---------------------------------------------------------------------------

type borelCantelliGen struct{}

func (g *borelCantelliGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		question    string
		correct     string
		distractors []string
		reason      string
	}
	claims := []claim{
		{
			question:    "The first Borel-Cantelli lemma says that if the probabilities of the events sum to a finite total, then...",
			correct:     "almost surely only finitely many of the events occur",
			distractors: []string{"almost surely infinitely many occur", "the events are independent", "the limit of the probabilities is zero"},
			reason:      "A summable total probability leaves only finitely many events occurring, and no independence is needed.",
		},
		{
			question:    "The second Borel-Cantelli lemma needs the events to be independent and the probabilities to...",
			correct:     "sum to infinity",
			distractors: []string{"sum to a finite total", "be equal", "decrease to zero"},
			reason:      "Independence with a divergent sum forces infinitely many of the events to occur.",
		},
		{
			question:    "When the events are independent and their probabilities sum to infinity, almost surely...",
			correct:     "infinitely many of them occur",
			distractors: []string{"only finitely many occur", "exactly one occurs", "none occur"},
			reason:      "This is the content of the second lemma.",
		},
		{
			question:    "Borel-Cantelli is used to prove...",
			correct:     "almost-sure results such as the strong law of large numbers",
			distractors: []string{"the central limit theorem", "Chebyshev's inequality", "Bayes' theorem"},
			reason:      "It converts statements about sums of probabilities into almost-sure statements about tails.",
		},
		{
			question:    "Both lemmas concern the probability that...",
			correct:     "infinitely many of a sequence of events occur",
			distractors: []string{"exactly one of the events occurs", "the union of the first finitely many occurs", "the intersection of all the events occurs"},
			reason:      "The event of interest is the limsup of the sequence, the set of outcomes in infinitely many of the events.",
		},
	}
	c := claims[rand.Intn(len(claims))]
	p := generator.MultipleChoice(c.question, c.correct, c.distractors...)
	p.Explanation = c.reason
	return p
}

// ---------------------------------------------------------------------------
// Slutsky's theorem (multiple choice)
// ---------------------------------------------------------------------------

type slutskyGen struct{}

func (g *slutskyGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		question    string
		correct     string
		distractors []string
		reason      string
	}
	claims := []claim{
		{
			question:    "Slutsky's theorem says that if X_n converges in distribution to X and Y_n converges in probability to a constant c, then X_n + Y_n converges in distribution to...",
			correct:     "X + c",
			distractors: []string{"the standard normal", "c times X", "the variable Y_n"},
			reason:      "A probability limit may be added into a distributional limit as a constant.",
		},
		{
			question:    "Slutsky's theorem is most often used to replace a random denominator by...",
			correct:     "its probability limit",
			distractors: []string{"its expectation", "its variance", "its median"},
			reason:      "A statistic divided by a consistent estimate of its scale converges to the standard limit.",
		},
		{
			question:    "The theorem combines convergence of one sequence in distribution with convergence of the other...",
			correct:     "in probability",
			distractors: []string{"in probability for both", "in distribution for both", "almost surely for both"},
			reason:      "One factor may be weak and the other must be strong enough to collapse to a constant.",
		},
		{
			question:    "If Y_n converges in probability to a nonzero constant c, then 1/Y_n converges in probability to...",
			correct:     "1/c",
			distractors: []string{"c squared", "zero", "one"},
			reason:      "The reciprocal is continuous at a nonzero limit, so the continuous mapping theorem applies.",
		},
		{
			question:    "Slutsky's theorem follows from...",
			correct:     "the continuous mapping theorem together with the weak convergence of the distributional factor",
			distractors: []string{"the strong law alone", "Chebyshev's inequality", "the Borel-Cantelli lemmas"},
			reason:      "It is an application of continuous mapping to a pair that is tight in one coordinate.",
		},
	}
	c := claims[rand.Intn(len(claims))]
	p := generator.MultipleChoice(c.question, c.correct, c.distractors...)
	p.Explanation = c.reason
	return p
}

// ---------------------------------------------------------------------------
// The delta method (numeric)
// ---------------------------------------------------------------------------

type deltaMethodGen struct{}

func (g *deltaMethodGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		fn      string // "square" or "sqrt"
		mu, sig int
		reason  string
	}
	claims := []claim{
		{fn: "square", mu: 2, sig: 1, reason: "The derivative is 2μ = 4, so the variance is 1·16 = 16."},
		{fn: "square", mu: 3, sig: 2, reason: "The derivative is 6, so σ²·6² = 4·36 = 144."},
		{fn: "square", mu: 1, sig: 3, reason: "The derivative is 2, so 9·4 = 36."},
		{fn: "square", mu: 4, sig: 1, reason: "The derivative is 8, so 1·64 = 64."},
		{fn: "sqrt", mu: 4, sig: 2, reason: "The derivative of the square root at 4 is 1/(2·2) = 0.25, so 4·0.0625 = 0.25."},
		{fn: "sqrt", mu: 1, sig: 2, reason: "The derivative at 1 is 1/2, so 4·0.25 = 1."},
	}
	c := claims[rand.Intn(len(claims))]
	var answer string
	var label string
	if c.fn == "square" {
		label = "x squared"
		answer = decimal6(float64(4 * c.mu * c.mu * c.sig * c.sig))
	} else {
		label = "the square root of x"
		answer = decimal6(float64(c.sig*c.sig) / float64(4*c.mu))
	}
	return generator.Problem{
		Question: "A sample mean of n observations has mean μ = " + strconv.Itoa(c.mu) + " and variance σ² = " +
			strconv.Itoa(c.sig*c.sig) + ". The delta method gives the asymptotic variance of g(sample mean) as σ² times the square of the derivative of g at μ. For g(x) equal to " +
			label + ", what is that asymptotic variance? (enter a decimal)",
		Answer:      answer,
		Explanation: "Differentiate the transformation and square it, then multiply by the asymptotic variance of the sample mean; " + c.reason,
		Facts: map[string]string{
			"fn": c.fn, "mu": strconv.Itoa(c.mu), "sigma": strconv.Itoa(c.sig), "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// Characteristic functions (multiple choice)
// ---------------------------------------------------------------------------

type characteristicGen struct{}

func (g *characteristicGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		question    string
		correct     string
		distractors []string
		reason      string
	}
	claims := []claim{
		{
			question:    "The characteristic function of X is...",
			correct:     "the expectation of e^{itX} as a function of the real parameter t",
			distractors: []string{"the expectation of e^{tX}", "the expectation of X e^{tX}", "the density of X"},
			reason:      "The imaginary unit in the exponent keeps the modulus at one.",
		},
		{
			question:    "Unlike the moment generating function, the characteristic function...",
			correct:     "always exists",
			distractors: []string{"exists only for small t", "requires finite moments", "is undefined for discrete variables"},
			reason:      "The modulus of e^{itX} is one, so the expectation is finite for every distribution.",
		},
		{
			question:    "The characteristic function of a symmetric distribution is...",
			correct:     "real-valued",
			distractors: []string{"purely imaginary", "complex with a nonzero imaginary part", "constant"},
			reason:      "A symmetric law makes the sine terms cancel, leaving the cosine terms.",
		},
		{
			question:    "A characteristic function is...",
			correct:     "uniformly continuous and bounded by one in modulus",
			distractors: []string{"unbounded", "always equal to one", "necessarily real"},
			reason:      "The defining integral is bounded by one, and uniform continuity follows from dominated convergence.",
		},
		{
			question:    "The characteristic function of a sum of independent variables is...",
			correct:     "the product of their characteristic functions",
			distractors: []string{"the sum of their characteristic functions", "the average of them", "the maximum of them"},
			reason:      "Independence factors the expectation of the exponential of the sum.",
		},
	}
	c := claims[rand.Intn(len(claims))]
	p := generator.MultipleChoice(c.question, c.correct, c.distractors...)
	p.Explanation = c.reason
	return p
}

// ---------------------------------------------------------------------------
// Moments from the characteristic function (numeric)
// ---------------------------------------------------------------------------

type characteristicMomentsGen struct{}

func (g *characteristicMomentsGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		pn, pd int
		reason string
	}
	claims := []claim{
		{pn: 1, pd: 2, reason: "p = 0.5, so Var(X) = 0.5·0.5 = 0.25."},
		{pn: 1, pd: 4, reason: "p = 0.25, so Var(X) = 0.25·0.75 = 0.1875."},
		{pn: 1, pd: 5, reason: "p = 0.2, so Var(X) = 0.2·0.8 = 0.16."},
		{pn: 2, pd: 5, reason: "p = 0.4, so Var(X) = 0.4·0.6 = 0.24."},
		{pn: 1, pd: 10, reason: "p = 0.1, so Var(X) = 0.1·0.9 = 0.09."},
		{pn: 3, pd: 10, reason: "p = 0.3, so Var(X) = 0.3·0.7 = 0.21."},
	}
	c := claims[rand.Intn(len(claims))]
	p := float64(c.pn) / float64(c.pd)
	answer := decimal6(p * (1 - p))
	return generator.Problem{
		Question: "A random variable X takes the value zero with probability 1 − p and one with probability p, so its characteristic function is (1 − p) + p e^{it}. For p = " +
			strconv.Itoa(c.pn) + "/" + strconv.Itoa(c.pd) + ", what is Var(X)? (enter a decimal)",
		Answer:      answer,
		Explanation: "Differentiating the characteristic function twice at zero gives the second moment, and the variance is the second moment minus the square of the mean; " + c.reason,
		Facts: map[string]string{
			"pn": strconv.Itoa(c.pn), "pd": strconv.Itoa(c.pd), "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// Characteristic functions of sums (multiple choice)
// ---------------------------------------------------------------------------

type characteristicSumsGen struct{}

func (g *characteristicSumsGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		question    string
		correct     string
		distractors []string
		reason      string
	}
	claims := []claim{
		{
			question:    "If X and Y are independent, the characteristic function of X + Y is...",
			correct:     "the product of their characteristic functions",
			distractors: []string{"their sum", "their difference", "their average"},
			reason:      "Independence factors the expectation of e^{it(X+Y)}.",
		},
		{
			question:    "Without independence, the characteristic function of a sum is...",
			correct:     "given by the joint characteristic function along a diagonal, not a simple product",
			distractors: []string{"still the product", "the sum of the two functions", "undefined"},
			reason:      "The joint transform is needed when the variables are dependent.",
		},
		{
			question:    "The characteristic function of a pair factorises if and only if...",
			correct:     "the variables are independent",
			distractors: []string{"they are uncorrelated", "they have the same distribution", "their means are zero"},
			reason:      "Factorisation of the joint transform is equivalent to independence, unlike a zero covariance.",
		},
		{
			question:    "Characteristic functions turn a convolution of distributions into...",
			correct:     "a multiplication",
			distractors: []string{"an addition", "a division", "a composition"},
			reason:      "The Fourier transform diagonalises convolution into pointwise multiplication.",
		},
		{
			question:    "The product of the characteristic functions of independent variables is...",
			correct:     "the characteristic function of their sum",
			distractors: []string{"the characteristic function of their product", "their joint density", "their covariance"},
			reason:      "This is the product rule for independent sums.",
		},
	}
	c := claims[rand.Intn(len(claims))]
	p := generator.MultipleChoice(c.question, c.correct, c.distractors...)
	p.Explanation = c.reason
	return p
}

// ---------------------------------------------------------------------------
// The continuity theorem (multiple choice)
// ---------------------------------------------------------------------------

type continuityTheoremGen struct{}

func (g *continuityTheoremGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		question    string
		correct     string
		distractors []string
		reason      string
	}
	claims := []claim{
		{
			question:    "The continuity theorem says that a sequence of distributions converges if and only if...",
			correct:     "their characteristic functions converge pointwise to a function continuous at zero",
			distractors: []string{"their densities converge", "their moments converge", "their means converge"},
			reason:      "Pointwise convergence of the transforms, with continuity at the origin, is equivalent to weak convergence.",
		},
		{
			question:    "If the characteristic functions converge to a limit that is itself a characteristic function, then...",
			correct:     "the distributions converge weakly to the corresponding distribution",
			distractors: []string{"the variables converge in probability", "the densities converge uniformly", "the supports converge"},
			reason:      "The limit transform identifies the limiting law.",
		},
		{
			question:    "The continuity theorem is the main tool in...",
			correct:     "proving the central limit theorem",
			distractors: []string{"proving Chebyshev's inequality", "computing conditional expectations", "finding stationary distributions"},
			reason:      "The characteristic function of a standardised sum has a recognisable limit, the standard normal transform.",
		},
		{
			question:    "Pointwise convergence of characteristic functions gives convergence of the distributions when the limit is...",
			correct:     "continuous at zero",
			distractors: []string{"differentiable everywhere", "bounded", "monotone"},
			reason:      "Continuity at the origin is the technical condition that rules out mass escaping to infinity.",
		},
		{
			question:    "The continuity theorem is used to establish convergence of...",
			correct:     "the distributions, not necessarily the variables",
			distractors: []string{"the variables almost surely", "the densities everywhere", "the moments"},
			reason:      "It is a weak-convergence statement, about the laws rather than the sample paths.",
		},
	}
	c := claims[rand.Intn(len(claims))]
	p := generator.MultipleChoice(c.question, c.correct, c.distractors...)
	p.Explanation = c.reason
	return p
}

// ---------------------------------------------------------------------------
// The central limit theorem by characteristic functions (multiple choice)
// ---------------------------------------------------------------------------

type characteristicCLTGen struct{}

func (g *characteristicCLTGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		question    string
		correct     string
		distractors []string
		reason      string
	}
	claims := []claim{
		{
			question:    "In the characteristic-function proof of the central limit theorem, the log of the transform of the standardised sum is expanded...",
			correct:     "as −σ²t²/(2n) plus a term that vanishes as n grows",
			distractors: []string{"as σ²t²/2", "as t", "as a constant"},
			reason:      "The quadratic term survives the multiplication of the n transforms and the rest vanishes.",
		},
		{
			question:    "Multiplying the characteristic functions of the standardised sum gives an expression whose limit is...",
			correct:     "e^{−t²/2}",
			distractors: []string{"e^{t²/2}", "one", "a linear function of t"},
			reason:      "The quadratic exponent accumulates to the standard normal transform.",
		},
		{
			question:    "The limit e^{−t²/2} is the characteristic function of...",
			correct:     "the standard normal distribution",
			distractors: []string{"the uniform distribution", "the exponential distribution", "the Cauchy distribution"},
			reason:      "That transform identifies the standard normal law uniquely.",
		},
		{
			question:    "The proof establishes convergence of the standardised sum...",
			correct:     "in distribution",
			distractors: []string{"almost surely", "in probability", "in mean square"},
			reason:      "The continuity theorem yields weak convergence, which is what the central limit theorem claims.",
		},
		{
			question:    "A key step in the proof uses independence to...",
			correct:     "factor the characteristic function of the sum into a product",
			distractors: []string{"compute the density directly", "apply Chebyshev's inequality", "bound the support"},
			reason:      "The product rule for independent sums is what makes the n-th power appear.",
		},
	}
	c := claims[rand.Intn(len(claims))]
	p := generator.MultipleChoice(c.question, c.correct, c.distractors...)
	p.Explanation = c.reason
	return p
}

// ---------------------------------------------------------------------------
// Markov chains (numeric)
// ---------------------------------------------------------------------------

type markovChainGen struct{}

func (g *markovChainGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		p00, p01, p11 string
		reason        string
	}
	claims := []claim{
		{p00: "0.5", p01: "0.5", p11: "0.5", reason: "0.5·0.5 + 0.5·0.5 = 0.5."},
		{p00: "0.6", p01: "0.4", p11: "0.7", reason: "0.6·0.4 + 0.4·0.7 = 0.52."},
		{p00: "0.8", p01: "0.2", p11: "0.9", reason: "0.8·0.2 + 0.2·0.9 = 0.34."},
		{p00: "0.7", p01: "0.3", p11: "0.6", reason: "0.7·0.3 + 0.3·0.6 = 0.39."},
		{p00: "0.9", p01: "0.1", p11: "0.8", reason: "0.9·0.1 + 0.1·0.8 = 0.17."},
		{p00: "0.4", p01: "0.6", p11: "0.5", reason: "0.4·0.6 + 0.6·0.5 = 0.54."},
	}
	c := claims[rand.Intn(len(claims))]
	f := func(s string) float64 { v, _ := strconv.ParseFloat(s, 64); return v }
	answer := decimal6(f(c.p00)*f(c.p01) + f(c.p01)*f(c.p11))
	return generator.Problem{
		Question: "A two-state Markov chain has P(state 0 to state 0) = " + c.p00 + ", P(state 0 to state 1) = " +
			c.p01 + " and P(state 1 to state 1) = " + c.p11 +
			". What is the two-step probability P(X_2 = 1 | X_0 = 0)? (enter a decimal)",
		Answer:      answer,
		Explanation: "A two-step transition sums over the intermediate state, here 0 to 0 to 1 plus 0 to 1 to 1; " + c.reason,
		Facts: map[string]string{
			"p00": c.p00, "p01": c.p01, "p11": c.p11, "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// Transition matrices (numeric)
// ---------------------------------------------------------------------------

type transitionMatrixGen struct{}

func (g *transitionMatrixGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		pi0, pi1, p01, p11 string
		reason             string
	}
	claims := []claim{
		{pi0: "0.5", pi1: "0.5", p01: "0.4", p11: "0.8", reason: "0.5·0.4 + 0.5·0.8 = 0.6."},
		{pi0: "0.6", pi1: "0.4", p01: "0.5", p11: "0.5", reason: "0.6·0.5 + 0.4·0.5 = 0.5."},
		{pi0: "0.3", pi1: "0.7", p01: "0.2", p11: "0.6", reason: "0.3·0.2 + 0.7·0.6 = 0.48."},
		{pi0: "0.8", pi1: "0.2", p01: "0.1", p11: "0.9", reason: "0.8·0.1 + 0.2·0.9 = 0.26."},
		{pi0: "0.4", pi1: "0.6", p01: "0.7", p11: "0.3", reason: "0.4·0.7 + 0.6·0.3 = 0.46."},
		{pi0: "0.2", pi1: "0.8", p01: "0.3", p11: "0.4", reason: "0.2·0.3 + 0.8·0.4 = 0.38."},
	}
	c := claims[rand.Intn(len(claims))]
	f := func(s string) float64 { v, _ := strconv.ParseFloat(s, 64); return v }
	answer := decimal6(f(c.pi0)*f(c.p01) + f(c.pi1)*f(c.p11))
	return generator.Problem{
		Question: "A chain starts with P(X_0 = 0) = " + c.pi0 + " and P(X_0 = 1) = " + c.pi1 +
			". Its one-step probabilities are P(state 0 to state 1) = " + c.p01 + " and P(state 1 to state 1) = " +
			c.p11 + ". What is P(X_1 = 1)? (enter a decimal)",
		Answer:      answer,
		Explanation: "The distribution after one step is the initial distribution multiplied by the transition matrix; " + c.reason,
		Facts: map[string]string{
			"pi0": c.pi0, "pi1": c.pi1, "p01": c.p01, "p11": c.p11, "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// Stationary distributions (numeric)
// ---------------------------------------------------------------------------

type stationaryGen struct{}

func (g *stationaryGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		p01, p10 string
		reason   string
	}
	claims := []claim{
		{p01: "0.3", p10: "0.7", reason: "0.3/(0.3 + 0.7) = 0.3."},
		{p01: "0.2", p10: "0.6", reason: "0.2/0.8 = 0.25."},
		{p01: "0.5", p10: "0.5", reason: "0.5/1 = 0.5."},
		{p01: "0.4", p10: "0.6", reason: "0.4/1 = 0.4."},
		{p01: "0.1", p10: "0.9", reason: "0.1/1 = 0.1."},
		{p01: "0.6", p10: "0.2", reason: "0.6/0.8 = 0.75."},
	}
	c := claims[rand.Intn(len(claims))]
	f := func(s string) float64 { v, _ := strconv.ParseFloat(s, 64); return v }
	up, down := f(c.p01), f(c.p10)
	answer := decimal6(up / (up + down))
	return generator.Problem{
		Question: "A two-state Markov chain has P(state 0 to state 1) = " + c.p01 +
			" and P(state 1 to state 0) = " + c.p10 +
			". What is the stationary probability of state 1? (enter a decimal)",
		Answer:      answer,
		Explanation: "Stationarity balances the flow between the states, so π1 = p01/(p01 + p10); " + c.reason,
		Facts: map[string]string{
			"p01": c.p01, "p10": c.p10, "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// Random walks (numeric)
// ---------------------------------------------------------------------------

type randomWalkGen struct{}

func (g *randomWalkGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		n, pn, pd int
		reason    string
	}
	claims := []claim{
		{n: 10, pn: 1, pd: 2, reason: "An unbiased step has variance one, so the sum has variance n = 10."},
		{n: 25, pn: 1, pd: 2, reason: "The variance is n = 25."},
		{n: 10, pn: 1, pd: 4, reason: "4·10·0.25·0.75 = 7.5."},
		{n: 20, pn: 1, pd: 2, reason: "The variance is n = 20."},
		{n: 10, pn: 2, pd: 5, reason: "4·10·0.4·0.6 = 9.6."},
		{n: 50, pn: 1, pd: 2, reason: "The variance is n = 50."},
	}
	c := claims[rand.Intn(len(claims))]
	p := float64(c.pn) / float64(c.pd)
	answer := decimal6(4 * float64(c.n) * p * (1 - p))
	return generator.Problem{
		Question: "A random walk takes a step of plus one with probability p = " + strconv.Itoa(c.pn) +
			"/" + strconv.Itoa(c.pd) + " and minus one otherwise. What is Var(S_n) after n = " +
			strconv.Itoa(c.n) + " steps? (enter a decimal)",
		Answer:      answer,
		Explanation: "Each step has variance 4p(1 − p), and independent steps add, so the sum has variance 4np(1 − p); " + c.reason,
		Facts: map[string]string{
			"n": strconv.Itoa(c.n), "pn": strconv.Itoa(c.pn), "pd": strconv.Itoa(c.pd), "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// Martingales (multiple choice)
// ---------------------------------------------------------------------------

type martingaleGen struct{}

func (g *martingaleGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		question    string
		correct     string
		distractors []string
		reason      string
	}
	claims := []claim{
		{
			question:    "A sequence is a martingale when...",
			correct:     "E[M_{n+1} | M_0, ..., M_n] = M_n",
			distractors: []string{"E[M_n] = 0 for every n", "the increments are independent", "M_n is constant in n"},
			reason:      "The conditional expectation of the next value, given the past, is the present value.",
		},
		{
			question:    "A martingale's expectation as a function of time is...",
			correct:     "constant",
			distractors: []string{"increasing without bound", "always zero", "decaying to zero"},
			reason:      "Taking expectations of the defining equation gives E[M_{n+1}] = E[M_n].",
		},
		{
			question:    "Which is the canonical example of a martingale?",
			correct:     "a symmetric random walk",
			distractors: []string{"a monotone increasing sequence", "the running maximum of a random walk", "a deterministic trend"},
			reason:      "A fair random walk has conditional mean equal to its current value.",
		},
		{
			question:    "The optional stopping theorem says that for a suitable stopping time T...",
			correct:     "E[M_T] = E[M_0] under conditions that rule out the abuse of stopping",
			distractors: []string{"E[M_T] = 0 always", "the stopped process diverges", "the stopping time has no expectation"},
			reason:      "Boundedness or integrability conditions are needed, which is why the theorem is easy to misuse.",
		},
		{
			question:    "For a symmetric random walk S_n, the quantity S_n² − n is...",
			correct:     "a martingale",
			distractors: []string{"increasing in n", "constant in n", "a stopping time"},
			reason:      "The compensating drift n removes the growth in the second moment, leaving a martingale.",
		},
	}
	c := claims[rand.Intn(len(claims))]
	p := generator.MultipleChoice(c.question, c.correct, c.distractors...)
	p.Explanation = c.reason
	return p
}
