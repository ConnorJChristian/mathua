package calculus

// Regression tests for the 2026-10-25 QA run (run 5). Each test names the
// defect it guards; the ones marked "fails on the old text" were checked to
// fail before the fix.

import (
	"fmt"
	"math/big"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
)

func run5Registry() *generator.Registry {
	reg := generator.NewRegistry()
	Register(reg)
	return reg
}

func run5Samples(t *testing.T, reg *generator.Registry, id string, n int) []generator.Problem {
	t.Helper()
	var out []generator.Problem
	for i := 0; i < n; i++ {
		p, err := reg.GenerateContext(id, generator.GeneratorContext{Difficulty: []float64{0.3, 0.6, 0.9}[i%3], Seed: int64(5000 + i)})
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		out = append(out, p)
	}
	return out
}

// The table item "global maximum of x^2 - 4x on [0,3]" said the maximum 0 was
// at x = 3, where f(3) = -3 (fails on the old text).
func TestRun5GlobalExtremaTableNamesTheMaximiser(t *testing.T) {
	seen := false
	for _, p := range run5Samples(t, run5Registry(), "calc.deriv.global_extrema", 2000) {
		if !strings.Contains(p.Question, "maximum VALUE of $f(x)=x^{2}-4x$ on $[0,3]$") {
			continue
		}
		seen = true
		if !strings.Contains(p.Explanation, "Global max value is 0 at $x=0$") {
			t.Fatalf("explanation names the wrong maximiser: %q", p.Explanation)
		}
	}
	if !seen {
		t.Skip("table item not drawn")
	}
}

// A tie (the cubic takes its maximum at x = -c and x = 2c) is reported with
// every point where the value is attained.
func TestRun5AttainedAtListsTies(t *testing.T) {
	if got := attainedAt([]int{-1, 2, -1, 1}, []int{2, 2, 2, -2}, 2, "x"); got != "\\(x=-1\\) and \\(x=2\\)" {
		t.Fatalf("attainedAt = %q", got)
	}
}

var run5DotRe = regexp.MustCompile(`\\nabla f\\cdot u = \((.*)\)/(\d+) = `)

// The directional-derivative explanation printed "(3\cdot 5\cdot 12)/13" for
// gradient (3, 0) and v = (5, 12): the zero term was dropped by the tidy pass
// and the rest read as a product (fails on the old text). The printed sum must
// evaluate to the numerator of the key.
func TestRun5DirectionalExplanationSumIsTrue(t *testing.T) {
	n := 0
	for _, p := range run5Samples(t, run5Registry(), "calc.deriv.directional", 3000) {
		m := run5DotRe.FindStringSubmatch(p.Explanation)
		if m == nil {
			continue // a table item
		}
		n++
		den, _ := strconv.Atoi(m[2])
		sum := 0
		for _, term := range strings.Split(m[1], " + ") {
			fs := strings.Split(term, "\\cdot ")
			if term == "0" {
				continue
			}
			if len(fs) != 2 {
				t.Fatalf("term %q in %q is not a product of two numbers", term, p.Explanation)
			}
			a, err1 := strconv.Atoi(strings.Trim(fs[0], "() "))
			b, err2 := strconv.Atoi(strings.Trim(fs[1], "() "))
			if err1 != nil || err2 != nil {
				t.Fatalf("cannot read %q in %q", term, p.Explanation)
			}
			sum += a * b
		}
		if got := big.NewRat(int64(sum), int64(den)).RatString(); got != p.Answer {
			t.Fatalf("explanation sum %d/%d = %s but the key is %s: %q", sum, den, got, p.Answer, p.Explanation)
		}
	}
	if n < 1000 {
		t.Fatalf("only %d computed items checked", n)
	}
}

// Q35 unit coefficients written by the generators (fails on the old text).
func TestRun5NoUnitArtefacts(t *testing.T) {
	reg := run5Registry()
	checks := []struct {
		id  string
		bad *regexp.Regexp
	}{
		{"calc.deriv.chain_rule", regexp.MustCompile(`\)\^1\b|\^\{1\}|·1 `)},
		{"calc.integral.ftc", regexp.MustCompile(`[tx]\^\{1\}|[tux]\^1\b|\)\^\{1\}`)},
		{"calc.multi.tangent_plane", regexp.MustCompile(`\(1\)\^2`)},
		{"calc.deriv.difference_quotient", regexp.MustCompile(`2·1·h`)},
	}
	for _, c := range checks {
		for _, p := range run5Samples(t, reg, c.id, 600) {
			for _, s := range []string{p.Question, p.Answer, p.Explanation} {
				if c.bad.MatchString(s) {
					t.Fatalf("%s: unit artefact in %q", c.id, s)
				}
			}
		}
	}
}

