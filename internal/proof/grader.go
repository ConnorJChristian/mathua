package proof

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/chuma-beep/mathua/internal/grader"
)

// Grader grades proofs: k judge runs, the median, a cache and a spend guard.
type Grader struct {
	Bank  *Bank
	Judge Judge // nil when no API key is configured
	Store Store
	Cfg   Config

	now func() time.Time

	mu       sync.Mutex
	inflight float64 // dollars reserved by gradings still running
	month    string
	spent    float64 // last spend read for month, for Serving
}

// New builds a grader. A judge is created only when an API key is set; without
// one every proof is reported ungraded, never wrong.
func New(cfg Config, bank *Bank, store Store) *Grader {
	cfg = cfg.normalized()
	g := &Grader{Bank: bank, Store: store, Cfg: cfg, now: time.Now}
	if store == nil {
		g.Store = NewMemStore()
	}
	if cfg.APIKey != "" {
		g.Judge = &AnthropicJudge{
			APIKey:    cfg.APIKey,
			BaseURL:   cfg.APIBase,
			Model:     cfg.Model,
			MaxTokens: cfg.MaxTokens,
			Client:    &http.Client{Timeout: cfg.Timeout},
		}
	}
	return g
}

// Outcome is one grading: the report, or why there is none.
type Outcome struct {
	Report *grader.ProofReport
	// Unavailable is the reason the proof was not graded, or "".
	Unavailable string
	// USD is what this grading cost (0 for a cached grade).
	USD   float64
	Usage Usage
	// Failed counts runs that returned no usable verdict.
	Failed int
}

// Enabled reports a grader that can grade: a judge and at least one problem.
func (g *Grader) Enabled() bool { return g != nil && g.Judge != nil && g.Bank.Len() > 0 }

// Serving reports whether proof problems should be served now: grading is
// enabled and, as far as the last reading of the spend says, this month's
// budget is not used up. Serving a proof that cannot be graded would waste
// the learner's writing.
func (g *Grader) Serving() bool {
	if !g.Enabled() {
		return false
	}
	month := g.now().UTC().Format("2006-01")
	g.mu.Lock()
	known := g.month == month
	spent := g.spent
	g.mu.Unlock()
	if !known {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		s, err := g.Store.ProofSpend(ctx, month)
		if err != nil {
			return false
		}
		g.mu.Lock()
		g.month, g.spent = month, s
		g.mu.Unlock()
		spent = s
	}
	return spent+g.estimate(nil, "") <= g.Cfg.MonthlyUSD
}

