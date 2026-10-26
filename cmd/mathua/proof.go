package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/chuma-beep/mathua/internal/proof"
	"github.com/chuma-beep/mathua/internal/pytrack"
	"github.com/chuma-beep/mathua/internal/recallproof"
)

const proofUsage = `usage:
  mathua proof list                       the proof problems this install serves
  mathua proof spend                      this month's judge spend against the budget
  mathua proof calibrate [-runs N] FILE   grade a CSV of problem_id,proof_text,human_score
  mathua proof calibrate -synthetic       grade the bank's own synthetic proofs
  mathua proof samples FILE               write the bank's synthetic proofs as a CSV
  mathua proof calibrate -recall [-n N] [-seed S] [-bank B] [-runs N]
                                          grade N written-answer flashcards' own backs
                                          (should pass) and wrong answers (should fail)
  mathua proof classify [-bank B] [-sample N] [-seed S] [-written|-self]
                                          which recall cards are written-answer cards
  mathua proof schemes [-bank B] [-concept C] [-limit N] [-dry-run] [-yes] [-parallel N]
                                          write the marking schemes of written-answer
                                          flashcards ahead of time, after showing the cost`

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
			kind := p.Kind
			if kind == "" {
				kind = proof.KindProof
			}
			fmt.Printf("%-30s %-28s %-7s L%d %d samples  %s\n", id, p.Concept, kind, p.Level, len(p.Samples), p.Title)
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
	case "spend", "calibrate", "schemes":
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
	if args[0] == "schemes" {
		return schemesCommand(g, args[1:])
	}
	if args[0] == "calibrate" && hasFlag(args[1:], "recall") {
		return calibrateRecallCommand(g, args[1:])
	}
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

// hasFlag reports whether -name (or --name, or -name=...) is among args.
func hasFlag(args []string, name string) bool {
	for _, a := range args {
		a = strings.TrimLeft(a, "-")
		if a == name || strings.HasPrefix(a, name+"=") {
			return true
		}
	}
	return false
}

// writtenCards lists the written-answer flashcards a command works on: those
// in an allowed bank (or in -bank, which overrides the exclusions), under the
// -concept prefix.
func writtenCards(rg *recallproof.Grader, bank, concept string) ([]pytrack.RecallCard, error) {
	t, err := pytrack.Default()
	if err != nil {
		return nil, err
	}
	var out []pytrack.RecallCard
	for _, c := range t.Bank.Cards() {
		if !c.Written {
			continue
		}
		if bank != "" {
			if !recallproof.SameBank(bank, c.Bank) {
				continue
			}
		} else if !rg.Cfg.BankAllowed(c.Bank) {
			continue
		}
		if concept != "" && !strings.HasPrefix(c.Concept, concept) {
			continue
		}
		out = append(out, c.RecallCard)
	}
	return out, nil
}

