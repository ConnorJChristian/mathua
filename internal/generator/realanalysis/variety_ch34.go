package realanalysis

// Concrete instances for Abbott chapters 3 and 4: compact, connected and
// nowhere dense sets named explicitly, ε–δ for actual functions, sequences
// that witness a missing limit, continuity of piecewise functions, roots
// located by the intermediate value theorem, extreme values on actual
// intervals, uniform continuity with a δ to check, and the discontinuities of
// actual functions.

import (
	"math/big"
	"math/rand"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/chuma-beep/mathua/internal/generator"
)

// oneOfFour picks one statement from yes and three from no.
func oneOfFour(q string, yes, no []string, reason func(correct string) string) mcClaim {
	correct := pick(yes)
	perm := rand.Perm(len(no))
	return mcClaim{
		q:           q,
		correct:     correct,
		distractors: []string{no[perm[0]], no[perm[1]], no[perm[2]]},
		reason:      upperFirst(reason(correct)),
	}
}

// upperFirst capitalises the first letter of a sentence that starts with a
// set or function name such as "the line y = 2x".
func upperFirst(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError || !unicode.IsLower(r) {
		return s
	}
	return string(unicode.ToUpper(r)) + s[n:]
}

// ---------------------------------------------------------------------------
// ra.limit.compact (multiple choice)
// ---------------------------------------------------------------------------

func compactSets() (yes, no []string) {
	a := randIn(-4, 4)
	b := a + randIn(1, 4)
	c := b + randIn(1, 3)
	d := c + randIn(1, 3)
	as, bs, cs, ds := itoa(a), itoa(b), itoa(c), itoa(d)
	yes = []string{
		"[" + as + ", " + bs + "]",
		"{" + as + " + 1/n : n ∈ N} ∪ {" + as + "}",
		"{" + as + ", " + bs + ", " + cs + "}",
		"[" + as + ", " + bs + "] ∪ [" + cs + ", " + ds + "]",
		"the Cantor set",
	}
	no = []string{
		"(" + as + ", " + bs + "]",
		"[" + as + ", ∞)",
		"{" + as + " + 1/n : n ∈ N}",
		"Q ∩ [" + as + ", " + bs + "]",
		"Z",
		"(" + as + ", " + bs + ")",
	}
	return
}

func compactWhichClaim() mcClaim {
	yes, no := compactSets()
	if rand.Intn(2) == 0 {
		return oneOfFour("Which of these subsets of R is compact?", yes, no, func(c string) string {
			return c + " is closed and bounded, so it is compact by Heine–Borel. Each of the others is unbounded or misses a limit point of itself, such as an endpoint, the point the sequence approaches, or an irrational in the interval."
		})
	}
	return oneOfFour("Which of these subsets of R is not compact?", no, yes, func(c string) string {
		return c + " is unbounded or misses one of its own limit points, so it is not closed and bounded and Heine–Borel says it is not compact. Each of the others is closed and bounded."
	})
}

func compactWitnessClaim() mcClaim {
	a := randIn(-4, 4)
	l := pick([]int{1, 2, 4})
	b := a + l
	as, bs, ls := itoa(a), itoa(b), strconv.Itoa(l)
	mid := ratS(rq(int64(2*a+l), 2))
	alt := "x_n = " + mid + " + (−1)^n·" + ratS(rq(int64(l), 4))
	if l == 4 {
		alt = "x_n = " + mid + " + (−1)^n"
	}
	correct := "x_n = " + as + " + " + ls + "/(n + 1)"
	return mcClaim{
		q:       "K = (" + as + ", " + bs + "]. Which sequence in K has no subsequence converging to a point of K, and so shows that K is not compact?",
		correct: correct,
		distractors: []string{
			"x_n = " + bs,
			"x_n = " + bs + " − " + ls + "/(2n)",
			alt,
		},
		reason: correct + " lies in K and converges to " + as + ", which is not in K, so every subsequence converges to " + as +
			" as well. Each of the other sequences has a subsequence converging to a point of K (" + bs + ", " + bs + ", or one of the two values it alternates between).",
	}
}

// ---------------------------------------------------------------------------
// ra.limit.connected (multiple choice)
// ---------------------------------------------------------------------------

func connectedSets() (yes, no []string) {
	a := randIn(-4, 3)
	b := a + randIn(1, 3)
	c := b + randIn(1, 3)
	d := c + randIn(1, 3)
	m := pick([]int{1, 4, 9})
	as, bs, cs, ds, ms := itoa(a), itoa(b), itoa(c), itoa(d), strconv.Itoa(m)
	yes = []string{
		"[" + as + ", " + bs + "] ∪ [" + bs + ", " + cs + "]",
		"(" + as + ", " + bs + "]",
		"{x : x² < " + ms + "}",
		"[" + as + ", ∞)",
		"(" + as + ", " + bs + ") ∪ [" + bs + ", " + cs + ")",
	}
	no = []string{
		"(" + as + ", " + bs + ") ∪ (" + bs + ", " + cs + ")",
		"[" + as + ", " + bs + "] ∪ [" + cs + ", " + ds + "]",
		"{x : x² > " + ms + "}",
		"R with the point " + bs + " removed",
		"Q",
		"[" + as + ", " + bs + "] ∪ {" + cs + "}",
	}
	return
}

