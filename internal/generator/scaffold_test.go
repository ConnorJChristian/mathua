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

// A topic with nine book exercises at one step moves on after three tried
// and a right answer (BookCap 3); the six left are no longer served first.
// Without a cap every one of them comes before the next step.
func TestScaffold_BookStepCap(t *testing.T) {
	build := func(tried int, cap int) *Scaffold {
		var items []ScaffoldItem
		for i := 1; i <= 9; i++ {
			items = append(items, item(fmt.Sprintf("ex %d", i), 2, true, i <= tried, i == 1 && tried > 0))
		}
		items = append(items, item("a1", 1, false, true, true), item("w3", 3, false, false, false))
		return &Scaffold{Items: items, BookCap: cap}
	}
	// Two tried: the third exercise, in book order.
	if p, _ := build(2, 3).Pick(nil); p.Question != "ex 3" {
		t.Fatalf("two of three tried: the next book exercise, got %q", p.Question)
	}
	// Three tried, one right: step 2 is done, the learner is at step 3.
	sc := build(3, 3)
	if !sc.Done(2) || sc.Frontier() != 3 {
		t.Fatalf("three tried and one right: step 2 done, frontier %d", sc.Frontier())
	}
	for i := 0; i < 20; i++ {
		if p, _ := sc.Pick(nil); p.Question != "w3" {
			t.Fatalf("capped: the step-3 question, got %q", p.Question)
		}
	}
	// No cap: all nine before step 3.
	if p, _ := build(3, 0).Pick(nil); p.Question != "ex 4" {
		t.Errorf("no cap: the fourth exercise, got %q", p.Question)
	}
	// Three tried, none right: the step stays open, and the untried
	// exercises join the step's untried pool instead of coming first.
	var items []ScaffoldItem
	for i := 1; i <= 9; i++ {
		items = append(items, item(fmt.Sprintf("ex %d", i), 2, true, i <= 3, false))
	}
	items = append(items, item("a1", 1, false, true, true), item("case", 2, false, false, false), item("w3", 3, false, false, false))
	open := &Scaffold{Items: items, BookCap: 3}
	if open.Done(2) || open.Frontier() != 2 {
		t.Fatalf("no right answer at step 2: still open")
	}
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		p, _ := open.Pick(nil)
		seen[p.Question] = true
		if p.Question == "w3" {
			t.Fatal("step 3 before a right answer at step 2")
		}
	}
	if !seen["case"] || !seen["ex 9"] {
		t.Errorf("capped step: the bank's case and the untried exercises share the pool, saw %v", seen)
	}
	// A step with no more exercises than the cap still needs them all.
	few := &Scaffold{Items: []ScaffoldItem{item("a1", 1, false, true, true), item("b1", 1, true, true, false), item("b2", 1, true, false, false), item("a2", 2, false, false, false)}, BookCap: 3}
	if p, _ := few.Pick(nil); p.Question != "b2" {
		t.Errorf("two exercises under a cap of three: both tried first, got %q", p.Question)
	}
}

func TestEnvBookStepCap(t *testing.T) {
	t.Setenv(BookStepCapEnv, "")
	if EnvBookStepCap() != DefaultBookStepCap {
		t.Errorf("unset: default %d, got %d", DefaultBookStepCap, EnvBookStepCap())
	}
	t.Setenv(BookStepCapEnv, "5")
	if EnvBookStepCap() != 5 || BookStepCapOf(nil) != 5 {
		t.Error("MATHUA_BOOK_STEP_CAP=5")
	}
	t.Setenv(BookStepCapEnv, "0")
	if EnvBookStepCap() != 0 {
		t.Error("0 switches the cap off")
	}
	// Under a cap, a leftover exercise comes back half the time, the
	// untried one first.
	hits := 0
	for i := 0; i < 400; i++ {
		switch PickBookLater([]bool{true, false}) {
		case 1:
			hits++
		case -1:
		default:
			t.Fatal("the untried exercise, or nothing")
		}
	}
	if hits < 120 || hits > 280 {
		t.Errorf("about half the time, got %d of 400", hits)
	}
}
