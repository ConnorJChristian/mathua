// Package modules finds the content modules a Mathua installation runs with.
//
// The base install is upstream Mathua: the engine, the web app and the
// mathematics corpus in data/. A module adds content on top of it, and every
// kind of content is optional:
//
//	modules/<id>/            committed modules (public, original content)
//	modules-local/<id>/      your own modules; git-ignored, never pushed
//	  module.json            name, description, tracks, domain labels
//	  concepts/*.json        concept-graph nodes, same format as data/concepts
//	  bank/*.json            question banks (internal/pytrack)
//	  proofs/*.json          proof problems and marking schemes (internal/proof)
//	  readings.json          per-concept "read this section" pointers
//	  courses.json           courses, same format as data/courses.json
//	  destinations.json      destinations, same format as data/destinations.json
//	  files/                 files the readings open (your PDFs); local only
//
// A module is switched off by listing its id under "disabled" in the
// git-ignored modules.local.json at the repository root. Modules are read
// once at startup; the web app bundles the concept list at build time, so
// adding or removing one takes a rebuild (`docker compose up --build`).
package modules

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Track is a side of the site the header switch can show. "math" is the
// base install's track and always exists; a module may declare more.
type Track struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	// Domains are the concept domains that belong to this track, in display
	// order. A concept whose domain no module claims is on the math track.
	Domains []string `json:"domains"`
	// NoPlacement marks a track whose first topics are open from the start,
	// so the placement diagnostic is never offered for it.
	NoPlacement bool `json:"no_placement,omitempty"`
}

// Manifest is module.json.
type Manifest struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Tracks       []Track           `json:"tracks,omitempty"`
	DomainLabels map[string]string `json:"domain_labels,omitempty"`
	// NoLibrary keeps a module's online sources (wikis, guides) off the
	// library shelves, which hold books (fork change).
	NoLibrary bool `json:"no_library,omitempty"`
}

// Module is one discovered, enabled module.
type Module struct {
	Manifest
	Dir   string // absolute
	Local bool   // from modules-local/
}

// Reading points a concept at the section of a source to read before
// practising it. File is relative to the module's files/ folder.
type Reading struct {
	Source string `json:"source"`         // "Hoy 12.1, pp. 474-484"
	Note   string `json:"note,omitempty"` // optional short original note
	File   string `json:"file,omitempty"` // "hoy-3e.pdf"
	Page   int    `json:"page,omitempty"` // PDF page to open at
	// EndPage is the PDF page the section ends on, so the embedded reader can
	// say where to stop (fork change).
	EndPage int    `json:"end_page,omitempty"`
	URL     string `json:"url,omitempty"` // or an https link
	Module  string `json:"-"`
}

// Book names the text a reading is in: "<module>/<file>" for a file, "" for
// a link. Courses list their authority texts by this key (fork change).
func (r Reading) Book() string {
	if r.File == "" {
		return ""
	}
	return r.Module + "/" + r.File
}

// BookMeta describes a file in a module's files/ folder for the library
// (fork change): books.json maps a file name to its title and author.
// SourcePrefix is trimmed from the front of each reading's source when the
// library lists it as a section ("Hoy " leaves "5.4 Rules of ...").
type BookMeta struct {
	Title        string `json:"title"`
	Author       string `json:"author,omitempty"`
	SourcePrefix string `json:"source_prefix,omitempty"`
	// Shelf groups the book in the library ("Econometrics"); empty uses the track.
	Shelf string `json:"shelf,omitempty"`
	// TOC is the book's table of contents (PDF pages), for the library.
	TOC    []TOCEntry `json:"toc,omitempty"`
	Module string     `json:"-"`
	File   string     `json:"-"`
}

// TOCEntry is one line of a book's table of contents.
type TOCEntry struct {
	Title   string `json:"title"`
	Page    int    `json:"page"`
	EndPage int    `json:"end_page,omitempty"`
}

// Video is a lecture or clip for a topic or a course (fork change):
// videos.json maps a concept id or a course id to a list of these. URL is an
// https link (YouTube or a direct video URL) or File is in the module's files/.
type Video struct {
	Title    string `json:"title"`
	URL      string `json:"url,omitempty"`
	File     string `json:"file,omitempty"`
	Start    int    `json:"start,omitempty"`    // seconds into the video
	Duration int    `json:"duration,omitempty"` // seconds, when known
	Note     string `json:"note,omitempty"`
	Module   string `json:"module,omitempty"`
}

