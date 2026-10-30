package pytrack

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
)

// A bank with one book exercise ("ex1", a typed answer), a practice version
// that unlocks after it ("pr1") and an ordinary item ("plain").
const bookBank = `{
 "chapter": "t", "title": "T",
 "concepts": [{
  "id": "t.topic", "book": {"title": "Topic"}, "intro": "",
  "items": [
   {"id": "plain", "kind": "mc", "level": 2, "prompt": "Pick one.", "correct": "a", "wrong": ["b", "c"], "explain": "a."},
   {"id": "ex1", "kind": "math", "level": 2, "prompt": "Enter the value asked for.",
    "parts": [{"answer": "7", "grading": "numeric", "tex": "7"}],
    "book_ref": {"file": "book.pdf", "module": "mod", "page": 40, "printed_page": 26, "label": "Book Exercise 1.1, #1"},
    "explain": "Seven."},
   {"id": "pr1", "kind": "math", "level": 2, "prompt": "Practice: what is 3 + 4?",
    "parts": [{"answer": "7", "grading": "numeric", "tex": "7"}],
    "unlocks_after": "ex1", "explain": "Seven."}
  ]
 }]
}`

// writeModule lays out <root>/mod/bank/t.json, and files/book.pdf when pdf.
func writeModule(t *testing.T, bank string, pdf bool) (bankDir string) {
	t.Helper()
	root := t.TempDir()
	bankDir = filepath.Join(root, "mod", "bank")
	if err := os.MkdirAll(bankDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bankDir, "t.json"), []byte(bank), 0o644); err != nil {
		t.Fatal(err)
	}
	if pdf {
		if err := os.MkdirAll(filepath.Join(root, "mod", "files"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "mod", "files", "book.pdf"), []byte("%PDF-1.4"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return bankDir
}

func bookTrack(t *testing.T, pdf bool) (*Track, *conceptGen) {
	t.Helper()
	b, err := LoadBank(writeModule(t, bookBank, pdf))
	if err != nil {
		t.Fatalf("bank: %v", err)
	}
	tr := &Track{Bank: b}
	return tr, &conceptGen{track: tr, c: b.Concept("t.topic")}
}

// fakeHistory answers Outcome from a fixed set of correct answer keys.
type fakeHistory struct{ correct, tried map[string]bool }

func (h fakeHistory) Outcome(answer, question string) (bool, bool) {
	return h.tried[answer] || h.correct[answer], h.correct[answer]
}

func TestBookRefRoundTrips(t *testing.T) {
	_, g := bookTrack(t, true)
	it := g.item("ex1")
	want := &generator.BookRef{File: "book.pdf", Module: "mod", Page: 40, PrintedPage: 26, Label: "Book Exercise 1.1, #1"}
	if !reflect.DeepEqual(it.BookRef, want) {
		t.Fatalf("book_ref = %+v, want %+v", it.BookRef, want)
	}
	if g.item("pr1").UnlocksAfter != "ex1" {
		t.Fatalf("unlocks_after lost: %+v", g.item("pr1"))
	}
	// Written back out, the fields come back as they were read.
	raw, err := json.Marshal(g.c.Items)
	if err != nil {
		t.Fatal(err)
	}
	var again []Item
	if err := json.Unmarshal(raw, &again); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(again, g.c.Items) {
		t.Fatalf("items do not round-trip:\n%s", raw)
	}
	for _, s := range []string{`"book_ref":{"file":"book.pdf","module":"mod","page":40,"printed_page":26,"label":"Book Exercise 1.1, #1"}`, `"unlocks_after":"ex1"`} {
		if !strings.Contains(string(raw), s) {
			t.Fatalf("marshalled items lack %s:\n%s", s, raw)
		}
	}
}

func TestBookRefValidation(t *testing.T) {
	for name, bad := range map[string]string{
		"no label":       strings.Replace(bookBank, `"label": "Book Exercise 1.1, #1"`, `"label": ""`, 1),
		"no page":        strings.Replace(bookBank, `"page": 40`, `"page": 0`, 1),
		"unknown unlock": strings.Replace(bookBank, `"unlocks_after": "ex1"`, `"unlocks_after": "nope"`, 1),
		"self unlock":    strings.Replace(bookBank, `"unlocks_after": "ex1"`, `"unlocks_after": "pr1"`, 1),
		"unknown field":  strings.Replace(bookBank, `"unlocks_after": "ex1"`, `"unlock_after": "ex1"`, 1),
		"arrow in label": strings.Replace(bookBank, `#1"`, `#1 -->"`, 1),
	} {
		if _, err := LoadBank(writeModule(t, bad, false)); err == nil {
			t.Errorf("%s: bank loaded", name)
		}
	}
}

// The served question opens with the marker and a visible line; with the
// PDF installed it carries the file's URL, without it it says so and still
// names the exercise and both page numbers.
func TestBookRefServedWithAndWithoutFile(t *testing.T) {
	for _, pdf := range []bool{true, false} {
		tr, g := bookTrack(t, pdf)
		p := tr.Problem(g.c, g.item("ex1"))
		if !strings.HasPrefix(p.Question, generator.BookMarker) {
			t.Fatalf("pdf=%v: no marker: %q", pdf, p.Question)
		}
		end := strings.Index(p.Question, "-->")
		var wire struct {
			generator.BookRef
			Available bool   `json:"available"`
			URL       string `json:"url"`
		}
		if err := json.Unmarshal([]byte(p.Question[len(generator.BookMarker):end]), &wire); err != nil {
			t.Fatalf("pdf=%v: marker JSON: %v", pdf, err)
		}
		if wire.Available != pdf || (wire.URL != "") != pdf || wire.Page != 40 {
			t.Fatalf("pdf=%v: wire %+v", pdf, wire)
		}
		if pdf && wire.URL != "/api/modules/mod/files/book.pdf" {
			t.Fatalf("url %q", wire.URL)
		}
		rest := p.Question[end:]
		for _, s := range []string{"Book Exercise 1.1, #1", "book page 26", "PDF page 40", "Enter the value asked for."} {
			if !strings.Contains(rest, s) {
				t.Fatalf("pdf=%v: visible text lacks %q: %q", pdf, s, rest)
			}
		}
		if strings.Contains(rest, "not installed") == pdf {
			t.Fatalf("pdf=%v: availability note wrong: %q", pdf, rest)
		}
		if p.Answer != "math:t.topic#ex1" {
			t.Fatalf("answer key %q", p.Answer)
		}
		if res := tr.Grade(p.Answer, "7"); !res.Correct {
			t.Fatalf("book item does not grade its own answer: %+v", res)
		}
	}
}

func serveMany(g *conceptGen, ctx generator.GeneratorContext, n int) map[string]int {
	seen := map[string]int{}
	for i := 0; i < n; i++ {
		p := g.Generate(ctx)
		key := p.Answer
		if strings.HasPrefix(key, "mc:") {
			key = "plain"
		} else {
			key = key[strings.Index(key, "#")+1:]
		}
		seen[key]++
	}
	return seen
}

// Practice that unlocks after a book exercise is hidden until the learner
// answers the exercise correctly, and the exercise comes first until then.
func TestUnlocksAfterHidesPracticeUntilBookItemCorrect(t *testing.T) {
	_, g := bookTrack(t, false)
	none := serveMany(g, generator.GeneratorContext{Difficulty: 0.6}, 60)
	if none["pr1"] > 0 || none["ex1"] != 60 {
		t.Fatalf("no history: want only the book exercise, got %v", none)
	}
	tried := fakeHistory{tried: map[string]bool{"math:t.topic#ex1": true}}
	missed := serveMany(g, generator.GeneratorContext{Difficulty: 0.6, History: tried}, 200)
	if missed["pr1"] > 0 {
		t.Fatalf("practice served after a wrong answer: %v", missed)
	}
	// About half the time (FORK.md, generator/scaffold.go): here the other
	// half goes to the untried plain item, so ex1 is Binomial(200, 1/2). The
	// bound used to be ex1 >= 100, the mean itself, which failed about one
	// run in two; 60..140 is more than five standard deviations either way.
	if missed["ex1"] < 60 || missed["ex1"] > 140 || missed["plain"] == 0 {
		t.Fatalf("a missed book exercise should come back often but not always: %v", missed)
	}
	solved := fakeHistory{correct: map[string]bool{"math:t.topic#ex1": true}}
	after := serveMany(g, generator.GeneratorContext{Difficulty: 0.6, History: solved}, 300)
	if after["pr1"] == 0 || after["plain"] == 0 {
		t.Fatalf("after a correct answer practice should mix in: %v", after)
	}
	if after["ex1"] > 200 {
		t.Fatalf("a solved book exercise is no longer preferred: %v", after)
	}
	if !g.ItemServable("pr1", generator.GeneratorContext{History: solved}) || g.ItemServable("pr1", generator.GeneratorContext{History: tried}) {
		t.Fatal("ItemServable disagrees with Generate")
	}
	if p, _ := g.ItemBookPending("ex1", generator.GeneratorContext{History: solved}); p {
		t.Fatal("a solved book exercise is still pending")
	}
}

// Placement (NoBookRefs with ObjectiveOnly) never serves a book exercise, and
// the objective count leaves book and gated items out.
func TestPlacementSkipsBookItems(t *testing.T) {
	_, g := bookTrack(t, true)
	solved := fakeHistory{correct: map[string]bool{"math:t.topic#ex1": true}}
	got := serveMany(g, generator.GeneratorContext{Difficulty: 0.6, ObjectiveOnly: true, NoCards: true, NoProofs: true, NoBookRefs: true, History: solved}, 100)
	if got["ex1"] > 0 {
		t.Fatalf("placement served a book exercise: %v", got)
	}
	if g.ObjectiveItems() != 1 {
		t.Fatalf("objective items = %d, want 1", g.ObjectiveItems())
	}
	// A review test (ObjectiveOnly without NoBookRefs) may ask it.
	review := serveMany(g, generator.GeneratorContext{Difficulty: 0.6, ObjectiveOnly: true, NoCards: true, NoProofs: true}, 100)
	if review["ex1"] == 0 {
		t.Fatalf("a review test never asked the book exercise: %v", review)
	}
}

// QuestionKey ignores the card marker and the book header, whose note
// changes when the PDF is installed.
func TestQuestionKeyIgnoresHeader(t *testing.T) {
	trWith, gWith := bookTrack(t, true)
	trWithout, gWithout := bookTrack(t, false)
	a := trWith.Problem(gWith.c, gWith.item("ex1")).Question
	b := trWithout.Problem(gWithout.c, gWithout.item("ex1")).Question
	if a == b {
		t.Fatal("the header should differ")
	}
	if generator.QuestionKey(a) != generator.QuestionKey("<!--card-->\n"+b) || generator.QuestionKey(a) == "" {
		t.Fatalf("keys differ: %q vs %q", generator.QuestionKey(a), generator.QuestionKey(b))
	}
}
