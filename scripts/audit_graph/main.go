// audit_graph reviews the knowledge graph this installation runs with: the
// base corpus plus every enabled module, local modules included (set
// MATHUA_PUBLIC_MODULES_ONLY=1 to leave them out, MATHUA_ROOT to audit
// another install's modules-local/). It is validate_graph's deeper cousin
// (internal/graphaudit): cycles, dangling prerequisites, isolated topics,
// module topics whose prerequisites never leave their module, look-alike
// concepts, chains through unrelated domains, key_prerequisites, missing
// labels, topics on no course, and a shortlist of cross-module
// prerequisites to add by hand.
//
//	go run ./scripts/audit_graph               the report
//	go run ./scripts/audit_graph -v            with every suggestion
//	go run ./scripts/audit_graph -review r.md  the suggestions as a review list
//	go run ./scripts/audit_graph -json r.json  the whole report
//
// The exit status is 1 on a hard finding (a cycle, a dangling or bad key
// prerequisite, a missing subdomain or domain label), 0 otherwise.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/graphaudit"
	"github.com/chuma-beep/mathua/internal/modules"
)

func main() {
	verbose := flag.Bool("v", false, "print every suggestion and uncovered topic")
	jsonOut := flag.String("json", "", "write the report as JSON to this file")
	review := flag.String("review", "", "write the suggestions as a Markdown review list to this file")
	minScore := flag.Float64("min", 0, "least label similarity for a suggestion (default 0.45)")
	perTopic := flag.Int("per-topic", 0, "most suggestions per topic (default 3)")
	flag.Parse()

	set, err := modules.Default()
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: modules: %v\n", err)
		os.Exit(1)
	}
	base := filepath.Join(set.Root, "data", "concepts")
	if _, err := os.Stat(base); err != nil {
		base = "data/concepts"
	}
	g, err := graphaudit.LoadInstall(base, set)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	// Courses: data/courses.json plus every module's.
	listed := map[string]bool{}
	for _, path := range append([]string{filepath.Join(set.Root, "data", "courses.json")}, set.CourseFiles()...) {
		raw, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
			os.Exit(1)
		}
		var courses []struct {
			Plan    []string `json:"plan"`
			Targets []string `json:"targets"`
		}
		if err := json.Unmarshal(raw, &courses); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL: %s: %v\n", path, err)
			os.Exit(1)
		}
		for _, c := range courses {
			for _, id := range append(append([]string(nil), c.Plan...), c.Targets...) {
				listed[id] = true
			}
		}
	}
	g.Listed = listed
	g.Read = map[string]bool{}
	for id := range set.Readings() {
		g.Read[id] = true
	}
	g.NotAPrerequisite = func(c *concepts.Concept) bool {
		return strings.HasPrefix(c.Label, "Practice ·")
	}

	r := graphaudit.Audit(g, graphaudit.Options{MinSuggestion: *minScore, MaxSuggestionsPerTopic: *perTopic})
	fmt.Print(r.Text(*verbose))
	if *jsonOut != "" {
		data, _ := json.MarshalIndent(r, "", "  ")
		if err := os.WriteFile(*jsonOut, data, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
			os.Exit(1)
		}
	}
	if *review != "" {
		if err := os.WriteFile(*review, []byte(reviewList(r)), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
			os.Exit(1)
		}
	}
	if len(r.HardErrors()) > 0 {
		os.Exit(1)
	}
}

// reviewList renders the suggestions and duplicates as Markdown, grouped by
// module, for a person to go through.
func reviewList(r *graphaudit.Report) string {
	var b strings.Builder
	b.WriteString("# Cross-module prerequisite review\n\n")
	b.WriteString("Candidates found by label similarity, best first. A link is added in the module's `src/` (`_prereqs`), then the bank is rebuilt. Score is an idf-weighted overlap of label words (1 = identical).\n\n")
	byModule := map[string][]graphaudit.Suggestion{}
	for _, s := range r.Suggestions {
		byModule[s.Module] = append(byModule[s.Module], s)
	}
	mods := make([]string, 0, len(byModule))
	for m := range byModule {
		mods = append(mods, m)
	}
	sort.Strings(mods)
	for _, m := range mods {
		fmt.Fprintf(&b, "## %s\n\n| score | topic | candidate prerequisite | shared words | note |\n|---|---|---|---|---|\n", m)
		for _, s := range byModule[m] {
			note := ""
			if s.OnlyOwnModule {
				note = "prerequisites stay in the module"
			}
			if s.SameDomain {
				if note != "" {
					note += "; "
				}
				note += "same domain"
			}
			fmt.Fprintf(&b, "| %.2f | `%s` %s | `%s` (%s) %s | %s | %s |\n", s.Score, s.Topic, s.TopicLabel,
				s.Candidate, orCorpus(s.CandModule), s.CandLabel, strings.Join(s.Shared, ", "), note)
		}
		b.WriteString("\n")
	}
	if len(r.Duplicates) > 0 {
		b.WriteString("## Look-alike concepts\n\nTwo ids for what may be one skill. Tie them with `encompasses` (the broader one encompasses the narrower) or make one the other's prerequisite; do not delete.\n\n| score | a | b | already linked |\n|---|---|---|---|\n")
		for _, d := range r.Duplicates {
			fmt.Fprintf(&b, "| %.2f | `%s` (%s) %s | `%s` (%s) %s | %v |\n", d.Score, d.A, orCorpus(d.ModuleA), d.LabelA, d.B, orCorpus(d.ModuleB), d.LabelB, d.Linked)
		}
	}
	return b.String()
}

func orCorpus(m string) string {
	if m == "" {
		return "corpus"
	}
	return m
}