func connectedWhichClaim() mcClaim {
	yes, no := connectedSets()
	if rand.Intn(2) == 0 {
		return oneOfFour("Which of these subsets of R is connected?", yes, no, func(c string) string {
			return c + " is an interval, and the connected subsets of R are exactly the intervals. Each of the others has a gap: a point between two of its elements that it does not contain."
		})
	}
	return oneOfFour("Which of these subsets of R is disconnected?", no, yes, func(c string) string {
		return c + " skips a point lying between two of its elements, so it is not an interval and splits into two separated pieces. Each of the others is an interval, hence connected."
	})
}

func connectedImageClaim() mcClaim {
	a := randIn(-4, 3)
	b := a + randIn(1, 4)
	c := b + randIn(1, 3)
	d := c + randIn(1, 3)
	as, bs, cs, ds := itoa(a), itoa(b), itoa(c), itoa(d)
	correct := "[" + as + ", " + bs + "]"
	return mcClaim{
		q:       "f is continuous on [0, 1]. Which of these sets could be the image f([0, 1])?",
		correct: correct,
		distractors: []string{
			"(" + as + ", " + bs + ")",
			"[" + as + ", " + bs + "] ∪ [" + cs + ", " + ds + "]",
			"[" + as + ", ∞)",
		},
		reason: "[0, 1] is compact and connected, and continuous functions preserve both, so the image is a closed bounded interval; " + correct +
			" is one (for example the image of a linear map). An open interval is not compact, a union with a gap is not connected, and an unbounded set is not compact.",
	}
}

// ---------------------------------------------------------------------------
// ra.metric.baire (multiple choice; Abbott 3.5)
// ---------------------------------------------------------------------------

func baireSets() (nowhere, dense []string) {
	a := randIn(-4, 4)
	b := a + randIn(1, 4)
	k := randIn(2, 5)
	as, bs, ks := itoa(a), itoa(b), strconv.Itoa(k)
	nowhere = []string{
		"Z",
		"{" + as + " + 1/n : n ∈ N}",
		"the Cantor set",
		"{" + as + ", " + bs + "}",
		"{" + ks + "n : n ∈ Z}",
	}
	dense = []string{
		"Q",
		"(" + as + ", " + bs + ")",
		"Q ∩ [" + as + ", " + bs + "]",
		"the irrational numbers",
		"[" + as + ", " + bs + "]",
	}
	return
}

func baireNowhereDenseClaim() mcClaim {
	nd, other := baireSets()
	if rand.Intn(2) == 0 {
		return oneOfFour("Which of these subsets of R is nowhere dense?", nd, other, func(c string) string {
			return "The closure of " + c + " contains no open interval, so it is nowhere dense. Each of the others has a closure containing an interval."
		})
	}
	return oneOfFour("Which of these subsets of R is not nowhere dense?", other, nd, func(c string) string {
		return "The closure of " + c + " contains an open interval, so it is not nowhere dense. Each of the others has a closure with empty interior."
	})
}

func baireCategoryClaim() mcClaim {
	a := randIn(-4, 4)
	b := a + randIn(1, 4)
	as, bs := itoa(a), itoa(b)
	first := []string{"Q", "Z", "the Cantor set", "Q ∩ [" + as + ", " + bs + "]", "the algebraic numbers"}
	second := []string{"R", "the irrational numbers", "[" + as + ", " + bs + "]", "(" + as + ", " + bs + ")"}
	if rand.Intn(2) == 0 {
		return oneOfFour("Which of these subsets of R is of first category, a countable union of nowhere dense sets?", first, second, func(c string) string {
			return c + " is a countable union of nowhere dense sets (countably many points, or a single nowhere dense set). Each of the others is not: by the Baire category theorem R is not of first category, so neither are the irrationals nor any interval."
		})
	}
	return oneOfFour("Which of these subsets of R is not of first category?", second, first, func(c string) string {
		return "By the Baire category theorem a complete metric space, and so any interval of R, is not a countable union of nowhere dense sets; if " + c +
			" were, then R (or an interval) would be too. Each of the others is a countable union of points or a single nowhere dense set."
	})
}

// ---------------------------------------------------------------------------
// ra.limit.function (numeric)
// ---------------------------------------------------------------------------

