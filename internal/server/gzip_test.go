package server

import (
	"bytes"
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
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

// A module file is served as it is on disk: a PDF (or an image) is already
// compressed, and gzipping it cost seconds per open, dropped the
// Content-Length pdf.js needs for progress and range loading, and wrapped
// ranged replies whose Content-Range no longer described the body.
func TestGzipLeavesCompressedFilesAlone(t *testing.T) {
	dir := t.TempDir()
	pdf := []byte("%PDF-1.4\n" + strings.Repeat("0123456789", 1000))
	if err := os.WriteFile(filepath.Join(dir, "book.pdf"), pdf, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "page.html"), []byte("<p>"+strings.Repeat("hello ", 200)+"</p>"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := GzipMiddleware(http.FileServer(http.Dir(dir)))
	get := func(method, path string, hdr map[string]string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	rec := get(http.MethodGet, "/book.pdf", nil)
	if rec.Code != 200 || rec.Header().Get("Content-Encoding") != "" {
		t.Fatalf("pdf: status %d, Content-Encoding %q; want raw bytes", rec.Code, rec.Header().Get("Content-Encoding"))
	}
	if got := rec.Header().Get("Content-Length"); got != strconv.Itoa(len(pdf)) {
		t.Errorf("pdf Content-Length = %q, want %d", got, len(pdf))
	}
	if !bytes.Equal(rec.Body.Bytes(), pdf) {
		t.Error("pdf body was altered")
	}
	if rec.Header().Get("Accept-Ranges") != "bytes" {
		t.Error("pdf lost Accept-Ranges")
	}

	rec = get(http.MethodGet, "/book.pdf", map[string]string{"Range": "bytes=0-99"})
	if rec.Code != http.StatusPartialContent || rec.Header().Get("Content-Encoding") != "" {
		t.Fatalf("range: status %d, Content-Encoding %q", rec.Code, rec.Header().Get("Content-Encoding"))
	}
	if rec.Body.Len() != 100 || rec.Header().Get("Content-Length") != "100" || !bytes.Equal(rec.Body.Bytes(), pdf[:100]) {
		t.Errorf("range body: %d bytes, Content-Length %q", rec.Body.Len(), rec.Header().Get("Content-Length"))
	}

	rec = get(http.MethodHead, "/book.pdf", nil)
	if rec.Header().Get("Content-Encoding") != "" || rec.Header().Get("Content-Length") != strconv.Itoa(len(pdf)) {
		t.Errorf("HEAD: Content-Encoding %q, Content-Length %q", rec.Header().Get("Content-Encoding"), rec.Header().Get("Content-Length"))
	}

	// Text is still compressed, and says so for caches.
	rec = get(http.MethodGet, "/page.html", nil)
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("html: Content-Encoding %q, want gzip", rec.Header().Get("Content-Encoding"))
	}
	if !strings.Contains(rec.Header().Get("Vary"), "Accept-Encoding") {
		t.Error("html: no Vary: Accept-Encoding")
	}
	zr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(zr)
	if !strings.HasPrefix(string(body), "<p>hello") {
		t.Errorf("html body = %.20q", body)
	}
}

func TestCompressible(t *testing.T) {
	for ct, want := range map[string]bool{
		"application/json": true, "text/html; charset=utf-8": true, "text/javascript": true,
		"image/svg+xml": true, "application/manifest+json": true,
		"application/pdf": false, "image/png": false, "image/jpeg": false, "font/woff2": false,
		"video/mp4": false, "application/zip": false, "application/octet-stream": false,
	} {
		if got := compressible(ct); got != want {
			t.Errorf("compressible(%q) = %v, want %v", ct, got, want)
		}
	}
}

// Q65: every 304 from the static file server logged "warning: gzip close:
// http: request method or response status code does not allow body",
// because the gzip footer was written to a reply that cannot have a body.
func TestGzipNotModifiedHasNoBodyAndNoWarning(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.js"), []byte(strings.Repeat("let a = 1;\n", 200)), 0o644); err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(prev)

	srv := httptest.NewServer(GzipMiddleware(http.FileServer(http.Dir(dir))))
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/app.js", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("If-Modified-Since", time.Now().Add(time.Hour).UTC().Format(http.TimeFormat))
	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNotModified {
		t.Fatalf("status %d, want 304", res.StatusCode)
	}
	if ce := res.Header.Get("Content-Encoding"); ce != "" {
		t.Errorf("304 carries Content-Encoding %q", ce)
	}
	if strings.Contains(logs.String(), "gzip close") {
		t.Errorf("304 logged a gzip warning: %s", logs.String())
	}
}
