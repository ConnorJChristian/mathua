// Package pytrack serves question banks: multiple choice, "what does this
// print", code tasks graded by hidden tests, and recall cards. A bank belongs
// to a module (internal/modules); the Python track is modules/python-dsh.
//
// The concepts themselves live in each module's concepts/ folder in the same
// format as data/concepts, so the concept graph, mastery evidence, spaced
// review and the planner treat them exactly as they treat mathematics. Only
// the questions and their grading are new.
//
// Banks are read from disk at startup. Hidden tests and fixtures stay on the
// server: the generator serves the prompt and the answer key, never a test.
package pytrack

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/chuma-beep/mathua/internal/generator"
)

// Book points a concept at the matching section of the Python Data Science
// Handbook. The book's text is CC-BY-NC-ND, so the track links to it and never
// adapts its prose.
type Book struct {
	Name    string `json:"name,omitempty"` // the book; defaults to the Python Data Science Handbook
	Section string `json:"section"`        // "2.02"
	Title   string `json:"title"`
	URL     string `json:"url"`
}

// Test is one hidden check for a code task. Code runs after the learner's code
// with the learner's names in scope plus the helpers in pyrunner.py; it fails
// by raising AssertionError, whose message the learner sees.
type Test struct {
	Name string `json:"name"`
	Code string `json:"code"`
	Hint string `json:"hint,omitempty"`
}

// Item kinds.
const (
	KindChoice = "mc"
	KindOutput = "output"
	KindCode   = "code"
	// KindRecall is a flashcard: front, back, and the learner's own verdict.
	KindRecall = "recall"
	// KindMath is a question with one or more typed mathematical answers,
	// graded by Mathua's own math graders (numeric, or SymPy equivalence),
	// exactly as Mathua's generated questions are. Imported Anki practice
	// problems use it (scripts/anki_typed.py).
	KindMath = "math"
	// KindTerm asks the learner to type a term, the concept's name as the
	// source uses it (fork change). The answer is Correct, with any other
	// spellings the source uses in Accept; grading ignores case, accents, a
	// leading article and surrounding punctuation (track.go normalizeTerm).
	KindTerm = "term"
)

// Part is one typed answer of a math item: "x_0 = [60]".
type Part struct {
	// Label is shown before the input as "<label> =", in LaTeX. Required when
	// an item has more than one part, so the learner knows which is which.
	Label string `json:"label,omitempty"`
	// Answer is the expected answer in the plain form the graders read, which
	// is what the math editor produces from what the learner types.
	Answer string `json:"answer"`
	// Grading is the grader: "numeric" or "expression".
	Grading string `json:"grading"`
	// Tex is the answer as the source wrote it, shown after answering.
	Tex string `json:"tex"`
}

// Item is one question.
type Item struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Level  int    `json:"level"` // 1 easy, 2 standard, 3 stretch
	Prompt string `json:"prompt"`

	// Multiple choice, and the answer of a term item.
	Correct string   `json:"correct,omitempty"`
	Wrong   []string `json:"wrong,omitempty"`
	// Accept lists other spellings of a term item's answer.
	Accept []string `json:"accept,omitempty"`

	// "What does this print?" Expected is checked against a real run by
	// scripts/python_bank.py, so it cannot drift from the library versions the
	// image pins.
	Code     string `json:"code,omitempty"`
	Expected string `json:"expected,omitempty"`

	// Code task.
	Starter  string `json:"starter,omitempty"`
	Setup    string `json:"setup,omitempty"`
	Solution string `json:"solution,omitempty"`
	Tests    []Test `json:"tests,omitempty"`

	// Recall card, and the worked solution of a math item.
	Back string `json:"back,omitempty"`
	// Grading overrides how a recall card is graded (fork change): "llm"
	// always serves it for a written answer graded by the proof judge, "self"
	// never does. Empty leaves it to Classify (written.go).
	Grading string `json:"grading,omitempty"`

	// Math item.
	Parts []Part `json:"parts,omitempty"`

	// BookRef makes a page of the learner's own book the item's prompt
	// (fork change): the item names an exercise and opens the PDF at it, and
	// the bank keeps only the reference, the answer key and an explanation
	// in its own words. Prompt then says what to enter, never the exercise.
	BookRef *generator.BookRef `json:"book_ref,omitempty"`
	// UnlocksAfter names another item of the same concept: this item is not
	// served to a learner until they have answered that one correctly at
	// least once (fork change). Practice versions of a book exercise use it.
	UnlocksAfter string `json:"unlocks_after,omitempty"`

	Explain string `json:"explain"`
}

