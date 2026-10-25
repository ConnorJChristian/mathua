package calculus

// Computed variants added after the 2026-10-25 QA run (Q47: pools that served
// only 3-7 distinct questions). Each family draws its numbers, computes the key
// in Go and writes an explanation from the same numbers, so the worked answer
// is true for the instance. The older table items stay in each generator.

import (
	"fmt"
	"math/big"
	"math/rand"
	"strings"

	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
)

// rs is the reduced fraction n/d (or an integer).
func rs(n, d int) string { return ratStr(int64(n), int64(d)) }

func iabs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// distinctFrom keeps the first three candidates that differ from the correct
// option and from each other, so every multiple-choice item has four options.
func distinctFrom(correct string, cands ...string) []string {
	seen := map[string]bool{correct: true}
	var out []string
	for _, c := range cands {
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		out = append(out, c)
		if len(out) == 3 {
			break
		}
	}
	return out
}

// dollarMath writes inline math as $...$, the form the registry serves, so an
// option's text and the stored key stay identical after canonicalisation.
func dollarMath(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\\(", "$"), "\\)", "$")
}

// xShift writes x - a without "x - 0" or "x - (-2)".
func xShift(v string, a int) string {
	switch {
	case a == 0:
		return v
	case a > 0:
		return fmt.Sprintf("%s - %d", v, a)
	}
	return fmt.Sprintf("%s + %d", v, -a)
}

// quotTeX writes a/c, followed by "= L" only when it reduces ("9/3 = 3").
func quotTeX(a, c int) string {
	q := fmt.Sprintf("%d/%d", a, c)
	if r := rs(a, c); r != q {
		return q + " = " + r
	}
	return q
}

// kPlusH writes k + h for the difference quotient at k ("h" when k = 0).
func kPlusH(k int) string {
	if k == 0 {
		return "h"
	}
	return fmt.Sprintf("%d + h", k)
}

// nonzero draws from [lo, hi] without 0 (a "0 + 2/n" reads as an artefact).
func nonzero(lo, hi int) int {
	for {
		if v := lo + rand.Intn(hi-lo+1); v != 0 {
			return v
		}
	}
}

func mcWith(q, correct, why string, cands ...string) generator.Problem {
	correct = dollarMath(correct)
	for i := range cands {
		cands[i] = dollarMath(cands[i])
	}
	p := generator.MultipleChoice(q, correct, distinctFrom(correct, cands...)...)
	p.Explanation = why
	return p
}

// ---------------------------------------------------------------------------
// calc.deriv.optimization (numeric)
// ---------------------------------------------------------------------------

func optimizationVariant() generator.Problem {
	switch rand.Intn(5) {
	case 0:
		P := 4 * (3 + rand.Intn(15))
		x := P / 4
		area := x * (P - 2*x)
		if rand.Intn(2) == 0 {
			return generator.Problem{
				Question:    fmt.Sprintf("A farmer has %d m of fence to enclose a rectangle along a straight river; the river side needs no fence. How long (in m) should each of the two sides perpendicular to the river be to maximise the area?", P),
				Answer:      fmt.Sprintf("%d", x),
				Explanation: fmt.Sprintf("With perpendicular sides \\(x\\), the side along the river is \\(%d - 2x\\), so \\(A(x) = x(%d - 2x)\\) and \\(A'(x) = %d - 4x = 0\\) at \\(x = %d\\). \\(A''(x) = -4 < 0\\), so this is the maximum.", P, P, P, x),
			}
		}
		return generator.Problem{
			Question:    fmt.Sprintf("A farmer has %d m of fence to enclose a rectangle along a straight river; the river side needs no fence. What is the largest area (in square metres) that can be enclosed?", P),
			Answer:      fmt.Sprintf("%d", area),
			Explanation: fmt.Sprintf("\\(A(x) = x(%d - 2x)\\) with \\(A'(x) = %d - 4x = 0\\) at \\(x = %d\\), so the sides are %d and %d and the area is %d.", P, P, x, x, P-2*x, area),
		}
	case 1:
		a := 1 + rand.Intn(4)
		qs := 2 + rand.Intn(11)
		b := 2 * a * qs
		c := 5 * rand.Intn(9)
		pi := []xyTerm{{-a, 2, 0}, {b, 1, 0}, {-c, 0, 0}}
		best := a*qs*qs - c
		f := texInVar(pi, "q")
		d := texInVar([]xyTerm{{-2 * a, 1, 0}, {b, 0, 0}}, "q")
		if rand.Intn(2) == 0 {
			return generator.Problem{
				Question:    fmt.Sprintf("A firm's profit is \\(\\pi(q) = %s\\). What output \\(q\\) maximises profit?", f),
				Answer:      fmt.Sprintf("%d", qs),
				Explanation: fmt.Sprintf("\\(\\pi'(q) = %s = 0\\) at \\(q = %d\\), and \\(\\pi''(q) = %d < 0\\), so this is the maximum.", d, qs, -2*a),
			}
		}
		return generator.Problem{
			Question:    fmt.Sprintf("A firm's profit is \\(\\pi(q) = %s\\). What is the maximum profit?", f),
			Answer:      fmt.Sprintf("%d", best),
			Explanation: fmt.Sprintf("\\(\\pi'(q) = %s = 0\\) at \\(q = %d\\) (a maximum, since \\(\\pi''(q) = %d < 0\\)), and \\(\\pi(%d) = %d\\).", d, qs, -2*a, qs, best),
		}
	case 2:
		k := 1 + rand.Intn(5)
		s := 6 * k
		vol := k * (s - 2*k) * (s - 2*k)
		if rand.Intn(2) == 0 {
			return generator.Problem{
				Question:    fmt.Sprintf("An open box is made from a %d cm by %d cm square sheet by cutting equal squares of side \\(x\\) from the corners and folding up the sides. Which \\(x\\) (in cm) gives the largest volume?", s, s),
				Answer:      fmt.Sprintf("%d", k),
				Explanation: fmt.Sprintf("\\(V(x) = x(%d - 2x)^{2}\\) and \\(V'(x) = (%d - 2x)(%d - 6x)\\). The root \\(x = %d\\) gives no box, so the maximum on \\((0, %d)\\) is at \\(x = %d\\).", s, s, s, s/2, s/2, k),
			}
		}
		return generator.Problem{
			Question:    fmt.Sprintf("An open box is made from a %d cm by %d cm square sheet by cutting equal squares from the corners and folding up the sides. What is the largest possible volume (in cubic cm)?", s, s),
			Answer:      fmt.Sprintf("%d", vol),
			Explanation: fmt.Sprintf("\\(V(x) = x(%d - 2x)^{2}\\), and \\(V'(x) = (%d - 2x)(%d - 6x) = 0\\) gives the maximum at \\(x = %d\\): \\(V = %d \\cdot %d^{2} = %d\\).", s, s, s, k, k, s-2*k, vol),
		}
	case 3:
		m := 2 + rand.Intn(14)
		return generator.Problem{
			Question:    fmt.Sprintf("Two positive numbers have product %d. What is the smallest possible value of their sum?", m*m),
			Answer:      fmt.Sprintf("%d", 2*m),
			Explanation: fmt.Sprintf("Minimise \\(S(x) = x + %d/x\\) for \\(x > 0\\): \\(S'(x) = 1 - %d/x^{2} = 0\\) at \\(x = %d\\), and \\(S''(x) > 0\\), so the minimum sum is \\(%d + %d = %d\\).", m*m, m*m, m, m, m, 2*m),
		}
	default:
		B := 1 + rand.Intn(3)
		qs := 2 + rand.Intn(9)
		C := 1 + rand.Intn(10)
		A := C + 2*B*qs
		price := A - B*qs
		demand := texInVar([]xyTerm{{A, 0, 0}, {-B, 1, 0}}, "q")
		profit := texInVar([]xyTerm{{-B, 2, 0}, {A - C, 1, 0}}, "q")
		if rand.Intn(2) == 0 {
			return generator.Problem{
				Question:    fmt.Sprintf("A monopolist faces inverse demand \\(p = %s\\) and has constant marginal cost %d. What output maximises profit?", demand, C),
				Answer:      fmt.Sprintf("%d", qs),
				Explanation: fmt.Sprintf("Profit is \\(\\pi(q) = (%s)q - %dq = %s\\), so \\(\\pi'(q) = %d - %dq = 0\\) at \\(q = %d\\).", demand, C, profit, A-C, 2*B, qs),
			}
		}
		return generator.Problem{
			Question:    fmt.Sprintf("A monopolist faces inverse demand \\(p = %s\\) and has constant marginal cost %d. What price does it charge at the profit-maximising output?", demand, C),
			Answer:      fmt.Sprintf("%d", price),
			Explanation: fmt.Sprintf("Profit is \\(\\pi(q) = %s\\), maximised where \\(\\pi'(q) = %d - %dq = 0\\), at \\(q = %d\\). The price is then \\(%d - %d \\cdot %d = %d\\).", profit, A-C, 2*B, qs, A, B, qs, price),
		}
	}
}

// ---------------------------------------------------------------------------
// calc.deriv.concept (numeric): templates other than the power rule at a point
// ---------------------------------------------------------------------------

