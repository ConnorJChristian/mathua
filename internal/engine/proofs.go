package engine

import (
	"math/rand"
	"strings"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
	"github.com/chuma-beep/mathua/internal/proof"
)

// Proof problems (internal/proof) in the practice of the topics they belong
// to. A module's proofs/ folder files each problem under a topic (its
// "concept"), usually a generated real-analysis topic. That topic then serves
// a proof a share of the time (MATHUA_PROOF_SHARE), the way it serves your
// flashcards (cards.go): marked as a card, so every answer host shows the
// question's own input, here a proof box. Proofs are served only while the
// grader can grade them (an API key and budget left), never in placement
// (NoCards) and never in a quiz (NoProofs).

// proofSource is what the engine needs from the proof grader.
type proofSource interface {
	Serving() bool
	ByConcept() map[string][]string
	ProblemFor(id string) (generator.Problem, bool)
	Share() float64
}

type proofMixer interface{ mixesProofs() }

func attachProofs(dag *concepts.DAG, reg *generator.Registry) {
	g, err := proof.Default()
	if err != nil || g == nil || g.Bank.Len() == 0 {
		return
	}
	attachProofsFrom(dag, reg, g)
}

func attachProofsFrom(dag *concepts.DAG, reg *generator.Registry, src proofSource) {
	if dag == nil || reg == nil || src == nil || src.Share() == 0 {
		return
	}
	for topic, ids := range src.ByConcept() {
		if dag.Concept(topic) == nil || !reg.Has(topic) {
			continue
		}
		ids := ids
		reg.Wrap(topic, func(base generator.Generator) generator.Generator {
			if _, done := base.(proofMixer); done {
				return base // a registry shared by several engines is wrapped once
			}
			m := &proofMixGen{base: base, src: src, ids: ids}
			if _, ok := base.(generator.GradedGenerator); ok {
				return &proofMixGradedGen{m}
			}
			return m
		})
	}
}

// proofMixGen serves a topic's own questions and, Share of the time, a proof.
type proofMixGen struct {
	base generator.Generator
	src  proofSource
	ids  []string
}

func (m *proofMixGen) mixesProofs() {}

// mixesCards keeps cards.go from wrapping this topic a second time when
// another engine is built on the same registry: its cards, if any, are
// already in the base underneath.
func (m *proofMixGen) mixesCards() {}

func (m *proofMixGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	if !ctx.NoCards && !ctx.NoProofs && !ctx.ObjectiveOnly {
		// Fork change: a book exercise to prove comes first until the
		// learner proves it (generator.PickBook), while proofs are graded.
		if !ctx.NoBookRefs {
			if ids, tried := m.pendingBooks(ctx); len(ids) > 0 && m.src.Serving() {
				if k := generator.PickBook(tried); k >= 0 {
					if p, ok := m.serve(ids[k]); ok {
						return p
					}
				}
			}
			if generator.HasBookPending(m.base, ctx) {
				return m.base.Generate(ctx)
			}
		}
		if rand.Float64() < m.src.Share() && m.src.Serving() {
			if ids := m.servable(ctx); len(ids) > 0 {
				if p, ok := m.serve(ids[rand.Intn(len(ids))]); ok {
					return p
				}
			}
		}
	}
	return m.base.Generate(ctx)
}

func (m *proofMixGen) serve(id string) (generator.Problem, bool) {
	p, ok := m.src.ProblemFor(id)
	if !ok || !grader.IsProofAnswer(p.Answer) {
		return generator.Problem{}, false
	}
	p.Question = CardMarker + "\n" + p.Question
	p.Raw = true
	return p, true
}

// proofGate is a proof source that knows each problem's book reference and
// unlocks_after (proof.Grader). Without it no problem is gated.
type proofGate interface {
	Problem(id string) *proof.Problem
}

func (m *proofMixGen) gate(id string) (book bool, after string) {
	if g, ok := m.src.(proofGate); ok {
		if p := g.Problem(id); p != nil {
			return p.BookRef != nil, p.UnlocksAfter
		}
	}
	return false, ""
}

func proofOutcome(id string, ctx generator.GeneratorContext) (attempted, correct bool) {
	if ctx.History == nil {
		return false, false
	}
	return ctx.History.Outcome(grader.ProofPrefix+id, "")
}

// servable lists the topic's proofs ctx may serve: a book exercise only
// outside placement, and a problem that unlocks after another only once the
// learner has proved that one.
func (m *proofMixGen) servable(ctx generator.GeneratorContext) []string {
	var out []string
	for _, id := range m.ids {
		book, after := m.gate(id)
		if book && ctx.NoBookRefs {
			continue
		}
		if after != "" {
			if _, correct := proofOutcome(after, ctx); !correct {
				continue
			}
		}
		out = append(out, id)
	}
	return out
}

