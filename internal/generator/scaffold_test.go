package generator

import (
	"fmt"
	"math/rand"
	"strings"
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

func srcItem(id string, step int, src string, book, attempted, correct bool) ScaffoldItem {
	it := item(id, step, book, attempted, correct)
	it.Source = src
	return it
}

// sources serves a lesson batch and names each question's source: its book
// id, or "gen" for a generated question.
func batchSources(t *testing.T, sc *Scaffold, n int) (order []string, probs []Problem) {
	t.Helper()
	served := map[string]bool{}
	for i := 0; i < n; i++ {
		p, ok := sc.Pick(served)
		if !ok {
			break
		}
		served[p.Question] = true
		src := p.SourceBook
		if src == "" {
			src = "gen"
		}
		order = append(order, src)
		probs = append(probs, p)
	}
	return order, probs
}

func genAt(step int) Problem {
	return Problem{Question: fmt.Sprintf("gen step %d #%d", step, rand.Int()), Answer: "x"}
}

// The course's own book, then the parallel books taking turns, then the
// practice set, at every step; the main book's cap and each parallel
// book's cap are honoured; excluded books and linked items of the main
// book are never served.
func TestScaffold_ParallelBooksBetweenMainBookAndPracticeSet(t *testing.T) {
	build := func() *Scaffold {
		var items []ScaffoldItem
		for step := 1; step <= 2; step++ {
			for i := 0; i < 4; i++ {
				items = append(items, srcItem(fmt.Sprintf("hoy ex %d.%d", step, i), step, "hoy", true, false, false))
			}
			for i := 0; i < 5; i++ {
				items = append(items, srcItem(fmt.Sprintf("schaum %d.%d", step, i), step, "schaums-calc-econ", false, false, false))
				items = append(items, srcItem(fmt.Sprintf("leithold %d.%d", step, i), step, "leithold", false, false, false))
			}
			items = append(items, srcItem(fmt.Sprintf("clash %d", step), step, "axler", false, false, false))
			linked := srcItem(fmt.Sprintf("linked hoy %d", step), step, "hoy", false, false, false)
			linked.Linked = true
			items = append(items, linked, item(fmt.Sprintf("unsourced %d", step), step, false, false, false))
		}
		sc := &Scaffold{Items: items, BookCap: 2, Share: 0.5, Generated: genAt}
		sc.ArrangeBooks(BookPolicy{Texts: []string{"hoy"}, ParallelCap: 3, Exclude: map[string]bool{"axler": true}}, "")
		return sc
	}
	sc := build()
	if sc.Main != "hoy" || fmt.Sprint(sc.Books) != "[schaums-calc-econ leithold]" {
		t.Fatalf("main %q books %v", sc.Main, sc.Books)
	}
	// A fresh learner: every Hoy exercise at the step first (the book step
	// cap counts what has been tried, as before), then Schaum's and
	// Leithold taking turns, three of each, then the practice set.
	order, probs := batchSources(t, sc, 11)
	want := "[hoy hoy hoy hoy schaums-calc-econ leithold schaums-calc-econ leithold schaums-calc-econ leithold]"
	if fmt.Sprint(order[:10]) != want {
		t.Fatalf("step 1 order %v, want %s", order, want)
	}
	if probs[10].Tier != TierPractice || probs[10].Step != 1 {
		t.Errorf("after the caps, the practice set at step 1: %v (%s)", order[10], probs[10].Tier)
	}
	if fmt.Sprint(probs[0].StepPath) != "[hoy schaums-calc-econ leithold ]" || probs[0].Step != 1 || probs[0].Tier != TierMain || probs[4].Tier != TierParallel {
		t.Errorf("path %q step %d tiers %s %s", probs[0].StepPath, probs[0].Step, probs[0].Tier, probs[4].Tier)
	}
	// Past the main book's cap (two tried), its untried exercises wait in
	// the practice set and the parallel books come first.
	sc = build()
	for i := 0; i < 2; i++ {
		sc.Items[i].Attempted, sc.Items[i].Correct = true, true
	}
	order, _ = batchSources(t, sc, 6)
	if fmt.Sprint(order) != "[schaums-calc-econ leithold schaums-calc-econ leithold schaums-calc-econ leithold]" {
		t.Errorf("past the book step cap: %v", order)
	}
	// Over many lessons nothing excluded or linked to the main book is
	// ever served.
	for i := 0; i < 50; i++ {
		_, probs := batchSources(t, build(), 20)
		for _, p := range probs {
			if p.Question == "clash 1" || p.Question == "clash 2" || p.Question == "linked hoy 1" || p.Question == "linked hoy 2" {
				t.Fatalf("served %q", p.Question)
			}
		}
	}
	// The step is not done until the main book's cap and each parallel
	// book's cap have been tried, whatever is right.
	mark := func(sc *Scaffold, prefix string, n int, correct bool) {
		for i := range sc.Items {
			if n > 0 && sc.Items[i].Step == 1 && strings.HasPrefix(fmt.Sprint(sc.Items[i].Serve()), "{"+prefix) {
				sc.Items[i].Attempted, sc.Items[i].Correct = true, correct
				n--
			}
		}
	}
	sc = build()
	mark(sc, "hoy ex", 2, true)
	mark(sc, "schaum", 3, false)
	if sc.Done(1) {
		t.Fatal("Leithold's three not tried: step 1 is not done")
	}
	mark(sc, "leithold", 3, false)
	if !sc.Done(1) || sc.Frontier() != 2 {
		t.Fatalf("caps met and Hoy right: step 2, got frontier %d", sc.Frontier())
	}
	// A parallel book's right answer counts like the main book's.
	sc = build()
	mark(sc, "hoy ex", 2, false)
	mark(sc, "schaum", 3, false)
	mark(sc, "leithold", 3, true)
	if !sc.Done(1) {
		t.Fatal("a right Leithold answer finishes the step")
	}
	order, _ = batchSources(t, sc, 5)
	if fmt.Sprint(order) != "[hoy hoy hoy hoy schaums-calc-econ]" {
		t.Errorf("step 2 starts with Hoy again: %v", order)
	}
}

// Without a course text on the topic, the book of its exercises is the
// main book; a book's exercises come in book order; the per-book cap 0
// serves every item of each book before the practice set.
func TestScaffold_ParallelBooksMainFromExercisesAndNoCap(t *testing.T) {
	items := []ScaffoldItem{
		srcItem("s1", 1, "schaums-linear-algebra", false, false, false),
		srcItem("s2", 1, "schaums-linear-algebra", false, false, false),
		srcItem("ax1", 1, "axler", true, false, false),
		srcItem("ax2", 1, "axler", true, false, false),
		srcItem("ab1", 1, "abbott", true, false, false),
		srcItem("ab2", 1, "abbott", true, false, false),
		srcItem("ab3", 1, "abbott", true, false, false),
		srcItem("ab4", 1, "abbott", true, false, false),
	}
	sc := &Scaffold{Items: items, Generated: genAt}
	sc.ArrangeBooks(BookPolicy{}, "")
	if sc.Main != "axler" || fmt.Sprint(sc.Books) != "[schaums-linear-algebra abbott]" {
		t.Fatalf("main %q books %v", sc.Main, sc.Books)
	}
	served := map[string]bool{}
	var qs []string
	for i := 0; i < 9; i++ {
		p, _ := sc.Pick(served)
		served[p.Question] = true
		qs = append(qs, p.Question)
	}
	if fmt.Sprint(qs[:2]) != "[ax1 ax2]" || qs[3] != "ab1" || qs[5] != "ab2" || qs[6] != "ab3" || qs[7] != "ab4" || !strings.HasPrefix(qs[8], "gen") {
		t.Errorf("order %v", qs)
	}
}

// A topic with one book only serves exactly as before: its exercises, then
// its other items mixed with the generated questions.
func TestScaffold_OneBookIsUnchanged(t *testing.T) {
	sc := &Scaffold{Items: []ScaffoldItem{
		srcItem("hoy ex", 1, "hoy", true, false, false),
		srcItem("hoy card", 1, "hoy", false, false, false),
		item("plain", 1, false, false, false),
	}, Share: 0.5, Generated: genAt}
	sc.ArrangeBooks(BookPolicy{Texts: []string{"hoy"}, ParallelCap: 3}, "")
	if len(sc.Books) != 0 {
		t.Fatalf("books %v", sc.Books)
	}
	// So does a deck of one book that no course names.
	deck := &Scaffold{Items: []ScaffoldItem{srcItem("w1", 1, "wooldridge", false, false, false), srcItem("w2", 1, "wooldridge", false, false, false)}}
	deck.ArrangeBooks(BookPolicy{Texts: []string{"islp"}, ParallelCap: 1}, "")
	if deck.Main != "wooldridge" || len(deck.Books) != 0 {
		t.Fatalf("one-book deck: main %q books %v", deck.Main, deck.Books)
	}
	gens := 0
	for i := 0; i < 200; i++ {
		served := map[string]bool{}
		p, _ := sc.Pick(served)
		if p.Question != "hoy ex" {
			t.Fatalf("book exercise first, got %q", p.Question)
		}
		served[p.Question] = true
		if p, _ = sc.Pick(served); strings.HasPrefix(p.Question, "gen") {
			gens++
		}
	}
	if gens < 60 || gens > 140 {
		t.Errorf("the rest shares with generated questions as before: %d of 200 generated", gens)
	}
}
