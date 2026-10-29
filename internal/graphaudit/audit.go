// Package graphaudit reviews the knowledge graph an installation runs with:
// the base corpus plus every enabled module's concepts, as one graph (fork
// change).
//
// Building the DAG (internal/concepts) already refuses a cycle, a dangling
// prerequisite or a duplicate id. The audit goes further and reports what a
// valid graph can still get wrong when several books and courses share it:
//
//   - a concept nothing leads to and nothing follows (isolated);
//   - a module topic whose prerequisites never leave its own module while the
//     corpus or another module teaches the same ground (the links a learner
//     struggling on the topic would be sent back along);
//   - two concepts that look like the same skill under two ids (a duplicate
//     to tie with `encompasses`, not to delete);
//   - prerequisite chains that wander through unrelated domains;
//   - key_prerequisites that cannot steer remediation, and labels or
//     subdomains that are missing;
//   - module topics no course lists and no reading covers.
//
// The cross-module suggestions are a shortlist for a person: the score is
// a label similarity, and a link is added by hand in the module's source.
package graphaudit

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/modules"
)

// Graph is the audited corpus: every concept with the module it came from.
// ModuleOf maps a concept id to its module id; the base corpus is "".
type Graph struct {
	Concepts []concepts.Concept
	ModuleOf map[string]string
	// DomainLabel reports whether a domain has a display label: upstream
	// domains are labelled by the web app, module domains by module.json.
	// Nil skips the check.
	DomainLabel func(domain string) bool
	// Listed is every concept id some course lists (plan or targets), and
	// Read every id a module's readings.json covers. Both may be nil, which
	// skips the coverage check.
	Listed map[string]bool
	Read   map[string]bool
	// NotAPrerequisite marks concepts never to suggest as a prerequisite
	// (imported practice sets, say). Nil suggests anything.
	NotAPrerequisite func(c *concepts.Concept) bool
	// TrackOf names the track a domain belongs to (internal/modules). Nil
	// puts every domain on one track.
	TrackOf func(domain string) string
}

// Suggestion is a candidate prerequisite for a module topic, found by label
// similarity in the rest of the graph.
type Suggestion struct {
	Topic      string  `json:"topic"`
	TopicLabel string  `json:"topic_label"`
	Module     string  `json:"module"`
	Candidate  string  `json:"candidate"`
	CandLabel  string  `json:"candidate_label"`
	CandModule string  `json:"candidate_module"`
	Score      float64 `json:"score"`
	// Shared are the label words the two have in common.
	Shared []string `json:"shared"`
	// SameDomain is set when both sit in one domain.
	SameDomain bool `json:"same_domain"`
	// OnlyOwnModule is set when the topic's prerequisites never leave its
	// module: the case the audit is most interested in.
	OnlyOwnModule bool `json:"only_own_module"`
}

// Duplicate is a pair of concepts whose labels nearly coincide.
type Duplicate struct {
	A, B           string
	LabelA, LabelB string
	ModuleA        string
	ModuleB        string
	Score          float64
	// Linked is set when one already depends on or encompasses the other.
	Linked bool
}

// Chain is a concept's longest prerequisite chain.
type Chain struct {
	ID      string
	Depth   int
	Domains []string // the domains along the chain, root first, runs collapsed
	Tracks  []string // the distinct tracks along the chain
	Path    []string // the chain itself, root first
	// Revisits counts domains the chain leaves and comes back to: a chain
	// that wanders (calculus > algebra > calculus) rather than descends.
	Revisits int
}

// Report is the result of Audit.
type Report struct {
	Concepts int
	Modules  map[string]int // concept count per module ("" is the corpus)

	// Hard findings: the graph should not ship with these.
	Cycles             [][]string // each a cycle of ids
	Dangling           []string   // "<id> requires <missing>"
	MissingSubdomain   []string
	MissingDomainLabel []string // domains
	BadKeyPrereqs      []string // "<id>: key prerequisite <kp> ..."

	// Soft findings: for review.
	Isolated         []string            // no prerequisites and no dependents
	Roots            map[string][]string // per module: concepts with no prerequisites
	OwnModuleOnly    map[string][]string // per module: topics whose prerequisites never leave the module
	ManyPrereqsNoKey []string            // concepts with 4+ prerequisites and no key_prerequisites
	Duplicates       []Duplicate
	DeepChains       []Chain // chains crossing many domains
	MaxDepth         Chain
	Uncovered        map[string][]string // per module: not on any course, no reading
	Suggestions      []Suggestion
	CrossModuleLinks map[string]int // per module: prerequisite links into another module or the corpus
	// ChainTracks is the threshold DeepChains was reported at.
	ChainTracks int
}