// functionLimitVariant: for f(x) = m·x + b at x0 the largest δ is ε/m. The
// facts keep the old names m, en and ed (ε = en/ed).
func functionLimitVariant() generator.Problem {
	m := randIn(2, 9)
	b := randNonzero(-6, 6)
	x0 := randIn(-3, 5)
	eps := pick([][2]int{{1, 10}, {1, 100}, {1, 2}, {1, 5}, {1, 20}, {3, 10}, {1, 1}, {2, 1}, {1, 4}})
	en, ed := eps[0], eps[1]
	answer := decimal6(float64(en) / float64(ed) / float64(m))
	es := ratDec(rq(int64(en), int64(ed)))
	fx := coefX(m, "x") + signedTerm(b)
	x0s := itoa(x0)
	return generator.Problem{
		Question: "f(x) = " + minus(fx) + " has limit f(" + x0s + ") at x = " + x0s + ". For ε = " + es + ", what is the largest δ such that |f(x) − f(" + x0s +
			")| < ε whenever |" + xPlus(-x0) + "| < δ? (enter a decimal)",
		Answer: answer,
		Explanation: "|f(x) − f(" + x0s + ")| = " + strconv.Itoa(m) + "|" + xPlus(-x0) + "|, so the condition is " + strconv.Itoa(m) + "|" + xPlus(-x0) + "| < " + es +
			", that is |" + xPlus(-x0) + "| < " + es + "/" + strconv.Itoa(m) + ". Any larger δ lets points through where the difference reaches ε, so the largest δ is " + answer + ".",
		Facts: map[string]string{
			"m": strconv.Itoa(m), "en": strconv.Itoa(en), "ed": strconv.Itoa(ed), "b": strconv.Itoa(b), "x0": strconv.Itoa(x0), "answer": answer,
		},
	}
}

// ---------------------------------------------------------------------------
// ra.limit.sequential (multiple choice)
// ---------------------------------------------------------------------------

func sequentialSinClaim() mcClaim {
	c := randIn(1, 9)
	cs := strconv.Itoa(c)
	type seq struct {
		name, arg, sin, val string
	}
	seqs := []seq{
		{"x_n = 1/(2nπ)", "2nπ", "0", "0"},
		{"x_n = 1/(2nπ + π/2)", "2nπ + π/2", "1", cs},
		{"x_n = 1/(2nπ + 3π/2)", "2nπ + 3π/2", "−1", itoa(-c)},
	}
	s := pick(seqs)
	return mcClaim{
		q:           "f(x) = " + cs + "·sin(1/x). The sequence " + s.name + " tends to 0. What is the limit of f(x_n)?",
		correct:     s.val,
		distractors: distractorsFrom(s.val, "0", cs, itoa(-c), "it does not exist"),
		reason: "1/x_n = " + s.arg + " and sin(" + s.arg + ") = " + s.sin + ", so f(x_n) is constantly " + s.val +
			". Different sequences tending to 0 give different limits, so by the sequential criterion f has no limit at 0.",
	}
}

func sequentialStepClaim() mcClaim {
	a := randIn(-5, 5)
	b := randIn(-5, 5)
	for b == a {
		b = randIn(-5, 5)
	}
	x0 := randIn(-3, 3)
	x0s := itoa(x0)
	pair := func(p, q int) string { return "f(x_n) → " + itoa(p) + " and f(y_n) → " + itoa(q) }
	correct := pair(a, b)
	return mcClaim{
		q: "f(x) = " + itoa(a) + " for x < " + x0s + " and f(x) = " + itoa(b) + " for x ≥ " + x0s + ". Take x_n = " + x0s + " − 1/n and y_n = " + x0s +
			" + 1/n, both tending to " + x0s + ". What happens to f(x_n) and f(y_n)?",
		correct:     correct,
		distractors: distractorsFrom(correct, pair(b, a), pair(a, a), pair(b, b)),
		reason: "Every x_n is below " + x0s + " and every y_n is at least " + x0s + ", so the image sequences are constant at " + itoa(a) + " and " + itoa(b) +
			". Two different limits along sequences tending to " + x0s + " show that the limit of f at " + x0s + " does not exist.",
	}
}

func sequentialDirichletClaim() mcClaim {
	c := randIn(-5, 5)
	d := randIn(-5, 5)
	for d == c {
		d = randIn(-5, 5)
	}
	x0 := randIn(-3, 3)
	pair := func(p, q int) string { return "f(r_n) → " + itoa(p) + " and f(t_n) → " + itoa(q) }
	correct := pair(c, d)
	return mcClaim{
		q: "f(x) = " + itoa(c) + " for rational x and f(x) = " + itoa(d) + " for irrational x. Let r_n be rationals and t_n irrationals, both tending to " + itoa(x0) +
			". What happens to f(r_n) and f(t_n)?",
		correct:     correct,
		distractors: distractorsFrom(correct, pair(d, c), pair(c, c), pair(d, d)),
		reason: "Both sequences exist by the density of Q and of the irrationals. The image sequences are constant at " + itoa(c) + " and " + itoa(d) +
			", so by the sequential criterion f has no limit at " + itoa(x0) + ", or at any other point.",
	}
}

