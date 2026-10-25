package realanalysis

// Concrete instances for Abbott chapter 1: suprema of explicit sets, the
// Archimedean property with numbers in it, nested intervals, the density proof
// run on actual endpoints, enumerations of Z and N × N, the diagonal argument
// on actual digits, and arithmetic in the extended reals.

import (
	"math"
	"math/big"
	"math/rand"
	"strconv"
	"strings"

	"github.com/chuma-beep/mathua/internal/generator"
)

// raSet is a concrete bounded set of reals with its supremum and infimum.
// sup and inf are nil when the bound is irrational; supS and infS always hold
// the display form.
type raSet struct {
	desc           string
	sup, inf       *big.Rat
	supS, infS     string
	supIn, infIn   bool
	supWhy, infWhy string
}

func boundText(in bool, word string) string {
	if in {
		return "it belongs to the set, so it is the " + word + "."
	}
	return "it does not belong to the set, so the set has no " + word + "."
}

// intervalSet is an interval with integer endpoints and random endpoint types.
func intervalSet() raSet {
	lo := randIn(-6, 4)
	hi := lo + randIn(1, 6)
	lc, hc := rand.Intn(2) == 0, rand.Intn(2) == 0
	left, right := " < x", " < "
	if lc {
		left = " ≤ x"
	}
	if hc {
		right = " ≤ "
	}
	s := raSet{
		desc:  "the set of all x with " + itoa(lo) + left + right + itoa(hi),
		sup:   ri(int64(hi)),
		inf:   ri(int64(lo)),
		supIn: hc, infIn: lc,
	}
	s.supS, s.infS = itoa(hi), itoa(lo)
	s.supWhy = "Every element is at most " + s.supS + ", and elements come as close to " + s.supS +
		" as we like, so no smaller number is an upper bound; " + boundText(hc, "maximum")
	s.infWhy = "Every element is at least " + s.infS + ", and elements come as close to " + s.infS +
		" as we like, so no larger number is a lower bound; " + boundText(lc, "minimum")
	return s
}

// sequenceSet is the set of values of a simple sequence indexed by n ∈ N.
func sequenceSet() raSet {
	c := randNonzero(-3, 5)
	k := randIn(1, 4)
	cs, ks := itoa(c), strconv.Itoa(k)
	var s raSet
	switch rand.Intn(4) {
	case 0: // c − k/n: increases to c.
		s.desc = "the set of all numbers " + cs + " − " + ks + "/n, where n runs over the positive integers"
		s.sup, s.inf = ri(int64(c)), ri(int64(c-k))
		s.supIn, s.infIn = false, true
		s.supS, s.infS = ratS(s.sup), ratS(s.inf)
		s.supWhy = "Every term is below " + s.supS + ", and by the Archimedean property " + ks + "/n is smaller than any ε > 0 for large n, so nothing below " +
			s.supS + " is an upper bound; " + boundText(false, "maximum")
		s.infWhy = "The terms increase with n, so the smallest is the first term, " + s.infS + "; " + boundText(true, "minimum")
	case 1: // c + k/n: decreases to c.
		s.desc = "the set of all numbers " + cs + " + " + ks + "/n, where n runs over the positive integers"
		s.sup, s.inf = ri(int64(c+k)), ri(int64(c))
		s.supIn, s.infIn = true, false
		s.supS, s.infS = ratS(s.sup), ratS(s.inf)
		s.supWhy = "The terms decrease with n, so the largest is the first term, " + s.supS + "; " + boundText(true, "maximum")
		s.infWhy = "Every term is above " + s.infS + ", and by the Archimedean property " + ks + "/n is smaller than any ε > 0 for large n, so nothing above " +
			s.infS + " is a lower bound; " + boundText(false, "minimum")
	case 2: // c + (−1)^n k/n: largest at n = 2, smallest at n = 1.
		s.desc = "the set of all numbers " + cs + " + (−1)^n·" + ks + "/n, where n runs over the positive integers"
		s.sup = radd(ri(int64(c)), rq(int64(k), 2))
		s.inf = ri(int64(c - k))
		s.supIn, s.infIn = true, true
		s.supS, s.infS = ratDec(s.sup), ratDec(s.inf)
		s.supWhy = "Odd n give terms below " + cs + ", and the even terms " + cs + " + " + ks + "/n decrease, so the largest term is at n = 2, namely " +
			s.supS + "; " + boundText(true, "maximum")
		s.infWhy = "Even n give terms above " + cs + ", and the odd terms " + cs + " − " + ks + "/n increase, so the smallest term is at n = 1, namely " +
			s.infS + "; " + boundText(true, "minimum")
	default: // kn/(n + 1) = k − k/(n + 1): increases to k from k/2.
		k = randIn(1, 6)
		ks = strconv.Itoa(k)
		s.desc = "the set of all numbers " + coefX(k, "n") + "/(n + 1), where n runs over the positive integers"
		s.sup, s.inf = ri(int64(k)), rq(int64(k), 2)
		s.supIn, s.infIn = false, true
		s.supS, s.infS = ratDec(s.sup), ratDec(s.inf)
		s.supWhy = "Each term equals " + ks + " − " + ks + "/(n + 1), which is below " + ks + " and comes arbitrarily close to it, so the supremum is " +
			ks + "; " + boundText(false, "maximum")
		s.infWhy = "The terms increase with n, so the smallest is the first term, " + s.infS + "; " + boundText(true, "minimum")
	}
	return s
}

