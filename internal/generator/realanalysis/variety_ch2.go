package realanalysis

// Concrete instances for Abbott chapter 2: limits and ε–N thresholds of
// explicit sequences, monotone recursions, subsequential limits, Cauchy
// sequences, sums of geometric and telescoping series, and the convergence
// tests applied to series with numbers in them.

import (
	"math/big"
	"math/rand"
	"strconv"
	"strings"

	"github.com/chuma-beep/mathua/internal/generator"
)

// ratSeq describes a_n = (a·n + b)/(c·n + d) with positive c, d and b ≥ 0.
type ratSeq struct{ a, b, c, d int }

func randRatSeq() ratSeq {
	return ratSeq{a: randIn(1, 9), b: randIn(0, 9), c: randIn(1, 6), d: randIn(1, 9)}
}

func (s ratSeq) String() string {
	num := coefX(s.a, "n")
	if s.b != 0 {
		num += " + " + strconv.Itoa(s.b)
	}
	return "(" + num + ")/(" + coefX(s.c, "n") + " + " + strconv.Itoa(s.d) + ")"
}

func (s ratSeq) limit() *big.Rat { return rq(int64(s.a), int64(s.c)) }

// overN is the sequence with numerator and denominator divided by n, without
// a "+ 0/n" when b = 0.
func (s ratSeq) overN() string {
	num := strconv.Itoa(s.a)
	if s.b != 0 {
		num = "(" + num + " + " + strconv.Itoa(s.b) + "/n)"
	}
	return num + "/(" + strconv.Itoa(s.c) + " + " + strconv.Itoa(s.d) + "/n)"
}

// errDen is c(c·v + d), the denominator of |a_n − a/c|, written "(v + d)"
// when c = 1 rather than "(1(v + d))" (Q35).
func (s ratSeq) errDen(v string) string {
	if s.c == 1 {
		return "(" + v + " + " + strconv.Itoa(s.d) + ")"
	}
	return "(" + strconv.Itoa(s.c) + "(" + coefX(s.c, v) + " + " + strconv.Itoa(s.d) + "))"
}

// limitQuot prints "a/c = " unless c = 1, where a/1 would be an artefact.
func limitQuot(s ratSeq) string {
	if s.c == 1 {
		return ""
	}
	return strconv.Itoa(s.a) + "/" + strconv.Itoa(s.c) + " = "
}

// ---------------------------------------------------------------------------
// ra.seq.convergence (numeric)
// ---------------------------------------------------------------------------

// sequenceVariant asks for the limit of a rational sequence, or for the
// least N beyond which it is within ε of its limit.
func sequenceVariant() generator.Problem {
	s := randRatSeq()
	diff := absInt(s.b*s.c - s.a*s.d)
	facts := map[string]string{
		"a": strconv.Itoa(s.a), "b": strconv.Itoa(s.b), "c": strconv.Itoa(s.c), "d": strconv.Itoa(s.d),
	}
	if rand.Intn(2) == 0 || diff == 0 {
		answer := ratAnswer(s.limit())
		facts["kind"], facts["answer"] = "limit", answer
		return generator.Problem{
			Question: "What is the limit of the sequence a_n = " + s.String() + " as n tends to infinity? (enter a decimal)",
			Answer:   answer,
			Explanation: "Dividing the numerator and denominator by n leaves " + s.overN() + ", and the terms over n tend to 0, so the limit is " +
				limitQuot(s) + answer + ".",
			Facts: facts,
		}
	}
	e := pick([]int{10, 100, 1000})
	// |a_n − a/c| = diff/(c(cn + d)), decreasing in n; find the least N with
	// diff·e < c(cN + d).
	n := 1
	for diff*e >= s.c*(s.c*n+s.d) {
		n++
	}
	answer := strconv.Itoa(n)
	ls := ratDec(s.limit())
	eps := decimal6(1 / float64(e))
	facts["kind"], facts["e"], facts["answer"] = "N", strconv.Itoa(e), answer
	return generator.Problem{
		Question: "The sequence a_n = " + s.String() + " converges to " + ls + ". What is the smallest N such that |a_n − " + ls +
			"| < " + eps + " for every n ≥ N? (enter a whole number)",
		Answer: answer,
		Explanation: "Subtracting the limit gives |a_n − " + ls + "| = " + strconv.Itoa(diff) + "/" + s.errDen("n") +
			", which decreases as n grows, so it is enough to find the first n where it drops below " + eps + ". That needs " +
			strings.TrimSuffix(strings.TrimPrefix(s.errDen("N"), "("), ")") + " > " + strconv.Itoa(diff*e) + ", which first happens at N = " + answer + ".",
		Facts: facts,
	}
}

// ---------------------------------------------------------------------------
// ra.seq.monotone (numeric)
// ---------------------------------------------------------------------------

