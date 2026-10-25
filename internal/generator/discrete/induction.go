package discrete

import (
	"fmt"
	"math/big"
	"math/rand"
	"strings"

	"github.com/chuma-beep/mathua/internal/generator"
)

// inductionGen asks about proofs by induction as multiple choice (the concept's
// grading type). Earlier items asked "what is P(1)?" with key 1, but P(1) is a
// statement, not a number, and "(one word)" items had two-word keys. The items
// below ask about the parts of an induction proof and about concrete sum
// formulas: the common value of both sides of P(m), the term added in the
// inductive step, and what the right side must become. Every key is computed.
type inductionGen struct{}

// inductionSum is one formula 1st term + ... + n-th term = rhs(n).
type inductionSum struct {
	lhs      string                 // TeX left side in n
	rhs      string                 // TeX right side in n
	term     func(i int64) *big.Rat // the i-th term
	termTeX  func(i int64) string   // the i-th term as written in a check
	rhsAt    func(m int64) string   // the right side with n = m substituted
	rhsK     string                 // TeX right side of P(k)
	nextTerm string                 // TeX term added going from P(k) to P(k+1)
	rhsK1    string                 // TeX right side of P(k+1)
	step     string                 // TeX algebra: rhsK + nextTerm = rhsK1
	wrongRHS []string
	wrongAdd []string
}

func rint(n int64) *big.Rat { return new(big.Rat).SetInt64(n) }

func pow64(b, e int64) int64 {
	r := int64(1)
	for ; e > 0; e-- {
		r *= b
	}
	return r
}

func ratTeX(r *big.Rat) string {
	if r.IsInt() {
		return r.Num().String()
	}
	return fmt.Sprintf("%s/%s", r.Num().String(), r.Denom().String())
}

