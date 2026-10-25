package engine

import (
	"math/rand"

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
	if !ctx.NoCards && !ctx.NoProofs && rand.Float64() < m.src.Share() && m.src.Serving() {
		if p, ok := m.src.ProblemFor(m.ids[rand.Intn(len(m.ids))]); ok && grader.IsProofAnswer(p.Answer) {
			p.Question = CardMarker + "\n" + p.Question
			p.Raw = true
			return p
		}
	}
	return m.base.Generate(ctx)
}

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

// proofsCountForMastery is a variable so tests can switch it.
var proofsCountForMastery = proof.CountsForMastery
