package probability

// The first probability-theory batch: twenty-five concepts that open the domain
// with the axioms, conditional probability, random variables, a few distribution
// results and the classical limit inequalities.
//
// The statistics domain already has sample spaces, basic and compound
// probability, conditional probability, Bayes' theorem, expected value, discrete
// and continuous random variables and the named distributions. This domain
// starts after those and goes deeper: it treats a probability as a measure on a
// sigma-algebra, defines a random variable as a measurable function, works with
// mass and density functions, moments and generating functions, and proves the
// first limit results.
//
// Nine concepts are multiple choice and sixteen are numeric. Every numeric
// generator publishes the parameters its answer follows from, so the tests
// recompute the answer rather than trusting the author's arithmetic.

import (
	"math"
	"math/rand"
	"strconv"
	"strings"

	"github.com/chuma-beep/mathua/internal/generator"
)

// decimal6 formats a probability or other decimal to at most six places, with
// trailing zeros removed, so 0.6400000000000001 becomes "0.64" and 7.0 becomes
// "7". The tests use the same formatter, which keeps the published answer and the
// recomputed one comparable without a tolerance.
func decimal6(v float64) string {
	s := strconv.FormatFloat(v, 'f', 6, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" || s == "-0" {
		s = "0"
	}
	return s
}

// ---------------------------------------------------------------------------
// Probability axioms (multiple choice)
// ---------------------------------------------------------------------------

type axiomsGen struct{}

func (g *axiomsGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		question    string
		correct     string
		distractors []string
		reason      string
	}
	claims := []claim{
		{
			question:    "Kolmogorov's axioms require that a probability P(A) is...",
			correct:     "a nonnegative number with P(Ω) = 1 and countable additivity",
			distractors: []string{"a positive integer", "always less than 1 and never 0", "defined only for equally likely outcomes"},
			reason:      "A probability is a measure: it is nonnegative, the whole space has measure one, and disjoint countable unions add.",
		},
		{
			question:    "For disjoint events A and B, countable additivity gives...",
			correct:     "P(A ∪ B) = P(A) + P(B)",
			distractors: []string{"P(A ∪ B) = P(A)P(B)", "P(A ∪ B) = P(A) + P(B) − P(A ∩ B)", "P(A ∪ B) = 1"},
			reason:      "The events do not overlap, so the intersection term vanishes and the measures add.",
		},
		{
			question:    "The complement rule states that the probability of \"not A\" is...",
			correct:     "1 − P(A)",
			distractors: []string{"P(A)", "1 divided by P(A)", "0"},
			reason:      "A and its complement partition the sample space, so their probabilities sum to one.",
		},
		{
			question:    "The empty event has probability...",
			correct:     "0",
			distractors: []string{"1", "undefined", "1/2"},
			reason:      "Ω ∪ ∅ is Ω as a disjoint union, so P(∅) must be zero.",
		},
		{
			question:    "A probability is a function whose domain is...",
			correct:     "the events of a sample space",
			distractors: []string{"the outcomes only", "the real numbers", "the random variables"},
			reason:      "It is defined on the sigma-algebra of events, not on individual outcomes or on numbers.",
		},
	}
	c := claims[rand.Intn(len(claims))]
	p := generator.MultipleChoice(c.question, c.correct, c.distractors...)
	p.Explanation = c.reason
	return p
}

// ---------------------------------------------------------------------------
// Sigma-algebras and events (multiple choice)
// ---------------------------------------------------------------------------

type sigmaAlgebraGen struct{}

func (g *sigmaAlgebraGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		question    string
		correct     string
		distractors []string
		reason      string
	}
	claims := []claim{
		{
			question:    "A sigma-algebra of events is closed under...",
			correct:     "complementation and countable unions",
			distractors: []string{"addition and multiplication", "intersection only", "taking limits of numbers"},
			reason:      "Those two operations generate countable intersections and differences as well.",
		},
		{
			question:    "The smallest sigma-algebra on a sample space Ω is...",
			correct:     "{∅, Ω}",
			distractors: []string{"the power set of Ω", "{Ω}", "the empty family"},
			reason:      "It must contain Ω, hence its complement ∅, and those two already form a sigma-algebra.",
		},
		{
			question:    "The power set of a countable sample space is a sigma-algebra because...",
			correct:     "it contains every subset, so it is closed under every set operation",
			distractors: []string{"it is finite", "it has one element", "it is closed under addition"},
			reason:      "Closure is automatic when every possible subset is already a member.",
		},
		{
			question:    "A measurable event is...",
			correct:     "a member of the sigma-algebra",
			distractors: []string{"any subset of Ω", "a single outcome", "a random variable"},
			reason:      "Membership in the sigma-algebra is exactly what makes a probability assignable to a set.",
		},
		{
			question:    "If A and B are events, then A ∩ B is an event because...",
			correct:     "a sigma-algebra is closed under countable intersections too",
			distractors: []string{"intersections are the same as unions", "B is the complement of A", "events are numbers"},
			reason:      "De Morgan's law writes an intersection as the complement of a union, so closure transfers.",
		},
	}
	c := claims[rand.Intn(len(claims))]
	p := generator.MultipleChoice(c.question, c.correct, c.distractors...)
	p.Explanation = c.reason
	return p
}

// ---------------------------------------------------------------------------
// Inclusion-exclusion (numeric)
// ---------------------------------------------------------------------------

type inclusionExclusionGen struct{}

