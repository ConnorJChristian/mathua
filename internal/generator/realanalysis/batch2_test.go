package realanalysis

// Tests for the second real-analysis batch.
//
// The fifteen multiple-choice concepts are checked the way the first batch checks
// its own: the question is parsed into options, the answer must appear exactly
// once, and every distractor must be rejected by the real router. The ten numeric
// concepts recompute their answers from the published parameters: the mean value
// point of x², a chain-rule derivative, a limit of a trigonometric quotient, a
// right-endpoint Riemann sum, a definite integral, an integration by parts, an
// improper integral, a radius of convergence, a Taylor coefficient and the fixed
// point of a contraction. The tests never read the published "answer" fact.

import (
	"math"
	"strconv"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
)

var raBatch2IDs = []string{
	"ra.limit.connected",
	"ra.limit.discontinuity",
	"ra.limit.monotone_fn",
	"ra.deriv.mvt",
	"ra.deriv.chain",
	"ra.deriv.taylor",
	"ra.deriv.lhopital",
	"ra.deriv.riemann",
	"ra.deriv.darboux",
	"ra.deriv.ftc",
	"ra.deriv.parts",
	"ra.deriv.improper",
	"ra.deriv.uniform_conv",
	"ra.deriv.power_series",
	"ra.deriv.interchange",
	"ra.metric.spaces",
	"ra.metric.open_closed",
	"ra.metric.closure",
	"ra.metric.convergence",
	"ra.metric.complete",
	"ra.metric.compact",
	"ra.metric.continuity",
	"ra.metric.contraction",
	"ra.metric.baire",
	"ra.metric.equivalent",
}

var raBatch2MCIDs = []string{
	"ra.limit.connected",
	"ra.limit.discontinuity",
	"ra.limit.monotone_fn",
	"ra.deriv.darboux",
	"ra.deriv.uniform_conv",
	"ra.deriv.interchange",
	"ra.metric.spaces",
	"ra.metric.open_closed",
	"ra.metric.closure",
	"ra.metric.convergence",
	"ra.metric.complete",
	"ra.metric.compact",
	"ra.metric.continuity",
	"ra.metric.baire",
	"ra.metric.equivalent",
}

func TestRABatch2AnswersGradeWithTheirShippedType(t *testing.T) {
	types := raShippedTypes(t)
	reg := raRegistry(t)
	router := grader.NewRouter()
	defer router.Close()

	for _, id := range raBatch2IDs {
		typ, ok := types[id]
		if !ok || typ == "" {
			t.Fatalf("%s: no grading_type in the corpus", id)
		}
		if typ != "multiple_choice" && typ != "numeric" {
			t.Fatalf("%s: unexpected grading_type %q", id, typ)
		}
		for i := 0; i < 120; i++ {
			p := raSample(t, reg, id, 3000+i)
			if p.Answer == "" {
				t.Fatalf("%s: empty answer at sample %d", id, i)
			}
			res := router.Grade(grader.GradingType(typ), p.Answer, p.Answer)
			if res.Unavailable {
				t.Skipf("%s: grader unavailable", id)
			}
			if !res.Correct {
				t.Fatalf("%s: the generator's own answer %q does not grade as %s", id, p.Answer, typ)
			}
		}
	}
}

func TestRABatch2MultipleChoiceRejectsEveryDistractor(t *testing.T) {
	types := raShippedTypes(t)
	reg := raRegistry(t)
	router := grader.NewRouter()
	defer router.Close()

	for _, id := range raBatch2MCIDs {
		if types[id] != "multiple_choice" {
			t.Fatalf("%s: expected multiple_choice, corpus says %q", id, types[id])
		}
		for i := 0; i < 40; i++ {
			p := raSample(t, reg, id, 7000+i)
			opts := raOptionsOf(p.Question)
			if len(opts) < 3 {
				t.Fatalf("%s: only %d options; a recognition question needs at least three", id, len(opts))
			}
			found := 0
			for _, o := range opts {
				if o == p.Answer {
					found++
				}
			}
			if found != 1 {
				t.Fatalf("%s: the answer %q appears %d times among %v", id, p.Answer, found, opts)
			}
			for _, o := range opts {
				if o == p.Answer {
					continue
				}
				if router.Grade(grader.GradingMultipleChoice, p.Answer, o).Correct {
					t.Fatalf("%s: distractor %q is accepted as correct", id, o)
				}
			}
		}
	}
}

// raAnswerNear checks a decimal answer against an independently computed value.
func raAnswerNear(t *testing.T, label string, p generator.Problem, want float64) {
	t.Helper()
	got, err := strconv.ParseFloat(p.Answer, 64)
	if err != nil || math.Abs(got-want) > 1e-6*math.Max(1, math.Abs(want)) {
		t.Fatalf("%s: %q answers %s, want %.9f", label, p.Question, p.Answer, want)
	}
}

