package engine

import (
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/leaderboard"
	"github.com/chuma-beep/mathua/internal/storage"
)

// TestBackfillWeeklyXP_RestoresTheWeekFromRealAttempts covers the repair this
// exists for: XP awarded before the ledger started cannot be re-derived from
// the counters, so it is replayed from the attempts that earned it. The replay
// must count one correct practice attempt, ignore a wrong one, ignore the
// diagnostic, ignore a guest, skip last week, and be safe to run twice.
func TestBackfillWeeklyXP_RestoresTheWeekFromRealAttempts(t *testing.T) {
	repo, err := storage.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { repo.Close() })

	learner, err := repo.CreateStudent("ada")
	if err != nil {
		t.Fatalf("create learner: %v", err)
	}
	guest, err := repo.ClaimGuestStudent("guest_replay_test", "Guest")
	if err != nil {
		t.Fatalf("claim guest: %v", err)
	}
	if err := repo.EnsureSession("s-learner", learner.ID); err != nil {
		t.Fatalf("learner session: %v", err)
	}
	if err := repo.EnsureSession("s-guest", guest.ID); err != nil {
		t.Fatalf("guest session: %v", err)
	}

	now := time.Now().UTC()
	record := func(session, student, source string, correct bool, at time.Time) {
		t.Helper()
		if _, err := repo.RecordAttempt(storage.AttemptEntry{
			SessionID: session, StudentID: student, ConceptID: "arith.add.single",
			Answer: "1", Expected: "1", Correct: correct, ElapsedSeconds: 3,
			Timestamp: at, Source: source,
		}); err != nil {
			t.Fatalf("record attempt: %v", err)
		}
	}
	// "Earlier this week": an hour ago, unless the week began less than an
	// hour ago (Monday just after midnight UTC), when an hour ago is last week.
	recent := now.Add(-time.Hour)
	if ws := leaderboard.WeekStart(now); recent.Before(ws) {
		recent = ws.Add((now.Sub(ws)) / 2)
	}
	record("s-learner", learner.ID, "practice", true, recent)
	record("s-learner", learner.ID, "practice", false, recent)
	record("s-learner", learner.ID, "diagnostic", true, recent)
	record("s-guest", guest.ID, "practice", true, recent)
	record("s-learner", learner.ID, "practice", true, now.AddDate(0, 0, -8))

	eng := &Engine{repo: repo}
	restored, err := eng.BackfillWeeklyXP(now)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if restored != 1 {
		t.Errorf("restored %d events, want 1", restored)
	}
	weekly, err := repo.WeeklyXPByStudent(now.AddDate(0, 0, -7))
	if err != nil {
		t.Fatalf("weekly: %v", err)
	}
	if weekly[learner.ID] <= 0 {
		t.Errorf("learner weekly XP = %d, want the replayed award", weekly[learner.ID])
	}
	if _, ok := weekly[guest.ID]; ok {
		t.Error("a guest attempt was replayed into the public week")
	}

	// Running again (every restart does) must not move the total.
	before := weekly[learner.ID]
	restored, err = eng.BackfillWeeklyXP(now)
	if err != nil {
		t.Fatalf("second backfill: %v", err)
	}
	if restored != 0 {
		t.Errorf("second run restored %d events, want 0", restored)
	}
	weekly, err = repo.WeeklyXPByStudent(now.AddDate(0, 0, -7))
	if err != nil {
		t.Fatalf("weekly after retry: %v", err)
	}
	if weekly[learner.ID] != before {
		t.Errorf("weekly XP moved from %d to %d on a retry", before, weekly[learner.ID])
	}
}
