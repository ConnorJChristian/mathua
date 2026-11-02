// Package books names the book a question comes from (fork change): the
// order of practice inside a topic serves the course's own book first, then
// the same topic in other books, then generated problem sets (FORK.md, "The
// order of practice inside a topic"; generator.Scaffold).
//
// A book has a stable id ("hoy", "schaums-calc-econ") and a short label for
// the lesson's progress line. registry.json holds the public rules: each
// book's aliases, matched against a normalised name, file, tag or deck
// segment, and the default preference order of parallel books. A module's
// books.json may give one of its files a `book_id` (modules.BookMeta) for a
// book the registry does not know; Register adds it.
package books

import (
	_ "embed"
	"encoding/json"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"unicode"
)

//go:embed registry.json
var registryJSON []byte

// Book is one book of the registry.
type Book struct {
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	Aliases []string `json:"aliases"`
}

type registry struct {
	Books  []Book   `json:"books"`
	Prefer []string `json:"prefer"`
}

type alias struct{ key, id string }

var (
	mu      sync.RWMutex
	loaded  bool
	byID    map[string]*Book
	aliases []alias // longest first
	prefer  []string
)

func load() {
	if loaded {
		return
	}
	var r registry
	if err := json.Unmarshal(registryJSON, &r); err != nil {
		panic("books: registry.json: " + err.Error())
	}
	byID = map[string]*Book{}
	for i := range r.Books {
		b := r.Books[i]
		byID[b.ID] = &b
		aliases = append(aliases, alias{b.ID, b.ID})
		for _, a := range b.Aliases {
			aliases = append(aliases, alias{Key(a), b.ID})
		}
	}
	sortAliases()
	prefer = r.Prefer
	loaded = true
}

func sortAliases() {
	sort.SliceStable(aliases, func(i, j int) bool { return len(aliases[i].key) > len(aliases[j].key) })
}

func ensure() {
	mu.RLock()
	ok := loaded
	mu.RUnlock()
	if ok {
		return
	}
	mu.Lock()
	load()
	mu.Unlock()
}

// Key is a name in the form aliases are matched in: lower case, accents and
// apostrophes dropped, a "web:"/"notes:" scheme, a folder and a ".pdf"
// extension removed, every other run of punctuation or space one '-'.
func Key(s string) string {
	s = strings.TrimSpace(s)
	for _, p := range []string{"web:", "notes:", "file:"} {
		s = strings.TrimPrefix(s, p)
	}
	if strings.Contains(s, "/") && !strings.Contains(s, " ") {
		s = path.Base(s)
	}
	s = strings.TrimSuffix(strings.TrimSuffix(s, ".pdf"), ".PDF")
	var b strings.Builder
	dash := false
	for _, r := range s {
		switch {
		case unicode.Is(unicode.Mn, r), r == '\'', r == '’':
			continue
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
			dash = false
		default:
			if !dash && b.Len() > 0 {
				b.WriteByte('-')
				dash = true
			}
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}

// Known is the registry book a name, file, tag or deck segment names, or ""
// when it names none.
func Known(name string) string {
	ensure()
	k := Key(name)
	if k == "" {
		return ""
	}
	mu.RLock()
	defer mu.RUnlock()
	for _, a := range aliases {
		if k == a.key || strings.HasPrefix(k, a.key+"-") {
			return a.id
		}
	}
	return ""
}

// ID is the stable id of the book a name names: the registry's, else the
// name's own Key (an authored source the registry does not list is still a
// book of its own). "" for an empty name.
func ID(name string) string {
	if id := Known(name); id != "" {
		return id
	}
	return Key(name)
}

// Register adds a book a module names (books.json `book_id`) with the names
// that should resolve to it: its file and its title. A known id keeps its
// label.
func Register(id, label string, names ...string) {
	ensure()
	id = Key(id)
	if id == "" {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	if _, ok := byID[id]; !ok {
		byID[id] = &Book{ID: id, Label: label}
		aliases = append(aliases, alias{id, id})
	}
	for _, n := range names {
		if k := Key(n); k != "" {
			aliases = append(aliases, alias{k, id})
		}
	}
	sortAliases()
}

// Label is a book's short label for the progress line: the registry's, or
// the id with its dashes as spaces.
func Label(id string) string {
	if id == "" {
		return ""
	}
	ensure()
	mu.RLock()
	b := byID[id]
	mu.RUnlock()
	if b != nil && b.Label != "" {
		return b.Label
	}
	s := strings.ReplaceAll(id, "-", " ")
	if s != "" {
		s = strings.ToUpper(s[:1]) + s[1:]
	}
	return s
}

// PreferenceEnv overrides the registry's preference order of parallel
// books: comma-separated book names or ids.
const PreferenceEnv = "MATHUA_BOOK_PREFERENCE"

// Preference is the global order of parallel books after a course's own
// texts: MATHUA_BOOK_PREFERENCE, else the registry's.
func Preference() []string {
	ensure()
	if v := strings.TrimSpace(os.Getenv(PreferenceEnv)); v != "" {
		var out []string
		for _, p := range strings.Split(v, ",") {
			if id := ID(p); id != "" {
				out = append(out, id)
			}
		}
		return out
	}
	mu.RLock()
	defer mu.RUnlock()
	return append([]string(nil), prefer...)
}

// Order sorts book ids for a step's parallel books: first in the order of
// first (a course's texts, as ids), then of Preference, then Schaum's
// outlines, then by label.
func Order(ids []string, first []string) []string {
	rank := map[string]int{}
	n := 0
	for _, list := range [][]string{first, Preference()} {
		for _, id := range list {
			if _, ok := rank[id]; !ok && id != "" {
				rank[id] = n
				n++
			}
		}
	}
	out := append([]string(nil), ids...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		ra, oka := rank[a]
		rb, okb := rank[b]
		switch {
		case oka && okb:
			return ra < rb
		case oka != okb:
			return oka
		}
		sa, sb := strings.HasPrefix(a, "schaums"), strings.HasPrefix(b, "schaums")
		if sa != sb {
			return sa
		}
		return Label(a) < Label(b)
	})
	return out
}
