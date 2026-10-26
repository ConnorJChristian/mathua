// Package recallproof grades written answers to flashcards that ask for a
// proof, a derivation or a justification (internal/pytrack written.go
// decides which cards those are).
//
// It reuses the proof grader (internal/proof) for everything about judging:
// the Anthropic judge, k runs and the median, "fully right" only when every
// run gives at least 6 of 7, the grade cache and the monthly spend guard.
// What it adds is the marking scheme. A flashcard has a front and a terse
// back, not a rubric, so the first time a card's written answer is graded a
// scheme is written for it by the model from the front and back (RefGrader
// style: a reference answer, 0-7 milestones, caps for fatal errors,
// alternatives), stored, and reused for every later answer. The scheme is
// stored in the proof grader's store (the proof_grades table) under a key
// that includes a hash of the card's front and back, so editing a card gets
// it a new scheme. Writing a scheme is billed to the same monthly budget.
//
// Whenever a scheme cannot be had (no key, the budget spent, the model
// judged the card ungradable, a failed call) the card is served and graded
// as the ordinary self-graded flashcard it is.
package recallproof

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chuma-beep/mathua/internal/grader"
	"github.com/chuma-beep/mathua/internal/proof"
	"github.com/chuma-beep/mathua/internal/pytrack"
)

// Grader is the written-answer grader (pytrack.WrittenGrader).
type Grader struct {
	P      *proof.Grader
	Writer SchemeWriter
	Cfg    Config

	now func() time.Time

	mu      sync.Mutex
	known   map[string]*record // by store key
	flights map[string]*flight
}

type flight struct {
	done chan struct{}
	rec  *record
	err  error
}

// New builds a grader on a proof grader. With an API key on the proof grader
// and no writer given, schemes are written by the proof grader's model.
func New(p *proof.Grader, cfg Config, w SchemeWriter) *Grader {
	cfg = cfg.normalized()
	if w == nil && p != nil && p.Cfg.APIKey != "" {
		w = &AnthropicWriter{
			APIKey:    p.Cfg.APIKey,
			BaseURL:   p.Cfg.APIBase,
			Model:     p.Cfg.Model,
			MaxTokens: cfg.SchemeMaxTokens,
			Client:    &http.Client{Timeout: cfg.SchemeTimeout},
		}
	}
	return &Grader{P: p, Writer: w, Cfg: cfg, now: time.Now, known: map[string]*record{}, flights: map[string]*flight{}}
}

// On reports that written answers can be graded at all.
func (g *Grader) On() bool {
	return g != nil && g.Cfg.On && g.P != nil && g.P.Judge != nil && g.Writer != nil
}

// ServeWritten reports whether the card should be served for a written
// answer now. It never calls the model.
func (g *Grader) ServeWritten(c pytrack.RecallCard) bool {
	if !g.On() || !g.Cfg.BankAllowed(c.Bank) {
		return false
	}
	est := g.EstimateGrade(c)
	rec, _ := g.lookup(context.Background(), c)
	switch {
	case rec == nil:
		est += g.EstimateScheme(c)
	case rec.Scheme == nil:
		if !g.retryDue(rec) {
			return false
		}
		est += g.EstimateScheme(c)
	}
	return g.P.Affordable(est)
}

// notGraded wraps a reason the answer was not graded. The learner keeps the
// answer and can submit again or mark the card themselves.
func notGraded(reason string) grader.Result {
	reason = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(reason), "."))
	return grader.Result{Unavailable: true, Feedback: reason + ". Nothing was recorded: submit again, or use “Just show me the answer” and mark it yourself."}
}

