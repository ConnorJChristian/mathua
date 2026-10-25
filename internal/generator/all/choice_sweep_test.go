package all

import (
	"math/rand"
	"regexp"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
)

var sweepOptionRe = regexp.MustCompile(`(?m)^([A-E])\) (.+)$`)

// Q44: for every served multiple-choice question, the learner's two ways of
// answering — clicking the right option (its text) and typing its letter —
// both grade right through the same path the engine uses.
func TestChoiceAnswersGradeRightAsClickedOrTyped(t *testing.T) {
	reg := Registry()
	types := conceptGradingTypes(t)
	router := grader.NewRouter()
	rand.Seed(44)
	for _, id := range reg.Concepts() {
		gen, _ := reg.Get(id)
		for i := 0; i < 12; i++ {
			p, err := reg.GenerateContext(id, generator.GeneratorContext{Difficulty: []float64{0.3, 0.6, 0.9}[i%3], Seed: rand.Int63()})
			if err != nil || grader.IsBankAnswer(p.Answer) {
				break
			}
			opts := sweepOptionRe.FindAllStringSubmatch(p.Question, -1)
			letter := ""
			for _, m := range opts {
				if strings.TrimSpace(m[2]) == strings.TrimSpace(p.Answer) {
					letter = m[1]
				}
			}
			if letter == "" {
				continue
			}
			grade := func(typed string) bool {
				a := grader.ResolveChoiceLetter(p.Question, p.Answer, typed)
				if gg, ok := gen.(generator.GradedGenerator); ok {
					return gg.Grade(p.Answer, a).Correct
				}
				return router.Grade(grader.GradingType(types[id]), p.Answer, a).Correct
			}
			if !grade(p.Answer) {
				t.Errorf("%s: clicking the right option %q graded wrong", id, p.Answer)
				break
			}
			if !grade(letter) {
				t.Errorf("%s: typing the right letter %s (%q) graded wrong", id, letter, p.Answer)
				break
			}
		}
	}
}