func sequentialRemovableClaim() mcClaim {
	a := randNonzero(-5, 6)
	as := itoa(a)
	sq := strconv.Itoa(a * a)
	correct := itoa(2 * a)
	return mcClaim{
		q:           "f(x) = (x² − " + sq + ")/(" + xPlus(-a) + ") for x ≠ " + as + ". For x_n = " + as + " + 1/n, what is the limit of f(x_n)?",
		correct:     correct,
		distractors: distractorsFrom(correct, as, "0", strconv.Itoa(a*a), "it does not exist"),
		reason: "For x ≠ " + as + ", f(x) = " + xPlus(a) + ", so f(x_n) = " + correct + " + 1/n, which tends to " + correct +
			". The same happens along every sequence tending to " + as + ", so the limit of f at " + as + " is " + correct + ".",
	}
}

// xPlus prints x + c with the sign folded in: "x + 3", "x − 2", "x".
func xPlus(c int) string {
	switch {
	case c > 0:
		return "x + " + strconv.Itoa(c)
	case c < 0:
		return "x − " + strconv.Itoa(-c)
	}
	return "x"
}

// parenNeg prints a negative integer in parentheses, for substitution.
func parenNeg(a int) string {
	if a < 0 {
		return "(" + itoa(a) + ")"
	}
	return strconv.Itoa(a)
}

// ---------------------------------------------------------------------------
// ra.limit.continuity (multiple choice)
// ---------------------------------------------------------------------------

func continuityPiecewiseClaim() mcClaim {
	a := randNonzero(-4, 4)
	b := randIn(-5, 5)
	x0 := randNonzero(-3, 3)
	k := a*x0 + b - x0*x0
	correct := itoa(k)
	left := coefX(a, "x") + signedTerm(b)
	if b == 0 {
		left = coefX(a, "x")
	}
	x0s := itoa(x0)
	return mcClaim{
		q:           "f(x) = " + minus(left) + " for x < " + x0s + " and f(x) = x² + k for x ≥ " + x0s + ". Which value of k makes f continuous at " + x0s + "?",
		correct:     correct,
		distractors: distractorsFrom(correct, itoa(k+1), itoa(k-1), itoa(a*x0+b+x0*x0), itoa(a*x0+b)),
		reason: "The left limit at " + x0s + " is " + itoa(a*x0+b) + " and the right limit and value are " + strconv.Itoa(x0*x0) + " + k. Continuity needs them equal, so k = " +
			itoa(a*x0+b) + " − " + strconv.Itoa(x0*x0) + " = " + correct + ".",
	}
}

func continuityFillClaim() mcClaim {
	c := randNonzero(-5, 6)
	cs := itoa(c)
	correct := itoa(2 * c)
	return mcClaim{
		q:           "f(x) = (x² − " + strconv.Itoa(c*c) + ")/(" + xPlus(-c) + ") for x ≠ " + cs + ", and f(" + cs + ") = k. Which k makes f continuous at " + cs + "?",
		correct:     correct,
		distractors: distractorsFrom(correct, cs, "0", strconv.Itoa(c*c), itoa(-2*c)),
		reason:      "Away from " + cs + ", f(x) = " + xPlus(c) + ", which tends to " + correct + " at " + cs + ". Continuity requires the value to equal the limit, so k = " + correct + ".",
	}
}

func continuityWhichClaim() mcClaim {
	c := randIn(2, 9)
	cs := strconv.Itoa(c)
	yes := []string{
		"f(x) = " + cs + "x·sin(1/x) for x ≠ 0, f(0) = 0",
		"f(x) = " + cs + "x for rational x and 0 for irrational x",
		"f(x) = |" + cs + "x|",
		"f(x) = " + cs + "x² for x ≠ 0, f(0) = 0",
	}
	no := []string{
		"f(x) = sin(1/x) for x ≠ 0, f(0) = 0",
		"f(x) = " + cs + " for x ≥ 0 and 0 for x < 0",
		"f(x) = " + cs + " for rational x and 0 for irrational x",
		"f(x) = 1/x for x ≠ 0, f(0) = " + cs,
		"f(x) = (x² + " + cs + "x)/x for x ≠ 0, f(0) = 0",
	}
	return oneOfFour("Which of these functions is continuous at x = 0?", yes, no, func(cr string) string {
		return "For " + cr + ", |f(x) − f(0)| tends to 0 as x tends to 0 (it is squeezed by a multiple of |x|), so f is continuous at 0. Each of the others has no limit at 0, or a limit different from f(0)."
	})
}

// ---------------------------------------------------------------------------
// ra.limit.ivt (multiple choice)
// ---------------------------------------------------------------------------

