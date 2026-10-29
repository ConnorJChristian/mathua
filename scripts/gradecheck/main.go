// Command gradecheck grades every question's own answer key through the
// server's grading path and reports what does not grade the way it should
// (internal/gradecheck).
//
//	go run ./scripts/gradecheck                 # the committed modules and every generator
//	go run ./scripts/gradecheck -local          # your modules-local/ too (or MATHUA_ROOT=.. for another install's)
//	go run ./scripts/gradecheck -local -code    # also run every code task's solution and starter
//	go run ./scripts/gradecheck -module anki -concept calc.deriv.chain_rule
//
// Run it from the repository root with a python3 that has SymPy on PATH, so
// the expression grader finds grading/sympy_service.py. Exit status 1 when
// anything failed. `make gradecheck` runs it over the local modules.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/chuma-beep/mathua/internal/generator/all"
	"github.com/chuma-beep/mathua/internal/gradecheck"
)

func main() {
	local := flag.Bool("local", false, "include modules-local/ (otherwise only the committed modules)")
	code := flag.Bool("code", false, "run code tasks through the Python runner (slow)")
	seeds := flag.Int("seeds", 3, "problems to sample per generator concept (0 skips generators)")
	module := flag.String("module", "", "comma-separated module ids to check")
	concept := flag.String("concept", "", "comma-separated concept ids to check")
	noVariants := flag.Bool("no-variants", false, "skip equivalent spellings of the keys")
	noWrong := flag.Bool("no-wrong", false, "skip the wrong answers")
	latex := flag.String("latex", "", "JSON map of each key's LaTeX to what the typed-LaTeX box submits (web/next-app/scripts/latex-plain.mjs)")
	verbose := flag.Bool("v", false, "log every concept")
	flag.Parse()

	if !*local {
		os.Setenv("MATHUA_PUBLIC_MODULES_ONLY", "1")
	}
	c, err := gradecheck.Load("data/concepts", all.Registry())
	if err != nil {
		fmt.Fprintln(os.Stderr, "load:", err)
		os.Exit(2)
	}
	opts := gradecheck.Options{Code: *code, Variants: !*noVariants, Wrong: !*noWrong, Seeds: *seeds}
	if *module != "" {
		opts.Modules = strings.Split(*module, ",")
	}
	if *concept != "" {
		opts.Concepts = strings.Split(*concept, ",")
	}
	if *latex != "" {
		raw, err := os.ReadFile(*latex)
		if err != nil {
			fmt.Fprintln(os.Stderr, "latex:", err)
			os.Exit(2)
		}
		if err := json.Unmarshal(raw, &opts.Editor); err != nil {
			fmt.Fprintln(os.Stderr, "latex:", err)
			os.Exit(2)
		}
	}
	if *verbose {
		opts.Log = func(f string, a ...any) { fmt.Fprintf(os.Stderr, f+"\n", a...) }
	}
	if *code {
		if err := c.RunnerReady(); err != nil {
			fmt.Fprintln(os.Stderr, "code tasks cannot run here:", err)
			os.Exit(2)
		}
	}
	bank := c.CheckBank(opts)
	var gens gradecheck.Report
	if *seeds > 0 && *module == "" {
		gens = c.CheckGenerators(opts)
	}
	for _, f := range bank.Failures {
		fmt.Println(f.String())
	}
	for _, f := range gens.Failures {
		fmt.Println(f.String())
	}
	fmt.Printf("\nbank: %d items, %d answers submitted, %d failures, %d skipped\n", bank.Items, bank.Checks, len(bank.Failures), bank.Skipped)
	fmt.Printf("generators: %d problems, %d answers submitted, %d failures\n", gens.Problems, gens.Checks, len(gens.Failures))
	if len(bank.Defects())+len(gens.Defects()) > 0 {
		os.Exit(1)
	}
}
