package server

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// GET /api/floor reports the learner's active floor, and GET /api/pins
// carries it beside the pins (fork change; engine/floor.go).
func TestFloorEndpoint(t *testing.T) {
	_, mux, _ := guestServer(t)
	token, _ := quizGuest(t, mux)
	get := func(path, tok string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", path, nil)
		if tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}
	if rec := get("/api/floor", ""); rec.Code != 401 {
		t.Errorf("anonymous: %d, want 401", rec.Code)
	}

	t.Setenv("MATHUA_REVIEW_FLOOR_DOMAINS", "arithmetic, fractions")
	rec := get("/api/floor", token)
	if rec.Code != 200 {
		t.Fatalf("floor: %d %s", rec.Code, rec.Body.String())
	}
	var f struct {
		Domains []string `json:"domains"`
		Source  string   `json:"source"`
		Scope   string   `json:"scope"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &f); err != nil {
		t.Fatal(err)
	}
	// A learner on no course: the environment floor limits reviews only.
	if f.Source != "env" || f.Scope != "reviews" || len(f.Domains) != 2 || f.Domains[0] != "arithmetic" {
		t.Errorf("floor = %+v", f)
	}

	t.Setenv("MATHUA_REVIEW_FLOOR_DOMAINS", "")
	var pins struct {
		Floor struct {
			Source string `json:"source"`
			Scope  string `json:"scope"`
		} `json:"floor"`
	}
	rec = get("/api/pins", token)
	if err := json.Unmarshal(rec.Body.Bytes(), &pins); err != nil || pins.Floor.Source != "none" || pins.Floor.Scope != "none" {
		t.Errorf("pins floor = %+v (%s)", pins.Floor, rec.Body.String())
	}
}