func ivtCubicClaim() mcClaim {
	for {
		p := randIn(-6, 6)
		q := randIn(-9, 9)
		f := func(x int) int { return x*x*x + p*x + q }
		ok := true
		for x := -3; x <= 3; x++ {
			if f(x) == 0 {
				ok = false
			}
		}
		if !ok {
			continue
		}
		var change, same []int
		for k := -3; k <= 2; k++ {
			if (f(k) < 0) != (f(k+1) < 0) {
				change = append(change, k)
			} else {
				same = append(same, k)
			}
		}
		if len(change) == 0 || len(same) < 3 {
			continue
		}
		iv := func(k int) string { return "[" + itoa(k) + ", " + itoa(k+1) + "]" }
		k := pick(change)
		perm := rand.Perm(len(same))
		fx := "x³" + minus(func() string {
			s := ""
			if p != 0 {
				s += signedTerm(p) + "x"
			}
			if q != 0 {
				s += signedTerm(q)
			}
			return s
		}())
		return mcClaim{
			q:           "f(x) = " + fx + ". On which interval does the intermediate value theorem, applied to the values of f at the endpoints, guarantee a zero of f?",
			correct:     iv(k),
			distractors: []string{iv(same[perm[0]]), iv(same[perm[1]]), iv(same[perm[2]])},
			reason: "f(" + itoa(k) + ") = " + itoa(f(k)) + " and f(" + itoa(k+1) + ") = " + itoa(f(k+1)) +
				" have opposite signs, and f is continuous, so it takes the value 0 in between. On each of the other intervals the endpoint values have the same sign, so the theorem guarantees nothing there.",
		}
	}
}

func ivtValueClaim() mcClaim {
	a := randIn(-3, 3)
	b := a + randIn(1, 4)
	fa := randIn(-9, 9)
	fb := randIn(-9, 9)
	for absInt(fb-fa) < 3 {
		fb = randIn(-9, 9)
	}
	lo, hi := min(fa, fb), max(fa, fb)
	v := randIn(lo+1, hi-1)
	correct := itoa(v)
	return mcClaim{
		q: "f is continuous on [" + itoa(a) + ", " + itoa(b) + "] with f(" + itoa(a) + ") = " + itoa(fa) + " and f(" + itoa(b) + ") = " + itoa(fb) +
			". Which value must f take somewhere in the interval?",
		correct:     correct,
		distractors: distractorsFrom(correct, itoa(hi+1), itoa(lo-1), itoa(hi+3), itoa(lo-4)),
		reason:      correct + " lies between " + itoa(fa) + " and " + itoa(fb) + ", so the intermediate value theorem gives a point where f equals it. Values outside that range need not be taken.",
	}
}

func ivtBisectClaim() mcClaim {
	a := randIn(1, 5)
	b := a + 1
	m := randIn(a*a+1, b*b-1)
	// Two bisection steps on f(x) = x² − m over [a, a + 1].
	lo, hi := ri(int64(a)), ri(int64(b))
	mr := ri(int64(m))
	for step := 0; step < 2; step++ {
		mid := rquo(radd(lo, hi), ri(2))
		if rmul(mid, mid).Cmp(mr) < 0 {
			lo = mid
		} else {
			hi = mid
		}
	}
	iv := func(x, y *big.Rat) string { return "[" + ratDec(x) + ", " + ratDec(y) + "]" }
	quarters := []string{}
	for i := 0; i < 4; i++ {
		quarters = append(quarters, iv(radd(ri(int64(a)), rq(int64(i), 4)), radd(ri(int64(a)), rq(int64(i+1), 4))))
	}
	correct := iv(lo, hi)
	ms := strconv.Itoa(m)
	return mcClaim{
		q: "Bisection is applied to f(x) = x² − " + ms + " on [" + strconv.Itoa(a) + ", " + strconv.Itoa(b) +
			"], keeping each time the half on which f changes sign. Which interval remains after two steps?",
		correct:     correct,
		distractors: distractorsFrom(correct, quarters...),
		reason: "Each step evaluates f at the midpoint and keeps the half whose endpoint values have opposite signs. After two steps the interval is " + correct +
			", which contains √" + ms + " ≈ " + decimal6(sqrtF(m)) + ".",
	}
}

// ---------------------------------------------------------------------------
// ra.limit.evt (multiple choice)
// ---------------------------------------------------------------------------