func derivConceptVariant() generator.Problem {
	switch rand.Intn(5) {
	case 0:
		a := []int{1, 2, 3, -1, -2}[rand.Intn(5)]
		b := rand.Intn(11) - 5
		c := rand.Intn(9) - 4
		k := rand.Intn(7) - 3
		f := texXY([]xyTerm{{a, 2, 0}, {b, 1, 0}, {c, 0, 0}})
		ans := 2*a*k + b
		return generator.Problem{
			Question:    fmt.Sprintf("Using \\(f'(k) = \\lim_{h \\to 0} \\frac{f(k+h) - f(k)}{h}\\), find \\(f'(%d)\\) for \\(f(x) = %s\\).", k, f),
			Answer:      fmt.Sprintf("%d", ans),
			Explanation: fmt.Sprintf("At \\(k = %d\\), \\(f(%s) - f(%d) = %s\\), so the difference quotient is \\(%s\\), which tends to %d as \\(h \\to 0\\).", k, kPlusH(k), k, texInVar([]xyTerm{{a, 2, 0}, {ans, 1, 0}}, "h"), texInVar([]xyTerm{{a, 1, 0}, {ans, 0, 0}}, "h"), ans),
		}
	case 1:
		a := 1 + rand.Intn(9)
		k := 1 + rand.Intn(5)
		return generator.Problem{
			Question:    fmt.Sprintf("What is the slope of the tangent line to \\(f(x) = %d/x\\) at \\(x = %d\\)? (enter a number or fraction)", a, k),
			Answer:      rs(-a, k*k),
			Explanation: fmt.Sprintf("\\(f(x) = %sx^{-1}\\), so \\(f'(x) = -%d/x^{2}\\) and \\(f'(%d) = -%d/%d = %s\\).", coefStr(a), a, k, a, k*k, rs(-a, k*k)),
		}
	case 2:
		a := 1 + rand.Intn(8)
		k := 1 + rand.Intn(6)
		return generator.Problem{
			Question:    fmt.Sprintf("What is the slope of the tangent line to \\(f(x) = %d\\sqrt{x}\\) at \\(x = %d\\)? (enter a number or fraction)", a, k*k),
			Answer:      rs(a, 2*k),
			Explanation: fmt.Sprintf("\\(f'(x) = \\frac{%d}{2\\sqrt{x}}\\), so \\(f'(%d) = \\frac{%d}{2 \\cdot %d} = %s\\).", a, k*k, a, k, rs(a, 2*k)),
		}
	case 3:
		a := 1 + rand.Intn(6)
		b := rand.Intn(11) - 5
		c := rand.Intn(20)
		t := 1 + rand.Intn(6)
		s := texInVar([]xyTerm{{a, 2, 0}, {b, 1, 0}, {c, 0, 0}}, "t")
		v := 2*a*t + b
		return generator.Problem{
			Question:    fmt.Sprintf("A particle's position is \\(s(t) = %s\\) metres after \\(t\\) seconds. What is its instantaneous velocity (m/s) at \\(t = %d\\)?", s, t),
			Answer:      fmt.Sprintf("%d", v),
			Explanation: fmt.Sprintf("Instantaneous velocity is the derivative \\(s'(t) = %s\\), so \\(s'(%d) = %d\\).", texInVar([]xyTerm{{2 * a, 1, 0}, {b, 0, 0}}, "t"), t, v),
		}
	default:
		a := []int{1, 2, 3, -1, -2}[rand.Intn(5)]
		b := rand.Intn(9) - 4
		k := rand.Intn(7) - 3
		f := []xyTerm{{a, 2, 0}, {b, 1, 0}}
		fk := a*k*k + b*k
		m := 2*a*k + b
		icpt := fk - m*k
		return generator.Problem{
			Question:    fmt.Sprintf("The tangent line to \\(f(x) = %s\\) at \\(x = %d\\) crosses the y-axis at what value?", texXY(f), k),
			Answer:      fmt.Sprintf("%d", icpt),
			Explanation: fmt.Sprintf("\\(f(%d) = %d\\) and \\(f'(x) = %s\\) gives slope %d at \\(x = %d\\). The tangent line \\(y = f(%d) + f'(%d)(x - %s)\\) simplifies to \\(y = %s\\), so it crosses the y-axis at %d.", k, fk, texXY([]xyTerm{{2 * a, 1, 0}, {b, 0, 0}}), m, k, k, k, paren(k), texXY([]xyTerm{{m, 1, 0}, {icpt, 0, 0}}), icpt),
		}
	}
}

// ---------------------------------------------------------------------------
// calc.series.gamma (numeric)
// ---------------------------------------------------------------------------

func gammaVariant() generator.Problem {
	switch rand.Intn(4) {
	case 0:
		n := 2 + rand.Intn(7)
		k := 1 + rand.Intn(3)
		v := 1
		var fs []string
		for i := 0; i < k; i++ {
			v *= n + i
			fs = append(fs, fmt.Sprint(n+i))
		}
		return generator.Problem{
			Question:    fmt.Sprintf("Using \\(\\Gamma(x+1) = x\\,\\Gamma(x)\\), what is \\(\\Gamma(%d)/\\Gamma(%d)\\)? (enter a number)", n+k, n),
			Answer:      fmt.Sprint(v),
			Explanation: fmt.Sprintf("Applying the recursion %s gives \\(\\Gamma(%d) = %s \\cdot \\Gamma(%d)\\), so the ratio is %d.", []string{"", "once", "twice", "three times"}[k], n+k, strings.Join(fs, " \\cdot "), n, v),
		}
	case 1:
		k := 2 + rand.Intn(6)
		v := calcFact(k)
		return generator.Problem{
			Question:    fmt.Sprintf("Evaluate \\(\\int_{0}^{\\infty} x^{%d} e^{-x}\\,dx\\). (enter a number)", k),
			Answer:      fmt.Sprint(v),
			Explanation: fmt.Sprintf("The integral is \\(\\Gamma(%d) = %d! = %d\\).", k+1, k, v),
		}
	case 2:
		m := 3 + rand.Intn(6)
		M := calcFact(m - 1)
		return generator.Problem{
			Question:    fmt.Sprintf("Given \\(\\Gamma(%d) = %d\\), use \\(\\Gamma(x+1) = x\\,\\Gamma(x)\\) to find \\(\\Gamma(%d)\\). (enter a number)", m, M, m+1),
			Answer:      fmt.Sprint(m * M),
			Explanation: fmt.Sprintf("\\(\\Gamma(%d) = %d \\cdot \\Gamma(%d) = %d \\cdot %d = %d\\).", m+1, m, m, m, M, m*M),
		}
	default:
		n := 1 + rand.Intn(4)
		num, den := 1, 1
		var fs []string
		for i := 1; i <= n; i++ {
			num *= 2*i - 1
			den *= 2
			fs = append(fs, fmt.Sprintf("\\frac{%d}{2}", 2*i-1))
		}
		return generator.Problem{
			Question:    fmt.Sprintf("Using \\(\\Gamma(x+1) = x\\,\\Gamma(x)\\), what is \\(\\Gamma(%d + \\tfrac{1}{2})/\\Gamma(\\tfrac{1}{2})\\)? (enter a fraction)", n),
			Answer:      rs(num, den),
			Explanation: fmt.Sprintf("Stepping down from \\(%d + \\tfrac{1}{2}\\) to \\(\\tfrac{1}{2}\\) multiplies by \\(%s\\), which is %s.", n, strings.Join(fs, " \\cdot "), rs(num, den)),
		}
	}
}

// ---------------------------------------------------------------------------
// calc.series.cauchy_product (numeric)
// ---------------------------------------------------------------------------

func cauchyProductVariant() generator.Problem {
	switch rand.Intn(3) {
	case 0:
		vals := []int{1, 2, 3, -1, -2}
		a := vals[rand.Intn(len(vals))]
		b := vals[rand.Intn(len(vals))]
		n := 2 + rand.Intn(4)
		c := 0
		var terms []string
		for k := 0; k <= n; k++ {
			t := ipow(a, k) * ipow(b, n-k)
			c += t
			terms = append(terms, paren(t))
		}
		return generator.Problem{
			Question: fmt.Sprintf("Form the Cauchy product of \\(\\sum_{n\\ge 0} (%sx)^{n}\\) and \\(\\sum_{n\\ge 0} (%sx)^{n}\\). What is the coefficient of \\(x^{%d}\\)? (enter a number)", coefStr(a), coefStr(b), n),
			Answer:   fmt.Sprint(c),
			Explanation: fmt.Sprintf("The coefficient of \\(x^{%d}\\) is \\(\\sum_{k=0}^{%d} a^{k} b^{%d-k}\\) with \\(a = %d\\), \\(b = %d\\): \\(%s = %d\\).",
				n, n, n, a, b, strings.Join(terms, " + "), c),
		}
	case 1:
		n := 2 + rand.Intn(8)
		c := (n + 1) * (n + 2) / 2
		return generator.Problem{
			Question:    fmt.Sprintf("Form the Cauchy product of \\(1 + x + x^{2} + \\cdots\\) and \\(1 + 2x + 3x^{2} + \\cdots\\). What is the coefficient of \\(x^{%d}\\)? (enter a number)", n),
			Answer:      fmt.Sprint(c),
			Explanation: fmt.Sprintf("The coefficient is \\(\\sum_{k=0}^{%d} 1 \\cdot (k+1) = 1 + 2 + \\cdots + %d = %d\\).", n, n+1, c),
		}
	default:
		p := 2 + rand.Intn(5)
		q := 2 + rand.Intn(5)
		ans := rs(p*q, (p-1)*(q-1))
		return generator.Problem{
			Question:    fmt.Sprintf("The series \\(\\sum_{n\\ge 0} (1/%d)^{n}\\) and \\(\\sum_{n\\ge 0} (1/%d)^{n}\\) converge absolutely. What is the sum of their Cauchy product? (enter a fraction)", p, q),
			Answer:      ans,
			Explanation: fmt.Sprintf("By Mertens' theorem the Cauchy product of two absolutely convergent series converges to the product of their sums: \\(\\frac{1}{1 - 1/%d} \\cdot \\frac{1}{1 - 1/%d} = \\frac{%d}{%d} \\cdot \\frac{%d}{%d} = %s\\).", p, q, p, p-1, q, q-1, ans),
		}
	}
}

