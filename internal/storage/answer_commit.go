package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// AnswerCommit is everything one graded answer writes (FSRS Phase 2, step
// 2): the progress row and the attempt, as before, and in shadow or active
// mode the review_log rows and the fsrs_* columns. CommitAnswer writes them
// in one transaction, so they land or roll back together.
type AnswerCommit struct {
	// Progress is upserted like UpsertProgress (the fsrs_* columns are not
	// touched by it). Nil writes no progress row.
	Progress *ConceptProgress
	// Attempt is inserted like RecordAttempt. Nil writes no attempt.
	Attempt *AttemptEntry
	// FSRS is nil in off mode and for an answer that is no FSRS event.
	FSRS *FSRSCommit
}

// FSRSCommit is the FSRS half of an answer.
type FSRSCommit struct {
	// Log is appended in order. An entry with an empty IdempotencyKey gets
	// "<attempt session id>:a<attempt row id>", the attempt's own key.
	Log []ReviewLogEntry
	// State, when set, has its fsrs_* fields written to the row
	// (WriteFSRSState's columns, nothing else).
	State *ConceptProgress
}

// CommitResult is what CommitAnswer did.
type CommitResult struct {
	// AttemptID is the new attempt row's id (0 when none was written).
	AttemptID int64
	// Duplicate is true when a review_log idempotency key was already
	// stored: the answer was committed before (a retry), and nothing was
	// written this time, the progress and the attempt included.
	Duplicate bool
}

// errDuplicateKey unwinds a commit whose log key is already stored.
var errDuplicateKey = errors.New("duplicate review_log key")

// commitHook, when set by a test, runs at each stage of a commit
// ("progress", "attempt", "log", "state") and fails it with its error.
var commitHook func(stage string) error

func hook(stage string) error {
	if commitHook != nil {
		return commitHook(stage)
	}
	return nil
}

const upsertProgressSQL = `
		INSERT INTO concept_progress
			(student_id, concept_id, status, streak, best_streak,
			 avg_response_time, attempts, last_attempted, last_reviewed,
			 next_review_due, sm2_repetitions, sm2_interval, sm2_efactor,
			 mastered_at, weakness_score, placement_seeded, implicit_credit)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(student_id, concept_id) DO UPDATE SET
			status            = excluded.status,
			streak            = excluded.streak,
			best_streak       = excluded.best_streak,
			avg_response_time = excluded.avg_response_time,
			attempts          = excluded.attempts,
			last_attempted    = excluded.last_attempted,
			last_reviewed     = excluded.last_reviewed,
			next_review_due   = excluded.next_review_due,
			sm2_repetitions   = excluded.sm2_repetitions,
			sm2_interval      = excluded.sm2_interval,
			sm2_efactor       = excluded.sm2_efactor,
			mastered_at       = excluded.mastered_at,
			weakness_score    = excluded.weakness_score`

func upsertProgressArgs(p *ConceptProgress) []interface{} {
	return []interface{}{
		p.StudentID, p.ConceptID, p.Status, p.Streak, p.BestStreak,
		p.AvgResponseTime, p.Attempts,
		nullTime(p.LastAttempted), nullTime(p.LastReviewed),
		nullTime(p.NextReviewDue), p.SM2Repetitions, p.SM2Interval,
		p.SM2EFactor, nullTime(p.MasteredAt), p.WeaknessScore, boolToInt(p.PlacementSeeded), p.ImplicitCredit,
	}
}