func evtQuadraticClaim() mcClaim {
	h := randIn(-3, 4)
	k := randIn(-5, 5)
	a := randIn(-4, 3)
	b := a + randIn(2, 5)
	f := func(x int) int { return (x-h)*(x-h) + k }
	maxV := max(f(a), f(b))
	minV := min(f(a), f(b))
	vertexIn := a <= h && h <= b
	if vertexIn {
		minV = k
	}
	fx := "(x" + signedTerm(-h) + ")²" + signedTerm(k)
	if h == 0 {
		fx = "x²" + signedTerm(k)
	}
	if k == 0 {
		fx = fx[:len(fx)-len(signedTerm(k))]
	}
	fx = minus(fx)
	ivs := "[" + itoa(a) + ", " + itoa(b) + "]"
	if rand.Intn(2) == 0 {
		correct := itoa(maxV)
		return mcClaim{
			q:           "f(x) = " + fx + " is continuous on " + ivs + ", so it attains a maximum there. What is the maximum value?",
			correct:     correct,
			distractors: distractorsFrom(correct, itoa(minV), itoa(f(a)), itoa(f(b)), itoa(k), itoa(maxV+1), itoa(maxV-1), itoa(maxV+2)),
			reason: "A parabola opening upward is largest at the endpoint farther from its vertex x = " + itoa(h) + ". f(" + itoa(a) + ") = " + itoa(f(a)) +
				" and f(" + itoa(b) + ") = " + itoa(f(b)) + ", so the maximum is " + correct + ".",
		}
	}
	correct := itoa(minV)
	why := "The vertex x = " + itoa(h) + " lies in " + ivs + ", so the minimum is the vertex value " + correct + "."
	if !vertexIn {
		why = "The vertex x = " + itoa(h) + " lies outside " + ivs + ", so f is monotone on the interval and the minimum is at the nearer endpoint: " + correct + "."
	}
	return mcClaim{
		q:           "f(x) = " + fx + " is continuous on " + ivs + ", so it attains a minimum there. What is the minimum value?",
		correct:     correct,
		distractors: distractorsFrom(correct, itoa(maxV), itoa(k), itoa(f(a)), itoa(f(b)), itoa(minV-1), itoa(minV+1), itoa(minV-2)),
		reason:      why,
	}
}

func evtDomainClaim() mcClaim {
	a := randIn(1, 4)
	b := a + randIn(1, 5)
	as, bs := strconv.Itoa(a), strconv.Itoa(b)
	correct := "[" + as + ", " + bs + "]"
	return mcClaim{
		q:       "On which of these intervals must f(x) = 1/x attain both a maximum and a minimum?",
		correct: correct,
		distractors: []string{
			"(0, " + bs + "]",
			"[" + as + ", ∞)",
			"(" + as + ", " + bs + "]",
		},
		reason: "f is continuous on the closed bounded interval " + correct + ", so the extreme value theorem applies: the maximum is 1/" + as + " and the minimum is 1/" + bs +
			". On (0, " + bs + "] f is unbounded; on [" + as + ", ∞) its infimum 0 is not attained; on (" + as + ", " + bs + "] its supremum 1/" + as + " is not attained.",
	}
}

// ---------------------------------------------------------------------------
// ra.limit.uniform (multiple choice)
// ---------------------------------------------------------------------------

func uniformLinearDeltaClaim() mcClaim {
	m := randIn(2, 9)
	b := randIn(-5, 5)
	eps := pick([]*big.Rat{rq(1, 10), rq(1, 100), rq(1, 20), rq(1, 5), rq(1, 2)})
	es := ratDec(eps)
	d := func(r *big.Rat) string { return "δ = " + ratDec(r) }
	correct := d(rquo(eps, ri(int64(m))))
	fx := coefX(m, "x") + signedTerm(b)
	if b == 0 {
		fx = coefX(m, "x")
	}
	return mcClaim{
		q:       "f(x) = " + minus(fx) + " on R and ε = " + es + ". Which δ guarantees |f(x) − f(y)| < ε for every pair with |x − y| < δ?",
		correct: correct,
		distractors: distractorsFrom(correct, d(eps), d(rmul(eps, ri(int64(m)))), d(rquo(rmul(eps, ri(2)), ri(int64(m)))),
			d(rquo(ri(int64(m)), eps))),
		reason: "|f(x) − f(y)| = " + strconv.Itoa(m) + "|x − y|, so |x − y| < ε/" + strconv.Itoa(m) + " = " + ratDec(rquo(eps, ri(int64(m)))) +
			" works at every point at once, which is uniform continuity. Every larger δ fails: pairs with |x − y| just below it give |f(x) − f(y)| ≥ ε.",
	}
}

func uniformSquareDeltaClaim() mcClaim {
	a := randIn(1, 5)
	eps := pick([]*big.Rat{rq(1, 10), rq(1, 100), rq(1, 20)})
	es := ratDec(eps)
	as := strconv.Itoa(a)
	d := func(r *big.Rat) string { return "δ = " + ratDec(r) }
	correct := d(rquo(eps, ri(int64(2*a))))
	return mcClaim{
		q:       "f(x) = x² on [0, " + as + "] and ε = " + es + ". Which δ guarantees |x² − y²| < ε for all x, y in [0, " + as + "] with |x − y| < δ?",
		correct: correct,
		distractors: distractorsFrom(correct, d(rquo(eps, ri(int64(a)))), d(rquo(rmul(eps, ri(2)), ri(int64(a)))), d(eps),
			d(rmul(eps, ri(2))), d(rmul(eps, ri(3)))),
		reason: "|x² − y²| = |x − y|·(x + y) ≤ |x − y|·" + strconv.Itoa(2*a) + " on [0, " + as + "], so δ = ε/" + strconv.Itoa(2*a) +
			" works. Each larger choice fails near x = " + as + ": with y = " + as + " − t and t just below δ, |x² − y²| = t(" + strconv.Itoa(2*a) + " − t) exceeds ε.",
	}
}

