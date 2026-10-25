package realanalysis

import (
	"sort"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
)

// raMinDistinct is the number of distinct question texts every real-analysis
// concept must supply at each difficulty, so a learner working through a
// section does not meet the same item twice before the exclude list runs out.
const raMinDistinct = 12

// raDistinctAt counts distinct question texts in n draws at one difficulty.
func raDistinctAt(reg *generator.Registry, id string, difficulty float64, n int) int {
	gen, err := reg.Get(id)
	if err != nil {
		return 0
	}
	seen := map[string]bool{}
	for i := 0; i < n; i++ {
		p := gen.Generate(generator.GeneratorContext{Difficulty: difficulty, Seed: int64(i)})
		seen[p.Question] = true
	}
	return len(seen)
}

func TestRAEveryConceptHasVariety(t *testing.T) {
	reg := raRegistry(t)
	ids := reg.Concepts()
	sort.Strings(ids)
	for _, id := range ids {
		lowest := -1
		for _, d := range []float64{0.3, 0.5, 0.8, 1.0} {
			if n := raDistinctAt(reg, id, d, 600); lowest < 0 || n < lowest {
				lowest = n
			}
		}
		t.Logf("%-28s %d", id, lowest)
		if lowest < raMinDistinct {
			t.Errorf("%s: only %d distinct questions, want at least %d", id, lowest, raMinDistinct)
		}
	}
}

// TestRAVariantsAreWellFormed sweeps every real-analysis concept far more
// densely than the router tests, so that each parameterised family is hit
// many times: multiple-choice items must show at least three options with the
// answer exactly once and every distractor rejected, numeric answers must grade
// as themselves, and no text may carry a bare dollar sign.
func TestRAVariantsAreWellFormed(t *testing.T) {
	types := raShippedTypes(t)
	reg := raRegistry(t)
	router := grader.NewRouter()
	defer router.Close()
	ids := reg.Concepts()
	sort.Strings(ids)
	for _, id := range ids {
		typ := types[id]
		for i := 0; i < 300; i++ {
			p := raSample(t, reg, id, 7000+i)
			for _, s := range []string{p.Question, p.Answer, p.Explanation} {
				if strings.Contains(s, "$") {
					t.Fatalf("%s: bare dollar sign in %q", id, s)
				}
			}
			if strings.TrimSpace(p.Explanation) == "" {
				t.Fatalf("%s: empty explanation for %q", id, p.Question)
			}
			if typ == "numeric" {
				if res := router.Grade(grader.GradingNumeric, p.Answer, p.Answer); !res.Unavailable && !res.Correct {
					t.Fatalf("%s: answer %q does not grade as itself", id, p.Answer)
				}
				continue
			}
			opts := raOptionsOf(p.Question)
			if len(opts) < 3 {
				t.Fatalf("%s: %d options in %q", id, len(opts), p.Question)
			}
			found := 0
			for _, o := range opts {
				if o == p.Answer {
					found++
				} else if router.Grade(grader.GradingMultipleChoice, p.Answer, o).Correct {
					t.Fatalf("%s: distractor %q accepted for %q", id, o, p.Answer)
				}
			}
			if found != 1 {
				t.Fatalf("%s: answer %q appears %d times in %q", id, p.Answer, found, p.Question)
			}
		}
	}
}