func monotoneVariant() generator.Problem {
	switch rand.Intn(3) {
	case 0:
		l := randIn(2, 10)
		a := l * (l - 1)
		answer := strconv.Itoa(l)
		return generator.Problem{
			Question: "The sequence x_1 = 0, x_{n+1} = √(" + strconv.Itoa(a) + " + x_n) is increasing and bounded above, so it converges. What is its limit? (enter a decimal)",
			Answer:   answer,
			Explanation: "The limit L satisfies L = √(" + strconv.Itoa(a) + " + L), so L² − L − " + strconv.Itoa(a) + " = 0, whose nonnegative root is " + answer +
				". The sequence starts at 0 and stays below that root, so it is the limit.",
			Facts: map[string]string{"kind": "sqrt", "a": strconv.Itoa(a), "answer": answer},
		}
	case 1:
		m := pick([]int{2, 3, 5, 6, 11})
		k := randIn(1, 9)
		l := rq(int64(k), int64(m-1))
		answer := ratAnswer(l)
		ls := ratDec(l)
		return generator.Problem{
			Question: "The sequence x_1 = 0, x_{n+1} = (x_n + " + strconv.Itoa(k) + ")/" + strconv.Itoa(m) +
				" is increasing and bounded above. What is its limit? (enter a decimal)",
			Answer: answer,
			Explanation: "If x_n < " + ls + " then x_{n+1} < (" + ls + " + " + strconv.Itoa(k) + ")/" + strconv.Itoa(m) + " = " + ls +
				", and x_{n+1} − x_n = (" + strconv.Itoa(k) + " − " + coefX(m-1, "x_n") + ")/" + strconv.Itoa(m) +
				" > 0, so the sequence is increasing and bounded by " + ls + ". By the monotone convergence theorem it has a limit L, and L = (L + " +
				strconv.Itoa(k) + ")/" + strconv.Itoa(m) + " gives L = " + ls + ".",
			Facts: map[string]string{"kind": "avg", "k": strconv.Itoa(k), "m": strconv.Itoa(m), "answer": answer},
		}
	default:
		b := randIn(2, 9)
		answer := strconv.Itoa(b)
		return generator.Problem{
			Question: "The sequence x_1 = 1, x_{n+1} = √(" + strconv.Itoa(b) + "·x_n) is increasing and bounded above. What is its limit? (enter a decimal)",
			Answer:   answer,
			Explanation: "If 1 ≤ x_n < " + answer + " then x_{n+1} = √(" + answer + "·x_n) lies between x_n and " + answer +
				", so the sequence increases and stays below " + answer + ". Its limit L ≥ 1 satisfies L² = " + answer + "L, so L = " + answer + ".",
			Facts: map[string]string{"kind": "geo", "b": strconv.Itoa(b), "answer": answer},
		}
	}
}

// ---------------------------------------------------------------------------
// ra.seq.series (numeric)
// ---------------------------------------------------------------------------

// harmonicTerms writes 1 + 1/2 + ... + 1/k out in full (k is at most 4 where
// it is used), so k = 2 reads "1 + 1/2", not "1 + … + 1/2".
func harmonicTerms(k int) string {
	t := []string{"1"}
	for i := 2; i <= k; i++ {
		t = append(t, "1/"+strconv.Itoa(i))
	}
	return strings.Join(t, " + ")
}

// harmonic returns 1 + 1/2 + … + 1/k exactly.
func harmonic(k int) *big.Rat {
	h := new(big.Rat)
	for j := 1; j <= k; j++ {
		h = radd(h, rq(1, int64(j)))
	}
	return h
}

func seriesVariant() generator.Problem {
	switch rand.Intn(4) {
	case 0:
		q := pick([]int{2, 3, 4, 5, 10})
		p := randIn(1, q-1)
		if rand.Intn(2) == 0 {
			p = -p
		}
		a := randIn(1, 6)
		r := rq(int64(p), int64(q))
		sum := rquo(ri(int64(a)), rsub(ri(1), r))
		answer := ratAnswer(sum)
		return generator.Problem{
			Question: "What is the sum of the geometric series Σ_{n=0}^∞ " + strconv.Itoa(a) + "·(" + ratS(r) + ")^n? (enter a decimal)",
			Answer:   answer,
			Explanation: "The ratio r = " + ratS(r) + " has |r| < 1, so the partial sums " + strconv.Itoa(a) + "(1 − r^(N+1))/(1 − r) converge to " +
				strconv.Itoa(a) + "/(1 − r) = " + ratS(sum) + ".",
			Facts: map[string]string{"kind": "geo", "a": strconv.Itoa(a), "p": strconv.Itoa(p), "q": strconv.Itoa(q), "answer": answer},
		}
	case 1:
		q := pick([]int{2, 3, 4, 5, 10})
		p := randIn(1, q-1)
		m := randIn(1, 3)
		r := rq(int64(p), int64(q))
		rm := ri(1)
		for i := 0; i < m; i++ {
			rm = rmul(rm, r)
		}
		sum := rquo(rm, rsub(ri(1), r))
		answer := ratAnswer(sum)
		return generator.Problem{
			Question: "What is the sum of Σ_{n=" + strconv.Itoa(m) + "}^∞ (" + ratS(r) + ")^n? (enter a decimal)",
			Answer:   answer,
			Explanation: "Factor out the first term: the series is " + powR(m) + "·Σ_{j=0}^∞ r^j = " + powR(m) + "/(1 − r) with r = " +
				ratS(r) + ", which is " + ratS(sum) + ".",
			Facts: map[string]string{"kind": "tail", "p": strconv.Itoa(p), "q": strconv.Itoa(q), "m": strconv.Itoa(m), "answer": answer},
		}
	case 2:
		k := randIn(1, 4)
		sum := rquo(harmonic(k), ri(int64(k)))
		answer := ratAnswer(sum)
		ks := strconv.Itoa(k)
		why := "Write 1/(n(n + " + ks + ")) = (1/" + ks + ")(1/n − 1/(n + " + ks + ")). In the partial sums everything cancels except the first " + ks +
			" positive terms and " + ks + " negative terms that tend to 0, so the sum is (1/" + ks + ")(" + harmonicTerms(k) + ") = " + ratS(sum) + "."
		if k == 1 {
			why = "Write 1/(n(n + 1)) = 1/n − 1/(n + 1). The partial sums telescope to 1 − 1/(N + 1), which tends to 1."
		}
		return generator.Problem{
			Question:    "What is the sum of the telescoping series Σ_{n=1}^∞ 1/(n(n + " + ks + "))? (enter a decimal)",
			Answer:      answer,
			Explanation: why,
			Facts:       map[string]string{"kind": "tele", "k": ks, "answer": answer},
		}
	default:
		n := pick([]int{1, 3, 4, 7, 9, 15, 19, 24, 39, 49, 99})
		sum := rq(int64(n), int64(n+1))
		answer := ratAnswer(sum)
		ns := strconv.Itoa(n)
		return generator.Problem{
			Question: "What is the partial sum s_" + ns + " = Σ_{k=1}^{" + ns + "} 1/(k(k + 1))? (enter a decimal)",
			Answer:   answer,
			Explanation: "Since 1/(k(k + 1)) = 1/k − 1/(k + 1), the sum telescopes to 1 − 1/" + strconv.Itoa(n+1) + " = " + answer +
				". As N grows, s_N = 1 − 1/(N + 1) tends to 1, the sum of the series.",
			Facts: map[string]string{"kind": "partial", "n": ns, "answer": answer},
		}
	}
}

