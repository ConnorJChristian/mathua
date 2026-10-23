// validate_graph checks the concept graph the installation runs with: the
// base corpus plus every enabled module (set MATHUA_PUBLIC_MODULES_ONLY=1 to
// leave out modules-local/). Building the DAG rejects cycles, dangling
// prerequisites and duplicate ids; this also rejects two concepts with the
// same label, which the learner could not tell apart.
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/modules"
)

func main() {
	set, err := modules.Default()
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: modules: %v\n", err)
		os.Exit(1)
	}
	dag, err := concepts.LoadDirs("data/concepts", set.ConceptDirs()...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	byLabel := map[string][]string{}
	for id, c := range dag.Concepts() {
		key := strings.ToLower(strings.TrimSpace(c.Label))
		byLabel[key] = append(byLabel[key], id)
	}
	var dups []string
	for label, ids := range byLabel {
		if len(ids) > 1 {
			sort.Strings(ids)
			dups = append(dups, fmt.Sprintf("%q: %s", label, strings.Join(ids, ", ")))
		}
	}
	if len(dups) > 0 {
		sort.Strings(dups)
		fmt.Fprintf(os.Stderr, "FAIL: %d duplicate labels\n  %s\n", len(dups), strings.Join(dups, "\n  "))
		os.Exit(1)
	}
	fmt.Printf("OK: %d concepts across %d domains, %d modules; no cycles, dangling prerequisites or duplicate labels\n",
		dag.Count(), len(dag.Domains()), len(set.Modules))
}
