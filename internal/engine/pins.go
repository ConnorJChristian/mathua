package engine

import (
	"encoding/json"
	"sort"
	"time"
)

// Fork change: this week's pins. C picks the present week's topics by hand
// and keeps them in circulation; the recommender serves the rest from the
// diagnostics and the course sequencing. Pins live in the student's settings
// JSON under "pins", so they need no migration and travel with the account.

type Pin struct {
	Concept string `json:"concept"`
	Since   string `json:"since"` // date pinned, YYYY-MM-DD
}

type pinSettings struct {
	Pins []Pin `json:"pins"`
}

// Pins lists the learner's pinned topics, oldest first. Unknown concepts are
// dropped.
func (e *Engine) Pins(studentID string) []Pin {
	if e.repo == nil || studentID == "" {
		return nil
	}
	raw, err := e.repo.GetSettings(studentID)
	if err != nil || raw == "" {
		return nil
	}
	var cfg pinSettings
	if json.Unmarshal([]byte(raw), &cfg) != nil {
		return nil
	}
	out := make([]Pin, 0, len(cfg.Pins))
	for _, p := range cfg.Pins {
		if e.dag.Concept(p.Concept) != nil {
			out = append(out, p)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Since < out[j].Since })
	return out
}

// SetPins replaces the pinned set. A topic already pinned keeps its date.
func (e *Engine) SetPins(studentID string, concepts []string) ([]Pin, error) {
	raw, err := e.repo.GetSettings(studentID)
	if err != nil {
		return nil, err
	}
	var all map[string]json.RawMessage
	if raw == "" || json.Unmarshal([]byte(raw), &all) != nil || all == nil {
		all = map[string]json.RawMessage{}
	}
	since := map[string]string{}
	for _, p := range e.Pins(studentID) {
		since[p.Concept] = p.Since
	}
	today := time.Now().Format("2006-01-02")
	var pins []Pin
	seen := map[string]bool{}
	for _, id := range concepts {
		if id == "" || seen[id] || e.dag.Concept(id) == nil {
			continue
		}
		seen[id] = true
		d := since[id]
		if d == "" {
			d = today
		}
		pins = append(pins, Pin{Concept: id, Since: d})
	}
	if pins == nil {
		pins = []Pin{}
	}
	b, _ := json.Marshal(pins)
	all["pins"] = b
	out, err := json.Marshal(all)
	if err != nil {
		return nil, err
	}
	return pins, e.repo.UpdateSettings(studentID, string(out))
}