// Note is a page of lecture notes (fork change): notes.json maps a concept
// id or a course id to a list of these. File is an HTML page in the module's
// files/ (a lecture notebook rendered to HTML, say) or URL an https link.
// Unit names the lecture the page belongs to, Seq its place in that unit.
type Note struct {
	Title   string `json:"title"`
	Unit    string `json:"unit,omitempty"`
	UnitSeq int    `json:"unit_seq,omitempty"` // the unit's place in the course
	Seq     int    `json:"seq,omitempty"`
	URL     string `json:"url,omitempty"`
	File    string `json:"file,omitempty"`
	Module  string `json:"module,omitempty"`
}

// Set is the enabled modules of one installation.
type Set struct {
	Root     string // repository or install root
	Modules  []*Module
	Disabled []string

	tracks      []Track
	domainTrack map[string]string
	readings    map[string][]Reading
	books       []BookMeta
	videos      map[string][]Video
	notes       map[string][]Note
	cards       map[string][]CardRef
	noLibrary   map[string]bool // domains of no_library modules
}

const MathTrack = "math"

// LocalConfig is modules.local.json.
type LocalConfig struct {
	Disabled []string `json:"disabled"`
}

var (
	defaultOnce sync.Once
	defaultSet  *Set
	defaultErr  error
)

// Default discovers modules once per process. MATHUA_PUBLIC_MODULES_ONLY=1
// leaves out modules-local/ (the test suites set it, so they pass the same on
// every machine). MATHUA_ROOT names the root;
// otherwise the nearest directory at or above the working directory (then
// the executable) that has a modules/ or data/concepts/ folder is used, so
// tests run from a package directory find the repository.
func Default() (*Set, error) {
	defaultOnce.Do(func() {
		root := os.Getenv("MATHUA_ROOT")
		if root == "" {
			root = findRoot()
		}
		if os.Getenv("MATHUA_PUBLIC_MODULES_ONLY") == "1" {
			defaultSet, defaultErr = LoadPublic(root)
			return
		}
		defaultSet, defaultErr = Load(root)
	})
	return defaultSet, defaultErr
}

func findRoot() string {
	var starts []string
	if wd, err := os.Getwd(); err == nil {
		starts = append(starts, wd)
	}
	if exe, err := os.Executable(); err == nil {
		starts = append(starts, filepath.Dir(exe))
	}
	for _, s := range starts {
		d := s
		for {
			if isDir(filepath.Join(d, "data", "concepts")) || hasManifests(filepath.Join(d, "modules")) {
				return d
			}
			parent := filepath.Dir(d)
			if parent == d {
				break
			}
			d = parent
		}
	}
	return "."
}