// schemesCommand writes marking schemes ahead of time, so the first answer to
// each card is graded without waiting for its scheme. It shows the estimated
// cost first and asks before spending.
func schemesCommand(g *proof.Grader, args []string) error {
	fs := flag.NewFlagSet("schemes", flag.ContinueOnError)
	bank := fs.String("bank", "", "only this bank (\"linear_algebra\" or \"anki_linear_algebra\"); overrides MATHUA_LLM_RECALL_EXCLUDE_BANKS")
	concept := fs.String("concept", "", "only cards whose concept id starts with this")
	limit := fs.Int("limit", 0, "write at most this many schemes (0: all)")
	dry := fs.Bool("dry-run", false, "show what would be written and the cost, write nothing")
	yes := fs.Bool("yes", false, "do not ask before spending")
	parallel := fs.Int("parallel", 4, "schemes written at once")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rg, err := recallproof.Default()
	if err != nil {
		return err
	}
	cards, err := writtenCards(rg, *bank, *concept)
	if err != nil {
		return err
	}
	ctx := context.Background()
	var todo []pytrack.RecallCard
	have, failed := 0, 0
	var typical, worst float64
	for _, c := range cards {
		if rg.HasScheme(ctx, c) {
			have++
			continue
		}
		if _, perm := rg.Failure(ctx, c); perm {
			failed++
			continue
		}
		if *limit > 0 && len(todo) >= *limit {
			continue
		}
		todo = append(todo, c)
		typical += rg.TypicalScheme(c)
		worst += rg.EstimateScheme(c)
	}
	month := time.Now().UTC().Format("2006-01")
	spent, _ := g.Store.ProofSpend(ctx, month)
	fmt.Printf("written-answer cards: %d (%d with a scheme, %d judged ungradable)\n", len(cards), have, failed)
	fmt.Printf("to write: %d schemes with %s\n", len(todo), g.Cfg.Model)
	fmt.Printf("estimated cost: about $%.2f (at most $%.2f)\n", typical, worst)
	fmt.Printf("budget: $%.2f of $%.2f spent in %s\n", spent, g.Cfg.MonthlyUSD, month)
	if len(todo) > 0 {
		fmt.Printf("then grading costs about $%.3f per answer (at most $%.3f)\n", rg.TypicalGrade(todo[0]), rg.EstimateGrade(todo[0]))
	}
	if *dry || len(todo) == 0 {
		return nil
	}
	if g.Judge == nil || rg.Writer == nil {
		return errors.New("ANTHROPIC_API_KEY is not set: no scheme can be written")
	}
	if !*yes {
		fmt.Print("write them? [y/N] ")
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if a := strings.ToLower(strings.TrimSpace(line)); a != "y" && a != "yes" {
			return errors.New("nothing written")
		}
	}
	if *parallel < 1 {
		*parallel = 1
	}
	var mu sync.Mutex
	written, errs := 0, 0
	stop := false
	jobs := make(chan pytrack.RecallCard)
	var wg sync.WaitGroup
	for i := 0; i < *parallel; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for c := range jobs {
				cctx, cancel := context.WithTimeout(ctx, rg.Cfg.SchemeTimeout+10*time.Second)
				_, err := rg.Scheme(cctx, c)
				cancel()
				mu.Lock()
				if err != nil {
					errs++
					fmt.Printf("%-40s failed: %v\n", c.Key, err)
					if strings.Contains(err.Error(), "budget") {
						stop = true
					}
				} else {
					written++
					fmt.Printf("%-40s written (%d/%d)\n", c.Key, written, len(todo))
				}
				mu.Unlock()
			}
		}()
	}
	for _, c := range todo {
		mu.Lock()
		halt := stop
		mu.Unlock()
		if halt {
			fmt.Println("stopped: this month's budget is used up")
			break
		}
		jobs <- c
	}
	close(jobs)
	wg.Wait()
	after, _ := g.Store.ProofSpend(ctx, month)
	fmt.Printf("\nwritten %d, failed %d, cost $%.4f (budget $%.2f of $%.2f)\n", written, errs, after-spent, after, g.Cfg.MonthlyUSD)
	return nil
}

// calibrateRecallCommand samples written-answer flashcards and grades each
// one's back (should pass) and a judge-written wrong answer (should fail).
func calibrateRecallCommand(g *proof.Grader, args []string) error {
	fs := flag.NewFlagSet("calibrate -recall", flag.ContinueOnError)
	fs.Bool("recall", true, "calibrate on written-answer flashcards")
	n := fs.Int("n", 20, "cards to sample")
	seed := fs.Int64("seed", 1, "random seed of the sample")
	bank := fs.String("bank", "", "only this bank")
	runs := fs.Int("runs", g.Cfg.Runs, "judge runs per answer")
	if err := fs.Parse(args); err != nil {
		return err
	}
	g.Cfg.Runs = *runs
	rg, err := recallproof.Default()
	if err != nil {
		return err
	}
	if !rg.On() {
		return errors.New("written-answer grading is off (ANTHROPIC_API_KEY unset or MATHUA_LLM_RECALL=0)")
	}
	cards, err := writtenCards(rg, *bank, "")
	if err != nil {
		return err
	}
	r := rand.New(rand.NewSource(*seed))
	r.Shuffle(len(cards), func(i, j int) { cards[i], cards[j] = cards[j], cards[i] })
	if *n < len(cards) {
		cards = cards[:*n]
	}
	if len(cards) == 0 {
		return errors.New("no written-answer cards to calibrate on")
	}
	// Each card: a scheme (if it has none), a wrong answer and two gradings.
	var est float64
	for _, c := range cards {
		est += rg.TypicalGrade(c)*2 + rg.TypicalScheme(c)*2
	}
	fmt.Printf("calibrating on %d written-answer cards with %s, %d runs each; about $%.2f\n", len(cards), g.Cfg.Model, g.Cfg.Runs, est)
	_, m := rg.Calibrate(context.Background(), cards, rg.VariantWriter(), os.Stdout)
	fmt.Println()
	fmt.Println(m)
	return nil
}
