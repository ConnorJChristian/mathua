package gradecheck

import (
	"os"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator/all"
	"github.com/chuma-beep/mathua/internal/pytrack"
)

// The suite runs against the committed modules only, so it passes the same
// on every machine whatever its modules-local/ holds. `go run
// ./scripts/gradecheck -local` (make gradecheck) covers the local ones.
func TestMain(m *testing.M) {
	os.Setenv("MATHUA_PUBLIC_MODULES_ONLY", "1")
	os.Exit(m.Run())
}

func load(t *testing.T) *Checker {
	t.Helper()
	c, err := Load("../../data/concepts", all.Registry())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return c
}

// Every question-bank item's own answer grades correct through the server's
// path, its plausible spellings too, and its wrong answers grade wrong.
func TestBankKeysGrade(t *testing.T) {
	c := load(t)
	rep := c.CheckBank(Options{Variants: true, Wrong: true})
	if rep.Items < 400 {
		t.Fatalf("only %d items checked; the committed banks hold more, so the check read the wrong thing", rep.Items)
	}
	for _, f := range rep.Defects() {
		t.Error(f.String())
	}
	t.Logf("bank: %d items, %d answers submitted, %d code tasks skipped", rep.Items, rep.Checks, rep.Skipped)
}

// Every generator's problem grades its own answer correct, as clicked or as
// typed, refuses an empty answer and a nearby wrong one, and the shared
// graders take the equivalent spellings. A custom grader's own format is
// reported, not failed.
func TestGeneratorKeysGrade(t *testing.T) {
	if testing.Short() {
		t.Skip("samples every generator against SymPy")
	}
	c := load(t)
	rep := c.CheckGenerators(Options{Variants: true, Wrong: true, Seeds: 3})
	if rep.Problems < 1000 {
		t.Fatalf("only %d problems generated", rep.Problems)
	}
	for _, f := range rep.Defects() {
		t.Error(f.String())
	}
	t.Logf("generators: %d problems, %d answers submitted, %d notes", rep.Problems, rep.Checks, len(rep.Failures)-len(rep.Defects()))
}

// Code tasks: the reference solution passes and the starter does not. Slow
// (one interpreter per task), so only with MATHUA_GRADECHECK_CODE=1.
func TestCodeTasksGrade(t *testing.T) {
	if os.Getenv("MATHUA_GRADECHECK_CODE") != "1" {
		t.Skip("set MATHUA_GRADECHECK_CODE=1 to run every code task")
	}
	c := load(t)
	if err := c.RunnerReady(); err != nil {
		t.Skip("python runner: " + err.Error())
	}
	rep := c.CheckBank(Options{Code: true, Wrong: true})
	for _, f := range rep.Defects() {
		t.Error(f.String())
	}
}

// The variant and wrong-answer generators themselves: what they produce for
// a few keys, so a change to them is visible.
func TestVariantsAndWrongs(t *testing.T) {
	v := variants(part("1/2", "numeric"))
	want(t, v, "0.500", "+1/2")
	v = variants(part("0.5", "numeric"))
	want(t, v, "0.50", ".5", "1/2", "+0.5")
	v = variants(part("3*x^2 + 2*x", "expression"))
	want(t, v, "3*x^{2} + 2*x", "3\\cdot x^2 + 2\\cdot x", "3*x² + 2*x", "3*x**2 + 2*x")
	v = variants(part("exp(x)", "expression"))
	want(t, v, "e^(x)")
	v = variants(part("{1, 2, 3}", "set"))
	want(t, v, "{3, 2, 1}", "\\{1, 2, 3\\}")
	v = variants(part("(2, ∞)", "interval"))
	want(t, v, "(2, oo)", "(2, \\infty)")
	w := wrongs(part("1/2", "numeric"))
	want(t, w, "1.5", "-0.5", "5")
	w = wrongs(part("increasing", "expression"))
	want(t, w, "increasing zz", "not increasing")
	w = wrongs(part("yes", "expression"))
	want(t, w, "no")
	if len(wrongs(part("[2, 5)", "interval"))) < 2 {
		t.Error("an interval needs wrong answers")
	}
}

func part(answer, grading string) pytrack.Part { return pytrack.Part{Answer: answer, Grading: grading} }

func want(t *testing.T, got []string, each ...string) {
	t.Helper()
	for _, e := range each {
		if !contains(got, e) {
			t.Errorf("missing %q in %q", e, got)
		}
	}
}
