package discrete

// Set algebra: rosters, set-builder notation, complements, differences,
// compound expressions, Cartesian products, power sets and counting.
//
// Difficulty chooses the shape of the question, not only the size of the
// numbers: one operation on two rosters, then complements, differences and
// set-builder rosters, then expressions with two operations, three sets or a
// Cartesian product. Every set is drawn at random, so the stems do not repeat.

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"

	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
)

// texSet shows a roster in LaTeX: braces must be escaped there, or they are
// read as grouping and disappear.
func texSet(s []int) string {
	if len(s) == 0 {
		return "\\varnothing"
	}
	return "\\{" + intsString(s, ", ") + "\\}"
}

// plainSet is the stored answer, which the set grader reads: {1,2,3} or ∅.
func plainSet(s []int) string { return formatSet(s) }

// randSubset draws k distinct elements of u, sorted.
func randSubset(u []int, k int) []int {
	p := rand.Perm(len(u))
	out := make([]int, 0, k)
	for _, i := range p[:k] {
		out = append(out, u[i])
	}
	sort.Ints(out)
	return out
}

func rangeInts(lo, hi int) []int {
	var out []int
	for v := lo; v <= hi; v++ {
		out = append(out, v)
	}
	return out
}

// twoSets draws A and B from u that overlap in at least one element and
// differ, so union, intersection and difference are all informative.
func twoSets(u []int, lo, hi int) ([]int, []int) {
	for {
		a := randSubset(u, lo+rand.Intn(hi-lo+1))
		b := randSubset(u, lo+rand.Intn(hi-lo+1))
		if len(intersection(a, b)) > 0 && len(difference(a, b)) > 0 && len(difference(b, a)) > 0 {
			return a, b
		}
	}
}

func setsAnswer(q string, ans []int, exp string, facts map[string]string) generator.Problem {
	return generator.Problem{Question: q, Answer: plainSet(ans), Explanation: exp, Facts: facts}
}

func (g *setOpsGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	switch {
	case ctx.Difficulty < 0.45:
		return setOpsBasic()
	case ctx.Difficulty < 0.7:
		if rand.Intn(3) == 0 {
			return setBuilderRoster()
		}
		return setOpsComplement()
	default:
		switch rand.Intn(5) {
		case 0:
			return setOpsThree()
		case 1:
			return cartesianRoster()
		case 2:
			return powerSetRoster()
		default:
			return setOpsCompound()
		}
	}
}

// Grade reads the answer as a set: order, repeats and spacing do not matter,
// and {} or ∅ is the empty set.
func (g *setOpsGen) Grade(expected, answer string) grader.Result {
	return grader.GradeSet(expected, answer)
}

func (g *setOpsGen) AnswerFormat() generator.AnswerFormat {
	return generator.AnswerFormat{Hint: "List the elements in braces, e.g. {1, 2, 3}; write ∅ or {} for the empty set", InputMode: "text"}
}

func setOpsBasic() generator.Problem {
	u := rangeInts(1, 9)
	a, b := twoSets(u, 3, 5)
	if rand.Intn(2) == 0 {
		r := union(a, b)
		return setsAnswer(
			fmt.Sprintf("Let \\(A = %s\\) and \\(B = %s\\). List the elements of \\(A \\cup B\\).", texSet(a), texSet(b)), r,
			fmt.Sprintf("The union collects every element that is in A or in B (or both), each listed once: A ∪ B = %s.", plainSet(r)),
			map[string]string{"op": "union"})
	}
	r := intersection(a, b)
	return setsAnswer(
		fmt.Sprintf("Let \\(A = %s\\) and \\(B = %s\\). List the elements of \\(A \\cap B\\).", texSet(a), texSet(b)), r,
		fmt.Sprintf("The intersection keeps only the elements in both A and B: A ∩ B = %s.", plainSet(r)),
		map[string]string{"op": "intersection"})
}

