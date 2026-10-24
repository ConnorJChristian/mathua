package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/auth"
	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/engine"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/planning"
	"github.com/chuma-beep/mathua/internal/storage"
)

// Phase 1 journey tests: the complete post-diagnostic learner path over HTTP,
// reusing the production endpoints. Tests only — no production changes. Each
// test covers one link of the chain in dependency order (G1..G5); a failure
// stops the story at a real defect rather than a test bug.

// journeyDiagnostic runs a full guest diagnostic over HTTP on the two-concept
// DAG, answering every question correctly and fast, and returns the student
// and session ids once the report marks completion.
func journeyDiagnostic(t *testing.T, mux *http.ServeMux) (studentID, sessionID string) {
	t.Helper()
	code, start := postJSON(t, mux, "/api/goal/diagnostic", `{"name":"journey","concept_ids":["a","b"]}`)
	if code != 200 {
		t.Fatalf("diag start: %d %v", code, start)
	}
	sid, _ := start["student_id"].(string)
	dsid, _ := start["session_id"].(string)
	cid, _ := start["concept_id"].(string)
	if sid == "" || dsid == "" {
		t.Fatalf("diag start missing ids: %v", start)
	}
	done := false
	for i := 0; i < 60 && !done; i++ {
		if cid == "" {
			t.Fatalf("diagnostic stalled without concept at answer %d", i)
		}
		body := fmt.Sprintf(`{"session_id":"%s","concept_id":"%s","answer":"4","elapsed":1.0}`, dsid, cid)
		var ans map[string]interface{}
		code, ans = postJSON(t, mux, "/api/goal/diagnostic/answer", body)
		if code != 200 {
			t.Fatalf("diag answer %d: %d %v", i, code, ans)
		}
		done, _ = ans["done"].(bool)
		cid, _ = ans["concept_id"].(string)
	}
	if !done {
		t.Fatal("diagnostic did not complete")
	}
	return sid, dsid
}

// G1 — completing a diagnostic then POST /api/goal/plan persists placement
// rows without granting mastery and marks the diagnostic complete.
func TestJourney_PlanPersistsPlacementWithoutMastery(t *testing.T) {
	s, mux := twoConceptServer(t)
	sid, dsid := journeyDiagnostic(t, mux)

	code, plan := postJSON(t, mux, "/api/goal/plan", `{"session_id":"`+dsid+`"}`)
	if code != 200 {
		t.Fatalf("plan: %d %v", code, plan)
	}
	prog, err := s.repo.GetAllProgress(sid)
	if err != nil {
		t.Fatalf("get progress: %v", err)
	}
	if len(prog) == 0 {
		t.Fatal("plan persisted no placement rows")
	}
	seeded := 0
	for cid, p := range prog {
		if p.Status == "MASTERED" {
			t.Errorf("placement wrote mastery for %s", cid)
		}
		if p.PlacementSeeded {
			seeded++
		}
	}
	if seeded == 0 {
		t.Error("no PlacementSeeded rows persisted")
	}
	st, err := s.repo.GetStudent(sid)
	if err != nil || st == nil || !st.DiagnosticCompleted {
		t.Fatalf("diagnostic not marked completed: %+v err=%v", st, err)
	}
	// The consumed session is gone: planning twice is a 404, not a re-seed.
	if code, _ := postJSON(t, mux, "/api/goal/plan", `{"session_id":"`+dsid+`"}`); code != 404 {
		t.Errorf("second plan = %d, want 404", code)
	}
}

// journeyAuthedServer builds the two-concept DAG with planner and auth so an
// authenticated guest can run the full diagnostic→plan→next chain.
func journeyAuthedServer(t *testing.T) (*http.ServeMux, storage.Repository) {
	t.Helper()
	d, _ := concepts.Build([]concepts.Concept{
		{ID: "a", Label: "A", Domain: "d", GradingType: "numeric", Prerequisites: []string{},
			MasteryThreshold: concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 60}},
		{ID: "b", Label: "B", Domain: "d", GradingType: "numeric", Prerequisites: []string{"a"},
			MasteryThreshold: concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 60}},
	})
	store, _ := storage.NewSQLiteStore(":memory:")
	t.Cleanup(func() { store.Close() })
	reg := generator.NewRegistry()
	reg.Register("a", &testGen{})
	reg.Register("b", &testGen{})
	s := New(engine.New(store, d, reg, nil, planning.New(d)), store, auth.New(store))
	mux := http.NewServeMux()
	s.Register(mux)
	return mux, store
}

