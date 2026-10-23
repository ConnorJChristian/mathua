package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/chuma-beep/mathua/internal/pytrack"
)

// trackOf reads the track a request is scoped to. The browser sends
// ?track=python from the Python side of the site; a request without it is the
// Math track, which is what every client did before the Python track existed.
func trackOf(r *http.Request) string {
	return pytrack.NormalizeTrack(r.URL.Query().Get("track"))
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

// pythonKPs is the teaching step for a Python concept: the bank's original
// note and a link to the matching book section. Python concepts have no
// lesson corpus, so without this the Learn page would open straight onto a
// question.
func pythonKPs(conceptID, label string) []map[string]interface{} {
	t, err := pytrack.Default()
	if err != nil {
		return nil
	}
	c := t.Bank.Concept(conceptID)
	if c == nil {
		return nil
	}
	body := c.Intro + "\n\nRead more: [Python Data Science Handbook §" + c.Book.Section + ", " + c.Book.Title + "](" + c.Book.URL + ")"
	return []map[string]interface{}{{
		"label":          label,
		"section":        "§" + c.Book.Section + " " + c.Book.Title,
		"subgoals":       []string{},
		"worked_example": body,
		"assets":         []interface{}{},
		"book_url":       c.Book.URL,
	}}
}