func setOpsComplement() generator.Problem {
	n := 8 + rand.Intn(5)
	u := rangeInts(1, n)
	a, b := twoSets(u, 3, 6)
	uTex := fmt.Sprintf("U = \\{1, 2, \\ldots, %d\\}", n)
	switch rand.Intn(4) {
	case 0:
		r := complement(a, u)
		return setsAnswer(
			fmt.Sprintf("The universal set is \\(%s\\) and \\(A = %s\\). List the elements of the complement \\(\\bar{A}\\).", uTex, texSet(a)), r,
			fmt.Sprintf("The complement is everything in U that is not in A: Ā = %s.", plainSet(r)),
			map[string]string{"op": "complement"})
	case 1:
		r := difference(a, b)
		return setsAnswer(
			fmt.Sprintf("Let \\(A = %s\\) and \\(B = %s\\). List the elements of \\(A - B\\).", texSet(a), texSet(b)), r,
			fmt.Sprintf("A − B keeps the elements of A that are not in B. Removing %s from A leaves %s.", plainSet(intersection(a, b)), plainSet(r)),
			map[string]string{"op": "difference"})
	case 2:
		r := difference(b, a)
		return setsAnswer(
			fmt.Sprintf("Let \\(A = %s\\) and \\(B = %s\\). List the elements of \\(B - A\\).", texSet(a), texSet(b)), r,
			fmt.Sprintf("B − A keeps the elements of B that are not in A. Removing %s from B leaves %s. Note that B − A is not A − B = %s.", plainSet(intersection(a, b)), plainSet(r), plainSet(difference(a, b))),
			map[string]string{"op": "difference"})
	default:
		ub := union(a, b)
		r := complement(ub, u)
		return setsAnswer(
			fmt.Sprintf("The universal set is \\(%s\\), \\(A = %s\\) and \\(B = %s\\). List the elements of \\(\\overline{A \\cup B}\\).", uTex, texSet(a), texSet(b)), r,
			fmt.Sprintf("First A ∪ B = %s. Its complement is what U has left: %s. (Equivalently Ā ∩ B̄, by De Morgan's law.)", plainSet(ub), plainSet(r)),
			map[string]string{"op": "complement_union"})
	}
}

func setBuilderRoster() generator.Problem {
	switch rand.Intn(4) {
	case 0:
		a := rand.Intn(6) - 3
		b := a + 3 + rand.Intn(3)
		r := rangeInts(a, b-1)
		return setsAnswer(
			fmt.Sprintf("List the elements of \\(\\{x \\in \\mathbb{Z} : %d \\le x < %d\\}\\).", a, b), r,
			fmt.Sprintf("The integers from %d up to but not including %d: %s.", a, b, plainSet(r)),
			map[string]string{"op": "builder"})
	case 1:
		k := 10 + rand.Intn(30)
		var r []int
		for x := 1; x*x < k; x++ {
			r = append(r, x)
		}
		return setsAnswer(
			fmt.Sprintf("Let \\(\\mathbb{Z}_+ = \\{1, 2, 3, \\ldots\\}\\). List the elements of \\(\\{x \\in \\mathbb{Z}_+ : x^2 < %d\\}\\).", k), r,
			fmt.Sprintf("Square the positive integers until the square reaches %d: the last one with x² < %d is %d, so the set is %s.", k, k, r[len(r)-1], plainSet(r)),
			map[string]string{"op": "builder"})
	case 2:
		a := rand.Intn(5) - 6
		b := a + 6 + rand.Intn(5)
		var r []int
		for x := a + 1; x <= b; x++ {
			if x%2 == 0 {
				r = append(r, x)
			}
		}
		return setsAnswer(
			fmt.Sprintf("List the elements of \\(\\{x \\in \\mathbb{Z} : %d < x \\le %d \\text{ and } x \\text{ is even}\\}\\).", a, b), r,
			fmt.Sprintf("The even integers strictly above %d and at most %d: %s.", a, b, plainSet(r)),
			map[string]string{"op": "builder"})
	default:
		c := 1 + rand.Intn(9)
		r := []int{-c, c}
		if rand.Intn(3) == 0 {
			return setsAnswer(
				fmt.Sprintf("List the elements of \\(\\{x \\in \\mathbb{R} : x^2 = -%d\\}\\).", c*c), nil,
				fmt.Sprintf("No real number has a negative square, so the set is empty: ∅. (Writing {0} would be wrong: 0² = 0 ≠ −%d.)", c*c),
				map[string]string{"op": "builder"})
		}
		return setsAnswer(
			fmt.Sprintf("List the elements of \\(\\{x \\in \\mathbb{R} : x^2 = %d\\}\\).", c*c), r,
			fmt.Sprintf("x² = %d has the two real roots ±%d, so the set is %s.", c*c, c, plainSet(r)),
			map[string]string{"op": "builder"})
	}
}