// HardErrors lists the findings a graph must not ship with.
func (r *Report) HardErrors() []string {
	var out []string
	for _, c := range r.Cycles {
		out = append(out, "cycle: "+strings.Join(c, " -> "))
	}
	for _, d := range r.Dangling {
		out = append(out, "dangling: "+d)
	}
	for _, id := range r.MissingSubdomain {
		out = append(out, "no subdomain: "+id)
	}
	for _, d := range r.MissingDomainLabel {
		out = append(out, "no domain label: "+d)
	}
	for _, k := range r.BadKeyPrereqs {
		out = append(out, "key prerequisite: "+k)
	}
	return out
}

// Options tune the soft checks.
type Options struct {
	// MinSuggestion is the least label similarity a suggestion needs
	// (0..1). Zero means the default.
	MinSuggestion float64
	// MaxSuggestionsPerTopic caps the shortlist for one topic. Zero means 3.
	MaxSuggestionsPerTopic int
	// DuplicateScore is the label similarity that reports a duplicate.
	// Zero means the default.
	DuplicateScore float64
	// ChainTracks is the number of distinct tracks along a longest chain
	// from which the chain is reported; a chain that revisits a domain it
	// left is reported whatever its tracks. Zero means 3.
	ChainTracks int
}

const (
	defaultMinSuggestion  = 0.45
	defaultDuplicateScore = 0.85
	defaultChainTracks    = 3
)

