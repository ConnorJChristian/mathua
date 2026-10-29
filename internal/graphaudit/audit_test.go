package graphaudit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/modules"
)

func c(id, label, domain, module string, prereqs ...string) (concepts.Concept, string) {
	return concepts.Concept{ID: id, Label: label, Domain: domain, Subdomain: domain + ".x", Prerequisites: prereqs,
		MasteryThreshold: concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 30}}, module
}

func graph(t *testing.T, items ...func() (concepts.Concept, string)) Graph {
	t.Helper()
	g := Graph{ModuleOf: map[string]string{}}
	for _, it := range items {
		cc, m := it()
		g.Concepts = append(g.Concepts, cc)
		g.ModuleOf[cc.ID] = m
	}
	return g
}

func TestAudit_HardFindings(t *testing.T) {
	g := graph(t,
		func() (concepts.Concept, string) { return c("a", "A", "d", "", "b") },
		func() (concepts.Concept, string) { return c("b", "B", "d", "", "a") },
		func() (concepts.Concept, string) { return c("x", "X", "d", "m", "missing") },
	)
	g.Concepts[2].Subdomain = ""
	g.Concepts[2].KeyPrerequisites = []string{"a"}
	g.DomainLabel = func(string) bool { return false }
	r := Audit(g, Options{})
	if len(r.Cycles) != 1 || strings.Join(r.Cycles[0], ",") != "a,b" {
		t.Errorf("cycles: %v", r.Cycles)
	}
	if len(r.Dangling) != 1 || r.Dangling[0] != "x requires missing" {
		t.Errorf("dangling: %v", r.Dangling)
	}
	if len(r.MissingSubdomain) != 1 || r.MissingSubdomain[0] != "x" {
		t.Errorf("subdomain: %v", r.MissingSubdomain)
	}
	if len(r.MissingDomainLabel) != 1 {
		t.Errorf("domain label: %v", r.MissingDomainLabel)
	}
	if len(r.BadKeyPrereqs) != 1 || !strings.Contains(r.BadKeyPrereqs[0], "not a prerequisite") {
		t.Errorf("key prerequisites: %v", r.BadKeyPrereqs)
	}
	if len(r.HardErrors()) != 5 {
		t.Errorf("hard errors: %v", r.HardErrors())
	}
}

func TestAudit_IsolatedOwnModuleAndSuggestions(t *testing.T) {
	g := graph(t,
		func() (concepts.Concept, string) { return c("calc.partial", "Partial derivatives", "calculus", "") },
		func() (concepts.Concept, string) {
			return c("calc.chain", "The chain rule", "calculus", "", "calc.partial")
		},
		func() (concepts.Concept, string) { return c("hoy.intro", "Reading the book", "econ", "hoy") },
		func() (concepts.Concept, string) {
			return c("hoy.partial", "Partial derivatives in economics", "econ", "hoy", "hoy.intro")
		},
		func() (concepts.Concept, string) { return c("hoy.alone", "An island", "econ", "hoy") },
		func() (concepts.Concept, string) {
			return c("anki.partial", "Practice · Partial derivatives", "calculus", "anki", "calc.partial")
		},
	)
	g.NotAPrerequisite = func(cc *concepts.Concept) bool { return strings.HasPrefix(cc.Label, "Practice") }
	g.Listed = map[string]bool{"hoy.partial": true}
	g.Read = map[string]bool{}
	r := Audit(g, Options{})
	if len(r.HardErrors()) != 0 {
		t.Fatalf("hard: %v", r.HardErrors())
	}
	if len(r.Isolated) != 1 || r.Isolated[0] != "hoy.alone" {
		t.Errorf("isolated: %v", r.Isolated)
	}
	if got := r.OwnModuleOnly["hoy"]; len(got) != 1 || got[0] != "hoy.partial" {
		t.Errorf("own-module-only: %v", r.OwnModuleOnly)
	}
	if r.CrossModuleLinks["anki"] != 1 || r.CrossModuleLinks["hoy"] != 0 {
		t.Errorf("cross links: %v", r.CrossModuleLinks)
	}
	if got := r.Uncovered["hoy"]; len(got) != 2 {
		t.Errorf("uncovered: %v", r.Uncovered)
	}
	// hoy.partial should be pointed at the corpus's partial derivatives, not
	// at the practice set, and the corpus gets no suggestions.
	var found bool
	for _, s := range r.Suggestions {
		if s.Topic == "hoy.partial" && s.Candidate == "calc.partial" {
			found = true
			if !s.OnlyOwnModule {
				t.Errorf("hoy.partial's prerequisites stay in its module: %+v", s)
			}
		}
		if s.Candidate == "anki.partial" {
			t.Errorf("a practice set must not be suggested as a prerequisite: %+v", s)
		}
		if s.Module == "" {
			t.Errorf("the corpus gets no suggestions: %+v", s)
		}
	}
	if !found {
		t.Errorf("expected hoy.partial <- calc.partial, got %+v", r.Suggestions)
	}
	// A look-alike across modules is reported once, as a duplicate.
	if len(r.Duplicates) == 0 {
		t.Errorf("anki.partial and calc.partial are look-alikes: %+v", r.Duplicates)
	}
}