// pendingBooks lists the servable book proofs not yet proved, in bank order.
func (m *proofMixGen) pendingBooks(ctx generator.GeneratorContext) ([]string, []bool) {
	var ids []string
	var tried []bool
	for _, id := range m.servable(ctx) {
		if book, _ := m.gate(id); !book {
			continue
		}
		if attempted, correct := proofOutcome(id, ctx); !correct {
			ids = append(ids, id)
			tried = append(tried, attempted)
		}
	}
	return ids, tried
}

// BookPending implements generator.BookPender.
func (m *proofMixGen) BookPending(ctx generator.GeneratorContext) bool {
	if ctx.NoBookRefs || ctx.ObjectiveOnly || ctx.NoCards || ctx.NoProofs {
		return generator.HasBookPending(m.base, ctx)
	}
	if ids, _ := m.pendingBooks(ctx); len(ids) > 0 && m.src.Serving() {
		return true
	}
	return generator.HasBookPending(m.base, ctx)
}

// ItemServable and ItemBookPending forward a wrapped bank topic's gating.
func (m *proofMixGen) ItemServable(item string, ctx generator.GeneratorContext) bool {
	if gi, ok := m.base.(generator.GatedItems); ok {
		return gi.ItemServable(item, ctx)
	}
	return true
}

func (m *proofMixGen) ItemBookPending(item string, ctx generator.GeneratorContext) (bool, bool) {
	if gi, ok := m.base.(generator.GatedItems); ok {
		return gi.ItemBookPending(item, ctx)
	}
	return false, false
}

// ObjectiveItems forwards a bank base's count (see mixGen.ObjectiveItems).
func (m *proofMixGen) ObjectiveItems() int { return objectiveItemsOf(m.base) }

func (m *proofMixGen) AnswerFormat() generator.AnswerFormat {
	if af, ok := m.base.(interface{ AnswerFormat() generator.AnswerFormat }); ok {
		return af.AnswerFormat()
	}
	return generator.AnswerFormat{}
}

func (m *proofMixGen) PlainText() bool {
	pt, ok := m.base.(generator.PlainTextGenerator)
	return ok && pt.PlainText()
}

// ProblemFor keeps a wrapped bank topic able to serve its items by id.
func (m *proofMixGen) ProblemFor(id string) (generator.Problem, bool) {
	if is, ok := m.base.(itemServer); ok {
		return is.ProblemFor(id)
	}
	return generator.Problem{}, false
}

type proofMixGradedGen struct{ *proofMixGen }

func (m *proofMixGradedGen) Grade(expected, answer string) grader.Result {
	return m.base.(generator.GradedGenerator).Grade(expected, answer)
}

// practiceOnly reports an answer that is practice rather than mastery
// evidence: a recall card the learner graded themselves, and a proof while
// MATHUA_PROOF_COUNTS_FOR_MASTERY is off (see proof.Config.CountsForMastery
// for why that is the default).
func practiceOnly(expected string) bool {
	if isSelfGraded(expected) {
		return true
	}
	return grader.IsProofAnswer(expected) && !proofsCountForMastery()
}

// plainRecall turns a written-answer flashcard (internal/pytrack written.go:
// the front, the proof marker, the recall marker, the back) back into a plain
// flashcard. Any other problem is returned unchanged.
func plainRecall(p generator.Problem) generator.Problem {
	if strings.HasPrefix(p.Answer, "recall:") && strings.Contains(p.Question, proof.Marker) {
		p.Question = strings.Replace(p.Question, proof.Marker+"\n\n", "", 1)
		p.Question = strings.Replace(p.Question, proof.Marker, "", 1)
	}
	return p
}

// proofsCountForMastery is a variable so tests can switch it.
var proofsCountForMastery = proof.CountsForMastery

// writtenTaskCatalog is what the course page needs to know about written
// tasks (proofs and writing problems, internal/proof): whether they can be
// graded here, and which topics they belong to.
type writtenTaskCatalog interface {
	Enabled() bool
	ByConcept() map[string][]string
}

// writtenTasks is the catalog the engine reports from: the process's proof
// grader unless a test sets another.
func (e *Engine) writtenTasks() writtenTaskCatalog {
	if e.writtenCatalog != nil {
		return e.writtenCatalog
	}
	g, err := proof.Default()
	if err != nil || g == nil {
		return nil
	}
	return g
}

// UngradedWrittenTasks counts the written tasks (proofs and writing problems)
// filed under the given topics that are left out of practice because nothing
// can grade them: no API key is set. Zero when they can be graded. Without
// this the course page never said such tasks existed.
func (e *Engine) UngradedWrittenTasks(topics []string) int {
	cat := e.writtenTasks()
	if cat == nil || cat.Enabled() {
		return 0
	}
	by := cat.ByConcept()
	n := 0
	seen := map[string]bool{}
	for _, t := range topics {
		if seen[t] {
			continue
		}
		seen[t] = true
		n += len(by[t])
	}
	return n
}

// SetWrittenTaskCatalog replaces the written-task catalog (tests).
func (e *Engine) SetWrittenTaskCatalog(c interface {
	Enabled() bool
	ByConcept() map[string][]string
}) {
	e.writtenCatalog = c
}
