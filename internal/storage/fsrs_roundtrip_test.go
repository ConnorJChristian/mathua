package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// The FSRS columns (plan 3.2) must survive every progress write path on
// both stores. This is the release blocker named in plan 4.7: a whole-row
// rewrite that drops a column resets stability to nothing within one
// answer, so a topic looks new and the plan order changes. The engine-level
// paths (PropagateWeakness, creditPrerequisites, ApplyGoalResults, the
// graded-answer upsert, mathua repair) are covered in
// internal/engine/fsrs_roundtrip_test.go and internal/repair; here the two
// store statements they all go through, on SQLite and Postgres.

func fsrsStores(t *testing.T) map[string]Repository {
	t.Helper()
	stores := map[string]Repository{"sq": newTestStore(t)}
	if dsn := os.Getenv("TEST_POSTGRES_DSN"); dsn != "" {
		pg, err := NewPostgresStore(dsn)
		if err != nil {
			t.Fatalf("open postgres: %v", err)
		}
		t.Cleanup(func() { pg.Close() })
		stores["pg"] = pg
	}
	return stores
}

func fsrsRow(sid, cid string) *ConceptProgress {
	s, d := 12.345678901, 4.567890123
	last := time.Date(2026, 11, 3, 18, 30, 0, 0, time.UTC)
	rev := last.Add(-72 * time.Hour)
	due := last.Add(12 * 24 * time.Hour)
	return &ConceptProgress{
		StudentID: sid, ConceptID: cid, Status: "MASTERED", Streak: 3, BestStreak: 5,
		AvgResponseTime: 8.5, Attempts: 9, LastAttempted: &last, LastReviewed: &rev, NextReviewDue: &due,
		SM2Repetitions: 2, SM2Interval: 7, SM2EFactor: 2.6, MasteredAt: &rev, WeaknessScore: 0.1,
		PlacementSeeded: false, ImplicitCredit: 0,
		FSRSStability: &s, FSRSDifficulty: &d, FSRSLastReview: &last, FSRSReps: 4, FSRSLapses: 1,
		FSRSState: "review", FSRSSeed: "mastery",
	}
}

func sameFSRS(t *testing.T, tag string, want, got *ConceptProgress) {
	t.Helper()
	if got == nil {
		t.Fatalf("[%s] row missing", tag)
	}
	f := func(p *float64) string {
		if p == nil {
			return "nil"
		}
		return fmt.Sprintf("%.12g", *p)
	}
	if f(want.FSRSStability) != f(got.FSRSStability) || f(want.FSRSDifficulty) != f(got.FSRSDifficulty) {
		t.Errorf("[%s] S/D: want %s/%s, got %s/%s", tag, f(want.FSRSStability), f(want.FSRSDifficulty), f(got.FSRSStability), f(got.FSRSDifficulty))
	}
	if (want.FSRSLastReview == nil) != (got.FSRSLastReview == nil) ||
		(want.FSRSLastReview != nil && !want.FSRSLastReview.Equal(*got.FSRSLastReview)) {
		t.Errorf("[%s] fsrs_last_review: want %v, got %v", tag, want.FSRSLastReview, got.FSRSLastReview)
	}
	if want.FSRSReps != got.FSRSReps || want.FSRSLapses != got.FSRSLapses || want.FSRSState != got.FSRSState || want.FSRSSeed != got.FSRSSeed {
		t.Errorf("[%s] reps/lapses/state/seed: want %d/%d/%q/%q, got %d/%d/%q/%q", tag,
			want.FSRSReps, want.FSRSLapses, want.FSRSState, want.FSRSSeed, got.FSRSReps, got.FSRSLapses, got.FSRSState, got.FSRSSeed)
	}
}