// Audit reviews the graph. It never fails: a cycle or a dangling
// prerequisite is a finding, so the script can show every problem at once.
func Audit(g Graph, opt Options) *Report {
	if opt.MinSuggestion == 0 {
		opt.MinSuggestion = defaultMinSuggestion
	}
	if opt.MaxSuggestionsPerTopic == 0 {
		opt.MaxSuggestionsPerTopic = 3
	}
	if opt.DuplicateScore == 0 {
		opt.DuplicateScore = defaultDuplicateScore
	}
	if opt.ChainTracks == 0 {
		opt.ChainTracks = defaultChainTracks
	}
	track := func(domain string) string {
		if g.TrackOf == nil {
			return ""
		}
		return g.TrackOf(domain)
	}
	r := &Report{
		ChainTracks:      opt.ChainTracks,
		Modules:          map[string]int{},
		Roots:            map[string][]string{},
		OwnModuleOnly:    map[string][]string{},
		Uncovered:        map[string][]string{},
		CrossModuleLinks: map[string]int{},
	}
	byID := make(map[string]*concepts.Concept, len(g.Concepts))
	ids := make([]string, 0, len(g.Concepts))
	for i := range g.Concepts {
		c := &g.Concepts[i]
		byID[c.ID] = c
		ids = append(ids, c.ID)
	}
	sort.Strings(ids)
	r.Concepts = len(ids)
	mod := func(id string) string { return g.ModuleOf[id] }
	for _, id := range ids {
		r.Modules[mod(id)]++
	}

	// Dangling, and the edge lists over the ids that exist.
	prereqs := map[string][]string{}
	dependents := map[string][]string{}
	for _, id := range ids {
		c := byID[id]
		for _, p := range c.Prerequisites {
			if byID[p] == nil {
				r.Dangling = append(r.Dangling, fmt.Sprintf("%s requires %s", id, p))
				continue
			}
			prereqs[id] = append(prereqs[id], p)
			dependents[p] = append(dependents[p], id)
		}
		for _, e := range c.Encompasses {
			if byID[e] == nil {
				r.Dangling = append(r.Dangling, fmt.Sprintf("%s encompasses %s", id, e))
			}
		}
		if c.Subdomain == "" {
			r.MissingSubdomain = append(r.MissingSubdomain, id)
		}
		kp := map[string]bool{}
		for _, k := range c.KeyPrerequisites {
			if kp[k] {
				r.BadKeyPrereqs = append(r.BadKeyPrereqs, fmt.Sprintf("%s: key prerequisite %s is listed twice", id, k))
			}
			kp[k] = true
			if byID[k] == nil {
				r.BadKeyPrereqs = append(r.BadKeyPrereqs, fmt.Sprintf("%s: key prerequisite %s does not exist", id, k))
				continue
			}
			found := false
			for _, p := range c.Prerequisites {
				if p == k {
					found = true
				}
			}
			if !found {
				r.BadKeyPrereqs = append(r.BadKeyPrereqs, fmt.Sprintf("%s: key prerequisite %s is not a prerequisite", id, k))
			}
		}
		if len(c.Prerequisites) >= 4 && len(c.KeyPrerequisites) == 0 {
			r.ManyPrereqsNoKey = append(r.ManyPrereqsNoKey, id)
		}
	}
	r.Cycles = cycles(ids, prereqs)
	inCycle := map[string]bool{}
	for _, cyc := range r.Cycles {
		for _, id := range cyc {
			inCycle[id] = true
		}
	}

	// Domain labels.
	if g.DomainLabel != nil {
		seen := map[string]bool{}
		for _, id := range ids {
			d := byID[id].Domain
			if seen[d] {
				continue
			}
			seen[d] = true
			if !g.DomainLabel(d) {
				r.MissingDomainLabel = append(r.MissingDomainLabel, d)
			}
		}
		sort.Strings(r.MissingDomainLabel)
	}

	// Isolated, roots, own-module-only, cross-module link counts, coverage.
	for _, id := range ids {
		c := byID[id]
		m := mod(id)
		if len(prereqs[id]) == 0 && len(dependents[id]) == 0 {
			r.Isolated = append(r.Isolated, id)
		}
		if len(c.Prerequisites) == 0 {
			r.Roots[m] = append(r.Roots[m], id)
		}
		if m != "" {
			cross := 0
			for _, p := range prereqs[id] {
				if mod(p) != m {
					cross++
				}
			}
			r.CrossModuleLinks[m] += cross
			if len(prereqs[id]) > 0 && cross == 0 {
				r.OwnModuleOnly[m] = append(r.OwnModuleOnly[m], id)
			}
			if g.Listed != nil && g.Read != nil && !g.Listed[id] && !g.Read[id] {
				r.Uncovered[m] = append(r.Uncovered[m], id)
			}
		}
	}

	// Ancestors (for "already covered" and cycle-safety) over the acyclic part.
	anc := map[string]map[string]bool{}
	var ancestors func(id string) map[string]bool
	ancestors = func(id string) map[string]bool {
		if a, ok := anc[id]; ok {
			return a
		}
		a := map[string]bool{}
		anc[id] = a // guards a cycle
		for _, p := range prereqs[id] {
			a[p] = true
			for x := range ancestors(p) {
				a[x] = true
			}
		}
		return a
	}
	for _, id := range ids {
		ancestors(id)
	}

	// Longest chains.
	depth := map[string]int{}
	next := map[string]string{} // the prerequisite that continues the longest chain
	var longest func(id string) int
	longest = func(id string) int {
		if d, ok := depth[id]; ok {
			return d
		}
		depth[id] = 0 // guards a cycle
		best, bestID := 0, ""
		for _, p := range prereqs[id] {
			if inCycle[p] {
				continue
			}
			if d := longest(p) + 1; d > best || (d == best && p < bestID) {
				best, bestID = d, p
			}
		}
		depth[id] = best
		next[id] = bestID
		return best
	}
	for _, id := range ids {
		c := Chain{ID: id, Depth: longest(id)}
		var path []string
		for cur := id; cur != ""; cur = next[cur] {
			path = append(path, cur)
		}
		seenDomain := map[string]bool{}
		seenTrack := map[string]bool{}
		for i := len(path) - 1; i >= 0; i-- {
			c.Path = append(c.Path, path[i])
			d := byID[path[i]].Domain
			if len(c.Domains) == 0 || c.Domains[len(c.Domains)-1] != d {
				if seenDomain[d] {
					c.Revisits++
				}
				seenDomain[d] = true
				c.Domains = append(c.Domains, d)
			}
			if t := track(d); !seenTrack[t] {
				seenTrack[t] = true
				c.Tracks = append(c.Tracks, t)
			}
		}
		if c.Depth > r.MaxDepth.Depth {
			r.MaxDepth = c
		}
		if len(c.Tracks) >= opt.ChainTracks || c.Revisits > 0 {
			r.DeepChains = append(r.DeepChains, c)
		}
	}
	sort.Slice(r.DeepChains, func(i, j int) bool {
		a, b := r.DeepChains[i], r.DeepChains[j]
		if len(a.Tracks) != len(b.Tracks) {
			return len(a.Tracks) > len(b.Tracks)
		}
		if a.Revisits != b.Revisits {
			return a.Revisits > b.Revisits
		}
		return a.ID < b.ID
	})

	// Label similarity: duplicates and suggestions.
	idx := newIndex(g.Concepts)
	for i, a := range ids {
		for _, b := range ids[i+1:] {
			// Two ids in one module are that module's business (a deck split
			// into sets, say), and the corpus is upstream's; a look-alike
			// across modules is where two books teach one skill.
			if mod(a) == mod(b) {
				continue
			}
			s := idx.similarity(a, b)
			if s < opt.DuplicateScore {
				continue
			}
			ca, cb := byID[a], byID[b]
			r.Duplicates = append(r.Duplicates, Duplicate{
				A: a, B: b, LabelA: ca.Label, LabelB: cb.Label,
				ModuleA: mod(a), ModuleB: mod(b), Score: s,
				Linked: ancestors(a)[b] || ancestors(b)[a] || encompasses(ca, b) || encompasses(cb, a),
			})
		}
	}
	sort.Slice(r.Duplicates, func(i, j int) bool {
		if r.Duplicates[i].Score != r.Duplicates[j].Score {
			return r.Duplicates[i].Score > r.Duplicates[j].Score
		}
		return r.Duplicates[i].A < r.Duplicates[j].A
	})

	for _, id := range ids {
		m := mod(id)
		if m == "" {
			continue // the corpus is upstream's; modules link into it, not it into them
		}
		c := byID[id]
		own := len(prereqs[id]) > 0
		for _, p := range prereqs[id] {
			if mod(p) != m {
				own = false
			}
		}
		var cands []Suggestion
		for _, other := range ids {
			if other == id || mod(other) == m {
				continue
			}
			oc := byID[other]
			if g.NotAPrerequisite != nil && g.NotAPrerequisite(oc) {
				continue
			}
			if ancestors(id)[other] || ancestors(other)[id] {
				continue // already below it, or above it (a link would cycle)
			}
			s, shared := idx.similarityShared(id, other)
			if s < opt.MinSuggestion {
				continue
			}
			cands = append(cands, Suggestion{
				Topic: id, TopicLabel: c.Label, Module: m,
				Candidate: other, CandLabel: oc.Label, CandModule: mod(other),
				Score: s, Shared: shared, SameDomain: c.Domain == oc.Domain,
				OnlyOwnModule: own,
			})
		}
		sort.Slice(cands, func(i, j int) bool {
			if cands[i].Score != cands[j].Score {
				return cands[i].Score > cands[j].Score
			}
			return cands[i].Candidate < cands[j].Candidate
		})
		if len(cands) > opt.MaxSuggestionsPerTopic {
			cands = cands[:opt.MaxSuggestionsPerTopic]
		}
		r.Suggestions = append(r.Suggestions, cands...)
	}
	return r
}

