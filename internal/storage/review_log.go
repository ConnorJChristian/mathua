package storage

import (
	"database/sql"
	"fmt"
	"time"
)

// The review_log table (FSRS plan 3.2): one row per scheduling event. Both
// stores share the row shape; only the placeholders differ.

// ReviewLogAtLayout is how review_log.at is stored: UTC, always nine
// fractional digits, so the text sorts in time order on both stores. Go's
// RFC3339Nano drops trailing zeros, so "17:00:00Z" sorted after
// "17:00:00.5Z" ('.' < 'Z') and events within one second read back out of
// order (FSRS Phase 1 review, M2). Reads accept either form.
const ReviewLogAtLayout = "2006-01-02T15:04:05.000000000Z"

// reviewLogAtFixedWidthSQL rewrites rows stored in the older RFC3339Nano
// form ("...:00Z" or "...:00.5Z") to ReviewLogAtLayout. It is plain SQL that
// SQLite and Postgres evaluate the same way, and it touches only rows
// shorter than the fixed width, so running it twice changes nothing.
const reviewLogAtFixedWidthSQL = `UPDATE review_log SET at = CASE
		WHEN length(at) = 20 THEN substr(at, 1, 19) || '.000000000Z'
		ELSE substr(at, 1, length(at) - 1) || substr('000000000', 1, 30 - length(at)) || 'Z'
	END
	WHERE length(at) < 30 AND at LIKE '%Z'`

// forkDataMigrationReviewLogAt is the schema_migrations version of that
// rewrite. Fork-owned data migrations take versions from 1001 up, apart from
// upstream's contiguous 1..n, so an upstream migration 6 cannot collide with
// one of ours and be skipped as "already applied".
const forkDataMigrationReviewLogAt = 1001

// reviewLogOrder is the read order: time, then insertion order for events
// stored at the same instant, so ties are stable.
const reviewLogOrder = ` ORDER BY at, id`

const reviewLogColumns = `id, student_id, concept_id, at, source, rating, elapsed_days,
	s_before, s_after, d_before, d_after, r_at_review, item_id, level, response_ms,
	idempotency_key, engine_mode, held`

func reviewLogArgs(e ReviewLogEntry) []interface{} {
	return []interface{}{
		e.StudentID, e.ConceptID, e.At.UTC().Format(ReviewLogAtLayout), e.Source, e.Rating, e.ElapsedDays,
		e.SBefore, e.SAfter, e.DBefore, e.DAfter, e.RAtReview, e.ItemID, e.Level, e.ResponseMS,
		e.IdempotencyKey, e.EngineMode, e.Held,
	}
}

