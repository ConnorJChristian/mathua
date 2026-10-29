package engine

import (
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
	"github.com/chuma-beep/mathua/internal/modules"
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
// A topic that is itself a question bank (an authored ma-program bank, a
// Python Data Science Handbook section) takes cards the same way; only a
// deck (anki.*) never does.

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

// itemServer is a bank generator that can serve one of its items by id
// (pytrack's conceptGen).
type itemServer interface {
	ProblemFor(itemID string) (generator.Problem, bool)
}

// cardPick is one card filed under a topic: the deck's generator and the item.
type cardPick struct {
	gen  itemServer
	item string
}

// attachCards wraps each generated topic that has cards: the ones filed
// under it in a module's cards.json, or, when no module files any cards,
// every card of each deck that lists it as a prerequisite.
func attachCards(dag *concepts.DAG, reg *generator.Registry) {
	attachCardsFrom(dag, reg, nil)
}

func attachCardsFrom(dag *concepts.DAG, reg *generator.Registry, filed map[string][]modules.CardRef) {
	if dag == nil || reg == nil {
		return
	}
	share := cardShare()
	if share == 0 {
		return
	}
	if filed == nil {
		if mods, err := modules.Default(); err == nil && mods != nil {
			filed = mods.Cards()
		}
	}
	wrapped := map[string]bool{}
	for topic, refs := range filed {
		tc := dag.Concept(topic)
		if tc == nil || strings.HasPrefix(topic, cardDeckPrefix) {
			continue
		}
		var picks []cardPick
		for _, ref := range refs {
			g, err := reg.Get(ref.Deck)
			if err != nil {
				continue
			}
			if is, ok := g.(itemServer); ok {
				if _, found := is.ProblemFor(ref.Item); found {
					picks = append(picks, cardPick{is, ref.Item})
				}
			}
		}
		if len(picks) == 0 {
			continue
		}
		wrapped[topic] = true
		reg.Wrap(topic, func(base generator.Generator) generator.Generator {
			if _, done := base.(cardMixer); done {
				return base
			}
			m := &mixGen{base: base, picks: picks, share: share}
			if _, ok := base.(generator.GradedGenerator); ok {
				return &mixGradedGen{m}
			}
			return m
		})
	}
	// The deck fallback is for installs with no filing at all. Where a
	// cards.json exists it is the whole truth: a topic it does not list gets
	// no cards. Falling back per topic served every card of every deck that
	// named the topic as a prerequisite ("Reduce 14/35" and "Flows/Stocks"
	// under systems by substitution; 57 topics in all; Q6).
	if len(filed) > 0 {
		return
	}
	decks := map[string][]string{} // topic -> decks covering it
	for _, c := range dag.Order() {
		if !strings.HasPrefix(c.ID, cardDeckPrefix) || !reg.Has(c.ID) {
			continue
		}
		for _, p := range c.Prerequisites {
			pc := dag.Concept(p)
			if pc == nil || wrapped[p] || strings.HasPrefix(p, cardDeckPrefix) {
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
	cards []generator.Generator // whole decks (no cards.json for this topic)
	picks []cardPick            // single cards filed under this topic
	share float64
}

func (m *mixGen) mixesCards() {}

func (m *mixGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	// Fork change: a book exercise filed under this topic comes first until
	// the learner answers it correctly (generator.PickBook), then the
	// topic's own book exercises, then the usual mix.
	if !ctx.NoBookRefs && !ctx.ObjectiveOnly {
		if p, ok := m.pendingBook(ctx); ok {
			return p
		}
		if generator.HasBookPending(m.base, ctx) {
			return m.base.Generate(ctx)
		}
	}
	if !ctx.NoCards && !ctx.ObjectiveOnly && rand.Float64() < m.share {
		var p generator.Problem
		if len(m.picks) > 0 {
			if ok := m.servable(ctx, false); len(ok) > 0 {
				pk := ok[rand.Intn(len(ok))]
				p, _ = pk.gen.ProblemFor(pk.item)
			}
		} else {
			p = m.cards[rand.Intn(len(m.cards))].Generate(ctx)
		}
		if p.Question != "" && grader.IsBankAnswer(p.Answer) {
			return m.asCard(p, ctx)
		}
	}
	// A course review test may ask a book exercise filed here (placement
	// sets NoBookRefs); other filed cards stay out of tests.
	if ctx.ObjectiveOnly && !ctx.NoBookRefs && rand.Float64() < m.share {
		if ok := m.servable(ctx, true); len(ok) > 0 {
			pk := ok[rand.Intn(len(ok))]
			if p, found := pk.gen.ProblemFor(pk.item); found && p.Question != "" && grader.IsBankAnswer(p.Answer) {
				return m.asCard(p, ctx)
			}
		}
	}
	return m.base.Generate(ctx)
}

// asCard marks a filed card for every answer host.
func (m *mixGen) asCard(p generator.Problem, ctx generator.GeneratorContext) generator.Problem {
	if ctx.NoProofs {
		// A quiz asks for no written answer: a written-answer
		// flashcard comes as the plain flashcard it also is.
		p = plainRecall(p)
	}
	p.Question = CardMarker + "\n" + p.Question
	p.Raw = true
	return p
}

// servable lists the filed cards ctx may serve (generator.GatedItems: an
// item that unlocks after another stays hidden until that one is answered
// correctly); with books, only book exercises.
func (m *mixGen) servable(ctx generator.GeneratorContext, books bool) []cardPick {
	var out []cardPick
	for _, pk := range m.picks {
		gi, gated := pk.gen.(generator.GatedItems)
		if gated && !gi.ItemServable(pk.item, ctx) {
			continue
		}
		if books && (!gated || !bookItem(gi, pk.item, ctx)) {
			continue
		}
		out = append(out, pk)
	}
	return out
}

// bookItem reports whether an item is a book exercise: one that is pending
// for a learner who has done nothing.
func bookItem(gi generator.GatedItems, item string, ctx generator.GeneratorContext) bool {
	none := ctx
	none.History = nil
	none.NoBookRefs = false
	pending, _ := gi.ItemBookPending(item, none)
	return pending
}

// pendingBook serves the first filed book exercise the learner has not
// answered correctly (generator.PickBook).
func (m *mixGen) pendingBook(ctx generator.GeneratorContext) (generator.Problem, bool) {
	var picks []cardPick
	var tried []bool
	for _, pk := range m.picks {
		gi, ok := pk.gen.(generator.GatedItems)
		if !ok || !gi.ItemServable(pk.item, ctx) {
			continue
		}
		if pending, attempted := gi.ItemBookPending(pk.item, ctx); pending {
			picks = append(picks, pk)
			tried = append(tried, attempted)
		}
	}
	k := generator.PickBook(tried)
	if k < 0 {
		return generator.Problem{}, false
	}
	p, found := picks[k].gen.ProblemFor(picks[k].item)
	if !found || p.Question == "" {
		return generator.Problem{}, false
	}
	return m.asCard(p, ctx), true
}

// BookPending implements generator.BookPender.
func (m *mixGen) BookPending(ctx generator.GeneratorContext) bool {
	if ctx.NoBookRefs || ctx.ObjectiveOnly {
		return false
	}
	for _, pk := range m.picks {
		if gi, ok := pk.gen.(generator.GatedItems); ok && gi.ItemServable(pk.item, ctx) {
			if pending, _ := gi.ItemBookPending(pk.item, ctx); pending {
				return true
			}
		}
	}
	return generator.HasBookPending(m.base, ctx)
}

// ItemServable and ItemBookPending forward a wrapped bank topic's gating, so
// it can still be filed under another topic.
func (m *mixGen) ItemServable(item string, ctx generator.GeneratorContext) bool {
	if gi, ok := m.base.(generator.GatedItems); ok {
		return gi.ItemServable(item, ctx)
	}
	return true
}

func (m *mixGen) ItemBookPending(item string, ctx generator.GeneratorContext) (bool, bool) {
	if gi, ok := m.base.(generator.GatedItems); ok {
		return gi.ItemBookPending(item, ctx)
	}
	return false, false
}

// ProblemFor keeps a wrapped bank topic able to serve its items by id.
func (m *mixGen) ProblemFor(id string) (generator.Problem, bool) {
	if is, ok := m.base.(itemServer); ok {
		return is.ProblemFor(id)
	}
	return generator.Problem{}, false
}

// ObjectiveItems forwards a bank base's count, so a wrapped bank topic still
// says how many items a test may serve. A generated base has none to count.
func (m *mixGen) ObjectiveItems() int { return objectiveItemsOf(m.base) }

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