func setOpsCompound() generator.Problem {
	n := 8 + rand.Intn(3)
	u := rangeInts(1, n)
	a, b := twoSets(u, 3, 5)
	uTex := fmt.Sprintf("U = \\{1, 2, \\ldots, %d\\}", n)
	switch rand.Intn(3) {
	case 0:
		ac := complement(a, u)
		r := intersection(ac, b)
		return setsAnswer(
			fmt.Sprintf("The universal set is \\(%s\\), \\(A = %s\\) and \\(B = %s\\). List the elements of \\(\\bar{A} \\cap B\\).", uTex, texSet(a), texSet(b)), r,
			fmt.Sprintf("Ā = %s. Intersecting with B keeps the elements of B outside A: %s. This is the same set as B − A.", plainSet(ac), plainSet(r)),
			map[string]string{"op": "compound"})
	case 1:
		r := union(difference(a, b), difference(b, a))
		return setsAnswer(
			fmt.Sprintf("Let \\(A = %s\\) and \\(B = %s\\). List the elements of \\((A - B) \\cup (B - A)\\).", texSet(a), texSet(b)), r,
			fmt.Sprintf("A − B = %s and B − A = %s; their union is the elements in exactly one of the sets: %s.", plainSet(difference(a, b)), plainSet(difference(b, a)), plainSet(r)),
			map[string]string{"op": "compound"})
	default:
		ib := intersection(a, b)
		r := complement(ib, u)
		return setsAnswer(
			fmt.Sprintf("The universal set is \\(%s\\), \\(A = %s\\) and \\(B = %s\\). List the elements of \\(\\overline{A \\cap B}\\).", uTex, texSet(a), texSet(b)), r,
			fmt.Sprintf("A ∩ B = %s, and its complement is the rest of U: %s. By De Morgan's law this is also Ā ∪ B̄.", plainSet(ib), plainSet(r)),
			map[string]string{"op": "compound"})
	}
}

func setOpsThree() generator.Problem {
	u := rangeInts(1, 9)
	a, b := twoSets(u, 3, 5)
	c := randSubset(u, 3+rand.Intn(2))
	if rand.Intn(2) == 0 {
		bc := union(b, c)
		r := intersection(a, bc)
		return setsAnswer(
			fmt.Sprintf("Let \\(A = %s\\), \\(B = %s\\) and \\(C = %s\\). List the elements of \\(A \\cap (B \\cup C)\\).", texSet(a), texSet(b), texSet(c)), r,
			fmt.Sprintf("Brackets first: B ∪ C = %s. Then keep the elements also in A: %s.", plainSet(bc), plainSet(r)),
			map[string]string{"op": "three"})
	}
	bc := intersection(b, c)
	r := union(a, bc)
	return setsAnswer(
		fmt.Sprintf("Let \\(A = %s\\), \\(B = %s\\) and \\(C = %s\\). List the elements of \\(A \\cup (B \\cap C)\\).", texSet(a), texSet(b), texSet(c)), r,
		fmt.Sprintf("Brackets first: B ∩ C = %s. Then add the elements of A: %s.", plainSet(bc), plainSet(r)),
		map[string]string{"op": "three"})
}

// cartesianRoster asks for A × B; the answer is a set of ordered pairs.
func cartesianRoster() generator.Problem {
	a := randSubset(rangeInts(1, 5), 2)
	letters := []string{"p", "q", "r", "s", "t"}
	i := rand.Intn(len(letters) - 1)
	b := []string{letters[i], letters[i+1]}
	if rand.Intn(3) == 0 {
		b = append(b, letters[(i+2)%len(letters)])
	}
	var pairs []string
	for _, x := range a {
		for _, y := range b {
			pairs = append(pairs, fmt.Sprintf("(%d,%s)", x, y))
		}
	}
	return generator.Problem{
		Question: fmt.Sprintf("Let \\(A = %s\\) and \\(B = \\{%s\\}\\). List the elements of the Cartesian product \\(A \\times B\\) as ordered pairs.",
			texSet(a), strings.Join(b, ", ")),
		Answer: "{" + strings.Join(pairs, ",") + "}",
		Explanation: fmt.Sprintf("Pair each element of A, in turn, with each element of B: %s. There are %d × %d = %d pairs; order inside a pair matters, so (%d, %s) is in A × B but (%s, %d) is not.",
			"{"+strings.Join(pairs, ", ")+"}", len(a), len(b), len(a)*len(b), a[0], b[0], b[0], a[0]),
		Facts: map[string]string{"op": "cartesian"},
	}
}