func encompasses(c *concepts.Concept, id string) bool {
	for _, e := range c.Encompasses {
		if e == id {
			return true
		}
	}
	return false
}

// cycles finds every strongly connected component with more than one
// member, or a self-loop (Tarjan), over the prerequisite edges.
func cycles(ids []string, prereqs map[string][]string) [][]string {
	index := map[string]int{}
	low := map[string]int{}
	onStack := map[string]bool{}
	var stack []string
	var out [][]string
	n := 0
	var visit func(v string)
	visit = func(v string) {
		index[v] = n
		low[v] = n
		n++
		stack = append(stack, v)
		onStack[v] = true
		for _, w := range prereqs[v] {
			if _, seen := index[w]; !seen {
				visit(w)
				if low[w] < low[v] {
					low[v] = low[w]
				}
			} else if onStack[w] && index[w] < low[v] {
				low[v] = index[w]
			}
		}
		if low[v] == index[v] {
			var comp []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				comp = append(comp, w)
				if w == v {
					break
				}
			}
			self := false
			for _, p := range prereqs[v] {
				if p == v {
					self = true
				}
			}
			if len(comp) > 1 || self {
				sort.Strings(comp)
				out = append(out, comp)
			}
		}
	}
	for _, id := range ids {
		if _, seen := index[id]; !seen {
			visit(id)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i][0] < out[j][0] })
	return out
}