// coefStr writes a coefficient in front of x: "", "-", "2", "-3".
func coefStr(a int) string {
	switch a {
	case 1:
		return ""
	case -1:
		return "-"
	}
	return fmt.Sprint(a)
}

// ---------------------------------------------------------------------------
// calc.multi.iterated_limits (multiple choice)
// ---------------------------------------------------------------------------

func iteratedLimitsVariant() generator.Problem {
	vals := []int{-4, -3, -2, -1, 1, 2, 3, 4, 5}
	switch rand.Intn(3) {
	case 0:
		a := vals[rand.Intn(len(vals))]
		b := vals[rand.Intn(len(vals))]
		for b == a {
			b = vals[rand.Intn(len(vals))]
		}
		f := fmt.Sprintf("\\frac{%s}{x^{2} + y^{2}}", texXY([]xyTerm{{a, 2, 0}, {b, 0, 2}}))
		inner, outer, want, other := "y", "x", a, b
		if rand.Intn(2) == 0 {
			inner, outer, want, other = "x", "y", b, a
		}
		q := fmt.Sprintf("For \\(f(x,y) = %s\\), what is \\(\\lim_{%s \\to 0} \\lim_{%s \\to 0} f(x,y)\\)?", f, outer, inner)
		why := fmt.Sprintf("With \\(%s \\ne 0\\) fixed, letting \\(%s \\to 0\\) leaves \\(%d%s^{2}/%s^{2} = %d\\), so the iterated limit is %d. The other order gives %d, so the double limit does not exist.", outer, inner, want, outer, outer, want, want, other)
		return mcWith(q, fmt.Sprint(want), why, fmt.Sprint(other), rs(a+b, 2), "0", fmt.Sprint(a+b), fmt.Sprint(-want), fmt.Sprint(want+other+1), fmt.Sprint(want+2), fmt.Sprint(want-2), fmt.Sprint(want+3))
	case 1:
		a := vals[rand.Intn(len(vals))]
		b := vals[rand.Intn(len(vals))]
		c := vals[rand.Intn(len(vals))]
		f := fmt.Sprintf("\\frac{%s}{x^{2} + y^{2}}", texXY([]xyTerm{{a, 2, 0}, {c, 1, 1}, {b, 0, 2}}))
		along := rs(a+b+c, 2)
		q := fmt.Sprintf("For \\(f(x,y) = %s\\), what is the limit of \\(f\\) as \\((x,y) \\to (0,0)\\) along the line \\(y = x\\)?", f)
		why := fmt.Sprintf("On \\(y = x\\) the numerator is \\(%s\\), so the quotient is \\(%s/2 = %s\\) for every \\(x \\ne 0\\). (Along the axes the limits are %d and %d.)", texXY([]xyTerm{{a + b + c, 2, 0}}), fmt.Sprint(a+b+c), along, a, b)
		if a+b+c == 0 {
			why = fmt.Sprintf("On \\(y = x\\) the numerator is \\((%s)x^{2} = 0\\), so the quotient is 0 for every \\(x \\ne 0\\). (Along the axes the limits are %d and %d.)", texXY([]xyTerm{{a, 0, 0}, {c, 0, 0}, {b, 0, 0}}), a, b)
		}
		return mcWith(q, along, why, fmt.Sprint(a), fmt.Sprint(b), rs(a+b, 2), "0", fmt.Sprint(a+b+c), rs(c, 2), rs(a+b+c+2, 2), rs(a+b+c-2, 2), rs(a+b+c+4, 2), rs(a+b+c-4, 2), rs(a+b+c+6, 2))
	default:
		k := 1 + rand.Intn(8)
		f := fmt.Sprintf("\\frac{%sxy^{2}}{x^{2} + y^{4}}", coefStr(k))
		along := rs(k, 2)
		q := fmt.Sprintf("For \\(f(x,y) = %s\\), both iterated limits at the origin are 0. What is the limit of \\(f\\) along the curve \\(x = y^{2}\\) as \\(y \\to 0\\)?", f)
		why := fmt.Sprintf("Substituting \\(x = y^{2}\\) gives \\(%dy^{4}/(2y^{4}) = %s\\), which differs from 0, so the double limit at the origin does not exist even though both iterated limits are 0.", k, along)
		return mcWith(q, along, why, "0", fmt.Sprint(k), fmt.Sprint(2*k), rs(k, 4))
	}
}

// ---------------------------------------------------------------------------
// calc.limit.limsup (multiple choice)
// ---------------------------------------------------------------------------

func limsupPair(sup, inf string) string { return "lim sup = " + sup + ", lim inf = " + inf }

func limsupVariant() generator.Problem {
	switch rand.Intn(3) {
	case 0:
		c := rand.Intn(9) - 4
		d := []int{1, 2, 3, -1, -2, -3}[rand.Intn(6)]
		e := 1 + rand.Intn(5)
		var parts []string
		if c != 0 {
			parts = append(parts, fmt.Sprint(c))
		}
		dTerm := fmt.Sprintf("%s(-1)^{n}", coefStr(iabs(d)))
		switch {
		case d < 0 && len(parts) > 0:
			parts = append(parts, "- "+dTerm)
		case d < 0:
			parts = append(parts, "-"+dTerm)
		case len(parts) > 0:
			parts = append(parts, "+ "+dTerm)
		default:
			parts = append(parts, dTerm)
		}
		parts = append(parts, fmt.Sprintf("+ \\frac{%d}{n}", e))
		term := strings.Join(parts, " ")
		even, odd := c+d, c-d
		hi, lo := c+iabs(d), c-iabs(d)
		q := fmt.Sprintf("What are the limit superior and limit inferior of \\(a_{n} = %s\\)?", term)
		why := fmt.Sprintf("The term \\(%d/n\\) tends to 0, so the even-indexed terms tend to %d and the odd-indexed terms to %d. The larger, %d, is the lim sup and the smaller, %d, the lim inf.", e, even, odd, hi, lo)
		return mcWith(q, limsupPair(fmt.Sprint(hi), fmt.Sprint(lo)), why,
			limsupPair(fmt.Sprint(lo), fmt.Sprint(hi)), limsupPair(fmt.Sprint(hi+e), fmt.Sprint(lo)),
			limsupPair(fmt.Sprint(c), fmt.Sprint(c)), limsupPair(fmt.Sprint(hi), fmt.Sprint(c)))
	case 1:
		c := 1 + rand.Intn(6)
		e := rand.Intn(9) - 4
		term := fmt.Sprintf("%s\\cos(n\\pi/2)", coefStr(c))
		if e != 0 {
			term = fmt.Sprintf("%d + %s", e, term)
		}
		q := fmt.Sprintf("What are the limit superior and limit inferior of \\(a_{n} = %s\\)?", term)
		why := fmt.Sprintf("\\(\\cos(n\\pi/2)\\) cycles through \\(0, -1, 0, 1\\), so the terms cycle through \\(%d, %d, %d, %d\\). The largest value taken infinitely often is %d and the smallest is %d.", e, e-c, e, e+c, e+c, e-c)
		return mcWith(q, limsupPair(fmt.Sprint(e+c), fmt.Sprint(e-c)), why,
			limsupPair(fmt.Sprint(e+c), fmt.Sprint(e)), limsupPair(fmt.Sprint(e), fmt.Sprint(e)),
			limsupPair(fmt.Sprint(e-c), fmt.Sprint(e+c)))
	default:
		// a_n = (-1)^n (1 + 1/n): s_N = sup{a_k : k >= N} is 1 + 1/m for the first even m >= N.
		N := 1 + rand.Intn(12)
		m := N
		if m%2 == 1 {
			m++
		}
		s := rs(m+1, m)
		q := fmt.Sprintf("For \\(a_{n} = (-1)^{n}(1 + 1/n)\\), what is \\(s_{%d} = \\sup\\{a_{k} : k \\ge %d\\}\\)?", N, N)
		why := fmt.Sprintf("Odd terms are negative, and even terms \\(1 + 1/k\\) decrease, so the supremum of the tail is the first even term with \\(k \\ge %d\\), at \\(k = %d\\): \\(1 + 1/%d = %s\\). These suprema decrease to the lim sup, 1.", N, m, m, s)
		return mcWith(q, s, why, "1", rs(m+2, m+1), rs(N+2, N+1), rs(m+3, m+2), "2")
	}
}

// ---------------------------------------------------------------------------
// calc.limit.supremum (symbolic)
// ---------------------------------------------------------------------------

