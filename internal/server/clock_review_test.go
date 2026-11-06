package server

// FSRS Phase 2 review, L4: the course page and the diagnostic attempt read
// the time through internal/clock, so a replay or a test on a simulated
// clock sees the same "now" the engine does. Each assertion failed while
// these read the wall clock.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/auth"
	"github.com/chuma-beep/mathua/internal/clock"
	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/engine"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/planning"
	"github.com/chuma-beep/mathua/internal/storage"
)

func TestClock_CourseViewDueAndDaysLeft(t *testing.T) {
	simNow := time.Date(2026, 11, 26, 17, 0, 0, 0, time.UTC)
	defer clock.NewSim(simNow).Install()()
	th := concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 60}
	d, _ := concepts.Build([]concepts.Concept{
		{ID: "a", Label: "A", Domain: "d", GradingType: "numeric", MasteryThreshold: th},
		{ID: "b", Label: "B", Domain: "d", GradingType: "numeric", Prerequisites: []string{"a"}, MasteryThreshold: th},
	})
	store, _ := storage.NewSQLiteStore(":memory:")
	t.Cleanup(func() { store.Close() })
	reg := generator.NewRegistry()
	for _, id := range []string{"a", "b"} {
		reg.Register(id, &testGen{})
	}
	cp := filepath.Join(t.TempDir(), "courses.json")
	os.WriteFile(cp, []byte(`[{"id":"plan.t","name":"T","term":"Fall","due":"2026-12-06","plan":["a","b"],
		"modules":[{"title":"Unit 1","topics":["a","b"]}]}]`), 0o644)
	pl, err := planning.Load(cp, d)
	if err != nil {
		t.Fatal(err)
	}
	s := New(engine.New(store, d, reg, nil, pl), store, auth.New(store))
	mux := http.NewServeMux()
	s.Register(mux)
	token, sid := guestTokenAndID(t, mux)
	mastered := simNow.Add(-10 * 24 * time.Hour)
	due := simNow.Add(-time.Hour) // due on the simulated clock, weeks ahead of the wall clock
	for _, cid := range []string{"a", "b"} {
		store.UpsertProgress(&storage.ConceptProgress{StudentID: sid, ConceptID: cid, Status: "MASTERED", Attempts: 5, Streak: 5,
			MasteredAt: &mastered, LastAttempted: &mastered, LastReviewed: &mastered, NextReviewDue: &due, SM2EFactor: 2.5, SM2Interval: 9})
	}
	rec := authedGet(t, mux, "/api/courses/view?id=plan.t", token)
	if rec.Code != 200 {
		t.Fatalf("view: %d %s", rec.Code, rec.Body.String())
	}
	var v courseView
	json.Unmarshal(rec.Body.Bytes(), &v)
	if v.DaysLeft == nil || *v.DaysLeft != 9 {
		t.Errorf("days left %v, want 9 (2026-12-06 from the simulated 2026-11-26 17:00Z)", v.DaysLeft)
	}
	if len(v.Modules) != 1 || len(v.Modules[0].Topics) != 2 || !v.Modules[0].Topics[1].Due {
		t.Errorf("b is due on the simulated clock: %+v", v.Modules)
	}
}

func TestClock_DiagnosticAttemptIsTheClocksUTC(t *testing.T) {
	simNow := time.Date(2026, 11, 26, 17, 0, 7, 0, time.UTC)
	defer clock.NewSim(simNow).Install()()
	d, _ := concepts.Build([]concepts.Concept{
		{ID: "a", Label: "A", Domain: "d", GradingType: "numeric", Prerequisites: []string{},
			MasteryThreshold: concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 60}},
	})
	store, _ := storage.NewSQLiteStore(":memory:")
	t.Cleanup(func() { store.Close() })
	reg := generator.NewRegistry()
	reg.Register("a", &testGen{})
	srv := New(engine.New(store, d, reg, nil, planning.New(d)), store, auth.New(store))
	mux := http.NewServeMux()
	srv.Register(mux)
	token, sid := quizGuest(t, mux)
	post := func(path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", path, bytes.NewReader([]byte(body)))
		req.Header.Set("Authorization", "Bearer "+token)
		mux.ServeHTTP(rec, req)
		return rec
	}
	start := post("/api/goal/diagnostic", `{"concept_ids":["a"]}`)
	var sres struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(start.Body.Bytes(), &sres); err != nil || sres.SessionID == "" {
		t.Fatalf("diag start: %d %s", start.Code, start.Body.String())
	}
	if ans := post("/api/goal/diagnostic/answer", `{"session_id":"`+sres.SessionID+`","concept_id":"a","answer":"4","elapsed":5}`); ans.Code != 200 {
		t.Fatalf("answer: %d %s", ans.Code, ans.Body.String())
	}
	atts, err := store.GetAttemptsForStudent(sid)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, a := range atts {
		if a.Source != "diagnostic" {
			continue
		}
		n++
		if !a.Timestamp.Equal(simNow) || a.Timestamp.Location() != time.UTC {
			t.Errorf("diagnostic attempt at %v, want the clock's %v in UTC", a.Timestamp, simNow)
		}
	}
	if n != 1 {
		t.Fatalf("%d diagnostic attempts", n)
	}
	ses, err := store.GetSession(sres.SessionID)
	if err != nil || ses == nil || !ses.StartedAt.Equal(simNow) {
		t.Errorf("EnsureSession started_at %+v (%v), want the clock's time", ses, err)
	}
}