// ---------------------------------------------------------------- labels

// stopwords carry no meaning for matching one topic to another.
var stopwords = map[string]bool{
	"the": true, "of": true, "and": true, "a": true, "an": true, "in": true, "for": true,
	"with": true, "to": true, "on": true, "by": true, "from": true, "as": true, "its": true,
	"is": true, "or": true, "at": true, "vs": true, "versus": true, "practice": true,
	"basics": true, "basic": true, "concept": true, "introduction": true, "introducing": true,
	"using": true, "use": true, "what": true, "how": true, "their": true, "it": true,
	"into": true, "one": true, "two": true, "n": true, "x": true, "y": true, "s": true,
	"i": true, "ii": true, "ch": true, "part": true, "more": true, "other": true,
}

// synonyms fold spellings of one idea onto one token.
var synonyms = map[string]string{
	"matrices": "matrix", "derivatives": "derivative", "differentiation": "derivative",
	"differentiate": "derivative", "differentiating": "derivative",
	"integrals": "integral", "integration": "integral", "integrate": "integral",
	"eigenvalues": "eigen", "eigenvalue": "eigen", "eigenvectors": "eigen", "eigenvector": "eigen",
	"equations": "equation", "functions": "function", "sets": "set", "systems": "system",
	"inverses": "inverse", "determinants": "determinant", "variables": "variable",
	"series": "series", "sequences": "sequence", "limits": "limit", "vectors": "vector",
	"maximization": "optimization", "maximisation": "optimization", "minimization": "optimization",
	"maximize": "optimization", "maximum": "extrema", "minimum": "extrema", "maxima": "extrema",
	"minima": "extrema", "optimisation": "optimization", "optimum": "optimization",
	"statics": "statics", "regressions": "regression", "regress": "regression",
	"ode": "differential", "odes": "differential", "homogeneous": "homogeneous",
	"neighbourhoods": "neighbourhood", "neighborhoods": "neighbourhood", "neighborhood": "neighbourhood",
	"models": "model", "multipliers": "multiplier", "tests": "test", "testing": "test",
	"estimates": "estimate", "estimation": "estimate", "estimating": "estimate", "estimator": "estimate",
	"predictions": "prediction", "predicting": "prediction", "predict": "prediction",
	"plots": "plot", "plotting": "plot", "charts": "plot", "chart": "plot", "graphs": "graph",
	"arrays": "array", "indexing": "index", "indices": "index", "probabilities": "probability",
	"distributions": "distribution", "properties": "property", "rules": "rule",
	"conditions": "condition", "points": "point", "lines": "line", "curves": "curve",
	"spaces": "space", "subspaces": "subspace", "transformations": "transformation",
	"transforms": "transformation", "maps": "map", "mappings": "map", "mapping": "map",
	"continuous": "continuity", "convergent": "convergence", "converge": "convergence",
	"converges": "convergence", "stable": "stability", "quadratics": "quadratic",
	"polynomials": "polynomial", "logarithms": "logarithm", "logarithmic": "logarithm",
	"logs": "logarithm", "log": "logarithm", "exponentials": "exponential", "exponents": "exponent",
	"supremum": "supremum", "sup": "supremum", "infimum": "supremum", "bounds": "bound", "bounded": "bound",
	"partials": "partial", "hessians": "hessian", "jacobians": "jacobian", "dummies": "dummy",
	"heteroskedasticity": "heteroskedasticity", "heteroscedasticity": "heteroskedasticity",
	"panels": "panel", "effects": "effect", "errors": "error", "residuals": "residual",
	"intervals": "interval", "values": "value", "constraints": "constraint", "constrained": "constraint",
	"compact": "compactness", "closed": "closed", "open": "open",
}