// hasManifests reports a modules folder that holds at least one module. A
// bare "modules" directory is not enough: internal/modules (this package's
// own source folder) is one.
func hasManifests(dir string) bool {
	m, _ := filepath.Glob(filepath.Join(dir, "*", "module.json"))
	return len(m) > 0
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// Load discovers the enabled modules under root: modules/ and modules-local/.
func Load(root string) (*Set, error) { return load(root, true) }

// LoadPublic discovers only the committed modules (modules/). Tests use it,
// so they pass the same on every machine whatever its modules-local/ holds.
func LoadPublic(root string) (*Set, error) { return load(root, false) }

func load(root string, withLocal bool) (*Set, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	s := &Set{Root: abs, domainTrack: map[string]string{}, readings: map[string][]Reading{}, videos: map[string][]Video{}, notes: map[string][]Note{}, cards: map[string][]CardRef{}}
	var cfg LocalConfig
	if raw, err := os.ReadFile(filepath.Join(abs, "modules.local.json")); err == nil {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("modules.local.json: %w", err)
		}
	}
	off := map[string]bool{}
	for _, id := range cfg.Disabled {
		off[id] = true
	}
	s.Disabled = cfg.Disabled
	if v := os.Getenv("MATHUA_DISABLE_MODULES"); v != "" {
		for _, id := range strings.Split(v, ",") {
			if id = strings.TrimSpace(id); id != "" {
				off[id] = true
			}
		}
	}

	seen := map[string]string{}
	for _, area := range []struct {
		dir   string
		local bool
	}{{"modules", false}, {"modules-local", true}} {
		if area.local && !withLocal {
			continue
		}
		entries, err := os.ReadDir(filepath.Join(abs, area.dir))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || strings.HasPrefix(e.Name(), "_") {
				continue
			}
			dir := filepath.Join(abs, area.dir, e.Name())
			m, err := readModule(dir)
			if err != nil {
				return nil, err
			}
			if m == nil {
				continue // a folder without module.json is not a module
			}
			m.Local = area.local
			if prev, dup := seen[m.ID]; dup {
				return nil, fmt.Errorf("module id %q is used by both %s and %s", m.ID, prev, dir)
			}
			seen[m.ID] = dir
			if off[m.ID] {
				continue
			}
			s.Modules = append(s.Modules, m)
		}
	}
	sort.SliceStable(s.Modules, func(i, j int) bool {
		if s.Modules[i].Local != s.Modules[j].Local {
			return !s.Modules[i].Local
		}
		return s.Modules[i].ID < s.Modules[j].ID
	})

	s.tracks = []Track{{ID: MathTrack, Label: "Math"}}
	trackIdx := map[string]int{MathTrack: 0}
	for _, m := range s.Modules {
		for _, t := range m.Tracks {
			if t.ID == "" || t.Label == "" {
				return nil, fmt.Errorf("module %s: every track needs an id and a label", m.ID)
			}
			if t.ID == MathTrack {
				return nil, fmt.Errorf("module %s: track id %q is reserved for the base install", m.ID, MathTrack)
			}
			i, ok := trackIdx[t.ID]
			if !ok {
				i = len(s.tracks)
				trackIdx[t.ID] = i
				s.tracks = append(s.tracks, Track{ID: t.ID, Label: t.Label, NoPlacement: t.NoPlacement})
			}
			// Two modules may add domains to one track (a public chapter set and
			// your own notes for the same subject, say).
			for _, d := range t.Domains {
				if m.NoLibrary {
					if s.noLibrary == nil {
						s.noLibrary = map[string]bool{}
					}
					s.noLibrary[d] = true
				}
				if owner, taken := s.domainTrack[d]; taken && owner != t.ID {
					return nil, fmt.Errorf("domain %q is claimed by tracks %q and %q", d, owner, t.ID)
				}
				if _, taken := s.domainTrack[d]; !taken {
					s.domainTrack[d] = t.ID
					s.tracks[i].Domains = append(s.tracks[i].Domains, d)
				}
			}
			s.tracks[i].NoPlacement = s.tracks[i].NoPlacement || t.NoPlacement
		}
		readings, err := readReadings(m)
		if err != nil {
			return nil, err
		}
		// A concept may have readings in several texts (from one module or
		// several). They are kept in module order; a course picks its own.
		for cid, rs := range readings {
			for _, r := range rs {
				r.Module = m.ID
				s.readings[cid] = append(s.readings[cid], r)
			}
		}
		books, err := readBooks(m)
		if err != nil {
			return nil, err
		}
		s.books = append(s.books, books...)
		videos, err := readVideos(m)
		if err != nil {
			return nil, err
		}
		for id, vs := range videos {
			for _, v := range vs {
				v.Module = m.ID
				s.videos[id] = append(s.videos[id], v)
			}
		}
		notes, err := readNotes(m)
		if err != nil {
			return nil, err
		}
		for id, ns := range notes {
			for _, n := range ns {
				n.Module = m.ID
				s.notes[id] = append(s.notes[id], n)
			}
		}
		cards, err := readCards(m)
		if err != nil {
			return nil, err
		}
		for topic, refs := range cards {
			s.cards[topic] = append(s.cards[topic], refs...)
		}
	}
	return s, nil
}

