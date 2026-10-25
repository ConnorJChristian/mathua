package discrete

import (
	"math/rand"
	"regexp"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
)

// distinctQuestions mirrors all.distinctQuestionLowerBound: the smallest number of distinct
// question texts seen across a difficulty sweep.
func distinctQuestions(gen generator.Generator) int {
	best := -1
	for _, d := range []float64{0.3, 0.5, 0.6, 0.8, 0.9, 1.0} {
		seen := map[string]bool{}
		for i := 0; i < 300; i++ {
			seed := int64(i)*2654435761 + int64(d*1000)
			rand.Seed(seed)
			p := gen.Generate(generator.GeneratorContext{Difficulty: d, Seed: seed})
			seen[p.Question] = true
		}
		if n := len(seen); best < 0 || n < best {
			best = n
		}
	}
	return best
}

// TestWidenedPoolsHaveVariety guards the concepts whose pools were widened so a learner does
// not see the same handful of items over and over.
func TestWidenedPoolsHaveVariety(t *testing.T) {
	reg := generator.NewRegistry()
	Register(reg)
	for _, id := range []string{
		"discrete.proof.contradiction",
	} {
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

// optionLetterWord is a stand-alone word "a" to "e": the choice grader accepts a typed option
// letter when it appears as a word in the key, so a key containing one would accept a wrong letter.
var optionLetterWord = regexp.MustCompile(`(?i)\b[a-e]\b`)

func TestContradictionChoicesGrade(t *testing.T) {
	router := grader.NewRouter()
	defer router.Close()
	for i := 0; i < 80; i++ {
		p := contradictionChoice()
		var opts []string
		for _, m := range batchOptionLine.FindAllStringSubmatch(p.Question, -1) {
			opts = append(opts, m[2])
		}
		if len(opts) != 4 {
			t.Fatalf("want 4 options, got %v in %q", opts, p.Question)
		}
		if optionLetterWord.MatchString(p.Answer) {
			t.Fatalf("key %q contains a stand-alone option letter", p.Answer)
		}
		found := 0
		for _, o := range opts {
			res := router.Grade(grader.GradingMultipleChoice, p.Answer, o)
			if o == p.Answer {
				found++
				if !res.Correct {
					t.Fatalf("key %q does not grade as correct", p.Answer)
				}
			} else if res.Correct {
				t.Fatalf("distractor %q accepted for key %q", o, p.Answer)
			}
		}
		if found != 1 {
			t.Fatalf("key %q appears %d times among %v", p.Answer, found, opts)
		}
		if p.Explanation == "" {
			t.Fatalf("no explanation for %q", p.Question)
		}
	}
}