func journeyPost(t *testing.T, mux *http.ServeMux, token, path, body string) (int, map[string]interface{}) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", path, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	mux.ServeHTTP(rec, req)
	var res map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	return rec.Code, res
}

func journeyGet(t *testing.T, mux *http.ServeMux, token, path string) (int, map[string]interface{}) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	mux.ServeHTTP(rec, req)
	var res map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	return rec.Code, res
}

// G2 — after plan-apply, /api/next offers the placed frontier (never the
// diagnostic again), and readiness plus practice agree on it.
func TestJourney_NextServesPlacedFrontier(t *testing.T) {
	mux, _ := journeyAuthedServer(t)
	token, _ := guestTokenAndID(t, mux)

	code, start := journeyPost(t, mux, token, "/api/goal/diagnostic", `{"concept_ids":["a","b"]}`)
	if code != 200 {
		t.Fatalf("diag start: %d %v", code, start)
	}
	dsid, _ := start["session_id"].(string)
	cid, _ := start["concept_id"].(string)
	done := false
	for i := 0; i < 60 && !done; i++ {
		code, ans := journeyPost(t, mux, token, "/api/goal/diagnostic/answer",
			fmt.Sprintf(`{"session_id":"%s","concept_id":"%s","answer":"4","elapsed":1.0}`, dsid, cid))
		if code != 200 {
			t.Fatalf("diag answer %d: %d %v", i, code, ans)
		}
		done, _ = ans["done"].(bool)
		cid, _ = ans["concept_id"].(string)
		if !done && cid == "" {
			t.Fatalf("diagnostic stalled at answer %d", i)
		}
	}
	if code, plan := journeyPost(t, mux, token, "/api/goal/plan", `{"session_id":"`+dsid+`"}`); code != 200 {
		t.Fatalf("plan: %d %v", code, plan)
	}

	code, next := journeyGet(t, mux, token, "/api/next")
	if code != 200 {
		t.Fatalf("next: %d %v", code, next)
	}
	primary, _ := next["primary"].(map[string]interface{})
	if primary == nil {
		t.Fatalf("next has no primary: %v", next)
	}
	action, _ := primary["action"].(map[string]interface{})
	href, _ := action["href"].(string)
	if !strings.HasPrefix(href, "/learn?concept=") {
		t.Fatalf("primary href = %q, want a /learn?concept= task (not diagnostic/onboard)", href)
	}
	concept := strings.TrimPrefix(href, "/learn?concept=")

	code, ready := journeyGet(t, mux, token, "/api/lessons/"+concept+"/readiness")
	if code != 200 {
		t.Fatalf("readiness: %d %v", code, ready)
	}
	if eligible, _ := ready["eligible"].(bool); !eligible {
		t.Fatalf("next task %q not eligible: %v", concept, ready)
	}
	code, _ = journeyGet(t, mux, token, "/api/lessons/"+concept+"/practice?count=1")
	if code != 200 {
		t.Fatalf("practice on next task: %d, want 200", code)
	}
}