type index struct {
	tokens map[string]map[string]bool // id -> token set (label + subdomain tail)
	idf    map[string]float64
}

func tokenize(s string) []string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == 'é':
			b.WriteRune('e')
		default:
			b.WriteRune(' ')
		}
	}
	var out []string
	for _, w := range strings.Fields(b.String()) {
		if stopwords[w] || len(w) < 2 {
			continue
		}
		if syn, ok := synonyms[w]; ok {
			w = syn
		} else if strings.HasSuffix(w, "ies") && len(w) > 4 {
			w = w[:len(w)-3] + "y"
		} else if strings.HasSuffix(w, "s") && !strings.HasSuffix(w, "ss") && len(w) > 3 {
			w = w[:len(w)-1]
		}
		out = append(out, w)
	}
	return out
}

func newIndex(cs []concepts.Concept) *index {
	ix := &index{tokens: map[string]map[string]bool{}, idf: map[string]float64{}}
	df := map[string]int{}
	for i := range cs {
		c := &cs[i]
		set := map[string]bool{}
		for _, t := range tokenize(c.Label) {
			set[t] = true
		}
		// The subdomain's last part names the topic's area ("ma_hoy_ch8",
		// "determinants"); its words count at a lower weight through idf only.
		if sub := c.Subdomain; sub != "" {
			tail := sub[strings.LastIndex(sub, ".")+1:]
			for _, t := range tokenize(strings.ReplaceAll(tail, "_", " ")) {
				if len(t) > 3 {
					set[t] = true
				}
			}
		}
		ix.tokens[c.ID] = set
		for t := range set {
			df[t]++
		}
	}
	n := float64(len(cs))
	for t, d := range df {
		ix.idf[t] = math.Log(1 + n/float64(d))
	}
	return ix
}

// similarity is the idf-weighted Jaccard index of two concepts' label words.
func (ix *index) similarity(a, b string) float64 {
	s, _ := ix.similarityShared(a, b)
	return s
}

func (ix *index) similarityShared(a, b string) (float64, []string) {
	ta, tb := ix.tokens[a], ix.tokens[b]
	if len(ta) == 0 || len(tb) == 0 {
		return 0, nil
	}
	var inter, union float64
	var shared []string
	for t := range ta {
		w := ix.idf[t]
		union += w
		if tb[t] {
			inter += w
			shared = append(shared, t)
		}
	}
	for t := range tb {
		if !ta[t] {
			union += ix.idf[t]
		}
	}
	if union == 0 {
		return 0, nil
	}
	sort.Strings(shared)
	return inter / union, shared
}

// ---------------------------------------------------------------- loading

// Load reads the graph an installation runs with: the corpus at base and the
// modules' concept folders, keyed by module id (the folder's parent name).
// It reads what concepts.LoadDirs reads, without refusing an invalid graph,
// so the audit can report every problem.
func Load(base string, moduleDirs map[string]string) (Graph, error) {
	mods := make([]string, 0, len(moduleDirs))
	for id := range moduleDirs {
		mods = append(mods, id)
	}
	sort.Strings(mods)
	dirs := make([]string, 0, len(mods))
	byDir := map[string]string{}
	for _, id := range mods {
		d := filepath.Clean(moduleDirs[id])
		dirs = append(dirs, d)
		byDir[d] = id
	}
	raw, from, err := concepts.ReadRawDirs(base, dirs...)
	if err != nil {
		return Graph{}, err
	}
	g := Graph{Concepts: raw, ModuleOf: make(map[string]string, len(raw))}
	for i, c := range raw {
		if m, ok := byDir[filepath.Clean(from[i])]; ok {
			g.ModuleOf[c.ID] = m
		} else {
			g.ModuleOf[c.ID] = ""
		}
	}
	return g, nil
}