// squareSet is {x : x² < s²} or {x : x² ≤ s²}.
func squareSet() raSet {
	r := randIn(2, 6)
	m := r * r
	closed := rand.Intn(2) == 0
	rel := " < "
	if closed {
		rel = " ≤ "
	}
	s := raSet{
		desc:  "the set of all real x with x²" + rel + strconv.Itoa(m),
		sup:   ri(int64(r)),
		inf:   ri(int64(-r)),
		supIn: closed, infIn: closed,
	}
	s.supS, s.infS = itoa(r), itoa(-r)
	s.supWhy = "x²" + rel + strconv.Itoa(m) + " describes the interval from " + s.infS + " to " + s.supS + ", so the supremum is " + s.supS + "; " + boundText(closed, "maximum")
	s.infWhy = "x²" + rel + strconv.Itoa(m) + " describes the interval from " + s.infS + " to " + s.supS + ", so the infimum is " + s.infS + "; " + boundText(closed, "minimum")
	return s
}

// rootSet is {r ∈ Q : r² < m} for a non-square m, whose bounds are irrational.
func rootSet() raSet {
	m := pick([]int{2, 3, 5, 6, 7, 10})
	ms := strconv.Itoa(m)
	s := raSet{
		desc: "the set of all rational r with r² < " + ms,
		supS: "√" + ms, infS: "−√" + ms,
	}
	s.supWhy = "√" + ms + " is an upper bound, and by the density of Q there are rationals below √" + ms +
		" as close to it as we like, so the least upper bound is √" + ms + "; it is irrational, so it is not in the set and the set has no maximum."
	s.infWhy = "−√" + ms + " is a lower bound, and by the density of Q there are rationals above −√" + ms +
		" as close to it as we like, so the greatest lower bound is −√" + ms + "; it is irrational, so it is not in the set and the set has no minimum."
	return s
}

// ---------------------------------------------------------------------------
// ra.order.supremum (numeric)
// ---------------------------------------------------------------------------

