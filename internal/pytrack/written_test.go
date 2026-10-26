package pytrack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
)

func recallItem(prompt string) *Item {
	return &Item{ID: "x", Kind: KindRecall, Prompt: prompt, Back: "Because it is."}
}

func TestClassifyWrittenAnswerCards(t *testing.T) {
	written := []string{
		`Prove that \(\sqrt 2\) is irrational.`,
		`Prove by induction that \(1 + 2 + \dots + n = n(n+1)/2\).`,
		`Disprove: every continuous function is differentiable.`,
		`Show that the sum of two even integers is even.`,
		`Let \(A\) be a square matrix with \(A^2 = 0\). Show that \(A\) is not invertible.`,
		`Use the total differential to show that the MRS of \(U = x^a y^b\) is \(ay/(bx)\).`,
		`Show \(f(x) = x^3\) is strictly increasing.`,
		`Let \(x_n \to 0\) and \(y_n\) be bounded. Show \(x_n y_n \to 0\).`,
		`Show algebraically that the two estimators coincide.`,
		`Show, by evaluating the determinant, that the system has a unique solution.`,
		`Derive the OLS slope estimator from the first-order conditions.`,
		`Explain why the sample mean is unbiased.`,
		`Is the matrix below symmetric? Justify your answer.`,
		`Verify that \(u = e^{x}\cos y\) is harmonic.`,
		`Confirm that the critical point is a maximum using the second-order condition.`,
		`Give an example, or argue that it is impossible: a bounded sequence with no convergent subsequence.`,
		`Demonstrate that the relation is an equivalence relation.`,
		`Show that the graph \(K_{3,3}\) is not planar.`, // graph theory: "graph" is the object
	}
	for _, p := range written {
		if c := Classify(recallItem(p)); !c.Written {
			t.Errorf("should be written-answer (%s): %s", c.Reason, p)
		}
	}
	self := []string{
		`What is the derivative of \(x^2\)?`,
		`Sketch the graph of \(y = x^2\) and show the vertex.`,
		`Use a graph to show how a tax shifts the supply curve.`,
		`Make a table of values for \(2^x\) and use it to confirm that it never equals zero.`,
		`Find \(\int_0^1 x\,dx\). Show your work.`,
		`Evaluate the improper integral, or show that it diverges: \(\int_1^\infty x^{-2}dx\).`,
		`Find the inverse of the matrix (or show that it has none).`,
		`What does the comparison show about convexity?`,
		`The results show that demand is elastic. What is the elasticity?`,
		`How do you prove that a set is a subspace?`,
		`Describe in words how you would prove that A is not a subset of B.`,
		`Definition 4.3 is helpful to show that a function is [...] at a point.`,
		`"Prove It" is the title of chapter 1. Who wrote it?`, // "prove" in a title
		`Solve \(2x = 4\) and verify your answer.`,
		`Data: wage2 (woo.data('wage2')). Regress log(wage) on educ and show that the slope is positive.`,
		`<img src="fig.png"> Prove the claim in the figure.`,
		`The figure below shows a cost curve. Prove that it is convex.`,
	}
	for _, p := range self {
		if c := Classify(recallItem(p)); c.Written {
			t.Errorf("should be self-graded (%s): %s", c.Reason, p)
		}
	}
	if c := Classify(&Item{Kind: KindChoice, Prompt: "Prove that 1 = 1"}); c.Written {
		t.Error("only a recall card is ever a written-answer card")
	}
}