func TestAudit_SuggestionNeverCycles(t *testing.T) {
	// y (module m) depends on x (corpus); x's label matches y's. A suggestion
	// y <- x is already a link; x <- y would cycle. Neither is suggested.
	g := graph(t,
		func() (concepts.Concept, string) { return c("x", "Lagrange multipliers", "calculus", "") },
		func() (concepts.Concept, string) {
			return c("y", "Lagrange multipliers in economics", "econ", "m", "x")
		},
		func() (concepts.Concept, string) {
			return c("z", "Lagrange multipliers with two constraints", "econ", "m2", "y")
		},
	)
	r := Audit(g, Options{})
	for _, s := range r.Suggestions {
		if s.Topic == "y" && s.Candidate == "x" || s.Topic == "z" {
			t.Errorf("suggested an existing ancestor: %+v", s)
		}
	}
}

func TestAudit_ChainsThatWander(t *testing.T) {
	g := graph(t,
		func() (concepts.Concept, string) { return c("a", "A", "arith", "") },
		func() (concepts.Concept, string) { return c("b", "B", "calc", "", "a") },
		func() (concepts.Concept, string) { return c("c", "C", "econ", "m", "b") },
		func() (concepts.Concept, string) { return c("d", "D", "calc", "m", "c") },
		func() (concepts.Concept, string) { return c("e", "E", "sql", "w") },
		func() (concepts.Concept, string) { return c("f", "F", "econ", "m", "e", "d") },
	)
	tracks := map[string]string{"arith": "math", "calc": "math", "econ": "econ", "sql": "workday"}
	g.TrackOf = func(d string) string { return tracks[d] }
	r := Audit(g, Options{})
	if r.MaxDepth.ID != "f" || r.MaxDepth.Depth != 4 {
		t.Errorf("longest chain: %+v", r.MaxDepth)
	}
	ids := map[string]Chain{}
	for _, ch := range r.DeepChains {
		ids[ch.ID] = ch
	}
	if ch, ok := ids["d"]; !ok || ch.Revisits != 1 {
		t.Errorf("d's chain leaves calculus and comes back: %+v", r.DeepChains)
	}
	if _, ok := ids["b"]; ok {
		t.Errorf("b descends straight: %+v", ids["b"])
	}
	// f's longest chain is through d (math and econ: 2 tracks) and revisits
	// calc; its other prerequisite e is on a third track but not on the
	// longest chain. The chain is reported for the revisit.
	if _, ok := ids["f"]; !ok {
		t.Errorf("f: %+v", r.DeepChains)
	}
}

func TestTokenize(t *testing.T) {
	got := strings.Join(tokenize("Practice · Derivatives of exponential and logarithmic functions (1 of 5)"), " ")
	if got != "derivative exponential logarithm function" {
		t.Errorf("tokens: %q", got)
	}
}

// The public graph (the corpus and the committed modules) must be clean: no
// cycles, dangling or bad key prerequisites, no missing subdomain or label,
// no topic cut off from everything. The softer findings are logged.
func TestPublicGraph(t *testing.T) {
	root := filepath.Join("..", "..")
	set, err := modules.LoadPublic(root)
	if err != nil {
		t.Fatal(err)
	}
	g, err := LoadInstall(filepath.Join(root, "data", "concepts"), set)
	if err != nil {
		t.Fatal(err)
	}
	g.NotAPrerequisite = func(cc *concepts.Concept) bool { return strings.HasPrefix(cc.Label, "Practice ·") }
	r := Audit(g, Options{})
	for _, h := range r.HardErrors() {
		t.Error(h)
	}
	if len(r.Isolated) > 0 {
		t.Errorf("isolated concepts: %v", r.Isolated)
	}
	if r.Concepts < 1248 {
		t.Errorf("the corpus alone has 1248 concepts, got %d", r.Concepts)
	}
	if _, ok := r.Modules["python-dsh"]; !ok {
		t.Errorf("python-dsh should be in the public graph: %v", r.Modules)
	}
	t.Log(r.Text(false))
	if os.Getenv("MATHUA_AUDIT_VERBOSE") == "1" {
		t.Log(r.Text(true))
	}
}
