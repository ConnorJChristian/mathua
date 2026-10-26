package algebra

import (
	"math"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/Knetic/govaluate"
	"github.com/chuma-beep/mathua/internal/generator"
)

// evalTeX evaluates the small LaTeX these generators print (\sqrt{}, \ln,
// \frac{}{}, (a)(b), 3x, x^2) at x. It is a second reading of the formula,
// independent of the generator's own reasoning about where it is defined:
// sqrt of a negative and ln of a non-positive are NaN, a zero denominator is
// infinite.
func evalTeX(t *testing.T, tex string, x float64) float64 {
	t.Helper()
	s := tex
	for i := 0; i < 4; i++ {
		s = regexp.MustCompile(`\\frac\{([^{}]*)\}\{([^{}]*)\}`).ReplaceAllString(s, "(($1)/($2))")
		s = regexp.MustCompile(`\\sqrt\{([^{}]*)\}`).ReplaceAllString(s, "sqrt($1)")
	}
	s = strings.ReplaceAll(s, "\\ln x", "ln(x)")
	s = strings.ReplaceAll(s, "\\ln", "ln")
	s = strings.ReplaceAll(s, ")(", ")*(")
	s = regexp.MustCompile(`(\d)x`).ReplaceAllString(s, "$1*x")
	s = regexp.MustCompile(`x\^\{?(\d)\}?`).ReplaceAllString(s, "(x**$1)")
	s = regexp.MustCompile(`\)\^\{?(\d)\}?`).ReplaceAllString(s, ")**$1")
	fns := map[string]govaluate.ExpressionFunction{
		"exp": func(a ...interface{}) (interface{}, error) { return math.Exp(a[0].(float64)), nil },
		"sqrt": func(a ...interface{}) (interface{}, error) {
			v := a[0].(float64)
			if v < 0 {
				return math.NaN(), nil
			}
			return math.Sqrt(v), nil
		},
		"ln": func(a ...interface{}) (interface{}, error) {
			v := a[0].(float64)
			if v <= 0 {
				return math.NaN(), nil
			}
			return math.Log(v), nil
		},
	}
	e, err := govaluate.NewEvaluableExpressionWithFunctions(s, fns)
	if err != nil {
		t.Fatalf("cannot read %q (from %q): %v", s, tex, err)
	}
	v, err := e.Evaluate(map[string]interface{}{"x": x})
	if err != nil {
		t.Fatalf("cannot evaluate %q at %v: %v", s, x, err)
	}
	return v.(float64)
}

// inAnswer reports whether x lies in an answer such as "[2, 5) ∪ (6, ∞)".
func inAnswer(t *testing.T, ans string, x float64) bool {
	t.Helper()
	for _, piece := range strings.Split(ans, "∪") {
		p := strings.TrimSpace(piece)
		ends := strings.Split(p[1:len(p)-1], ",")
		lo, hi := num(t, ends[0]), num(t, ends[1])
		okLo := x > lo || (p[0] == '[' && x == lo)
		okHi := x < hi || (p[len(p)-1] == ']' && x == hi)
		if okLo && okHi {
			return true
		}
	}
	return false
}

func num(t *testing.T, s string) float64 {
	t.Helper()
	s = strings.TrimSpace(s)
	switch s {
	case "∞":
		return math.Inf(1)
	case "-∞":
		return math.Inf(-1)
	}
	if i := strings.Index(s, "/"); i > 0 {
		return num(t, s[:i]) / num(t, s[i+1:])
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		t.Fatalf("not a number: %q", s)
	}
	return v
}

// gen is the generator's raw output, before the registry rewrites \( \) to
// dollars, so the tests read exactly what the generator wrote.
func gen(id string, d float64, seed int64) generator.Problem {
	reg := generator.NewRegistry()
	Register(reg)
	g, _ := reg.Get(id)
	rand.Seed(seed)
	return g.Generate(generator.GeneratorContext{Difficulty: d, Seed: seed})
}

