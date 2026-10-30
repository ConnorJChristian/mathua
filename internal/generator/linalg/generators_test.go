package linalg

import (
	"math/rand"
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

func TestVectorConceptGen(t *testing.T) { fuzzGen(t, &vectorConceptGen{}) }

// The concept draws one of three questions (upstream eabb8e1a picks them by
// knowledge point; the fork has none, so it draws). Each kind must come up and
// each must answer itself.
func TestVectorConceptGenDrawsAllThreeQuestions(t *testing.T) {
	g := &vectorConceptGen{}
	seen := map[string]int{}
	for i := 0; i < 300; i++ {
		p := g.Generate(generator.GeneratorContext{Difficulty: rand.Float64()})
		switch {
		case strings.Contains(p.Question, "magnitude of vector"):
			seen["magnitude"]++
		case strings.Contains(p.Question, "points in the direction of"):
			seen["component"]++
			x, _ := strconv.Atoi(p.Facts["x"])
			scale, _ := strconv.Atoi(p.Facts["scale"])
			mag, _ := strconv.Atoi(p.Facts["magnitude"])
			y, _ := strconv.Atoi(p.Facts["y"])
			if x*x+y*y != mag*mag {
				t.Fatalf("direction (%d,%d) does not have magnitude %d", x, y, mag)
			}
			if want := strconv.Itoa(x * scale); p.Answer != want {
				t.Fatalf("component answer %q, want x*scale %q: %q", p.Answer, want, p.Question)
			}
		case strings.Contains(p.Question, `\mathbf{u}`) && strings.Contains(p.Question, `\mathbf{v}`):
			seen["operation"]++
			u1, _ := strconv.Atoi(p.Facts["u1"])
			v1, _ := strconv.Atoi(p.Facts["v1"])
			if want := strconv.Itoa(u1 + v1); p.Answer != want {
				t.Fatalf("sum answer %q, want u1+v1 %q: %q", p.Answer, want, p.Question)
			}
		default:
			t.Fatalf("unexpected vector question: %q", p.Question)
		}
	}
	for _, kind := range []string{"magnitude", "component", "operation"} {
		if seen[kind] == 0 {
			t.Errorf("no %s question in 300 draws: %v", kind, seen)
		}
	}
}
func TestVectorAddGen(t *testing.T)        { fuzzGen(t, &vectorAddGen{}) }
func TestVectorDotGen(t *testing.T)        { fuzzGen(t, &vectorDotGen{}) }
func TestMatrixConceptGen(t *testing.T)    { fuzzGen(t, &matrixConceptGen{}) }
func TestMatrixAddGen(t *testing.T)        { fuzzGen(t, &matrixAddGen{}) }
func TestMatrixMultGen(t *testing.T)       { fuzzGen(t, &matrixMultGen{}) }
func TestMatrixIdentityGen(t *testing.T)   { fuzzGen(t, &matrixIdentityGen{}) }
func TestDet2x2Gen(t *testing.T)           { fuzzGen(t, &det2x2Gen{}) }
func TestDet3x3Gen(t *testing.T)           { fuzzGen(t, &det3x3Gen{}) }
func TestSystemsMatrixGen(t *testing.T)    { fuzzGen(t, &systemsMatrixGen{}) }
func TestCramerGen(t *testing.T)           { fuzzGen(t, &cramerGen{}) }
func TestEigenConceptGen(t *testing.T)     { fuzzGen(t, &eigenConceptGen{}) }
func TestEigenComputeGen(t *testing.T)     { fuzzGen(t, &eigenComputeGen{}) }
func TestTransformationsGen(t *testing.T)  { fuzzGen(t, &transformationsGen{}) }
func TestSpanGen(t *testing.T)             { fuzzGen(t, &spanGen{}) }
func TestBasisGen(t *testing.T)            { fuzzGen(t, &basisGen{}) }
func TestDiagonalizationGen(t *testing.T)  { fuzzGen(t, &diagonalizationGen{}) }
func TestRankGen(t *testing.T)             { fuzzGen(t, &rankGen{}) }
func TestCosineSimilarityGen(t *testing.T) { fuzzGen(t, &cosineSimilarityGen{}) }
func TestParametricGen(t *testing.T)       { fuzzGen(t, &parametricGen{}) }

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

func TestRandomMatrixCoefficientsAreNonsingular(t *testing.T) {
	for i := 0; i < 500; i++ {
		scale := 1 + rand.Intn(5)
		a, b, c, d := randomMatrixCoefficients(scale)
		if a*d-b*c == 0 {
			t.Fatalf("singular matrix from scale %d: [[%d,%d],[%d,%d]]", scale, a, b, c, d)
		}
	}
}