// "(enter -1/6)" printed the key in the question (fails on the old text).
func TestRun5TaylorDoesNotLeakTheKey(t *testing.T) {
	for _, p := range run5Samples(t, run5Registry(), "calc.series.taylor", 2000) {
		if strings.Contains(p.Question, "(enter "+p.Answer+")") {
			t.Fatalf("question gives the answer: %q", p.Question)
		}
	}
}

// "(left limit $=-1$, right limit $=1$." had an unclosed parenthesis.
func TestRun5ContinuityExplanationParensBalance(t *testing.T) {
	for _, p := range run5Samples(t, run5Registry(), "calc.limit.continuity", 600) {
		if strings.Count(p.Explanation, "(") != strings.Count(p.Explanation, ")") {
			t.Fatalf("unbalanced parentheses: %q", p.Explanation)
		}
	}
}

// The open-ended "Give a function continuous but not differentiable at 0"
// accepted only |x|; the item now names its candidates.
func TestRun5DiffContinuityItemIsClosed(t *testing.T) {
	for _, p := range run5Samples(t, run5Registry(), "calc.deriv.diff_continuity", 1000) {
		if strings.HasPrefix(p.Question, "Give a function") {
			t.Fatalf("open-ended item still served: %q", p.Question)
		}
	}
}

// Q47: every pool the review listed serves at least 12 distinct questions.
func TestRun5PoolsHaveVariety(t *testing.T) {
	reg := run5Registry()
	for _, id := range []string{
		"calc.multi.iterated_limits", "calc.deriv.optimization", "calc.series.direct_compare",
		"calc.series.gamma", "calc.series.cauchy_product", "calc.limit.supremum", "calc.limit.limsup",
		"calc.limit.epsilon_delta", "calc.seq.bounded", "calc.seq.cauchy", "calc.seq.convergence",
		"calc.seq.monotone", "calc.seq.function_sequences", "calc.series.function_series",
		"calc.series.radius", "calc.series.absolute", "calc.series.alternating", "calc.series.fourier",
		"calc.limit.uniform_continuity", "calc.deriv.concept",
	} {
		gen, err := reg.Get(id)
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		if n := distinctQuestions(gen); n < 12 {
			t.Errorf("%s: only %d distinct questions, want >= 12", id, n)
		}
	}
}

// calc.deriv.concept was one template ("slope of ax^n at x=k").
func TestRun5DerivConceptHasSeveralTemplates(t *testing.T) {
	kinds := map[string]bool{}
	for _, p := range run5Samples(t, run5Registry(), "calc.deriv.concept", 600) {
		switch {
		case strings.HasPrefix(p.Question, "What is the slope of $f(x)="):
			kinds["power"] = true
		case strings.Contains(p.Question, "\\lim_{h \\to 0}"):
			kinds["limit"] = true
		case strings.Contains(p.Question, "/x$"):
			kinds["reciprocal"] = true
		case strings.Contains(p.Question, "\\sqrt{x}"):
			kinds["root"] = true
		case strings.Contains(p.Question, "velocity"):
			kinds["velocity"] = true
		case strings.Contains(p.Question, "y-axis"):
			kinds["tangent"] = true
		}
	}
	if len(kinds) < 5 {
		t.Fatalf("only templates %v", kinds)
	}
}

var run5Ints = regexp.MustCompile(`-?\d+`)

func run5Nums(s string) []int {
	var out []int
	for _, m := range run5Ints.FindAllString(s, -1) {
		v, _ := strconv.Atoi(m)
		out = append(out, v)
	}
	return out
}

