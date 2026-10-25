package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Local event log (fork change, opt-in).
//
// A self-hosted install can keep a record of how its learner uses the app
// beyond the attempts table: questions shown but not answered, readings
// opened, "Practice anyway", pins, queue clicks, time to first keystroke,
// time away from the tab. It exists for the install's own analysis (a
// learner profile for simulated QA runs, grader near-misses, friction), so it
// is off unless MATHUA_EVENT_LOG_DIR names a directory, and it never leaves
// that directory: events are appended as JSON lines to
// <dir>/events-YYYY-MM.jsonl. Nothing is sent anywhere.
//
// POST /api/events  {"events":[{"kind":"...","ts":<ms>,"concept":"...","page":"...","data":{...}}]}
// GET  /api/events  {"enabled":true|false}

const (
	maxEventsPerBatch = 200
	maxEventKindLen   = 48
)

type clientEvent struct {
	Kind    string                 `json:"kind"`
	TS      int64                  `json:"ts"`
	Concept string                 `json:"concept,omitempty"`
	Page    string                 `json:"page,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

type eventLog struct {
	dir string
	mu  sync.Mutex
}

func newEventLog() *eventLog {
	dir := strings.TrimSpace(os.Getenv("MATHUA_EVENT_LOG_DIR"))
	if dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil
	}
	return &eventLog{dir: dir}
}

func (l *eventLog) append(studentID string, evs []clientEvent) error {
	now := time.Now().UTC()
	l.mu.Lock()
	defer l.mu.Unlock()
	f, err := os.OpenFile(filepath.Join(l.dir, "events-"+now.Format("2006-01")+".jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, e := range evs {
		kind := strings.TrimSpace(e.Kind)
		if kind == "" || len(kind) > maxEventKindLen {
			continue
		}
		rec := map[string]interface{}{
			"at":      now.Format(time.RFC3339Nano),
			"student": studentID,
			"kind":    kind,
		}
		if e.TS > 0 {
			rec["client_ts"] = e.TS
		}
		if e.Concept != "" {
			rec["concept"] = e.Concept
		}
		if e.Page != "" {
			rec["page"] = e.Page
		}
		if len(e.Data) > 0 {
			rec["data"] = e.Data
		}
		if err := enc.Encode(rec); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, map[string]bool{"enabled": s.events != nil})
		return
	case http.MethodPost:
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	if s.events == nil {
		// Off: accepted and dropped, so a client never retries.
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var body struct {
		Events []clientEvent `json:"events"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if len(body.Events) > maxEventsPerBatch {
		body.Events = body.Events[:maxEventsPerBatch]
	}
	studentID, _ := r.Context().Value(authStudentKey{}).(string)
	if err := s.events.append(studentID, body.Events); err != nil {
		writeError(w, "could not write events", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
