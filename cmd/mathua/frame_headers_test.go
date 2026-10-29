package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Lecture notes are module files shown in an iframe on the app's own origin,
// so those responses may be framed by the same origin; every other response
// stays unframeable.
func TestModuleFilesMayBeFramedBySameOriginOnly(t *testing.T) {
	h := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	cases := map[string]string{
		"/api/modules/ma-program/files/lectures/econ5283/pie.html": "SAMEORIGIN",
		"/api/modules/hegemony/files/hss/c1-1.html":               "SAMEORIGIN",
		"/api/modules/ma-program/readings":                        "DENY",
		"/api/courses/view":                                       "DENY",
		"/learn":                                                  "DENY",
	}
	for path, want := range cases {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if got := rec.Header().Get("X-Frame-Options"); got != want {
			t.Errorf("%s: X-Frame-Options = %q, want %q", path, got, want)
		}
	}
}
