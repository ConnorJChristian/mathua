package storage

import (
	"testing"
	"time"
)

// TestActiveLearnerCounts_SplitsGuestsAndWindows pins the two numbers the admin
// overview shows. activeNow is a live session, activeToday is an answered
// question, and neither may count the same learner twice across sessions or
// count a session older than the window.
func TestActiveLearnerCounts_SplitsGuestsAndWindows(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().UTC()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	ada, err := store.CreateStudent("ada")
	if err != nil {
		t.Fatalf("create ada: %v", err)
	}
	bob, err := store.CreateStudent("bob")
	if err != nil {
		t.Fatalf("create bob: %v", err)
	}
	guest, err := store.ClaimGuestStudent("guest_active_test", "Guest")
	if err != nil {
		t.Fatalf("guest: %v", err)
	}

	live := []*Student{ada, bob, guest}
	for i, st := range live {
		sid := "sess-live-" + st.ID
		if err := store.EnsureSession(sid, st.ID); err != nil {
			t.Fatalf("session %d: %v", i, err)
		}
		if _, err := store.db.Exec(
			`INSERT INTO active_sessions (session_id, student_id, concept_id, expected_answer, attempt_id, question)
			 VALUES (?, ?, 'c1', '1', ?, 'q')`, sid, st.ID, "att-live-"+st.ID); err != nil {
			t.Fatalf("active row %d: %v", i, err)
		}
	}
	// Ada also has a stale seat from earlier; it must not add to the count.
	oldSid := "sess-old-" + ada.ID
	if err := store.EnsureSession(oldSid, ada.ID); err != nil {
		t.Fatalf("old session: %v", err)
	}
	if _, err := store.db.Exec(
		`INSERT INTO active_sessions (session_id, student_id, concept_id, expected_answer, attempt_id, question, updated_at)
		 VALUES (?, ?, 'c1', '1', ?, 'q', datetime('now', '-2 hours'))`, oldSid, ada.ID, "att-old-"+ada.ID); err != nil {
		t.Fatalf("stale row: %v", err)
	}

	// Ada and the guest answered today; Bob's only attempt is from yesterday.
	record := func(st *Student, when time.Time, key string) {
		t.Helper()
		if _, err := store.RecordAttempt(AttemptEntry{
			SessionID:      "sess-live-" + st.ID,
			StudentID:      st.ID,
			ConceptID:      "c1",
			Answer:         "1",
			Expected:       "1",
			Correct:        true,
			ElapsedSeconds: 4,
			Timestamp:      when,
			Question:       "q",
			Source:         "practice",
		}); err != nil {
			t.Fatalf("attempt %s: %v", key, err)
		}
	}
	// "Earlier today" must stay inside the UTC day: an hour before now is
	// yesterday when the test runs between 00:00 and 01:00 UTC.
	earlierToday := dayStart.Add(now.Sub(dayStart) / 2)
	record(ada, earlierToday, "ada")
	record(guest, earlierToday, "guest")
	record(bob, now.AddDate(0, 0, -1), "bob-yesterday")

	activeNow, activeNowGuests, activeToday, activeTodayGuests, err :=
		store.ActiveLearnerCounts(now.Add(-15*time.Minute), dayStart)
	if err != nil {
		t.Fatalf("counts: %v", err)
	}
	if activeNow != 2 || activeNowGuests != 1 {
		t.Errorf("active now = %d accounts / %d guests, want 2/1", activeNow, activeNowGuests)
	}
	if activeToday != 1 || activeTodayGuests != 1 {
		t.Errorf("active today = %d accounts / %d guests, want 1/1", activeToday, activeTodayGuests)
	}
}
