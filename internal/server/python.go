package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/chuma-beep/mathua/internal/pytrack"
)

// trackOf reads the track a request is scoped to. The browser sends
// ?track=<id> from a module's side of the site; a request without it is the
// math track, which is what every client did before tracks existed.
func (s *Server) trackOf(r *http.Request) string {
	return s.eng.NormalizeTrack(r.URL.Query().Get("track"))
}

// GET /api/modules/{module}/files/{path} — a file from an enabled module's
// files/ folder, which is where your own PDFs live (modules-local/*/files is
// git-ignored). Readings link here so "Open section" opens your copy at the
// right page. Only regular files under files/ are served.
func (s *Server) handleModuleFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, `{"error":"method not allowed"}`, 405)
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/api/modules/")
	parts := strings.SplitN(rest, "/", 3)
	if len(parts) != 3 || parts[1] != "files" || parts[2] == "" {
		http.NotFound(w, r)
		return
	}
	rel, err := url.PathUnescape(parts[2])
	if err != nil {
		http.NotFound(w, r)
		return
	}
	p := s.eng.Modules().FilePath(parts[0], path.Clean(rel))
	if p == "" {
		http.NotFound(w, r)
		return
	}
	st, err := os.Stat(p)
	if err != nil || !st.Mode().IsRegular() {
		http.NotFound(w, r)
		return
	}
	// Your files, for you: never cached by a shared proxy, never indexed.
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("X-Robots-Tag", "noindex")
	http.ServeFile(w, r, p)
}

// readingText is a concept's reading as a teaching step: what to read, and
// where. PDF, when set, lets the lesson embed the book at that page; Texts
// lists every text the concept is read in, so the lesson can switch between
// them (fork change).
type readingText struct {
	Source, Body string
	PDF          *readingPDF
	Texts        []readingOption
}

// readingPDF is what the lesson's embedded reader needs: the file, the PDF
// page the section starts on, and (when known) the page it ends on.
type readingPDF struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Page    int    `json:"page,omitempty"`
	EndPage int    `json:"end_page,omitempty"`
}

// readingOption is one text a concept can be read in: your own file (a
// module reading) or an online book (a question bank's source). Key matches
// a course's authority texts: "<module>/<file>" or "web:<book name>".
type readingOption struct {
	Key      string `json:"key"`
	Kind     string `json:"kind"` // pdf, file, link or web
	Book     string `json:"book"`
	Source   string `json:"source"`
	URL      string `json:"url,omitempty"`
	Page     int    `json:"page,omitempty"`
	EndPage  int    `json:"end_page,omitempty"`
	Selected bool   `json:"selected,omitempty"`
}

// bankBookName is the name a question bank gives its source.
func bankBookName(b pytrack.Book) string {
	if b.Name == "" && strings.Contains(b.URL, "PythonDataScienceHandbook") {
		return "Python Data Science Handbook"
	}
	return b.Name
}

// textMatches reports whether an authority-text key names this option. A
// "web:" key matches any online book whose name contains it, so a course can
// say "web:Statistical Learning" for every ISLP-based source.
func textMatches(key string, o readingOption) bool {
	if strings.HasPrefix(key, "web:") {
		return o.Kind == "web" && strings.Contains(strings.ToLower(o.Book), strings.ToLower(strings.TrimPrefix(key, "web:")))
	}
	return key == o.Key
}

// readingOptions lists the texts a concept is read in: module readings first,
// then the question bank's own source.
func (s *Server) readingOptions(conceptID string) []readingOption {
	mods := s.eng.Modules()
	titles := map[string]string{}
	for _, b := range mods.Books() {
		titles[b.Module+"/"+b.File] = b.Title
	}
	var out []readingOption
	for _, rd := range mods.ReadingsFor(conceptID) {
		o := readingOption{Source: rd.Source, Page: rd.Page, EndPage: rd.EndPage}
		switch {
		case rd.File != "":
			o.Key = rd.Book()
			o.Book = titles[o.Key]
			if o.Book == "" {
				o.Book = rd.File
			}
			o.URL = "/api/modules/" + url.PathEscape(rd.Module) + "/files/" + escapePath(rd.File)
			o.Kind = "file"
			if strings.EqualFold(path.Ext(rd.File), ".pdf") {
				o.Kind = "pdf"
			}
		case rd.URL != "":
			o.Key, o.Kind, o.Book, o.URL = "link:"+rd.URL, "link", rd.Source, rd.URL
		default:
			continue
		}
		out = append(out, o)
	}
	if t, err := pytrack.Default(); err == nil {
		if c := t.Bank.Concept(conceptID); c != nil {
			if name := bankBookName(c.Book); name != "" {
				src := c.Book.Title
				if c.Book.Section != "" {
					src += ", " + c.Book.Section
				}
				dup := false
				for _, o := range out {
					dup = dup || o.Book == name
				}
				if !dup {
					out = append(out, readingOption{Key: "web:" + name, Kind: "web", Book: name, Source: src, URL: c.Book.URL})
				}
			}
		}
	}
	return out
}