func (g *inclusionExclusionGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		n, a, b, ab int
		reason      string
	}
	claims := []claim{
		{n: 100, a: 40, b: 30, ab: 10, reason: "(40 + 30 − 10)/100 = 0.6."},
		{n: 50, a: 20, b: 15, ab: 5, reason: "(20 + 15 − 5)/50 = 0.6."},
		{n: 20, a: 10, b: 8, ab: 3, reason: "(10 + 8 − 3)/20 = 0.75."},
		{n: 100, a: 50, b: 40, ab: 20, reason: "(50 + 40 − 20)/100 = 0.7."},
		{n: 200, a: 80, b: 60, ab: 20, reason: "(80 + 60 − 20)/200 = 0.6."},
		{n: 25, a: 15, b: 12, ab: 5, reason: "(15 + 12 − 5)/25 = 0.88."},
	}
	c := claims[rand.Intn(len(claims))]
	answer := decimal6(float64(c.a+c.b-c.ab) / float64(c.n))
	return generator.Problem{
		Question: "A sample space has " + strconv.Itoa(c.n) + " equally likely outcomes. Event A has " +
			strconv.Itoa(c.a) + " outcomes, event B has " + strconv.Itoa(c.b) + ", and A ∩ B has " +
			strconv.Itoa(c.ab) + ". What is P(A ∪ B)? (enter a decimal)",
		Answer:      answer,
		Explanation: "By inclusion-exclusion P(A ∪ B) = P(A) + P(B) − P(A ∩ B), and " + c.reason,
		Facts: map[string]string{
			"n": strconv.Itoa(c.n), "a": strconv.Itoa(c.a), "b": strconv.Itoa(c.b),
			"ab": strconv.Itoa(c.ab), "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// Continuity of probability (multiple choice)
// ---------------------------------------------------------------------------

type continuityGen struct{}

func (g *continuityGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		question    string
		correct     string
		distractors []string
		reason      string
	}
	claims := []claim{
		{
			question:    "Continuity from below states that if events A_n increase to A, then...",
			correct:     "P(A_n) tends to P(A)",
			distractors: []string{"P(A_n) tends to 1", "P(A_n) tends to 0", "P(A_n) is eventually constant"},
			reason:      "The probabilities of an increasing sequence converge to the probability of the union.",
		},
		{
			question:    "Continuity from above requires a decreasing sequence of events together with...",
			correct:     "the first event having finite probability",
			distractors: []string{"the events being disjoint", "the events being independent", "the events being equally likely"},
			reason:      "Without a finite starting measure the subtraction of limits can fail.",
		},
		{
			question:    "Continuity of probability follows from...",
			correct:     "countable additivity",
			distractors: []string{"finite additivity alone", "independence of the events", "the multiplication rule"},
			reason:      "Writing an increasing union as disjoint differences and summing is exactly countable additivity.",
		},
		{
			question:    "For increasing events A_n with union A, the value P(A) equals...",
			correct:     "the limit of P(A_n)",
			distractors: []string{"the sum of the P(A_n)", "the maximum of the P(A_n)", "P(A_1)"},
			reason:      "That is continuity from below, the monotone convergence of the probability measure.",
		},
		{
			question:    "Continuity from below computes the probability of an infinite union as...",
			correct:     "the limit of the probabilities of the finite partial unions",
			distractors: []string{"a product of probabilities", "an average of probabilities", "a derivative of probabilities"},
			reason:      "Partial unions increase to the full union, so their probabilities have the union's probability as limit.",
		},
	}
	c := claims[rand.Intn(len(claims))]
	p := generator.MultipleChoice(c.question, c.correct, c.distractors...)
	p.Explanation = c.reason
	return p
}

// ---------------------------------------------------------------------------
// The multiplication rule (numeric)
// ---------------------------------------------------------------------------

type multiplicationGen struct{}

func (g *multiplicationGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		n, a, b int
		reason  string
	}
	claims := []claim{
		{n: 20, a: 10, b: 6, reason: "P(A ∩ B) = (10/20)(6/10) = 6/20 = 0.3."},
		{n: 50, a: 20, b: 15, reason: "(20/50)(15/20) = 15/50 = 0.3."},
		{n: 12, a: 4, b: 3, reason: "(4/12)(3/4) = 3/12 = 0.25."},
		{n: 25, a: 10, b: 6, reason: "(10/25)(6/10) = 6/25 = 0.24."},
		{n: 40, a: 16, b: 12, reason: "(16/40)(12/16) = 12/40 = 0.3."},
		{n: 8, a: 2, b: 1, reason: "(2/8)(1/2) = 1/8 = 0.125."},
	}
	c := claims[rand.Intn(len(claims))]
	answer := decimal6(float64(c.b) / float64(c.n))
	return generator.Problem{
		Question: "An experiment has " + strconv.Itoa(c.n) + " equally likely outcomes. Event A has " +
			strconv.Itoa(c.a) + " of them, and among those A-outcomes " + strconv.Itoa(c.b) +
			" also lie in event B. What is P(A ∩ B)? (enter a decimal)",
		Answer:      answer,
		Explanation: "By the multiplication rule P(A ∩ B) = P(A)·P(B | A), and " + c.reason,
		Facts: map[string]string{
			"n": strconv.Itoa(c.n), "a": strconv.Itoa(c.a), "b": strconv.Itoa(c.b), "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// Bayes' theorem with priors (numeric)
// ---------------------------------------------------------------------------

type bayesGen struct{}

func (g *bayesGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		tA, aA, tB, aB int
		reason         string
	}
	claims := []claim{
		{tA: 4, aA: 3, tB: 4, aB: 1, reason: "(3/4)/((3/4) + (1/4)) = 3/4 = 0.75."},
		{tA: 5, aA: 4, tB: 5, aB: 1, reason: "(4/5)/((4/5) + (1/5)) = 4/5 = 0.8."},
		{tA: 10, aA: 7, tB: 10, aB: 3, reason: "(7/10)/((7/10) + (3/10)) = 0.7."},
		{tA: 10, aA: 9, tB: 10, aB: 3, reason: "(9/10)/((9/10) + (3/10)) = 9/12 = 0.75."},
		{tA: 5, aA: 2, tB: 5, aB: 2, reason: "(2/5)/((2/5) + (2/5)) = 1/2 = 0.5."},
		{tA: 10, aA: 3, tB: 10, aB: 1, reason: "(3/10)/((3/10) + (1/10)) = 3/4 = 0.75."},
	}
	c := claims[rand.Intn(len(claims))]
	pa := float64(c.aA) / float64(c.tA)
	pb := float64(c.aB) / float64(c.tB)
	answer := decimal6(pa / (pa + pb))
	return generator.Problem{
		Question: "Two boxes are equally likely to be chosen. Box one holds " + strconv.Itoa(c.aA) +
			" red marbles out of " + strconv.Itoa(c.tA) + "; box two holds " + strconv.Itoa(c.aB) +
			" red out of " + strconv.Itoa(c.tB) + ". A red marble is drawn. What is P(box one | red)? (enter a decimal)",
		Answer: answer,
		Explanation: "The equal priors cancel in Bayes' theorem, so the posterior is " +
			"(aA/tA)/((aA/tA) + (aB/tB)); " + c.reason,
		Facts: map[string]string{
			"tA": strconv.Itoa(c.tA), "aA": strconv.Itoa(c.aA),
			"tB": strconv.Itoa(c.tB), "aB": strconv.Itoa(c.aB), "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// Pairwise and mutual independence (multiple choice)
// ---------------------------------------------------------------------------

type independenceGen struct{}

func (g *independenceGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		question    string
		correct     string
		distractors []string
		reason      string
	}
	claims := []claim{
		{
			question:    "Two events A and B are independent exactly when...",
			correct:     "P(A ∩ B) = P(A)P(B)",
			distractors: []string{"P(A ∪ B) = P(A) + P(B)", "A and B are disjoint", "P(A) = P(B)"},
			reason:      "The product rule is the definition; disjointness and equal probabilities are unrelated conditions.",
		},
		{
			question:    "Three events A, B and C are mutually independent when...",
			correct:     "every subcollection satisfies the product rule",
			distractors: []string{"each pair is independent", "they are pairwise disjoint", "P(A ∩ B ∩ C) = P(A)P(B)P(C) alone"},
			reason:      "Mutual independence needs all four equations: the three pair products and the triple product.",
		},
		{
			question:    "Pairwise independence of three events...",
			correct:     "does not imply mutual independence",
			distractors: []string{"implies mutual independence", "requires the events to be disjoint", "requires equal probabilities"},
			reason:      "The classic two-dice example is pairwise but not mutually independent, because the triple product fails.",
		},
		{
			question:    "If P(A) is positive and A and B are independent, then P(B | A) equals...",
			correct:     "P(B)",
			distractors: []string{"P(A)", "P(A)P(B)", "0"},
			reason:      "Dividing the product rule by P(A) shows conditioning on A does not change B's probability.",
		},
		{
			question:    "Two disjoint events with positive probability are...",
			correct:     "dependent",
			distractors: []string{"independent", "complementary", "equally likely"},
			reason:      "Knowing one occurred rules out the other, so the conditional probability collapses to zero.",
		},
	}
	c := claims[rand.Intn(len(claims))]
	p := generator.MultipleChoice(c.question, c.correct, c.distractors...)
	p.Explanation = c.reason
	return p
}

// ---------------------------------------------------------------------------
// Random variables (multiple choice)
// ---------------------------------------------------------------------------

type rvDefinitionGen struct{}

func (g *rvDefinitionGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		question    string
		correct     string
		distractors []string
		reason      string
	}
	claims := []claim{
		{
			question:    "A random variable is...",
			correct:     "a measurable function from the sample space to the real numbers",
			distractors: []string{"a number chosen at random", "an outcome of an experiment", "a probability between 0 and 1"},
			reason:      "It is a function: it maps each outcome to a real value, and measurability makes its sets assignable probabilities.",
		},
		{
			question:    "The distribution of a random variable is...",
			correct:     "the probability it assigns to its values",
			distractors: []string{"its set of possible values", "its mean", "the sample space it is defined on"},
			reason:      "The distribution is the push-forward of the underlying probability, not the value set or a summary number.",
		},
		{
			question:    "A discrete random variable takes...",
			correct:     "countably many values",
			distractors: []string{"finitely many values only", "any real values", "only values in [0, 1]"},
			reason:      "Countable includes the infinite case, such as a Poisson variable on the nonnegative integers.",
		},
		{
			question:    "A continuous random variable is typically described by...",
			correct:     "a probability density",
			distractors: []string{"a probability mass function", "a finite list of outcomes", "a single number"},
			reason:      "Individual points have probability zero, so probabilities come from integrating a density.",
		},
		{
			question:    "Two random variables are identically distributed when...",
			correct:     "they have the same distribution, though they may be defined on different spaces",
			distractors: []string{"they are equal at every outcome", "they have the same mean only", "they are independent"},
			reason:      "Distribution is a property of the law, not of a shared domain, equality, mean or independence.",
		},
	}
	c := claims[rand.Intn(len(claims))]
	p := generator.MultipleChoice(c.question, c.correct, c.distractors...)
	p.Explanation = c.reason
	return p
}

// ---------------------------------------------------------------------------
// The distribution function (multiple choice)
// ---------------------------------------------------------------------------

type cdfGen struct{}

func (g *cdfGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		question    string
		correct     string
		distractors []string
		reason      string
	}
	claims := []claim{
		{
			question:    "The distribution function F(x) of a random variable X is defined as...",
			correct:     "P(X ≤ x)",
			distractors: []string{"P(X = x)", "P(X ≥ x)", "the density of X at x"},
			reason:      "The CDF accumulates probability up to and including x, from the left.",
		},
		{
			question:    "A distribution function is...",
			correct:     "non-decreasing and right-continuous, with limits 0 and 1 at the ends",
			distractors: []string{"strictly increasing and continuous", "always differentiable", "bounded away from zero"},
			reason:      "It may jump and may be flat; only monotonicity, right-continuity and the two limits are forced.",
		},
		{
			question:    "In terms of F, the probability P(a < X ≤ b) equals...",
			correct:     "F(b) − F(a)",
			distractors: []string{"F(a) − F(b)", "F(b) + F(a)", "F(b)F(a)"},
			reason:      "Subtracting the accumulated probability below a leaves the probability of the interval ending at b.",
		},
		{
			question:    "The CDF of a continuous random variable is...",
			correct:     "continuous",
			distractors: []string{"a step function", "always constant", "undefined at its jumps"},
			reason:      "No point has positive probability, so the CDF has no jumps.",
		},
		{
			question:    "The CDF of a discrete random variable...",
			correct:     "is a step function that jumps by the mass function at each value",
			distractors: []string{"is continuous", "is constant between all values", "has no jumps"},
			reason:      "The probability of a single value becomes a jump of exactly that size.",
		},
	}
	c := claims[rand.Intn(len(claims))]
	p := generator.MultipleChoice(c.question, c.correct, c.distractors...)
	p.Explanation = c.reason
	return p
}

// ---------------------------------------------------------------------------
// The probability mass function (numeric)
// ---------------------------------------------------------------------------

type pmfGen struct{}

func (g *pmfGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		weights []int
		index   int // 1-based
		reason  string
	}
	claims := []claim{
		{weights: []int{2, 4, 6, 8}, index: 4, reason: "The weights sum to 20, so the last one has probability 8/20 = 0.4."},
		{weights: []int{1, 1, 2}, index: 3, reason: "They sum to 4, so the third is 2/4 = 0.5."},
		{weights: []int{3, 5, 2}, index: 1, reason: "They sum to 10, so the first is 3/10 = 0.3."},
		{weights: []int{2, 3, 5}, index: 2, reason: "They sum to 10, so the second is 3/10 = 0.3."},
		{weights: []int{4, 6, 10}, index: 1, reason: "They sum to 20, so the first is 4/20 = 0.2."},
		{weights: []int{5, 3, 2, 10}, index: 4, reason: "They sum to 20, so the fourth is 10/20 = 0.5."},
	}
	c := claims[rand.Intn(len(claims))]
	total := 0
	for _, w := range c.weights {
		total += w
	}
	weight := c.weights[c.index-1]
	answer := decimal6(float64(weight) / float64(total))
	parts := make([]string, len(c.weights))
	for i, w := range c.weights {
		parts[i] = strconv.Itoa(w)
	}
	return generator.Problem{
		Question: "A discrete random variable X has probabilities proportional to the weights " +
			strings.Join(parts, ", ") + " on its outcomes. What is the probability of the outcome with weight " +
			strconv.Itoa(weight) + "? (enter a decimal)",
		Answer:      answer,
		Explanation: "A mass function must sum to one, so each weight is divided by the total: " + c.reason,
		Facts: map[string]string{
			"weights": strings.Join(parts, ","), "index": strconv.Itoa(c.index),
			"total": strconv.Itoa(total), "weight": strconv.Itoa(weight), "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// The probability density function (numeric)
// ---------------------------------------------------------------------------

type pdfGen struct{}

func (g *pdfGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		a      string
		reason string
	}
	claims := []claim{
		{a: "0.5", reason: "Integrating 2x from 0 to 0.5 gives 0.5² = 0.25."},
		{a: "0.3", reason: "0.3² = 0.09."},
		{a: "0.8", reason: "0.8² = 0.64."},
		{a: "0.9", reason: "0.9² = 0.81."},
		{a: "0.2", reason: "0.2² = 0.04."},
		{a: "0.7", reason: "0.7² = 0.49."},
	}
	c := claims[rand.Intn(len(claims))]
	a, _ := strconv.ParseFloat(c.a, 64)
	answer := decimal6(a * a)
	return generator.Problem{
		Question: "A continuous random variable X has density f(x) = 2x on the interval [0, 1]. " +
			"What is P(X ≤ " + c.a + ")? (enter a decimal)",
		Answer:      answer,
		Explanation: "P(X ≤ a) is the integral of 2x from 0 to a, which is a²; " + c.reason,
		Facts:       map[string]string{"a": c.a, "answer": answer},
	}
}

// ---------------------------------------------------------------------------
// Expectation of a function (numeric)
// ---------------------------------------------------------------------------

type expectationGen struct{}

func (g *expectationGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		values  []int
		weights []int
		den     int
		reason  string
	}
	claims := []claim{
		{values: []int{0, 1, 2}, weights: []int{2, 3, 5}, den: 10, reason: "0·0.2 + 1·0.3 + 4·0.5 = 2.3."},
		{values: []int{0, 2}, weights: []int{5, 5}, den: 10, reason: "0·0.5 + 4·0.5 = 2."},
		{values: []int{1, 2, 3}, weights: []int{25, 25, 50}, den: 100, reason: "1·0.25 + 4·0.25 + 9·0.5 = 5.75."},
		{values: []int{0, 1, 2, 3}, weights: []int{1, 2, 3, 4}, den: 10, reason: "0 + 0.2 + 4·0.3 + 9·0.4 = 5."},
		{values: []int{1, 4}, weights: []int{6, 4}, den: 10, reason: "1·0.6 + 16·0.4 = 7."},
		{values: []int{0, 1}, weights: []int{2, 8}, den: 10, reason: "0·0.2 + 1·0.8 = 0.8."},
	}
	c := claims[rand.Intn(len(claims))]
	sum := 0
	for i, w := range c.weights {
		sum += c.values[i] * c.values[i] * w
	}
	answer := decimal6(float64(sum) / float64(c.den))
	vals := make([]string, len(c.values))
	wts := make([]string, len(c.weights))
	// The walkthrough's working (fork change): each probability, and each
	// term x²·p(x), computed here rather than read from the reason.
	probs := make([]string, len(c.values))
	terms := make([]string, len(c.values))
	for i := range c.values {
		vals[i] = strconv.Itoa(c.values[i])
		wts[i] = strconv.Itoa(c.weights[i])
		probs[i] = decimal6(float64(c.weights[i]) / float64(c.den))
		terms[i] = strconv.Itoa(c.values[i]*c.values[i]) + "·" + probs[i]
	}
	return generator.Problem{
		Question: "A discrete random variable X takes the values " + strings.Join(vals, ", ") +
			" with probabilities proportional to the weights " + strings.Join(wts, ", ") +
			" (the weights sum to " + strconv.Itoa(c.den) + "). What is E[X²]? (enter a decimal)",
		Answer: answer,
		Explanation: "The law of the unconscious statistician gives E[g(X)] = Σ g(x)p(x), so E[X²] = Σ x²p(x); " +
			c.reason,
		Facts: map[string]string{
			"values": strings.Join(vals, ","), "weights": strings.Join(wts, ","),
			"den": strconv.Itoa(c.den), "answer": answer,
			"probs": strings.Join(probs, ", "), "terms": strings.Join(terms, " + "),
		},
	}
}

// ---------------------------------------------------------------------------
// Variance by the second moment (numeric)
// ---------------------------------------------------------------------------

type varianceGen struct{}

func (g *varianceGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		values  []int
		weights []int
		den     int
		reason  string
	}
	claims := []claim{
		{values: []int{0, 2}, weights: []int{5, 5}, den: 10, reason: "E[X] = 1 and E[X²] = 2, so Var(X) = 2 − 1 = 1."},
		{values: []int{0, 1, 2}, weights: []int{25, 50, 25}, den: 100, reason: "E[X] = 1 and E[X²] = 1.5, so Var(X) = 0.5."},
		{values: []int{0, 1, 2}, weights: []int{2, 3, 5}, den: 10, reason: "E[X] = 1.3 and E[X²] = 2.3, so Var(X) = 2.3 − 1.69 = 0.61."},
		{values: []int{1, 4}, weights: []int{6, 4}, den: 10, reason: "E[X] = 2.2 and E[X²] = 7, so Var(X) = 7 − 4.84 = 2.16."},
		{values: []int{0, 2, 4}, weights: []int{50, 25, 25}, den: 100, reason: "E[X] = 1.5 and E[X²] = 5, so Var(X) = 5 − 2.25 = 2.75."},
		{values: []int{0, 3}, weights: []int{5, 5}, den: 10, reason: "E[X] = 1.5 and E[X²] = 4.5, so Var(X) = 4.5 − 2.25 = 2.25."},
	}
	c := claims[rand.Intn(len(claims))]
	sw, sw2 := 0, 0
	for i, w := range c.weights {
		sw += c.values[i] * w
		sw2 += c.values[i] * c.values[i] * w
	}
	// Var = (den·Σx²w − (Σxw)²) / den², kept in integers until the last step.
	num := c.den*sw2 - sw*sw
	answer := decimal6(float64(num) / float64(c.den*c.den))
	vals := make([]string, len(c.values))
	wts := make([]string, len(c.weights))
	meanTerms := make([]string, len(c.values))
	sqTerms := make([]string, len(c.values))
	for i := range c.values {
		vals[i] = strconv.Itoa(c.values[i])
		wts[i] = strconv.Itoa(c.weights[i])
		prob := decimal6(float64(c.weights[i]) / float64(c.den))
		meanTerms[i] = vals[i] + "·" + prob
		sqTerms[i] = strconv.Itoa(c.values[i]*c.values[i]) + "·" + prob
	}
	mean := float64(sw) / float64(c.den)
	return generator.Problem{
		Question: "A discrete random variable X takes the values " + strings.Join(vals, ", ") +
			" with probabilities proportional to the weights " + strings.Join(wts, ", ") +
			" (the weights sum to " + strconv.Itoa(c.den) + "). What is Var(X) = E[X²] − (E[X])²? (enter a decimal)",
		Answer:      answer,
		Explanation: "Compute the mean and the second moment, then subtract the square of the mean; " + c.reason,
		Facts: map[string]string{
			"values": strings.Join(vals, ","), "weights": strings.Join(wts, ","),
			"den": strconv.Itoa(c.den), "answer": answer,
			"mean_terms": strings.Join(meanTerms, " + "), "mean": decimal6(mean),
			"sq_terms": strings.Join(sqTerms, " + "), "second": decimal6(float64(sw2) / float64(c.den)),
			"mean_sq": decimal6(mean * mean),
		},
	}
}

// ---------------------------------------------------------------------------
// Moment generating functions (numeric)
// ---------------------------------------------------------------------------

type mgfGen struct{}

func (g *mgfGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		a, b   int
		reason string
	}
	claims := []claim{
		{a: 2, b: 2, reason: "Writing M(t) = e^{μt + σ²t²/2} reads σ² = 2b = 4."},
		{a: 3, b: 1, reason: "σ² = 2b = 2."},
		{a: 5, b: 3, reason: "σ² = 2b = 6."},
		{a: 1, b: 2, reason: "σ² = 2b = 4."},
		{a: 4, b: 2, reason: "σ² = 2b = 4."},
		{a: 0, b: 1, reason: "σ² = 2b = 2."},
	}
	c := claims[rand.Intn(len(claims))]
	answer := strconv.Itoa(2 * c.b)
	return generator.Problem{
		Question: "A random variable has moment generating function M(t) = e^{" + strconv.Itoa(c.a) +
			"t + " + strconv.Itoa(c.b) + "t²}. What is Var(X)? (enter a number)",
		Answer: answer,
		Explanation: "The exponent has the form μt + σ²t²/2, so the coefficient of t² is σ²/2 and σ² = 2b; " +
			c.reason,
		Facts: map[string]string{
			"a": strconv.Itoa(c.a), "b": strconv.Itoa(c.b), "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// Transformations of random variables (numeric)
// ---------------------------------------------------------------------------

type transformGen struct{}

func (g *transformGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		v      string
		c      int
		reason string
	}
	claims := []claim{
		{v: "4", c: 3, reason: "Var(3X) = 9·4 = 36."},
		{v: "2", c: 5, reason: "Var(5X) = 25·2 = 50."},
		{v: "9", c: 2, reason: "Var(2X) = 4·9 = 36."},
		{v: "1", c: 7, reason: "Var(7X) = 49·1 = 49."},
		{v: "0.5", c: 4, reason: "Var(4X) = 16·0.5 = 8."},
		{v: "3", c: 2, reason: "Var(2X) = 4·3 = 12."},
	}
	c := claims[rand.Intn(len(claims))]
	v, _ := strconv.ParseFloat(c.v, 64)
	answer := decimal6(float64(c.c*c.c) * v)
	return generator.Problem{
		Question: "A random variable X has variance " + c.v + ". What is the variance of the transformed variable Y = " +
			strconv.Itoa(c.c) + "X? (enter a number)",
		Answer:      answer,
		Explanation: "Scaling by a constant scales the variance by the square of the constant: Var(cX) = c²Var(X); " + c.reason,
		Facts:       map[string]string{"v": c.v, "c": strconv.Itoa(c.c), "answer": answer},
	}
}

// ---------------------------------------------------------------------------
// Order statistics (numeric)
// ---------------------------------------------------------------------------

type orderStatsGen struct{}

func (g *orderStatsGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		n      int
		x      string
		reason string
	}
	claims := []claim{
		{n: 3, x: "0.5", reason: "0.5³ = 0.125."},
		{n: 2, x: "0.9", reason: "0.9² = 0.81."},
		{n: 5, x: "0.5", reason: "0.5⁵ = 0.03125."},
		{n: 4, x: "0.5", reason: "0.5⁴ = 0.0625."},
		{n: 2, x: "0.5", reason: "0.5² = 0.25."},
		{n: 3, x: "0.8", reason: "0.8³ = 0.512."},
	}
	c := claims[rand.Intn(len(claims))]
	x, _ := strconv.ParseFloat(c.x, 64)
	answer := decimal6(pow(x, c.n))
	return generator.Problem{
		Question: "Let n independent observations be uniform on [0, 1]. What is P(maximum ≤ " + c.x +
			")? (enter a decimal)",
		Answer: answer,
		Explanation: "The maximum is at most x exactly when all n observations are, so the probability is x^n; " +
			c.reason,
		Facts: map[string]string{"n": strconv.Itoa(c.n), "x": c.x, "answer": answer},
	}
}

func pow(x float64, n int) float64 {
	out := 1.0
	for i := 0; i < n; i++ {
		out *= x
	}
	return out
}

// ---------------------------------------------------------------------------
// The memoryless property (multiple choice)
// ---------------------------------------------------------------------------

type memorylessGen struct{}

func (g *memorylessGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		question    string
		correct     string
		distractors []string
		reason      string
	}
	claims := []claim{
		{
			question:    "The memoryless property says that for s, t positive...",
			correct:     "P(X > s + t | X > s) = P(X > t)",
			distractors: []string{"P(X > s + t) = P(X > s)P(X > t)", "the past predicts the future", "s and t are independent"},
			reason:      "Conditioning on survival past s leaves the remaining lifetime distributed as it was at the start.",
		},
		{
			question:    "Which discrete distribution is memoryless?",
			correct:     "the geometric",
			distractors: []string{"the binomial", "the Poisson", "the hypergeometric"},
			reason:      "Repeated independent trials with a fixed success probability restart at every step.",
		},
		{
			question:    "Which continuous distribution is memoryless?",
			correct:     "the exponential",
			distractors: []string{"the uniform", "the normal", "the beta"},
			reason:      "It is the unique continuous distribution whose survival function satisfies S(s+t) = S(s)S(t).",
		},
		{
			question:    "For a memoryless lifetime, having already survived s units...",
			correct:     "does not change the distribution of the remaining lifetime",
			distractors: []string{"makes failure more likely", "makes failure less likely", "makes failure certain"},
			reason:      "The conditional law of the residual life equals the original law, so age carries no information.",
		},
		{
			question:    "The memoryless property is equivalent to the survival equation...",
			correct:     "P(X > s + t) = P(X > s)P(X > t)",
			distractors: []string{"P(X > s + t) = P(X > s) + P(X > t)", "P(X > s + t) = P(X > t) − P(X > s)", "P(X > s + t) = P(X > s) / P(X > t)"},
			reason:      "Dividing the conditional definition through by the ordinary survival probabilities gives the multiplicative form.",
		},
	}
	c := claims[rand.Intn(len(claims))]
	p := generator.MultipleChoice(c.question, c.correct, c.distractors...)
	p.Explanation = c.reason
	return p
}