// Concept is the bank entry for one concept in the graph.
type Concept struct {
	ID    string `json:"id"`
	Book  Book   `json:"book"`
	Intro string `json:"intro"`
	Items []Item `json:"items"`
}

// Chapter is one bank file.
type Chapter struct {
	Chapter  string    `json:"chapter"`
	Title    string    `json:"title"`
	Concepts []Concept `json:"concepts"`
}

// Bank is the loaded question bank.
type Bank struct {
	chapters []Chapter
	concepts map[string]*Concept
	items    map[string]*Item // "<concept>#<item>"
	// bankOf names the bank file each concept is in, without ".json".
	bankOf map[string]string
	// moduleDirs maps a module id to its folder, for the files/ a book
	// reference points into: every module whose bank/ was loaded, by the
	// folder's name, and its sibling folders (another module of the same
	// install). Default replaces it with the module set's own lookup.
	moduleDirs []string
	// FileExists reports whether a module's files/ folder has a file. Nil
	// uses moduleDirs.
	FileExists func(module, file string) bool
}

// LoadBank reads and validates every *.json bank file in the given folders
// (each module's bank/ folder), in folder order and then file-name order.
func LoadBank(dirs ...string) (*Bank, error) {
	b := &Bank{concepts: map[string]*Concept{}, items: map[string]*Item{}, bankOf: map[string]string{}}
	var files []string // the bank file of each chapter, in order
	for _, dir := range dirs {
		b.moduleDirs = append(b.moduleDirs, filepath.Dir(filepath.Clean(dir)))
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") && e.Name() != "skip.json" {
				names = append(names, e.Name())
			}
		}
		sort.Strings(names)
		// Fork change: a bank's skip.json lists item ids not to serve (cards
		// that need a figure or a numbered equation they do not show, OCR
		// damage): {"<item id>": "<reason>"}. The items stay in the bank
		// files, so a re-import keeps the list meaningful.
		skip := map[string]string{}
		if raw, err := os.ReadFile(filepath.Join(dir, "skip.json")); err == nil {
			if err := json.Unmarshal(raw, &skip); err != nil {
				return nil, fmt.Errorf("%s: %w", filepath.Join(dir, "skip.json"), err)
			}
		}
		for _, name := range names {
			raw, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				return nil, err
			}
			var ch Chapter
			dec := json.NewDecoder(strings.NewReader(string(raw)))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&ch); err != nil {
				return nil, fmt.Errorf("%s: %w", filepath.Join(dir, name), err)
			}
			if len(skip) > 0 {
				for i := range ch.Concepts {
					kept := ch.Concepts[i].Items[:0]
					for _, it := range ch.Concepts[i].Items {
						if _, drop := skip[it.ID]; !drop {
							kept = append(kept, it)
						}
					}
					ch.Concepts[i].Items = kept
				}
			}
			b.chapters = append(b.chapters, ch)
			files = append(files, strings.TrimSuffix(name, ".json"))
		}
	}
	for ci := range b.chapters {
		ch := &b.chapters[ci]
		for i := range ch.Concepts {
			c := &ch.Concepts[i]
			if err := validateConcept(c); err != nil {
				return nil, fmt.Errorf("bank/%s: %w", ch.Chapter, err)
			}
			if _, dup := b.concepts[c.ID]; dup {
				return nil, fmt.Errorf("concept %q appears twice in the bank", c.ID)
			}
			b.concepts[c.ID] = c
			b.bankOf[c.ID] = files[ci]
			for j := range c.Items {
				it := &c.Items[j]
				b.items[itemKey(c.ID, it.ID)] = it
			}
		}
	}
	return b, nil
}

func itemKey(conceptID, itemID string) string { return conceptID + "#" + itemID }