// supremumVariant asks for the supremum or the infimum of a concrete set with
// rational bounds. The facts lo and hi are the infimum and the supremum as
// decimals, and ask says which one is the answer.
func supremumVariant() generator.Problem {
	var s raSet
	switch rand.Intn(3) {
	case 0:
		s = intervalSet()
	case 1:
		s = sequenceSet()
	default:
		s = squareSet()
	}
	ask, word, val, why := "sup", "supremum", s.sup, s.supWhy
	if rand.Intn(2) == 0 {
		ask, word, val, why = "inf", "infimum", s.inf, s.infWhy
	}
	answer := ratAnswer(val)
	return generator.Problem{
		Question:    "What is the " + word + " of " + s.desc + "? (enter a decimal)",
		Answer:      answer,
		Explanation: why,
		Facts: map[string]string{
			"lo": ratAnswer(s.inf), "hi": ratAnswer(s.sup), "ask": ask, "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// ra.order.completeness (multiple choice)
// ---------------------------------------------------------------------------

// completenessBoundClaim asks which statement about sup A (or inf A) is true
// for a concrete set: the value, and whether it is attained.
func completenessBoundClaim() mcClaim {
	var s raSet
	switch rand.Intn(4) {
	case 0:
		s = intervalSet()
	case 1:
		s = sequenceSet()
	case 2:
		s = squareSet()
	default:
		s = rootSet()
	}
	op, ext, val, other, in, why := "sup", "maximum", s.supS, s.infS, s.supIn, s.supWhy
	if rand.Intn(2) == 0 {
		op, ext, val, other, in, why = "inf", "minimum", s.infS, s.supS, s.infIn, s.infWhy
	}
	attained := func(v string, yes bool) string {
		if yes {
			return op + " A = " + v + ", and " + v + " belongs to A, so it is the " + ext + " of A"
		}
		return op + " A = " + v + ", but " + v + " does not belong to A, so A has no " + ext
	}
	correct := attained(val, in)
	return mcClaim{
		q:       "Let A be " + s.desc + ". Which statement about " + op + " A is correct?",
		correct: correct,
		distractors: distractorsFrom(correct,
			attained(val, !in),
			attained(other, in),
			op+" A does not exist, because A has no "+ext,
			attained(other, !in)),
		reason: why,
	}
}

// completenessApproxClaim is Abbott's Lemma 1.3.8 with numbers: if s = sup A
// then for every ε > 0 some element of A exceeds s − ε.
func completenessApproxClaim() mcClaim {
	s := randIn(-5, 9)
	eps := pick([]*big.Rat{rq(1, 10), rq(1, 2), rq(1, 100), rq(1, 4), ri(1), rq(1, 1000)})
	ss := itoa(s)
	cut := ratDec(rsub(ri(int64(s)), eps))
	es := ratDec(eps)
	correct := "some element a of A satisfies a > " + cut
	return mcClaim{
		q:       "A is a nonempty set with sup A = " + ss + ". Taking ε = " + es + ", which statement must be true?",
		correct: correct,
		distractors: []string{
			"every element a of A satisfies a > " + cut,
			"some element a of A satisfies a > " + ss,
			ss + " belongs to A",
		},
		reason: "Because " + ss + " is the least upper bound, " + cut + " = " + ss + " − " + es +
			" is not an upper bound, so some element of A exceeds it. Nothing forces every element above it, no element exceeds the supremum, and the supremum need not belong to A.",
	}
}

// completenessScaleClaim asks for sup(cA) or sup(c + A) from sup A and inf A.
func completenessScaleClaim() mcClaim {
	t := randIn(-5, 3)
	s := t + randIn(1, 6)
	c := randNonzero(-4, 4)
	ts, ss, cs := itoa(t), itoa(s), itoa(c)
	if rand.Intn(3) == 0 {
		correct := itoa(s + c)
		return mcClaim{
			q:           "A is bounded with sup A = " + ss + " and inf A = " + ts + ". For c + A = {c + a : a ∈ A} with c = " + cs + ", what is sup(c + A)?",
			correct:     correct,
			distractors: distractorsFrom(correct, itoa(t+c), ss, itoa(s*c), itoa(s-c)),
			reason:      "Adding c shifts every element, and every upper bound, by c, so sup(c + A) = c + sup A = " + correct + ".",
		}
	}
	var want int
	var why string
	if c > 0 {
		want = c * s
		why = "Multiplying by a positive c preserves order, so upper bounds of A scale to upper bounds of cA and sup(cA) = c·sup A = " + itoa(want) + "."
	} else {
		want = c * t
		why = "Multiplying by a negative c reverses order, so lower bounds of A become upper bounds of cA and the greatest lower bound becomes the least upper bound: sup(cA) = c·inf A = " + itoa(want) + "."
	}
	correct := itoa(want)
	return mcClaim{
		q:           "A is bounded with sup A = " + ss + " and inf A = " + ts + ". For cA = {c·a : a ∈ A} with c = " + cs + ", what is sup(cA)?",
		correct:     correct,
		distractors: distractorsFrom(correct, itoa(c*s), itoa(c*t), ss, itoa(s+c), itoa(-c*s)),
		reason:      why,
	}
}

// ---------------------------------------------------------------------------
// ra.order.archimedean (numeric)
// ---------------------------------------------------------------------------

// archimedeanVariant finds the least positive integer n with n·y > x, for
// y = pn/pd and x = xn/xd. With x = 1 this is the least N with 1/N < ε.
// The answer is floor(x/y) + 1, published through the facts.
func archimedeanVariant() generator.Problem {
	var pn, pd, xn, xd int
	// ε or y: a fraction or a short decimal.
	if rand.Intn(2) == 0 {
		pd = randIn(3, 60)
		pn = randIn(1, min(5, pd-1))
	} else {
		pd = pick([]int{10, 100, 1000})
		pn = randIn(1, 9)
	}
	if g := gcd(pn, pd); g > 1 {
		pn, pd = pn/g, pd/g
	}
	ys := strconv.Itoa(pn) + "/" + strconv.Itoa(pd)
	if pd == 10 || pd == 100 || pd == 1000 {
		ys = decimal6(float64(pn) / float64(pd))
	}
	var q string
	switch rand.Intn(3) {
	case 0:
		xn, xd = 1, 1
		q = "By the Archimedean property there is a positive integer N with 1/N < ε. For ε = " + ys + ", what is the smallest such N? (enter a whole number)"
	case 1:
		xn, xd = randIn(2, 9), 1
		q = "For the sequence a_n = " + strconv.Itoa(xn) + "/n, the Archimedean property gives an N with " + strconv.Itoa(xn) +
			"/N < ε. For ε = " + ys + ", what is the smallest such N? (enter a whole number)"
	default:
		xd = randIn(1, 4)
		xn = randIn(xd+1, 12*xd)
		if g := gcd(xn, xd); g > 1 {
			xn, xd = xn/g, xd/g
		}
		xs := strconv.Itoa(xn)
		if xd != 1 {
			xs += "/" + strconv.Itoa(xd)
		}
		q = "The Archimedean property says that for x = " + xs + " and y = " + ys +
			" there is a positive integer n with n·y > x. What is the smallest such n? (enter a whole number)"
	}
	// n·y > x  ⇔  n > x/y = xn·pd/(xd·pn).
	n := (xn*pd)/(xd*pn) + 1
	answer := strconv.Itoa(n)
	return generator.Problem{
		Question: q,
		Answer:   answer,
		Explanation: "The condition is equivalent to n > " + ratS(rq(int64(xn*pd), int64(xd*pn))) +
			", so the smallest integer that works is the floor of that ratio plus one, which is " + answer + ".",
		Facts: map[string]string{
			"pn": strconv.Itoa(pn), "pd": strconv.Itoa(pd), "xn": strconv.Itoa(xn), "xd": strconv.Itoa(xd), "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// ra.order.nested_intervals (multiple choice)
// ---------------------------------------------------------------------------

func nestedIntersectionClaim() mcClaim {
	a := randNonzero(-4, 5)
	b := a + randIn(1, 4)
	k := randIn(1, 3)
	as, bs, ks := itoa(a), itoa(b), strconv.Itoa(k)
	empty := "the empty set"
	point := "{" + as + "}"
	var interval, correct, why string
	var others []string
	switch rand.Intn(7) {
	case 0:
		interval = "[" + as + " − " + ks + "/n, " + as + " + " + ks + "/n]"
		correct = point
		others = []string{empty, "[" + itoa(a-k) + ", " + itoa(a+k) + "]", "(" + itoa(a-k) + ", " + itoa(a+k) + ")"}
		why = "Every I_n contains " + as + ", and any other point x is excluded once " + ks + "/n < |x − " + as + "|, so the intersection is {" + as + "}."
	case 1:
		interval = "[" + as + ", " + as + " + " + ks + "/n]"
		correct = point
		others = []string{empty, "[" + as + ", " + itoa(a+k) + "]", "{" + itoa(a+k) + "}"}
		why = as + " lies in every I_n, and any x > " + as + " is excluded once " + ks + "/n < x − " + as + ", so the intersection is {" + as + "}."
	case 2:
		interval = "(" + as + ", " + as + " + " + ks + "/n)"
		correct = empty
		others = []string{point, "(" + as + ", " + itoa(a+k) + ")", "{" + itoa(a+k) + "}"}
		why = as + " is in no I_n because the intervals are open at " + as + ", and any x > " + as +
			" is excluded once " + ks + "/n < x − " + as + ". The intervals are not closed, so the nested interval property does not apply."
	case 3:
		interval = "(" + as + " − " + ks + "/n, " + as + " + " + ks + "/n)"
		correct = point
		others = []string{empty, "(" + itoa(a-k) + ", " + itoa(a+k) + ")", "[" + itoa(a-k) + ", " + itoa(a+k) + "]"}
		why = "These intervals are open, but each still contains " + as + ", and any other x is excluded for large n, so the intersection is {" + as + "}. Openness only removes the guarantee; it does not force emptiness."
	case 4:
		interval = "[" + as + ", " + bs + " + " + ks + "/n]"
		correct = "[" + as + ", " + bs + "]"
		others = []string{"[" + as + ", " + bs + ")", point, empty}
		why = "Every point of [" + as + ", " + bs + "] lies in every I_n, and any x > " + bs + " is excluded once " + ks + "/n < x − " + bs + ", so the intersection is [" + as + ", " + bs + "]."
	case 5:
		interval = "(" + as + ", " + bs + " + " + ks + "/n)"
		correct = "(" + as + ", " + bs + "]"
		others = []string{"(" + as + ", " + bs + ")", "[" + as + ", " + bs + "]", empty}
		why = bs + " lies in every I_n because " + bs + " < " + bs + " + " + ks + "/n, every x > " + bs + " is eventually excluded, and " + as + " is never included, so the intersection is (" + as + ", " + bs + "]."
	default:
		interval = "[" + as + " + n, ∞)"
		correct = empty
		others = []string{"[" + itoa(a+1) + ", ∞)", "{" + itoa(a+1) + "}", "the whole real line"}
		why = "These intervals are closed and nested but unbounded: any real x is excluded once n > x − " + as + ", so the intersection is empty. Boundedness is a hypothesis of the nested interval property."
	}
	return mcClaim{
		q:           "For each positive integer n let I_n = " + interval + ". What is the intersection of all the I_n?",
		correct:     correct,
		distractors: distractorsFrom(correct, others...),
		reason:      why,
	}
}

func nestedBisectionClaim() mcClaim {
	a := randIn(-3, 3)
	l := randIn(1, 6)
	b := a + l
	k := randIn(2, 8)
	pow := int64(1) << k
	if rand.Intn(2) == 0 {
		want := rq(int64(l), pow)
		correct := ratS(want)
		return mcClaim{
			q: "Start with I_1 = [" + itoa(a) + ", " + itoa(b) + "] and form each next interval by keeping one closed half of the previous one. After " +
				strconv.Itoa(k) + " halvings, what is the length of the interval?",
			correct: correct,
			distractors: distractorsFrom(correct,
				ratS(rq(int64(l), pow/2)), ratS(rq(int64(l), pow*2)), ratS(rq(int64(l), int64(2*k))), ratS(rq(int64(l), int64(k)))),
			reason: "Each halving divides the length by 2, so after " + strconv.Itoa(k) + " halvings the length is " + strconv.Itoa(l) + "/2^" + strconv.Itoa(k) + " = " + correct + ".",
		}
	}
	d := randIn(1, 3)
	ten := 1
	for i := 0; i < d; i++ {
		ten *= 10
	}
	// Smallest k with l/2^k < 1/ten, that is 2^k > l·ten.
	need := 0
	for (1 << need) <= l*ten {
		need++
	}
	correct := strconv.Itoa(need)
	eps := decimal6(1 / float64(ten))
	return mcClaim{
		q: "Start with I_1 = [" + itoa(a) + ", " + itoa(b) + "] and repeatedly keep one closed half. What is the smallest number of halvings after which the interval has length less than " +
			eps + "?",
		correct:     correct,
		distractors: distractorsFrom(correct, strconv.Itoa(need-1), strconv.Itoa(need+1), strconv.Itoa(need+2)),
		reason: "After k halvings the length is " + strconv.Itoa(l) + "/2^k, which is below " + eps + " exactly when 2^k > " + strconv.Itoa(l*ten) +
			"; the smallest such k is " + correct + ", since 2^" + strconv.Itoa(need-1) + " = " + strconv.Itoa(1<<(need-1)) + " is not large enough.",
	}
}

// ---------------------------------------------------------------------------
// ra.order.density (multiple choice)
// ---------------------------------------------------------------------------

// densityEndpoints returns a < b as hundredths, a = A/100 and b = B/100.
func densityEndpoints() (int, int) {
	A := randIn(5, 395)
	B := A + randIn(2, 40)
	return A, B
}

func hundredths(n int) string { return decimal6(float64(n) / 100) }

// densityProofClaim runs Abbott's proof of Theorem 1.4.3 on actual endpoints:
// the least n with 1/n < b − a, then the integer m with m − 1 ≤ na < m.
func densityProofClaim() mcClaim {
	A, B := densityEndpoints()
	g := B - A
	n := 100/g + 1     // least n with 1/n < g/100, i.e. n·g > 100
	m := (n*A)/100 + 1 // m − 1 ≤ n·A/100 < m
	frac := func(p int) string { return strconv.Itoa(p) + "/" + strconv.Itoa(n) }
	correct := frac(m)
	return mcClaim{
		q: "Run the proof of the density of Q with a = " + hundredths(A) + " and b = " + hundredths(B) +
			": choose the smallest n with 1/n < b − a, then the integer m with m − 1 ≤ na < m. Which rational m/n does the proof produce?",
		correct: correct,
		distractors: distractorsFrom(correct, frac(m-1), frac(m+1), strconv.Itoa(m)+"/"+strconv.Itoa(n+1),
			frac(m+2)),
		reason: "b − a = " + hundredths(g) + ", so the smallest n with 1/n < " + hundredths(g) + " is n = " + strconv.Itoa(n) +
			". Then na = " + decimal6(float64(n*A)/100) + ", so m = " + strconv.Itoa(m) + ", and a < " + correct + " < b because 1/n < b − a.",
	}
}

// densityIrrationalClaim asks for an irrational strictly between a and b.
func densityIrrationalClaim() mcClaim {
	A, B := densityEndpoints()
	g := float64(B-A) / 100
	n := int(math.Floor(math.Sqrt2/g)) + 1 // least n with √2/n < b − a
	as, bs, ns := hundredths(A), hundredths(B), strconv.Itoa(n)
	correct := as + " + √2/" + ns
	mid := decimal6(float64(A+B) / 200)
	return mcClaim{
		q:       "Which of these is an irrational number strictly between " + as + " and " + bs + "?",
		correct: correct,
		distractors: distractorsFrom(correct,
			mid,
			as+" − √2/"+ns,
			bs+" + √2/"+ns),
		reason: "√2/" + ns + " ≈ " + decimal6(math.Sqrt2/float64(n)) + " is positive and less than b − a = " + decimal6(g) +
			", so " + correct + " lies strictly between the endpoints, and a rational plus a nonzero rational multiple of √2 is irrational. " +
			mid + " lies between them but is rational, and the other two lie outside the interval.",
	}
}

// densityCountClaim counts the fractions m/n strictly between a and b.
func densityCountClaim() mcClaim {
	A, B := densityEndpoints()
	n := randIn(3, 15)
	count := 0
	for m := (A*n)/100 - 1; m*100 <= B*n+100; m++ {
		if 100*m > A*n && 100*m < B*n {
			count++
		}
	}
	correct := strconv.Itoa(count)
	return mcClaim{
		q:           "How many fractions m/" + strconv.Itoa(n) + " with m an integer lie strictly between " + hundredths(A) + " and " + hundredths(B) + "?",
		correct:     correct,
		distractors: distractorsFrom(correct, strconv.Itoa(count+1), strconv.Itoa(count+2), strconv.Itoa(count-1), strconv.Itoa(count+3)),
		reason: "The condition " + hundredths(A) + " < m/" + strconv.Itoa(n) + " < " + hundredths(B) + " is " + decimal6(float64(A*n)/100) +
			" < m < " + decimal6(float64(B*n)/100) + ", and the number of integers m in that range is " + correct + ".",
	}
}

// ---------------------------------------------------------------------------
// ra.order.countable and ra.order.uncountable (multiple choice)
// ---------------------------------------------------------------------------

var raCountableSets = []string{
	"the integers Z",
	"the even integers",
	"the rational numbers Q",
	"N × N, the pairs of positive integers",
	"the rationals in [0, 1]",
	"the finite subsets of N",
	"the polynomials with integer coefficients",
	"the algebraic numbers",
	"the finite strings of 0s and 1s",
	"the prime numbers",
	"Q × Q",
	"the numbers k/2^n with k an integer and n a positive integer",
}

var raUncountableSets = []string{
	"the real numbers R",
	"the irrational numbers",
	"the open interval (0, 1)",
	"the set of all subsets of N",
	"the infinite sequences of 0s and 1s",
	"the Cantor set",
	"the transcendental numbers",
	"the interval [0, 0.001]",
	"R × R",
}

// countabilityClaim picks one set of the asked kind and three of the other.
func countabilityClaim(wantCountable bool) mcClaim {
	yes, no := raCountableSets, raUncountableSets
	word := "countable"
	why := "is countable: it can be listed as a sequence (a countable union of countable sets, or a subset of one), while each of the others contains a copy of an interval or of the infinite 0-1 sequences and so is uncountable."
	if !wantCountable {
		yes, no = no, yes
		word = "uncountable"
		why = "is uncountable: it contains a copy of an interval or of the infinite 0-1 sequences, where Cantor's diagonal argument applies, while each of the others can be listed as a sequence."
	}
	correct := pick(yes)
	perm := rand.Perm(len(no))
	return mcClaim{
		q:           "Which of these sets is " + word + "?",
		correct:     correct,
		distractors: []string{no[perm[0]], no[perm[1]], no[perm[2]]},
		reason:      "Of the four, only " + correct + " " + why,
	}
}

// zAtPosition is the k-th term of the listing 0, 1, −1, 2, −2, ...
func zAtPosition(k int) int {
	if k%2 == 0 {
		return k / 2
	}
	return -(k - 1) / 2
}

func countableZListingClaim() mcClaim {
	listing := "0, 1, −1, 2, −2, 3, −3, …"
	if rand.Intn(2) == 0 {
		k := randIn(6, 60)
		z := zAtPosition(k)
		correct := itoa(z)
		return mcClaim{
			q:           "Z is listed as " + listing + " Which integer is in position " + strconv.Itoa(k) + " of this list?",
			correct:     correct,
			distractors: distractorsFrom(correct, itoa(-z), itoa(zAtPosition(k+1)), itoa(zAtPosition(k-1)), itoa(z+1), itoa(z-1), itoa(-z+1)),
			reason:      "Even positions 2j hold j and odd positions 2j + 1 hold −j, so position " + strconv.Itoa(k) + " holds " + correct + ".",
		}
	}
	z := randNonzero(-30, 30)
	pos := 2 * z
	if z < 0 {
		pos = -2*z + 1
	}
	correct := strconv.Itoa(pos)
	return mcClaim{
		q:           "Z is listed as " + listing + " In which position does " + itoa(z) + " appear?",
		correct:     correct,
		distractors: distractorsFrom(correct, strconv.Itoa(pos+1), strconv.Itoa(pos-1), strconv.Itoa(absInt(z)), strconv.Itoa(2*absInt(z)+2)),
		reason:      "A positive j sits in position 2j and −j sits in position 2j + 1, so " + itoa(z) + " is in position " + correct + ".",
	}
}

func gcd(a, b int) int {
	a, b = absInt(a), absInt(b)
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// pairAt is the k-th pair (m, n) when N × N is listed along the diagonals
// m + n = 2, 3, 4, … with m increasing within a diagonal.
func pairAt(k int) (int, int) {
	pos := 0
	for s := 2; ; s++ {
		for m := 1; m < s; m++ {
			pos++
			if pos == k {
				return m, s - m
			}
		}
	}
}

func countablePairClaim() mcClaim {
	k := randIn(5, 45)
	m, n := pairAt(k)
	ps := func(a, b int) string { return "(" + strconv.Itoa(a) + ", " + strconv.Itoa(b) + ")" }
	correct := ps(m, n)
	m1, n1 := pairAt(k + 1)
	m0, n0 := pairAt(k - 1)
	return mcClaim{
		q: "N × N is listed along its diagonals: first the pairs with m + n = 2, then m + n = 3, and so on, with m increasing inside each diagonal: (1, 1), (1, 2), (2, 1), (1, 3), (2, 2), (3, 1), … Which pair is in position " +
			strconv.Itoa(k) + "?",
		correct:     correct,
		distractors: distractorsFrom(correct, ps(n, m), ps(m1, n1), ps(m0, n0), ps(m+1, n)),
		reason: "The diagonal m + n = s holds s − 1 pairs, so the first diagonals fill positions 1, 2–3, 4–6, 7–10 and so on; counting along gives " +
			correct + " in position " + strconv.Itoa(k) + ".",
	}
}

func uncountableDiagonalClaim() mcClaim {
	// The rule uses two digits other than 0 and 9, as in Abbott, so that x has
	// a single decimal expansion.
	p := randIn(1, 8)
	q := randIn(1, 7)
	if q >= p {
		q++
	}
	var rows [4][4]int
	for i := range rows {
		for j := range rows[i] {
			rows[i][j] = rand.Intn(10)
		}
		// Make the special digit appear on the diagonal often enough that the
		// answer is not always pppp.
		if rand.Intn(5) < 2 {
			rows[i][i] = p
		}
	}
	var lines []string
	for i, r := range rows {
		lines = append(lines, "f("+strconv.Itoa(i+1)+") = 0."+strconv.Itoa(r[0])+strconv.Itoa(r[1])+strconv.Itoa(r[2])+strconv.Itoa(r[3])+"…")
	}
	digits := func(f func(i int) int) string {
		s := "0."
		for i := 0; i < 4; i++ {
			s += strconv.Itoa(f(i))
		}
		return s + "…"
	}
	rule := func(d, a, b int) int {
		if d == a {
			return b
		}
		return a
	}
	ps, qs := strconv.Itoa(p), strconv.Itoa(q)
	correct := digits(func(i int) int { return rule(rows[i][i], p, q) })
	return mcClaim{
		q: "A list of reals begins " + strings.Join(lines, "; ") +
			". The diagonal argument builds x whose n-th digit is " + ps + " if the n-th digit of f(n) is not " + ps + ", and " + qs +
			" if it is " + ps + ". What are the first four digits of x?",
		correct: correct,
		distractors: distractorsFrom(correct,
			digits(func(i int) int { return rows[i][i] }),
			digits(func(i int) int { return rule(rows[0][i], p, q) }),
			digits(func(i int) int { return rule(rows[i][i], q, p) }),
			digits(func(i int) int { return rule(rows[i][3-i], p, q) }),
			digits(func(i int) int { return (rows[i][i] + 1) % 10 }),
		),
		reason: "Read the diagonal digits " + strconv.Itoa(rows[0][0]) + ", " + strconv.Itoa(rows[1][1]) + ", " + strconv.Itoa(rows[2][2]) + ", " + strconv.Itoa(rows[3][3]) +
			" and replace each by " + ps + ", or by " + qs + " where the digit is " + ps + ". Then x differs from f(n) in the n-th digit, so x is not on the list.",
	}
}

func uncountablePowerSetClaim() mcClaim {
	k := randIn(3, 10)
	want := 1 << k
	correct := strconv.Itoa(want)
	fact := 1
	for i := 2; i <= k; i++ {
		fact *= i
	}
	return mcClaim{
		q:           "A set A has " + strconv.Itoa(k) + " elements. How many elements does its power set P(A) have?",
		correct:     correct,
		distractors: distractorsFrom(correct, strconv.Itoa(k*k), strconv.Itoa(2*k), strconv.Itoa(fact), strconv.Itoa(want/2)),
		reason: "Each subset is a yes-or-no choice for every element, so there are 2^" + strconv.Itoa(k) + " = " + correct +
			" subsets. Cantor's theorem extends the point: P(A) is always strictly larger than A, which for A = N gives an uncountable set.",
	}
}

// ---------------------------------------------------------------------------
// ra.order.extended (multiple choice)
// ---------------------------------------------------------------------------

func extendedArithmeticClaim() mcClaim {
	c := randNonzero(-9, 9)
	cs := itoa(c)
	if c < 0 {
		cs = "(" + cs + ")"
	}
	opts := []string{"+∞", "−∞", "0", "undefined"}
	var expr, correct, why string
	switch rand.Intn(5) {
	case 0:
		expr, correct, why = cs+" + (+∞)", "+∞", "Adding a finite number to +∞ gives +∞."
	case 1:
		expr, correct, why = cs+" − (+∞)", "−∞", "Subtracting +∞ from a finite number gives −∞."
	case 2:
		expr = cs + " · (+∞)"
		if c > 0 {
			correct, why = "+∞", "A positive number times +∞ is +∞."
		} else {
			correct, why = "−∞", "A negative number times +∞ is −∞; the sign follows the usual rule."
		}
	case 3:
		expr = cs + " · (−∞)"
		if c > 0 {
			correct, why = "−∞", "A positive number times −∞ is −∞."
		} else {
			correct, why = "+∞", "A negative number times −∞ is +∞; the sign follows the usual rule."
		}
	default:
		expr, correct, why = cs+" / (+∞)", "0", "A finite number divided by an infinite one is 0 by convention."
	}
	return mcClaim{
		q:           "In the extended real numbers, what is " + expr + "?",
		correct:     correct,
		distractors: distractorsFrom(correct, opts...),
		reason:      why + " Only forms such as ∞ − ∞ and 0 · ∞ are left undefined.",
	}
}

func extendedSupInfClaim() mcClaim {
	c := randNonzero(-5, 5)
	cs := itoa(c)
	plus := func(k int) string { return itoa(c + k) }
	pair := func(s, i string) string { return "sup = " + s + ", inf = " + i }
	var desc, correct, why string
	var others []string
	switch rand.Intn(4) {
	case 0:
		desc = "{n + " + cs + " : n ∈ N}"
		if c < 0 {
			desc = "{n − " + itoa(-c) + " : n ∈ N}"
		}
		correct = pair("+∞", plus(1))
		others = []string{pair(plus(1), "−∞"), pair("+∞", cs), pair("+∞", "−∞")}
		why = "The set is unbounded above, so its supremum is +∞; its smallest element is at n = 1."
	case 1:
		desc = "{" + cs + " − n² : n ∈ N}"
		correct = pair(plus(-1), "−∞")
		others = []string{pair("+∞", plus(-1)), pair(cs, "−∞"), pair("+∞", "−∞")}
		why = "The largest element is at n = 1, and the set is unbounded below, so its infimum is −∞."
	case 2:
		desc = "{(−1)^n·n + " + cs + " : n ∈ N}"
		correct = pair("+∞", "−∞")
		others = []string{pair(plus(1), plus(-1)), pair("+∞", plus(-1)), pair(plus(2), "−∞")}
		why = "Even n give arbitrarily large elements and odd n arbitrarily negative ones, so the set is unbounded in both directions."
	default:
		desc = "{" + cs + " + 1/n : n ∈ N}"
		correct = pair(plus(1), cs)
		others = []string{pair("+∞", cs), pair(plus(1), "−∞"), pair(cs, plus(1))}
		why = "The set is bounded: its largest element is at n = 1 and its infimum " + cs + " is approached but not attained."
	}
	return mcClaim{
		q:           "In the extended real numbers, what are the supremum and infimum of " + desc + "?",
		correct:     correct,
		distractors: distractorsFrom(correct, others...),
		reason:      why,
	}
}
