package server

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// http.FileServer's error path deletes Content-Encoding (Go 1.23 and later),
// while GzipMiddleware set it once, before the handler ran, and went on
// gzipping the body. A browser asking for a page the export does not have
// (/home, any mistyped path) got gzip bytes labelled as plain text and showed
// them as garbage.
func TestGzipSurvivesFileServerError(t *testing.T) {
	h := GzipMiddleware(http.FileServer(http.Dir(t.TempDir())))
	req := httptest.NewRequest(http.MethodGet, "/home", nil)
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q on a gzipped body", got)
	}
	zr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(zr)
	if string(body) != "404 page not found\n" {
		t.Errorf("body = %q", body)
	}
}

// The ordinary path still compresses and still says so.
func TestGzipPlainWrite(t *testing.T) {
	h := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatal("no Content-Encoding on a gzipped body")
	}
	zr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(zr)
	if string(body) != `{"ok":true}` {
		t.Errorf("body = %q", body)
	}
}
