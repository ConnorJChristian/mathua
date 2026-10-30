package pytrack

import (
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
)

// A topic whose bank lists a book word problem (level 3) first, then a
// worked book case (level 2) with a practice version unlocking after it, a
// plain worked case and two "apply the rule" items (level 1).
const scaffoldBank = `{
 "chapter": "t", "title": "T",
 "concepts": [{
  "id": "t.rule", "book": {"title": "Rule"}, "intro": "",
  "items": [
   {"id": "word", "kind": "math", "level": 3, "prompt": "Enter the firm's output.",
    "parts": [{"answer": "30", "grading": "numeric", "tex": "30"}],
    "book_ref": {"file": "book.pdf", "module": "mod", "page": 9, "label": "Book Exercise 1.2, #9"}, "explain": "."},
   {"id": "case", "kind": "math", "level": 2, "prompt": "Enter the derivative at 2.",
    "parts": [{"answer": "12", "grading": "numeric", "tex": "12"}],
    "book_ref": {"file": "book.pdf", "module": "mod", "page": 8, "label": "Book Exercise 1.2, #3"}, "explain": "."},
   {"id": "case_pr", "kind": "math", "level": 2, "prompt": "Differentiate x^3 at 1.",
    "parts": [{"answer": "3", "grading": "numeric", "tex": "3"}], "unlocks_after": "case", "explain": "."},
   {"id": "case2", "kind": "math", "level": 2, "prompt": "Differentiate 2x^2 at 3.",
    "parts": [{"answer": "12", "grading": "numeric", "tex": "12"}], "explain": "."},
   {"id": "apply1", "kind": "math", "level": 1, "prompt": "Differentiate x^2.",
    "parts": [{"answer": "2*x", "grading": "expression", "tex": "2x"}], "explain": "."},
   {"id": "apply2", "kind": "math", "level": 1, "prompt": "Differentiate x^5.",
    "parts": [{"answer": "5*x^4", "grading": "expression", "tex": "5x^4"}], "explain": "."}
  ]
 }]
}`

func itemOf(p generator.Problem) string { return p.Answer[strings.Index(p.Answer, "#")+1:] }

// A fresh learner climbs: the rule first (never the book word problem the
// bank lists first), then the book's worked case before the bank's own,
// then the word problem; practice unlocks only after its book exercise.
func TestScaffold_BankTopicClimbs(t *testing.T) {
	b, err := LoadBank(writeModule(t, scaffoldBank, false))
	if err != nil {
		t.Fatal(err)
	}
	g := &conceptGen{track: &Track{Bank: b}, c: b.Concept("t.rule")}
	key := func(id string) string { return "math:t.rule#" + id }
	h := fakeHistory{correct: map[string]bool{}, tried: map[string]bool{}}
	serve := func() string { return itemOf(g.Generate(generator.GeneratorContext{Difficulty: 0.9, History: h})) }

	for i := 0; i < 30; i++ {
		if id := serve(); id != "apply1" && id != "apply2" {
			t.Fatalf("fresh learner: want an apply-the-rule item, got %s", id)
		}
	}
	h.correct[key("apply1")] = true
	if id := serve(); id != "case" {
		t.Fatalf("step 1 right: the book's worked case first, got %s", id)
	}
	h.tried[key("case")] = true // missed it
	got := map[string]int{}
	for i := 0; i < 200; i++ {
		got[serve()]++
	}
	if got["word"] > 0 || got["case_pr"] > 0 || got["case"] == 0 || got["case2"] == 0 {
		t.Fatalf("book case missed: step 2 continues (case again, case2), no word problem, no locked practice: %v", got)
	}
	h.correct[key("case")] = true
	if id := serve(); id != "word" {
		t.Fatalf("step 2 right: the word problem, got %s", id)
	}

	// A lesson batch for a fresh learner fills from step 1 and climbs in
	// order when the step runs out.
	reg := generator.NewRegistry()
	reg.Register("t.rule", g)
	batch, err := reg.BatchGenerateContext("t.rule", 4, generator.GeneratorContext{Difficulty: 0.4, History: fakeHistory{}})
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, p := range batch {
		order = append(order, itemOf(p))
	}
	if len(order) != 4 || !strings.HasPrefix(order[0], "apply") || !strings.HasPrefix(order[1], "apply") || order[2] != "case" || order[3] != "case2" {
		t.Errorf("batch order %v, want the two apply items, then the book case, then case2", order)
	}

	// A review of the topic samples across the steps.
	h.correct[key("word")] = true
	seen := map[string]bool{}
	for i := 0; i < 300; i++ {
		seen[itemOf(g.Generate(generator.GeneratorContext{Difficulty: 0.6, History: h, Review: true}))] = true
	}
	if !seen["case2"] || !seen["apply1"] || !seen["word"] {
		t.Errorf("a review should sample across the steps: %v", seen)
	}
}
