package proof

import (
	"sync"

	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
	"github.com/chuma-beep/mathua/internal/modules"
)

var (
	defaultOnce sync.Once
	defaultG    *Grader
	defaultErr  error
)

// Default is the process-wide proof grader: the enabled modules' proofs/
// folders and the MATHUA_PROOF_* environment. It installs itself as the
// router's proof grader. Its store is in memory until SetStore gives it the
// database. With MATHUA_PUBLIC_MODULES_ONLY=1, or with no module that has a
// proofs/ folder, the bank is empty and no proof is ever served.
func Default() (*Grader, error) {
	defaultOnce.Do(func() {
		mods, err := modules.Default()
		if err != nil {
			defaultErr = err
			return
		}
		b, err := LoadBank(mods.ProofDirs()...)
		if err != nil {
			defaultErr = err
			return
		}
		defaultG = New(ConfigFromEnv(), b, nil)
		grader.SetProofGrader(defaultG.Grade)
	})
	return defaultG, defaultErr
}

// SetStore gives the grader its persistent store (the database), so the
// monthly spend and the grade cache survive restarts.
func (g *Grader) SetStore(s Store) {
	if g == nil || s == nil {
		return
	}
	g.mu.Lock()
	g.Store = s
	g.month = ""
	g.mu.Unlock()
}

// CountsForMastery reports whether proof results are mastery evidence in this
// process (MATHUA_PROOF_COUNTS_FOR_MASTERY; see Config.CountsForMastery).
func CountsForMastery() bool {
	g, err := Default()
	return err == nil && g != nil && g.Cfg.CountsForMastery
}

// ByConcept lists the problem ids filed under each topic.
func (g *Grader) ByConcept() map[string][]string { return g.Bank.ByConcept() }

// ProblemFor serves one problem by id.
func (g *Grader) ProblemFor(id string) (generator.Problem, bool) { return g.Bank.ProblemFor(id) }

// Share is the fraction of a topic's practice that is proofs.
func (g *Grader) Share() float64 { return g.Cfg.Share }
