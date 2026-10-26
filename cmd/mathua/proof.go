package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/chuma-beep/mathua/internal/proof"
	"github.com/chuma-beep/mathua/internal/pytrack"
)

const proofUsage = `usage:
  mathua proof list                       the proof problems this install serves
  mathua proof spend                      this month's judge spend against the budget
  mathua proof calibrate [-runs N] FILE   grade a CSV of problem_id,proof_text,human_score
  mathua proof calibrate -synthetic       grade the bank's own synthetic proofs
  mathua proof samples FILE               write the bank's synthetic proofs as a CSV
  mathua proof classify [-bank B] [-sample N] [-seed S] [-written|-self]
                                          which recall cards are written-answer cards`

// proofCommand is the proof grader's operator tool. Calibration spends real
// API credit and is counted against the same monthly budget as the app.
func proofCommand(args []string) error {
	if len(args) == 0 {
		return errors.New(proofUsage)
	}
	if args[0] == "classify" {
		return classifyCommand(args[1:])
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

// classifyCommand reports, per bank, how many recall cards are written-answer
// cards, and prints a random sample with each one's classification.
func classifyCommand(args []string) error {
	fs := flag.NewFlagSet("classify", flag.ContinueOnError)
	bank := fs.String("bank", "", "only this bank (file name without .json)")
	sample := fs.Int("sample", 30, "how many cards to print")
	seed := fs.Int64("seed", 1, "random seed of the sample")
	onlyWritten := fs.Bool("written", false, "sample written-answer cards only")
	onlySelf := fs.Bool("self", false, "sample self-graded cards only")
	if err := fs.Parse(args); err != nil {
		return err
	}
	t, err := pytrack.Default()
	if err != nil {
		return err
	}
	type count struct{ recall, written int }
	counts := map[string]*count{}
	var banks []string
	var pool []pytrack.ClassifiedCard
	reasons := map[string]int{}
	for _, c := range t.Bank.Cards() {
		if *bank != "" && c.Bank != *bank {
			continue
		}
		n := counts[c.Bank]
		if n == nil {
			n = &count{}
			counts[c.Bank] = n
			banks = append(banks, c.Bank)
		}
		n.recall++
		if c.Written {
			n.written++
			reasons[c.Reason]++
		}
		if (*onlyWritten && !c.Written) || (*onlySelf && c.Written) {
			continue
		}
		pool = append(pool, c)
	}
	sort.Strings(banks)
	var tr, tw int
	fmt.Printf("%-34s %7s %8s\n", "bank", "recall", "written")
	for _, b := range banks {
		fmt.Printf("%-34s %7d %8d\n", b, counts[b].recall, counts[b].written)
		tr += counts[b].recall
		tw += counts[b].written
	}
	fmt.Printf("%-34s %7d %8d\n\nwritten-answer cards by keyword:", "total", tr, tw)
	var ks []string
	for k := range reasons {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool { return reasons[ks[i]] > reasons[ks[j]] })
	for _, k := range ks {
		fmt.Printf(" %s %d;", k, reasons[k])
	}
	fmt.Println()
	r := rand.New(rand.NewSource(*seed))
	r.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	if *sample > len(pool) {
		*sample = len(pool)
	}
	for _, c := range pool[:*sample] {
		verdict := "self-graded"
		if c.Written {
			verdict = "WRITTEN"
		}
		reason := c.Reason
		if reason == "" {
			reason = "no keyword"
		}
		fmt.Printf("\n[%s] %s (%s)\n  %s\n", verdict, c.Key, reason, shortLine(c.Prompt, 220))
	}
	return nil
}

// shortLine shortens text to one line of at most n runes.
func shortLine(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}
