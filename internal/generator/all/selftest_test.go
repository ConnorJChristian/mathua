package all

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
)

// doubledCommand matches a LaTeX control sequence that was written with two
// backslashes instead of one. A row separator inside a matrix or a cases block
// is also two backslashes, but a separator is followed by a space or by the
// next row's first variable, never by a command name, so requiring two letters
// tells them apart (`\\y` is a row break, `\\nabla` is a doubled command).
//
// The corpus shipped `\\mathbf{F}` and `\\nabla` in three calculus stems and
// `\\equiv` in a congruence stem, so a learner read the backslashes and the
// command name instead of the symbol (report 127).
var doubledCommand = regexp.MustCompile(`\\\\[A-Za-z]{2,}`)

// loadGradingTypes reads every concept file and returns grading_type by id.
func loadGradingTypes(t *testing.T) map[string]grader.GradingType {
	t.Helper()
	files, err := filepath.Glob("../../../data/concepts/*.json")
	if err != nil || len(files) == 0 {
		t.Fatalf("no concept files: %v", err)
	}
	out := map[string]grader.GradingType{}
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		type entry struct {
			ID          string `json:"id"`
			GradingType string `json:"grading_type"`
		}
		var list []entry
		if err := json.Unmarshal(raw, &list); err != nil {
			var wrapper struct {
				Concepts []entry `json:"concepts"`
			}
			if err := json.Unmarshal(raw, &wrapper); err != nil {
				t.Fatalf("%s: %v", file, err)
			}
			list = wrapper.Concepts
		}
		for _, c := range list {
			out[c.ID] = grader.GradingType(c.GradingType)
		}
	}
	if len(out) < 1000 {
		t.Fatalf("read only %d concepts, so the sweep would prove nothing", len(out))
	}
	return out
}

// gradeLikeTheEngine mirrors Engine.gradeAnswer: a generator that carries its
// own Grade decides the mark, otherwise the router decides by grading type.
func gradeLikeTheEngine(t *testing.T, reg *generator.Registry, types map[string]grader.GradingType, id string, p generator.Problem) (grader.Result, bool) {
	t.Helper()
	if gen, err := reg.Get(id); err == nil {
		if custom, ok := gen.(interface {
			Grade(string, string) grader.Result
		}); ok {
			return custom.Grade(p.Answer, p.Answer), true
		}
	}
	return grader.NewRouter().Grade(types[id], p.Answer, p.Answer), true
}

// TestEveryConceptGradesItsOwnAnswer is the check that the corpus is
// answerable at all. Every generated problem is graded against its own answer,
// the way the engine grades a learner's. A concept that fails cannot be
// answered correctly by anybody: linalg.matrix.add once answered with a matrix
// literal while the numeric router could not read one, and only the custom
// grader kept it alive.
func TestEveryConceptGradesItsOwnAnswer(t *testing.T) {
	reg := generator.NewRegistry()
	Register(reg)
	types := loadGradingTypes(t)

	checked := 0
	skipped := 0
	for _, id := range reg.Concepts() {
		if types[id] == "" {
			continue
		}
		for i := 0; i < 3; i++ {
			p, err := reg.GenerateContext(id, generator.GeneratorContext{Difficulty: 0.5, Seed: int64(i) + 1})
			if err != nil {
				t.Errorf("%s: generate: %v", id, err)
				break
			}
			res, _ := gradeLikeTheEngine(t, reg, types, id, p)
			if res.Unavailable {
				// SymPy is not installed in every environment; that is an
				// infrastructure gap rather than an unanswerable question.
				skipped++
				break
			}
			checked++
			if !res.Correct {
				t.Errorf("%s (%s) cannot grade its own answer %q: %s\nquestion: %s",
					id, types[id], p.Answer, res.Feedback, p.Question)
				break
			}
		}
	}
	if checked < 1000 {
		t.Fatalf("checked only %d concepts (%d skipped), so the sweep proved nothing", checked, skipped)
	}
	t.Logf("graded %d problems across the corpus, %d skipped for a missing symbolic grader", checked, skipped)
}

// TestNoQuestionPrintsADoubledLatexCommand guards the class of defect where a
// generator writes `\\mathbf` in Go source and the learner sees the backslashes,
// because the extra escape reaches LaTeX as a line break followed by the
// command's name in italic.
func TestNoQuestionPrintsADoubledLatexCommand(t *testing.T) {
	reg := generator.NewRegistry()
	Register(reg)

	checked := 0
	for _, id := range reg.Concepts() {
		for i := 0; i < 4; i++ {
			p, err := reg.GenerateContext(id, generator.GeneratorContext{Difficulty: 0.6, Seed: int64(i) + 1})
			if err != nil {
				t.Errorf("%s: generate: %v", id, err)
				break
			}
			checked++
			if where := doubledCommand.FindString(p.Question); where != "" {
				t.Errorf("%s: the question prints %q, which is a command with one backslash too many: %s", id, where, p.Question)
			}
			if where := doubledCommand.FindString(p.Explanation); where != "" {
				t.Errorf("%s: the explanation prints %q, which is a command with one backslash too many: %s", id, where, p.Explanation)
			}
		}
	}
	if checked < 1000 {
		t.Fatalf("checked only %d problems, so the sweep proved nothing", checked)
	}
}