// ---------------------------------------------------------------------------
// ra.seq.subsequence (multiple choice)
// ---------------------------------------------------------------------------

func setOf(xs ...int) string {
	s := "{"
	for i, x := range xs {
		if i > 0 {
			s += ", "
		}
		s += itoa(x)
	}
	return s + "}"
}

func subsequenceLimitsClaim() mcClaim {
	c := randIn(1, 5)
	k := randIn(1, 5)
	switch rand.Intn(3) {
	case 0:
		correct := setOf(-c, c)
		return mcClaim{
			q:           "What is the set of all subsequential limits of a_n = " + strconv.Itoa(c) + "·(−1)^n + " + strconv.Itoa(k) + "/n?",
			correct:     correct,
			distractors: distractorsFrom(correct, setOf(c), setOf(0), setOf(-c, 0, c), setOf(-c+k, c+k)),
			reason: "The even terms " + strconv.Itoa(c) + " + " + strconv.Itoa(k) + "/n tend to " + strconv.Itoa(c) + " and the odd terms tend to " + itoa(-c) +
				". Every subsequence has infinitely many even or infinitely many odd terms, so no other limit is possible.",
		}
	case 1:
		m := randIn(2, 6)
		vals := make([]int, m)
		for i := range vals {
			vals[i] = i
		}
		correct := setOf(vals...)
		return mcClaim{
			q:           "Let a_n be the remainder when n is divided by " + strconv.Itoa(m) + ". What is the set of all subsequential limits of (a_n)?",
			correct:     correct,
			distractors: distractorsFrom(correct, setOf(vals[1:]...), setOf(append(vals, m)...), setOf(0), "the empty set, because (a_n) diverges"),
			reason: "The sequence cycles through 1, …, " + strconv.Itoa(m-1) + ", 0, so each of the values 0 to " + strconv.Itoa(m-1) +
				" occurs infinitely often and gives a constant subsequence; a convergent subsequence of integers is eventually constant, so there are no others.",
		}
	default:
		correct := setOf(-c, 0, c)
		return mcClaim{
			q:           "What is the set of all subsequential limits of a_n = " + strconv.Itoa(c) + "·cos(nπ/2)?",
			correct:     correct,
			distractors: distractorsFrom(correct, setOf(-c, c), setOf(0), setOf(c), setOf(-c, 0)),
			reason: "cos(nπ/2) cycles through 0, −1, 0, 1, so the sequence takes the values 0, " + itoa(-c) + " and " + strconv.Itoa(c) +
				" infinitely often; those three constant subsequences give every possible limit.",
		}
	}
}

func subsequenceSameLimitClaim() mcClaim {
	s := randRatSeq()
	idx := pick([]string{"a_{2k}", "a_{k²}", "a_{3k + 1}", "a_{2^k}", "a_{5k}"})
	correct := ratS(s.limit())
	return mcClaim{
		q:       "a_n = " + s.String() + ". What is the limit, as k tends to infinity, of the subsequence " + idx + "?",
		correct: correct,
		distractors: distractorsFrom(correct, ratS(rq(int64(2*s.a), int64(s.c))), ratS(rq(int64(s.a), int64(2*s.c))),
			ratS(rq(int64(s.b), int64(s.d))), ratS(rq(int64(s.a+s.b), int64(s.c+s.d))), "0"),
		reason: "The full sequence converges to the ratio of leading coefficients, " + correct +
			", and every subsequence of a convergent sequence converges to the same limit.",
	}
}