// The domain key is checked point by point: on a grid of quarter steps, x is
// in the answer exactly when the formula gives a finite real number there.
func TestDomainKeyMatchesWhereTheFormulaIsDefined(t *testing.T) {
	reg := generator.NewRegistry()
	Register(reg)
	for i := 0; i < 240; i++ {
		d := []float64{0.3, 0.6, 0.9}[i%3]
		p, _ := reg.GenerateContext("alg.func.domain", generator.GeneratorContext{Difficulty: d, Seed: int64(500 + i)})
		fn := p.Facts["fn"]
		for k := -48; k <= 48; k++ {
			x := float64(k) / 4
			v := evalTeX(t, fn, x)
			defined := !math.IsNaN(v) && !math.IsInf(v, 0)
			if defined != inAnswer(t, p.Answer, x) {
				t.Fatalf("f(x) = %s at x = %v: defined=%v, but the key %q says %v", fn, x, defined, p.Answer, !defined)
			}
		}
	}
}

// Difficulty moves the domain question from one condition to combined ones.
func TestDomainDifficultyCombinesConditions(t *testing.T) {
	reg := generator.NewRegistry()
	Register(reg)
	count := func(d float64) (two int) {
		for i := 0; i < 200; i++ {
			p, _ := reg.GenerateContext("alg.func.domain", generator.GeneratorContext{Difficulty: d, Seed: int64(i + 1)})
			if strings.Count(p.Answer, "∪") > 0 && strings.Contains(p.Facts["fn"], "sqrt") ||
				strings.Contains(p.Facts["fn"], "+ \\ln") || strings.Contains(p.Facts["fn"], ")(") {
				two++
			}
		}
		return
	}
	if lo, hi := count(0.3), count(0.95); lo != 0 || hi < 60 {
		t.Errorf("combined-condition domains: %d at 0.3 (want 0), %d of 200 at 0.95 (want 60+)", lo, hi)
	}
}

// The interval stem must open its math: the old generator printed
// "\1 < x < 7)." for want of a "(".
func TestIntervalQuestionsHaveBalancedMath(t *testing.T) {
	for i := 0; i < 300; i++ {
		p := gen("alg.ineq.interval", []float64{0.3, 0.6, 0.9}[i%3], int64(i+1))
		if strings.Count(p.Question, "\\(") != strings.Count(p.Question, "\\)") {
			t.Fatalf("unbalanced math in %q", p.Question)
		}
		if regexp.MustCompile(`\\-?\d`).MatchString(p.Question) {
			t.Fatalf("a backslash before a number in %q", p.Question)
		}
	}
}

// The solved-inequality key is checked on a grid against the inequality.
func TestIntervalSolveKeysMatchTheInequality(t *testing.T) {
	re := regexp.MustCompile(`Solve \\\((.*?) (<|\\le|>|\\ge) (-?\d+)\\\)`)
	checked := 0
	for i := 0; i < 400; i++ {
		p := gen("alg.ineq.interval", 0.6, int64(i+1))
		m := re.FindStringSubmatch(p.Question)
		if m == nil || strings.Count(p.Question, "<")+strings.Count(p.Question, "\\le") > 1 {
			continue
		}
		rhs := num(t, m[3])
		for k := -40; k <= 40; k++ {
			x := float64(k) / 4
			lhs := evalTeX(t, m[1], x)
			var holds bool
			switch m[2] {
			case "<":
				holds = lhs < rhs
			case "\\le":
				holds = lhs <= rhs
			case ">":
				holds = lhs > rhs
			default:
				holds = lhs >= rhs
			}
			if holds != inAnswer(t, p.Answer, x) {
				t.Fatalf("%s at x = %v: inequality %v, key %q disagrees", p.Question, x, holds, p.Answer)
			}
		}
		checked++
	}
	if checked < 50 {
		t.Fatalf("only %d solve questions checked", checked)
	}
}

