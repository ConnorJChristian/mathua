package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/modules"
	"github.com/chuma-beep/mathua/internal/pytrack"
)

// diffGen is a generated topic that says the difficulty it was asked for.
type diffGen struct{}

func (diffGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return generator.Problem{Question: fmt.Sprintf("generated at %.2f #%d", ctx.Difficulty, generator.StepOfDifficulty(ctx.Difficulty)), Answer: "1"}
}

// stepHistory is a mapHistory that also knows the generated steps done.
type stepHistory struct {
	mapHistory
	genTop int
}

func (h stepHistory) GeneratedCorrectAt(step int) bool { return h.genTop >= step }

// A generated topic (derivative rules) with cards filed under it: a level-1
// card and a level-2 book exercise. A fresh learner gets step 1 (generated
// questions at the easy end, or the level-1 card), never the book exercise;
// once step 1 has a right answer the book exercise comes first.
func TestScaffold_GeneratedTopicWithFiledBookItem(t *testing.T) {
	bank := `{"chapter":"d","title":"D","concepts":[{"id":"deck.d","book":{"title":"D"},"intro":"","items":[
	 {"id":"card1","kind":"math","level":1,"prompt":"Differentiate x^4.","parts":[{"answer":"4*x^3","grading":"expression","tex":"4x^3"}],"explain":"."},
	 {"id":"book2","kind":"math","level":2,"prompt":"Enter the slope.","parts":[{"answer":"6","grading":"numeric","tex":"6"}],
	  "book_ref":{"file":"b.pdf","module":"m","page":5,"label":"Book Exercise 4.1, #2"},"explain":"."}]}]}`
	dir := filepath.Join(t.TempDir(), "m", "bank")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "d.json"), []byte(bank), 0o644)
	b, err := pytrack.LoadBank(dir)
	if err != nil {
		t.Fatal(err)
	}
	th := concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 30}
	d, err := concepts.Build([]concepts.Concept{
		{ID: "calc.rules", Label: "Rules", Domain: "calculus", GradingType: "numeric", MasteryThreshold: th},
		{ID: "deck.d", Label: "Deck", Domain: "calculus", GradingType: "bank", Prerequisites: []string{"calc.rules"}, MasteryThreshold: th},
	})
	if err != nil {
		t.Fatal(err)
	}
	reg := generator.NewRegistry()
	reg.Register("calc.rules", diffGen{})
	if err := (&pytrack.Track{Bank: b}).Register(reg); err != nil {
		t.Fatal(err)
	}
	attachCardsFrom(d, reg, map[string][]modules.CardRef{"calc.rules": {{Deck: "deck.d", Item: "card1"}, {Deck: "deck.d", Item: "book2"}}})

	fresh := stepHistory{mapHistory: mapHistory{}}
	for i := 0; i < 60; i++ {
		p, _ := reg.GenerateContext("calc.rules", generator.GeneratorContext{Difficulty: 0.85, History: fresh})
		if strings.Contains(p.Answer, "book2") {
			t.Fatal("fresh learner served the step-2 book exercise")
		}
		if strings.HasPrefix(p.Question, "generated") && !strings.HasSuffix(p.Question, "#1") {
			t.Fatalf("fresh learner served a generated question above step 1: %q", p.Question)
		}
		if strings.HasPrefix(p.Question, "generated") && (p.Difficulty == nil || *p.Difficulty > 0.45) {
			t.Fatalf("the attempt must record the step served, got %v", p.Difficulty)
		}
	}
	climbed := stepHistory{mapHistory: mapHistory{}, genTop: 1}
	p, _ := reg.GenerateContext("calc.rules", generator.GeneratorContext{Difficulty: 0.4, History: climbed})
	if !strings.Contains(p.Answer, "book2") {
		t.Fatalf("step 1 right: the book exercise next, got %q / %q", p.Question, p.Answer)
	}
}
