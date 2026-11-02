package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/modules"
	"github.com/chuma-beep/mathua/internal/planning"
	"github.com/chuma-beep/mathua/internal/pytrack"
)

// policyHistory is a stepHistory in a course's book context.
type policyHistory struct {
	stepHistory
	policy generator.BookPolicy
}

func (h policyHistory) BookPolicy() generator.BookPolicy { return h.policy }

func recallJSON(id, src string, level int, extra string) string {
	s := ""
	if src != "" {
		s = fmt.Sprintf(`,"source_book":%q`, src)
	}
	return fmt.Sprintf(`{"id":%q,"kind":"recall","level":%d,"prompt":"Q %s","back":"A %s","explain":""%s%s}`, id, level, id, id, s, extra)
}

// A generated topic (derivative rules) in a Hoy course, with Hoy's book
// exercises, Schaum's and Leithold cards and an Axler card filed under it,
// and an Abbott section tied to it by a module's `encompasses`: Hoy first,
// then Schaum's and Leithold taking turns (with Abbott's linked exercise),
// three of each, then the generated problem sets; Axler (excluded by the
// course) and the linked section's Hoy item never.
func TestParallelBooks_ServedOrderOnAGeneratedTopic(t *testing.T) {
	var deck []string
	for i := 1; i <= 2; i++ {
		deck = append(deck, recallJSON(fmt.Sprintf("hoy%d", i), "", 1, `,"book_ref":{"file":"hoy-3e.pdf","module":"m","page":5,"label":"Hoy Exercise 5.4"}`))
	}
	for i := 1; i <= 4; i++ {
		deck = append(deck, recallJSON(fmt.Sprintf("schaum%d", i), "Schaums_Calc_Econ", 1, ""))
		deck = append(deck, recallJSON(fmt.Sprintf("leithold%d", i), "Leithold", 1, ""))
	}
	deck = append(deck, recallJSON("axler1", "Axler_Linear_Algebra_Done_Right", 1, ""))
	sec := []string{
		recallJSON("abbott1", "", 1, `,"book_ref":{"file":"abbott-2e.pdf","module":"m","page":9,"label":"Abbott Exercise 5.2.1"}`),
		recallJSON("hoyalso", "hoy", 1, ""),
	}
	bank := `{"chapter":"d","title":"D","concepts":[{"id":"deck.d","book":{"title":"D"},"intro":"","items":[` + strings.Join(deck, ",") + `]},
	 {"id":"ma.abbott.5_2","book":{"name":"Abbott, Understanding Analysis (2e)","title":"5.2"},"intro":"","items":[` + strings.Join(sec, ",") + `]}]}`
	dir := filepath.Join(t.TempDir(), "m", "bank")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "d.json"), []byte(bank), 0o644)
	b, err := pytrack.LoadBank(dir)
	if err != nil {
		t.Fatal(err)
	}
	th := concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 30}
	d, err := concepts.Build([]concepts.Concept{
		{ID: "calc.rules", Label: "Rules", Domain: "calculus", GradingType: "numeric", MasteryThreshold: th},
		{ID: "deck.d", Label: "Deck", Domain: "calculus", GradingType: "bank", Prerequisites: []string{"calc.rules"}, MasteryThreshold: th},
		{ID: "ma.abbott.5_2", Label: "5.2", Domain: "analysis", GradingType: "bank", Prerequisites: []string{"calc.rules"}, MasteryThreshold: th},
	})
	if err != nil {
		t.Fatal(err)
	}
	reg := generator.NewRegistry()
	reg.Register("calc.rules", diffGen{})
	if err := (&pytrack.Track{Bank: b}).Register(reg); err != nil {
		t.Fatal(err)
	}
	var refs []modules.CardRef
	for _, id := range []string{"axler1", "leithold1", "schaum1", "hoy1", "leithold2", "schaum2", "hoy2", "leithold3", "schaum3", "leithold4", "schaum4"} {
		refs = append(refs, modules.CardRef{Deck: "deck.d", Item: id})
	}
	attachCardsLinked(d, reg, map[string][]modules.CardRef{"calc.rules": refs},
		map[string][]string{"calc.rules": {"ma.abbott.5_2"}, "ma.abbott.5_2": {"calc.rules"}})

	h := policyHistory{stepHistory{mapHistory: mapHistory{}}, generator.BookPolicy{
		Texts: []string{"hoy", "schaums-calc-econ"}, ParallelCap: 3, Exclude: map[string]bool{"axler": true}}}
	for run := 0; run < 20; run++ {
		probs, err := reg.BatchGenerateContext("calc.rules", 14, generator.GeneratorContext{Difficulty: 0.4, History: h})
		if err != nil {
			t.Fatal(err)
		}
		var order []string
		for _, p := range probs {
			if strings.Contains(p.Question, "Q axler1") || strings.Contains(p.Question, "Q hoyalso") {
				t.Fatalf("served %q", p.Question)
			}
			src := p.SourceBook
			if p.Tier == generator.TierPractice {
				// The practice set: generated questions, and the books'
				// items past their caps.
				src = "practice"
			}
			order = append(order, src)
		}
		want := "[hoy hoy schaums-calc-econ abbott leithold schaums-calc-econ leithold schaums-calc-econ leithold practice practice practice practice practice]"
		if got := fmt.Sprint(order); got != want {
			t.Fatalf("served %v, want %s", order, want)
		}
		if fmt.Sprint(probs[0].StepPath) != "[hoy schaums-calc-econ abbott leithold ]" || probs[0].Tier != generator.TierMain || probs[2].Tier != generator.TierParallel {
			t.Fatalf("path %q tier %s/%s", probs[0].StepPath, probs[0].Tier, probs[2].Tier)
		}
	}
	// No course context: the book exercises' book is the main book still.
	plain := stepHistory{mapHistory: mapHistory{}}
	p, _ := reg.GenerateContext("calc.rules", generator.GeneratorContext{Difficulty: 0.4, History: plain})
	if p.SourceBook != "hoy" {
		t.Errorf("first served %q from %q", p.Question, p.SourceBook)
	}
}

// A course's texts, parallel_book_cap and parallel_exclude_books make the
// topic's book policy; the course's own texts are never excluded.
func TestParallelBooks_CoursePolicy(t *testing.T) {
	two, zero := 2, 0
	pl := planning.NewWithCourses(nil, []*planning.Course{
		{ID: "c.hoy", Plan: []string{"t"}, Texts: []string{"m/hoy-3e.pdf", "m/schaums-calc-econ.pdf"}, ParallelBookCap: &two, ParallelExcludeBooks: []string{"Axler", "hoy"}},
		{ID: "c.abbott", Plan: []string{"t"}, Texts: []string{"m/abbott-2e.pdf"}, ParallelBookCap: &zero},
	})
	e := &Engine{planner: pl}
	p := e.bookPolicy("t", []string{"c.hoy", "c.abbott"})
	if fmt.Sprint(p.Texts) != "[hoy schaums-calc-econ abbott]" || p.ParallelCap != 2 || !p.Exclude["axler"] || p.Exclude["hoy"] {
		t.Fatalf("policy %+v", p)
	}
	if p := e.bookPolicy("t", nil); p.ParallelCap != generator.EnvParallelBookCap() || p.Texts != nil {
		t.Fatalf("no course: %+v", p)
	}
}
