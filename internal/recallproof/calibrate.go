package recallproof

import (
	"context"
	"fmt"
	"io"

	"github.com/chuma-beep/mathua/internal/proof"
	"github.com/chuma-beep/mathua/internal/pytrack"
)

// Calibration for written-answer flashcards (mathua proof calibrate -recall).
// There are no human grades to compare with, so each sampled card is graded
// twice: its own back, which should pass, and a plausibly wrong answer the
// model writes against the card's scheme, which should fail. The two error
// rates are the judge's false-fail rate on terse correct answers and its
// false-pass rate on confident wrong ones; the second is the one that
// matters (proof.Config.CountsForMastery).

// CalCard is one sampled card's outcome.
type CalCard struct {
	Card pytrack.RecallCard
	// Err is why the card could not be calibrated (no scheme, the budget),
	// or "".
	Err string
	// BackScore and BackPass grade the card's own back; BackErr is why it
	// was not graded.
	BackScore int
	BackPass  bool
	BackErr   string
	// Wrong is the wrong answer; WrongScore and WrongPass its grade.
	Wrong      string
	WrongScore int
	WrongPass  bool
	WrongErr   string
}

// CalMatrix is the confusion matrix over graded answers.
type CalMatrix struct {
	Cards int
	// BackPass/BackFail: the card's back passed or did not (should pass).
	BackPass, BackFail int
	// WrongPass/WrongFail: the wrong answer passed or did not (should fail).
	WrongPass, WrongFail int
	Ungraded             int
	USD                  float64
}

// String is the report printed after a run.
func (m CalMatrix) String() string {
	pct := func(a, n int) float64 {
		if n == 0 {
			return 0
		}
		return 100 * float64(a) / float64(n)
	}
	return fmt.Sprintf(`cards               %d (ungraded answers %d)
                    judged pass   judged fail
card's own back     %11d   %11d   (should pass; false-fail %.0f%%)
wrong answer        %11d   %11d   (should fail; false-pass %.0f%%)
cost                $%.4f`,
		m.Cards, m.Ungraded, m.BackPass, m.BackFail, pct(m.BackFail, m.BackPass+m.BackFail),
		m.WrongPass, m.WrongFail, pct(m.WrongPass, m.WrongPass+m.WrongFail), m.USD)
}

// Calibrate grades each card's back and a wrong answer to it, writing the
// card's scheme first when it has none. Everything it spends counts against
// the monthly budget; the cost reported is the change in the month's spend.
func (g *Grader) Calibrate(ctx context.Context, cards []pytrack.RecallCard, vw VariantWriter, progress io.Writer) ([]CalCard, CalMatrix) {
	month := g.now().UTC().Format("2006-01")
	before, _ := g.P.Store.ProofSpend(ctx, month)
	var out []CalCard
	var m CalMatrix
	for _, c := range cards {
		r := CalCard{Card: c}
		s, err := g.Scheme(ctx, c)
		if err != nil {
			r.Err = "no scheme: " + err.Error()
		} else {
			p := s.Problem(c)
			if o := g.P.GradeProof(ctx, &p, CardText(c.Back)); o.Unavailable != "" {
				r.BackErr = o.Unavailable
			} else {
				r.BackScore, r.BackPass = o.Report.Score, o.Report.FullyCorrect
			}
			r.Wrong, err = g.wrongAnswer(ctx, c, s, vw)
			if err != nil {
				r.WrongErr = "no wrong answer: " + err.Error()
			} else if o := g.P.GradeProof(ctx, &p, r.Wrong); o.Unavailable != "" {
				r.WrongErr = o.Unavailable
			} else {
				r.WrongScore, r.WrongPass = o.Report.Score, o.Report.FullyCorrect
			}
		}
		m.add(r)
		out = append(out, r)
		if progress != nil {
			fmt.Fprintln(progress, r.line())
		}
	}
	after, _ := g.P.Store.ProofSpend(ctx, month)
	m.USD = after - before
	return out, m
}

func (m *CalMatrix) add(r CalCard) {
	m.Cards++
	if r.Err != "" {
		m.Ungraded += 2
		return
	}
	switch {
	case r.BackErr != "":
		m.Ungraded++
	case r.BackPass:
		m.BackPass++
	default:
		m.BackFail++
	}
	switch {
	case r.WrongErr != "":
		m.Ungraded++
	case r.WrongPass:
		m.WrongPass++
	default:
		m.WrongFail++
	}
}

func (r CalCard) line() string {
	if r.Err != "" {
		return fmt.Sprintf("%-40s %s", r.Card.Key, r.Err)
	}
	grade := func(score int, pass bool, err string) string {
		if err != "" {
			return "ungraded"
		}
		v := "fail"
		if pass {
			v = "PASS"
		}
		return fmt.Sprintf("%d/7 %s", score, v)
	}
	return fmt.Sprintf("%-40s back %-10s wrong %-10s", r.Card.Key, grade(r.BackScore, r.BackPass, r.BackErr), grade(r.WrongScore, r.WrongPass, r.WrongErr))
}

// wrongAnswer has the model write a wrong answer, on the budget.
func (g *Grader) wrongAnswer(ctx context.Context, c pytrack.RecallCard, s *Scheme, vw VariantWriter) (string, error) {
	if vw == nil {
		return "", fmt.Errorf("no writer for wrong answers")
	}
	est := g.P.Cfg.Prices.Cost(usageOf(len(variantInstructions)+len(c.Prompt)+len(s.Reference)+2000, 1500))
	month, release, err := g.P.Reserve(ctx, est)
	if err != nil {
		return "", err
	}
	cctx, cancel := context.WithTimeout(ctx, g.Cfg.SchemeTimeout)
	defer cancel()
	text, u, err := vw.WrongAnswer(cctx, c, s)
	release()
	g.P.RecordSpend(month, g.P.Cfg.Prices.Cost(u), u)
	return text, err
}

// VariantWriter is the scheme writer as a wrong-answer writer, when it is one.
func (g *Grader) VariantWriter() VariantWriter {
	vw, _ := g.Writer.(VariantWriter)
	return vw
}

// usageOf is a pessimistic usage: characters at three a token.
func usageOf(chars, outputTokens int) proof.Usage {
	return proof.Usage{InputTokens: chars / 3, OutputTokens: outputTokens}
}
