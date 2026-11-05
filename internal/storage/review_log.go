package storage

import (
	"database/sql"
	"fmt"
	"time"
)

// The review_log table (FSRS plan 3.2): one row per scheduling event. Both
// stores share the row shape; only the placeholders differ.

const reviewLogColumns = `id, student_id, concept_id, at, source, rating, elapsed_days,
	s_before, s_after, d_before, d_after, r_at_review, item_id, level, response_ms,
	idempotency_key, engine_mode`

func reviewLogArgs(e ReviewLogEntry) []interface{} {
	return []interface{}{
		e.StudentID, e.ConceptID, e.At.UTC().Format(time.RFC3339Nano), e.Source, e.Rating, e.ElapsedDays,
		e.SBefore, e.SAfter, e.DBefore, e.DAfter, e.RAtReview, e.ItemID, e.Level, e.ResponseMS,
		e.IdempotencyKey, e.EngineMode,
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
			&e.IdempotencyKey, &e.EngineMode); err != nil {
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
		s_before, s_after, d_before, d_after, r_at_review, item_id, level, response_ms, idempotency_key, engine_mode)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
	rows, err := s.db.Query(q+` ORDER BY at, id`, args...)
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
		s_before, s_after, d_before, d_after, r_at_review, item_id, level, response_ms, idempotency_key, engine_mode)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
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
	rows, err := s.db.Query(q+` ORDER BY at, id`, args...)
	if err != nil {
		return nil, fmt.Errorf("review_log: %w", err)
	}
	return scanReviewLog(rows)
}