func subsequenceOddEvenClaim() mcClaim {
	c := randIn(1, 6)
	k := randIn(1, 5)
	odd := rand.Intn(2) == 0
	idx, want, why := "a_{2k}", c, "even indices make (−1)^n = 1"
	if odd {
		idx, want, why = "a_{2k+1}", -c, "odd indices make (−1)^n = −1"
	}
	correct := itoa(want)
	return mcClaim{
		q:           "a_n = (−1)^n·(" + strconv.Itoa(c) + " + " + strconv.Itoa(k) + "/n). What is the limit of the subsequence " + idx + "?",
		correct:     correct,
		distractors: distractorsFrom(correct, itoa(-want), "0", itoa(want+k), "it does not converge"),
		reason:      "The " + why + ", and " + strconv.Itoa(k) + "/n tends to 0, so the subsequence tends to " + correct + ". The full sequence diverges because its odd and even subsequences have different limits.",
	}
}

// ---------------------------------------------------------------------------
// ra.seq.bolzano_weierstrass (multiple choice)
// ---------------------------------------------------------------------------

// Every unbounded sequence here has |a_n| → ∞, so it has no convergent
// subsequence at all; every bounded one has one by Bolzano–Weierstrass.
func bwBoundedSeqs(c, k int) []string {
	cs, ks := strconv.Itoa(c), strconv.Itoa(k)
	return []string{
		cs + "·sin(n)",
		"(−1)^n·" + cs + " + " + ks + "/n",
		"(−1)^n·n/(n + " + ks + ")",
		cs + "·cos(n²)",
		cs + " + (−1)^n",
		"sin(n)/(" + ks + " + cos(n)/2)",
	}
}

func bwUnboundedSeqs(c, k int) []string {
	cs, ks := strconv.Itoa(c), strconv.Itoa(k)
	return []string{
		"(−1)^n·n",
		cs + "n + sin(n)",
		"√n + " + cs,
		"n²/(n + " + ks + ")",
		"(−1)^n·√n",
		cs + "·ln(n + 1)",
	}
}

func bwWhichClaim() mcClaim {
	c, k := randIn(2, 6), randIn(1, 5)
	b, u := bwBoundedSeqs(c, k), bwUnboundedSeqs(c, k)
	if rand.Intn(2) == 0 {
		correct := pick(b)
		perm := rand.Perm(len(u))
		return mcClaim{
			q:           "Which of these sequences must have a convergent subsequence?",
			correct:     "a_n = " + correct,
			distractors: []string{"a_n = " + u[perm[0]], "a_n = " + u[perm[1]], "a_n = " + u[perm[2]]},
			reason:      "a_n = " + correct + " is bounded, so Bolzano–Weierstrass gives a convergent subsequence. Each of the others satisfies |a_n| → ∞, so every subsequence is unbounded and none converges.",
		}
	}
	correct := pick(u)
	perm := rand.Perm(len(b))
	return mcClaim{
		q:           "Three of these sequences are bounded. Which one has no convergent subsequence?",
		correct:     "a_n = " + correct,
		distractors: []string{"a_n = " + b[perm[0]], "a_n = " + b[perm[1]], "a_n = " + b[perm[2]]},
		reason:      "a_n = " + correct + " satisfies |a_n| → ∞, so every subsequence is unbounded and none converges. The other three are bounded, and Bolzano–Weierstrass gives each a convergent subsequence.",
	}
}

func bwBisectionClaim() mcClaim {
	m := randIn(1, 8)
	k := randIn(2, 8)
	want := rq(int64(2*m), int64(1)<<k)
	correct := ratS(want)
	return mcClaim{
		q: "In the proof of Bolzano–Weierstrass, a sequence with |a_n| ≤ " + strconv.Itoa(m) + " starts in I_0 = [" + itoa(-m) + ", " + strconv.Itoa(m) +
			"], and each step keeps a closed half containing infinitely many terms. What is the length of I_" + strconv.Itoa(k) + "?",
		correct: correct,
		distractors: distractorsFrom(correct, ratS(rq(int64(m), int64(1)<<k)), ratS(rq(int64(2*m), int64(1)<<(k-1))),
			ratS(rq(int64(2*m), int64(1)<<(k+1))), ratS(rq(int64(2*m), int64(k)))),
		reason: "I_0 has length " + strconv.Itoa(2*m) + " and each step halves it, so I_" + strconv.Itoa(k) + " has length " + strconv.Itoa(2*m) + "/2^" + strconv.Itoa(k) +
			" = " + correct + ". The lengths tend to 0, which is what pins down the limit of the chosen subsequence.",
	}
}

func bwCosineClaim() mcClaim {
	c := 2 * randIn(1, 4)
	r := randIn(0, 5)
	// cos(rπ/3) for r = 0..5.
	cosv := []*big.Rat{ri(1), rq(1, 2), rq(-1, 2), ri(-1), rq(-1, 2), rq(1, 2)}
	want := rmul(ri(int64(c)), cosv[r])
	correct := ratS(want)
	idx := "a_{6k}"
	if r != 0 {
		idx = "a_{6k + " + strconv.Itoa(r) + "}"
	}
	return mcClaim{
		q:           "a_n = " + strconv.Itoa(c) + "·cos(nπ/3) is bounded, so Bolzano–Weierstrass applies. What is the limit of the subsequence " + idx + "?",
		correct:     correct,
		distractors: distractorsFrom(correct, strconv.Itoa(c), itoa(-c), strconv.Itoa(c/2), itoa(-c/2), "0"),
		reason: "cos(nπ/3) repeats with period 6, so " + idx + " is the constant " + strconv.Itoa(c) + "·cos(" + strconv.Itoa(r) + "π/3) = " + correct +
			". The sequence has four subsequential limits, so it does not converge itself.",
	}
}