func readVideos(m *Module) (map[string][]Video, error) {
	raw, err := os.ReadFile(filepath.Join(m.Dir, "videos.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var byID map[string][]Video
	if err := json.Unmarshal(raw, &byID); err != nil {
		return nil, fmt.Errorf("%s/videos.json: %w", m.ID, err)
	}
	for id, vs := range byID {
		for _, v := range vs {
			if v.File != "" && !safeRel(v.File) {
				return nil, fmt.Errorf("%s/videos.json: %s: file %q must be a path inside files/", m.ID, id, v.File)
			}
			if v.File == "" && !strings.HasPrefix(v.URL, "https://") {
				return nil, fmt.Errorf("%s/videos.json: %s: a video needs a file or an https url", m.ID, id)
			}
		}
	}
	return byID, nil
}

// CardRef names one question-bank item: the bank concept (a deck) and the
// item id in it (fork change). cards.json maps a topic id to the cards that
// are about it, so a card is practised with its topic rather than with the
// whole deck it was imported in.
type CardRef struct {
	Deck string `json:"deck"`
	Item string `json:"item"`
}

func readCards(m *Module) (map[string][]CardRef, error) {
	raw, err := os.ReadFile(filepath.Join(m.Dir, "cards.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var byTopic map[string][]CardRef
	if err := json.Unmarshal(raw, &byTopic); err != nil {
		return nil, fmt.Errorf("%s/cards.json: %w", m.ID, err)
	}
	return byTopic, nil
}

// LibraryHidden reports a domain whose module keeps its sources off the
// library shelves (no_library).
func (s *Set) LibraryHidden(domain string) bool { return s != nil && s.noLibrary[domain] }

// Cards is every topic's filed cards (cards.json), keyed by topic id.
func (s *Set) Cards() map[string][]CardRef {
	out := make(map[string][]CardRef, len(s.cards))
	for k, v := range s.cards {
		out[k] = append([]CardRef(nil), v...)
	}
	return out
}

func readNotes(m *Module) (map[string][]Note, error) {
	raw, err := os.ReadFile(filepath.Join(m.Dir, "notes.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var byID map[string][]Note
	if err := json.Unmarshal(raw, &byID); err != nil {
		return nil, fmt.Errorf("%s/notes.json: %w", m.ID, err)
	}
	for id, ns := range byID {
		for _, n := range ns {
			if n.File != "" && !safeRel(n.File) {
				return nil, fmt.Errorf("%s/notes.json: %s: file %q must be a path inside files/", m.ID, id, n.File)
			}
			if n.File == "" && !strings.HasPrefix(n.URL, "https://") {
				return nil, fmt.Errorf("%s/notes.json: %s: a note needs a file or an https url", m.ID, id)
			}
		}
	}
	return byID, nil
}

// NotesFor lists the lecture notes attached to a concept or course id
// (notes.json), fork change.
func (s *Set) NotesFor(id string) []Note {
	if s == nil {
		return nil
	}
	return append([]Note(nil), s.notes[id]...)
}

// VideosFor lists the videos attached to a concept or course id (videos.json).
func (s *Set) VideosFor(id string) []Video {
	if s == nil {
		return nil
	}
	return append([]Video(nil), s.videos[id]...)
}

func readBooks(m *Module) ([]BookMeta, error) {
	raw, err := os.ReadFile(filepath.Join(m.Dir, "books.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var byFile map[string]BookMeta
	if err := json.Unmarshal(raw, &byFile); err != nil {
		return nil, fmt.Errorf("%s/books.json: %w", m.ID, err)
	}
	var out []BookMeta
	for f, b := range byFile {
		if !safeRel(f) {
			return nil, fmt.Errorf("%s/books.json: file %q must be a path inside files/", m.ID, f)
		}
		b.Module, b.File = m.ID, f
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].File < out[j].File })
	return out, nil
}

// Books lists the described files of every enabled module (books.json).
func (s *Set) Books() []BookMeta { return append([]BookMeta(nil), s.books...) }

// Readings is every concept's readings, keyed by concept id.
func (s *Set) Readings() map[string][]Reading {
	out := make(map[string][]Reading, len(s.readings))
	for k, v := range s.readings {
		out[k] = append([]Reading(nil), v...)
	}
	return out
}

func readModule(dir string) (*Module, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "module.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var man Manifest
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&man); err != nil {
		return nil, fmt.Errorf("%s/module.json: %w", dir, err)
	}
	if man.ID == "" {
		man.ID = filepath.Base(dir)
	}
	if strings.ContainsAny(man.ID, "/\\. ") {
		return nil, fmt.Errorf("%s/module.json: id %q may not contain '/', '\\', '.' or spaces", dir, man.ID)
	}
	if man.Name == "" {
		man.Name = man.ID
	}
	return &Module{Manifest: man, Dir: dir}, nil
}

// readReadings reads readings.json: concept id to one reading, or to a list
// of readings in different texts.
func readReadings(m *Module) (map[string][]Reading, error) {
	raw, err := os.ReadFile(filepath.Join(m.Dir, "readings.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var entries map[string]json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("%s/readings.json: %w", m.ID, err)
	}
	out := make(map[string][]Reading, len(entries))
	for cid, e := range entries {
		var list []Reading
		if t := strings.TrimSpace(string(e)); strings.HasPrefix(t, "[") {
			if err := json.Unmarshal(e, &list); err != nil {
				return nil, fmt.Errorf("%s/readings.json: %s: %w", m.ID, cid, err)
			}
		} else {
			var one Reading
			if err := json.Unmarshal(e, &one); err != nil {
				return nil, fmt.Errorf("%s/readings.json: %s: %w", m.ID, cid, err)
			}
			list = []Reading{one}
		}
		for _, r := range list {
			if r.Source == "" {
				return nil, fmt.Errorf("%s/readings.json: %s has no source", m.ID, cid)
			}
			if r.File != "" && !safeRel(r.File) {
				return nil, fmt.Errorf("%s/readings.json: %s: file %q must be a path inside files/", m.ID, cid, r.File)
			}
		}
		out[cid] = list
	}
	return out, nil
}

func safeRel(p string) bool {
	if p == "" || filepath.IsAbs(p) || strings.HasPrefix(p, "/") || strings.HasPrefix(p, "\\") {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(p))
	return clean != ".." && !strings.HasPrefix(clean, "../")
}

// dirsNamed lists <module>/<name> for every module that has it.
func (s *Set) dirsNamed(name string) []string {
	var out []string
	for _, m := range s.Modules {
		if p := filepath.Join(m.Dir, name); isDir(p) {
			out = append(out, p)
		}
	}
	return out
}

func (s *Set) filesNamed(name string) []string {
	var out []string
	for _, m := range s.Modules {
		p := filepath.Join(m.Dir, name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			out = append(out, p)
		}
	}
	return out
}

// ConceptDirs are the modules' concepts/ folders.
func (s *Set) ConceptDirs() []string { return s.dirsNamed("concepts") }

// BankDirs are the modules' bank/ folders.
func (s *Set) BankDirs() []string { return s.dirsNamed("bank") }

// ProofDirs are the modules' proofs/ folders: proof problems with their
// marking schemes (internal/proof).
func (s *Set) ProofDirs() []string { return s.dirsNamed("proofs") }

// CourseFiles are the modules' courses.json files.
func (s *Set) CourseFiles() []string { return s.filesNamed("courses.json") }

// DestinationFiles are the modules' destinations.json files.
func (s *Set) DestinationFiles() []string { return s.filesNamed("destinations.json") }

// Tracks lists the math track first, then the modules' tracks.
func (s *Set) Tracks() []Track {
	out := make([]Track, len(s.tracks))
	copy(out, s.tracks)
	return out
}

// TrackOf names the track a concept domain belongs to.
func (s *Set) TrackOf(domain string) string {
	if t, ok := s.domainTrack[domain]; ok {
		return t
	}
	return MathTrack
}

// Normalize maps a request parameter onto a known track id. Anything unknown
// is the math track, so a client that never sends one keeps the base
// behaviour.
func (s *Set) Normalize(track string) string {
	track = strings.ToLower(strings.TrimSpace(track))
	for _, t := range s.tracks {
		if t.ID == track {
			return t.ID
		}
	}
	return MathTrack
}

// HasPlacement reports whether a track offers the placement diagnostic.
func (s *Set) HasPlacement(track string) bool {
	for _, t := range s.tracks {
		if t.ID == track {
			return !t.NoPlacement
		}
	}
	return true
}

// DomainLabel returns a module's label for a domain, or "".
func (s *Set) DomainLabel(domain string) string {
	for _, m := range s.Modules {
		if l, ok := m.DomainLabels[domain]; ok {
			return l
		}
	}
	return ""
}

// Reading returns a concept's first reading, if a module has one.
func (s *Set) Reading(conceptID string) (Reading, bool) {
	rs := s.readings[conceptID]
	if len(rs) == 0 {
		return Reading{}, false
	}
	return rs[0], true
}

// ReadingsFor lists every reading a concept has, one per text.
func (s *Set) ReadingsFor(conceptID string) []Reading {
	return append([]Reading(nil), s.readings[conceptID]...)
}

// PickReading chooses a concept's reading for a learner working through
// courses whose authority texts are `texts`, in priority order: the first
// text that has a reading wins, and with none of them, the first reading.
func (s *Set) PickReading(conceptID string, texts []string) (Reading, bool) {
	rs := s.readings[conceptID]
	if len(rs) == 0 {
		return Reading{}, false
	}
	for _, t := range texts {
		for _, r := range rs {
			if r.Book() == t {
				return r, true
			}
		}
	}
	return rs[0], true
}

// Module returns an enabled module by id.
func (s *Set) Module(id string) *Module {
	for _, m := range s.Modules {
		if m.ID == id {
			return m
		}
	}
	return nil
}

// FilePath resolves a file inside an enabled module's files/ folder, or
// returns "" when the path would leave it.
func (s *Set) FilePath(moduleID, rel string) string {
	m := s.Module(moduleID)
	if m == nil || !safeRel(rel) {
		return ""
	}
	base := filepath.Join(m.Dir, "files")
	p := filepath.Join(base, filepath.FromSlash(rel))
	if r, err := filepath.Rel(base, p); err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return ""
	}
	return p
}
