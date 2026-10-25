package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/chuma-beep/mathua/internal/proof"
)

const proofUsage = `usage:
  mathua proof list                       the proof problems this install serves
  mathua proof spend                      this month's judge spend against the budget
  mathua proof calibrate [-runs N] FILE   grade a CSV of problem_id,proof_text,human_score
  mathua proof calibrate -synthetic       grade the bank's own synthetic proofs
  mathua proof samples FILE               write the bank's synthetic proofs as a CSV`

// proofCommand is the proof grader's operator tool. Calibration spends real
// API credit and is counted against the same monthly budget as the app.
func proofCommand(args []string) error {
	if len(args) == 0 {
		return errors.New(proofUsage)
	}
	g, err := proof.Default()
	if err != nil {
		return err
	}
	switch args[0] {
	case "list":
		for _, id := range g.Bank.IDs() {
			p := g.Bank.Problem(id)
			fmt.Printf("%-30s %-28s L%d %d samples  %s\n", id, p.Concept, p.Level, len(p.Samples), p.Title)
		}
		fmt.Printf("%d problems\n", g.Bank.Len())
		return nil
	case "samples":
		if len(args) != 2 {
			return errors.New(proofUsage)
		}
		f, err := os.Create(args[1])
		if err != nil {
			return err
		}
		defer f.Close()
		return proof.WriteCalibrationCSV(f, g.Bank.SampleRows())
	case "spend", "calibrate":
	default:
		return errors.New(proofUsage)
	}
	repo, _, err := openRepo()
	if err != nil {
		return err
	}
	defer repo.Close()
	if st, ok := repo.(proof.Store); ok {
		g.SetStore(st)
	}
	month := time.Now().UTC().Format("2006-01")
	if args[0] == "spend" {
		usd, err := g.Store.ProofSpend(context.Background(), month)
		if err != nil {
			return err
		}
		fmt.Printf("%s: $%.4f of $%.2f (model %s, %d runs per proof)\n", month, usd, g.Cfg.MonthlyUSD, g.Cfg.Model, g.Cfg.Runs)
		return nil
	}
	fs := flag.NewFlagSet("calibrate", flag.ContinueOnError)
	runs := fs.Int("runs", g.Cfg.Runs, "judge runs per proof")
	synthetic := fs.Bool("synthetic", false, "grade the bank's synthetic proofs")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	g.Cfg.Runs = *runs
	var rows []proof.CalRow
	if *synthetic {
		rows = g.Bank.SampleRows()
	} else {
		if fs.NArg() != 1 {
			return errors.New(proofUsage)
		}
		f, err := os.Open(fs.Arg(0))
		if err != nil {
			return err
		}
		rows, err = proof.ReadCalibrationCSV(f)
		f.Close()
		if err != nil {
			return err
		}
	}
	if len(rows) == 0 {
		return errors.New("no rows to calibrate")
	}
	fmt.Printf("calibrating %d proofs with %s, %d runs each\n", len(rows), g.Cfg.Model, g.Cfg.Runs)
	_, stats := proof.Calibrate(context.Background(), g, rows, os.Stdout)
	fmt.Println()
	fmt.Println(stats)
	return nil
}
