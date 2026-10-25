// Package proof grades written proofs with a language-model judge against a
// marking scheme authored in advance.
//
// The design follows two findings on grading proofs with language models
// (ProofGrader, arXiv 2510.13888; RefGrader, arXiv 2510.09021): a judge given
// a reference solution and a milestone marking scheme agrees with human
// graders far better than one asked to grade from the statement alone, and
// the median of several sampled runs is steadier than any single run. So:
//
//   - every problem carries its statement, a reference proof, a 0-7 marking
//     scheme of milestones, caps for the common fatal errors and the accepted
//     alternative routes. These are authored once, offline, in a module's
//     proofs/ folder; nothing is written at grading time;
//   - the judge runs k times concurrently and the median score is the grade;
//     a proof counts as correct only when every run gives it at least 6 of 7;
//   - grading strictly is the instruction, because the costly error is a
//     flawed proof passed as correct.
//
// Problems are content, so they live in modules (normally a git-ignored
// modules-local/ module, since marking schemes follow a textbook). The public
// build ships none.
package proof

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
)

// Marker is placed in a served proof question, after the statement. Every
// answer host recognises it and shows a multi-line proof box instead of its
// usual input (web/next-app/lib/pythonQuestion.ts PROOF_MARKER).
const Marker = "<!--proof-answer-->"

// Milestone is one line of the marking scheme.
type Milestone struct {
	ID     string `json:"id"`
	Points int    `json:"points"`
	Desc   string `json:"desc"`
}

// Cap is a fatal error and the most a proof making it can score.
type Cap struct {
	Error string `json:"error"`
	Max   int    `json:"max"`
}

// Sample is a synthetic proof with the score its author intends, for a
// day-one sanity check of the judge (mathua proof calibrate -synthetic).
type Sample struct {
	// Kind is "correct", "subtle" (plausible with a real gap) or "circular".
	Kind  string `json:"kind"`
	Proof string `json:"proof"`
	Score int    `json:"score"`
	Note  string `json:"note,omitempty"`
}

// Problem is one proof problem and its marking scheme.
type Problem struct {
	ID string `json:"id"`
	// Concept is the topic whose practice serves this problem.
	Concept string `json:"concept"`
	Title   string `json:"title"`
	// Source names where the statement is from ("Thm 1.4.2, p. 21").
	Source string `json:"source,omitempty"`
	Level  int    `json:"level"`
	// Statement is what to prove, Markdown with LaTeX.
	Statement string `json:"statement"`
	// Given lists what the learner may cite without proof.
	Given string `json:"given,omitempty"`
	// Reference is a complete proof at the intended level of rigour.
	Reference    string      `json:"reference_proof"`
	Milestones   []Milestone `json:"milestones"`
	Caps         []Cap       `json:"caps,omitempty"`
	Alternatives []string    `json:"alternatives,omitempty"`
	Samples      []Sample    `json:"samples,omitempty"`
}

// File is one proofs/*.json file.
type File struct {
	Chapter  string    `json:"chapter"`
	Title    string    `json:"title"`
	Problems []Problem `json:"problems"`
}

// Bank is the loaded proof problems.
type Bank struct {
	order    []string
	problems map[string]*Problem
}

// LoadBank reads and validates every *.json file in the given folders (each
// module's proofs/ folder), in folder order and then file-name order.
func LoadBank(dirs ...string) (*Bank, error) {
	b := &Bank{problems: map[string]*Problem{}}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		var names []string
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
				names = append(names, e.Name())
			}
		}
		sort.Strings(names)
		for _, name := range names {
			path := filepath.Join(dir, name)
			raw, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			var f File
			dec := json.NewDecoder(strings.NewReader(string(raw)))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&f); err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			for i := range f.Problems {
				p := f.Problems[i]
				if err := p.Validate(); err != nil {
					return nil, fmt.Errorf("%s: %w", path, err)
				}
				if _, dup := b.problems[p.ID]; dup {
					return nil, fmt.Errorf("%s: proof problem %q appears twice", path, p.ID)
				}
				b.problems[p.ID] = &p
				b.order = append(b.order, p.ID)
			}
		}
	}
	return b, nil
}

// NewBank builds a bank from problems already in memory (tests, tools).
func NewBank(ps ...Problem) (*Bank, error) {
	b := &Bank{problems: map[string]*Problem{}}
	for i := range ps {
		p := ps[i]
		if err := p.Validate(); err != nil {
			return nil, err
		}
		if _, dup := b.problems[p.ID]; dup {
			return nil, fmt.Errorf("proof problem %q appears twice", p.ID)
		}
		b.problems[p.ID] = &p
		b.order = append(b.order, p.ID)
	}
	return b, nil
}

