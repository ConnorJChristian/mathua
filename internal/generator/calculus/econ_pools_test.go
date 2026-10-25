package calculus

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
)

// The ECON 5153 topics below served three to five distinct questions each, so
// a learner could not collect six distinct recent answers for mastery without
// repeats. Their pools were widened with parameterised variants.
var econPoolIDs = []string{
	"calc.deriv.linear_approx", "calc.deriv.total_diff", "calc.multi.lagrange_two",
	"calc.deriv.global_extrema", "calc.deriv.inverse_func", "calc.deriv.log_diff",
	"calc.deriv.partial_higher", "calc.deriv.second_test", "calc.integral.ftc",
}

func TestEconPoolsHaveVariety(t *testing.T) {
	reg := generator.NewRegistry()
	Register(reg)
	for _, id := range econPoolIDs {
		gen, err := reg.Get(id)
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		n := distinctQuestions(gen)
		t.Logf("%s: %d distinct questions", id, n)
		if n < 12 {
			t.Errorf("%s: only %d distinct questions, want >= 12", id, n)
		}
	}
}

// Every generated answer must grade correct against itself and against the
// same answer typed without explicit multiplication signs, and a shifted
// answer must be rejected, under the grading type the corpus ships.
func TestEconPoolAnswersGradeAsTyped(t *testing.T) {
	types := calcShippedTypes(t)
	reg := generator.NewRegistry()
	Register(reg)
	router := grader.NewRouter()
	defer router.Close()
	rand.Seed(5153)
	for _, id := range econPoolIDs {
		typ := grader.GradingType(types[id])
		seen := map[string]bool{}
		for i := 0; i < 400 && len(seen) < 60; i++ {
			p, err := reg.GenerateContext(id, generator.GeneratorContext{Difficulty: []float64{0.3, 0.6, 0.9}[i%3], Seed: rand.Int63()})
			if err != nil {
				t.Fatalf("%s: %v", id, err)
			}
			if seen[p.Question] {
				continue
			}
			seen[p.Question] = true
			if strings.TrimSpace(p.Explanation) == "" {
				t.Errorf("%s: empty explanation for %q", id, p.Question)
			}
			res := router.Grade(typ, p.Answer, p.Answer)
			if res.Unavailable {
				t.Skipf("grader unavailable")
			}
			if !res.Correct {
				t.Errorf("%s: answer %q does not grade against itself", id, p.Answer)
				continue
			}
			if typ == grader.GradingMultipleChoice {
				continue
			}
			if implicit := strings.ReplaceAll(p.Answer, "*", ""); implicit != p.Answer {
				if r := router.Grade(typ, p.Answer, implicit); !r.Correct && !r.Unavailable {
					t.Errorf("%s: answer %q rejects the typed form %q", id, p.Answer, implicit)
				}
			}
			if !strings.Contains(p.Answer, "=") && !isWord(p.Answer) {
				if r := router.Grade(typ, p.Answer, "("+p.Answer+") + 1"); r.Correct {
					t.Errorf("%s: answer %q accepts the wrong answer (%s) + 1", id, p.Answer, p.Answer)
				}
			}
		}
	}
}

func isWord(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && r != ' ' {
			return false
		}
	}
	return true
}