// ---------------------------------------------------------------------------
// ra.seq.cauchy (multiple choice)
// ---------------------------------------------------------------------------

func cauchyConvSeqs(c, k int) []string {
	cs, ks := strconv.Itoa(c), strconv.Itoa(k)
	return []string{
		cs + " + " + ks + "/n",
		"(−1)^n·" + ks + "/n",
		ks + "/n²",
		"(" + cs + "n + 1)/(n + " + ks + ")",
		"1/2 + 1/4 + … + 1/2^n",
		cs + "·(1/2)^n",
	}
}

func cauchyDivSeqs(c, k int) []string {
	cs := strconv.Itoa(c)
	return []string{
		"(−1)^n·" + cs,
		"√n",
		"ln n",
		"1 + 1/2 + … + 1/n",
		cs + "n",
		cs + "·sin(nπ/2)",
	}
}

func cauchyWhichClaim() mcClaim {
	c, k := randIn(2, 6), randIn(1, 5)
	conv, div := cauchyConvSeqs(c, k), cauchyDivSeqs(c, k)
	if rand.Intn(2) == 0 {
		correct := pick(conv)
		perm := rand.Perm(len(div))
		return mcClaim{
			q:           "Which of these sequences is a Cauchy sequence?",
			correct:     "a_n = " + correct,
			distractors: []string{"a_n = " + div[perm[0]], "a_n = " + div[perm[1]], "a_n = " + div[perm[2]]},
			reason: "In R a sequence is Cauchy exactly when it converges. a_n = " + correct +
				" converges; each of the others diverges, either by oscillating or by growing without bound.",
		}
	}
	correct := pick(div)
	perm := rand.Perm(len(conv))
	return mcClaim{
		q:           "Which of these sequences is not a Cauchy sequence?",
		correct:     "a_n = " + correct,
		distractors: []string{"a_n = " + conv[perm[0]], "a_n = " + conv[perm[1]], "a_n = " + conv[perm[2]]},
		reason: "In R a sequence is Cauchy exactly when it converges. a_n = " + correct +
			" diverges, so it is not Cauchy; each of the others converges.",
	}
}

func cauchyTrapClaim() mcClaim {
	m := randIn(2, 9)
	ms := strconv.Itoa(m)
	correct := "it is not Cauchy, because |x_{" + ms + "n} − x_n| = ln " + ms + " for every n"
	return mcClaim{
		q:       "x_n = ln n satisfies |x_{n+1} − x_n| → 0. Which statement about (x_n) is correct?",
		correct: correct,
		distractors: []string{
			"it is Cauchy, because consecutive terms get arbitrarily close",
			"it is Cauchy, because it is increasing",
			"it converges to ln " + ms,
		},
		reason: "The Cauchy condition compares any two late terms, not only neighbours. Taking m = " + ms + "n gives |x_m − x_n| = ln(" + ms +
			"n) − ln n = ln " + ms + " no matter how large n is, so the terms never cluster; indeed ln n → ∞.",
	}
}

func cauchyThresholdClaim() mcClaim {
	k := randIn(1, 5)
	p := pick([]int{1, 2, 3, 4, 5, 6, 7, 8})
	// |k/m − k/n| < k/N for m, n ≥ N, and k/(N−1) − k/n exceeds ε for large n
	// once k/(N−1) > ε, so the least N is the least N with k/N ≤ ε = p/100.
	n := (100*k + p - 1) / p
	correct := strconv.Itoa(n)
	eps := hundredths(p)
	ks := strconv.Itoa(k)
	return mcClaim{
		q:           "For a_n = " + ks + "/n, what is the smallest N such that |a_m − a_n| < " + eps + " for all m, n ≥ N?",
		correct:     correct,
		distractors: distractorsFrom(correct, strconv.Itoa(n-1), strconv.Itoa(n+1), strconv.Itoa(2*n), strconv.Itoa(n/2)),
		reason: "For m, n ≥ N both terms lie in (0, " + ks + "/N], so |a_m − a_n| < " + ks + "/N, and the gap comes arbitrarily close to " + ks +
			"/N. The condition therefore holds exactly when " + ks + "/N ≤ " + eps + ", that is N ≥ " + decimal6(float64(100*k)/float64(p)) + ", so the smallest N is " + correct + ".",
	}
}

// ---------------------------------------------------------------------------
// Series pool shared by absolute convergence and rearrangement
// ---------------------------------------------------------------------------

const (
	seriesAbs = iota
	seriesCond
	seriesDiv
)

type raSeries struct {
	desc  string
	class int
	why   string
}