func supremumVariant() generator.Problem {
	wantSup := rand.Intn(2) == 0
	word := "supremum"
	if !wantSup {
		word = "infimum"
	}
	switch rand.Intn(4) {
	case 0:
		a := nonzero(-5, 5)
		b := 1 + rand.Intn(6)
		set := fmt.Sprintf("\\{%d - %d/n : n \\in \\mathbb{N}\\}", a, b)
		if wantSup {
			return generator.Problem{
				Question:    fmt.Sprintf("What is the supremum of \\(%s\\)? (enter a number)", set),
				Answer:      fmt.Sprint(a),
				Explanation: fmt.Sprintf("Every element is below %d, and \\(%d - %d/n\\) gets within any \\(\\varepsilon > 0\\) of %d for large \\(n\\), so the supremum is %d. It is not attained.", a, a, b, a, a),
			}
		}
		return generator.Problem{
			Question:    fmt.Sprintf("What is the infimum of \\(%s\\)? (enter a number)", set),
			Answer:      fmt.Sprint(a - b),
			Explanation: fmt.Sprintf("The elements increase with \\(n\\), so the smallest is at \\(n = 1\\): \\(%d - %d = %d\\). It is attained, so it is also the minimum.", a, b, a-b),
		}
	case 1:
		a := nonzero(-5, 5)
		b := 1 + rand.Intn(6)
		c := 2 + rand.Intn(4)
		set := fmt.Sprintf("\\{%d + %d/n : n \\in \\mathbb{N}, n \\ge %d\\}", a, b, c)
		if wantSup {
			return generator.Problem{
				Question:    fmt.Sprintf("What is the supremum of \\(%s\\)? (enter a number or fraction)", set),
				Answer:      rs(a*c+b, c),
				Explanation: fmt.Sprintf("The elements decrease as \\(n\\) grows, so the largest is at \\(n = %d\\): \\(%d + %d/%d = %s\\), attained.", c, a, b, c, rs(a*c+b, c)),
			}
		}
		return generator.Problem{
			Question:    fmt.Sprintf("What is the infimum of \\(%s\\)? (enter a number)", set),
			Answer:      fmt.Sprint(a),
			Explanation: fmt.Sprintf("Every element exceeds %d and \\(%d/n \\to 0\\), so no larger number is a lower bound: the infimum is %d, not attained.", a, b, a),
		}
	case 2:
		m := 1 + rand.Intn(9)
		strict := rand.Intn(2) == 0
		rel, att := "<", "is not in the set"
		if !strict {
			rel, att = "\\le", "is in the set"
		}
		set := fmt.Sprintf("\\{x \\in \\mathbb{R} : x^{2} %s %d\\}", rel, m*m)
		ans := m
		if !wantSup {
			ans = -m
		}
		return generator.Problem{
			Question:    fmt.Sprintf("What is the %s of \\(%s\\)? (enter a number)", word, set),
			Answer:      fmt.Sprint(ans),
			Explanation: fmt.Sprintf("The set is the interval with endpoints −%d and %d, so its %s is %d, which %s.", m, m, word, ans, att),
		}
	default:
		a := 1 + rand.Intn(8)
		b := rand.Intn(a)
		set := fmt.Sprintf("\\{(n + %d)/(n + %d) : n \\in \\mathbb{N}\\}", a, b)
		if b == 0 {
			set = fmt.Sprintf("\\{(n + %d)/n : n \\in \\mathbb{N}\\}", a)
		}
		if wantSup {
			return generator.Problem{
				Question:    fmt.Sprintf("What is the supremum of \\(%s\\)? (enter a number or fraction)", set),
				Answer:      rs(1+a, 1+b),
				Explanation: fmt.Sprintf("\\((n + %d)/(n + %d) = 1 + %d/(n + %d)\\) decreases in \\(n\\), so the largest element is at \\(n = 1\\): %s.", a, b, a-b, b, rs(1+a, 1+b)),
			}
		}
		return generator.Problem{
			Question:    fmt.Sprintf("What is the infimum of \\(%s\\)? (enter a number)", set),
			Answer:      "1",
			Explanation: fmt.Sprintf("\\((n + %d)/(n + %d) = 1 + %d/(n + %d)\\) is above 1 and tends to 1, so the infimum is 1, not attained.", a, b, a-b, b),
		}
	}
}

// ---------------------------------------------------------------------------
// calc.limit.epsilon_delta and calc.limit.uniform_continuity (symbolic)
// ---------------------------------------------------------------------------

func epsilonDeltaVariant() generator.Problem {
	E := []int{10, 20, 50, 100, 1000}[rand.Intn(5)]
	switch rand.Intn(3) {
	case 0:
		m := []int{2, 3, 4, 5, 6, 7, -2, -3, -4, -5}[rand.Intn(10)]
		b := rand.Intn(11) - 5
		a := rand.Intn(7) - 3
		L := m*a + b
		f := texXY([]xyTerm{{m, 1, 0}, {b, 0, 0}})
		return generator.Problem{
			Question:    fmt.Sprintf("To prove \\(\\lim_{x \\to %d} (%s) = %d\\), what is the largest \\(\\delta\\) that works for \\(\\varepsilon = 1/%d\\)? (enter a fraction)", a, f, L, E),
			Answer:      rs(1, iabs(m)*E),
			Explanation: fmt.Sprintf("\\(|(%s) - %s| = %d|%s|\\), which is below \\(\\varepsilon\\) exactly when \\(|%s| < \\varepsilon/%d\\). So \\(\\delta = \\varepsilon/%d = %s\\).", f, paren(L), iabs(m), xShift("x", a), xShift("x", a), iabs(m), iabs(m), rs(1, iabs(m)*E)),
		}
	case 1:
		m := []int{2, 3, 4, 5, 6, 7, 8, 9, -2, -3, -6}[rand.Intn(11)]
		b := rand.Intn(11) - 5
		a := rand.Intn(7) - 3
		f := texXY([]xyTerm{{m, 1, 0}, {b, 0, 0}})
		return generator.Problem{
			Question:    fmt.Sprintf("In the \\(\\varepsilon\\)-\\(\\delta\\) proof that \\(\\lim_{x \\to %d} (%s) = %d\\), one takes \\(\\delta = \\varepsilon/K\\). What is the smallest \\(K\\) that works? (enter a number)", a, f, m*a+b),
			Answer:      fmt.Sprint(iabs(m)),
			Explanation: fmt.Sprintf("\\(|f(x) - L| = %d|%s|\\), so \\(|%s| < \\varepsilon/%d\\) is exactly what is needed: \\(K = %d\\).", iabs(m), xShift("x", a), xShift("x", a), iabs(m), iabs(m)),
		}
	default:
		a := []int{1, 2, 3, 4, 5, -1, -2, -3}[rand.Intn(8)]
		K := 2*iabs(a) + 1
		return generator.Problem{
			Question:    fmt.Sprintf("To prove \\(\\lim_{x \\to %d} x^{2} = %d\\), first require \\(\\delta \\le 1\\), then \\(\\delta = \\min(1, \\varepsilon/K)\\). What is the \\(K\\) obtained from the bound \\(|%s| \\le K\\)? (enter a number)", a, a*a, xShift("x", -a)),
			Answer:      fmt.Sprint(K),
			Explanation: fmt.Sprintf("\\(|x^{2} - %d| = |%s|\\,|%s|\\). If \\(|%s| < 1\\) then \\(|%s| \\le |%s| + %d < %d\\), so \\(K = %d\\) and \\(|x^{2} - %d| < K\\delta \\le \\varepsilon\\).", a*a, xShift("x", a), xShift("x", -a), xShift("x", a), xShift("x", -a), xShift("x", a), 2*iabs(a), K, K, a*a),
		}
	}
}