// GradeWritten grades a written answer to the card.
func (g *Grader) GradeWritten(c pytrack.RecallCard, answer string) grader.Result {
	if !g.On() {
		return notGraded("Written answers are not being graded on this server")
	}
	if !g.Cfg.BankAllowed(c.Bank) {
		return notGraded("Written answers to this deck are not graded (MATHUA_LLM_RECALL_EXCLUDE_BANKS)")
	}
	if strings.TrimSpace(answer) == "" {
		return notGraded("Write your answer in the box first")
	}
	ctx, cancel := context.WithTimeout(context.Background(), g.P.Cfg.Timeout+5*time.Second)
	defer cancel()
	s, err := g.Scheme(ctx, c)
	if err != nil {
		return notGraded("This card's marking scheme could not be written (" + err.Error() + ")")
	}
	p := s.Problem(c)
	out := g.P.GradeProof(ctx, &p, answer)
	if out.Unavailable != "" {
		return notGraded(strings.TrimSuffix(out.Unavailable, " Nothing was recorded; submit again."))
	}
	r := out.Report
	// The card stays a flashcard: its grade is practice, recorded exactly as
	// a self-mark (engine.practiceOnly), whatever proof grades count for.
	r.CountsForMastery = false
	res := grader.Result{
		Correct:  r.FullyCorrect,
		Score:    float64(r.Score) / float64(r.Max),
		Feedback: fmt.Sprintf("Score %d/%d. %s", r.Score, r.Max, r.Feedback),
		Proof:    r,
	}
	if !r.FullyCorrect {
		res.Diagnosis = r.FirstError
	}
	return res
}

// Scheme returns the card's marking scheme, writing it the first time. A
// scheme being written is shared by every caller that asks for it meanwhile,
// and the writing goes on after ctx ends, so a grading that timed out leaves
// a scheme for the retry.
func (g *Grader) Scheme(ctx context.Context, c pytrack.RecallCard) (*Scheme, error) {
	rec, err := g.lookup(ctx, c)
	if err != nil {
		return nil, err
	}
	if rec != nil {
		if rec.Scheme != nil {
			return rec.Scheme, nil
		}
		if !g.retryDue(rec) {
			return nil, errors.New(rec.Failed)
		}
	}
	if g.Writer == nil {
		return nil, errors.New("no scheme writer is configured")
	}
	key := storeKey(c)
	g.mu.Lock()
	f := g.flights[key]
	if f == nil {
		f = &flight{done: make(chan struct{})}
		g.flights[key] = f
		go g.write(c, key, f)
	}
	g.mu.Unlock()
	select {
	case <-f.done:
		if f.err != nil {
			return nil, f.err
		}
		return f.rec.Scheme, nil
	case <-ctx.Done():
		return nil, errors.New("writing it took too long; it is still being written, so submit again in a minute")
	}
}

// write writes and stores one card's scheme (Scheme's flight).
func (g *Grader) write(c pytrack.RecallCard, key string, f *flight) {
	defer func() {
		g.mu.Lock()
		delete(g.flights, key)
		g.mu.Unlock()
		close(f.done)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), g.Cfg.SchemeTimeout)
	defer cancel()
	month, release, err := g.P.Reserve(ctx, g.EstimateScheme(c))
	if err != nil {
		// The budget, not the card: nothing to remember.
		f.err = err
		return
	}
	s, u, err := g.Writer.WriteScheme(ctx, c)
	release()
	usd := g.P.Cfg.Prices.Cost(u)
	g.P.RecordSpend(month, usd, u)
	rec := &record{Card: c.Key, Hash: CardHash(c), Model: g.P.Cfg.Model, USD: usd, Created: g.now().UTC()}
	if err == nil {
		err = s.Validate(c)
	}
	if err != nil {
		rec.Failed = err.Error()
		rec.Permanent = errors.Is(err, ErrNotGradable)
		if !rec.Permanent {
			rec.RetryAt = g.now().Add(g.Cfg.RetryAfter).UTC()
		}
		log.Printf("recallproof: no marking scheme for %s: %v", c.Key, err)
		f.err = err
	} else {
		rec.Scheme = s
	}
	f.rec = rec
	g.remember(key, rec)
	if raw, jerr := json.Marshal(rec); jerr == nil {
		if perr := g.P.Store.PutProofGrade(context.Background(), key, string(raw)); perr != nil {
			log.Printf("recallproof: scheme for %s not stored: %v", c.Key, perr)
		}
	}
}

