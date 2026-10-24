package engine

import (
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
	"github.com/chuma-beep/mathua/internal/modules"
	"github.com/chuma-beep/mathua/internal/storage"
)

type cardStub struct{}

func (cardStub) Generate(generator.GeneratorContext) generator.Problem {
	return generator.Problem{Question: "front <!--recall-back--> back", Answer: "recall:anki.a#1"}
}
func (cardStub) PlainText() bool { return true }

// A deck's cards come up in the practice of the generated topic it covers,
// never in placement, and grade as cards.
func TestCardsServedInTopicPractice(t *testing.T) {
	t.Setenv("MATHUA_CARD_SHARE", "1")
	th := concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 60}
	d, err := concepts.Build([]concepts.Concept{
		{ID: "a", Label: "A", Domain: "d", GradingType: "numeric", MasteryThreshold: th},
		{ID: "anki.a", Label: "Practice · A", Domain: "d", GradingType: "bank", Prerequisites: []string{"a"}, MasteryThreshold: th},
	})
	if err != nil {
		t.Fatal(err)
	}
	reg := generator.NewRegistry()
	reg.Register("a", &testGen{answer: "42"})
	reg.Register("anki.a", cardStub{})
	store, err := storage.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	e := New(store, d, reg, nil, nil)
	New(store, d, reg, nil, nil) // a second engine on the same registry must not double-wrap

	p, err := reg.GenerateContext("a", generator.GeneratorContext{Difficulty: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(p.Question, CardMarker) || p.Answer != "recall:anki.a#1" {
		t.Fatalf("want a card in the topic's practice, got %q / %q", p.Question, p.Answer)
	}
	if strings.Count(p.Question, CardMarker) != 1 {
		t.Fatalf("card marked more than once: %q", p.Question)
	}
	p, _ = reg.GenerateContext("a", generator.GeneratorContext{Difficulty: 0.5, NoCards: true})
	if p.Answer != "42" {
		t.Fatalf("placement must get the generated question, got %q", p.Answer)
	}
	// Graded as a card (the bank grader), not as a number.
	if !grader.IsBankAnswer("recall:anki.a#1") || grader.IsBankAnswer("42") {
		t.Fatal("IsBankAnswer")
	}
	res := e.GradeAnswer("a", "42", "42")
	if !res.Correct {
		t.Fatalf("generated question still grades numerically: %+v", res)
	}
}

type deckStub struct{ items map[string]string }

func (d deckStub) Generate(generator.GeneratorContext) generator.Problem {
	return generator.Problem{Question: "whole deck <!--recall-back--> x", Answer: "recall:deck#any"}
}
func (d deckStub) PlainText() bool { return true }
func (d deckStub) ProblemFor(id string) (generator.Problem, bool) {
	q, ok := d.items[id]
	if !ok {
		return generator.Problem{}, false
	}
	return generator.Problem{Question: q + " <!--recall-back--> back", Answer: "recall:deck#" + id}, true
}

// A topic with cards filed under it (cards.json) serves exactly those cards,
// not the whole deck.
func TestFiledCardsServeOnlyTheirTopic(t *testing.T) {
	t.Setenv("MATHUA_CARD_SHARE", "1")
	th := concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 60}
	d, _ := concepts.Build([]concepts.Concept{
		{ID: "a", Label: "A", Domain: "d", GradingType: "numeric", MasteryThreshold: th},
		{ID: "b", Label: "B", Domain: "d", GradingType: "numeric", MasteryThreshold: th},
		{ID: "anki.deck", Label: "Practice · deck", Domain: "d", GradingType: "bank", Prerequisites: []string{"a"}, MasteryThreshold: th},
	})
	reg := generator.NewRegistry()
	reg.Register("a", &testGen{answer: "1"})
	reg.Register("b", &testGen{answer: "2"})
	reg.Register("anki.deck", deckStub{items: map[string]string{"c1": "card one", "c2": "card two"}})
	attachCardsFrom(d, reg, map[string][]modules.CardRef{
		"b": {{Deck: "anki.deck", Item: "c2"}, {Deck: "anki.deck", Item: "missing"}},
	})
	for i := 0; i < 5; i++ {
		p, _ := reg.GenerateContext("b", generator.GeneratorContext{Difficulty: 0.5})
		if p.Answer != "recall:deck#c2" {
			t.Fatalf("b should serve only its filed card, got %q", p.Answer)
		}
	}
	// a is not in cards.json, so the deck-level attachment still applies.
	p, _ := reg.GenerateContext("a", generator.GeneratorContext{Difficulty: 0.5})
	if p.Answer != "recall:deck#any" {
		t.Fatalf("a should fall back to the whole deck, got %q", p.Answer)
	}
}
