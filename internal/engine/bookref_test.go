package engine

import (
	"strings"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/modules"
	"github.com/chuma-beep/mathua/internal/proof"
	"github.com/chuma-beep/mathua/internal/storage"
)

// gatedDeck is a deck whose items "book" (a book exercise) and "practice"
// (unlocks after it) follow the generator.GatedItems rules, plus "other".
type gatedDeck struct{}

func (gatedDeck) Generate(generator.GeneratorContext) generator.Problem {
	return generator.Problem{Question: "deck", Answer: "math:deck#other"}
}
func (gatedDeck) PlainText() bool { return true }
func (gatedDeck) ProblemFor(id string) (generator.Problem, bool) {
	q := "item " + id + " <!--typed-answer []-->"
	if id == "book" {
		q = generator.BookHeader(generator.BookRef{File: "b.pdf", Module: "m", Page: 9, Label: "Book Ex 1, #1"}, false) + "\n\n" + q
	}
	return generator.Problem{Question: q, Answer: "math:deck#" + id}, true
}
func (gatedDeck) correct(ctx generator.GeneratorContext, id string) (bool, bool) {
	if ctx.History == nil {
		return false, false
	}
	return ctx.History.Outcome("math:deck#"+id, "")
}
func (d gatedDeck) ItemServable(id string, ctx generator.GeneratorContext) bool {
	switch id {
	case "book":
		return !ctx.NoBookRefs
	case "practice":
		_, ok := d.correct(ctx, "book")
		return ok
	}
	return true
}
func (d gatedDeck) ItemBookPending(id string, ctx generator.GeneratorContext) (bool, bool) {
	if id != "book" || ctx.NoBookRefs {
		return false, false
	}
	tried, ok := d.correct(ctx, "book")
	return !ok, tried
}

type mapHistory map[string]bool // answer key -> correct (present = tried)

func (h mapHistory) Outcome(answer, _ string) (bool, bool) {
	c, tried := h[answer]
	return tried, c
}

func gatedRegistry(t *testing.T) *generator.Registry {
	t.Helper()
	th := concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 60}
	d, _ := concepts.Build([]concepts.Concept{{ID: "a", Label: "A", Domain: "d", GradingType: "numeric", MasteryThreshold: th}})
	reg := generator.NewRegistry()
	reg.Register("a", &testGen{answer: "42"})
	reg.Register("deck", gatedDeck{})
	attachCardsFrom(d, reg, map[string][]modules.CardRef{
		"a": {{Deck: "deck", Item: "book"}, {Deck: "deck", Item: "practice"}, {Deck: "deck", Item: "other"}},
	})
	return reg
}

func countAnswers(t *testing.T, reg *generator.Registry, ctx generator.GeneratorContext, n int) map[string]int {
	t.Helper()
	got := map[string]int{}
	for i := 0; i < n; i++ {
		p, err := reg.GenerateContext("a", ctx)
		if err != nil {
			t.Fatal(err)
		}
		got[p.Answer]++
		if strings.HasPrefix(p.Answer, "math:") && !strings.HasPrefix(p.Question, CardMarker+"\n") {
			t.Fatalf("a filed item is served without the card marker: %q", p.Question)
		}
	}
	return got
}

// A book exercise filed under a generated topic is served first, and its
// practice version only once the learner has answered the exercise correctly.
func TestFiledBookExerciseFirstThenPractice(t *testing.T) {
	t.Setenv("MATHUA_CARD_SHARE", "0.5")
	reg := gatedRegistry(t)

	fresh := countAnswers(t, reg, generator.GeneratorContext{Difficulty: 0.5}, 40)
	if fresh["math:deck#book"] != 40 {
		t.Fatalf("a learner who has not tried the book exercise should get it first: %v", fresh)
	}
	missed := countAnswers(t, reg, generator.GeneratorContext{Difficulty: 0.5, History: mapHistory{"math:deck#book": false}}, 300)
	if missed["math:deck#practice"] > 0 || missed["math:deck#book"] == 0 || missed["42"] == 0 {
		t.Fatalf("after a miss: the exercise comes back, practice stays hidden: %v", missed)
	}
	solved := countAnswers(t, reg, generator.GeneratorContext{Difficulty: 0.5, History: mapHistory{"math:deck#book": true}}, 300)
	if solved["math:deck#practice"] == 0 || solved["42"] == 0 {
		t.Fatalf("after a correct answer the practice version mixes in: %v", solved)
	}
	// Placement never gets the book exercise; a review test may.
	place := countAnswers(t, reg, generator.GeneratorContext{Difficulty: 0.5, NoCards: true, NoProofs: true, ObjectiveOnly: true, NoBookRefs: true}, 100)
	if place["math:deck#book"] > 0 || place["42"] != 100 {
		t.Fatalf("placement: %v", place)
	}
	review := countAnswers(t, reg, generator.GeneratorContext{Difficulty: 0.5, NoCards: true, NoProofs: true, ObjectiveOnly: true}, 200)
	if review["math:deck#book"] == 0 || review["math:deck#practice"] > 0 || review["math:deck#other"] > 0 {
		t.Fatalf("review test: only the book exercise of the filed cards: %v", review)
	}
}