var inductionSums = []inductionSum{
	{
		lhs: "1 + 2 + \\cdots + n", rhs: "\\frac{n(n+1)}{2}",
		term:    func(i int64) *big.Rat { return rint(i) },
		termTeX: func(i int64) string { return fmt.Sprint(i) },
		rhsAt:   func(m int64) string { return fmt.Sprintf("\\frac{%d \\cdot %d}{2}", m, m+1) },
		rhsK:    "\\frac{k(k+1)}{2}", nextTerm: "k+1", rhsK1: "\\frac{(k+1)(k+2)}{2}",
		step:     "\\frac{k(k+1)}{2} + (k+1) = (k+1)\\left(\\frac{k}{2} + 1\\right) = \\frac{(k+1)(k+2)}{2}",
		wrongRHS: []string{"\\frac{k(k+1)}{2} + 1", "\\frac{(k+2)(k+3)}{2}", "\\frac{k(k+1)}{2}"},
		wrongAdd: []string{"k", "k+2", "2k+1"},
	},
	{
		lhs: "1 + 3 + 5 + \\cdots + (2n-1)", rhs: "n^{2}",
		term:    func(i int64) *big.Rat { return rint(2*i - 1) },
		termTeX: func(i int64) string { return fmt.Sprint(2*i - 1) },
		rhsAt:   func(m int64) string { return fmt.Sprintf("%d^{2}", m) },
		rhsK:    "k^{2}", nextTerm: "2k+1", rhsK1: "(k+1)^{2}",
		step:     "k^{2} + (2k+1) = (k+1)^{2}",
		wrongRHS: []string{"k^{2} + 1", "(k+2)^{2}", "k^{2} + 2k"},
		wrongAdd: []string{"2k-1", "2k+3", "k+1"},
	},
	{
		lhs: "1^{2} + 2^{2} + \\cdots + n^{2}", rhs: "\\frac{n(n+1)(2n+1)}{6}",
		term:    func(i int64) *big.Rat { return rint(i * i) },
		termTeX: func(i int64) string { return fmt.Sprintf("%d^{2}", i) },
		rhsAt:   func(m int64) string { return fmt.Sprintf("\\frac{%d \\cdot %d \\cdot %d}{6}", m, m+1, 2*m+1) },
		rhsK:    "\\frac{k(k+1)(2k+1)}{6}", nextTerm: "(k+1)^{2}", rhsK1: "\\frac{(k+1)(k+2)(2k+3)}{6}",
		step:     "\\frac{k(k+1)(2k+1)}{6} + (k+1)^{2} = \\frac{(k+1)\\left(2k^{2} + 7k + 6\\right)}{6} = \\frac{(k+1)(k+2)(2k+3)}{6}",
		wrongRHS: []string{"\\frac{(k+1)(k+2)(2k+1)}{6}", "\\frac{k(k+1)(2k+1)}{6} + 1", "\\frac{(k+1)(k+2)(2k+2)}{6}"},
		wrongAdd: []string{"k^{2}", "(k+2)^{2}", "k^{2}+1"},
	},
	{
		lhs: "1^{3} + 2^{3} + \\cdots + n^{3}", rhs: "\\left(\\frac{n(n+1)}{2}\\right)^{2}",
		term:    func(i int64) *big.Rat { return rint(i * i * i) },
		termTeX: func(i int64) string { return fmt.Sprintf("%d^{3}", i) },
		rhsAt:   func(m int64) string { return fmt.Sprintf("\\left(\\frac{%d \\cdot %d}{2}\\right)^{2}", m, m+1) },
		rhsK:    "\\left(\\frac{k(k+1)}{2}\\right)^{2}", nextTerm: "(k+1)^{3}", rhsK1: "\\left(\\frac{(k+1)(k+2)}{2}\\right)^{2}",
		step:     "\\frac{k^{2}(k+1)^{2}}{4} + (k+1)^{3} = \\frac{(k+1)^{2}\\left(k^{2} + 4k + 4\\right)}{4} = \\left(\\frac{(k+1)(k+2)}{2}\\right)^{2}",
		wrongRHS: []string{"\\left(\\frac{(k+1)(k+2)}{2}\\right)^{3}", "\\left(\\frac{k(k+1)}{2}\\right)^{2} + 1", "\\left(\\frac{(k+2)(k+3)}{2}\\right)^{2}"},
		wrongAdd: []string{"k^{3}", "(k+2)^{3}", "(k+1)^{2}"},
	},
	{
		lhs: "1 + 2 + 4 + \\cdots + 2^{n-1}", rhs: "2^{n} - 1",
		term:    func(i int64) *big.Rat { return rint(pow64(2, i-1)) },
		termTeX: func(i int64) string { return fmt.Sprint(pow64(2, i-1)) },
		rhsAt:   func(m int64) string { return fmt.Sprintf("2^{%d} - 1", m) },
		rhsK:    "2^{k} - 1", nextTerm: "2^{k}", rhsK1: "2^{k+1} - 1",
		step:     "2^{k} - 1 + 2^{k} = 2 \\cdot 2^{k} - 1 = 2^{k+1} - 1",
		wrongRHS: []string{"2^{k+1}", "2^{k+1} + 1", "2^{k} + 1"},
		wrongAdd: []string{"2^{k+1}", "2^{k-1}", "2k"},
	},
	{
		lhs: "1 + 3 + 9 + \\cdots + 3^{n-1}", rhs: "\\frac{3^{n} - 1}{2}",
		term:    func(i int64) *big.Rat { return rint(pow64(3, i-1)) },
		termTeX: func(i int64) string { return fmt.Sprint(pow64(3, i-1)) },
		rhsAt:   func(m int64) string { return fmt.Sprintf("\\frac{3^{%d} - 1}{2}", m) },
		rhsK:    "\\frac{3^{k} - 1}{2}", nextTerm: "3^{k}", rhsK1: "\\frac{3^{k+1} - 1}{2}",
		step:     "\\frac{3^{k} - 1}{2} + 3^{k} = \\frac{3 \\cdot 3^{k} - 1}{2} = \\frac{3^{k+1} - 1}{2}",
		wrongRHS: []string{"\\frac{3^{k+1} + 1}{2}", "\\frac{3^{k} - 1}{2} + 1", "3^{k+1} - 1"},
		wrongAdd: []string{"3^{k+1}", "3^{k-1}", "3k"},
	},
	{
		lhs: "1 \\cdot 2 + 2 \\cdot 3 + \\cdots + n(n+1)", rhs: "\\frac{n(n+1)(n+2)}{3}",
		term:    func(i int64) *big.Rat { return rint(i * (i + 1)) },
		termTeX: func(i int64) string { return fmt.Sprintf("%d \\cdot %d", i, i+1) },
		rhsAt:   func(m int64) string { return fmt.Sprintf("\\frac{%d \\cdot %d \\cdot %d}{3}", m, m+1, m+2) },
		rhsK:    "\\frac{k(k+1)(k+2)}{3}", nextTerm: "(k+1)(k+2)", rhsK1: "\\frac{(k+1)(k+2)(k+3)}{3}",
		step:     "\\frac{k(k+1)(k+2)}{3} + (k+1)(k+2) = (k+1)(k+2)\\left(\\frac{k}{3} + 1\\right) = \\frac{(k+1)(k+2)(k+3)}{3}",
		wrongRHS: []string{"\\frac{k(k+1)(k+2)}{3} + 1", "\\frac{(k+1)(k+2)(k+3)}{6}", "\\frac{(k+2)(k+3)(k+4)}{3}"},
		wrongAdd: []string{"k(k+1)", "(k+2)(k+3)", "(k+1)^{2}"},
	},
	{
		lhs: "\\frac{1}{1 \\cdot 2} + \\frac{1}{2 \\cdot 3} + \\cdots + \\frac{1}{n(n+1)}", rhs: "\\frac{n}{n+1}",
		term:    func(i int64) *big.Rat { return big.NewRat(1, i*(i+1)) },
		termTeX: func(i int64) string { return fmt.Sprintf("\\frac{1}{%d}", i*(i+1)) },
		rhsAt:   func(m int64) string { return fmt.Sprintf("\\frac{%d}{%d}", m, m+1) },
		rhsK:    "\\frac{k}{k+1}", nextTerm: "\\frac{1}{(k+1)(k+2)}", rhsK1: "\\frac{k+1}{k+2}",
		step:     "\\frac{k}{k+1} + \\frac{1}{(k+1)(k+2)} = \\frac{k^{2} + 2k + 1}{(k+1)(k+2)} = \\frac{k+1}{k+2}",
		wrongRHS: []string{"\\frac{k}{k+1} + 1", "\\frac{k+2}{k+3}", "\\frac{k}{k+2}"},
		wrongAdd: []string{"\\frac{1}{k(k+1)}", "\\frac{1}{(k+2)(k+3)}", "\\frac{1}{k+1}"},
	},
}

