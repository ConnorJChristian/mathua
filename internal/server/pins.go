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
		writeJSON(w, map[string]interface{}{"pins": s.describePins(studentID, pins)})
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
