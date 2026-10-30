package server

import (
	"fmt"
	"testing"
	"time"
)

// A diagnostic used to live only in the server's map, so a deploy answered the
// next question with 404 "diagnostic session not found" — the screenshot a
// learner sent in was that message. The test simulates the restart precisely:
// the map entry disappears, the row does not.
func TestDiagnosticSession_SurvivesMemoryLoss(t *testing.T) {
	s, mux := twoConceptServer(t)

	code, start := postJSON(t, mux, "/api/goal/diagnostic", `{"name":"restart","concept_ids":["a","b"]}`)
	if code != 200 {
		t.Fatalf("start: %d %v", code, start)
	}
	dsid, _ := start["session_id"].(string)
	cid, _ := start["concept_id"].(string)
	if dsid == "" || cid == "" {
		t.Fatalf("start missing ids: %v", start)
	}

	code, ans := postJSON(t, mux, "/api/goal/diagnostic/answer",
		fmt.Sprintf(`{"session_id":"%s","concept_id":"%s","answer":"4","elapsed":1.0}`, dsid, cid))
	if code != 200 {
		t.Fatalf("first answer: %d %v", code, ans)
	}
	nextCID, _ := ans["concept_id"].(string)
	if nextCID == "" {
		t.Fatalf("diagnostic ended after one answer on a two-concept path: %v", ans)
	}

	// The restart: memory forgets, storage does not.
	s.mu.Lock()
	delete(s.diagSessions, dsid)
	delete(s.diagCreated, dsid)
	s.mu.Unlock()
	if _, found, err := s.repo.LoadDiagnosticSession(dsid); err != nil || !found {
		t.Fatalf("session was not persisted before the first answer: found=%v err=%v", found, err)
	}

	code, ans = postJSON(t, mux, "/api/goal/diagnostic/answer",
		fmt.Sprintf(`{"session_id":"%s","concept_id":"%s","answer":"4","elapsed":1.0}`, dsid, nextCID))
	if code != 200 {
		t.Fatalf("answer after simulated restart: %d %v", code, ans)
	}
}

// The finished run must outlive completion by exactly one request: the plan
// endpoint reads it and then deletes it. Deleting at completion 404'd the plan
// the learner was about to ask for, which the journey tests caught.
func TestDiagnosticSession_PlanConsumesTheFinishedRun(t *testing.T) {
	s, mux := twoConceptServer(t)
	_, dsid := journeyDiagnostic(t, mux)

	if _, found, _ := s.repo.LoadDiagnosticSession(dsid); !found {
		t.Fatal("completed run was not persisted, so a restart would lose the plan")
	}
	code, plan := postJSON(t, mux, "/api/goal/plan", `{"session_id":"`+dsid+`"}`)
	if code != 200 {
		t.Fatalf("plan: %d %v", code, plan)
	}
	if _, found, _ := s.repo.LoadDiagnosticSession(dsid); found {
		t.Error("plan left the consumed run in storage")
	}
}

// A row that no longer decodes is a dead end, not a permanent 500: drop it so
// the next request 404s once instead of failing every time.
func TestDiagnosticSession_UnreadableRunIsDropped(t *testing.T) {
	s, mux := twoConceptServer(t)
	if err := s.repo.SaveDiagnosticSession("broken", "", []byte("{")); err != nil {
		t.Fatalf("seed broken row: %v", err)
	}

	code, _ := postJSON(t, mux, "/api/goal/diagnostic/answer",
		`{"session_id":"broken","concept_id":"a","answer":"4","elapsed":1.0}`)
	if code != 404 {
		t.Fatalf("unreadable session = %d, want 404", code)
	}
	if _, found, _ := s.repo.LoadDiagnosticSession("broken"); found {
		t.Error("unreadable run was left in storage")
	}
}

// A wiped learner must not be able to resume the run from the row the wipe
// forgot: ResetProgress and account deletion both go through this helper.
func TestDiagnosticSession_DropStudentDeletesTheRow(t *testing.T) {
	s, mux := twoConceptServer(t)
	code, start := postJSON(t, mux, "/api/goal/diagnostic", `{"name":"wipe","concept_ids":["a","b"]}`)
	if code != 200 {
		t.Fatalf("start: %d %v", code, start)
	}
	sid, _ := start["student_id"].(string)
	dsid, _ := start["session_id"].(string)

	s.dropStudentLiveSessions(sid)

	if _, found, _ := s.repo.LoadDiagnosticSession(dsid); found {
		t.Error("student's diagnostic row survived the wipe")
	}
}

// The onboarding page keeps the session id for seven days and tells the
// learner that leaving the site does not throw the run away (RESUME_TTL_MS in
// web/next-app/app/onboard/page.tsx). The janitor then deleted the row an hour
// after the last answer, so a learner back from a break of an hour or more got
// 404 "diagnostic session not found", which the page shows as "Session expired,
// the server restarted and forgot this run". The server had not restarted. The
// test runs the janitor's pass on a clock two hours on: the map entry may go,
// the row and the run must not. A week on, the row goes too, because the
// browser has stopped offering it by then.
func TestDiagnosticSession_SurvivesAnHourAway(t *testing.T) {
	s, mux := twoConceptServer(t)

	code, start := postJSON(t, mux, "/api/goal/diagnostic", `{"name":"break","concept_ids":["a","b"]}`)
	if code != 200 {
		t.Fatalf("start: %d %v", code, start)
	}
	dsid, _ := start["session_id"].(string)
	cid, _ := start["concept_id"].(string)
	if dsid == "" || cid == "" {
		t.Fatalf("start missing ids: %v", start)
	}
	code, ans := postJSON(t, mux, "/api/goal/diagnostic/answer",
		fmt.Sprintf(`{"session_id":"%s","concept_id":"%s","answer":"4","elapsed":1.0}`, dsid, cid))
	if code != 200 {
		t.Fatalf("first answer: %d %v", code, ans)
	}
	nextCID, _ := ans["concept_id"].(string)
	if nextCID == "" {
		t.Fatalf("diagnostic ended after one answer on a two-concept path: %v", ans)
	}

	// The learner comes back two hours later; the janitor has run meanwhile.
	s.sweepExpiredSessions(time.Now().Add(2 * time.Hour))
	s.mu.Lock()
	_, inMemory := s.diagSessions[dsid]
	s.mu.Unlock()
	if inMemory {
		t.Fatal("the hourly eviction of the in-memory run no longer runs")
	}
	if _, found, err := s.repo.LoadDiagnosticSession(dsid); err != nil || !found {
		t.Fatalf("the row was swept an hour after the last answer: found=%v err=%v", found, err)
	}
	code, ans = postJSON(t, mux, "/api/goal/diagnostic/answer",
		fmt.Sprintf(`{"session_id":"%s","concept_id":"%s","answer":"4","elapsed":1.0}`, dsid, nextCID))
	if code != 200 {
		t.Fatalf("answer after a two-hour break: %d %v", code, ans)
	}

	// The row is still bounded: once the browser has stopped offering the run,
	// the server stops keeping it.
	s.sweepExpiredSessions(time.Now().Add(diagnosticRetention + time.Hour))
	if _, found, _ := s.repo.LoadDiagnosticSession(dsid); found {
		t.Error("the row outlived the browser's resume window")
	}
}