// powerSetRoster asks for P(X) of a two-element set; the answer is a set of sets.
func powerSetRoster() generator.Problem {
	el := randSubset(rangeInts(1, 9), 2)
	x, y := el[0], el[1]
	return generator.Problem{
		Question: fmt.Sprintf("List the elements of the power set \\(\\mathcal{P}(X)\\) for \\(X = %s\\).", texSet(el)),
		Answer:   fmt.Sprintf("{∅,{%d},{%d},{%d,%d}}", x, y, x, y),
		Explanation: fmt.Sprintf("The subsets of X are ∅, {%d}, {%d} and X itself, so P(X) = {∅, {%d}, {%d}, {%d, %d}}: 2² = 4 elements, each of them a set.",
			x, y, x, y, x, y),
		Facts: map[string]string{"op": "powerset"},
	}
}

// ----- counting with two sets -----

func (g *vennGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	both := 2 + rand.Intn(8)
	onlyA := 3 + rand.Intn(10)
	onlyB := 3 + rand.Intn(10)
	none := 2 + rand.Intn(10)
	a, b := onlyA+both, onlyB+both
	un := onlyA + onlyB + both
	total := un + none
	frames := []struct{ who, aName, bName string }{
		{"households", "own a car", "own a bicycle"},
		{"firms", "export", "import"},
		{"workers", "hold a degree", "work part time"},
		{"consumers", "bought good X", "bought good Y"},
	}
	f := frames[rand.Intn(len(frames))]
	intro := fmt.Sprintf("In a survey of %d %s, %d %s (set A), %d %s (set B) and %d do both.", total, f.who, a, f.aName, b, f.bName, both)
	tier := 0
	switch {
	case ctx.Difficulty >= 0.7:
		tier = 2
	case ctx.Difficulty >= 0.45:
		tier = 1
	}
	var q string
	var ans int
	var exp string
	switch tier {
	case 0:
		if rand.Intn(2) == 0 {
			q, ans = intro+" How many are in \\(A \\cup B\\)?", un
			exp = fmt.Sprintf("Adding |A| and |B| counts the %d in both twice, so subtract it once: |A ∪ B| = %d + %d − %d = %d.", both, a, b, both, un)
		} else {
			q, ans = intro+" How many are in \\(A - B\\) (in A but not in B)?", onlyA
			exp = fmt.Sprintf("|A − B| = |A| − |A ∩ B| = %d − %d = %d.", a, both, onlyA)
		}
	case 1:
		if rand.Intn(2) == 0 {
			q, ans = intro+" How many are in neither set, \\(\\overline{A \\cup B}\\)?", none
			exp = fmt.Sprintf("|A ∪ B| = %d + %d − %d = %d, so the complement has %d − %d = %d.", a, b, both, un, total, un, none)
		} else {
			q, ans = intro+" How many are in exactly one of the two sets?", onlyA+onlyB
			exp = fmt.Sprintf("Only A: %d − %d = %d. Only B: %d − %d = %d. Exactly one: %d + %d = %d.", a, both, onlyA, b, both, onlyB, onlyA, onlyB, onlyA+onlyB)
		}
	default:
		// Recover the overlap from the union: the reverse of the formula.
		q = fmt.Sprintf("In a survey of %d %s, %d %s (set A), %d %s (set B) and %d do neither. How many are in \\(A \\cap B\\)?", total, f.who, a, f.aName, b, f.bName, none)
		ans = both
		exp = fmt.Sprintf("|A ∪ B| = %d − %d = %d. From |A ∪ B| = |A| + |B| − |A ∩ B|: |A ∩ B| = %d + %d − %d = %d.", total, none, un, a, b, un, both)
	}
	return generator.Problem{Question: q, Answer: strconv.Itoa(ans), Explanation: exp}
}

// ----- power sets -----

func (g *powersetGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	p := powersetProblem(ctx)
	p.Facts = map[string]string{"value": p.Answer}
	return p
}

