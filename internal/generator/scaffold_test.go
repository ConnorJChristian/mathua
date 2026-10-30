package generator

import (
	"fmt"
	"testing"
)

func item(id string, step int, book, attempted, correct bool) ScaffoldItem {
	return ScaffoldItem{Step: step, Book: book, Attempted: attempted, Correct: correct,
		Serve: func() (Problem, bool) { return Problem{Question: id, Answer: id}, true }}
}

// A fresh learner starts at step 1 even when the bank lists a step-3 book
// exercise first; within a step the book exercise comes before the rest.
func TestScaffold_FreshLearnerStartsAtStepOneBookFirst(t *testing.T) {
	sc := Scaffold{Items: []ScaffoldItem{
		item("word problem (book)", 3, true, false, false),
		item("worked case", 2, false, false, false),
		item("apply 1", 1, false, false, false),
		item("apply book", 1, true, false, false),
		item("apply 2", 1, false, false, false),
	}}
	for i := 0; i < 20; i++ {
		p, ok := sc.Pick(nil)
		if !ok || p.Question != "apply book" {
			t.Fatalf("want the step-1 book exercise first, got %q", p.Question)
		}
	}
	// A lesson batch: the book exercise, then the step's other items, then
	// the next step; never step 3 before steps 1 and 2.
	served := map[string]bool{}
	var order []string
	for i := 0; i < 5; i++ {
		p, ok := sc.Pick(served)
		if !ok {
			break
		}
		served[p.Question] = true
		order = append(order, p.Question)
	}
	if fmt.Sprint(order[:1]) != "[apply book]" || order[3] != "worked case" || order[4] != "word problem (book)" {
		t.Errorf("batch order %v", order)
	}
}

// The step climbs as steps are done, and never reaches 3 before a right
// answer at 1, whatever the ladder asks.
func TestScaffold_Climbs(t *testing.T) {
	steps := func(sc Scaffold, _ float64) int {
		p, _ := sc.Pick(nil)
		return map[string]int{"a1": 1, "a2": 2, "a3": 3, "": 0}[p.Question]
	}
	tried := Scaffold{Items: []ScaffoldItem{item("a1", 1, false, true, false), item("a2", 2, false, false, false), item("a3", 3, false, false, false)}}
	if s := steps(tried, 0.9); s != 1 {
		t.Errorf("step 1 missed: still step 1 even at the ladder's top, got %d", s)
	}
	one := Scaffold{Items: []ScaffoldItem{item("a1", 1, false, true, true), item("a2", 2, false, false, false), item("a3", 3, false, false, false)}}
	if s := steps(one, 0.9); s != 2 {
		t.Errorf("step 1 right: step 2, got %d", s)
	}
	if s := steps(one, 0.4); s != 2 {
		t.Errorf("the Learn ladder starting over at 0.4 does not send a learner back to step 1, got %d", s)
	}
	two := Scaffold{Items: []ScaffoldItem{item("a1", 1, false, true, true), item("a2", 2, false, true, true), item("a3", 3, false, false, false)}}
	if s := steps(two, 0.6); s != 3 {
		t.Errorf("steps 1 and 2 right: step 3, got %d", s)
	}
	done := Scaffold{Items: []ScaffoldItem{item("a1", 1, false, true, true), item("a2", 2, false, true, true), item("a3", 3, false, true, true)}}
	if _, ok := done.Pick(nil); ok {
		t.Error("every step done: Pick leaves it to the caller's sampling")
	}
	// An untried book exercise keeps its step open even after a right
	// answer at it.
	book := Scaffold{Items: []ScaffoldItem{item("a1", 1, false, true, true), item("b1", 1, true, false, false), item("a2", 2, false, false, false)}}
	if p, _ := book.Pick(nil); p.Question != "b1" {
		t.Errorf("untried book exercise at step 1 first, got %q", p.Question)
	}
	// Only step 3: step 3 it is.
	only3 := Scaffold{Items: []ScaffoldItem{item("a3", 3, false, false, false)}}
	if s := steps(only3, 0.4); s != 3 {
		t.Errorf("a topic with only step 3 serves it, got %d", s)
	}
}

// A generated topic: generated questions serve a step when the bank has
// nothing there, at that step's difficulty; GeneratedDone moves the step.
func TestScaffold_GeneratedFillsAStep(t *testing.T) {
	var asked []int
	sc := Scaffold{
		Items: []ScaffoldItem{item("book case", 2, true, false, false)},
		Generated: func(step int) Problem {
			asked = append(asked, step)
			return Problem{Question: fmt.Sprintf("gen %d", step)}
		},
		Share: 0.5,
	}
	if p, _ := sc.Pick(nil); p.Question != "gen 1" {
		t.Fatalf("fresh: a generated step-1 question, got %q", p.Question)
	}
	sc.GeneratedDone[1] = true
	if p, _ := sc.Pick(nil); p.Question != "book case" {
		t.Fatalf("step 1 done: the book's worked case, got %q", p.Question)
	}
	if DifficultyOfStep(1) > 0.45 || StepOfDifficulty(DifficultyOfStep(3)) != 3 || StepOfDifficulty(0.4) != 1 || StepOfDifficulty(0.55) != 2 || StepOfDifficulty(0.85) != 3 {
		t.Error("step and difficulty disagree with the Learn ladder (0.4, +0.15)")
	}
}