// inductionValue is the common value of both sides of P(m).
func inductionValue(f inductionSum, m int64) *big.Rat {
	v := new(big.Rat)
	for i := int64(1); i <= m; i++ {
		v.Add(v, f.term(i))
	}
	return v
}

// inline wraps TeX in $...$, the form the registry's canonicaliser serves, so
// an option's text and the stored key stay byte-identical.
func inline(tex string) string { return "$" + tex + "$" }

func (g *inductionGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	switch rand.Intn(6) {
	case 0:
		return inductionPartsItem()
	case 1, 2:
		return inductionValueItem()
	case 3:
		return inductionNextTermItem()
	default:
		return inductionStepItem()
	}
}

func inductionPartsItem() generator.Problem {
	type part struct{ q, correct, why string }
	parts := []part{
		{"In a proof by induction that P(n) holds for every integer n \\(\\ge 1\\), which part shows that P(1) is true?", "the base case",
			"The base case checks the first instance directly; the inductive step then carries the truth from each k to k + 1."},
		{"In a proof by induction, which part shows that P(k) implies P(k + 1) for every k \\(\\ge 1\\)?", "the inductive step",
			"The inductive step assumes P(k) (the inductive hypothesis) and derives P(k + 1)."},
		{"In the inductive step, the assumption that P(k) is true for a fixed but arbitrary k is called...", "the inductive hypothesis",
			"The inductive hypothesis is the assumption P(k) used to prove P(k + 1)."},
	}
	all := []string{"the base case", "the inductive step", "the inductive hypothesis", "the contrapositive"}
	p := parts[rand.Intn(len(parts))]
	var ds []string
	for _, o := range all {
		if o != p.correct {
			ds = append(ds, o)
		}
	}
	pr := generator.MultipleChoice(p.q, p.correct, ds...)
	pr.Explanation = p.why
	return pr
}