// |x - c| < r is the open interval of radius r about c, and so on: the key
// is checked on a grid against the absolute-value condition.
func TestAbsoluteValueSetsMatchTheCondition(t *testing.T) {
	re := regexp.MustCompile(`\|(.*?)\| (<|\\le|>) (\d+)\\\}`)
	checked := 0
	for i := 0; i < 400; i++ {
		p := gen("alg.func.absolute_value", []float64{0.6, 0.9}[i%2], int64(i+1))
		m := re.FindStringSubmatch(p.Question)
		if m == nil {
			continue
		}
		r := num(t, m[3])
		for k := -60; k <= 60; k++ {
			x := float64(k) / 4
			v := math.Abs(evalTeX(t, m[1], x))
			holds := map[string]bool{"<": v < r, "\\le": v <= r, ">": v > r}[m[2]]
			if holds != inAnswer(t, p.Answer, x) {
				t.Fatalf("%s at x = %v: condition %v, key %q disagrees", p.Question, x, holds, p.Answer)
			}
		}
		checked++
	}
	if checked < 100 {
		t.Fatalf("only %d absolute-value sets checked", checked)
	}
}

// The inverse undoes f: f(f⁻¹(y)) = y at sample points of the inverse's
// domain. The key is read by the same evaluator, independently of the
// explanation.
func TestInverseKeysUndoTheFunction(t *testing.T) {
	for i := 0; i < 300; i++ {
		p := gen("alg.func.inverse", []float64{0.3, 0.6, 0.9}[i%3], int64(i+1))
		fn := p.Facts["fn"]
		restrict := ""
		if j := strings.Index(fn, ", \\;"); j > 0 {
			fn, restrict = fn[:j], fn[j:]
		}
		inv := strings.NewReplacer("sqrt", "\\sqrt", "exp(x)", "e^x").Replace(p.Answer)
		hits := 0
		for k := -40; k <= 40; k++ {
			y := float64(k)/4 + 0.13
			xv, ok := evalAnswer(p.Answer, y)
			if !ok {
				continue
			}
			if restrict != "" {
				h := num(t, regexp.MustCompile(`-?\d+$`).FindString(restrict))
				if strings.Contains(restrict, "ge") && xv < h-1e-9 || strings.Contains(restrict, "le") && xv > h+1e-9 {
					t.Fatalf("%s: f⁻¹(%v) = %v is outside the restricted domain%s", p.Question, y, xv, restrict)
				}
			}
			back := evalTeX(t, strings.ReplaceAll(fn, "e^{x}", "exp(x)"), xv)
			if math.Abs(back-y) > 1e-6 {
				t.Fatalf("%s: f(f⁻¹(%v)) = %v (key %q, read as %q)", p.Question, y, back, p.Answer, inv)
			}
			hits++
		}
		if hits == 0 {
			t.Fatalf("%s: the key %q is defined nowhere on the grid", p.Question, p.Answer)
		}
	}
}

func evalAnswer(ans string, x float64) (float64, bool) {
	s := strings.ReplaceAll(ans, "e^x", "exp(x)")
	s = strings.ReplaceAll(s, "^", "**")
	fns := map[string]govaluate.ExpressionFunction{
		"sqrt": func(a ...interface{}) (interface{}, error) { return math.Sqrt(a[0].(float64)), nil },
		"ln":   func(a ...interface{}) (interface{}, error) { return math.Log(a[0].(float64)), nil },
		"exp":  func(a ...interface{}) (interface{}, error) { return math.Exp(a[0].(float64)), nil },
	}
	e, err := govaluate.NewEvaluableExpressionWithFunctions(s, fns)
	if err != nil {
		return 0, false
	}
	v, err := e.Evaluate(map[string]interface{}{"x": x})
	if err != nil {
		return 0, false
	}
	f := v.(float64)
	return f, !math.IsNaN(f) && !math.IsInf(f, 0)
}

// Every inverse-domain question has four distinct options.
func TestInverseDomainHasFourOptions(t *testing.T) {
	opt := regexp.MustCompile(`\n\n[A-D]\) `)
	for i := 0; i < 300; i++ {
		p := gen("alg.func.inverse_domain", []float64{0.3, 0.6, 0.9}[i%3], int64(i+1))
		if n := len(opt.FindAllString(p.Question, -1)); n != 4 {
			t.Fatalf("%d options in %q", n, p.Question)
		}
	}
}

