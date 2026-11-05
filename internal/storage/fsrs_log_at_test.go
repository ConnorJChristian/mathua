package storage

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

// review_log.at is stored fixed-width (ReviewLogAtLayout) so it sorts as
// text; rows written in the older RFC3339Nano form are rewritten by fork
// data migration 1001 (FSRS Phase 1 review, M2).

func TestReviewLogAt_StoredFixedWidth(t *testing.T) {
	for tag, repo := range fsrsStores(t) {
		st, err := repo.CreateStudent(fmt.Sprintf("fsrs-at-%s-%d", tag, time.Now().UnixNano()))
		if err != nil {
			t.Fatal(err)
		}
		at := time.Date(2026, 11, 5, 17, 0, 0, 0, time.FixedZone("PST", -8*3600))
		if _, err := repo.InsertReviewLog(ReviewLogEntry{StudentID: st.ID, ConceptID: "a", At: at, Source: "review", Rating: 3,
			IdempotencyKey: "at:" + st.ID, EngineMode: "shadow"}); err != nil {
			t.Fatal(err)
		}
		var raw string
		db, ph := rawDB(repo)
		if err := db.QueryRow("SELECT at FROM review_log WHERE student_id = "+ph(1), st.ID).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		if want := "2026-11-06T01:00:00.000000000Z"; raw != want {
			t.Errorf("[%s] stored at %q, want %q (UTC, nine digits)", tag, raw, want)
		}
		// Ties at one instant read back in insertion order (id).
		for i := 0; i < 3; i++ {
			if _, err := repo.InsertReviewLog(ReviewLogEntry{StudentID: st.ID, ConceptID: "b", At: at, Source: "review", Rating: i + 1,
				IdempotencyKey: fmt.Sprintf("tie:%d:%s", i, st.ID), EngineMode: "shadow"}); err != nil {
				t.Fatal(err)
			}
		}
		log, _ := repo.ReviewLogFor(st.ID, "b")
		for i, e := range log {
			if e.Rating != i+1 {
				t.Errorf("[%s] tie order: row %d has rating %d", tag, i, e.Rating)
			}
		}
		_ = repo.DeleteAccount(st.ID)
	}
}

// An older SQLite file holding RFC3339Nano rows is rewritten on open, once,
// and then reads back in time order.
func TestReviewLogAt_MigratesOlderRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	old, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	st, _ := old.CreateStudent("older-log")
	whole := time.Date(2026, 11, 5, 17, 0, 0, 0, time.UTC)
	stamps := []time.Time{whole, whole.Add(500 * time.Millisecond), whole.Add(time.Second + 123456789), whole.Add(2*time.Second + 100)}
	// Insert newest first, in the old format, so neither id order nor the
	// old text order is time order.
	for i := len(stamps) - 1; i >= 0; i-- {
		if _, err := old.db.Exec(`INSERT INTO review_log (student_id, concept_id, at, source, rating, idempotency_key)
			VALUES (?, 'a', ?, 'review', 3, ?)`, st.ID, stamps[i].Format(time.RFC3339Nano), fmt.Sprintf("old:%d", i)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := old.db.Exec("DELETE FROM schema_migrations WHERE version = ?", forkDataMigrationReviewLogAt); err != nil {
		t.Fatal(err)
	}
	old.Close()
	repo, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("reopen (migrate): %v", err)
	}
	defer repo.Close()
	rows, err := repo.db.Query("SELECT at FROM review_log ORDER BY at")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for rows.Next() {
		var s string
		_ = rows.Scan(&s)
		got = append(got, s)
	}
	rows.Close()
	for i, s := range got {
		if want := stamps[i].Format(ReviewLogAtLayout); s != want {
			t.Errorf("migrated row %d = %q, want %q", i, s, want)
		}
	}
	log, err := repo.ReviewLogFor(st.ID, "a")
	if err != nil || len(log) != len(stamps) {
		t.Fatalf("%d rows, %v", len(log), err)
	}
	for i, e := range log {
		if !e.At.Equal(stamps[i]) {
			t.Errorf("row %d at %v, want %v", i, e.At, stamps[i])
		}
	}
	// Recorded once: a second open is a no-op.
	var n int
	_ = repo.db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", forkDataMigrationReviewLogAt).Scan(&n)
	if n != 1 {
		t.Errorf("schema_migrations rows for %d = %d", forkDataMigrationReviewLogAt, n)
	}
}

// The rewrite is plain SQL both stores evaluate the same way: old-format
// rows planted directly come out fixed-width, and a second run is a no-op.
func TestReviewLogAt_RewriteSQLOnBothStores(t *testing.T) {
	for tag, repo := range fsrsStores(t) {
		st, err := repo.CreateStudent(fmt.Sprintf("fsrs-rw-%s-%d", tag, time.Now().UnixNano()))
		if err != nil {
			t.Fatal(err)
		}
		db, ph := rawDB(repo)
		whole := time.Date(2026, 11, 5, 17, 0, 0, 0, time.UTC)
		stamps := []time.Time{whole, whole.Add(500 * time.Millisecond), whole.Add(123456789), whole.Add(999999999)}
		for i, at := range stamps {
			if _, err := db.Exec(`INSERT INTO review_log (student_id, concept_id, at, source, rating, idempotency_key)
				VALUES (`+ph(1)+`, 'a', `+ph(2)+`, 'review', 3, `+ph(3)+`)`, st.ID, at.Format(time.RFC3339Nano), fmt.Sprintf("rw:%d:%s", i, st.ID)); err != nil {
				t.Fatal(err)
			}
		}
		for pass := 0; pass < 2; pass++ {
			if _, err := db.Exec(reviewLogAtFixedWidthSQL); err != nil {
				t.Fatalf("[%s] rewrite: %v", tag, err)
			}
			for i, at := range stamps {
				var raw string
				if err := db.QueryRow("SELECT at FROM review_log WHERE idempotency_key = "+ph(1), fmt.Sprintf("rw:%d:%s", i, st.ID)).Scan(&raw); err != nil {
					t.Fatal(err)
				}
				if want := at.Format(ReviewLogAtLayout); raw != want {
					t.Errorf("[%s] pass %d row %d = %q, want %q", tag, pass, i, raw, want)
				}
			}
		}
		_ = repo.DeleteAccount(st.ID)
	}
}

func rawDB(repo Repository) (*sql.DB, func(int) string) {
	switch r := repo.(type) {
	case *SQLiteStore:
		return r.db, func(int) string { return "?" }
	case *PostgresStore:
		return r.db, func(n int) string { return fmt.Sprintf("$%d", n) }
	}
	panic("unknown store")
}