const insertAttemptSQL = `
		INSERT INTO attempts
			(session_id, student_id, concept_id, answer, expected,
			 correct, elapsed_seconds, timestamp, question, source, explanation, difficulty)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

func insertAttemptArgs(entry *AttemptEntry) []interface{} {
	return []interface{}{
		entry.SessionID, entry.StudentID, entry.ConceptID,
		entry.Answer, entry.Expected,
		boolToInt(entry.Correct), entry.ElapsedSeconds,
		entry.Timestamp.Format(time.RFC3339),
		entry.Question, entry.Source, entry.Explanation, entry.Difficulty,
	}
}

const insertReviewLogSQL = `INSERT INTO review_log (student_id, concept_id, at, source, rating, elapsed_days,
		s_before, s_after, d_before, d_after, r_at_review, item_id, level, response_ms, idempotency_key, engine_mode, held)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(idempotency_key) DO NOTHING`

// dollarPlaceholders turns "?" placeholders into Postgres' $1..$n (the
// statements above hold no literal question marks).
func dollarPlaceholders(q string) string {
	var b strings.Builder
	n := 0
	for _, r := range q {
		if r == '?' {
			n++
			b.WriteString("$" + strconv.Itoa(n))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// commitDialect is what differs between the stores.
type commitDialect struct {
	ph func(string) string
	// insertAttempt inserts and returns the row id.
	insertAttempt func(tx *sql.Tx, q string, args []interface{}) (int64, error)
}

func commitAnswer(db *sql.DB, d commitDialect, c AnswerCommit) (CommitResult, error) {
	tx, err := db.Begin()
	if err != nil {
		return CommitResult{}, fmt.Errorf("commit answer: %w", err)
	}
	defer tx.Rollback()
	var res CommitResult
	if c.Progress != nil {
		if _, err := tx.Exec(d.ph(upsertProgressSQL), upsertProgressArgs(c.Progress)...); err != nil {
			return CommitResult{}, fmt.Errorf("commit answer: upsert progress: %w", err)
		}
		if err := hook("progress"); err != nil {
			return CommitResult{}, err
		}
	}
	if c.Attempt != nil {
		id, err := d.insertAttempt(tx, d.ph(insertAttemptSQL), insertAttemptArgs(c.Attempt))
		if err != nil {
			return CommitResult{}, fmt.Errorf("commit answer: record attempt: %w", err)
		}
		res.AttemptID = id
		if err := hook("attempt"); err != nil {
			return CommitResult{}, err
		}
	}
	if c.FSRS != nil {
		for _, e := range c.FSRS.Log {
			if e.IdempotencyKey == "" {
				if c.Attempt == nil {
					return CommitResult{}, fmt.Errorf("commit answer: review_log row without a key or an attempt")
				}
				e.IdempotencyKey = fmt.Sprintf("%s:a%d", c.Attempt.SessionID, res.AttemptID)
			}
			r, err := tx.Exec(d.ph(insertReviewLogSQL), reviewLogArgs(e)...)
			if err != nil {
				return CommitResult{}, fmt.Errorf("commit answer: review_log: %w", err)
			}
			if n, _ := r.RowsAffected(); n == 0 {
				// Already committed: a retry. Nothing of this one lands.
				return CommitResult{Duplicate: true}, nil
			}
			if err := hook("log"); err != nil {
				return CommitResult{}, err
			}
		}
		if p := c.FSRS.State; p != nil {
			q := d.ph(fmt.Sprintf(writeFSRSStateSQL, "?", "?", "?", "?", "?", "?", "?", "?", "?"))
			r, err := tx.Exec(q, fsrsStateArgs(p)...)
			if err != nil {
				return CommitResult{}, fmt.Errorf("commit answer: fsrs state: %w", err)
			}
			if n, _ := r.RowsAffected(); n == 0 {
				return CommitResult{}, fmt.Errorf("commit answer: fsrs state %s/%s: %w", p.StudentID, p.ConceptID, ErrNoProgressRow)
			}
			if err := hook("state"); err != nil {
				return CommitResult{}, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return CommitResult{}, fmt.Errorf("commit answer: %w", err)
	}
	return res, nil
}

// CommitAnswer writes one graded answer in a transaction (AnswerCommit).
func (s *SQLiteStore) CommitAnswer(c AnswerCommit) (CommitResult, error) {
	return commitAnswer(s.db, commitDialect{
		ph: func(q string) string { return q },
		insertAttempt: func(tx *sql.Tx, q string, args []interface{}) (int64, error) {
			r, err := tx.Exec(q, args...)
			if err != nil {
				return 0, err
			}
			return r.LastInsertId()
		},
	}, c)
}

// CommitAnswer writes one graded answer in a transaction (AnswerCommit).
func (s *PostgresStore) CommitAnswer(c AnswerCommit) (CommitResult, error) {
	return commitAnswer(s.db, commitDialect{
		ph: dollarPlaceholders,
		insertAttempt: func(tx *sql.Tx, q string, args []interface{}) (int64, error) {
			var id int64
			err := tx.QueryRow(q+" RETURNING id", args...).Scan(&id)
			return id, err
		},
	}, c)
}