// lookup finds what is known of the card: in memory, else in the store. A
// card never seen is nil.
func (g *Grader) lookup(ctx context.Context, c pytrack.RecallCard) (*record, error) {
	key := storeKey(c)
	g.mu.Lock()
	rec, ok := g.known[key]
	g.mu.Unlock()
	if ok {
		return rec, nil
	}
	if g.P == nil || g.P.Store == nil {
		return nil, nil
	}
	raw, found, err := g.P.Store.ProofGrade(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("the scheme store could not be read: %v", err)
	}
	if !found {
		return nil, nil
	}
	var r record
	if err := json.Unmarshal([]byte(raw), &r); err != nil || (r.Scheme == nil && r.Failed == "") {
		return nil, nil
	}
	g.remember(key, &r)
	return &r, nil
}

func (g *Grader) remember(key string, r *record) {
	g.mu.Lock()
	g.known[key] = r
	g.mu.Unlock()
}

// retryDue reports whether a failed card may be tried again.
func (g *Grader) retryDue(r *record) bool {
	return !r.Permanent && !g.now().Before(r.RetryAt)
}

// HasScheme reports whether the card already has a stored scheme.
func (g *Grader) HasScheme(ctx context.Context, c pytrack.RecallCard) bool {
	r, _ := g.lookup(ctx, c)
	return r != nil && r.Scheme != nil
}

// Failure is why the card has no scheme ("" when it has one or was never
// tried), and whether that lasts until the card is edited.
func (g *Grader) Failure(ctx context.Context, c pytrack.RecallCard) (string, bool) {
	r, _ := g.lookup(ctx, c)
	if r == nil || r.Scheme != nil {
		return "", false
	}
	return r.Failed, r.Permanent
}

// EstimateScheme is the pessimistic cost of writing the card's scheme: the
// whole output budget spent, text at three characters a token.
func (g *Grader) EstimateScheme(c pytrack.RecallCard) float64 {
	chars := len(schemeInstructions) + len(c.Prompt) + len(c.Back) + 300
	return g.P.Cfg.Prices.Cost(proof.Usage{InputTokens: chars / 3, OutputTokens: g.Cfg.SchemeMaxTokens})
}

// EstimateGrade is the pessimistic cost of grading one answer to the card.
// The problem block is reckoned at the prompt and back plus 4,000 characters
// of scheme and reference, and the answer at 2,000 characters.
func (g *Grader) EstimateGrade(c pytrack.RecallCard) float64 {
	p := proof.Problem{Statement: c.Prompt, Reference: c.Back + strings.Repeat(" ", 4000)}
	return g.P.EstimateGrade(&p, strings.Repeat(" ", 2000))
}

// TypicalScheme and TypicalGrade are what writing a scheme and grading an
// answer usually cost, for the CLI's estimate next to the pessimistic one:
// about 1,200 output tokens for a scheme and 500 per judge run. The runs of
// one grading start together, so each writes the prompt cache.
func (g *Grader) TypicalScheme(c pytrack.RecallCard) float64 {
	chars := len(schemeInstructions) + len(c.Prompt) + len(c.Back) + 300
	return g.P.Cfg.Prices.Cost(proof.Usage{InputTokens: chars / 4, OutputTokens: 1200})
}

// TypicalGrade: see TypicalScheme.
func (g *Grader) TypicalGrade(c pytrack.RecallCard) float64 {
	block := (len(c.Prompt) + len(c.Back) + 3500 + 3200) / 4 // instructions, problem, scheme, reference
	run := proof.Usage{CacheCreationInputTokens: block, InputTokens: 800 / 4, OutputTokens: 500}
	return float64(g.P.Cfg.Runs) * g.P.Cfg.Prices.Cost(run)
}
