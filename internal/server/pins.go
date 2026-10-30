package server

import (
	"net/http"

	"github.com/chuma-beep/mathua/internal/engine"
)

// Fork change: this week's pins. GET /api/pins lists them; PUT {"concepts":
// [...]} replaces the set (a topic already pinned keeps its date). The Study
// tab serves pinned topics right after due reviews, and the course page's
// Modules tab is where they are picked.
func (s *Server) handlePins(w http.ResponseWriter, r *http.Request) {
	studentID, _ := r.Context().Value(authStudentKey{}).(string)
	if studentID == "" {
		writeError(w, "not authenticated", 401)
		return
	}
	switch r.Method {
	case http.MethodGet:
		pins := s.eng.Pins(studentID)
		// The active floor travels with the pins: a pin is how a learner
		// brings back a topic below it (engine/floor.go).
		writeJSON(w, map[string]interface{}{"pins": s.describePins(studentID, pins), "floor": s.eng.Floor(studentID)})
	case http.MethodPut:
		var req struct {
			Concepts []string `json:"concepts"`
		}
		if err := decodeJSON(w, r, &req); err != nil {
			writeError(w, "invalid request", 400)
			return
		}
		pins, err := s.eng.SetPins(studentID, req.Concepts)
		if err != nil {
			writeError(w, "failed to save pins", 500)
			return
		}
		writeJSON(w, map[string]interface{}{"pins": s.describePins(studentID, pins)})
	default:
		http.Error(w, `{"error":"method not allowed"}`, 405)
	}
}

// GET /api/floor: the learner's active floor, the domains whose topics are
// never served to them unless a followed course lists one or they pin it
// (fork change; engine/floor.go). {"domains": [...], "source":
// "courses"|"env"|"none", "scope": "all"|"reviews"|"none"}.
func (s *Server) handleFloor(w http.ResponseWriter, r *http.Request) {
	studentID, _ := r.Context().Value(authStudentKey{}).(string)
	if studentID == "" {
		writeError(w, "not authenticated", 401)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, 405)
		return
	}
	writeJSON(w, s.eng.Floor(studentID))
}

type pinView struct {
	Concept string `json:"concept"`
	Label   string `json:"label"`
	Since   string `json:"since"`
	Done    bool   `json:"done"` // mastered since it was pinned
}

func (s *Server) describePins(studentID string, pins []engine.Pin) []pinView {
	out := make([]pinView, 0, len(pins))
	dag := s.eng.GetDAG()
	progress, _ := s.repo.GetAllProgress(studentID)
	for _, p := range pins {
		v := pinView{Concept: p.Concept, Since: p.Since}
		if c := dag.Concept(p.Concept); c != nil {
			v.Label = c.Label
		}
		if pr := progress[p.Concept]; pr != nil && pr.Status == "MASTERED" {
			v.Done = true
		}
		out = append(out, v)
	}
	return out
}