// G3 — refusal flips to eligibility through demonstrated mastery: a fresh
// learner is refused on b, answers a to mastery over HTTP, then readiness
// reports eligible and practice serves.
func TestJourney_RefusalFlipsToEligibleAfterMastery(t *testing.T) {
	mux, _ := gatedServer(t)
	token, _ := guestTokenAndID(t, mux)

	if rec := authedGet(t, mux, "/api/lessons/b/readiness", token); rec.Code != 200 {
		t.Fatalf("readiness: %d", rec.Code)
	} else {
		var body struct {
			Eligible bool `json:"eligible"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.Eligible {
			t.Fatal("fresh learner eligible on b before demonstrating a")
		}
	}

	// testGen serves one unique question per fetch (identical texts dedupe,
	// by design — real generators vary); answer it repeatedly to the floor.
	answered := 0
	for round := 0; round < 12 && answered < 8; round++ {
		code, prac := journeyGet(t, mux, token, "/api/lessons/a/practice?count=5")
		if code != 200 {
			t.Fatalf("practice a: %d %v", code, prac)
		}
		qs, _ := prac["questions"].([]interface{})
		if len(qs) == 0 {
			t.Fatalf("practice a served no questions: %v", prac)
		}
		for _, qi := range qs {
			q, _ := qi.(map[string]interface{})
			text, _ := q["question"].(string)
			code, ans := journeyPost(t, mux, token, "/api/study/answer",
				fmt.Sprintf(`{"concept_id":"a","answer":"4","elapsed":3.0,"question":%q}`, text))
			if code != 200 {
				t.Fatalf("study answer: %d %v", code, ans)
			}
			if correct, _ := ans["correct"].(bool); !correct {
				t.Fatalf("expected correct grade, got %v", ans)
			}
			answered++
			if answered >= 8 {
				break
			}
		}
	}
	if answered < 8 {
		t.Fatalf("only %d answers recorded, need 8 for the mastery floor", answered)
	}

	code, ready := journeyGet(t, mux, token, "/api/lessons/b/readiness")
	if code != 200 {
		t.Fatalf("readiness: %d %v", code, ready)
	}
	if eligible, _ := ready["eligible"].(bool); !eligible {
		t.Fatalf("still ineligible after mastering a: %v", ready)
	}
	if code, _ := journeyGet(t, mux, token, "/api/lessons/b/practice?count=1"); code != 200 {
		t.Fatalf("practice b after mastery: %d, want 200", code)
	}
}

// G4 — study answers move HTTP-visible progress: three correct answers take
// a from UNSEEN to PRACTICING as observed through GET /api/progress.
func TestJourney_AnswersAdvanceProgressOverHTTP(t *testing.T) {
	mux, _ := gatedServer(t)
	token, sid := guestTokenAndID(t, mux)

	code, before := journeyGet(t, mux, token, "/api/progress/"+sid)
	if code != 200 {
		t.Fatalf("progress: %d %v", code, before)
	}
	if _, ok := before["a"]; ok {
		t.Fatalf("concept a already has progress before any answer: %v", before["a"])
	}

	for i := 0; i < 3; i++ {
		code, prac := journeyGet(t, mux, token, "/api/lessons/a/practice?count=1")
		if code != 200 {
			t.Fatalf("practice: %d %v", code, prac)
		}
		qs, _ := prac["questions"].([]interface{})
		if len(qs) == 0 {
			t.Fatalf("no questions served: %v", prac)
		}
		text, _ := qs[0].(map[string]interface{})["question"].(string)
		code, ans := journeyPost(t, mux, token, "/api/study/answer",
			fmt.Sprintf(`{"concept_id":"a","answer":"4","elapsed":3.0,"question":%q}`, text))
		if code != 200 {
			t.Fatalf("study answer %d: %d %v", i, code, ans)
		}
	}

	code, after := journeyGet(t, mux, token, "/api/progress/"+sid)
	if code != 200 {
		t.Fatalf("progress: %d %v", code, after)
	}
	row, _ := after["a"].(map[string]interface{})
	if row == nil {
		t.Fatalf("no progress row for a after 3 correct answers: %v", after)
	}
	if status, _ := row["status"].(string); status != "PRACTICING" {
		t.Errorf("a.status = %q, want PRACTICING", status)
	}
	if attempts, _ := row["attempts"].(float64); attempts != 3 {
		t.Errorf("a.attempts = %v, want 3", attempts)
	}
	// Streak is 2, not 3: the streak resets to 1 on each tier advance
	// (UNSEEN→LEARNING→PRACTICING) because it no longer gates mastery —
	// evidence does. Pin the rule rather than assuming streak==answers.
	if streak, _ := row["streak"].(float64); streak != 2 {
		t.Errorf("a.streak = %v, want 2 (reset on tier advance)", streak)
	}
}

// G5 — leaving mid-lesson and returning corrupts nothing: re-fetching
// practice creates no duplicate sessions or attempts, and a previously
// served question still grades. Each submit records exactly one attempt.
func TestJourney_LeaveAndReturnWithoutDuplication(t *testing.T) {
	mux, store := gatedServer(t)
	token, sid := guestTokenAndID(t, mux)

	answer := func(question string) map[string]interface{} {
		t.Helper()
		code, ans := journeyPost(t, mux, token, "/api/study/answer",
			fmt.Sprintf(`{"concept_id":"a","answer":"4","elapsed":3.0,"question":%q}`, question))
		if code != 200 {
			t.Fatalf("study answer: %d %v", code, ans)
		}
		return ans
	}
	serveOne := func() string {
		t.Helper()
		code, prac := journeyGet(t, mux, token, "/api/lessons/a/practice?count=5")
		if code != 200 {
			t.Fatalf("practice: %d %v", code, prac)
		}
		qs, _ := prac["questions"].([]interface{})
		if len(qs) == 0 {
			t.Fatalf("no questions served: %v", prac)
		}
		text, _ := qs[0].(map[string]interface{})["question"].(string)
		if text == "" {
			t.Fatalf("empty question text: %v", qs[0])
		}
		return text
	}
	countAttempts := func() int {
		t.Helper()
		atts, err := store.GetAttemptsForStudent(sid)
		if err != nil {
			t.Fatalf("get attempts: %v", err)
		}
		return len(atts)
	}

	// Enter, answer once, leave.
	q1 := serveOne()
	answer(q1)
	if n := countAttempts(); n != 1 {
		t.Fatalf("attempts after 1 answer = %d, want exactly 1", n)
	}

	// Return: re-fetch practice, answer the previously served question.
	_ = serveOne()
	answer(q1)
	if n := countAttempts(); n != 2 {
		t.Fatalf("attempts after return + 1 answer = %d, want exactly 2", n)
	}

	// Progress agrees with the attempt log: no hidden duplicates.
	prog, err := store.GetAllProgress(sid)
	if err != nil {
		t.Fatalf("get progress: %v", err)
	}
	if prog["a"] == nil || prog["a"].Attempts != 2 {
		t.Fatalf("progress attempts = %+v, want 2 recorded attempts", prog["a"])
	}
}

// Fork fix: the placement summary judges each concept once. It used to add an
// entry per attempt and call every unhurried right answer weak, so a concept
// answered right twice could be listed as weak twice.
func TestJourney_PlacementSummaryListsEachConceptOnce(t *testing.T) {
	mux, _ := journeyAuthedServer(t)
	token, _ := guestTokenAndID(t, mux)

	code, start := journeyPost(t, mux, token, "/api/goal/diagnostic", `{"concept_ids":["a","b"]}`)
	if code != 200 {
		t.Fatalf("diag start: %d %v", code, start)
	}
	dsid, _ := start["session_id"].(string)
	cid, _ := start["concept_id"].(string)
	done := false
	for i := 0; i < 60 && !done; i++ {
		code, ans := journeyPost(t, mux, token, "/api/goal/diagnostic/answer",
			fmt.Sprintf(`{"session_id":"%s","concept_id":"%s","answer":"4","elapsed":99.0}`, dsid, cid))
		if code != 200 {
			t.Fatalf("diag answer %d: %d %v", i, code, ans)
		}
		done, _ = ans["done"].(bool)
		cid, _ = ans["concept_id"].(string)
		if !done && cid == "" {
			t.Fatalf("diagnostic stalled at answer %d", i)
		}
	}
	code, plan := journeyPost(t, mux, token, "/api/goal/plan", `{"session_id":"`+dsid+`"}`)
	if code != 200 {
		t.Fatalf("plan: %d %v", code, plan)
	}
	listed := map[string]string{}
	weak, _ := plan["weak_areas"].(map[string]interface{})
	for _, items := range weak {
		for _, it := range items.([]interface{}) {
			label, _ := it.(map[string]interface{})["label"].(string)
			if prev, ok := listed[label]; ok {
				t.Errorf("%q listed twice (%s, weak)", label, prev)
			}
			listed[label] = "weak"
		}
	}
	strong, _ := plan["strong_areas"].(map[string]interface{})
	for _, items := range strong {
		for _, it := range items.([]interface{}) {
			label, _ := it.(string)
			if prev, ok := listed[label]; ok {
				t.Errorf("%q listed twice (%s, strong)", label, prev)
			}
			listed[label] = "strong"
		}
	}
	if len(listed) == 0 {
		t.Fatalf("summary listed nothing: %v", plan)
	}
}