// Grade is the grader.SetProofGrader hook: expected is "proof:<id>".
func (g *Grader) Grade(expected, answer string) grader.Result {
	id := strings.TrimPrefix(expected, grader.ProofPrefix)
	p := g.Bank.Problem(id)
	if p == nil {
		return grader.Result{Feedback: "This proof problem is no longer in the bank; load a new question", Unavailable: true}
	}
	ctx, cancel := context.WithTimeout(context.Background(), g.Cfg.Timeout+5*time.Second)
	defer cancel()
	out := g.GradeProof(ctx, p, answer)
	if out.Unavailable != "" {
		return grader.Result{Feedback: out.Unavailable, Unavailable: true}
	}
	r := out.Report
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

// NormalizeProof is the form of a proof that is graded and cached: line
// endings unified, trailing spaces and surrounding blank lines removed.
func NormalizeProof(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// CacheKey identifies a grade: the problem and its marking scheme, the model
// and the proof text.
func (g *Grader) CacheKey(p *Problem, proof string) string {
	h := sha256.Sum256([]byte(p.ID + "\x00" + p.Fingerprint() + "\x00" + g.Cfg.Model + "\x00" + NormalizeProof(proof)))
	return hex.EncodeToString(h[:])
}

// estimate is a pessimistic dollar cost of one grading: every run writes the
// cache and spends its whole output budget. Text is reckoned at three
// characters a token, which overcounts English and LaTeX.
func (g *Grader) estimate(p *Problem, proof string) float64 {
	chars := len(instructions) + len(proof) + 200
	if p != nil {
		chars += len(problemBlock(p))
	} else {
		chars += 6000
	}
	u := Usage{CacheCreationInputTokens: chars / 3, OutputTokens: g.Cfg.MaxTokens}
	return float64(g.Cfg.Runs) * g.Cfg.Prices.Cost(u)
}

// reserve holds an estimate against the month's budget while a grading runs.
func (g *Grader) reserve(ctx context.Context, est float64) (month string, release func(), err error) {
	month = g.now().UTC().Format("2006-01")
	spent, err := g.Store.ProofSpend(ctx, month)
	if err != nil {
		return "", nil, fmt.Errorf("the proof budget could not be read: %v", err)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.month, g.spent = month, spent
	if spent+g.inflight+est > g.Cfg.MonthlyUSD {
		return "", nil, fmt.Errorf("this month's proof-grading budget is used up ($%.2f of $%.2f); raise MATHUA_PROOF_MONTHLY_USD or wait for next month", spent, g.Cfg.MonthlyUSD)
	}
	g.inflight += est
	return month, func() {
		g.mu.Lock()
		g.inflight -= est
		g.mu.Unlock()
	}, nil
}

type run struct {
	v     Verdict
	usage Usage
	err   error
}

// GradeProof grades one proof of p.
func (g *Grader) GradeProof(ctx context.Context, p *Problem, proof string) Outcome {
	proof = NormalizeProof(proof)
	if proof == "" {
		return Outcome{Unavailable: "Write your proof in the box first"}
	}
	if len(proof) > maxProofCharacters {
		return Outcome{Unavailable: fmt.Sprintf("That proof is longer than %d characters; shorten it and submit again", maxProofCharacters)}
	}
	key := g.CacheKey(p, proof)
	if raw, ok, err := g.Store.ProofGrade(ctx, key); err == nil && ok {
		var r grader.ProofReport
		if json.Unmarshal([]byte(raw), &r) == nil {
			r.Cached = true
			r.CountsForMastery = g.Cfg.CountsForMastery
			return Outcome{Report: &r}
		}
	}
	if g.Judge == nil {
		return Outcome{Unavailable: "Proof grading is off: ANTHROPIC_API_KEY is not set on the server"}
	}
	month, release, err := g.reserve(ctx, g.estimate(p, proof))
	if err != nil {
		return Outcome{Unavailable: err.Error()}
	}
	defer release()

	k := g.Cfg.Runs
	runs := make([]run, k)
	var wg sync.WaitGroup
	for i := 0; i < k; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			v, u, err := g.Judge.Judge(ctx, p, proof)
			runs[i] = run{v, u, err}
		}(i)
	}
	wg.Wait()

	out := Outcome{}
	var valid []Verdict
	var firstErr error
	for _, r := range runs {
		out.Usage = out.Usage.Add(r.usage)
		out.USD += g.Cfg.Prices.Cost(r.usage)
		if r.err != nil {
			out.Failed++
			if firstErr == nil {
				firstErr = r.err
			}
			continue
		}
		valid = append(valid, r.v)
	}
	if out.USD > 0 {
		if err := g.Store.AddProofSpend(context.Background(), month, out.USD, out.Usage.InputTokens+out.Usage.CacheCreationInputTokens+out.Usage.CacheReadInputTokens, out.Usage.OutputTokens); err != nil {
			log.Printf("proof: spend not recorded: %v", err)
		}
		g.mu.Lock()
		if g.month == month {
			g.spent += out.USD
		}
		g.mu.Unlock()
	}
	// A grade needs a majority of the runs. Fewer valid verdicts than that is
	// not a grade, whatever they say.
	if len(valid) < k/2+1 {
		reason := "the judge did not return enough verdicts"
		if firstErr != nil {
			reason = firstErr.Error()
			if errors.Is(firstErr, ErrRefused) {
				reason = "the judge declined to grade this proof"
			} else if errors.Is(firstErr, ErrUnparseable) {
				reason = "the judge's reply could not be read"
			}
		}
		out.Unavailable = "Your proof was not graded (" + reason + "). Nothing was recorded; submit again."
		return out
	}
	out.Report = g.aggregate(p, valid)
	if raw, err := json.Marshal(out.Report); err == nil {
		if err := g.Store.PutProofGrade(context.Background(), key, string(raw)); err != nil {
			log.Printf("proof: grade not cached: %v", err)
		}
	}
	return out
}

// aggregate turns the valid runs into one report. The grade is the median
// score (the lower middle for an even count, to err strict); the milestone
// checklist, first error and feedback are those of a run that gave the median,
// so the explanation shown matches the score shown. Fully correct needs every
// valid run at the pass mark or above.
func (g *Grader) aggregate(p *Problem, valid []Verdict) *grader.ProofReport {
	scores := make([]int, len(valid))
	for i, v := range valid {
		scores[i] = v.Score
	}
	sorted := append([]int(nil), scores...)
	sort.Ints(sorted)
	median := sorted[(len(sorted)-1)/2]
	rep := valid[0]
	for _, v := range valid {
		if v.Score == median {
			rep = v
			break
		}
	}
	full := true
	for _, s := range scores {
		if s < PassMark {
			full = false
		}
	}
	met := map[string]MilestoneVerdict{}
	for _, m := range rep.Milestones {
		met[m.ID] = m
	}
	r := &grader.ProofReport{
		Score:            median,
		Max:              MaxScore,
		Runs:             scores,
		FullyCorrect:     full,
		FirstError:       rep.FirstError,
		Feedback:         rep.Feedback,
		CountsForMastery: g.Cfg.CountsForMastery,
		Model:            g.Cfg.Model,
	}
	for _, m := range p.Milestones {
		v := met[m.ID]
		r.Milestones = append(r.Milestones, grader.ProofMilestone{ID: m.ID, Desc: m.Desc, Points: m.Points, Met: v.Met, Note: v.Note})
	}
	return r
}