func uniformContinuityVariant() generator.Problem {
	E := []int{10, 20, 50, 100}[rand.Intn(4)]
	switch rand.Intn(4) {
	case 0:
		m := []int{2, 3, 4, 5, 7, -2, -3, -5}[rand.Intn(8)]
		b := rand.Intn(9) - 4
		f := texXY([]xyTerm{{m, 1, 0}, {b, 0, 0}})
		return generator.Problem{
			Question:    fmt.Sprintf("\\(f(x) = %s\\) is uniformly continuous on \\(\\mathbb{R}\\). What is the largest \\(\\delta\\) such that \\(|x - y| < \\delta\\) implies \\(|f(x) - f(y)| < 1/%d\\) for all \\(x, y\\)? (enter a fraction)", f, E),
			Answer:      rs(1, iabs(m)*E),
			Explanation: fmt.Sprintf("\\(|f(x) - f(y)| = %d|x - y|\\), the same at every point, so \\(\\delta = (1/%d)/%d = %s\\) works everywhere and no larger one does.", iabs(m), E, iabs(m), rs(1, iabs(m)*E)),
		}
	case 1:
		a := 1 + rand.Intn(6)
		return generator.Problem{
			Question:    fmt.Sprintf("On \\([0, %d]\\), \\(f(x) = x^{2}\\) satisfies \\(|f(x) - f(y)| \\le K|x - y|\\). What is the smallest such \\(K\\)? (enter a number)", a),
			Answer:      fmt.Sprint(2 * a),
			Explanation: fmt.Sprintf("\\(|x^{2} - y^{2}| = |x + y|\\,|x - y| \\le %d|x - y|\\) on \\([0, %d]\\), with \\(|x + y|\\) approaching %d near the right end, so \\(K = %d\\) and \\(\\delta = \\varepsilon/%d\\) gives uniform continuity there.", 2*a, a, 2*a, 2*a, 2*a),
		}
	case 2:
		a := 1 + rand.Intn(4)
		return generator.Problem{
			Question:    fmt.Sprintf("On \\([0, %d]\\), \\(f(x) = x^{3}\\) satisfies \\(|f(x) - f(y)| \\le K|x - y|\\). What is the smallest such \\(K\\)? (enter a number)", a),
			Answer:      fmt.Sprint(3 * a * a),
			Explanation: fmt.Sprintf("By the mean value theorem \\(|x^{3} - y^{3}| = 3c^{2}|x - y|\\) for some \\(c\\) between them, and \\(3c^{2} \\le %d\\) on \\([0, %d]\\) with values arbitrarily close to it, so \\(K = %d\\).", 3*a*a, a, 3*a*a),
		}
	default:
		k := []int{2, 3, 4, 5, -2, -3}[rand.Intn(6)]
		return generator.Problem{
			Question:    fmt.Sprintf("For \\(f(x) = %s\\sin x\\) on \\(\\mathbb{R}\\), \\(|f(x) - f(y)| \\le %d|x - y|\\). Using this bound, what \\(\\delta\\) guarantees \\(|f(x) - f(y)| < 1/%d\\) whenever \\(|x - y| < \\delta\\)? (enter a fraction)", coefStr(k), iabs(k), E),
			Answer:      rs(1, iabs(k)*E),
			Explanation: fmt.Sprintf("\\(|\\sin x - \\sin y| \\le |x - y|\\), so \\(|f(x) - f(y)| \\le %d|x - y| < %d\\delta\\). Taking \\(\\delta = (1/%d)/%d = %s\\) works at every point, which is uniform continuity.", iabs(k), iabs(k), E, iabs(k), rs(1, iabs(k)*E)),
		}
	}
}

// ---------------------------------------------------------------------------
// Sequences: calc.seq.bounded, cauchy, convergence, monotone, function_sequences
// ---------------------------------------------------------------------------

func boundedSeqVariant() generator.Problem {
	upper := rand.Intn(2) == 0
	word, what := "least upper bound (supremum)", "supremum"
	if !upper {
		word, what = "greatest lower bound (infimum)", "infimum"
	}
	switch rand.Intn(3) {
	case 0:
		c := nonzero(-4, 4)
		k := 1 + rand.Intn(8)
		seq := fmt.Sprintf("%d + (-1)^{n}\\frac{%d}{n}", c, k)
		if upper {
			return generator.Problem{
				Question:    fmt.Sprintf("What is the %s of \\(a_{n} = %s\\), \\(n \\ge 1\\)? (enter a number or fraction)", word, seq),
				Answer:      rs(2*c+k, 2),
				Explanation: fmt.Sprintf("Odd terms are below %d; even terms \\(%d + %d/n\\) decrease, so the largest term is \\(a_{2} = %d + %d/2 = %s\\).", c, c, k, c, k, rs(2*c+k, 2)),
			}
		}
		return generator.Problem{
			Question:    fmt.Sprintf("What is the %s of \\(a_{n} = %s\\), \\(n \\ge 1\\)? (enter a number)", word, seq),
			Answer:      fmt.Sprint(c - k),
			Explanation: fmt.Sprintf("Even terms are above %d; odd terms \\(%d - %d/n\\) increase, so the smallest term is \\(a_{1} = %d - %d = %d\\).", c, c, k, c, k, c-k),
		}
	case 1:
		k := 1 + rand.Intn(9)
		seq := fmt.Sprintf("\\frac{%dn}{n + 1}", k)
		ans, why := fmt.Sprint(k), fmt.Sprintf("\\(%dn/(n + 1) = %d - %d/(n + 1)\\) increases towards %d without reaching it, so the %s is %d.", k, k, k, k, what, k)
		if !upper {
			ans, why = rs(k, 2), fmt.Sprintf("The terms increase, so the smallest is \\(a_{1} = %d/2\\), which is the %s: %s.", k, what, rs(k, 2))
		}
		return generator.Problem{
			Question:    fmt.Sprintf("What is the %s of \\(a_{n} = %s\\), \\(n \\ge 1\\)? (enter a number or fraction)", word, seq),
			Answer:      ans,
			Explanation: why,
		}
	default:
		a := 1 + rand.Intn(5)
		b := a + 1 + rand.Intn(8)
		seq := fmt.Sprintf("\\frac{%sn + %d}{n + 1}", coefStr(a), b)
		ans, why := rs(a+b, 2), fmt.Sprintf("\\(a_{n} = %d + %d/(n + 1)\\) decreases, so its largest term is \\(a_{1} = %s\\).", a, b-a, rs(a+b, 2))
		if !upper {
			ans, why = fmt.Sprint(a), fmt.Sprintf("\\(a_{n} = %d + %d/(n + 1)\\) decreases towards %d without reaching it, so the %s is %d.", a, b-a, a, what, a)
		}
		return generator.Problem{
			Question:    fmt.Sprintf("What is the %s of \\(a_{n} = %s\\), \\(n \\ge 1\\)? (enter a number or fraction)", word, seq),
			Answer:      ans,
			Explanation: why,
		}
	}
}

func cauchySeqVariant() generator.Problem {
	E := []int{10, 20, 50, 100}[rand.Intn(4)]
	switch rand.Intn(3) {
	case 0:
		k := 1 + rand.Intn(9)
		return generator.Problem{
			Question:    fmt.Sprintf("For \\(a_{n} = %d/n\\), if \\(m, n \\ge N\\) then \\(|a_{m} - a_{n}| < %d/N\\). What is the smallest \\(N\\) with \\(%d/N < 1/%d\\), which makes the terms past \\(N\\) lie within \\(1/%d\\) of each other? (enter a whole number)", k, k, k, E, E),
			Answer:      fmt.Sprint(k*E + 1),
			Explanation: fmt.Sprintf("Both terms lie in \\((0, %d/N]\\), so their distance is below \\(%d/N\\). \\(%d/N < 1/%d\\) means \\(N > %d\\), so the smallest \\(N\\) is %d.", k, k, k, E, k*E, k*E+1),
		}
	case 1:
		q := 2 + rand.Intn(4)
		N := 0
		p := 1
		for p <= E {
			p *= q
			N++
		}
		return generator.Problem{
			Question:    fmt.Sprintf("For \\(a_{n} = 1/%d^{n}\\), \\(|a_{m} - a_{n}| < 1/%d^{N}\\) whenever \\(m, n \\ge N\\). What is the smallest \\(N\\) with \\(1/%d^{N} < 1/%d\\)? (enter a whole number)", q, q, q, E),
			Answer:      fmt.Sprint(N),
			Explanation: fmt.Sprintf("We need \\(%d^{N} > %d\\): \\(%d^{%d} = %d\\) is the first power above %d, so \\(N = %d\\).", q, E, q, N, p, E, N),
		}
	default:
		c := 1 + rand.Intn(9)
		return generator.Problem{
			Question:    fmt.Sprintf("For \\(a_{n} = %d(-1)^{n}\\), what is \\(|a_{n+1} - a_{n}|\\) for every \\(n\\)? (enter a number)", c),
			Answer:      fmt.Sprint(2 * c),
			Explanation: fmt.Sprintf("Consecutive terms are %d and −%d in some order, so they are always %d apart. For \\(\\varepsilon < %d\\) no \\(N\\) works, so the sequence is not Cauchy (and does not converge).", c, c, 2*c, 2*c),
		}
	}
}

func seqConvergenceVariant() generator.Problem {
	const ask = "(answer: yes, <limit> or no)"
	switch rand.Intn(4) {
	case 0:
		a := 1 + rand.Intn(9)
		b := rand.Intn(9) + 1
		c := 1 + rand.Intn(5)
		d := 1 + rand.Intn(9)
		L := rs(a, c)
		return generator.Problem{
			Question:    fmt.Sprintf("Does \\(a_{n} = \\frac{%sn + %d}{%sn + %d}\\) converge? If so, to what? %s", coefStr(a), b, coefStr(c), d, ask),
			Answer:      "yes, " + L,
			Explanation: fmt.Sprintf("Dividing top and bottom by \\(n\\): \\((%d + %d/n)/(%d + %d/n) \\to %s\\).", a, b, c, d, quotTeX(a, c)),
		}
	case 1:
		a := 1 + rand.Intn(5)
		b := rand.Intn(9) + 1
		c := 1 + rand.Intn(5)
		d := 1 + rand.Intn(9)
		return generator.Problem{
			Question:    fmt.Sprintf("Does \\(a_{n} = \\frac{%sn^{2} + %d}{%sn + %d}\\) converge? If so, to what? %s", coefStr(a), b, coefStr(c), d, ask),
			Answer:      "no",
			Explanation: fmt.Sprintf("Dividing by \\(n\\): \\((%dn + %d/n)/(%d + %d/n)\\), whose numerator grows without bound, so \\(a_{n} \\to \\infty\\) and the sequence diverges.", a, b, c, d),
		}
	case 2:
		k := nonzero(-5, 5)
		m := 1 + rand.Intn(5)
		return generator.Problem{
			Question:    fmt.Sprintf("Does \\(a_{n} = %d + \\frac{%d(-1)^{n}}{n}\\) converge? If so, to what? %s", k, m, ask),
			Answer:      fmt.Sprintf("yes, %d", k),
			Explanation: fmt.Sprintf("\\(|a_{n} - %s| = %d/n \\to 0\\), so the sequence converges to %d even though it oscillates around it.", paren(k), m, k),
		}
	default:
		p := 1 + rand.Intn(6)
		q := 1 + rand.Intn(6)
		for q == p {
			q = 1 + rand.Intn(6)
		}
		if p < q {
			return generator.Problem{
				Question:    fmt.Sprintf("Does \\(a_{n} = (%d/%d)^{n}\\) converge? If so, to what? %s", p, q, ask),
				Answer:      "yes, 0",
				Explanation: fmt.Sprintf("The ratio \\(%d/%d\\) lies in \\((0, 1)\\), so its powers tend to 0.", p, q),
			}
		}
		return generator.Problem{
			Question:    fmt.Sprintf("Does \\(a_{n} = (%d/%d)^{n}\\) converge? If so, to what? %s", p, q, ask),
			Answer:      "no",
			Explanation: fmt.Sprintf("The ratio \\(%d/%d\\) exceeds 1, so its powers grow without bound.", p, q),
		}
	}
}