func validateConcept(c *Concept) error {
	if c.ID == "" {
		return fmt.Errorf("concept with empty id")
	}
	if c.Book.Title == "" {
		return fmt.Errorf("concept %q has no source section", c.ID)
	}
	if len(c.Items) == 0 {
		return fmt.Errorf("concept %q has no questions", c.ID)
	}
	ids := make(map[string]bool, len(c.Items))
	for _, it := range c.Items {
		ids[it.ID] = true
	}
	seen := map[string]bool{}
	for _, it := range c.Items {
		where := fmt.Sprintf("concept %q item %q", c.ID, it.ID)
		if err := it.BookRef.Validate(); err != nil {
			return fmt.Errorf("%s: %w", where, err)
		}
		if it.UnlocksAfter != "" {
			if it.UnlocksAfter == it.ID || !ids[it.UnlocksAfter] {
				return fmt.Errorf("%s: unlocks_after %q must name another item of the concept", where, it.UnlocksAfter)
			}
		}
		if it.BookRef != nil && (it.Kind == KindOutput || it.Kind == KindCode) {
			return fmt.Errorf("%s: a book reference takes a typed, term, choice or recall answer", where)
		}
		if it.ID == "" || strings.ContainsAny(it.ID, "#: ") {
			return fmt.Errorf("%s: id must be non-empty without '#', ':' or spaces", where)
		}
		if seen[it.ID] {
			return fmt.Errorf("%s: duplicate id", where)
		}
		seen[it.ID] = true
		if it.Level < 1 || it.Level > 3 {
			return fmt.Errorf("%s: level must be 1-3", where)
		}
		if strings.TrimSpace(it.Prompt) == "" {
			return fmt.Errorf("%s: prompt is required", where)
		}
		if it.Kind != KindRecall && it.Kind != KindMath && strings.TrimSpace(it.Explain) == "" {
			return fmt.Errorf("%s: explain is required", where)
		}
		if it.Grading != "" && it.Kind != KindRecall {
			return fmt.Errorf("%s: only a recall card takes a grading override", where)
		}
		switch it.Kind {
		case KindChoice:
			if strings.TrimSpace(it.Correct) == "" || len(it.Wrong) < 2 && !binaryChoice(it) {
				return fmt.Errorf("%s: multiple choice needs correct and at least two wrong options", where)
			}
			for _, w := range it.Wrong {
				if strings.TrimSpace(w) == strings.TrimSpace(it.Correct) {
					return fmt.Errorf("%s: a wrong option equals the correct one", where)
				}
			}
			for _, o := range append([]string{it.Correct}, it.Wrong...) {
				if strings.Contains(o, "\n") {
					return fmt.Errorf("%s: options must be one line", where)
				}
			}
		case KindOutput:
			if strings.TrimSpace(it.Code) == "" || it.Expected == "" {
				return fmt.Errorf("%s: output question needs code and expected", where)
			}
		case KindCode:
			if strings.TrimSpace(it.Solution) == "" || len(it.Tests) == 0 {
				return fmt.Errorf("%s: code task needs a solution and tests", where)
			}
			for _, t := range it.Tests {
				if strings.TrimSpace(t.Code) == "" || t.Name == "" {
					return fmt.Errorf("%s: every test needs a name and code", where)
				}
			}
		case KindRecall:
			if it.Grading != "" && it.Grading != GradingLLM && it.Grading != GradingSelf {
				return fmt.Errorf("%s: grading must be %q or %q", where, GradingLLM, GradingSelf)
			}
			if strings.TrimSpace(it.Back) == "" {
				return fmt.Errorf("%s: recall card needs a back", where)
			}
			if strings.Contains(it.Prompt, RecallMarker) {
				return fmt.Errorf("%s: the front may not contain the recall marker", where)
			}
		case KindMath:
			if len(it.Parts) == 0 || len(it.Parts) > MaxParts {
				return fmt.Errorf("%s: a math item needs 1-%d parts", where, MaxParts)
			}
			labels := map[string]bool{}
			for _, p := range it.Parts {
				if strings.TrimSpace(p.Answer) == "" || strings.Contains(p.Answer, "\n") {
					return fmt.Errorf("%s: every part needs a one-line answer", where)
				}
				switch p.Grading {
				case "numeric", "expression", "set", "interval":
				default:
					return fmt.Errorf("%s: part grading must be numeric, expression, set or interval", where)
				}
				if len(it.Parts) > 1 && strings.TrimSpace(p.Label) == "" {
					return fmt.Errorf("%s: every part of a multi-part item needs a label", where)
				}
				if labels[p.Label] {
					return fmt.Errorf("%s: two parts share a label", where)
				}
				labels[p.Label] = true
			}
			if strings.Contains(it.Prompt, TypedMarker) {
				return fmt.Errorf("%s: the prompt may not contain the typed-answer marker", where)
			}
		case KindTerm:
			if strings.TrimSpace(it.Correct) == "" || strings.Contains(it.Correct, "\n") {
				return fmt.Errorf("%s: a term item needs a one-line answer", where)
			}
			for _, a := range it.Accept {
				if strings.TrimSpace(a) == "" || strings.Contains(a, "\n") {
					return fmt.Errorf("%s: every accepted spelling must be one non-empty line", where)
				}
			}
			if len(it.Wrong) > 0 {
				return fmt.Errorf("%s: a term item has no wrong options", where)
			}
			if strings.Contains(it.Prompt, TermMarker) {
				return fmt.Errorf("%s: the prompt may not contain the term-answer marker", where)
			}
		default:
			return fmt.Errorf("%s: unknown kind %q", where, it.Kind)
		}
	}
	return nil
}

