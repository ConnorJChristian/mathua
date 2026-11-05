package generator_test

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/solutions"
	"github.com/chuma-beep/mathua/internal/walkthrough"
)

// The step templates added for task 0c (fork change) must say what the
// instance computes: every number a step quotes is a fact the generator
// published, and here each fact is recomputed from the instance's inputs
// with this test's own arithmetic, then looked for in the rendered steps.
// A template that quoted a stale or mislabelled fact would fail here, not in
// front of the learner.

type stepCheck struct {
	concept string
	// recompute reads the inputs off the facts, recomputes the derived
	// values independently, and returns the numbers (as the facts write them)
	// that the steps must quote, with the answer.
	recompute func(t *testing.T, facts map[string]string, answer string) []string
}

func num(t *testing.T, facts map[string]string, k string) float64 {
	t.Helper()
	v, err := strconv.ParseFloat(strings.Trim(facts[k], "()"), 64)
	if err != nil {
		t.Fatalf("fact %q = %q is not a number", k, facts[k])
	}
	return v
}

func fmtNum(v float64) string {
	if v == math.Trunc(v) {
		return strconv.Itoa(int(v))
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

var stepChecks = []stepCheck{
	{"calc.deriv.marginal_analysis", func(t *testing.T, f map[string]string, answer string) []string {
		a, b, x0 := num(t, f, "a"), num(t, f, "b"), num(t, f, "x0")
		m := 2*a*x0 + b
		if fmtNum(m) != answer {
			t.Errorf("marginal cost: 2·%v·%v + %v = %v, answer %s", a, x0, b, m, answer)
		}
		return []string{fmtNum(2 * a), fmtNum(m)}
	}},
	{"calc.deriv.partial", func(t *testing.T, f map[string]string, answer string) []string {
		a, b, x0, y0 := num(t, f, "a"), num(t, f, "b"), num(t, f, "x0"), num(t, f, "y0")
		var v1, v2 float64
		if f["dir"] == "x" {
			v1, v2 = 2*a*x0*y0, b*y0*y0
		} else {
			v1, v2 = a*x0*x0, 2*b*x0*y0
		}
		if fmtNum(v1+v2) != answer {
			t.Errorf("partial: %v + %v = %v, answer %s", v1, v2, v1+v2, answer)
		}
		return []string{fmtNum(v1), fmtNum(v2), fmtNum(v1 + v2)}
	}},
	{"calc.deriv.difference_quotient", func(t *testing.T, f map[string]string, answer string) []string {
		x0, h := num(t, f, "x0"), num(t, f, "h")
		fx0h, fx0 := (x0+h)*(x0+h), x0*x0
		q := (fx0h - fx0) / h
		if fmtNum(q) != answer {
			t.Errorf("difference quotient: (%v - %v)/%v = %v, answer %s", fx0h, fx0, h, q, answer)
		}
		return []string{fmtNum(fx0h), fmtNum(fx0), fmtNum(fx0h - fx0), fmtNum(2 * x0)}
	}},
	{"calc.deriv.differential", func(t *testing.T, f map[string]string, answer string) []string {
		x0, dx := num(t, f, "x0"), num(t, f, "dx")
		dy := 2 * x0 * dx
		if fmt.Sprintf("%.2f", dy) != answer {
			t.Errorf("differential: 2·%v·%v = %.2f, answer %s", x0, dx, dy, answer)
		}
		x1 := x0 + dx
		return []string{fmtNum(2 * x0), fmt.Sprintf("%.2f", dy), fmt.Sprintf("%.2f", x1*x1-x0*x0)}
	}},
	{"calc.deriv.sum_rule", func(t *testing.T, f map[string]string, answer string) []string {
		a, n, b, m := num(t, f, "a"), num(t, f, "n"), num(t, f, "b"), num(t, f, "m")
		return []string{fmtNum(a * n), fmtNum(b * m), fmtNum(n - 1), fmtNum(m - 1)}
	}},
	{"calc.deriv.implicit", func(t *testing.T, f map[string]string, answer string) []string {
		x0, y0, rr := num(t, f, "x0"), num(t, f, "y0"), num(t, f, "rr")
		if x0*x0+y0*y0 != rr {
			t.Errorf("implicit: the point (%v,%v) is not on x²+y²=%v", x0, y0, rr)
		}
		return []string{fmtNum(x0), fmtNum(y0)}
	}},
	{"calc.multi.implicit_function", func(t *testing.T, f map[string]string, answer string) []string {
		x, y := num(t, f, "x"), num(t, f, "y")
		slope := -x / y
		if got, _ := strconv.ParseFloat(answer, 64); math.Abs(got-slope) > 1e-5 {
			t.Errorf("implicit function: -%v/%v = %v, answer %s", x, y, slope, answer)
		}
		return []string{fmtNum(2 * x), fmtNum(2 * y)}
	}},
	{"calc.seq.recursive_value", func(t *testing.T, f map[string]string, answer string) []string {
		v, r, c, steps := num(t, f, "start"), num(t, f, "r"), num(t, f, "c"), num(t, f, "steps")
		var quote []string
		for i := 0; i < int(steps); i++ {
			v = r*v + c
			quote = append(quote, fmtNum(v))
		}
		if fmtNum(v) != answer {
			t.Errorf("recurrence: %v after %v steps, answer %s", v, steps, answer)
		}
		return quote
	}},
	{"linalg.det.2x2", func(t *testing.T, f map[string]string, answer string) []string {
		a, b, c, d := num(t, f, "a"), num(t, f, "b"), num(t, f, "c"), num(t, f, "d")
		if fmtNum(a*d-b*c) != answer {
			t.Errorf("det: %v·%v - %v·%v = %v, answer %s", a, d, b, c, a*d-b*c, answer)
		}
		return []string{fmtNum(a * d), fmtNum(b * c)}
	}},
	{"linalg.systems.matrix", func(t *testing.T, f map[string]string, answer string) []string {
		a, b, c, d := num(t, f, "a"), num(t, f, "b"), num(t, f, "c"), num(t, f, "d")
		e1, e2 := num(t, f, "e1"), num(t, f, "e2")
		det := a*d - b*c
		x, y := (e1*d-b*e2)/det, (a*e2-e1*c)/det
		if fmt.Sprintf("(%s,%s)", fmtNum(x), fmtNum(y)) != answer {
			t.Errorf("system: Cramer gives (%v,%v), answer %s", x, y, answer)
		}
		return []string{fmtNum(det), fmtNum(e1*d - b*e2), fmtNum(a*e2 - e1*c), fmtNum(x), fmtNum(y)}
	}},
	{"linalg.sys.cramer", func(t *testing.T, f map[string]string, answer string) []string {
		a, b, c, d := num(t, f, "a"), num(t, f, "b"), num(t, f, "c"), num(t, f, "d")
		det, detv := a*d-b*c, num(t, f, "detv")
		if fmtNum(detv/det) != answer {
			t.Errorf("cramer: %v/%v, answer %s", detv, det, answer)
		}
		return []string{fmtNum(det), fmtNum(detv)}
	}},
	{"prob.rv.variance", func(t *testing.T, f map[string]string, answer string) []string {
		vals, wts, den := strings.Split(f["values"], ","), strings.Split(f["weights"], ","), num(t, f, "den")
		var mean, second float64
		for i := range vals {
			v, _ := strconv.ParseFloat(vals[i], 64)
			w, _ := strconv.ParseFloat(wts[i], 64)
			mean += v * w / den
			second += v * v * w / den
		}
		variance := second - mean*mean
		if got, _ := strconv.ParseFloat(answer, 64); math.Abs(got-variance) > 1e-6 {
			t.Errorf("variance: %v - %v² = %v, answer %s", second, mean, variance, answer)
		}
		return []string{fmtNum(mean), fmtNum(second), fmtNum(math.Round(mean*mean*1e6) / 1e6)}
	}},
	{"prob.cov.covariance", func(t *testing.T, f map[string]string, answer string) []string {
		w := strings.Split(f["weights"], ",")
		var ws [4]float64
		for i := range w {
			ws[i], _ = strconv.ParseFloat(w[i], 64)
		}
		total := ws[0] + ws[1] + ws[2] + ws[3]
		exy, ex, ey := ws[3]/total, (ws[2]+ws[3])/total, (ws[1]+ws[3])/total
		if got, _ := strconv.ParseFloat(answer, 64); math.Abs(got-(exy-ex*ey)) > 1e-6 {
			t.Errorf("covariance: %v - %v·%v, answer %s", exy, ex, ey, answer)
		}
		return []string{fmtNum(exy), fmtNum(ex), fmtNum(ey)}
	}},
	{"stat.zscore.value", func(t *testing.T, f map[string]string, answer string) []string {
		x, mean, sd := num(t, f, "x"), num(t, f, "mean"), num(t, f, "sd")
		if fmtNum((x-mean)/sd) != answer {
			t.Errorf("z: (%v - %v)/%v, answer %s", x, mean, sd, answer)
		}
		return []string{fmtNum(x - mean)}
	}},
	{"stat.infer.ttest", func(t *testing.T, f map[string]string, answer string) []string {
		xbar, mu, s, n := num(t, f, "xbar"), num(t, f, "mu"), num(t, f, "s"), num(t, f, "n")
		se := s / math.Sqrt(n)
		if got, _ := strconv.ParseFloat(answer, 64); math.Abs(got-(xbar-mu)/se) > 1e-6 {
			t.Errorf("t: (%v - %v)/%v, answer %s", xbar, mu, se, answer)
		}
		return []string{fmtNum(xbar - mu), fmtNum(se)}
	}},
	{"prob.limit.clt", func(t *testing.T, f map[string]string, answer string) []string {
		n, mu, sigma, x := num(t, f, "n"), num(t, f, "mu"), num(t, f, "sigma"), num(t, f, "x")
		se := sigma * math.Sqrt(n)
		if got, _ := strconv.ParseFloat(answer, 64); math.Abs(got-(x-n*mu)/se) > 1e-6 {
			t.Errorf("clt: (%v - %v)/%v, answer %s", x, n*mu, se, answer)
		}
		return []string{fmtNum(n * mu), fmtNum(x - n*mu), fmtNum(se)}
	}},
}

func TestWalkthroughStepsRecompute(t *testing.T) {
	l, err := solutions.Load(solutionsDir)
	if err != nil {
		t.Fatal(err)
	}
	reg := generator.NewRegistry()
	reg.SetSolutions(l)
	registerAllDomains(reg)
	for _, sc := range stepChecks {
		t.Run(sc.concept, func(t *testing.T) {
			if _, ok := l.Schema(sc.concept); !ok {
				t.Fatalf("no schema for %s", sc.concept)
			}
			covered := 0
			for i := 0; i < 20; i++ {
				p, err := reg.GenerateContext(sc.concept, generator.GeneratorContext{Difficulty: 0.3 + 0.035*float64(i), Seed: int64(500 + i)})
				if err != nil {
					t.Fatal(err)
				}
				explain, steps := walkthrough.Split(p.Explanation)
				if len(steps) == 0 {
					continue // a fixed-table draw the schema does not cover
				}
				covered++
				// The steps and the closing line are the walkthrough.
				joined := strings.Join(append(steps, explain), "\n")
				for _, want := range sc.recompute(t, p.Facts, p.Answer) {
					if !strings.Contains(joined, want) {
						t.Errorf("%s: steps never quote the recomputed value %q:\n%s", p.Question, want, joined)
					}
				}
			}
			if covered == 0 {
				t.Errorf("no instance of %s rendered steps", sc.concept)
			}
		})
	}
}