func monotoneSeqVariant() generator.Problem {
	switch rand.Intn(3) {
	case 0:
		k := 2 + rand.Intn(8)
		c := 1 + rand.Intn(12)
		L := rs(c, k-1)
		return generator.Problem{
			Question:    fmt.Sprintf("The sequence \\(x_{1} = 0\\), \\(x_{n+1} = (x_{n} + %d)/%d\\) is increasing and bounded above, so it converges. What is its limit? (enter a number or fraction)", c, k),
			Answer:      L,
			Explanation: fmt.Sprintf("By the monotone convergence theorem the limit \\(L\\) exists, and passing to the limit in the recursion gives \\(L = (L + %d)/%d\\), so \\(%dL = %d\\) and \\(L = %s\\).", c, k, k-1, c, L),
		}
	case 1:
		L := 2 + rand.Intn(6)
		c := L*L - L
		return generator.Problem{
			Question:    fmt.Sprintf("The sequence \\(x_{1} = 0\\), \\(x_{n+1} = \\sqrt{%d + x_{n}}\\) is increasing and bounded above by %d, so it converges. What is its limit? (enter a number)", c, L),
			Answer:      fmt.Sprint(L),
			Explanation: fmt.Sprintf("The limit satisfies \\(L = \\sqrt{%d + L}\\), so \\(L^{2} - L - %d = 0\\), that is \\((L - %d)(L + %d) = 0\\). The terms are nonnegative, so \\(L = %d\\).", c, c, L, L-1, L),
		}
	default:
		k := 2 + rand.Intn(8)
		m := 1 + rand.Intn(6)
		return generator.Problem{
			Question:    fmt.Sprintf("\\(a_{n} = %d - %d/n\\) is increasing. What is the smallest upper bound of the sequence, which is also its limit? (enter a number)", k, m),
			Answer:      fmt.Sprint(k),
			Explanation: fmt.Sprintf("\\(a_{n+1} - a_{n} = %d/n - %d/(n+1) > 0\\), every term is below %d, and \\(%d/n \\to 0\\), so the supremum and the limit are both %d.", m, m, k, m, k),
		}
	}
}

func functionSequencesVariant() generator.Problem {
	switch rand.Intn(3) {
	case 0:
		k := 1 + rand.Intn(9)
		E := []int{10, 20, 100}[rand.Intn(3)]
		return generator.Problem{
			Question:    fmt.Sprintf("\\(f_{n}(x) = x/n\\) converges uniformly to 0 on \\([0, %d]\\). What is the smallest \\(n\\) with \\(\\sup_{x \\in [0, %d]} |f_{n}(x)| < 1/%d\\)? (enter a whole number)", k, k, E),
			Answer:      fmt.Sprint(k*E + 1),
			Explanation: fmt.Sprintf("The supremum is \\(%d/n\\), attained at \\(x = %d\\). \\(%d/n < 1/%d\\) means \\(n > %d\\), so \\(n = %d\\).", k, k, k, E, k*E, k*E+1),
		}
	case 1:
		p := 1 + rand.Intn(4)
		q := p + 1 + rand.Intn(4)
		n := 2 + rand.Intn(3)
		num, den := ipow(p, n), ipow(q, n)
		return generator.Problem{
			Question:    fmt.Sprintf("On \\([0, %d/%d]\\), what is \\(\\sup_{x} |f_{%d}(x) - f(x)|\\) for \\(f_{n}(x) = x^{n}\\) and its pointwise limit \\(f = 0\\)? (enter a fraction)", p, q, n),
			Answer:      rs(num, den),
			Explanation: fmt.Sprintf("\\(x^{%d}\\) increases on the interval, so the supremum is at the right end: \\((%d/%d)^{%d} = %s\\). These suprema tend to 0, so the convergence is uniform on \\([0, %d/%d]\\) (unlike on \\([0, 1]\\)).", n, p, q, n, rs(num, den), p, q),
		}
	default:
		k := 1 + rand.Intn(9)
		n := 1 + rand.Intn(6)
		return generator.Problem{
			Question:    fmt.Sprintf("For \\(f_{n}(x) = \\frac{%dx}{1 + n^{2}x^{2}}\\) on \\([0, \\infty)\\), what is \\(\\sup_{x \\ge 0} f_{%d}(x)\\)? (enter a number or fraction)", k, n),
			Answer:      rs(k, 2*n),
			Explanation: fmt.Sprintf("\\(f_{n}'(x) = %d(1 - n^{2}x^{2})/(1 + n^{2}x^{2})^{2}\\) vanishes at \\(x = 1/n\\), where \\(f_{n} = %d/(2n)\\). For \\(n = %d\\) that is %s, and since \\(%d/(2n) \\to 0\\) the sequence converges uniformly to 0.", k, k, n, rs(k, 2*n), k),
		}
	}
}

// ---------------------------------------------------------------------------
// Series: function_series, radius, absolute, alternating, fourier, direct_compare
// ---------------------------------------------------------------------------

func functionSeriesVariant() generator.Problem {
	switch rand.Intn(3) {
	case 0:
		q := 2 + rand.Intn(7)
		p := rand.Intn(2*q-1) - (q - 1)
		if p == 0 {
			p = 1
		}
		return generator.Problem{
			Question:    fmt.Sprintf("The series \\(\\sum_{n \\ge 0} x^{n}\\) converges pointwise on \\((-1, 1)\\). What is its sum at \\(x = %s\\)? (enter a fraction)", rs(p, q)),
			Answer:      rs(q, q-p),
			Explanation: fmt.Sprintf("Its sum is \\(1/(1 - x)\\), and \\(1/(1 - (%s)) = %s\\).", rs(p, q), rs(q, q-p)),
		}
	case 1:
		a := 2 + rand.Intn(7)
		b := 1 + rand.Intn(a-1)
		return generator.Problem{
			Question:    fmt.Sprintf("On \\([-%d, %d]\\), the terms of \\(\\sum_{n \\ge 0} x^{n}/%d^{n}\\) satisfy \\(|x^{n}/%d^{n}| \\le M_{n} = (%d/%d)^{n}\\). What is \\(\\sum_{n \\ge 0} M_{n}\\), the bound the Weierstrass M-test uses? (enter a fraction)", b, b, a, a, b, a),
			Answer:      rs(a, a-b),
			Explanation: fmt.Sprintf("\\(\\sum (%d/%d)^{n}\\) is geometric with ratio below 1, so it converges to \\(1/(1 - %s) = %s\\). By the M-test the series converges uniformly on \\([-%d, %d]\\).", b, a, rs(b, a), rs(a, a-b), b, b),
		}
	default:
		k := 2 + rand.Intn(7)
		j := rand.Intn(2*k-1) - (k - 1)
		return generator.Problem{
			Question:    fmt.Sprintf("\\(\\sum_{n \\ge 0} (x/%d)^{n}\\) converges for \\(|x| < %d\\). What is its sum at \\(x = %d\\)? (enter a number or fraction)", k, k, j),
			Answer:      rs(k, k-j),
			Explanation: fmt.Sprintf("It is geometric with ratio \\(x/%d\\), so the sum is \\(1/(1 - x/%d) = %d/(%d - x)\\), which at \\(x = %d\\) is %s.", k, k, k, k, j, rs(k, k-j)),
		}
	}
}