func powersetProblem(ctx generator.GeneratorContext) generator.Problem {
	switch {
	case ctx.Difficulty < 0.45:
		n := 2 + rand.Intn(6)
		if rand.Intn(2) == 0 {
			return generator.Problem{
				Question:    fmt.Sprintf("A set \\(X\\) has %d elements. How many elements does its power set \\(\\mathcal{P}(X)\\) have?", n),
				Answer:      strconv.Itoa(1 << n),
				Explanation: fmt.Sprintf("Each of the %d elements is either in a subset or not, independently, so there are 2^%d = %d subsets. The count includes ∅ and X itself.", n, n, 1<<n),
			}
		}
		return generator.Problem{
			Question:    fmt.Sprintf("A set \\(X\\) has %d elements. How many proper subsets does it have (subsets other than \\(X\\) itself)?", n),
			Answer:      strconv.Itoa(1<<n - 1),
			Explanation: fmt.Sprintf("X has 2^%d = %d subsets in all. Leaving out X itself gives %d proper subsets (∅ is one of them).", n, 1<<n, 1<<n-1),
		}
	case ctx.Difficulty < 0.7:
		if rand.Intn(2) == 0 {
			n := 4 + rand.Intn(5)
			k := 1 + rand.Intn(n-1)
			c := binom(n, k)
			return generator.Problem{
				Question:    fmt.Sprintf("A set has %d elements. How many of its subsets have exactly %d %s?", n, k, plural(k, "element", "elements")),
				Answer:      strconv.Itoa(c),
				Explanation: fmt.Sprintf("A %d-element subset is a choice of %d of the %d elements, order ignored: C(%d, %d) = %d. Summing C(%d, k) over every k gives the whole power set, 2^%d = %d.", k, k, n, n, k, c, n, n, 1<<n),
			}
		}
		m := 2 + rand.Intn(4)
		n := 2 + rand.Intn(4)
		return generator.Problem{
			Question:    fmt.Sprintf("\\(A\\) has %d elements and \\(B\\) has %d. How many elements does the Cartesian product \\(A \\times B\\) have?", m, n),
			Answer:      strconv.Itoa(m * n),
			Explanation: fmt.Sprintf("Each of the %d elements of A pairs with each of the %d elements of B: %d × %d = %d ordered pairs.", m, n, m, n, m*n),
		}
	default:
		n := 3 + rand.Intn(6)
		k := 1 + rand.Intn(n-1)
		c := 1 << (n - k)
		return generator.Problem{
			Question: fmt.Sprintf("\\(X\\) has %d elements and \\(S \\subseteq X\\) has %d. How many subsets of \\(X\\) contain \\(S\\)?", n, k),
			Answer:   strconv.Itoa(c),
			Explanation: fmt.Sprintf("A subset containing S is S together with any subset of the other %d − %d = %d elements, so there are 2^%d = %d of them.",
				n, k, n-k, n-k, c),
		}
	}
}

