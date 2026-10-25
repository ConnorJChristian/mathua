package storage

import (
	"context"
	"database/sql"
	"fmt"
)

// The proof grader's persistence (internal/proof.Store): the month's spend
// and the grade cache. See the proof_spend and proof_grades schema comments.

func (s *SQLiteStore) ProofSpend(ctx context.Context, month string) (float64, error) {
	return proofSpend(ctx, s.db, "SELECT usd FROM proof_spend WHERE month = ?", month)
}

func (s *SQLiteStore) AddProofSpend(ctx context.Context, month string, usd float64, in, out int) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO proof_spend (month, usd, input_tokens, output_tokens, gradings) VALUES (?, ?, ?, ?, 1)
		ON CONFLICT(month) DO UPDATE SET usd = usd + excluded.usd,
			input_tokens = input_tokens + excluded.input_tokens,
			output_tokens = output_tokens + excluded.output_tokens,
			gradings = gradings + 1
	`, month, usd, in, out)
	if err != nil {
		return fmt.Errorf("add proof spend: %w", err)
	}
	return nil
}

func (s *SQLiteStore) ProofGrade(ctx context.Context, key string) (string, bool, error) {
	return proofGrade(ctx, s.db, "SELECT result FROM proof_grades WHERE cache_key = ?", key)
}

func (s *SQLiteStore) PutProofGrade(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO proof_grades (cache_key, result) VALUES (?, ?)
		ON CONFLICT(cache_key) DO UPDATE SET result = excluded.result
	`, key, value)
	if err != nil {
		return fmt.Errorf("put proof grade: %w", err)
	}
	return nil
}

func (s *PostgresStore) ProofSpend(ctx context.Context, month string) (float64, error) {
	return proofSpend(ctx, s.db, "SELECT usd FROM proof_spend WHERE month = $1", month)
}

func (s *PostgresStore) AddProofSpend(ctx context.Context, month string, usd float64, in, out int) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO proof_spend (month, usd, input_tokens, output_tokens, gradings) VALUES ($1, $2, $3, $4, 1)
		ON CONFLICT (month) DO UPDATE SET usd = proof_spend.usd + EXCLUDED.usd,
			input_tokens = proof_spend.input_tokens + EXCLUDED.input_tokens,
			output_tokens = proof_spend.output_tokens + EXCLUDED.output_tokens,
			gradings = proof_spend.gradings + 1
	`, month, usd, in, out)
	if err != nil {
		return fmt.Errorf("add proof spend: %w", err)
	}
	return nil
}

func (s *PostgresStore) ProofGrade(ctx context.Context, key string) (string, bool, error) {
	return proofGrade(ctx, s.db, "SELECT result FROM proof_grades WHERE cache_key = $1", key)
}

func (s *PostgresStore) PutProofGrade(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO proof_grades (cache_key, result) VALUES ($1, $2)
		ON CONFLICT (cache_key) DO UPDATE SET result = EXCLUDED.result
	`, key, value)
	if err != nil {
		return fmt.Errorf("put proof grade: %w", err)
	}
	return nil
}

func proofSpend(ctx context.Context, db *sql.DB, q, month string) (float64, error) {
	var usd float64
	err := db.QueryRowContext(ctx, q, month).Scan(&usd)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read proof spend: %w", err)
	}
	return usd, nil
}

func proofGrade(ctx context.Context, db *sql.DB, q, key string) (string, bool, error) {
	var v string
	err := db.QueryRowContext(ctx, q, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read proof grade: %w", err)
	}
	return v, true, nil
}