func writeBank(t *testing.T, items string) *Bank {
	t.Helper()
	dir := t.TempDir()
	bank := `{"chapter":"x","title":"X","concepts":[{"id":"anki.x.a","book":{"name":"Deck","section":"","title":"Ch 2","url":""},"intro":"","items":[` + items + `]}]}`
	if err := os.WriteFile(filepath.Join(dir, "anki_x.json"), []byte(bank), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := LoadBank(dir)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestGradingOverride(t *testing.T) {
	b := writeBank(t, `
{"id":"a","kind":"recall","level":2,"prompt":"Prove that 2 is even.","back":"2 = 2*1.","grading":"self"},
{"id":"b","kind":"recall","level":2,"prompt":"Name the two Peano axioms you use most.","back":"Induction and successor.","grading":"llm"},
{"id":"c","kind":"recall","level":2,"prompt":"Prove that 2 is even.","back":"2 = 2*1."}`)
	if c := Classify(b.Item("anki.x.a#a")); c.Written || c.Reason != "grading: self" {
		t.Errorf("grading: self should win over the keyword: %+v", c)
	}
	if c := Classify(b.Item("anki.x.a#b")); !c.Written || c.Reason != "grading: llm" {
		t.Errorf("grading: llm should make any card written-answer: %+v", c)
	}
	if c := Classify(b.Item("anki.x.a#c")); !c.Written {
		t.Errorf("no override leaves it to the keywords: %+v", c)
	}
	if b.BankOf("anki.x.a") != "anki_x" {
		t.Errorf("BankOf = %q", b.BankOf("anki.x.a"))
	}
	for _, bad := range []string{
		`{"id":"a","kind":"recall","level":2,"prompt":"P","back":"B","grading":"judge"}`,
		`{"id":"a","kind":"mc","level":2,"prompt":"P","correct":"1","wrong":["2","3"],"explain":"E","grading":"llm"}`,
	} {
		dir := t.TempDir()
		bank := `{"chapter":"x","title":"X","concepts":[{"id":"anki.x.a","book":{"name":"D","section":"","title":"","url":""},"intro":"","items":[` + bad + `]}]}`
		os.WriteFile(filepath.Join(dir, "x.json"), []byte(bank), 0o644)
		if _, err := LoadBank(dir); err == nil {
			t.Errorf("accepted a bad grading override: %s", bad)
		}
	}
}

// fakeWritten is a written-answer grader that records what it is asked.
type fakeWritten struct {
	on, serve bool
	graded    []string
	result    grader.Result
}

func (f *fakeWritten) On() bool                       { return f.on }
func (f *fakeWritten) ServeWritten(c RecallCard) bool { return f.serve }
func (f *fakeWritten) GradeWritten(c RecallCard, a string) grader.Result {
	f.graded = append(f.graded, c.Key+"|"+c.Bank+"|"+a)
	return f.result
}

func installWritten(t *testing.T, w WrittenGrader) {
	t.Helper()
	SetWrittenGrader(w)
	t.Cleanup(func() { SetWrittenGrader(nil) })
}

const writtenItems = `
{"id":"w","kind":"recall","level":2,"prompt":"Prove that the sum of two odd numbers is even.","back":"(2a+1)+(2b+1) = 2(a+b+1)."},
{"id":"s","kind":"recall","level":2,"prompt":"What is 2+2?","back":"4"},
{"id":"m","kind":"mc","level":2,"prompt":"Pick 1","correct":"1","wrong":["2","3"],"explain":"It is 1."}`

func TestWrittenCardServing(t *testing.T) {
	b := writeBank(t, writtenItems)
	tr := &Track{Bank: b}
	c := b.Concept("anki.x.a")
	w := &c.Items[0]

	// No grader (no API key): an ordinary recall card.
	p := tr.Problem(c, w)
	if strings.Contains(p.Question, WrittenMarker) || !strings.Contains(p.Question, RecallMarker) {
		t.Fatalf("without a grader the card should be plain recall: %q", p.Question)
	}

	f := &fakeWritten{on: true, serve: true}
	installWritten(t, f)
	p = tr.Problem(c, w)
	front := strings.Index(p.Question, "Prove that")
	mark := strings.Index(p.Question, WrittenMarker)
	back := strings.Index(p.Question, RecallMarker)
	if !(front == 0 && mark > front && back > mark) || !strings.HasSuffix(p.Question, "2(a+b+1).") {
		t.Fatalf("a written-answer card is front, written marker, recall marker, back: %q", p.Question)
	}
	if !strings.HasPrefix(p.Answer, prefixRecall) {
		t.Errorf("a written-answer card keeps its recall key: %q", p.Answer)
	}
	if q := tr.Problem(c, &c.Items[1]).Question; strings.Contains(q, WrittenMarker) {
		t.Error("a card that asks for no argument is never served for a written answer")
	}
	// The grader says no (bank excluded, budget spent, scheme failed).
	f.serve = false
	if q := tr.Problem(c, w).Question; strings.Contains(q, WrittenMarker) {
		t.Error("a card the grader will not take is served as plain recall")
	}
	f.serve = true

	// A quiz asks for no written answer; placement and course tests serve
	// no recall card at all.
	g := &conceptGen{track: tr, c: c}
	for i := 0; i < 200; i++ {
		if q := g.Generate(generator.GeneratorContext{Difficulty: 0.6, NoProofs: true}).Question; strings.Contains(q, WrittenMarker) {
			t.Fatalf("a quiz served a written-answer box: %q", q)
		}
		for _, ctx := range []generator.GeneratorContext{
			{Difficulty: 0.6, NoCards: true, NoProofs: true, ObjectiveOnly: true}, // placement
			{Difficulty: 0.6, NoProofs: true, NoCards: true, ObjectiveOnly: true}, // course review test
		} {
			if p := g.Generate(ctx); strings.HasPrefix(p.Answer, prefixRecall) {
				t.Fatalf("placement or a course test served a recall card: %q", p.Question)
			}
		}
	}
	if got := PlainRecall(tr.Problem(c, w)).Question; got != "Prove that the sum of two odd numbers is even.\n\n"+RecallMarker+"\n\n(2a+1)+(2b+1) = 2(a+b+1)." {
		t.Errorf("PlainRecall: %q", got)
	}
}

func TestWrittenCardGrading(t *testing.T) {
	b := writeBank(t, writtenItems)
	tr := &Track{Bank: b}
	key := prefixRecall + "anki.x.a#w"

	// No grader: a written answer is ungraded, never a miss.
	if r := tr.Grade(key, "Let the odd numbers be 2a+1 and 2b+1 ..."); !r.Unavailable || r.Correct {
		t.Errorf("no grader should leave the answer ungraded: %+v", r)
	}

	f := &fakeWritten{on: true, serve: true, result: grader.Result{Correct: true, Score: 1, Feedback: "Score 7/7."}}
	installWritten(t, f)
	if r := tr.Grade(key, "my proof"); !r.Correct || len(f.graded) != 1 || f.graded[0] != "anki.x.a#w|anki_x|my proof" {
		t.Errorf("a written answer goes to the judge: %+v %v", r, f.graded)
	}
	// The escape hatch: the learner revealed the back and marked it.
	if r := tr.Grade(key, "right"); !r.Correct || r.Unavailable {
		t.Errorf("'right' is the learner's own mark: %+v", r)
	}
	if r := tr.Grade(key, "wrong"); r.Correct || r.Unavailable {
		t.Errorf("'wrong' is the learner's own mark: %+v", r)
	}
	if len(f.graded) != 1 {
		t.Errorf("a self-mark must not reach the judge: %v", f.graded)
	}
	// Text typed for a card that is not a written-answer card, or one no
	// longer in the bank: ungraded, never a miss.
	for _, k := range []string{prefixRecall + "anki.x.a#s", prefixRecall + "anki.x.a#gone"} {
		if r := tr.Grade(k, "some text"); !r.Unavailable {
			t.Errorf("%s: text that cannot be graded should be ungraded: %+v", k, r)
		}
	}
	// The judge's own failure passes through as ungraded.
	f.result = grader.Result{Unavailable: true, Feedback: "budget spent"}
	if r := tr.Grade(key, "another proof"); !r.Unavailable {
		t.Errorf("an ungraded judge result must stay ungraded: %+v", r)
	}
}