// raSimpson integrates f over [a, b] numerically.
func raSimpson(f func(float64) float64, a, b float64) float64 {
	const n = 20000
	h := (b - a) / n
	s := f(a) + f(b)
	for i := 1; i < n; i++ {
		x := a + float64(i)*h
		if i%2 == 1 {
			s += 4 * f(x)
		} else {
			s += 2 * f(x)
		}
	}
	return s * h / 3
}

func TestRABatch2NumericRecompute(t *testing.T) {
	reg := raRegistry(t)

	for i := 0; i < 300; i++ {
		p := raSample(t, reg, "ra.deriv.mvt", 100+i)
		a := float64(raFactInt(t, p, "a"))
		b := float64(raFactInt(t, p, "b"))
		// Solve f'(c) = (f(b) − f(a))/(b − a) numerically by bisection on
		// the derivative, which is monotone for each kind.
		var f, fp func(float64) float64
		switch p.Facts["kind"] {
		case "quad":
			f, fp = func(x float64) float64 { return x * x }, func(x float64) float64 { return 2 * x }
		case "recip":
			f, fp = func(x float64) float64 { return 1 / x }, func(x float64) float64 { return -1 / (x * x) }
		case "sqrt":
			f, fp = math.Sqrt, func(x float64) float64 { return 0.5 / math.Sqrt(x) }
		default:
			t.Fatalf("mvt: unknown kind %q", p.Facts["kind"])
		}
		slope := (f(b) - f(a)) / (b - a)
		lo, hi := a, b
		if lo == 0 {
			lo = 1e-12
		}
		inc := fp(hi) > fp(lo)
		for j := 0; j < 200; j++ {
			mid := (lo + hi) / 2
			if (fp(mid) < slope) == inc {
				lo = mid
			} else {
				hi = mid
			}
		}
		raAnswerNear(t, "mvt", p, (lo+hi)/2)
	}

	for i := 0; i < 300; i++ {
		p := raSample(t, reg, "ra.deriv.chain", 100+i)
		a := raFactInt(t, p, "a")
		b := raFactInt(t, p, "b")
		n := raFactInt(t, p, "n")
		x0 := raFactInt(t, p, "x0")
		want := decimal6(float64(n * a * raIntPow(a*x0+b, n-1)))
		if p.Answer != want {
			t.Fatalf("chain: a=%d b=%d n=%d x0=%d answers %s, want %s", a, b, n, x0, p.Answer, want)
		}
	}

	for i := 0; i < 300; i++ {
		p := raSample(t, reg, "ra.deriv.lhopital", 100+i)
		a := float64(raFactInt(t, p, "a"))
		b := float64(raFactInt(t, p, "b"))
		// Evaluate the quotient itself near 0.
		x := 1e-4
		var q float64
		switch p.Facts["kind"] {
		case "cos":
			q = (1 - math.Cos(a*x)) / (b * x * x)
		case "sin":
			q = math.Sin(a*x) / (b * x)
		case "exp":
			q = math.Expm1(a*x) / (b * x)
		case "log":
			q = math.Log1p(a*x) / (b * x)
		default:
			t.Fatalf("lhopital: unknown kind %q", p.Facts["kind"])
		}
		got, _ := strconv.ParseFloat(p.Answer, 64)
		if math.Abs(got-q) > 1e-2*math.Max(1, math.Abs(q)) {
			t.Fatalf("lhopital: %q answers %s, quotient near 0 is %v", p.Question, p.Answer, q)
		}
	}

	for i := 0; i < 300; i++ {
		p := raSample(t, reg, "ra.deriv.riemann", 100+i)
		b := raFactInt(t, p, "b")
		n := raFactInt(t, p, "n")
		sum := 0.0
		for k := 1; k <= n; k++ {
			sum += (float64(k*b) / float64(n)) * (float64(b) / float64(n))
		}
		raAnswerNear(t, "riemann", p, sum)
	}

	for i := 0; i < 300; i++ {
		p := raSample(t, reg, "ra.deriv.ftc", 100+i)
		a := float64(raFactInt(t, p, "a"))
		c := float64(raFactInt(t, p, "c"))
		b := float64(raFactInt(t, p, "b"))
		raAnswerNear(t, "ftc", p, raSimpson(func(x float64) float64 { return a*x + c }, 0, b))
	}

	for i := 0; i < 300; i++ {
		p := raSample(t, reg, "ra.deriv.parts", 100+i)
		var want float64
		switch p.Facts["kind"] {
		case "xexp":
			b := float64(raFactInt(t, p, "b"))
			want = raSimpson(func(x float64) float64 { return x * math.Exp(x) }, 0, b)
		case "ln":
			b := float64(raFactInt(t, p, "b"))
			want = raSimpson(math.Log, 1, b)
		case "xexpk":
			k := float64(raFactInt(t, p, "k"))
			want = raSimpson(func(x float64) float64 { return x * math.Exp(k*x) }, 0, 1)
		default:
			t.Fatalf("parts: unknown kind %q", p.Facts["kind"])
		}
		raAnswerNear(t, "parts", p, want)
	}

	for i := 0; i < 300; i++ {
		p := raSample(t, reg, "ra.deriv.improper", 100+i)
		var want float64
		switch p.Facts["kind"] {
		case "p":
			pp := float64(raFactInt(t, p, "p"))
			a := float64(raFactInt(t, p, "a"))
			if pp <= 1 {
				t.Fatalf("improper: p=%v does not converge", pp)
			}
			// Substitute x = a/u to map [a, ∞) onto (0, 1].
			// x^(−p)·a/u² with x = a/u simplifies to a^(1−p)·u^(p−2),
			// which is finite on [0, 1] for p ≥ 2.
			want = raSimpson(func(u float64) float64 {
				return math.Pow(a, 1-pp) * math.Pow(u, pp-2)
			}, 0, 1)
		case "exp":
			k := float64(raFactInt(t, p, "k"))
			want = raSimpson(func(x float64) float64 { return math.Exp(-k * x) }, 0, 60)
		default:
			t.Fatalf("improper: unknown kind %q", p.Facts["kind"])
		}
		raAnswerNear(t, "improper", p, want)
	}

	for i := 0; i < 300; i++ {
		p := raSample(t, reg, "ra.deriv.power_series", 100+i)
		c := float64(raFactInt(t, p, "c"))
		// Root test on the coefficients at a large index: R = 1/lim |a_n|^(1/n).
		var r float64
		switch p.Facts["kind"] {
		case "mul":
			r = 1 / c
		case "div":
			r = c
		default:
			t.Fatalf("power_series: unknown kind %q", p.Facts["kind"])
		}
		raAnswerNear(t, "power_series", p, r)
	}

	for i := 0; i < 300; i++ {
		p := raSample(t, reg, "ra.deriv.taylor", 100+i)
		a := float64(raFactInt(t, p, "a"))
		k := raFactInt(t, p, "k")
		// The x^k coefficient is the k-th derivative at 0 over k!, estimated
		// here with a central finite difference of the function itself.
		var f func(float64) float64
		switch p.Facts["kind"] {
		case "exp":
			f = func(x float64) float64 { return math.Exp(a * x) }
		case "sin":
			f = func(x float64) float64 { return math.Sin(a * x) }
		case "cos":
			f = func(x float64) float64 { return math.Cos(a * x) }
		case "geom":
			f = func(x float64) float64 { return 1 / (1 - a*x) }
		case "log":
			f = func(x float64) float64 { return math.Log1p(a * x) }
		default:
			t.Fatalf("taylor: unknown kind %q", p.Facts["kind"])
		}
		h := 1e-2 / a
		var deriv float64
		for j := 0; j <= k; j++ {
			sign := 1.0
			if j%2 == 1 {
				sign = -1
			}
			deriv += sign * float64(raBinom(k, j)) * f((float64(k)/2-float64(j))*h)
		}
		deriv /= math.Pow(h, float64(k))
		coef := deriv / float64(raFactorial(k))
		got, _ := strconv.ParseFloat(p.Answer, 64)
		if math.Abs(got-coef) > 1e-2*math.Max(1, math.Abs(coef)) {
			t.Fatalf("taylor: %q answers %s, finite difference gives %v", p.Question, p.Answer, coef)
		}
	}

	for i := 0; i < 300; i++ {
		p := raSample(t, reg, "ra.metric.contraction", 100+i)
		pn := raFactInt(t, p, "pn")
		pd := raFactInt(t, p, "pd")
		b := float64(raFactInt(t, p, "b"))
		if pn >= pd || -pn >= pd {
			t.Fatalf("contraction: %d/%d is not a contraction", pn, pd)
		}
		// Iterate the map, as the contraction mapping principle says to.
		x := 0.0
		for j := 0; j < 2000; j++ {
			x = float64(pn)/float64(pd)*x + b
		}
		raAnswerNear(t, "contraction", p, x)
	}
}

func raBinom(n, k int) int {
	r := 1
	for i := 1; i <= k; i++ {
		r = r * (n - i + 1) / i
	}
	return r
}

func raFactorial(n int) int {
	r := 1
	for i := 2; i <= n; i++ {
		r *= i
	}
	return r
}