func randAltSeries() raSeries {
	a, b := randIn(1, 5), randIn(1, 9)
	bs := strconv.Itoa(b)
	switch rand.Intn(6) {
	case 0:
		type pw struct {
			s   string
			big bool
		}
		p := pick([]pw{{"1/3", false}, {"1/2", false}, {"2/3", false}, {"", false}, {"3/2", true}, {"2", true}, {"3", true}, {"5/4", true}})
		den := "n"
		ps := "1"
		if p.s != "" {
			den, ps = "n^("+p.s+")", p.s
		}
		if p.s == "2" || p.s == "3" {
			den = "n^" + p.s
		}
		if p.big {
			return raSeries{"Σ (−1)^n/" + den, seriesAbs,
				"Σ 1/" + den + " is a p-series with p = " + ps + " > 1, so the series converges absolutely."}
		}
		return raSeries{"Σ (−1)^n/" + den, seriesCond,
			"Σ 1/" + den + " is a p-series with p = " + ps + " ≤ 1 and diverges, but 1/" + den + " decreases to 0, so the alternating series test gives convergence: the series is conditionally convergent."}
	case 1:
		return raSeries{"Σ (−1)^n/(" + coefX(a, "n") + " + " + bs + ")", seriesCond,
			"The terms 1/(" + coefX(a, "n") + " + " + bs + ") decrease to 0, so the alternating series converges, but by limit comparison with the harmonic series their sum diverges: the series is conditionally convergent."}
	case 2:
		return raSeries{"Σ (−1)^n/(n² + " + bs + ")", seriesAbs,
			"|(−1)^n/(n² + " + bs + ")| ≤ 1/n², and Σ 1/n² converges, so the series converges absolutely."}
	case 3:
		return raSeries{"Σ (−1)^n·n/(n + " + bs + ")", seriesDiv,
			"The terms have absolute value n/(n + " + bs + "), which tends to 1, so they do not tend to 0 and the series diverges."}
	case 4:
		p, q := randIn(1, 6), randIn(1, 6)
		for p == q {
			q = randIn(1, 6)
		}
		r := "(−" + strconv.Itoa(p) + "/" + strconv.Itoa(q) + ")^n"
		if p < q {
			return raSeries{"Σ " + r, seriesAbs, "It is geometric with |r| = " + ratS(rq(int64(p), int64(q))) + " < 1, so it converges absolutely."}
		}
		return raSeries{"Σ " + r, seriesDiv, "It is geometric with |r| = " + ratS(rq(int64(p), int64(q))) + " > 1, so its terms do not tend to 0 and it diverges."}
	default:
		return raSeries{"Σ (−1)^n/√(n + " + bs + ")", seriesCond,
			"1/√(n + " + bs + ") decreases to 0, so the alternating series converges, but Σ 1/√(n + " + bs + ") diverges by comparison with a p-series with p = 1/2: the series is conditionally convergent."}
	}
}

func absoluteClassClaim() mcClaim {
	s := randAltSeries()
	labels := []string{"absolutely convergent", "conditionally convergent", "divergent"}
	correct := labels[s.class]
	return mcClaim{
		q:           "Classify the series " + s.desc + ", with n running from 1.",
		correct:     correct,
		distractors: distractorsFrom(correct, labels...),
		reason:      s.why,
	}
}

func absoluteErrorClaim() mcClaim {
	n := pick([]int{3, 4, 9, 19, 24, 49, 99})
	p := randIn(1, 2)
	den := "k"
	if p == 2 {
		den = "k²"
	}
	term := func(j int) string {
		v := j
		if p == 2 {
			v = j * j
		}
		return "1/" + strconv.Itoa(v)
	}
	correct := term(n + 1)
	return mcClaim{
		q: "The alternating series Σ_{k=1}^∞ (−1)^(k+1)/" + den + " has sum S and partial sums s_N. What bound does the alternating series test give for |S − s_" +
			strconv.Itoa(n) + "|?",
		correct:     correct,
		distractors: distractorsFrom(correct, term(n), term(n+2), "1/"+strconv.Itoa(2*(n+1)), term(2*n)),
		reason: "For an alternating series whose terms decrease to 0, the error after N terms is at most the first omitted term, a_{N+1}; here that is " +
			correct + ".",
	}
}

// ---------------------------------------------------------------------------
// ra.seq.rearrangement (multiple choice)
// ---------------------------------------------------------------------------

func rearrangeGreedyClaim() mcClaim {
	tHund := pick([]int{50, 80, 100, 120, 130, 150, 160, 180, 200, 210})
	t := rq(int64(tHund), 100)
	sum := new(big.Rat)
	count := 0
	for sum.Cmp(t) <= 0 {
		count++
		sum = radd(sum, rq(1, int64(2*count-1)))
	}
	correct := strconv.Itoa(count)
	ts := hundredths(tHund)
	return mcClaim{
		q: "Riemann's rearrangement of the alternating harmonic series 1 − 1/2 + 1/3 − … aimed at " + ts +
			" starts by taking the positive terms 1, 1/3, 1/5, … until the partial sum first exceeds " + ts + ". How many positive terms does it take?",
		correct:     correct,
		distractors: distractorsFrom(correct, strconv.Itoa(count-1), strconv.Itoa(count+1), strconv.Itoa(count+2), strconv.Itoa(count+3)),
		reason: "Adding 1, 1/3, 1/5, … in turn, the running total first exceeds " + ts + " after " + correct + " terms, at " + decimal6(func() float64 { f, _ := sum.Float64(); return f }()) +
			". This works for any target because the positive terms alone diverge.",
	}
}

func rearrangeClassClaim() mcClaim {
	s := randAltSeries()
	for s.class == seriesDiv {
		s = randAltSeries()
	}
	same := "every rearrangement converges to the same sum"
	any := "some rearrangement converges to a different sum, and in fact to any real number chosen in advance"
	none := "no rearrangement converges"
	only := "rearrangements change the sum only when finitely many terms are moved"
	correct, why := same, " Absolute convergence makes the sum independent of the order of the terms."
	if s.class == seriesCond {
		correct, why = any, " By Riemann's rearrangement theorem a conditionally convergent series can be reordered to converge to any real number."
	}
	return mcClaim{
		q:           "What happens when the terms of " + s.desc + " are rearranged?",
		correct:     correct,
		distractors: distractorsFrom(correct, same, any, none, only),
		reason:      s.why + why,
	}
}

