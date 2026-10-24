package engine

import (
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
)

// Fork change: your flashcards in the practice of the topics they are about.
//
// Imported Anki decks (the anki module) are question-bank concepts, "Practice ·
// …", each a successor of the generated topics it covers (anki.limits_continuity
// needs calc.limit.continuity). On their own they only come up once the
// recommender reaches them, and a study plan never does: they are nobody's
// prerequisite. So each generated topic a deck covers also serves that deck's
// cards in its own practice, lessons, reviews and mastery tests: about half its
// questions (MATHUA_CARD_SHARE, 0 to 1) are your cards, the rest generated.
//
// A card keeps its own form (a recall flashcard, a typed math answer, a
// multiple choice) and grades as the card it is (grader.IsBankAnswer); the
// question carries CardMarker so every answer host shows the card's input.

// CardMarker opens a flashcard served on a generated topic.
const CardMarker = "<!--card-->"

// cardDeckPrefix names the decks that are practice for their prerequisites.
const cardDeckPrefix = "anki."

func cardShare() float64 {
	if v, err := strconv.ParseFloat(os.Getenv("MATHUA_CARD_SHARE"), 64); err == nil && v >= 0 && v <= 1 {
		return v
	}
	return 0.5
}

// attachCards wraps each generated topic that one or more decks cover.
func attachCards(dag *concepts.DAG, reg *generator.Registry) {
	if dag == nil || reg == nil {
		return
	}
	share := cardShare()
	if share == 0 {
		return
	}
	decks := map[string][]string{} // topic -> decks covering it
	for _, c := range dag.Order() {
		if !strings.HasPrefix(c.ID, cardDeckPrefix) || !reg.Has(c.ID) {
			continue
		}
		for _, p := range c.Prerequisites {
			pc := dag.Concept(p)
			if pc == nil || strings.HasPrefix(p, cardDeckPrefix) || pc.GradingType == "bank" || pc.GradingType == "python" {
				continue
			}
			decks[p] = append(decks[p], c.ID)
		}
	}
	for topic, ids := range decks {
		var cards []generator.Generator
		for _, id := range ids {
			if g, err := reg.Get(id); err == nil {
				cards = append(cards, g)
			}
		}
		if len(cards) == 0 {
			continue
		}
		reg.Wrap(topic, func(base generator.Generator) generator.Generator {
			if _, done := base.(cardMixer); done {
				return base // a registry shared by several engines is wrapped once
			}
			m := &mixGen{base: base, cards: cards, share: share}
			if _, ok := base.(generator.GradedGenerator); ok {
				return &mixGradedGen{m}
			}
			return m
		})
	}
}

type cardMixer interface{ mixesCards() }

// mixGen serves a topic's generated questions and, share of the time, a card.
type mixGen struct {
	base  generator.Generator
	cards []generator.Generator
	share float64
}

func (m *mixGen) mixesCards() {}

func (m *mixGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	if !ctx.NoCards && rand.Float64() < m.share {
		p := m.cards[rand.Intn(len(m.cards))].Generate(ctx)
		if p.Question != "" && grader.IsBankAnswer(p.Answer) {
			p.Question = CardMarker + "\n" + p.Question
			p.Raw = true
			return p
		}
	}
	return m.base.Generate(ctx)
}

// The topic's own declarations still describe its generated questions.
func (m *mixGen) AnswerFormat() generator.AnswerFormat {
	if af, ok := m.base.(interface{ AnswerFormat() generator.AnswerFormat }); ok {
		return af.AnswerFormat()
	}
	return generator.AnswerFormat{}
}

func (m *mixGen) PlainText() bool {
	pt, ok := m.base.(generator.PlainTextGenerator)
	return ok && pt.PlainText()
}

// mixGradedGen keeps a custom grader for the topic's generated questions.
type mixGradedGen struct{ *mixGen }

func (m *mixGradedGen) Grade(expected, answer string) grader.Result {
	return m.base.(generator.GradedGenerator).Grade(expected, answer)
}
