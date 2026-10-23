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

// moduleKPs is the teaching step for a module concept, which has no lesson
// corpus: the question bank's original note, or a pointer to the section of
// your own book to read (a module's readings.json), opened from your files.
func (s *Server) moduleKPs(conceptID, label string) []map[string]interface{} {
	step := func(section, body string) []map[string]interface{} {
		return []map[string]interface{}{{
			"label":          label,
			"section":        section,
			"subgoals":       []string{},
			"worked_example": body,
			"assets":         []interface{}{},
		}}
	}
	if rd, ok := s.eng.Modules().Reading(conceptID); ok {
		var b strings.Builder
		if rd.Note != "" {
			b.WriteString(rd.Note + "\n\n")
		}
		fmt.Fprintf(&b, "**Read first:** %s", rd.Source)
		switch {
		case rd.File != "":
			link := "/api/modules/" + url.PathEscape(rd.Module) + "/files/" + escapePath(rd.File)
			if rd.Page > 0 {
				link += fmt.Sprintf("#page=%d", rd.Page)
			}
			fmt.Fprintf(&b, "\n\n[Open section](%s)", link)
		case rd.URL != "":
			fmt.Fprintf(&b, "\n\n[Open section](%s)", rd.URL)
		}
		b.WriteString("\n\nPress Next when you have read it.")
		return step(rd.Source, b.String())
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
		recall := 0
		for _, it := range c.Items {
			if it.Kind == pytrack.KindRecall {
				recall++
			}
		}
		body := fmt.Sprintf("**From:** %s\n\n%d cards.", src, len(c.Items))
		if recall > 0 {
			body += " For each card, answer in your head, reveal the back, then say whether you had it."
		}
		return step(c.Book.Title, body)
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