func radiusVariant() generator.Problem {
	k := 2 + rand.Intn(8)
	switch rand.Intn(5) {
	case 0:
		return generator.Problem{
			Question:    fmt.Sprintf("What is the radius of convergence of \\(\\sum_{n \\ge 0} x^{n}/%d^{n}\\)?", k),
			Answer:      fmt.Sprint(k),
			Explanation: fmt.Sprintf("Ratio test: \\(|a_{n+1}/a_{n}| = |x|/%d < 1\\) exactly when \\(|x| < %d\\), so \\(R = %d\\).", k, k, k),
		}
	case 1:
		return generator.Problem{
			Question:    fmt.Sprintf("What is the radius of convergence of \\(\\sum_{n \\ge 0} %d^{n}x^{n}\\)? (enter a fraction)", k),
			Answer:      rs(1, k),
			Explanation: fmt.Sprintf("Root test: \\(\\sqrt[n]{|%d^{n}x^{n}|} = %d|x| < 1\\) exactly when \\(|x| < 1/%d\\), so \\(R = %s\\).", k, k, k, rs(1, k)),
		}
	case 2:
		p := 1 + rand.Intn(3)
		return generator.Problem{
			Question:    fmt.Sprintf("What is the radius of convergence of \\(\\sum_{n \\ge 1} n^{%d}x^{n}/%d^{n}\\)?", p, k),
			Answer:      fmt.Sprint(k),
			Explanation: fmt.Sprintf("Ratio test: \\(\\left(\\frac{n+1}{n}\\right)^{%d}\\frac{|x|}{%d} \\to |x|/%d\\), so the polynomial factor does not change the radius: \\(R = %d\\).", p, k, k, k),
		}
	case 3:
		a := rand.Intn(9) - 4
		return generator.Problem{
			Question:    fmt.Sprintf("What is the radius of convergence of \\(\\sum_{n \\ge 1} \\frac{(%s)^{n}}{n \\cdot %d^{n}}\\)?", xShift("x", a), k),
			Answer:      fmt.Sprint(k),
			Explanation: fmt.Sprintf("Ratio test: \\(\\frac{n}{n+1}\\cdot\\frac{|%s|}{%d} \\to |%s|/%d\\), so the series converges for \\(|%s| < %d\\): \\(R = %d\\) (the centre %d does not affect the radius).", xShift("x", a), k, xShift("x", a), k, xShift("x", a), k, k, a),
		}
	default:
		m := 2 + rand.Intn(5)
		return generator.Problem{
			Question:    fmt.Sprintf("What is the radius of convergence of \\(\\sum_{n \\ge 0} x^{2n}/%d^{n}\\)?", m*m),
			Answer:      fmt.Sprint(m),
			Explanation: fmt.Sprintf("It is geometric in \\(x^{2}/%d\\), which converges when \\(x^{2} < %d\\), that is \\(|x| < %d\\): \\(R = %d\\).", m*m, m*m, m, m),
		}
	}
}

func absoluteVariant() generator.Problem {
	const ask = "(type absolute, conditional or divergent)"
	switch rand.Intn(5) {
	case 0:
		ps := []struct {
			tex    string
			num    int
			den    int
			answer string
		}{{"1/2", 1, 2, "conditional"}, {"1/3", 1, 3, "conditional"}, {"1", 1, 1, "conditional"}, {"3/2", 3, 2, "absolute"}, {"2", 2, 1, "absolute"}, {"3", 3, 1, "absolute"}, {"5/4", 5, 4, "absolute"}}
		p := ps[rand.Intn(len(ps))]
		why := fmt.Sprintf("\\(\\sum 1/n^{%s}\\) is a p-series with \\(p = %s > 1\\), so the series converges absolutely.", p.tex, p.tex)
		if p.answer == "conditional" {
			why = fmt.Sprintf("\\(1/n^{%s}\\) decreases to 0, so the alternating series converges, but \\(\\sum 1/n^{%s}\\) is a p-series with \\(p = %s \\le 1\\) and diverges: the convergence is conditional.", p.tex, p.tex, p.tex)
		}
		return generator.Problem{
			Question:    fmt.Sprintf("How does \\(\\sum_{n \\ge 1} (-1)^{n}/n^{%s}\\) converge? %s", p.tex, ask),
			Answer:      p.answer,
			Explanation: why,
		}
	case 1:
		k := 2 + rand.Intn(7)
		j := 1 + rand.Intn(9)
		return generator.Problem{
			Question:    fmt.Sprintf("How does \\(\\sum_{n \\ge 1} \\frac{(-1)^{n}}{%dn + %d}\\) converge? %s", k, j, ask),
			Answer:      "conditional",
			Explanation: fmt.Sprintf("The terms \\(1/(%dn + %d)\\) decrease to 0, so the alternating series converges; but they are comparable to \\(1/n\\) (the ratio tends to \\(1/%d\\)), so the absolute series diverges.", k, j, k),
		}
	case 2:
		k := 1 + rand.Intn(9)
		return generator.Problem{
			Question:    fmt.Sprintf("How does \\(\\sum_{n \\ge 1} (-1)^{n}\\frac{n}{n + %d}\\) converge? %s", k, ask),
			Answer:      "divergent",
			Explanation: fmt.Sprintf("\\(n/(n + %d) \\to 1\\), so the terms do not tend to 0 and the series diverges by the n-th term test.", k),
		}
	case 3:
		k := 2 + rand.Intn(8)
		return generator.Problem{
			Question:    fmt.Sprintf("How does \\(\\sum_{n \\ge 0} (-1)^{n}/%d^{n}\\) converge? %s", k, ask),
			Answer:      "absolute",
			Explanation: fmt.Sprintf("\\(\\sum (1/%d)^{n}\\) is geometric with ratio \\(1/%d < 1\\), so the series converges absolutely.", k, k),
		}
	default:
		k := 1 + rand.Intn(9)
		return generator.Problem{
			Question:    fmt.Sprintf("How does \\(\\sum_{n \\ge 1} \\frac{\\cos(%dn)}{n^{2}}\\) converge? %s", k, ask),
			Answer:      "absolute",
			Explanation: fmt.Sprintf("\\(|\\cos(%dn)/n^{2}| \\le 1/n^{2}\\) and \\(\\sum 1/n^{2}\\) converges, so the series converges absolutely by comparison.", k),
		}
	}
}

func alternatingVariant() generator.Problem {
	switch rand.Intn(3) {
	case 0:
		p := 1 + rand.Intn(3)
		N := 3 + rand.Intn(10)
		den := ipow(N+1, p)
		pw := ""
		if p > 1 {
			pw = fmt.Sprintf("^{%d}", p)
		}
		return generator.Problem{
			Question:    fmt.Sprintf("The sum of \\(\\sum_{n \\ge 1} (-1)^{n+1}/n%s\\) is estimated by its first %d terms. By the alternating series estimate, what is the bound on the error? (enter a fraction)", pw, N),
			Answer:      rs(1, den),
			Explanation: fmt.Sprintf("The terms decrease to 0, so the error is at most the first omitted term, \\(1/%d%s = %s\\).", N+1, pw, rs(1, den)),
		}
	case 1:
		p := 1 + rand.Intn(2)
		E := []int{10, 20, 50, 100, 1000}[rand.Intn(5)]
		N := 0
		for ipow(N+1, p) <= E {
			N++
		}
		pw := ""
		if p > 1 {
			pw = "^{2}"
		}
		return generator.Problem{
			Question:    fmt.Sprintf("How many terms of \\(\\sum_{n \\ge 1} (-1)^{n+1}/n%s\\) does the alternating series estimate require so that the error is below \\(1/%d\\)? (enter a whole number)", pw, E),
			Answer:      fmt.Sprint(N),
			Explanation: fmt.Sprintf("After \\(N\\) terms the error is at most \\(1/(N+1)%s\\), which is below \\(1/%d\\) once \\((N+1)%s > %d\\). The first such \\(N\\) is %d.", pw, E, pw, E, N),
		}
	default:
		N := 2 + rand.Intn(5)
		s := new(big.Rat)
		var terms []string
		for n := 1; n <= N; n++ {
			t := big.NewRat(1, int64(n))
			if n%2 == 0 {
				t.Neg(t)
				terms = append(terms, fmt.Sprintf("- 1/%d", n))
			} else if n == 1 {
				terms = append(terms, "1")
			} else {
				terms = append(terms, fmt.Sprintf("+ 1/%d", n))
			}
			s.Add(s, t)
		}
		return generator.Problem{
			Question:    fmt.Sprintf("What is the partial sum \\(S_{%d}\\) of \\(\\sum_{n \\ge 1} (-1)^{n+1}/n\\)? (enter a fraction)", N),
			Answer:      s.RatString(),
			Explanation: fmt.Sprintf("\\(S_{%d} = %s = %s\\). The full sum is \\(\\ln 2\\), and \\(S_{%d}\\) differs from it by less than \\(1/%d\\).", N, strings.Join(terms, " "), s.RatString(), N, N+1),
		}
	}
}

