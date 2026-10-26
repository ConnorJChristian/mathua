package engine

import (
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/modules"
	"github.com/chuma-beep/mathua/internal/proof"
)

// writtenDeck serves one written-answer flashcard (internal/pytrack
// written.go): front, proof marker, recall marker, back.
type writtenDeck struct{}

const writtenQ = "Prove it.\n\n" + proof.Marker + "\n\n<!--recall-back-->\n\nThe back."

func (writtenDeck) Generate(generator.GeneratorContext) generator.Problem {
	return generator.Problem{Question: writtenQ, Answer: "recall:deck#w"}
}
func (writtenDeck) PlainText() bool { return true }
func (writtenDeck) ProblemFor(id string) (generator.Problem, bool) {
	return generator.Problem{Question: writtenQ, Answer: "recall:deck#" + id}, id == "w"
}

// A written-answer flashcard in a generated topic's practice keeps its answer
// box in study and review, comes as a plain flashcard in a quiz, and is never
// served in placement or a course test.
func TestWrittenCardsInTopicPractice(t *testing.T) {
	t.Setenv("MATHUA_CARD_SHARE", "1")
	th := concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 60}
	d, _ := concepts.Build([]concepts.Concept{
		{ID: "a", Label: "A", Domain: "d", GradingType: "numeric", MasteryThreshold: th},
		{ID: "anki.deck", Label: "Practice · deck", Domain: "d", GradingType: "bank", Prerequisites: []string{"a"}, MasteryThreshold: th},
	})
	reg := generator.NewRegistry()
	reg.Register("a", &testGen{answer: "1"})
	reg.Register("anki.deck", writtenDeck{})
	attachCardsFrom(d, reg, map[string][]modules.CardRef{"a": {{Deck: "anki.deck", Item: "w"}}})

	p, _ := reg.GenerateContext("a", generator.GeneratorContext{Difficulty: 0.5})
	if !strings.Contains(p.Question, proof.Marker) {
		t.Fatalf("study should keep the written-answer box: %q", p.Question)
	}
	p, _ = reg.GenerateContext("a", generator.GeneratorContext{Difficulty: 0.5, NoProofs: true})
	if strings.Contains(p.Question, proof.Marker) || !strings.Contains(p.Question, "<!--recall-back-->") || p.Answer != "recall:deck#w" {
		t.Fatalf("a quiz should get the plain flashcard: %q", p.Question)
	}
	for _, ctx := range []generator.GeneratorContext{
		{Difficulty: 0.5, NoCards: true, NoProofs: true, ObjectiveOnly: true}, // placement (diagnostic)
		{Difficulty: 0.5, NoProofs: true, NoCards: true, ObjectiveOnly: true}, // course review test
		{Difficulty: 0.5, ObjectiveOnly: true},
	} {
		if p, _ := reg.GenerateContext("a", ctx); p.Answer != "1" {
			t.Fatalf("placement and course tests must not serve a flashcard: %+v -> %q", ctx, p.Answer)
		}
	}
}

// A written answer graded by the judge is recorded exactly as a self-graded
// card: practice, never evidence, never a miss toward a halt, even when
// proof grades count for mastery.
func TestWrittenCardGradeIsPractice(t *testing.T) {
	old := proofsCountForMastery
	defer func() { proofsCountForMastery = old }()
	for _, counts := range []bool{false, true} {
		proofsCountForMastery = func() bool { return counts }
		if !practiceOnly("recall:anki.x#w") {
			t.Errorf("counts=%v: a flashcard's grade must be practice only", counts)
		}
	}
}
