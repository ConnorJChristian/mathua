package all

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
)

// The registry tidies question text ("1x + -3" → "x - 3"), so the learner
// sees and types the tidy form. Every stored answer must still accept that
// form, whether a generator grades its own answers or the router does.
func TestTidiedAnswersStillGrade(t *testing.T) {
	reg := Registry()
	types := conceptGradingTypes(t)
	router := grader.NewRouter()
	rand.Seed(7)
	checked := 0
	for _, id := range reg.Concepts() {
		gen, _ := reg.Get(id)
		for i := 0; i < 30; i++ {
			p, err := reg.GenerateContext(id, generator.GeneratorContext{Difficulty: []float64{0.3, 0.6, 0.9}[i%3], Seed: rand.Int63()})
			if err != nil {
				break
			}
			tidy := generator.TidyAnswer(p.Answer)
			if tidy == p.Answer {
				continue
			}
			checked++
			var res grader.Result
			if gg, ok := gen.(generator.GradedGenerator); ok {
				res = gg.Grade(p.Answer, tidy)
			} else {
				res = router.Grade(grader.GradingType(types[id]), p.Answer, tidy)
			}
			if !res.Correct {
				t.Errorf("%s: answer %q, tidy form %q graded wrong (%s)", id, p.Answer, tidy, types[id])
				break
			}
		}
	}
	t.Logf("%d tidied answers checked", checked)
}

var untidyRe = regexp.MustCompile(`[a-z0-9)}]\s*\+\s*-\s*\d|(^|[\s(=+\-])1[a-z]\^`)

// A ratchet on the sweep that found Q17: no served question shows "+ -3"
// or a "1x^" coefficient, outside the drills that mean it.
func TestServedQuestionsAreTidy(t *testing.T) {
	reg := Registry()
	rand.Seed(11)
	for _, id := range reg.Concepts() {
		if regexp.MustCompile(`^arith\.|integer|\.neg`).MatchString(id) {
			continue
		}
		for i := 0; i < 20; i++ {
			p, err := reg.GenerateContext(id, generator.GeneratorContext{Difficulty: []float64{0.3, 0.6, 0.9}[i%3], Seed: rand.Int63()})
			if err != nil {
				break
			}
			if p.Raw {
				break
			}
			if m := untidyRe.FindString(p.Question); m != "" && hasLetterVar(p.Question) {
				t.Errorf("%s: %q in %q", id, m, p.Question)
				break
			}
		}
	}
}

var letterVarRe = regexp.MustCompile(`\d[a-z]|[a-z]\^|[+\-=]\s*[a-z]\b`)

func hasLetterVar(s string) bool { return letterVarRe.MatchString(s) }

func conceptGradingTypes(t *testing.T) map[string]string {
	files, _ := filepath.Glob("../../../data/concepts/*.json")
	types := map[string]string{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var list []struct {
			ID          string `json:"id"`
			GradingType string `json:"grading_type"`
		}
		if json.Unmarshal(b, &list) != nil {
			continue
		}
		for _, c := range list {
			if _, ok := types[c.ID]; !ok {
				types[c.ID] = c.GradingType
			}
		}
	}
	return types
}