func fourierVariant() generator.Problem {
	const conv = "with \\(a_{n} = \\frac{1}{\\pi}\\int_{-\\pi}^{\\pi} f(x)\\cos(nx)\\,dx\\) and \\(b_{n} = \\frac{1}{\\pi}\\int_{-\\pi}^{\\pi} f(x)\\sin(nx)\\,dx\\)"
	n := 1 + rand.Intn(6)
	switch rand.Intn(4) {
	case 0:
		a := 1 + rand.Intn(5)
		b := []int{1, 2, 3, -1, -2}[rand.Intn(5)]
		c := []int{1, 2, 3, -1, -3}[rand.Intn(5)]
		k := 1 + rand.Intn(4)
		m := 1 + rand.Intn(4)
		f := fmt.Sprintf("%d + %s\\cos(%dx) + %s\\sin(%dx)", a, coefStr(b), k, coefStr(c), m)
		switch rand.Intn(3) {
		case 0:
			return generator.Problem{
				Question:    fmt.Sprintf("For \\(f(x) = %s\\) on \\([-\\pi, \\pi]\\), %s, what is \\(a_{0}\\)?", f, conv),
				Answer:      fmt.Sprint(2 * a),
				Explanation: fmt.Sprintf("The cosine and sine terms integrate to 0 over a period, so \\(a_{0} = \\frac{1}{\\pi}\\cdot %d \\cdot 2\\pi = %d\\), twice the average value.", a, 2*a),
			}
		case 1:
			return generator.Problem{
				Question:    fmt.Sprintf("For \\(f(x) = %s\\) on \\([-\\pi, \\pi]\\), %s, what is \\(a_{%d}\\)?", f, conv, k),
				Answer:      fmt.Sprint(b),
				Explanation: fmt.Sprintf("By orthogonality only the \\(\\cos(%dx)\\) term contributes, and \\(\\frac{1}{\\pi}\\int_{-\\pi}^{\\pi} \\cos^{2}(%dx)\\,dx = 1\\), so \\(a_{%d} = %d\\).", k, k, k, b),
			}
		default:
			return generator.Problem{
				Question:    fmt.Sprintf("For \\(f(x) = %s\\) on \\([-\\pi, \\pi]\\), %s, what is \\(b_{%d}\\)?", f, conv, m),
				Answer:      fmt.Sprint(c),
				Explanation: fmt.Sprintf("By orthogonality only the \\(\\sin(%dx)\\) term contributes, and \\(\\frac{1}{\\pi}\\int_{-\\pi}^{\\pi} \\sin^{2}(%dx)\\,dx = 1\\), so \\(b_{%d} = %d\\).", m, m, m, c),
			}
		}
	case 1:
		num := 2
		if n%2 == 0 {
			num = -2
		}
		return generator.Problem{
			Question:    fmt.Sprintf("For \\(f(x) = x\\) on \\((-\\pi, \\pi)\\), %s, what is \\(b_{%d}\\)? (enter a number or fraction)", conv, n),
			Answer:      rs(num, n),
			Explanation: fmt.Sprintf("Integrating by parts, \\(b_{n} = \\frac{2(-1)^{n+1}}{n}\\); for \\(n = %d\\) this is %s. (All \\(a_{n}\\) are 0 because \\(x\\) is odd.)", n, rs(num, n)),
		}
	case 2:
		ans, why := "0", fmt.Sprintf("\\(b_{n} = \\frac{2(1 - (-1)^{n})}{n\\pi}\\), which is 0 for even \\(n\\) such as %d.", n)
		if n%2 == 1 {
			ans = fmt.Sprintf("4/(%d*pi)", n)
			if n == 1 {
				ans = "4/pi"
			}
			why = fmt.Sprintf("\\(b_{n} = \\frac{2(1 - (-1)^{n})}{n\\pi}\\), which for odd \\(n = %d\\) is \\(\\frac{4}{%d\\pi}\\).", n, n)
		}
		return generator.Problem{
			Question:    fmt.Sprintf("Let \\(f(x) = -1\\) on \\((-\\pi, 0)\\) and \\(f(x) = 1\\) on \\((0, \\pi)\\), %s. What is \\(b_{%d}\\)? (enter in terms of pi)", conv, n),
			Answer:      ans,
			Explanation: why + " The function is odd, so every \\(a_{n}\\) is 0.",
		}
	default:
		num := 4
		if n%2 == 1 {
			num = -4
		}
		return generator.Problem{
			Question:    fmt.Sprintf("For \\(f(x) = x^{2}\\) on \\([-\\pi, \\pi]\\), %s, what is \\(a_{%d}\\)? (enter a number or fraction)", conv, n),
			Answer:      rs(num, n*n),
			Explanation: fmt.Sprintf("Integrating by parts twice, \\(a_{n} = \\frac{4(-1)^{n}}{n^{2}}\\) for \\(n \\ge 1\\); for \\(n = %d\\) this is %s. (All \\(b_{n}\\) are 0 because \\(x^{2}\\) is even.)", n, rs(num, n*n)),
		}
	}
}

// Grade accepts the chosen option's text as written. The concept is graded as
// symbolic, and the symbolic grader rejects a sentence with inline maths in it
// ("Input contains disallowed characters") even when it is the key itself;
// anything else goes to that grader as before.
func (g *directCompareGen) Grade(expected, userAnswer string) grader.Result {
	if strings.EqualFold(strings.TrimSpace(expected), strings.TrimSpace(userAnswer)) {
		return grader.Result{Correct: true, Score: 1}
	}
	return grader.NewRouter().Grade(grader.GradingSymbolic, expected, userAnswer)
}

// directCompareVariant asks which direct comparison settles a concrete series.
func directCompareVariant() generator.Problem {
	type fam struct {
		series, correct, why string
		wrong                []string
	}
	k := 1 + rand.Intn(9)
	p := 2 + rand.Intn(3)
	var f fam
	switch rand.Intn(6) {
	case 0:
		f = fam{
			series:  fmt.Sprintf("\\frac{1}{n^{%d} + %d}", p, k),
			correct: fmt.Sprintf("converges: its terms are below \\(1/n^{%d}\\), whose series converges", p),
			why:     fmt.Sprintf("\\(0 < 1/(n^{%d} + %d) < 1/n^{%d}\\), and \\(\\sum 1/n^{%d}\\) is a p-series with \\(p = %d > 1\\).", p, k, p, p, p),
			wrong: []string{fmt.Sprintf("diverges: its terms are above \\(1/n^{%d}\\), whose series diverges", p),
				"converges: its terms are below \\(1/n\\), whose series converges",
				fmt.Sprintf("diverges: its terms are above \\(1/(n + %d)\\), whose series diverges", k)},
		}
	case 1:
		f = fam{
			series:  fmt.Sprintf("\\frac{n + %d}{n^{2}}", k),
			correct: "diverges: its terms are above \\(1/n\\), whose series diverges",
			why:     fmt.Sprintf("\\((n + %d)/n^{2} = 1/n + %d/n^{2} > 1/n\\), and the harmonic series diverges, so the larger series diverges too.", k, k),
			wrong: []string{"converges: its terms are below \\(1/n^{2}\\), whose series converges",
				fmt.Sprintf("converges: its terms are below \\(%d/n^{2}\\), whose series converges", k+1),
				"diverges: its terms are above \\(1/n^{2}\\), whose series converges"},
		}
	case 2:
		f = fam{
			series:  fmt.Sprintf("\\frac{\\sin^{2}(%dn)}{n^{%d}}", k, p),
			correct: fmt.Sprintf("converges: its terms lie between 0 and \\(1/n^{%d}\\), whose series converges", p),
			why:     fmt.Sprintf("\\(0 \\le \\sin^{2}(%dn)/n^{%d} \\le 1/n^{%d}\\), and \\(\\sum 1/n^{%d}\\) converges because \\(%d > 1\\).", k, p, p, p, p),
			wrong: []string{"diverges: \\(\\sin^{2}\\) does not tend to 0, so neither do the terms",
				fmt.Sprintf("diverges: its terms are above \\(1/n^{%d}\\), whose series diverges", p),
				"no comparison applies, because the terms are sometimes 0"},
		}
	case 3:
		b := 2 + rand.Intn(8)
		f = fam{
			series:  fmt.Sprintf("\\frac{1}{%d^{n} + n}", b),
			correct: fmt.Sprintf("converges: its terms are below \\((1/%d)^{n}\\), a convergent geometric series", b),
			why:     fmt.Sprintf("\\(0 < 1/(%d^{n} + n) < 1/%d^{n}\\), and \\(\\sum (1/%d)^{n}\\) is geometric with ratio below 1.", b, b, b),
			wrong: []string{fmt.Sprintf("diverges: its terms are above \\(1/(%dn)\\), whose series diverges", b),
				"converges: its terms are below \\(1/n\\), whose series converges",
				fmt.Sprintf("diverges: its terms are above \\((1/%d)^{n}\\), a divergent series", b)},
		}
	case 4:
		f = fam{
			series:  fmt.Sprintf("\\frac{\\ln(n + %d)}{n}", k+1),
			correct: "diverges: its terms are at least \\(1/n\\), whose series diverges",
			why:     fmt.Sprintf("For \\(n \\ge 1\\), \\(n + %d \\ge 3 > e\\), so \\(\\ln(n + %d) > 1\\) and the terms exceed \\(1/n\\); the harmonic series diverges, so this series diverges too.", k+1, k+1),
			wrong: []string{"converges: its terms are below \\(1/n^{2}\\), whose series converges",
				"converges: \\(\\ln(n)/n \\to 0\\), so the series converges",
				"diverges: its terms are above \\(1/n^{2}\\), whose series converges"},
		}
	default:
		f = fam{
			series:  fmt.Sprintf("\\frac{%d + \\cos n}{n^{%d}}", k+1, p),
			correct: fmt.Sprintf("converges: its terms lie between 0 and \\(%d/n^{%d}\\), whose series converges", k+2, p),
			why:     fmt.Sprintf("\\(0 < (%d + \\cos n)/n^{%d} \\le %d/n^{%d}\\), and a constant times a convergent p-series (\\(p = %d > 1\\)) converges.", k+1, p, k+2, p, p),
			wrong: []string{fmt.Sprintf("diverges: its terms are above \\(%d/n\\), whose series diverges", k),
				"diverges: \\(\\cos n\\) has no limit, so the terms do not tend to 0",
				"no comparison applies, because \\(\\cos n\\) changes sign"},
		}
	}
	q := fmt.Sprintf("Use direct comparison to decide whether \\(\\sum_{n \\ge 1} %s\\) converges. Which conclusion and reason are correct?", f.series)
	return mcWith(q, f.correct, f.why, f.wrong...)
}
