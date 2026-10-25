package server

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The local event log is off by default and, when on, appends what it is sent
// to a JSON-lines file in the configured directory and nowhere else.
func TestEventLog(t *testing.T) {
	_, mux := fixedBankServer(t)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/events", nil))
	if !strings.Contains(rec.Body.String(), `"enabled":false`) {
		t.Fatalf("off by default: %s", rec.Body.String())
	}
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("POST", "/api/events", strings.NewReader(`{"events":[{"kind":"x"}]}`)))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("off: POST should be accepted and dropped, got %d", rec.Code)
	}

	dir := t.TempDir()
	t.Setenv("MATHUA_EVENT_LOG_DIR", dir)
	_, mux = fixedBankServer(t)
	body := `{"events":[{"kind":"question_shown","ts":1,"concept":"a","data":{"surface":"learn"}},{"kind":""},{"kind":"page_view","page":"/today"}]}`
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("POST", "/api/events", strings.NewReader(body)))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("on: %d %s", rec.Code, rec.Body.String())
	}
	files, _ := filepath.Glob(filepath.Join(dir, "events-*.jsonl"))
	if len(files) != 1 {
		t.Fatalf("want one monthly file, got %v", files)
	}
	f, _ := os.Open(files[0])
	defer f.Close()
	var kinds []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var m map[string]interface{}
		if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
			t.Fatal(err)
		}
		kinds = append(kinds, m["kind"].(string))
	}
	if strings.Join(kinds, ",") != "question_shown,page_view" {
		t.Fatalf("logged %v (the empty kind must be dropped)", kinds)
	}
}