func uniformWhichClaim() mcClaim {
	a := randIn(1, 5)
	as := strconv.Itoa(a)
	yes := []string{"x² on [0, " + as + "]", "√x on [0, ∞)", "sin x on R", "1/x on [" + as + ", ∞)", strconv.Itoa(a+1) + "x + 1 on R"}
	no := []string{"x² on R", "1/x on (0, " + as + ")", "sin(1/x) on (0, " + as + ")", "x³ on R", "1/x² on (0, 1)"}
	if rand.Intn(2) == 0 {
		return oneOfFour("Which of these functions is uniformly continuous on the set given?", yes, no, func(c string) string {
			return c + " is uniformly continuous: it is continuous on a compact set, or its slope is bounded, or (for √x) |√x − √y| ≤ √|x − y|. Each of the others has pairs of points arbitrarily close together whose images stay a fixed distance apart."
		})
	}
	return oneOfFour("Which of these functions is not uniformly continuous on the set given?", no, yes, func(c string) string {
		return c + " is not uniformly continuous: there are pairs x_n, y_n with |x_n − y_n| → 0 while |f(x_n) − f(y_n)| stays bounded away from 0. Each of the others is uniformly continuous."
	})
}

// ---------------------------------------------------------------------------
// ra.limit.discontinuity (multiple choice)
// ---------------------------------------------------------------------------

var discontinuityLabels = []string{
	"f is continuous there",
	"a removable discontinuity",
	"a jump discontinuity",
	"an essential discontinuity",
}

func discontinuityClassifyClaim() mcClaim {
	c := randNonzero(-4, 4)
	cs := itoa(c)
	var desc, why string
	var cls int
	switch rand.Intn(7) {
	case 0, 1:
		a := randNonzero(-3, 3)
		l := randIn(-6, 6)
		r := l
		e := l
		switch rand.Intn(3) {
		case 0:
			cls = 0
		case 1:
			cls = 1
			e = l + randNonzero(-3, 3)
		default:
			cls = 2
			r = l + randNonzero(-3, 3)
			e = pick([]int{l, r})
		}
		// left: a·x + (l − a·c); right: −a·x + (r + a·c).
		left := coefX(a, "x") + signedTerm(l-a*c)
		right := coefX(-a, "x") + signedTerm(r+a*c)
		desc = "f(x) = " + minus(left) + " for x < " + cs + ", f(" + cs + ") = " + itoa(e) + ", and f(x) = " + minus(right) + " for x > " + cs
		why = "The left limit is " + itoa(l) + ", the right limit is " + itoa(r) + " and the value is " + itoa(e) + "."
		switch cls {
		case 0:
			why += " All three agree, so f is continuous at " + cs + "."
		case 1:
			why += " The one-sided limits agree but differ from the value, so redefining f(" + cs + ") removes the discontinuity."
		default:
			why += " Both one-sided limits exist but differ, which is a jump."
		}
	case 2:
		k := randNonzero(-6, 6)
		desc = "f(x) = (x² − " + strconv.Itoa(c*c) + ")/(" + xPlus(-c) + ") for x ≠ " + cs + ", and f(" + cs + ") = " + itoa(k)
		if k == 2*c {
			cls = 0
			why = "Away from " + cs + ", f(x) = " + xPlus(c) + ", which tends to " + itoa(2*c) + ", and that equals f(" + cs + "), so f is continuous there."
		} else {
			cls = 1
			why = "Away from " + cs + ", f(x) = " + xPlus(c) + ", which tends to " + itoa(2*c) + ", but f(" + cs + ") = " + itoa(k) + ", so the discontinuity is removable."
		}
	case 3:
		desc = "f(x) = sin(1/(" + xPlus(-c) + ")) for x ≠ " + cs + ", and f(" + cs + ") = 0"
		cls = 3
		why = "As x approaches " + cs + " from either side, f oscillates between −1 and 1, so neither one-sided limit exists: the discontinuity is essential."
	case 4:
		desc = "f(x) = |" + xPlus(-c) + "|/(" + xPlus(-c) + ") for x ≠ " + cs + ", and f(" + cs + ") = 0"
		cls = 2
		why = "f is −1 to the left of " + cs + " and 1 to the right, so both one-sided limits exist and differ: a jump."
	case 5:
		desc = "f(x) = 1/(" + xPlus(-c) + ") for x ≠ " + cs + ", and f(" + cs + ") = 0"
		cls = 3
		why = "f is unbounded on both sides of " + cs + ", so neither one-sided limit exists as a real number: the discontinuity is essential."
	default:
		half := rand.Intn(2) == 0
		if half {
			desc = "f(x) = ⌊x⌋, the greatest integer at most x, at x = " + cs + " + 1/2"
			cls = 0
			why = "⌊x⌋ is constant on (" + cs + ", " + itoa(c+1) + "), so it is continuous at every non-integer point."
		} else {
			desc = "f(x) = ⌊x⌋, the greatest integer at most x, at x = " + cs
			cls = 2
			why = "⌊x⌋ tends to " + itoa(c-1) + " from the left and equals " + cs + " from the right, so both one-sided limits exist and differ: a jump."
		}
		return mcClaim{
			q:           "What kind of point is this for " + desc + "?",
			correct:     discontinuityLabels[cls],
			distractors: distractorsFrom(discontinuityLabels[cls], discontinuityLabels...),
			reason:      why,
		}
	}
	correct := discontinuityLabels[cls]
	return mcClaim{
		q:           desc + ". What happens at x = " + cs + "?",
		correct:     correct,
		distractors: distractorsFrom(correct, discontinuityLabels...),
		reason:      why,
	}
}