func binom(n, k int) int {
	c := 1
	for i := 1; i <= k; i++ {
		c = c * (n - k + i) / i
	}
	return c
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// ----- relations, functions and partitions -----

type pair struct{ a, b int }

func pairsString(rel []pair) string {
	parts := make([]string, len(rel))
	for i, p := range rel {
		parts[i] = fmt.Sprintf("(%d, %d)", p.a, p.b)
	}
	return "\\{" + strings.Join(parts, ", ") + "\\}"
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// randomRelation draws a relation on {1, ..., n}, biased towards the
// properties being nearly true so the counterexample has to be found.
func randomRelation(n int) []pair {
	var rel []pair
	for a := 1; a <= n; a++ {
		for b := 1; b <= n; b++ {
			p := 0.35
			if a == b {
				p = 0.8
			}
			if rand.Float64() < p {
				rel = append(rel, pair{a, b})
			}
		}
	}
	if len(rel) == 0 {
		rel = append(rel, pair{1, 1})
	}
	return rel
}

func relationProperty(rel []pair, n int, prop string) (bool, string) {
	has := map[pair]bool{}
	for _, p := range rel {
		has[p] = true
	}
	switch prop {
	case "reflexive":
		for a := 1; a <= n; a++ {
			if !has[pair{a, a}] {
				return false, fmt.Sprintf("(%d, %d) is missing, so %d is not related to itself.", a, a, a)
			}
		}
		return true, "Every element is related to itself: (1, 1) through (" + strconv.Itoa(n) + ", " + strconv.Itoa(n) + ") are all present."
	case "symmetric":
		for _, p := range rel {
			if !has[pair{p.b, p.a}] {
				return false, fmt.Sprintf("(%d, %d) is in R but (%d, %d) is not.", p.a, p.b, p.b, p.a)
			}
		}
		return true, "Every pair (a, b) in R has its reverse (b, a) in R too."
	default:
		for _, p := range rel {
			for _, q := range rel {
				if p.b == q.a && !has[pair{p.a, q.b}] {
					return false, fmt.Sprintf("(%d, %d) and (%d, %d) are in R, but (%d, %d) is not.", p.a, p.b, q.a, q.b, p.a, q.b)
				}
			}
		}
		return true, "Whenever (a, b) and (b, c) are in R, so is (a, c): checking every chain finds no gap."
	}
}

func (g *relationsGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	switch tierOfSets(ctx.Difficulty) {
	case 0:
		n := 3
		rel := randomRelation(n)
		if rand.Intn(3) == 0 {
			return closureCount(rel, n)
		}
		prop := []string{"reflexive", "symmetric", "transitive"}[rand.Intn(3)]
		ok, why := relationProperty(rel, n, prop)
		return generator.Problem{
			Question:    fmt.Sprintf("On \\(A = \\{1, 2, 3\\}\\), is the relation \\(R = %s\\) %s? (yes/no)", pairsString(rel), prop),
			Answer:      yesNo(ok),
			Explanation: why,
		}
	case 1:
		if rand.Intn(2) == 0 {
			return partitionQuestion()
		}
		return functionQuestion()
	default:
		switch rand.Intn(3) {
		case 0:
			m := 2 + rand.Intn(3)
			n := m + 2 + rand.Intn(7)
			q := fmt.Sprintf("On \\(\\{1, 2, \\ldots, %d\\}\\), let \\(a \\sim b\\) when \\(a - b\\) is divisible by %d. How many equivalence classes are there?", n, m)
			return generator.Problem{Question: q, Answer: strconv.Itoa(m),
				Explanation: fmt.Sprintf("Two numbers are related exactly when they leave the same remainder on division by %d. Every remainder 0, …, %d occurs among 1, …, %d (since %d ≥ %d), so there are %d classes, and they partition the set.", m, m-1, n, n, m, m)}
		case 1:
			bell := []int{1, 1, 2, 5, 15, 52}
			n := 2 + rand.Intn(4)
			return generator.Problem{
				Question:    fmt.Sprintf("How many different partitions does a set with %d elements have?", n),
				Answer:      strconv.Itoa(bell[n]),
				Explanation: fmt.Sprintf("Count the ways to split %d elements into nonempty, disjoint blocks that cover the set (the Bell number B_%d): %d. For 3 elements they are {123}, {1|23}, {2|13}, {3|12}, {1|2|3}.", n, n, bell[n]),
			}
		default:
			m := 2 + rand.Intn(3)
			n := 2 + rand.Intn(3)
			return generator.Problem{
				Question:    fmt.Sprintf("\\(A\\) has %d elements and \\(B\\) has %d. How many functions \\(f : A \\to B\\) are there?", m, n),
				Answer:      strconv.Itoa(intPow(n, m)),
				Explanation: fmt.Sprintf("A function assigns each of the %d elements of A exactly one image, and each has %d choices: %d^%d = %d.", m, n, n, m, intPow(n, m)),
			}
		}
	}
}

// closureCount asks how many pairs must be added to make R reflexive, or
// symmetric: the smallest such relation containing R.
func closureCount(rel []pair, n int) generator.Problem {
	has := map[pair]bool{}
	for _, p := range rel {
		has[p] = true
	}
	var missing []string
	prop := "reflexive"
	if rand.Intn(2) == 0 {
		for a := 1; a <= n; a++ {
			if !has[pair{a, a}] {
				missing = append(missing, fmt.Sprintf("(%d, %d)", a, a))
			}
		}
	} else {
		prop = "symmetric"
		for _, p := range rel {
			if !has[pair{p.b, p.a}] {
				missing = append(missing, fmt.Sprintf("(%d, %d)", p.b, p.a))
			}
		}
	}
	why := fmt.Sprintf("R is already %s, so no pair is needed.", prop)
	if len(missing) > 0 {
		why = fmt.Sprintf("The missing pairs are %s, so %d must be added.", strings.Join(missing, ", "), len(missing))
	}
	return generator.Problem{
		Question:    fmt.Sprintf("On \\(A = \\{1, 2, 3\\}\\), how many ordered pairs must be added to \\(R = %s\\) to make it %s?", pairsString(rel), prop),
		Answer:      strconv.Itoa(len(missing)),
		Explanation: why,
	}
}

func intPow(b, e int) int {
	r := 1
	for i := 0; i < e; i++ {
		r *= b
	}
	return r
}

func tierOfSets(d float64) int {
	switch {
	case d >= 0.7:
		return 2
	case d >= 0.45:
		return 1
	}
	return 0
}

// partitionQuestion shows blocks of {1, ..., n} that are a partition or fail
// to be one in exactly one way: an element left out, an element in two
// blocks, or an empty block.
func partitionQuestion() generator.Problem {
	n := 5 + rand.Intn(3)
	perm := rand.Perm(n)
	k := 2 + rand.Intn(2)
	cuts := randSubset(rangeInts(1, n-1), k-1)
	var blocks [][]int
	start := 0
	for _, c := range append(cuts, n) {
		var b []int
		for _, i := range perm[start:c] {
			b = append(b, i+1)
		}
		sort.Ints(b)
		blocks = append(blocks, b)
		start = c
	}
	flaw := rand.Intn(4)
	why := "The blocks are nonempty, do not overlap, and together contain every element, so they form a partition."
	switch flaw {
	case 1:
		bi := rand.Intn(len(blocks))
		for len(blocks[bi]) < 2 {
			bi = rand.Intn(len(blocks))
		}
		gone := blocks[bi][len(blocks[bi])-1]
		blocks[bi] = blocks[bi][:len(blocks[bi])-1]
		why = fmt.Sprintf("%d is in no block, so the blocks do not cover the set.", gone)
	case 2:
		x := blocks[0][0]
		blocks[1] = append([]int{x}, blocks[1]...)
		sort.Ints(blocks[1])
		why = fmt.Sprintf("%d is in two blocks, so the blocks are not disjoint.", x)
	case 3:
		if rand.Intn(2) == 0 {
			blocks = append(blocks, nil)
			why = "One block is ∅, and the blocks of a partition must be nonempty."
		} else {
			flaw = 0
		}
	}
	parts := make([]string, len(blocks))
	for i, b := range blocks {
		parts[i] = texSetGo(b)
	}
	return generator.Problem{
		Question:    fmt.Sprintf("Is \\(\\{%s\\}\\) a partition of \\(\\{1, 2, \\ldots, %d\\}\\)? (yes/no)", strings.Join(parts, ", "), n),
		Answer:      yesNo(flaw == 0),
		Explanation: why,
	}
}

func texSetGo(s []int) string { return texSet(s) }

// functionQuestion shows a set of pairs from A to B and asks whether it is a
// function: every element of A appears exactly once as a first coordinate.
func functionQuestion() generator.Problem {
	letters := []string{"p", "q", "r"}
	var rel []string
	why := "Each of 1, 2, 3 appears exactly once as a first coordinate, so every element of A has exactly one image: a function."
	flaw := rand.Intn(3)
	missing := 1 + rand.Intn(3)
	double := 1 + rand.Intn(3)
	for a := 1; a <= 3; a++ {
		if flaw == 1 && a == missing {
			continue
		}
		rel = append(rel, fmt.Sprintf("(%d, %s)", a, letters[rand.Intn(3)]))
		if flaw == 2 && a == double {
			rel = append(rel, fmt.Sprintf("(%d, %s)", a, letters[(rand.Intn(2)+1+a)%3]))
		}
	}
	switch flaw {
	case 1:
		why = fmt.Sprintf("%d has no image, so the pairs do not define a function on all of A.", missing)
	case 2:
		why = fmt.Sprintf("%d appears with two different images, and a function gives each element exactly one.", double)
	}
	if flaw == 2 {
		// The two images must differ, or the "second" pair is the same pair.
		seen := map[string]bool{}
		var dedup []string
		for _, r := range rel {
			if !seen[r] {
				seen[r] = true
				dedup = append(dedup, r)
			}
		}
		if len(dedup) < len(rel) {
			rel = dedup
			flaw = 0
			why = "Each of 1, 2, 3 appears exactly once as a first coordinate, so every element of A has exactly one image: a function."
		}
	}
	return generator.Problem{
		Question:    fmt.Sprintf("Let \\(A = \\{1, 2, 3\\}\\) and \\(B = \\{p, q, r\\}\\). Is \\(\\{%s\\}\\) a function from \\(A\\) to \\(B\\)? (yes/no)", strings.Join(rel, ", ")),
		Answer:      yesNo(flaw == 0),
		Explanation: why,
	}
}