// MaxParts is the most typed answers one math item may ask for.
const MaxParts = 4

// binaryChoice is a two-option question whose options are each other's
// opposite (True/False, Yes/No, and in Spanish Sí/No and Verdadero/Falso),
// where a third option would be invented.
func binaryChoice(it Item) bool {
	if len(it.Wrong) != 1 {
		return false
	}
	pair := strings.ToLower(strings.TrimSpace(it.Correct)) + "/" + strings.ToLower(strings.TrimSpace(it.Wrong[0]))
	switch pair {
	case "true/false", "false/true", "yes/no", "no/yes",
		"sí/no", "no/sí", "verdadero/falso", "falso/verdadero":
		return true
	}
	return false
}

// negativeOption reports whether a binary choice option is the negative one,
// which is served second.
func negativeOption(o string) bool {
	switch strings.ToLower(strings.TrimSpace(o)) {
	case "false", "no", "falso":
		return true
	}
	return false
}

// fileExists reports whether a module's files/ folder has file.
func (b *Bank) fileExists(module, file string) bool {
	if b.FileExists != nil {
		return b.FileExists(module, file)
	}
	if !safeRel(file) || module == "" || strings.ContainsAny(module, "/\\") || module == ".." {
		return false
	}
	seen := map[string]bool{}
	for _, md := range b.moduleDirs {
		for _, dir := range []string{md, filepath.Join(filepath.Dir(md), module)} {
			if filepath.Base(dir) != module || seen[dir] {
				continue
			}
			seen[dir] = true
			if st, err := os.Stat(filepath.Join(dir, "files", filepath.FromSlash(file))); err == nil && !st.IsDir() {
				return true
			}
		}
	}
	return false
}

// safeRel reports a relative path that stays inside its folder.
func safeRel(p string) bool {
	if p == "" || filepath.IsAbs(p) || strings.HasPrefix(p, "/") || strings.HasPrefix(p, "\\") {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(p))
	return clean != ".." && !strings.HasPrefix(clean, "../")
}

// Concept returns the bank entry for a concept, or nil.
func (b *Bank) Concept(id string) *Concept { return b.concepts[id] }

// ConceptIDs lists every concept with a bank entry, in bank order.
func (b *Bank) ConceptIDs() []string {
	var out []string
	for _, ch := range b.chapters {
		for _, c := range ch.Concepts {
			out = append(out, c.ID)
		}
	}
	return out
}

// BankOf names the bank file a concept is in, without ".json"
// ("anki_calculus"), or "" for an unknown concept.
func (b *Bank) BankOf(conceptID string) string { return b.bankOf[conceptID] }

// Item resolves "<concept>#<item>".
func (b *Bank) Item(key string) *Item { return b.items[key] }

// Chapters returns the bank files in order.
func (b *Bank) Chapters() []Chapter { return b.chapters }
