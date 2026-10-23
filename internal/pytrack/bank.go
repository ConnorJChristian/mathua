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
)

// Item is one question.
type Item struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Level  int    `json:"level"` // 1 easy, 2 standard, 3 stretch
	Prompt string `json:"prompt"`

	// Multiple choice.
	Correct string   `json:"correct,omitempty"`
	Wrong   []string `json:"wrong,omitempty"`

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

	// Recall card.
	Back string `json:"back,omitempty"`

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
}

// LoadBank reads and validates every *.json bank file in the given folders
// (each module's bank/ folder), in folder order and then file-name order.
func LoadBank(dirs ...string) (*Bank, error) {
	b := &Bank{concepts: map[string]*Concept{}, items: map[string]*Item{}}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
				names = append(names, e.Name())
			}
		}
		sort.Strings(names)
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
			b.chapters = append(b.chapters, ch)
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
	seen := map[string]bool{}
	for _, it := range c.Items {
		where := fmt.Sprintf("concept %q item %q", c.ID, it.ID)
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
		if it.Kind != KindRecall && strings.TrimSpace(it.Explain) == "" {
			return fmt.Errorf("%s: explain is required", where)
		}
		switch it.Kind {
		case KindChoice:
			if strings.TrimSpace(it.Correct) == "" || len(it.Wrong) < 2 {
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
			if strings.TrimSpace(it.Back) == "" {
				return fmt.Errorf("%s: recall card needs a back", where)
			}
			if strings.Contains(it.Prompt, RecallMarker) {
				return fmt.Errorf("%s: the front may not contain the recall marker", where)
			}
		default:
			return fmt.Errorf("%s: unknown kind %q", where, it.Kind)
		}
	}
	return nil
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

// Item resolves "<concept>#<item>".
func (b *Bank) Item(key string) *Item { return b.items[key] }

// Chapters returns the bank files in order.
func (b *Bank) Chapters() []Chapter { return b.chapters }