func inductionValueItem() generator.Problem {
	f := inductionSums[rand.Intn(len(inductionSums))]
	m := int64(3 + rand.Intn(5))
	v := inductionValue(f, m)
	// Keep the key at least two characters (one-character keys collide with option letters).
	for len(ratTeX(v)) < 2 {
		m++
		v = inductionValue(f, m)
	}
	correct := ratTeX(v)
	cands := []string{ratTeX(inductionValue(f, m+1)), ratTeX(inductionValue(f, m-1)),
		ratTeX(new(big.Rat).Add(v, f.term(m+1))), ratTeX(new(big.Rat).Sub(v, f.term(1))), ratTeX(new(big.Rat).Add(v, rint(1)))}
	var ds []string
	seen := map[string]bool{correct: true}
	for _, c := range cands {
		if !seen[c] && len(ds) < 3 {
			seen[c] = true
			ds = append(ds, c)
		}
	}
	var terms []string
	for i := int64(1); i <= m; i++ {
		terms = append(terms, f.termTeX(i))
	}
	q := fmt.Sprintf("Let P(n) be the statement %s. Checking P(%d) by hand, both sides come out equal. What is their common value?",
		inline(f.lhs+" = "+f.rhs), m)
	pr := generator.MultipleChoice(q, correct, ds...)
	pr.Explanation = fmt.Sprintf("P(%d) says %s. The left side is %s = %s, and the right side is %s = %s, so P(%d) is true and both sides equal %s.",
		m, inline(strings.Join(terms, " + ")+" = "+f.rhsAt(m)), inline(strings.Join(terms, " + ")), correct, inline(f.rhsAt(m)), correct, m, correct)
	return pr
}

func inductionNextTermItem() generator.Problem {
	f := inductionSums[rand.Intn(len(inductionSums))]
	ds := make([]string, len(f.wrongAdd))
	for i, w := range f.wrongAdd {
		ds[i] = inline(w)
	}
	q := fmt.Sprintf("Let P(n) be the statement %s. Going from P(k) to P(k + 1), which term is added to the left side?", inline(f.lhs+" = "+f.rhs))
	pr := generator.MultipleChoice(q, inline(f.nextTerm), ds...)
	pr.Explanation = fmt.Sprintf("The left side of P(k + 1) is the left side of P(k) plus its (k + 1)-th term, which is %s.", inline(f.nextTerm))
	return pr
}

func inductionStepItem() generator.Problem {
	f := inductionSums[rand.Intn(len(inductionSums))]
	ds := make([]string, len(f.wrongRHS))
	for i, w := range f.wrongRHS {
		ds[i] = inline(w)
	}
	q := fmt.Sprintf("Let P(n) be the statement %s. In the inductive step you assume P(k) and add %s to both sides. To finish, the right side must simplify to which expression?",
		inline(f.lhs+" = "+f.rhs), inline(f.nextTerm))
	pr := generator.MultipleChoice(q, inline(f.rhsK1), ds...)
	pr.Explanation = fmt.Sprintf("P(k + 1) has right side %s, so that is the target: %s. This proves P(k + 1) from P(k).", inline(f.rhsK1), inline(f.step))
	return pr
}