// ---------------------------------------------------------------------------
// ra.limit.monotone_fn (multiple choice)
// ---------------------------------------------------------------------------

func monotoneFloorCountClaim() mcClaim {
	k := randIn(1, 5)
	a := randIn(-3, 2)
	b := a + randIn(1, 4)
	if k == 1 && b == a+1 {
		b++
	}
	// Jumps of ⌊kx⌋ in the open interval (a, b): integers j with ka < j < kb.
	count := k*b - k*a - 1
	correct := strconv.Itoa(count)
	f := "⌊" + coefX(k, "x") + "⌋"
	return mcClaim{
		q:           "f(x) = " + f + " is increasing. How many points of discontinuity does it have in the open interval (" + itoa(a) + ", " + itoa(b) + ")?",
		correct:     correct,
		distractors: distractorsFrom(correct, strconv.Itoa(count+1), strconv.Itoa(count+2), strconv.Itoa(count-1), strconv.Itoa(b-a), strconv.Itoa(count+3)),
		reason: "f jumps exactly where " + coefX(k, "x") + " is an integer, at x = j/" + strconv.Itoa(k) + ". Inside (" + itoa(a) + ", " + itoa(b) + ") that needs " +
			itoa(k*a) + " < j < " + itoa(k*b) + ", which allows " + correct + " integers j. As for every monotone function, each discontinuity is a jump.",
	}
}

func monotoneJumpClaim() mcClaim {
	c := randIn(1, 6)
	n := randIn(-3, 5)
	cs, ns := strconv.Itoa(c), itoa(n)
	f := cs + "⌊x⌋ + x"
	left := c*(n-1) + n
	right := c*n + n
	if rand.Intn(2) == 0 {
		correct := itoa(left)
		return mcClaim{
			q:           "f(x) = " + f + " is increasing. What is the left-hand limit of f at x = " + ns + "?",
			correct:     correct,
			distractors: distractorsFrom(correct, itoa(right), itoa(left-1), itoa(c*n), itoa(left+c-1)),
			reason: "Just left of " + ns + ", ⌊x⌋ = " + itoa(n-1) + ", so f(x) = " + cs + "·" + parenNeg(n-1) + " + x, which tends to " + correct +
				". The right-hand limit is f(" + ns + ") = " + itoa(right) + ", so f has a jump of " + cs + " there.",
		}
	}
	correct := cs
	return mcClaim{
		q:           "f(x) = " + f + " is increasing. How large is its jump at x = " + ns + ", the right-hand limit minus the left-hand limit?",
		correct:     correct,
		distractors: distractorsFrom(correct, strconv.Itoa(c+1), "1", "0", strconv.Itoa(c*2)),
		reason: "The left-hand limit is " + cs + "·" + parenNeg(n-1) + " + " + parenNeg(n) + " = " + itoa(left) + " and the right-hand limit is " + cs + "·" + parenNeg(n) + " + " + parenNeg(n) +
			" = " + itoa(right) + ", so the jump is " + correct + ".",
	}
}

func monotoneTotalJumpClaim() mcClaim {
	a := randIn(-3, 3)
	b := a + randIn(1, 5)
	fa := randIn(-6, 4)
	fb := fa + randIn(2, 9)
	correct := itoa(fb - fa)
	return mcClaim{
		q: "f is increasing on [" + itoa(a) + ", " + itoa(b) + "] with f(" + itoa(a) + ") = " + itoa(fa) + " and f(" + itoa(b) + ") = " + itoa(fb) +
			". What is the largest possible total of the sizes of all its jumps in the open interval?",
		correct:     correct,
		distractors: distractorsFrom(correct, itoa(fb), itoa(fa+fb), itoa(fb-fa+1), "there is no bound, since there can be infinitely many jumps"),
		reason: "The jumps are disjoint gaps inside [f(" + itoa(a) + "), f(" + itoa(b) + ")], so their total is at most " + itoa(fb) + " − " + parenNeg(fa) + " = " + correct +
			"; a single step function reaches it. This is also why there are at most countably many jumps: only finitely many can exceed 1/n for each n.",
	}
}
