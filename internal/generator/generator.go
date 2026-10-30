package generator

import "github.com/chuma-beep/mathua/internal/grader"

type GeneratorContext struct {
	Difficulty float64
	Seed       int64
	// NoCards asks for the topic's generated questions only, never one of
	// your flashcards (engine/cards.go): placement measures with questions
	// that grade themselves, not self-graded cards.
	NoCards bool
	// NoProofs asks for no proof problem (internal/proof). Quizzes set it:
	// a quiz is a timed assessment whose answers are mastery evidence, and a
	// written proof is neither quick nor, by default, evidence.
	NoProofs bool
	// ObjectiveOnly asks for a question the server grades by itself and that
	// is quick to answer: a generated question, or a bank item that is
	// multiple choice, "what does this print" or typed math. Never a recall
	// card (self-graded), a code task (slow, sandboxed) or a proof. Course
	// placement and review tests set it. A bank topic with no such item
	// returns an empty Problem; callers check ObjectiveItems first.
	ObjectiveOnly bool
	// NoBookRefs asks for no question whose prompt is a page of the
	// learner's book (BookRef, fork change). Placement sets it: it must stay
	// objective and quick, and a book page needs the PDF and reading time.
	// Course review tests and practice leave it off.
	NoBookRefs bool
	// History is what this learner has done with the topic's questions, for
	// items that unlock after another is answered correctly and for serving
	// book exercises first. Nil means no learner is known: a gated item is
	// then never served.
	History History
	// Review asks for a review of a topic already learned (fork change): it
	// samples across the scaffold steps by difficulty instead of climbing
	// them (scaffold.go).
	Review bool
	// Served is what this request has already served, by question text (a
	// lesson batch): a scaffold serves none of it again (scaffold.go). The
	// registry's batch sets it; nil is nothing.
	Served map[string]bool
}

// ObjectiveItemCounter is implemented by question-bank generators: how many
// of the topic's items an ObjectiveOnly request can serve. A generator that
// does not implement it is a generated topic, which is objective throughout.
type ObjectiveItemCounter interface {
	ObjectiveItems() int
}

type Problem struct {
	Question    string
	Answer      string
	Explanation string
	// Facts are the named values this instance was built from — the operands
	// and the derived quantities, already computed and already reduced. A
	// corpus solution schema interpolates them into authored prose; it never
	// recomputes them, so an explanation cannot contradict the answer.
	//
	// Optional: a schema with no facts behind it falls back to Explanation.
	// Keys are snake_case and stable; renaming one is a content change and the
	// contract test will say so.
	Facts map[string]string

	// Difficulty the question was generated at, 0.3-1.0. Nil means unknown — a problem
	// that did not come from a difficulty-parameterised generator, such as a curated row
	// read straight out of the questions table.
	//
	// This field exists because difficulty used to go *into* GeneratorContext and stop
	// there. Every surface that generated a question and later graded it had to thread the
	// value back out by hand, and two of the three did not: the study and quiz paths
	// recorded `attempts.difficulty = NULL`. Mastery evidence therefore depended on which
	// screen the learner happened to use, which is not a property a learner should be able
	// to change by tapping a different button. Carrying it on the Problem makes generation
	// and grading agree by construction — the registry fills this in, so no caller can
	// generate a question and forget what it was generated at.
	Difficulty *float64

	// Raw marks a problem served exactly as generated, skipping the LaTeX
	// canonicaliser and the solution schemas: a question-bank item (one of
	// your flashcards) served on a generated concept. See engine/cards.go.
	Raw bool
}

type Generator interface {
	Generate(ctx GeneratorContext) Problem
}

// GradedGenerator is a Generator that can grade its own answers.
// Useful for non-standard formats like "5 R 3" or "3 sqrt(2)".
type GradedGenerator interface {
	Generator
	Grade(expected, userAnswer string) grader.Result
}