// ---------------------------------------------------------------------------
// ra.seq.cauchy_criterion (multiple choice)
// ---------------------------------------------------------------------------

func cauchyCritGeometricClaim() mcClaim {
	q := randIn(2, 5)
	d := randIn(1, 4)
	ten := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(d)), nil)
	// Tail after N terms of Σ_{k≥1} (1/q)^k is 1/(q^N (q − 1)); want it < 10^−d.
	n := 0
	pow := big.NewInt(1)
	for {
		v := new(big.Int).Mul(pow, big.NewInt(int64(q-1)))
		if v.Cmp(ten) > 0 {
			break
		}
		n++
		pow.Mul(pow, big.NewInt(int64(q)))
	}
	correct := strconv.Itoa(n)
	eps := decimal6(1 / float64(ten.Int64()))
	qs := strconv.Itoa(q)
	return mcClaim{
		q: "For Σ_{k=1}^∞ (1/" + qs + ")^k, the Cauchy criterion needs |s_m − s_n| < ε for m > n ≥ N, and |s_m − s_n| is at most the tail Σ_{k=N+1}^∞ (1/" +
			qs + ")^k. What is the smallest N for which that tail is less than " + eps + "?",
		correct:     correct,
		distractors: distractorsFrom(correct, strconv.Itoa(n-1), strconv.Itoa(n+1), strconv.Itoa(n+2), strconv.Itoa(n-2)),
		reason: "The tail is (1/" + qs + ")^(N+1)/(1 − 1/" + qs + ") = 1/(" + qs + "^N·" + strconv.Itoa(q-1) + "), which is below " + eps + " exactly when " + qs + "^N·" +
			strconv.Itoa(q-1) + " > " + ten.String() + "; the smallest such N is " + correct + ".",
	}
}

func cauchyCritHarmonicClaim() mcClaim {
	n := randIn(2, 7)
	block := func(lo, hi int) *big.Rat {
		s := new(big.Rat)
		for k := lo; k <= hi; k++ {
			s = radd(s, rq(1, int64(k)))
		}
		return s
	}
	want := block(n+1, 2*n)
	correct := ratS(want)
	ns := strconv.Itoa(n)
	return mcClaim{
		q:           "For the harmonic series, what is the block sum 1/(n + 1) + 1/(n + 2) + … + 1/(2n) when n = " + ns + "?",
		correct:     correct,
		distractors: distractorsFrom(correct, "1/2", ratS(block(n, 2*n)), ratS(block(n+1, 2*n+1)), ratS(rq(int64(n), int64(n+1)))),
		reason: "Adding the " + ns + " terms from 1/" + strconv.Itoa(n+1) + " to 1/" + strconv.Itoa(2*n) + " gives " + correct +
			". Each term is at least 1/(2n), so every such block is at least 1/2; the blocks never get small, so the harmonic series fails the Cauchy criterion and diverges.",
	}
}

func cauchyCritTermClaim() mcClaim {
	c := randIn(1, 9)
	cs := strconv.Itoa(c)
	bad := []string{
		"Σ n/(n + " + cs + ")",
		"Σ (−1)^n·" + cs,
		"Σ (1 + " + cs + "/n)",
		"Σ cos(1/n)",
	}
	good := []string{
		"Σ 1/n",
		"Σ " + cs + "/n²",
		"Σ (−1)^n/n",
		"Σ 1/(n + " + cs + ")",
		"Σ sin(1/n²)",
		"Σ (1/2)^n",
	}
	correct := pick(bad)
	perm := rand.Perm(len(good))
	return mcClaim{
		q:           "Which series diverges because its terms do not tend to 0?",
		correct:     correct,
		distractors: []string{good[perm[0]], good[perm[1]], good[perm[2]]},
		reason: "The terms of " + correct + " do not tend to 0, so a block of one term cannot be made small and the Cauchy criterion fails. " +
			"The terms of each of the others tend to 0, which is necessary for convergence but, as Σ 1/n shows, not sufficient.",
	}
}

// ---------------------------------------------------------------------------
// ra.seq.tests (multiple choice)
// ---------------------------------------------------------------------------

// powN prints n^p, or n when p = 1.
// powR is r^m, or r when m = 1.
func powR(m int) string {
	if m == 1 {
		return "r"
	}
	return "r^" + strconv.Itoa(m)
}

// powParen is (base)^p, or base when p = 1 ("((n + 1)/n)^1" is an artefact).
func powParen(base string, p int) string {
	if p == 1 {
		return base
	}
	return "(" + base + ")^" + strconv.Itoa(p)
}

func powN(p int) string {
	if p == 1 {
		return "n"
	}
	return "n^" + strconv.Itoa(p)
}

func ratioVerdict(l string, cmp int) string {
	switch {
	case cmp < 0:
		return "the ratio limit is " + l + " < 1, so the series converges absolutely"
	case cmp == 0:
		return "the ratio limit is 1, so the ratio test is inconclusive"
	}
	return "the ratio limit is " + l + " > 1, so the series diverges"
}

