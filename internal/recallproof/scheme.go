package recallproof

import (
	"crypto/sha256"
	"errors"
	"encoding/hex"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/chuma-beep/mathua/internal/proof"
	"github.com/chuma-beep/mathua/internal/pytrack"
)

// Scheme is a marking scheme written for one flashcard, in the proof
// grader's terms (internal/proof Problem): milestones worth 7 points in all,
// caps for fatal errors, accepted alternative routes and a reference answer,
// expanded from the card's back when the back is terse.
type Scheme struct {
	Reference    string            `json:"reference"`
	Given        string            `json:"given,omitempty"`
	Milestones   []proof.Milestone `json:"milestones"`
	Caps         []proof.Cap       `json:"caps,omitempty"`
	Alternatives []string          `json:"alternatives,omitempty"`
}

// record is what is stored for a card: its scheme, or why there is none.
type record struct {
	Card   string  `json:"card"` // the item key, "<concept>#<item>"
	Hash   string  `json:"hash"` // CardHash when it was written
	Scheme *Scheme `json:"scheme,omitempty"`
	// Failed is why no scheme could be written. Permanent failures (the
	// model judged the card ungradable as a written answer) last until the
	// card is edited; others until RetryAt.
	Failed    string    `json:"failed,omitempty"`
	Permanent bool      `json:"permanent,omitempty"`
	RetryAt   time.Time `json:"retry_at,omitempty"`
	Model     string    `json:"model,omitempty"`
	USD       float64   `json:"usd,omitempty"`
	Created   time.Time `json:"created"`
}

// CardHash identifies a card's content: editing its front or back changes it,
// and with it the storage key, so the card gets a new scheme.
func CardHash(c pytrack.RecallCard) string {
	h := sha256.Sum256([]byte(c.Prompt + "\x00" + c.Back))
	return hex.EncodeToString(h[:8])
}

// storeKey is where a card's scheme is kept in the proof grader's store (the
// proof_grades table): beside the grade cache, under its own prefix.
func storeKey(c pytrack.RecallCard) string {
	return "recall-scheme:v1:" + c.Key + ":" + CardHash(c)
}

var (
	brRe    = regexp.MustCompile(`(?i)<br\s*/?>|</(p|div|li|tr)>`)
	tagRe   = regexp.MustCompile(`(?s)<[^>]*>`)
	blankRe = regexp.MustCompile(`\n{3,}`)
)

// CardText is a card's front or back as plain text for the model: line
// breaks kept, other tags removed, entities decoded, LaTeX untouched.
func CardText(s string) string {
	s = brRe.ReplaceAllString(s, "\n")
	s = tagRe.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(l)
	}
	return strings.TrimSpace(blankRe.ReplaceAllString(strings.Join(lines, "\n"), "\n\n"))
}

// Problem is the card and its scheme as a proof problem for the judge. The
// reference shown to the judge is the scheme's, followed by the card's own
// back, so a scheme that drifted from the card cannot quietly replace it.
func (s *Scheme) Problem(c pytrack.RecallCard) proof.Problem {
	level := c.Level
	if level < 1 || level > 3 {
		level = 2
	}
	stmt := CardText(c.Prompt)
	title := strings.Join(strings.Fields(stmt), " ")
	if r := []rune(title); len(r) > 80 {
		title = string(r[:80]) + "…"
	}
	ref := strings.TrimSpace(s.Reference)
	if back := CardText(c.Back); back != "" {
		ref += "\n\nTHE FLASHCARD'S OWN ANSWER (terse; the reference above expands it)\n" + back
	}
	return proof.Problem{
		ID:           "recall." + strings.NewReplacer("#", "~", " ", "_", "\n", "_").Replace(c.Key),
		Concept:      c.Concept,
		Title:        title,
		Level:        level,
		Statement:    strings.ReplaceAll(stmt, proof.Marker, ""),
		Given:        s.Given,
		Reference:    ref,
		Milestones:   s.Milestones,
		Caps:         s.Caps,
		Alternatives: s.Alternatives,
	}
}

// Validate checks the scheme makes a usable proof problem for the card.
func (s *Scheme) Validate(c pytrack.RecallCard) error {
	if strings.TrimSpace(s.Reference) == "" {
		return errors.New("the scheme has no reference answer")
	}
	p := s.Problem(c)
	return p.Validate()
}