// ---------------------------------------------------------------------------
// Markov's inequality (numeric)
// ---------------------------------------------------------------------------

type markovGen struct{}

func (g *markovGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		mean, a int
		reason  string
	}
	claims := []claim{
		{mean: 12, a: 24, reason: "12/24 = 0.5."},
		{mean: 10, a: 25, reason: "10/25 = 0.4."},
		{mean: 15, a: 30, reason: "15/30 = 0.5."},
		{mean: 8, a: 16, reason: "8/16 = 0.5."},
		{mean: 9, a: 12, reason: "9/12 = 0.75."},
		{mean: 10, a: 40, reason: "10/40 = 0.25."},
	}
	c := claims[rand.Intn(len(claims))]
	answer := decimal6(float64(c.mean) / float64(c.a))
	return generator.Problem{
		Question: "A nonnegative random variable has mean " + strconv.Itoa(c.mean) +
			". By Markov's inequality, what is the upper bound on P(X ≥ " + strconv.Itoa(c.a) + ")? (enter a decimal)",
		Answer:      answer,
		Explanation: "Markov's inequality gives P(X ≥ a) ≤ E[X]/a; " + c.reason,
		Facts: map[string]string{
			"mean": strconv.Itoa(c.mean), "a": strconv.Itoa(c.a), "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// Chebyshev's inequality (numeric)
// ---------------------------------------------------------------------------

type chebyshevGen struct{}

func (g *chebyshevGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		sigma, d int
		reason   string
	}
	claims := []claim{
		{sigma: 2, d: 5, reason: "(2/5)² = 0.16."},
		{sigma: 3, d: 10, reason: "(3/10)² = 0.09."},
		{sigma: 4, d: 5, reason: "(4/5)² = 0.64."},
		{sigma: 5, d: 20, reason: "(5/20)² = 0.0625."},
		{sigma: 1, d: 4, reason: "(1/4)² = 0.0625."},
		{sigma: 6, d: 10, reason: "(6/10)² = 0.36."},
	}
	c := claims[rand.Intn(len(claims))]
	answer := decimal6(float64(c.sigma*c.sigma) / float64(c.d*c.d))
	return generator.Problem{
		Question: "A random variable has standard deviation " + strconv.Itoa(c.sigma) +
			". Chebyshev's inequality bounds P(|X − μ| ≥ " + strconv.Itoa(c.d) + ") by what value? (enter a decimal)",
		Answer:      answer,
		Explanation: "Chebyshev gives P(|X − μ| ≥ d) ≤ σ²/d²; " + c.reason,
		Facts: map[string]string{
			"sigma": strconv.Itoa(c.sigma), "d": strconv.Itoa(c.d), "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// The central limit theorem (numeric)
// ---------------------------------------------------------------------------

type cltGen struct{}

func (g *cltGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		n, mu, sigma, x int
		reason          string
	}
	claims := []claim{
		{n: 100, mu: 1, sigma: 2, x: 110, reason: "(110 − 100)/(2·10) = 10/20 = 0.5."},
		{n: 64, mu: 0, sigma: 1, x: 8, reason: "(8 − 0)/(1·8) = 1."},
		{n: 25, mu: 2, sigma: 1, x: 60, reason: "(60 − 50)/(1·5) = 2."},
		{n: 36, mu: 0, sigma: 2, x: 6, reason: "(6 − 0)/(2·6) = 6/12 = 0.5."},
		{n: 16, mu: 5, sigma: 2, x: 88, reason: "(88 − 80)/(2·4) = 8/8 = 1."},
		{n: 100, mu: 3, sigma: 2, x: 320, reason: "(320 − 300)/(2·10) = 20/20 = 1."},
	}
	c := claims[rand.Intn(len(claims))]
	z := float64(c.x-c.n*c.mu) / (float64(c.sigma) * sqrtInt(c.n))
	answer := decimal6(z)
	return generator.Problem{
		Question: "The sum S of n independent copies of a variable with mean μ and variance σ² is standardised as " +
			"z = (x − nμ)/(σ√n). For n = " + strconv.Itoa(c.n) + ", μ = " + strconv.Itoa(c.mu) +
			", σ = " + strconv.Itoa(c.sigma) + " and x = " + strconv.Itoa(c.x) + ", what is z? (enter a number)",
		Answer:      answer,
		Explanation: "Substituting the parameters into the standardisation gives the z-score; " + c.reason,
		Facts: map[string]string{
			"n": strconv.Itoa(c.n), "mu": strconv.Itoa(c.mu), "sigma": strconv.Itoa(c.sigma),
			"x": strconv.Itoa(c.x), "answer": answer,
			"nmu": strconv.Itoa(c.n * c.mu), "dev": strconv.Itoa(c.x - c.n*c.mu), "sqrt_n": decimal6(sqrtInt(c.n)),
			"se": decimal6(float64(c.sigma) * sqrtInt(c.n)),
		},
	}
}

func sqrtInt(n int) float64 {
	if n <= 0 {
		return 0
	}
	x := float64(n)
	for i := 0; i < 40; i++ {
		x = (x + float64(n)/x) / 2
	}
	return x
}

// ---------------------------------------------------------------------------
// The law of total probability (numeric)
// ---------------------------------------------------------------------------

type totalProbabilityGen struct{}

func (g *totalProbabilityGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		w1, a1, t1, w2, a2, t2 int
		reason                 string
	}
	claims := []claim{
		{w1: 1, a1: 3, t1: 4, w2: 1, a2: 1, t2: 4, reason: "(1/2)(3/4) + (1/2)(1/4) = 0.5."},
		{w1: 1, a1: 4, t1: 5, w2: 1, a2: 2, t2: 5, reason: "(1/2)(4/5) + (1/2)(2/5) = 0.6."},
		{w1: 1, a1: 1, t1: 1, w2: 3, a2: 1, t2: 2, reason: "(1/4)(1) + (3/4)(1/2) = 0.625."},
		{w1: 1, a1: 1, t1: 5, w2: 4, a2: 3, t2: 5, reason: "(1/5)(1/5) + (4/5)(3/5) = 0.52."},
		{w1: 2, a1: 1, t1: 2, w2: 3, a2: 1, t2: 4, reason: "(2/5)(1/2) + (3/5)(1/4) = 0.35."},
		{w1: 1, a1: 7, t1: 10, w2: 1, a2: 1, t2: 10, reason: "(1/2)(7/10) + (1/2)(1/10) = 0.4."},
	}
	c := claims[rand.Intn(len(claims))]
	total := c.w1 + c.w2
	totalProb := float64(c.w1)/float64(total)*float64(c.a1)/float64(c.t1) +
		float64(c.w2)/float64(total)*float64(c.a2)/float64(c.t2)
	answer := decimal6(totalProb)
	return generator.Problem{
		Question: "Two boxes are chosen with probabilities proportional to the weights " +
			strconv.Itoa(c.w1) + " and " + strconv.Itoa(c.w2) + ". Box one yields the event with probability " +
			strconv.Itoa(c.a1) + "/" + strconv.Itoa(c.t1) + ", box two with probability " +
			strconv.Itoa(c.a2) + "/" + strconv.Itoa(c.t2) +
			". What is the total probability of the event? (enter a decimal)",
		Answer:      answer,
		Explanation: "The law of total probability averages the conditional probabilities with the prior weights; " + c.reason,
		Facts: map[string]string{
			"w1": strconv.Itoa(c.w1), "a1": strconv.Itoa(c.a1), "t1": strconv.Itoa(c.t1),
			"w2": strconv.Itoa(c.w2), "a2": strconv.Itoa(c.a2), "t2": strconv.Itoa(c.t2),
			"answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// Joint probability mass functions (numeric)
// ---------------------------------------------------------------------------

type jointPMFGen struct{}

func (g *jointPMFGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	pairs := [][2]int{{0, 0}, {0, 1}, {1, 0}, {1, 1}}
	type claim struct {
		weights []int
		index   int
		reason  string
	}
	claims := []claim{
		{weights: []int{1, 2, 3, 4}, index: 3, reason: "The weights sum to 10 and the pair (1, 0) carries 3, so the mass is 3/10 = 0.3."},
		{weights: []int{2, 2, 1, 5}, index: 4, reason: "They sum to 10 and (1, 1) carries 5, so 5/10 = 0.5."},
		{weights: []int{3, 3, 1, 3}, index: 2, reason: "They sum to 10 and (0, 1) carries 3, so 3/10 = 0.3."},
		{weights: []int{4, 1, 1, 4}, index: 3, reason: "They sum to 10 and (1, 0) carries 1, so 1/10 = 0.1."},
		{weights: []int{5, 2, 2, 1}, index: 1, reason: "They sum to 10 and (0, 0) carries 5, so 5/10 = 0.5."},
		{weights: []int{1, 1, 2, 6}, index: 2, reason: "They sum to 10 and (0, 1) carries 1, so 1/10 = 0.1."},
	}
	c := claims[rand.Intn(len(claims))]
	total := 0
	for _, w := range c.weights {
		total += w
	}
	weight := c.weights[c.index-1]
	pair := pairs[c.index-1]
	answer := decimal6(float64(weight) / float64(total))
	parts := make([]string, len(c.weights))
	for i, w := range c.weights {
		parts[i] = strconv.Itoa(w)
	}
	return generator.Problem{
		Question: "A joint mass function on the four pairs (0,0), (0,1), (1,0) and (1,1) is proportional to the weights " +
			strings.Join(parts, ", ") + " in that order. What is P(X = " + strconv.Itoa(pair[0]) +
			", Y = " + strconv.Itoa(pair[1]) + ")? (enter a decimal)",
		Answer:      answer,
		Explanation: "A joint mass function sums to one over its support, so each weight is divided by the total; " + c.reason,
		Facts: map[string]string{
			"weights": strings.Join(parts, ","), "index": strconv.Itoa(c.index),
			"total": strconv.Itoa(total), "weight": strconv.Itoa(weight),
			"pair": strconv.Itoa(pair[0]) + "," + strconv.Itoa(pair[1]), "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// The Poisson process (numeric)
// ---------------------------------------------------------------------------

type poissonProcessGen struct{}

func (g *poissonProcessGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		m, k   int
		reason string
	}
	claims := []claim{
		{m: 2, k: 1, reason: "e^(-2)·2^1/1! = 0.270671 to six places."},
		{m: 3, k: 2, reason: "e^(-3)·3^2/2! = 0.224042 to six places."},
		{m: 1, k: 0, reason: "e^(-1) = 0.367879 to six places."},
		{m: 4, k: 4, reason: "e^(-4)·4^4/4! = 0.195367 to six places."},
		{m: 2, k: 2, reason: "e^(-2)·2^2/2! = 0.270671 to six places."},
		{m: 3, k: 0, reason: "e^(-3) = 0.049787 to six places."},
	}
	c := claims[rand.Intn(len(claims))]
	answer := decimal6(math.Exp(-float64(c.m)) * pow(float64(c.m), c.k) / factF(c.k))
	return generator.Problem{
		Question: "In a Poisson process the number of arrivals in a fixed interval has mean " +
			strconv.Itoa(c.m) + ". What is the probability of exactly " + strconv.Itoa(c.k) +
			" arrivals? (round to six decimals)",
		Answer: answer,
		Explanation: "The count in an interval is Poisson with mean equal to the rate times the length, so " +
			"P(N = k) = e^(-m)·m^k/k!; " + c.reason,
		Facts: map[string]string{
			"m": strconv.Itoa(c.m), "k": strconv.Itoa(c.k), "answer": answer,
		},
	}
}

func factF(n int) float64 {
	out := 1.0
	for i := 2; i <= n; i++ {
		out *= float64(i)
	}
	return out
}

// ---------------------------------------------------------------------------
// The exponential family (multiple choice)
// ---------------------------------------------------------------------------

type exponentialFamilyGen struct{}

func (g *exponentialFamilyGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		question    string
		correct     string
		distractors []string
		reason      string
	}
	claims := []claim{
		{
			question:    "A distribution is a member of the exponential family when its mass or density can be written as...",
			correct:     "h(x) exp(η(θ)T(x) − A(θ))",
			distractors: []string{"h(x) + η(θ)T(x)", "η(θ)T(x) divided by A(θ)", "h(x) exp(η(θ) + T(x))"},
			reason:      "The density splits into an x-only part and an exponential whose argument is linear in a statistic and a parameter.",
		},
		{
			question:    "In the exponential-family form, the quantity T(x) is called...",
			correct:     "the sufficient statistic",
			distractors: []string{"the natural parameter", "the normalising constant", "the support"},
			reason:      "It is the data summary that the natural parameter multiplies.",
		},
		{
			question:    "Which of these distributions is a member of the exponential family?",
			correct:     "the Poisson",
			distractors: []string{"the Cauchy", "the uniform on the interval from 0 to θ", "a two-component Gaussian mixture"},
			reason:      "The Poisson density separates; the Cauchy has a logarithmic term that does not, and the other two have a parameter-dependent support or a mixture structure.",
		},
		{
			question:    "The natural parameter η(θ) is...",
			correct:     "the parameter that multiplies the sufficient statistic",
			distractors: []string{"always equal to the mean", "the normalising constant", "the support of the distribution"},
			reason:      "It is the coefficient of T(x) in the exponent.",
		},
		{
			question:    "The log-partition function A(θ)...",
			correct:     "normalises the density so it integrates to one",
			distractors: []string{"is the sufficient statistic", "is the mean of the distribution", "is the support"},
			reason:      "It is whatever constant makes the integral over the support equal to one.",
		},
	}
	c := claims[rand.Intn(len(claims))]
	p := generator.MultipleChoice(c.question, c.correct, c.distractors...)
	p.Explanation = c.reason
	return p
}

// ---------------------------------------------------------------------------
// The weak law of large numbers (multiple choice)
// ---------------------------------------------------------------------------

type weakLawGen struct{}

func (g *weakLawGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	type claim struct {
		question    string
		correct     string
		distractors []string
		reason      string
	}
	claims := []claim{
		{
			question:    "The weak law of large numbers says that the sample mean...",
			correct:     "converges in probability to the population mean",
			distractors: []string{"converges almost surely to the population mean", "is exactly the population mean", "converges in distribution to the standard normal"},
			reason:      "Convergence in probability is the weak mode; almost-sure convergence is the strong law, and a normal limit is the central limit theorem.",
		},
		{
			question:    "Convergence in probability to a constant means that for every positive ε...",
			correct:     "the probability that the difference exceeds ε tends to zero",
			distractors: []string{"the variable eventually equals the constant", "the densities converge pointwise", "the means form a convergent sequence"},
			reason:      "The defining requirement is that the probability of a departure larger than ε vanishes.",
		},
		{
			question:    "The standard proof of the weak law applies Chebyshev's inequality, so it needs the observations to have finite...",
			correct:     "variance",
			distractors: []string{"support", "median", "mode"},
			reason:      "Chebyshev bounds the tail by σ²/(n ε²), which needs a finite variance.",
		},
		{
			question:    "Chebyshev's inequality bounds the probability that the sample mean departs from μ by at least ε by...",
			correct:     "σ² divided by n ε²",
			distractors: []string{"σ² divided by ε²", "one divided by n", "σ divided by n ε"},
			reason:      "The sample mean has variance σ²/n, and Chebyshev divides that by ε².",
		},
		{
			question:    "Which of the two laws is the weaker claim?",
			correct:     "the weak law of large numbers",
			distractors: []string{"the strong law of large numbers", "they are equivalent", "neither follows from the other"},
			reason:      "Almost-sure convergence implies convergence in probability, so the strong law is the stronger claim.",
		},
	}
	c := claims[rand.Intn(len(claims))]
	p := generator.MultipleChoice(c.question, c.correct, c.distractors...)
	p.Explanation = c.reason
	return p
}
