package server

import (
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/chuma-beep/mathua/internal/engine"
	"github.com/chuma-beep/mathua/internal/pytrack"
)

// Fork change: the library. Every source the install's topics are drawn from,
// as a book with a table of contents whose entries link to the topics:
//
//   - PDFs: your own files, from the modules' readings.json (and books.json
//     for titles). Each section opens in the embedded reader at its page.
//   - Online books: the question banks' sources that are books with section
//     links (the Python Data Science Handbook, the Workday manuals). Not
//     flashcard decks, and not a source that is one of the PDFs.
//
// GET /api/library

type libraryTopic struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type librarySection struct {
	Title   string         `json:"title"`
	Level   int            `json:"level"` // 0: a chapter from the book's contents; 1: a section inside one
	Page    int            `json:"page,omitempty"`
	EndPage int            `json:"end_page,omitempty"`
	URL     string         `json:"url,omitempty"`
	Topics  []libraryTopic `json:"topics"`
}

type libraryBook struct {
	ID        string           `json:"id"`
	Kind      string           `json:"kind"`            // "pdf" or "web"
	Track     string           `json:"track"`           // the track most of its topics are on
	Shelf     string           `json:"shelf,omitempty"` // a books.json shelf, when set
	Title     string           `json:"title"`
	Author    string           `json:"author,omitempty"`
	URL       string           `json:"url,omitempty"` // the PDF itself, for kind pdf
	Available bool             `json:"available"`     // the PDF is in the module's files/
	Sections  []librarySection `json:"sections"`
	Topics    int              `json:"topic_count"`
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func slug(s string) string {
	return strings.Trim(slugRe.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

func (s *Server) handleLibrary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, 405)
		return
	}
	writeJSON(w, map[string]interface{}{"books": s.library()})
}