// Keys of the new computed families, recomputed from the question text.
func TestRun5VariantKeysRecomputed(t *testing.T) {
	check := func(name string, gen func() generator.Problem, want func(q string) (string, bool)) {
		matched := 0
		defer func() {
			if matched < 20 {
				t.Errorf("%s: only %d items recomputed", name, matched)
			}
		}()
		for i := 0; i < 1500; i++ {
			rand.Seed(int64(77 + i))
			p := gen()
			w, ok := want(p.Question)
			if !ok {
				continue
			}
			matched++
			if w != p.Answer {
				t.Fatalf("%s: %q keyed %q, recomputed %q", name, p.Question, p.Answer, w)
			}
		}
	}
	check("optimization", optimizationVariant, func(q string) (string, bool) {
		n := run5Nums(q)
		switch {
		case strings.Contains(q, "river") && strings.Contains(q, "perpendicular"):
			return fmt.Sprint(n[0] / 4), true
		case strings.Contains(q, "river"):
			return fmt.Sprint(n[0] * n[0] / 8), true
		case strings.Contains(q, "open box") && strings.Contains(q, "Which"):
			return fmt.Sprint(n[0] / 6), true
		case strings.Contains(q, "open box"):
			s := n[0]
			best := 0
			for x := 1; 2*x < s; x++ {
				if v := x * (s - 2*x) * (s - 2*x); v > best {
					best = v
				}
			}
			return fmt.Sprint(best), true
		case strings.Contains(q, "product"):
			for m := 1; m*m <= n[0]; m++ {
				if m*m == n[0] {
					return fmt.Sprint(2 * m), true
				}
			}
		}
		return "", false
	})
	check("cauchy product", cauchyProductVariant, func(q string) (string, bool) {
		if !strings.Contains(q, "(x)^{n}") && !strings.Contains(q, "x)^{n}$ and") {
			return "", false
		}
		re := regexp.MustCompile(`\((-?\d*)x\)\^\{n\}`)
		ms := re.FindAllStringSubmatch(q, -1)
		if len(ms) != 2 {
			return "", false
		}
		coef := func(s string) int {
			switch s {
			case "":
				return 1
			case "-":
				return -1
			}
			v, _ := strconv.Atoi(s)
			return v
		}
		a, b := coef(ms[0][1]), coef(ms[1][1])
		n := run5Nums(q[strings.LastIndex(q, "x^{"):])[0]
		c := 0
		for k := 0; k <= n; k++ {
			c += ipow(a, k) * ipow(b, n-k)
		}
		return fmt.Sprint(c), true
	})
	check("gamma ratio", gammaVariant, func(q string) (string, bool) {
		re := regexp.MustCompile(`\\Gamma\((\d+)\)/\\Gamma\((\d+)\)`)
		m := re.FindStringSubmatch(q)
		if m == nil {
			return "", false
		}
		hi, _ := strconv.Atoi(m[1])
		lo, _ := strconv.Atoi(m[2])
		return fmt.Sprint(calcFact(hi-1) / calcFact(lo-1)), true
	})
	check("sequence limit", seqConvergenceVariant, func(q string) (string, bool) {
		re := regexp.MustCompile(`\\frac\{(\d*)n \+ (\d+)\}\{(\d*)n \+ (\d+)\}`)
		m := re.FindStringSubmatch(q)
		if m == nil {
			return "", false
		}
		num := func(s string) int {
			if s == "" {
				return 1
			}
			v, _ := strconv.Atoi(s)
			return v
		}
		return "yes, " + big.NewRat(int64(num(m[1])), int64(num(m[3]))).RatString(), true
	})
	check("alternating error bound", alternatingVariant, func(q string) (string, bool) {
		if !strings.Contains(q, "first") {
			return "", false
		}
		p := 1
		if m := regexp.MustCompile(`/n\^\{(\d)\}`).FindStringSubmatch(q); m != nil {
			p, _ = strconv.Atoi(m[1])
		}
		N := run5Nums(q[strings.Index(q, "first"):])[0]
		return big.NewRat(1, int64(ipow(N+1, p))).RatString(), true
	})
}

// Every new multiple-choice family offers four options with the key exactly once.
func TestRun5MCVariantsHaveFourOptions(t *testing.T) {
	opt := regexp.MustCompile(`\n\n([A-E])\) ([^\n]*)`)
	for name, gen := range map[string]func() generator.Problem{
		"iterated": iteratedLimitsVariant, "limsup": limsupVariant, "direct compare": directCompareVariant,
	} {
		for i := 0; i < 3000; i++ {
			rand.Seed(int64(i))
			p := gen()
			ms := opt.FindAllStringSubmatch(p.Question, -1)
			if len(ms) != 4 {
				t.Fatalf("%s: %d options in %q", name, len(ms), p.Question)
			}
			hits := 0
			for _, m := range ms {
				if m[2] == p.Answer {
					hits++
				}
			}
			if hits != 1 {
				t.Fatalf("%s: key %q appears %d times in %q", name, p.Answer, hits, p.Question)
			}
		}
	}
}