func scanReviewLog(rows *sql.Rows) ([]ReviewLogEntry, error) {
	defer rows.Close()
	var out []ReviewLogEntry
	for rows.Next() {
		var e ReviewLogEntry
		var at string
		if err := rows.Scan(&e.ID, &e.StudentID, &e.ConceptID, &at, &e.Source, &e.Rating, &e.ElapsedDays,
			&e.SBefore, &e.SAfter, &e.DBefore, &e.DAfter, &e.RAtReview, &e.ItemID, &e.Level, &e.ResponseMS,
			&e.IdempotencyKey, &e.EngineMode, &e.Held); err != nil {
			return nil, fmt.Errorf("scan review_log: %w", err)
		}
		t, err := time.Parse(time.RFC3339Nano, at)
		if err != nil {
			return nil, fmt.Errorf("review_log at %q: %w", at, err)
		}
		e.At = t
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) InsertReviewLog(e ReviewLogEntry) (bool, error) {
	if e.IdempotencyKey == "" {
		return false, fmt.Errorf("insert review_log: empty idempotency key")
	}
	res, err := s.db.Exec(`INSERT INTO review_log (student_id, concept_id, at, source, rating, elapsed_days,
		s_before, s_after, d_before, d_after, r_at_review, item_id, level, response_ms, idempotency_key, engine_mode, held)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(idempotency_key) DO NOTHING`, reviewLogArgs(e)...)
	if err != nil {
		return false, fmt.Errorf("insert review_log: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *SQLiteStore) ReviewLogFor(studentID, conceptID string) ([]ReviewLogEntry, error) {
	q := `SELECT ` + reviewLogColumns + ` FROM review_log WHERE student_id = ?`
	args := []interface{}{studentID}
	if conceptID != "" {
		q += ` AND concept_id = ?`
		args = append(args, conceptID)
	}
	rows, err := s.db.Query(q+reviewLogOrder, args...)
	if err != nil {
		return nil, fmt.Errorf("review_log: %w", err)
	}
	return scanReviewLog(rows)
}

func (s *PostgresStore) InsertReviewLog(e ReviewLogEntry) (bool, error) {
	if e.IdempotencyKey == "" {
		return false, fmt.Errorf("insert review_log: empty idempotency key")
	}
	res, err := s.db.Exec(`INSERT INTO review_log (student_id, concept_id, at, source, rating, elapsed_days,
		s_before, s_after, d_before, d_after, r_at_review, item_id, level, response_ms, idempotency_key, engine_mode, held)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT(idempotency_key) DO NOTHING`, reviewLogArgs(e)...)
	if err != nil {
		return false, fmt.Errorf("insert review_log: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *PostgresStore) ReviewLogFor(studentID, conceptID string) ([]ReviewLogEntry, error) {
	q := `SELECT ` + reviewLogColumns + ` FROM review_log WHERE student_id = $1`
	args := []interface{}{studentID}
	if conceptID != "" {
		q += ` AND concept_id = $2`
		args = append(args, conceptID)
	}
	rows, err := s.db.Query(q+reviewLogOrder, args...)
	if err != nil {
		return nil, fmt.Errorf("review_log: %w", err)
	}
	return scanReviewLog(rows)
}

// FSRSError is one fsrs_errors row: an FSRS event dropped from a committed
// answer (FSRS Phase 2 review, M2). Stage is "computed" (fsrsCommit failed)
// or "written" (the commit with the event failed and the answer was
// committed without it).
type FSRSError struct {
	ID        int64
	At        time.Time
	StudentID string
	ConceptID string
	AttemptID int64
	Stage     string
	Error     string
}

const fsrsErrorColumns = `id, at, student_id, concept_id, attempt_id, stage, error`

func fsrsErrorArgs(e FSRSError) []interface{} {
	return []interface{}{e.At.UTC().Format(ReviewLogAtLayout), e.StudentID, e.ConceptID, e.AttemptID, e.Stage, e.Error}
}

func scanFSRSErrors(rows *sql.Rows) ([]FSRSError, error) {
	defer rows.Close()
	var out []FSRSError
	for rows.Next() {
		var e FSRSError
		var at string
		if err := rows.Scan(&e.ID, &at, &e.StudentID, &e.ConceptID, &e.AttemptID, &e.Stage, &e.Error); err != nil {
			return nil, fmt.Errorf("scan fsrs_errors: %w", err)
		}
		t, err := time.Parse(time.RFC3339Nano, at)
		if err != nil {
			return nil, fmt.Errorf("fsrs_errors at %q: %w", at, err)
		}
		e.At = t
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) RecordFSRSError(e FSRSError) error {
	_, err := s.db.Exec(`INSERT INTO fsrs_errors (at, student_id, concept_id, attempt_id, stage, error)
		VALUES (?, ?, ?, ?, ?, ?)`, fsrsErrorArgs(e)...)
	if err != nil {
		return fmt.Errorf("insert fsrs_errors: %w", err)
	}
	return nil
}

func (s *SQLiteStore) FSRSErrors(studentID string) ([]FSRSError, error) {
	q, args := `SELECT `+fsrsErrorColumns+` FROM fsrs_errors`, []interface{}{}
	if studentID != "" {
		q += ` WHERE student_id = ?`
		args = append(args, studentID)
	}
	rows, err := s.db.Query(q+` ORDER BY at, id`, args...)
	if err != nil {
		return nil, fmt.Errorf("fsrs_errors: %w", err)
	}
	return scanFSRSErrors(rows)
}

func (s *PostgresStore) RecordFSRSError(e FSRSError) error {
	_, err := s.db.Exec(`INSERT INTO fsrs_errors (at, student_id, concept_id, attempt_id, stage, error)
		VALUES ($1, $2, $3, $4, $5, $6)`, fsrsErrorArgs(e)...)
	if err != nil {
		return fmt.Errorf("insert fsrs_errors: %w", err)
	}
	return nil
}

func (s *PostgresStore) FSRSErrors(studentID string) ([]FSRSError, error) {
	q, args := `SELECT `+fsrsErrorColumns+` FROM fsrs_errors`, []interface{}{}
	if studentID != "" {
		q += ` WHERE student_id = $1`
		args = append(args, studentID)
	}
	rows, err := s.db.Query(q+` ORDER BY at, id`, args...)
	if err != nil {
		return nil, fmt.Errorf("fsrs_errors: %w", err)
	}
	return scanFSRSErrors(rows)
}