func (s *Server) library() []libraryBook {
	mods := s.eng.Modules()
	dag := s.eng.GetDAG()
	label := func(id string) string {
		if c := dag.Concept(id); c != nil {
			return c.Label
		}
		return ""
	}
	meta := map[string]int{}
	books := mods.Books()
	for i, b := range books {
		meta[b.Module+"/"+b.File] = i
	}

	// PDFs from readings.
	var out []libraryBook
	pdfIdx := map[string]int{}
	secIdx := map[string]int{}
	readings := mods.Readings()
	ids := make([]string, 0, len(readings))
	for id := range readings {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	hasPDF := map[string]bool{}
	for _, cid := range ids {
		for _, rd := range readings[cid] {
			lb := label(cid)
			if rd.File == "" || lb == "" || !strings.EqualFold(path.Ext(rd.File), ".pdf") {
				continue
			}
			hasPDF[cid] = true
			key := rd.Module + "/" + rd.File
			bi, ok := pdfIdx[key]
			if !ok {
				b := libraryBook{ID: slug(key), Kind: "pdf", Title: rd.File,
					URL: "/api/modules/" + url.PathEscape(rd.Module) + "/files/" + escapePath(rd.File)}
				if mi, ok := meta[key]; ok {
					b.Title, b.Author, b.Shelf = books[mi].Title, books[mi].Author, books[mi].Shelf
				}
				if p := mods.FilePath(rd.Module, rd.File); p != "" {
					if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() {
						b.Available = true
					}
				}
				bi = len(out)
				pdfIdx[key] = bi
				out = append(out, b)
			}
			title := rd.Source
			if mi, ok := meta[key]; ok && books[mi].SourcePrefix != "" {
				title = strings.TrimPrefix(title, books[mi].SourcePrefix)
			}
			sk := key + "|" + title
			si, ok := secIdx[sk]
			if !ok {
				si = len(out[bi].Sections)
				secIdx[sk] = si
				out[bi].Sections = append(out[bi].Sections, librarySection{Title: title, Page: rd.Page, EndPage: rd.EndPage})
			}
			out[bi].Sections[si].Topics = append(out[bi].Sections[si].Topics, libraryTopic{cid, lb})
		}
	}
	// Lay each PDF's own contents underneath: chapters become level-0 rows
	// (taking the topics of a reading that is the chapter itself), and the
	// finer reading sections sit inside them at level 1.
	for i := range out {
		key := ""
		for k, bi := range pdfIdx {
			if bi == i {
				key = k
			}
		}
		mi, ok := meta[key]
		if !ok || len(books[mi].TOC) == 0 {
			continue
		}
		var merged []librarySection
		used := map[int]bool{}
		for _, t := range books[mi].TOC {
			row := librarySection{Title: t.Title, Page: t.Page, EndPage: t.EndPage}
			for si, sct := range out[i].Sections {
				if !used[si] && sct.Page == t.Page && (sct.EndPage == t.EndPage || strings.Contains(sct.Title, t.Title) || strings.Contains(t.Title, sct.Title)) {
					row.Topics = append(row.Topics, sct.Topics...)
					used[si] = true
				}
			}
			merged = append(merged, row)
		}
		for si, sct := range out[i].Sections {
			if !used[si] {
				sct.Level = 1
				merged = append(merged, sct)
			}
		}
		out[i].Sections = merged
	}
	for i := range out {
		sort.SliceStable(out[i].Sections, func(a, b int) bool {
			sa, sb := out[i].Sections[a], out[i].Sections[b]
			if sa.Page != sb.Page {
				return sa.Page < sb.Page
			}
			if sa.Level != sb.Level {
				return sa.Level < sb.Level
			}
			return sa.Title < sb.Title
		})
	}
	// Described PDFs that no reading points at yet still belong on the shelf.
	for _, b := range books {
		key := b.Module + "/" + b.File
		if _, ok := pdfIdx[key]; ok || !strings.EqualFold(path.Ext(b.File), ".pdf") {
			continue
		}
		lb := libraryBook{ID: slug(key), Kind: "pdf", Title: b.Title, Author: b.Author, Shelf: b.Shelf,
			URL: "/api/modules/" + url.PathEscape(b.Module) + "/files/" + escapePath(b.File), Sections: []librarySection{}}
		for _, t := range b.TOC {
			lb.Sections = append(lb.Sections, librarySection{Title: t.Title, Page: t.Page, EndPage: t.EndPage, Topics: []libraryTopic{}})
		}
		if p := mods.FilePath(b.Module, b.File); p != "" {
			if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() {
				lb.Available = true
			}
		}
		out = append(out, lb)
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].Title < out[b].Title })

	// Online books, from the question banks, in bank order. Only books: a
	// flashcard deck (anki.*) is practice, not a source; a source with no
	// section links is not a book you can open; a source that is one of the
	// PDFs above is that book already; and a module may keep its online
	// sources off the shelves (no_library). Fork change.
	pdfTitles := make([]string, 0, len(out))
	for _, b := range out {
		pdfTitles = append(pdfTitles, normTitle(b.Title))
	}
	sameAsPDF := func(name string) bool {
		n := normTitle(name)
		for _, p := range pdfTitles {
			if p == "" || n == "" {
				continue
			}
			short := 24
			if len(p) < short {
				short = len(p)
			}
			if strings.Contains(n, p[:short]) || strings.Contains(p, n) {
				return true
			}
		}
		return false
	}
	if t, err := pytrack.Default(); err == nil {
		webIdx := map[string]int{}
		var web []libraryBook
		for _, cid := range t.Bank.ConceptIDs() {
			c := t.Bank.Concept(cid)
			lb := label(cid)
			if c == nil || lb == "" || strings.HasPrefix(cid, "anki.") || c.Book.URL == "" {
				continue
			}
			if dc := dag.Concept(cid); dc != nil && mods.LibraryHidden(dc.Domain) {
				continue
			}
			name := bankBookName(c.Book)
			if name == "" || sameAsPDF(name) {
				continue
			}
			bi, ok := webIdx[name]
			if !ok {
				bi = len(web)
				webIdx[name] = bi
				web = append(web, libraryBook{ID: slug(name), Kind: "web", Title: name, Available: true})
			}
			title := c.Book.Title
			if c.Book.Section != "" && !strings.Contains(title, c.Book.Section) {
				title = strings.TrimSpace(c.Book.Section + " " + title)
			}
			sk := name + "|" + title
			si, ok := secIdx[sk]
			if !ok {
				si = len(web[bi].Sections)
				secIdx[sk] = si
				web[bi].Sections = append(web[bi].Sections, librarySection{Title: title, URL: c.Book.URL})
			}
			web[bi].Sections[si].Topics = append(web[bi].Sections[si].Topics, libraryTopic{cid, lb})
		}
		out = append(out, web...)
	}
	for i := range out {
		n := 0
		votes := map[string]int{}
		for si := range out[i].Sections {
			if out[i].Sections[si].Topics == nil {
				out[i].Sections[si].Topics = []libraryTopic{}
			}
		}
		for _, sct := range out[i].Sections {
			n += len(sct.Topics)
			for _, tp := range sct.Topics {
				if c := dag.Concept(tp.ID); c != nil {
					votes[mods.TrackOf(c.Domain)]++
				}
			}
		}
		out[i].Topics = n
		best := ""
		for tr, v := range votes {
			if best == "" || v > votes[best] || (v == votes[best] && tr < best) {
				best = tr
			}
		}
		if best == "" {
			best = "math"
		}
		out[i].Track = best
		if out[i].Sections == nil {
			out[i].Sections = []librarySection{}
		}
	}
	return out
}

// normTitle lowercases a title and keeps only letters and digits, so
// "An Introduction to Statistical Learning (Python)" and "James et al., An
// Introduction to Statistical Learning with Python" compare by content.
func normTitle(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	out := b.String()
	for _, lead := range []string{"anintroductionto", "introductionto", "jamesetal", "the"} {
		out = strings.TrimPrefix(out, lead)
	}
	return out
}

// courseTexts is the authority texts, in priority order, of the courses a
// lesson request comes through (fork change): the `course` query parameter
// (comma-separated) when the learner opened the topic from a course, or else
// the learner's followed courses whose path includes the topic.
func (s *Server) courseTexts(r *http.Request, conceptID string) []string {
	pl := s.eng.GetPlanner()
	if pl == nil {
		return nil
	}
	var ids []string
	if q := r.URL.Query().Get("course"); q != "" {
		ids = engine.ActiveCourses(q)
	} else if sid := s.lessonStudentID(r); sid != "" {
		if st, err := s.repo.GetStudent(sid); err == nil && st != nil {
			for _, id := range engine.ActiveCourses(st.CourseID) {
				if p, err := pl.PathForCourse(id); err == nil {
					for _, c := range p.Concepts {
						if c.ID == conceptID {
							ids = append(ids, id)
							break
						}
					}
				}
			}
		}
	}
	var texts []string
	for _, id := range ids {
		if c := pl.Course(id); c != nil {
			texts = append(texts, c.TextsFor(conceptID)...)
		}
	}
	return texts
}