// Validate checks a problem's marking scheme is usable.
func (p *Problem) Validate() error {
	where := fmt.Sprintf("proof problem %q", p.ID)
	if p.ID == "" || strings.ContainsAny(p.ID, " \n#") {
		return fmt.Errorf("%s: id must be non-empty without spaces or '#'", where)
	}
	if p.Concept == "" {
		return fmt.Errorf("%s: concept is required", where)
	}
	if strings.TrimSpace(p.Title) == "" || strings.TrimSpace(p.Statement) == "" || strings.TrimSpace(p.Reference) == "" {
		return fmt.Errorf("%s: title, statement and reference_proof are required", where)
	}
	if p.Level < 1 || p.Level > 3 {
		return fmt.Errorf("%s: level must be 1-3", where)
	}
	if strings.Contains(p.Statement, Marker) {
		return fmt.Errorf("%s: the statement may not contain the proof marker", where)
	}
	total := 0
	seen := map[string]bool{}
	for _, m := range p.Milestones {
		if m.ID == "" || seen[m.ID] {
			return fmt.Errorf("%s: milestone ids must be non-empty and unique", where)
		}
		seen[m.ID] = true
		if m.Points < 1 || strings.TrimSpace(m.Desc) == "" {
			return fmt.Errorf("%s: milestone %q needs points and a description", where, m.ID)
		}
		total += m.Points
	}
	if total != MaxScore {
		return fmt.Errorf("%s: milestone points sum to %d, want %d", where, total, MaxScore)
	}
	for _, c := range p.Caps {
		if strings.TrimSpace(c.Error) == "" || c.Max < 0 || c.Max >= MaxScore {
			return fmt.Errorf("%s: a cap needs an error and a max of 0-%d", where, MaxScore-1)
		}
	}
	for _, s := range p.Samples {
		if s.Score < 0 || s.Score > MaxScore || strings.TrimSpace(s.Proof) == "" {
			return fmt.Errorf("%s: a sample needs a proof and a score of 0-%d", where, MaxScore)
		}
	}
	return nil
}

// Fingerprint changes whenever anything the judge is shown about the problem
// changes, so editing a rubric invalidates the grades cached under it.
func (p *Problem) Fingerprint() string {
	raw, _ := json.Marshal(struct {
		S, G, R string
		M       []Milestone
		C       []Cap
		A       []string
	}{p.Statement, p.Given, p.Reference, p.Milestones, p.Caps, p.Alternatives})
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:8])
}

// Problem returns a problem by id, or nil.
func (b *Bank) Problem(id string) *Problem {
	if b == nil {
		return nil
	}
	return b.problems[id]
}

// IDs lists the problems in bank order.
func (b *Bank) IDs() []string {
	if b == nil {
		return nil
	}
	return append([]string(nil), b.order...)
}

// Len is the number of problems.
func (b *Bank) Len() int {
	if b == nil {
		return 0
	}
	return len(b.order)
}

// ByConcept lists the problem ids served in each topic's practice.
func (b *Bank) ByConcept() map[string][]string {
	out := map[string][]string{}
	if b == nil {
		return out
	}
	for _, id := range b.order {
		c := b.problems[id].Concept
		out[c] = append(out[c], id)
	}
	return out
}

// Served renders a problem as a served question. The answer key names the
// problem; the marking scheme never leaves the server. The reference proof is
// the explanation, shown after grading.
func Served(p *Problem) generator.Problem {
	var q strings.Builder
	q.WriteString("**Prove.** ")
	q.WriteString(strings.TrimSpace(p.Statement))
	if g := strings.TrimSpace(p.Given); g != "" {
		q.WriteString("\n\n*You may use:* ")
		q.WriteString(g)
	}
	q.WriteString("\n\n")
	q.WriteString(Marker)
	exp := "**A reference proof.**\n\n" + strings.TrimSpace(p.Reference)
	if p.Source != "" {
		exp += "\n\nSource: " + p.Source
	}
	return generator.Problem{
		Question:    q.String(),
		Answer:      grader.ProofPrefix + p.ID,
		Explanation: exp,
		Raw:         true,
	}
}

// ProblemFor serves one problem by id. With it the bank is an item server for
// the engine's card mixing (internal/engine/cards.go).
func (b *Bank) ProblemFor(id string) (generator.Problem, bool) {
	p := b.Problem(id)
	if p == nil {
		return generator.Problem{}, false
	}
	return Served(p), true
}
