package fractions

import (
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
)

func fuzzGen(t *testing.T, gen generator.Generator) {
	t.Helper()
	for i := 0; i < 100; i++ {
		d := rand.Float64()
		p := gen.Generate(generator.GeneratorContext{Difficulty: d})
		if p.Question == "" || p.Answer == "" || p.Explanation == "" {
			t.Errorf("empty field at difficulty=%.2f: q=%q a=%q e=%q", d, p.Question, p.Answer, p.Explanation)
		}
	}
}

func TestFracConceptGen(t *testing.T)      { fuzzGen(t, &fracConceptGen{}) }
func TestFracPartsGen(t *testing.T)        { fuzzGen(t, &fracPartsGen{}) }
func TestFracNumberLineGen(t *testing.T)   { fuzzGen(t, &fracNumberLineGen{}) }
func TestFracEquivalentGen(t *testing.T)   { fuzzGen(t, &fracEquivalentGen{}) }
func TestFracSimplifyGen(t *testing.T)     { fuzzGen(t, &fracSimplifyGen{}) }
func TestFracCompareGen(t *testing.T)      { fuzzGen(t, &fracCompareGen{}) }
func TestFracBenchmarkGen(t *testing.T)    { fuzzGen(t, &fracBenchmarkGen{}) }
func TestFracToDecimalGen(t *testing.T)    { fuzzGen(t, &fracToDecimalGen{}) }
func TestFracOpSameDenGen(t *testing.T)    { fuzzGen(t, &fracOpSameDenGen{}) }
func TestFracOpDiffDenGen(t *testing.T)    { fuzzGen(t, &fracOpDiffDenGen{}) }
func TestFracWordGen(t *testing.T)         { fuzzGen(t, &fracWordGen{}) }
func TestFracMultGen(t *testing.T)         { fuzzGen(t, &fracMultGen{}) }
func TestFracDivGen(t *testing.T)          { fuzzGen(t, &fracDivGen{}) }
func TestFracMixedConvertGen(t *testing.T) { fuzzGen(t, &fracMixedConvertGen{}) }
func TestFracMixedOpGen(t *testing.T)      { fuzzGen(t, &fracMixedOpGen{}) }
func TestFracMixedMultGen(t *testing.T)    { fuzzGen(t, &fracMixedMultGen{}) }

func TestFuzz(t *testing.T) {
	reg := generator.NewRegistry()
	Register(reg)
	for _, id := range reg.Concepts() {
		gen, _ := reg.Get(id)
		t.Run(id, func(t *testing.T) {
			for i := 0; i < 100; i++ {
				d := rand.Float64()
				p := gen.Generate(generator.GeneratorContext{Difficulty: d})
				if p.Question == "" || p.Answer == "" || p.Explanation == "" {
					t.Fatalf("empty field at difficulty=%.2f for %s: q=%q a=%q", d, id, p.Question, p.Answer)
				}
			}
		})
	}
}

func TestFracSubWordNamesTheWholeCake(t *testing.T) {
	g := &fracSubWordGen{}
	for i := 0; i < 200; i++ {
		p := g.Generate(generator.GeneratorContext{Difficulty: rand.Float64()})
		if !strings.Contains(p.Question, "of the whole cake") {
			t.Fatalf("question does not say the subtrahend is of the whole cake: %q", p.Question)
		}
		if !strings.Contains(p.Question, "How much of the whole cake is left?") {
			t.Fatalf("question does not state the quantity being asked for: %q", p.Question)
		}
	}
}

func TestCrossComparePrintedProductsMatchFacts(t *testing.T) {
	g := &crossCompareGen{}
	re := regexp.MustCompile(`\\\((\d+) \\times (\d+) - (\d+) \\times (\d+)\\\)`)
	for i := 0; i < 500; i++ {
		p := g.Generate(generator.GeneratorContext{Difficulty: rand.Float64()})
		m := re.FindStringSubmatch(p.Question)
		if m == nil {
			t.Fatalf("question does not show the cross products: %q", p.Question)
		}
		nums := make([]int, 4)
		for j, raw := range m[1:] {
			v, err := strconv.Atoi(raw)
			if err != nil {
				t.Fatalf("parse %q: %v", raw, err)
			}
			nums[j] = v
		}
		if got, want := strconv.Itoa(nums[0]*nums[1]), p.Facts["left"]; got != want {
			t.Fatalf("printed first product is %s, facts say %s: %q", got, want, p.Question)
		}
		if got, want := strconv.Itoa(nums[2]*nums[3]), p.Facts["right"]; got != want {
			t.Fatalf("printed second product is %s, facts say %s: %q", got, want, p.Question)
		}
		if want := p.Facts["difference"] + "/1"; p.Answer != want {
			t.Fatalf("answer %q, want %q", p.Answer, want)
		}
	}
}