func TestFSRSColumns_SurviveEveryStoreWrite(t *testing.T) {
	for tag, repo := range fsrsStores(t) {
		st, err := repo.CreateStudent(fmt.Sprintf("fsrs-rt-%s-%d", tag, time.Now().UnixNano()))
		if err != nil {
			t.Fatal(err)
		}
		// 1. Single-row upsert, insert then update (the graded-answer path).
		// It never writes the FSRS columns: the insert takes the defaults
		// and an update keeps what is stored, whatever the struct carries.
		// WriteFSRSState is their one writer.
		row := fsrsRow(st.ID, "a")
		if err := repo.UpsertProgress(row); err != nil {
			t.Fatal(err)
		}
		got, _ := repo.GetProgress(st.ID, "a")
		sameFSRS(t, tag+" upsert insert (no state)", &ConceptProgress{FSRSState: "none"}, got)
		if err := repo.WriteFSRSState(row); err != nil {
			t.Fatal(err)
		}
		got, _ = repo.GetProgress(st.ID, "a")
		sameFSRS(t, tag+" write state", row, got)
		if got.Streak != row.Streak || got.Attempts != row.Attempts {
			t.Errorf("[%s] WriteFSRSState changed other columns: %+v", tag, got)
		}
		stale := *row
		stale.Streak = 4
		s0 := 1.0
		stale.FSRSStability, stale.FSRSReps, stale.FSRSState = &s0, 0, "none"
		if err := repo.UpsertProgress(&stale); err != nil {
			t.Fatal(err)
		}
		got, _ = repo.GetProgress(st.ID, "a")
		sameFSRS(t, tag+" upsert update keeps state", row, got)
		if got.Streak != 4 {
			t.Errorf("[%s] the update lost streak", tag)
		}
		row.Streak = 4
		*row.FSRSStability = 30.5
		row.FSRSReps = 5
		row.FSRSState = "relearning"
		if err := repo.WriteFSRSState(row); err != nil {
			t.Fatal(err)
		}
		got, _ = repo.GetProgress(st.ID, "a")
		sameFSRS(t, tag+" write state update", row, got)
		if err := repo.WriteFSRSState(fsrsRow(st.ID, "no-row")); !errors.Is(err, ErrNoProgressRow) {
			t.Errorf("[%s] WriteFSRSState on a missing row: %v, want ErrNoProgressRow", tag, err)
		}
		// 2. Batch upsert of a loaded snapshot with one other column changed
		// (PropagateWeakness, creditPrerequisites, ApplyGoalResults).
		all, _ := repo.GetAllProgress(st.ID)
		cp := *all["a"]
		cp.StudentID, cp.ConceptID = st.ID, "a"
		cp.WeaknessScore = 0.3
		fresh := fsrsRow(st.ID, "b") // a struct with FSRS fields set: a batch insert ignores them
		fresh.Status = "UNSEEN"
		if err := repo.UpsertProgressBatch([]*ConceptProgress{&cp, fresh}); err != nil {
			t.Fatal(err)
		}
		all, _ = repo.GetAllProgress(st.ID)
		sameFSRS(t, tag+" batch", row, all["a"])
		if all["a"].WeaknessScore != 0.3 {
			t.Errorf("[%s] batch did not write weakness", tag)
		}
		if b := all["b"]; b.FSRSStability != nil || b.FSRSState != "none" || b.FSRSSeed != "" {
			t.Errorf("[%s] a fresh row has FSRS state: %+v", tag, b)
		}
		// 3. A row with no state round-trips as no state (NULL, not 0).
		none := &ConceptProgress{StudentID: st.ID, ConceptID: "c", Status: "MASTERED"}
		if err := repo.UpsertProgress(none); err != nil {
			t.Fatal(err)
		}
		got, _ = repo.GetProgress(st.ID, "c")
		if got.FSRSStability != nil || got.FSRSDifficulty != nil || got.FSRSLastReview != nil || got.FSRSState != "none" {
			t.Errorf("[%s] no-state row came back with state: %+v", tag, got)
		}
		// 4. Reset wipes the state with the row; the log goes too.
		if _, err := repo.InsertReviewLog(ReviewLogEntry{StudentID: st.ID, ConceptID: "a", At: time.Now().UTC(), Source: "review", Rating: 3, IdempotencyKey: "k-" + st.ID, EngineMode: "shadow"}); err != nil {
			t.Fatal(err)
		}
		if err := repo.ResetProgress(st.ID); err != nil {
			t.Fatal(err)
		}
		if got, _ := repo.GetProgress(st.ID, "a"); got != nil {
			t.Errorf("[%s] progress survives reset", tag)
		}
		if log, _ := repo.ReviewLogFor(st.ID, ""); len(log) != 0 {
			t.Errorf("[%s] review_log survives reset: %d rows", tag, len(log))
		}
		if err := repo.DeleteAccount(st.ID); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReviewLog_IdempotentAndOrdered(t *testing.T) {
	for tag, repo := range fsrsStores(t) {
		st, err := repo.CreateStudent(fmt.Sprintf("fsrs-log-%s-%d", tag, time.Now().UnixNano()))
		if err != nil {
			t.Fatal(err)
		}
		base := time.Date(2026, 11, 4, 9, 0, 0, 0, time.UTC)
		e := ReviewLogEntry{StudentID: st.ID, ConceptID: "a", At: base, Source: "review", Rating: 3, ElapsedDays: 2,
			SBefore: 2.3065, SAfter: 10.96, DBefore: 2.118, DAfter: 2.111, RAtReview: 0.9, ItemID: "q1", Level: 2, ResponseMS: 4100,
			IdempotencyKey: "sess:1:" + st.ID, EngineMode: "shadow"}
		ins, err := repo.InsertReviewLog(e)
		if err != nil || !ins {
			t.Fatalf("[%s] first insert: %v %v", tag, ins, err)
		}
		// The same key again writes nothing, even with different fields.
		dup := e
		dup.Rating = 1
		ins, err = repo.InsertReviewLog(dup)
		if err != nil || ins {
			t.Fatalf("[%s] duplicate key inserted: %v %v", tag, ins, err)
		}
		// An earlier event inserted later still reads first.
		earlier := e
		earlier.At = base.Add(-48 * time.Hour)
		earlier.IdempotencyKey = "sess:0:" + st.ID
		earlier.Source = "seed"
		if _, err := repo.InsertReviewLog(earlier); err != nil {
			t.Fatal(err)
		}
		other := e
		other.ConceptID = "b"
		other.IdempotencyKey = "sess:2:" + st.ID
		if _, err := repo.InsertReviewLog(other); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.InsertReviewLog(ReviewLogEntry{StudentID: st.ID, ConceptID: "a"}); err == nil {
			t.Errorf("[%s] an empty idempotency key was accepted", tag)
		}
		log, err := repo.ReviewLogFor(st.ID, "a")
		if err != nil || len(log) != 2 {
			t.Fatalf("[%s] log for a: %d %v", tag, len(log), err)
		}
		if log[0].Source != "seed" || !log[0].At.Equal(earlier.At) || log[1].Rating != 3 || log[1].SAfter != 10.96 || log[1].ItemID != "q1" || log[1].ResponseMS != 4100 || log[1].EngineMode != "shadow" {
			t.Errorf("[%s] log order or fields: %+v", tag, log)
		}
		all, _ := repo.ReviewLogFor(st.ID, "")
		if len(all) != 3 {
			t.Errorf("[%s] whole log: %d rows", tag, len(all))
		}
		if err := repo.DeleteAccount(st.ID); err != nil {
			t.Fatal(err)
		}
		if left, _ := repo.ReviewLogFor(st.ID, ""); len(left) != 0 {
			t.Errorf("[%s] review_log survives account deletion", tag)
		}
	}
}

// A database created before the columns existed migrates in place: its rows
// read as having no FSRS state and the new columns are writable.
func TestFSRSColumns_MigrateAnOlderSQLiteFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	old, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	st, _ := old.CreateStudent("older")
	if err := old.UpsertProgress(&ConceptProgress{StudentID: st.ID, ConceptID: "a", Status: "MASTERED", Streak: 3}); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		"ALTER TABLE concept_progress DROP COLUMN fsrs_stability", "ALTER TABLE concept_progress DROP COLUMN fsrs_difficulty",
		"ALTER TABLE concept_progress DROP COLUMN fsrs_last_review", "ALTER TABLE concept_progress DROP COLUMN fsrs_reps",
		"ALTER TABLE concept_progress DROP COLUMN fsrs_lapses", "ALTER TABLE concept_progress DROP COLUMN fsrs_state",
		"ALTER TABLE concept_progress DROP COLUMN fsrs_seed", "DROP INDEX idx_review_log_topic", "DROP TABLE review_log",
	} {
		if _, err := old.db.Exec(q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	old.Close()
	repo, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("reopen (migrate): %v", err)
	}
	defer repo.Close()
	got, err := repo.GetProgress(st.ID, "a")
	if err != nil || got == nil {
		t.Fatalf("read after migration: %+v %v", got, err)
	}
	if got.FSRSStability != nil || got.FSRSState != "none" || got.Streak != 3 {
		t.Errorf("migrated row: %+v", got)
	}
	row := fsrsRow(st.ID, "a")
	if err := repo.WriteFSRSState(row); err != nil {
		t.Fatal(err)
	}
	got, _ = repo.GetProgress(st.ID, "a")
	sameFSRS(t, "migrated", row, got)
	if ins, err := repo.InsertReviewLog(ReviewLogEntry{StudentID: st.ID, ConceptID: "a", At: time.Now().UTC(), Source: "review", Rating: 3, IdempotencyKey: "m1"}); err != nil || !ins {
		t.Errorf("review_log after migration: %v %v", ins, err)
	}
}

// A whole-row rewrite from a snapshot taken before a review commit must not
// roll that review back (FSRS Phase 1 review, Phase 2 note: lost updates).
// Snapshot, then an FSRS write lands, then the batch and the single upsert
// write the stale snapshot: the newer FSRS state survives both.
func TestFSRSColumns_InterleavedWriteSurvivesSnapshotRewrite(t *testing.T) {
	for tag, repo := range fsrsStores(t) {
		st, err := repo.CreateStudent(fmt.Sprintf("fsrs-race-%s-%d", tag, time.Now().UnixNano()))
		if err != nil {
			t.Fatal(err)
		}
		row := fsrsRow(st.ID, "a")
		if err := repo.UpsertProgress(row); err != nil {
			t.Fatal(err)
		}
		if err := repo.WriteFSRSState(row); err != nil {
			t.Fatal(err)
		}
		snap, _ := repo.GetAllProgress(st.ID) // the request's snapshot
		newer := fsrsRow(st.ID, "a")          // a review commits in between
		s2 := 40.25
		newer.FSRSStability, newer.FSRSReps = &s2, 5
		if err := repo.WriteFSRSState(newer); err != nil {
			t.Fatal(err)
		}
		cp := *snap["a"]
		cp.WeaknessScore = 0.4
		if err := repo.UpsertProgressBatch([]*ConceptProgress{&cp}); err != nil {
			t.Fatal(err)
		}
		got, _ := repo.GetProgress(st.ID, "a")
		sameFSRS(t, tag+" batch after interleaved write", newer, got)
		if got.WeaknessScore != 0.4 {
			t.Errorf("[%s] batch did not write weakness", tag)
		}
		cp.Streak = 7
		if err := repo.UpsertProgress(&cp); err != nil {
			t.Fatal(err)
		}
		got, _ = repo.GetProgress(st.ID, "a")
		sameFSRS(t, tag+" upsert after interleaved write", newer, got)
		_ = repo.DeleteAccount(st.ID)
	}
}
