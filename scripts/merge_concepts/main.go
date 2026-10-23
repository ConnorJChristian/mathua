package main

// Merge the per-domain concept files into the copy the web app imports
// (`web/next-app/data/concepts.json`, typed by `web/next-app/lib/conceptData.ts`).
//
// `data/concepts/enrichment.json` is skipped on purpose. It is an overlay keyed by
// id -- encompasses, key_prerequisites, interference_group, variants -- and the
// loader applies it to the domain nodes rather than treating its entries as nodes,
// so including it would put ten id-only records into a file the browser reads as
// the graph. Before this skip the merge could never have been re-run without
// adding them; the copy in the tree was a snapshot taken before the file existed
// and it had quietly drifted 29 `grading_type` corrections behind the corpus.
import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chuma-beep/mathua/internal/modules"
)

// overlayFiles are id-keyed overlays, not concept lists.
var overlayFiles = map[string]bool{"enrichment.json": true}

// creditToken names the licence family whose attribution has to reach the page. It is read from
// `data/lessons/mapping.json` (the ingest map) and kept only for concepts that have a lesson, because
// a token for a lesson nobody can open would ask a page to credit text it never shows.
//
// ORCCA is the reason the field exists. Its Licensing section requires a fixed sentence on "every
// digital format page view" of its content, which a browser cannot read out of the corpus, so the
// merge carries the token and `components/LessonCredits.tsx` holds the sentence. OpenStax and
// Algebrica state their terms in prose rather than a fixed sentence, and their credit is already the
// lesson body's first line, so they need no token.
var creditTokens = map[string]string{"orcca": "orcca"}

type lessonEntry struct {
	ConceptID string `json:"concept_id"`
	Source    string `json:"source"`
}

type mapEntry struct {
	Source string `json:"source"`
}

func readJSON(path string, v any) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", path, err)
		os.Exit(1)
	}
	if err := json.Unmarshal(data, v); err != nil {
		fmt.Fprintf(os.Stderr, "parse %s: %v\n", path, err)
		os.Exit(1)
	}
}

// credits returns concept id -> credit token, for lessons that exist and whose publisher asks for a
// per-page-view attribution.
func credits() map[string]string {
	var lessons []lessonEntry
	readJSON("data/lessons/lessons.json", &lessons)
	taught := make(map[string]bool, len(lessons))
	for _, l := range lessons {
		taught[l.ConceptID] = true
	}
	var mapping map[string]mapEntry
	readJSON("data/lessons/mapping.json", &mapping)
	out := map[string]string{}
	for id, m := range mapping {
		if token, ok := creditTokens[m.Source]; ok && taught[id] {
			out[id] = token
		}
	}
	return out
}

func main() {
	// By default only committed modules are merged, so the files this writes
	// (which are committed) never carry your modules-local/ content. The
	// Docker build and `make local` pass -local, so the web app you run
	// includes your own modules. MATHUA_INCLUDE_LOCAL_MODULES=1 does the same.
	local := flag.Bool("local", os.Getenv("MATHUA_INCLUDE_LOCAL_MODULES") == "1", "also merge modules-local/")
	flag.Parse()
	public := new(bool)
	*public = !*local

	byConcept := credits()
	all := readConceptDir("data/concepts", byConcept)

	mods, err := modules.Load(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "modules: %v\n", err)
		os.Exit(1)
	}
	type trackOut struct {
		ID      string   `json:"id"`
		Label   string   `json:"label"`
		Domains []string `json:"domains"`
	}
	tracks := []trackOut{}
	labels := map[string]string{}
	used := 0
	for _, m := range mods.Modules {
		if *public && m.Local {
			continue
		}
		used++
		dir := filepath.Join(m.Dir, "concepts")
		if st, err := os.Stat(dir); err == nil && st.IsDir() {
			all = append(all, readConceptDir(dir, nil)...)
		}
		for d, l := range m.DomainLabels {
			labels[d] = l
		}
	}
	// Tracks from the modules that were merged, in the server's order.
	merged := map[string]bool{}
	for _, m := range mods.Modules {
		if !(*public && m.Local) {
			for _, t := range m.Tracks {
				merged[t.ID] = true
			}
		}
	}
	for _, t := range mods.Tracks() {
		if t.ID == modules.MathTrack || merged[t.ID] {
			d := t.Domains
			if d == nil {
				d = []string{}
			}
			tracks = append(tracks, trackOut{ID: t.ID, Label: t.Label, Domains: d})
		}
	}

	out, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile("web/next-app/data/concepts.json", out, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		os.Exit(1)
	}
	tj, _ := json.MarshalIndent(map[string]any{"tracks": tracks, "domain_labels": labels}, "", "  ")
	if err := os.WriteFile("web/next-app/data/tracks.json", append(tj, '\n'), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write tracks: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("OK: %d concepts merged into web/next-app/data/concepts.json (%d modules), %d with a per-page credit; %d tracks\n",
		len(all), used, len(byConcept), len(tracks))
}

func readConceptDir(dir string, byConcept map[string]string) []any {
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read dir: %v\n", err)
		os.Exit(1)
	}
	var all []any
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") || overlayFiles[e.Name()] {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			fmt.Fprintf(os.Stderr, "read %s: %v\n", e.Name(), err)
			os.Exit(1)
		}
		var batch []map[string]any
		if err := json.Unmarshal(data, &batch); err != nil {
			fmt.Fprintf(os.Stderr, "parse %s: %v\n", e.Name(), err)
			os.Exit(1)
		}
		for _, c := range batch {
			if id, ok := c["id"].(string); ok {
				if token, ok := byConcept[id]; ok {
					c["lesson_credit"] = token
				} else {
					delete(c, "lesson_credit")
				}
			}
			all = append(all, c)
		}
	}
	return all
}