// readingStep is the "read this section first" step for a concept that a
// module gives a reading, whether the concept is a module's or the base
// graph's. With several texts, `texts` (the authority texts of the courses
// the learner reached the topic through, in priority order) picks the one
// shown first; the rest stay one click away.
func (s *Server) readingStep(conceptID string, texts []string) (readingText, bool) {
	if len(s.eng.Modules().ReadingsFor(conceptID)) == 0 {
		return readingText{}, false
	}
	opts := s.readingOptions(conceptID)
	if len(opts) == 0 {
		return readingText{}, false
	}
	pick := 0
	found := false
	for _, t := range texts {
		for i, o := range opts {
			if textMatches(t, o) {
				pick, found = i, true
				break
			}
		}
		if found {
			break
		}
	}
	opts[pick].Selected = true
	o := opts[pick]

	var b strings.Builder
	note := ""
	if rd, ok := s.eng.Modules().Reading(conceptID); ok {
		note = rd.Note
	}
	if note == "" {
		// A module concept's own bank note still teaches; keep it above the reading.
		if t, err := pytrack.Default(); err == nil {
			if c := t.Bank.Concept(conceptID); c != nil {
				note = strings.TrimSpace(c.Intro)
			}
		}
	}
	if note != "" {
		b.WriteString(note + "\n\n")
	}
	var pdf *readingPDF
	switch o.Kind {
	case "pdf":
		fmt.Fprintf(&b, "**Read first:** %s", o.Source)
		link := o.URL
		if o.Page > 0 {
			link += fmt.Sprintf("#page=%d", o.Page)
		}
		pdf = &readingPDF{Title: o.Source, URL: o.URL, Page: o.Page, EndPage: o.EndPage}
		fmt.Fprintf(&b, "\n\nThe section is open below. [Open it in a new tab](%s)", link)
	case "web":
		fmt.Fprintf(&b, "**Read first:** %s, in %s", o.Source, o.Book)
		if o.URL != "" {
			fmt.Fprintf(&b, "\n\n[Open section](%s)", o.URL)
		}
	default:
		fmt.Fprintf(&b, "**Read first:** %s", o.Source)
		link := o.URL
		if o.Page > 0 {
			link += fmt.Sprintf("#page=%d", o.Page)
		}
		fmt.Fprintf(&b, "\n\n[Open section](%s)", link)
	}
	if len(opts) > 1 {
		b.WriteString("\n\nOther texts cover this too: switch below.")
	}
	b.WriteString("\n\nPress Next when you have read it.")
	rt := readingText{Source: o.Source, Body: b.String(), PDF: pdf}
	if len(opts) > 1 {
		rt.Texts = opts
	}
	return rt, true
}

