package storage

import (
	"database/sql"
	"errors"
	"fmt"
)

// The fsrs_* columns of concept_progress have one writer: WriteFSRSState.
// UpsertProgress and UpsertProgressBatch never write them (a new row takes
// the column defaults, an existing row keeps what is stored at write time),
// because every progress writer builds its row from a snapshot read earlier
// in the request, and a whole-row rewrite of that snapshot would roll back
// a review committed in between (FSRS Phase 1 review, Phase 2 note: lost
// updates). Phase 2's review commit calls WriteFSRSState with the log row
// in one transaction.

// ErrNoProgressRow is returned by WriteFSRSState when the (learner, topic)
// has no concept_progress row: FSRS state belongs to a row that exists.
var ErrNoProgressRow = errors.New("no concept_progress row")

const writeFSRSStateSQL = `UPDATE concept_progress SET
		fsrs_stability = %s, fsrs_difficulty = %s, fsrs_last_review = %s, fsrs_reps = %s,
		fsrs_lapses = %s, fsrs_state = %s, fsrs_seed = %s
	WHERE student_id = %s AND concept_id = %s`

func fsrsStateArgs(p *ConceptProgress) []interface{} {
	return []interface{}{
		nullFloat(p.FSRSStability), nullFloat(p.FSRSDifficulty), nullTime(p.FSRSLastReview), p.FSRSReps,
		p.FSRSLapses, fsrsStateOrNone(p.FSRSState), p.FSRSSeed, p.StudentID, p.ConceptID,
	}
}

func writeFSRSState(db *sql.DB, q string, p *ConceptProgress) error {
	res, err := db.Exec(q, fsrsStateArgs(p)...)
	if err != nil {
		return fmt.Errorf("write fsrs state: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("write fsrs state %s/%s: %w", p.StudentID, p.ConceptID, ErrNoProgressRow)
	}
	return nil
}

// WriteFSRSState sets the seven fsrs_* columns of an existing row from p
// (its other fields are ignored) and nothing else.
func (s *SQLiteStore) WriteFSRSState(p *ConceptProgress) error {
	return writeFSRSState(s.db, fmt.Sprintf(writeFSRSStateSQL, "?", "?", "?", "?", "?", "?", "?", "?", "?"), p)
}

// WriteFSRSState sets the seven fsrs_* columns of an existing row from p
// (its other fields are ignored) and nothing else.
func (s *PostgresStore) WriteFSRSState(p *ConceptProgress) error {
	return writeFSRSState(s.db, fmt.Sprintf(writeFSRSStateSQL, "$1", "$2", "$3", "$4", "$5", "$6", "$7", "$8", "$9"), p)
}