// proofGateStub is a proof source with a book proof and a practice proof
// that unlocks after it.
type proofGateStub struct{ proofSrcStub }

func (proofGateStub) ByConcept() map[string][]string {
	return map[string][]string{"a": {"pbook", "ppractice"}}
}
func (proofGateStub) Problem(id string) *proof.Problem {
	switch id {
	case "pbook":
		return &proof.Problem{ID: id, Concept: "a", BookRef: &generator.BookRef{File: "b.pdf", Module: "m", Page: 3, Label: "Book Ex 2, #4"}}
	case "ppractice":
		return &proof.Problem{ID: id, Concept: "a", UnlocksAfter: "pbook"}
	}
	return nil
}

func TestBookProofFirstThenPractice(t *testing.T) {
	reg := generator.NewRegistry()
	reg.Register("a", &testGen{answer: "42"})
	attachProofsFrom(proofDAG(t), reg, proofGateStub{proofSrcStub{serving: true, share: 0.5}})
	count := func(ctx generator.GeneratorContext, n int) map[string]int {
		got := map[string]int{}
		for i := 0; i < n; i++ {
			p, _ := reg.GenerateContext("a", ctx)
			got[p.Answer]++
		}
		return got
	}
	if got := count(generator.GeneratorContext{Difficulty: 0.5}, 30); got["proof:pbook"] != 30 {
		t.Fatalf("the book proof should come first: %v", got)
	}
	if got := count(generator.GeneratorContext{Difficulty: 0.5, History: mapHistory{"proof:pbook": false}}, 300); got["proof:ppractice"] > 0 {
		t.Fatalf("practice proof served before the book proof was proved: %v", got)
	}
	if got := count(generator.GeneratorContext{Difficulty: 0.5, History: mapHistory{"proof:pbook": true}}, 300); got["proof:ppractice"] == 0 || got["42"] == 0 {
		t.Fatalf("after the book proof, practice proofs mix in: %v", got)
	}
}

// The engine's History reads the attempts table: a bank item counts as
// answered correctly once any graded answer to it was right, on that topic.
func TestLearnerHistoryFromAttempts(t *testing.T) {
	store, err := storage.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	st, err := store.CreateStudent("learner")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureSession("s", st.ID); err != nil {
		t.Fatal(err)
	}
	rec := func(concept, expected, question string, correct bool) {
		t.Helper()
		if _, err := store.RecordAttempt(storage.AttemptEntry{SessionID: "s", StudentID: st.ID, ConceptID: concept, Answer: "x",
			Expected: expected, Correct: correct, Timestamp: time.Now(), Question: question, Source: "practice"}); err != nil {
			t.Fatal(err)
		}
	}
	rec("a", "math:deck#book", "<!--card-->\nq", false)
	rec("a", "math:deck#book", "<!--card-->\nq", true)
	rec("a", "math:deck#other", "q2", false)
	rec("a", "mc:yes", "<!--card-->\nIs it? A) yes B) no", true)
	rec("a", "42", "generated", true)
	rec("b", "math:deck#practice", "q3", true)
	h := newItemHistory(store, st.ID, "a")
	for _, c := range []struct {
		answer, question string
		tried, correct   bool
	}{
		{"math:deck#book", "", true, true},
		{"math:deck#other", "", true, false},
		{"math:deck#practice", "", false, false}, // another topic
		{"mc:yes", "Is it? A) yes B) no", true, true},
		{"mc:yes", "Something else", false, false},
		{"42", "", false, false}, // generated questions are not items
	} {
		tried, correct := h.Outcome(c.answer, c.question)
		if tried != c.tried || correct != c.correct {
			t.Errorf("%s %q: got (%v, %v), want (%v, %v)", c.answer, c.question, tried, correct, c.tried, c.correct)
		}
	}
	if newItemHistory(store, "", "a") != nil {
		t.Fatal("no learner, no history")
	}
}