// moduleKPs is the teaching step for a module concept, which has no lesson
// corpus: the question bank's original note, or a pointer to the section of
// your own book to read (a module's readings.json), opened from your files.
func (s *Server) moduleKPs(conceptID, label string, texts []string) []map[string]interface{} {
	step := func(section, body string) []map[string]interface{} {
		return []map[string]interface{}{{
			"label":          label,
			"section":        section,
			"subgoals":       []string{},
			"worked_example": body,
			"assets":         []interface{}{},
		}}
	}
	if rd, ok := s.readingStep(conceptID, texts); ok {
		kp := step(rd.Source, rd.Body)
		if rd.PDF != nil {
			kp[0]["reading"] = rd.PDF
		}
		if rd.Texts != nil {
			kp[0]["texts"] = rd.Texts
		}
		return kp
	}
	t, err := pytrack.Default()
	if err != nil {
		return nil
	}
	c := t.Bank.Concept(conceptID)
	if c == nil {
		return nil
	}
	if strings.TrimSpace(c.Intro) == "" {
		// A bank with no note of its own, such as imported flashcards: say
		// where the cards come from and how the set works.
		src := c.Book.Title
		if c.Book.Name != "" {
			src = c.Book.Name + " › " + c.Book.Title
		}
		return step(c.Book.Title, fmt.Sprintf("**From:** %s\n\n%s", src, cardMix(c.Items)))
	}
	body := c.Intro
	section := c.Book.Title
	if c.Book.URL != "" {
		name := c.Book.Name
		if name == "" {
			name = "Python Data Science Handbook"
		}
		body += "\n\nRead more: [" + name + " §" + c.Book.Section + ", " + c.Book.Title + "](" + c.Book.URL + ")"
		section = "§" + c.Book.Section + " " + c.Book.Title
	}
	return step(section, body)
}

func escapePath(p string) string {
	segs := strings.Split(p, "/")
	for i, s := range segs {
		segs[i] = url.PathEscape(s)
	}
	return strings.Join(segs, "/")
}

// maxRunCode bounds what the Run button may send. Practice tasks are a few
// lines; the cap only exists so a paste of a whole notebook is refused early.
const maxRunCode = 16 * 1024

// POST /api/python/run  {"code": "...", "task": "<served answer key>"}
//
// Runs learner code without grading it, for the code box's Run button: the
// task's hidden fixture first (so the variables the prompt promises exist),
// then the code, returning printed output, a trimmed traceback and any
// matplotlib figures as PNG. Nothing is recorded: running is not answering.
func (s *Server) handlePythonRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, 405)
		return
	}
	var req struct {
		Code string `json:"code"`
		Task string `json:"task"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, "invalid request", 400)
		return
	}
	if len(req.Code) > maxRunCode {
		writeError(w, "code too long", 400)
		return
	}
	t, err := pytrack.Default()
	if err != nil {
		writeError(w, "python track unavailable", 503)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(t.Runner.TimeLimit())*time.Second+t.Runner.WallExtra+5*time.Second)
	defer cancel()
	res, err := t.RunCode(ctx, req.Task, req.Code)
	if err != nil {
		if errors.Is(err, pytrack.ErrUnavailable) {
			writeError(w, "The Python runner is unavailable on this server: "+err.Error(), 503)
			return
		}
		writeError(w, err.Error(), 500)
		return
	}
	figures := res.Figures
	if figures == nil {
		figures = []string{}
	}
	writeJSON(w, map[string]interface{}{
		"stdout":           res.Stdout,
		"stdout_truncated": res.StdoutTruncated,
		"error":            res.Error,
		"figures":          figures,
		"timed_out":        res.TimedOut,
		"time_limit":       t.Runner.TimeLimit(),
	})
}

// cardMix says how many cards a flashcard set has and how each kind is
// answered: "57 cards: 41 you type, 2 you choose, 14 you flip."
func cardMix(items []pytrack.Item) string {
	var typed, choice, flip, other int
	for _, it := range items {
		switch it.Kind {
		case pytrack.KindMath:
			typed++
		case pytrack.KindChoice:
			choice++
		case pytrack.KindRecall:
			flip++
		default:
			other++
		}
	}
	noun := func(n int) string {
		if n == 1 {
			return "1 card"
		}
		return fmt.Sprintf("%d cards", n)
	}
	var parts []string
	if typed > 0 {
		parts = append(parts, fmt.Sprintf("%d you type", typed))
	}
	if choice > 0 {
		parts = append(parts, fmt.Sprintf("%d you choose", choice))
	}
	if flip > 0 {
		parts = append(parts, fmt.Sprintf("%d you flip", flip))
	}
	out := noun(len(items)) + "."
	if len(parts) > 1 || (len(parts) == 1 && other > 0) {
		out = noun(len(items)) + ": " + strings.Join(parts, ", ") + "."
	}
	var how []string
	if typed > 0 {
		how = append(how, "Type an answer and Mathua checks it; any equivalent form counts.")
	}
	if flip > 0 {
		how = append(how, "For a flip card, answer in your head, reveal the back, then say whether you had it.")
	}
	if len(how) > 0 {
		out += " " + strings.Join(how, " ")
	}
	return out
}
