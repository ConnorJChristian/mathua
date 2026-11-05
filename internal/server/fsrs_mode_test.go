package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// GET /api/config says which FSRS mode the server runs (MATHUA_FSRS). With
// the flag unset it is "off": the default until Phase 3.
func TestConfig_ExposesFSRSMode(t *testing.T) {
	_, mux, _ := guestServer(t)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/config", nil))
	var cfg map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &cfg); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if cfg["fsrs_mode"] != "off" {
		t.Errorf("fsrs_mode = %v, want \"off\" (the default; the suite runs with MATHUA_FSRS unset)", cfg["fsrs_mode"])
	}
}