func testsRatioClaim() mcClaim {
	k := randIn(2, 9)
	p := randIn(1, 4)
	ks, ps := strconv.Itoa(k), strconv.Itoa(p)
	var series, correct, why string
	var others []string
	switch rand.Intn(5) {
	case 0:
		series = "Σ " + powN(p) + "/" + ks + "^n"
		correct = ratioVerdict("1/"+ks, -1)
		others = []string{ratioVerdict(ks, 1), ratioVerdict("1", 0), ratioVerdict("0", -1)}
		why = "a_{n+1}/a_n = " + powParen("(n + 1)/n", p) + "·(1/" + ks + "), which tends to 1/" + ks + "."
	case 1:
		series = "Σ " + ks + "^n/" + powN(p)
		correct = ratioVerdict(ks, 1)
		others = []string{ratioVerdict("1/"+ks, -1), ratioVerdict("1", 0), ratioVerdict("0", -1)}
		why = "a_{n+1}/a_n = " + ks + "·" + powParen("n/(n + 1)", p) + ", which tends to " + ks + "."
	case 2:
		series = "Σ " + ks + "^n/n!"
		correct = ratioVerdict("0", -1)
		others = []string{ratioVerdict(ks, 1), ratioVerdict("1", 0), ratioVerdict("1/"+ks, -1)}
		why = "a_{n+1}/a_n = " + ks + "/(n + 1), which tends to 0."
	case 3:
		series = "Σ 1/" + powN(p)
		correct = ratioVerdict("1", 0)
		others = []string{ratioVerdict("0", -1), ratioVerdict(ps, 1), ratioVerdict("1/"+ps, -1)}
		if p == 1 {
			others = []string{ratioVerdict("0", -1), ratioVerdict("2", 1), ratioVerdict("1/2", -1)}
		}
		why = "a_{n+1}/a_n = " + powParen("n/(n + 1)", p) + ", which tends to 1; the p-series test, not the ratio test, decides this series."
	default:
		a := randIn(1, 9)
		b := randIn(1, 9)
		for a == b {
			b = randIn(1, 9)
		}
		r := ratS(rq(int64(a), int64(b)))
		series = "Σ n·(" + strconv.Itoa(a) + "/" + strconv.Itoa(b) + ")^n"
		cmp := -1
		if a > b {
			cmp = 1
		}
		inv := ratS(rq(int64(b), int64(a)))
		correct = ratioVerdict(r, cmp)
		others = []string{ratioVerdict(inv, -cmp), ratioVerdict("1", 0), ratioVerdict("0", -1)}
		why = "a_{n+1}/a_n = ((n + 1)/n)·(" + strconv.Itoa(a) + "/" + strconv.Itoa(b) + "), which tends to " + r + "."
	}
	return mcClaim{
		q:           "Apply the ratio test to " + series + ". Which conclusion is correct?",
		correct:     correct,
		distractors: distractorsFrom(correct, others...),
		reason:      why,
	}
}

func testsComparisonClaim() mcClaim {
	type pw struct {
		s   string
		big bool
	}
	p := pick([]pw{{"1/2", false}, {"1", false}, {"3/2", true}, {"2", true}, {"3", true}})
	c := randIn(1, 9)
	den := "n^" + p.s
	if p.s == "1" {
		den = "n"
	} else if p.s == "1/2" || p.s == "3/2" {
		den = "n^(" + p.s + ")"
	}
	conv := "converges, by comparison with the convergent p-series Σ 1/" + den
	div := "diverges, by limit comparison with the divergent p-series Σ 1/" + den
	correct := div
	why := "The terms behave like 1/" + den + ": their ratio tends to 1, and Σ 1/" + den + " diverges because p ≤ 1."
	if p.big {
		correct = conv
		why = "0 < 1/(" + den + " + " + strconv.Itoa(c) + ") < 1/" + den + ", and Σ 1/" + den + " converges because p > 1."
	}
	return mcClaim{
		q:           "What does a comparison test say about Σ 1/(" + den + " + " + strconv.Itoa(c) + ")?",
		correct:     correct,
		distractors: distractorsFrom(correct, conv, div, "diverges, because its terms do not tend to 0", "converges, because its terms tend to 0"),
		reason:      why + " Terms tending to 0 is necessary but never sufficient on its own.",
	}
}

func testsRootClaim() mcClaim {
	a := randIn(1, 9)
	b := randIn(1, 9)
	for a == b {
		b = randIn(1, 9)
	}
	c := randIn(1, 5)
	r := ratS(rq(int64(a), int64(b)))
	root := func(l string, cmp int) string {
		if cmp < 0 {
			return "the n-th root of |a_n| tends to " + l + " < 1, so the series converges"
		}
		return "the n-th root of |a_n| tends to " + l + " > 1, so the series diverges"
	}
	cmp := -1
	if a > b {
		cmp = 1
	}
	correct := root(r, cmp)
	return mcClaim{
		q:       "Apply the root test to Σ (" + coefX(a, "n") + "/(" + coefX(b, "n") + " + " + strconv.Itoa(c) + "))^n. Which conclusion is correct?",
		correct: correct,
		distractors: distractorsFrom(correct, root(ratS(rq(int64(b), int64(a))), -cmp),
			"the n-th root of |a_n| tends to 1, so the root test is inconclusive",
			root("0", -1)),
		reason: "The n-th root of the n-th term is " + coefX(a, "n") + "/(" + coefX(b, "n") + " + " + strconv.Itoa(c) + "), which tends to " + r + ".",
	}
}