// Text renders the report for a terminal.
func (r *Report) Text(verbose bool) string {
	var b strings.Builder
	p := func(format string, args ...any) { fmt.Fprintf(&b, format, args...) }
	mods := make([]string, 0, len(r.Modules))
	for m := range r.Modules {
		mods = append(mods, m)
	}
	sort.Strings(mods)
	p("%d concepts", r.Concepts)
	for _, m := range mods {
		name := m
		if name == "" {
			name = "corpus"
		}
		p("; %s %d", name, r.Modules[m])
	}
	p("\n")
	hard := r.HardErrors()
	p("hard: %d cycles, %d dangling, %d without subdomain, %d domains without label, %d key-prerequisite problems\n",
		len(r.Cycles), len(r.Dangling), len(r.MissingSubdomain), len(r.MissingDomainLabel), len(r.BadKeyPrereqs))
	for _, h := range hard {
		p("  FAIL %s\n", h)
	}
	p("isolated (no prerequisites, no dependents): %d\n", len(r.Isolated))
	for _, id := range r.Isolated {
		p("  %s\n", id)
	}
	p("cross-module prerequisite links per module:\n")
	for _, m := range mods {
		if m == "" {
			continue
		}
		p("  %-20s %4d links out of the module; %3d topics stay inside it; %3d roots; %3d not on a course or reading\n",
			m, r.CrossModuleLinks[m], len(r.OwnModuleOnly[m]), len(r.Roots[m]), len(r.Uncovered[m]))
	}
	p("duplicates (label similarity >= threshold): %d\n", len(r.Duplicates))
	for _, d := range r.Duplicates {
		tag := ""
		if d.Linked {
			tag = " (linked)"
		}
		p("  %.2f %s (%s) ~ %s (%s)%s\n", d.Score, d.A, or(d.ModuleA, "corpus"), d.B, or(d.ModuleB, "corpus"), tag)
	}
	p("longest chain: %s, %d deep through %s\n", r.MaxDepth.ID, r.MaxDepth.Depth, strings.Join(r.MaxDepth.Domains, " > "))
	p("chains that wander (%d+ tracks, or back into a domain they left): %d\n", r.ChainTracks, len(r.DeepChains))
	for i, c := range r.DeepChains {
		if i >= 10 && !verbose {
			p("  ... %d more\n", len(r.DeepChains)-i)
			break
		}
		p("  %s (%d deep, %d tracks, %d revisits): %s\n", c.ID, c.Depth, len(c.Tracks), c.Revisits, strings.Join(c.Domains, " > "))
	}
	p("4+ prerequisites and no key_prerequisites: %d\n", len(r.ManyPrereqsNoKey))
	p("suggested cross-module prerequisites: %d\n", len(r.Suggestions))
	if verbose {
		for _, s := range r.Suggestions {
			own := ""
			if s.OnlyOwnModule {
				own = " [own-module only]"
			}
			p("  %.2f %s (%s) <- %s (%s) [%s]%s\n", s.Score, s.Topic, s.Module, s.Candidate, or(s.CandModule, "corpus"), strings.Join(s.Shared, ","), own)
		}
		for _, m := range mods {
			if len(r.Uncovered[m]) > 0 {
				p("not on a course or reading, %s: %s\n", m, strings.Join(r.Uncovered[m], ", "))
			}
		}
	}
	return b.String()
}

func or(a, b string) string {
	if a == "" {
		return b
	}
	return a
}

// LoadInstall loads the graph of an installation: the corpus at base and the
// concepts of every enabled module in set. A domain counts as labelled when
// the corpus uses it (the web app labels those) or a module.json labels it.
func LoadInstall(base string, set *modules.Set) (Graph, error) {
	dirs := map[string]string{}
	for _, m := range set.Modules {
		d := filepath.Join(m.Dir, "concepts")
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			dirs[m.ID] = d
		}
	}
	g, err := Load(base, dirs)
	if err != nil {
		return g, err
	}
	corpus := map[string]bool{}
	for _, c := range g.Concepts {
		if g.ModuleOf[c.ID] == "" {
			corpus[c.Domain] = true
		}
	}
	g.DomainLabel = func(d string) bool { return corpus[d] || set.DomainLabel(d) != "" }
	g.TrackOf = set.TrackOf
	return g, nil
}
