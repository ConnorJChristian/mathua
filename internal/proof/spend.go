package proof

import (
	"context"
	"log"
	"time"
)

// The spend guard for callers outside this package that call the model on
// the proof grader's account (internal/recallproof writes marking schemes for
// flashcards). Their calls count against the same monthly budget.

// Reserve holds est dollars against this month's budget while a call runs.
// It fails when the budget would be exceeded. Call release when done, and
// RecordSpend with what the call actually cost.
func (g *Grader) Reserve(ctx context.Context, est float64) (month string, release func(), err error) {
	return g.reserve(ctx, est)
}

// RecordSpend adds a finished call's cost to the month's spend.
func (g *Grader) RecordSpend(month string, usd float64, u Usage) {
	if usd <= 0 {
		return
	}
	if err := g.Store.AddProofSpend(context.Background(), month, usd, u.InputTokens+u.CacheCreationInputTokens+u.CacheReadInputTokens, u.OutputTokens); err != nil {
		log.Printf("proof: spend not recorded: %v", err)
	}
	g.mu.Lock()
	if g.month == month {
		g.spent += usd
	}
	g.mu.Unlock()
}

// Affordable reports whether, as far as the last reading of the spend says,
// est more dollars fit in this month's budget. Unlike Serving it does not
// need proof problems in the bank. The spend is read once a month and then
// kept up to date in memory.
func (g *Grader) Affordable(est float64) bool {
	if g == nil || g.Judge == nil {
		return false
	}
	month := g.now().UTC().Format("2006-01")
	g.mu.Lock()
	known := g.month == month
	spent := g.spent + g.inflight
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
		spent = s + g.inflight
		g.mu.Unlock()
	}
	return spent+est <= g.Cfg.MonthlyUSD
}

// EstimateGrade is the pessimistic cost of grading proof against p (every
// run writes the cache and spends its whole output budget).
func (g *Grader) EstimateGrade(p *Problem, proof string) float64 { return g.estimate(p, proof) }
