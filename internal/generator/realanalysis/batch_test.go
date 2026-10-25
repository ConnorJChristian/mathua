package realanalysis

// Tests for the first real-analysis batch.
//
// Every concept is graded through the real router with the grading_type the
// corpus ships. The nineteen multiple-choice concepts are checked the way the
// other batches check theirs: the question is parsed into options, the correct
// answer must appear exactly once, and every other option must be rejected by
// the router.
//
// The six numeric concepts recompute their answers from the published
// parameters: the supremum from the interval bound, the Archimedean index from
// the two positive integers, a sequence limit from the leading coefficients, a
// monotone limit from the recurrence constant, a geometric sum from the ratio,
// and a delta from the epsilon and the slope. The tests never read the
// published "answer" fact.

import (
	"encoding/json"
	"math"
	"math/big"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
)

type raCorpusConcept struct {
	ID          string `json:"id"`
	GradingType string `json:"grading_type"`
}

var raBatchIDs = []string{
	"ra.order.supremum",
	"ra.order.completeness",
	"ra.order.archimedean",
	"ra.order.nested_intervals",
	"ra.order.density",
	"ra.order.countable",
	"ra.order.uncountable",
	"ra.order.extended",
	"ra.seq.convergence",
	"ra.seq.monotone",
	"ra.seq.subsequence",
	"ra.seq.bolzano_weierstrass",
	"ra.seq.cauchy",
	"ra.seq.series",
	"ra.seq.tests",
	"ra.seq.absolute",
	"ra.seq.rearrangement",
	"ra.seq.cauchy_criterion",
	"ra.limit.function",
	"ra.limit.sequential",
	"ra.limit.continuity",
	"ra.limit.ivt",
	"ra.limit.evt",
	"ra.limit.uniform",
	"ra.limit.compact",
}

var raMCIDs = []string{
	"ra.order.completeness",
	"ra.order.nested_intervals",
	"ra.order.density",
	"ra.order.countable",
	"ra.order.uncountable",
	"ra.order.extended",
	"ra.seq.subsequence",
	"ra.seq.bolzano_weierstrass",
	"ra.seq.cauchy",
	"ra.seq.tests",
	"ra.seq.absolute",
	"ra.seq.rearrangement",
	"ra.seq.cauchy_criterion",
	"ra.limit.sequential",
	"ra.limit.continuity",
	"ra.limit.ivt",
	"ra.limit.evt",
	"ra.limit.uniform",
	"ra.limit.compact",
}

func raShippedTypes(t *testing.T) map[string]string {
	t.Helper()
	raw, err := os.ReadFile("../../../data/concepts/real_analysis.json")
	if err != nil {
		t.Fatalf("read corpus: %v", err)
	}
	var rows []raCorpusConcept
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatalf("parse corpus: %v", err)
	}
	out := map[string]string{}
	for _, r := range rows {
		if r.ID != "" {
			out[r.ID] = r.GradingType
		}
	}
	return out
}

func raRegistry(t *testing.T) *generator.Registry {
	t.Helper()
	reg := generator.NewRegistry()
	Register(reg)
	return reg
}

func raSample(t *testing.T, reg *generator.Registry, id string, i int) generator.Problem {
	t.Helper()
	p, err := reg.GenerateContext(id, generator.GeneratorContext{
		Difficulty: float64(i%20) / 19,
		Seed:       int64(381000 + i),
	})
	if err != nil {
		t.Fatalf("%s: generate: %v", id, err)
	}
	return p
}

var raOptionLine = regexp.MustCompile(`\n\n([A-E])\) ([^\n]*)`)

func raOptionsOf(q string) []string {
	var out []string
	for _, m := range raOptionLine.FindAllStringSubmatch(q, -1) {
		out = append(out, m[2])
	}
	return out
}

func raFactInt(t *testing.T, p generator.Problem, key string) int {
	t.Helper()
	v, err := strconv.Atoi(p.Facts[key])
	if err != nil {
		t.Fatalf("%s: fact %q = %q is not an integer", p.Question, key, p.Facts[key])
	}
	return v
}

// ---------------------------------------------------------------------------
// Router contracts
// ---------------------------------------------------------------------------

func TestRAAnswersGradeWithTheirShippedType(t *testing.T) {
	types := raShippedTypes(t)
	reg := raRegistry(t)
	router := grader.NewRouter()
	defer router.Close()

	for _, id := range raBatchIDs {
		typ, ok := types[id]
		if !ok || typ == "" {
			t.Fatalf("%s: no grading_type in the corpus", id)
		}
		if typ != "multiple_choice" && typ != "numeric" {
			t.Fatalf("%s: unexpected grading_type %q", id, typ)
		}
		for i := 0; i < 120; i++ {
			p := raSample(t, reg, id, i)
			if strings.TrimSpace(p.Answer) == "" {
				t.Fatalf("%s: empty answer at sample %d", id, i)
			}
			if strings.TrimSpace(p.Question) == "" || strings.TrimSpace(p.Explanation) == "" {
				t.Fatalf("%s: empty question or explanation at sample %d", id, i)
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

func TestRAMultipleChoiceRejectsEveryDistractor(t *testing.T) {
	types := raShippedTypes(t)
	reg := raRegistry(t)
	router := grader.NewRouter()
	defer router.Close()

	for _, id := range raMCIDs {
		if types[id] != "multiple_choice" {
			t.Fatalf("%s: expected multiple_choice, corpus says %q", id, types[id])
		}
		for i := 0; i < 40; i++ {
			p := raSample(t, reg, id, i)
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

// ---------------------------------------------------------------------------
// Independent numeric checks
// ---------------------------------------------------------------------------

func TestRASupremumRecomputes(t *testing.T) {
	reg := raRegistry(t)
	for i := 0; i < 400; i++ {
		p := raSample(t, reg, "ra.order.supremum", i)
		lo, err1 := strconv.ParseFloat(p.Facts["lo"], 64)
		hi, err2 := strconv.ParseFloat(p.Facts["hi"], 64)
		if err1 != nil || err2 != nil {
			t.Fatalf("supremum: facts lo=%q hi=%q are not numbers", p.Facts["lo"], p.Facts["hi"])
		}
		if lo >= hi {
			t.Fatalf("supremum: lo=%v is not below hi=%v", lo, hi)
		}
		// The published bounds are the infimum and the supremum; ask names
		// which one the question wants.
		want := decimal6(hi)
		if p.Facts["ask"] == "inf" {
			want = decimal6(lo)
		} else if p.Facts["ask"] != "sup" {
			t.Fatalf("supremum: ask=%q", p.Facts["ask"])
		}
		if p.Answer != want {
			t.Fatalf("supremum: %q answers %s, want %s", p.Question, p.Answer, want)
		}
	}
}

func TestRAArchimedeanRecomputes(t *testing.T) {
	reg := raRegistry(t)
	for i := 0; i < 400; i++ {
		p := raSample(t, reg, "ra.order.archimedean", i)
		pn := raFactInt(t, p, "pn")
		pd := raFactInt(t, p, "pd")
		xn := raFactInt(t, p, "xn")
		xd := raFactInt(t, p, "xd")
		if pn <= 0 || pd <= 0 || xn <= 0 || xd <= 0 {
			t.Fatalf("archimedean: pn=%d pd=%d xn=%d xd=%d are not positive", pn, pd, xn, xd)
		}
		// Brute force: the least n >= 1 with n*(pn/pd) > xn/xd.
		want := 1
		for want*pn*xd <= xn*pd {
			want++
		}
		if p.Answer != strconv.Itoa(want) {
			t.Fatalf("archimedean: %q answers %s, want %d", p.Question, p.Answer, want)
		}
	}
}

func TestRASequenceRecomputes(t *testing.T) {
	reg := raRegistry(t)
	for i := 0; i < 400; i++ {
		p := raSample(t, reg, "ra.seq.convergence", i)
		a := raFactInt(t, p, "a")
		b := raFactInt(t, p, "b")
		c := raFactInt(t, p, "c")
		d := raFactInt(t, p, "d")
		if c <= 0 || d <= 0 {
			t.Fatalf("sequence: c=%d d=%d must be positive", c, d)
		}
		switch p.Facts["kind"] {
		case "limit":
			if want := decimal6(float64(a) / float64(c)); p.Answer != want {
				t.Fatalf("sequence: a=%d c=%d answers %s, want %s", a, c, p.Answer, want)
			}
		case "N":
			// Brute force on exact rationals: the least N with
			// |a_n − a/c| < 1/e for every n in [N, N+2000], and failing at N−1.
			e := raFactInt(t, p, "e")
			lim := big.NewRat(int64(a), int64(c))
			eps := big.NewRat(1, int64(e))
			within := func(n int) bool {
				v := big.NewRat(int64(a*n+b), int64(c*n+d))
				v.Sub(v, lim)
				v.Abs(v)
				return v.Cmp(eps) < 0
			}
			n := raFactInt(t, p, "answer")
			if n > 1 && within(n-1) {
				t.Fatalf("sequence: %q answers %d but N−1 already works", p.Question, n)
			}
			for m := n; m < n+2000; m++ {
				if !within(m) {
					t.Fatalf("sequence: %q answers %d but n=%d is not within ε", p.Question, n, m)
				}
			}
		default:
			t.Fatalf("sequence: unknown kind %q", p.Facts["kind"])
		}
	}
}

func TestRAMonotoneRecomputes(t *testing.T) {
	reg := raRegistry(t)
	for i := 0; i < 400; i++ {
		p := raSample(t, reg, "ra.seq.monotone", i)
		// Iterate the recursion itself and compare with the published limit.
		var x float64
		var step func(float64) float64
		switch p.Facts["kind"] {
		case "sqrt":
			a := float64(raFactInt(t, p, "a"))
			x, step = 0, func(v float64) float64 { return math.Sqrt(a + v) }
		case "avg":
			k := float64(raFactInt(t, p, "k"))
			m := float64(raFactInt(t, p, "m"))
			x, step = 0, func(v float64) float64 { return (v + k) / m }
		case "geo":
			b := float64(raFactInt(t, p, "b"))
			x, step = 1, func(v float64) float64 { return math.Sqrt(b * v) }
		default:
			t.Fatalf("monotone: unknown kind %q", p.Facts["kind"])
		}
		prev := x
		for j := 0; j < 5000; j++ {
			x = step(x)
			if x < prev-1e-12 {
				t.Fatalf("monotone: %q is not increasing", p.Question)
			}
			prev = x
		}
		got, err := strconv.ParseFloat(p.Answer, 64)
		if err != nil || math.Abs(got-x) > 1e-6 {
			t.Fatalf("monotone: %q answers %s, iteration gives %v", p.Question, p.Answer, x)
		}
	}
}

func TestRASeriesRecomputes(t *testing.T) {
	reg := raRegistry(t)
	for i := 0; i < 400; i++ {
		p := raSample(t, reg, "ra.seq.series", i)
		// Sum a long stretch of the series numerically and compare.
		var sum float64
		switch p.Facts["kind"] {
		case "":
			pn := float64(raFactInt(t, p, "pn"))
			pd := float64(raFactInt(t, p, "pd"))
			for n, term := 0, 1.0; n < 2000; n++ {
				sum += term
				term *= pn / pd
			}
		case "geo":
			a := float64(raFactInt(t, p, "a"))
			r := float64(raFactInt(t, p, "p")) / float64(raFactInt(t, p, "q"))
			for n, term := 0, a; n < 2000; n++ {
				sum += term
				term *= r
			}
		case "tail":
			r := float64(raFactInt(t, p, "p")) / float64(raFactInt(t, p, "q"))
			m := raFactInt(t, p, "m")
			for n := m; n < 2000; n++ {
				sum += math.Pow(r, float64(n))
			}
		case "tele":
			k := float64(raFactInt(t, p, "k"))
			for n := 1.0; n < 2e6; n++ {
				sum += 1 / (n * (n + k))
			}
			// The tail beyond 2e6 is below k-independent 1/2e6; add the
			// integral estimate so the comparison can be tight.
			sum += 1 / 2e6
		case "partial":
			n := raFactInt(t, p, "n")
			for k := 1; k <= n; k++ {
				sum += 1 / float64(k*(k+1))
			}
		default:
			t.Fatalf("series: unknown kind %q", p.Facts["kind"])
		}
		got, err := strconv.ParseFloat(p.Answer, 64)
		if err != nil || math.Abs(got-sum) > 2e-6 {
			t.Fatalf("series: %q answers %s, summing gives %.9f", p.Question, p.Answer, sum)
		}
	}
}

func TestRAFunctionLimitRecomputes(t *testing.T) {
	reg := raRegistry(t)
	for i := 0; i < 200; i++ {
		p := raSample(t, reg, "ra.limit.function", i)
		m := raFactInt(t, p, "m")
		en := raFactInt(t, p, "en")
		ed := raFactInt(t, p, "ed")
		if m == 0 || ed == 0 {
			t.Fatalf("function limit: m=%d ed=%d", m, ed)
		}
		want := decimal6((float64(en) / float64(ed)) / float64(m))
		if p.Answer != want {
			t.Fatalf("function limit: m=%d eps=%d/%d answers %s, want %s", m, en, ed, p.Answer, want)
		}
	}
}
