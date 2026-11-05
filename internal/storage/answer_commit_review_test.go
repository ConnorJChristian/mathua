package storage

// FSRS Phase 2 independent reviewer: CommitAnswer on SQLite and (with
// TEST_POSTGRES_DSN) Postgres. A failure between the progress write and
// the review_log write rolls back both, the row keeping its old values; a
// failure after the log write (the state's row missing) rolls back the
// log too; a retry with a stored key writes nothing at all; rows in one
// second read back in time order, then insertion order.

import (
	"errors"
	"testing"
	"time"
)

func rvCommit(sid, sess, cid, key, status string, at time.Time) AnswerCommit {
	p := fsrsRow(sid, cid)
	p.Status, p.Attempts = status, 10
	p.FSRSStability, p.FSRSDifficulty, p.FSRSLastReview = nil, nil, nil
	p.FSRSReps, p.FSRSLapses, p.FSRSState, p.FSRSSeed = 0, 0, "none", ""
	s, d := 2.3065, 2.118103970459016
	state := *p
	state.FSRSStability, state.FSRSDifficulty, state.FSRSLastReview = &s, &d, &at
	state.FSRSReps, state.FSRSState, state.FSRSSeed = 1, "review", "mastery"
	return AnswerCommit{
		Progress: p,
		Attempt: &AttemptEntry{SessionID: sess, StudentID: sid, ConceptID: cid, Answer: status, Expected: "4",
			Correct: true, ElapsedSeconds: 12, Timestamp: at, Source: "review"},
		FSRS: &FSRSCommit{
			Log: []ReviewLogEntry{{StudentID: sid, ConceptID: cid, At: at, Source: "review", Rating: 3,
				SAfter: s, DAfter: d, IdempotencyKey: key, EngineMode: "shadow"}},
			State: &state,
		},
	}
}

type rvSnap struct {
	status   string
	attempts int // the row's counter
	rows     int // attempt rows
	logs     int
	fsrs     string
}

func rvTake(t *testing.T, repo Repository, sid, cid string) rvSnap {
	t.Helper()
	var s rvSnap
	if p, err := repo.GetProgress(sid, cid); err != nil {
		t.Fatal(err)
	} else if p != nil {
		s.status, s.attempts, s.fsrs = p.Status, p.Attempts, p.FSRSState
	}
	atts, err := repo.GetAttemptsForStudent(sid)
	if err != nil {
		t.Fatal(err)
	}
	s.rows = len(atts)
	l, err := repo.ReviewLogFor(sid, cid)
	if err != nil {
		t.Fatal(err)
	}
	s.logs = len(l)
	return s
}

func TestReview2_CommitFailureBetweenProgressAndLogRollsBackBoth(t *testing.T) {
	for name, repo := range fsrsStores(t) {
		sid := commitFixture(t, repo)
		sess, err := repo.CreateSession(sid)
		if err != nil {
			t.Fatal(err)
		}
		const cid = "calc.deriv.partial"
		old := fsrsRow(sid, cid)
		old.Status, old.Attempts = "PRACTICING", 2
		old.FSRSStability, old.FSRSDifficulty, old.FSRSLastReview, old.FSRSState, old.FSRSSeed, old.FSRSReps, old.FSRSLapses = nil, nil, nil, "none", "", 0, 0
		if err := repo.UpsertProgress(old); err != nil {
			t.Fatal(err)
		}
		before := rvTake(t, repo, sid, cid)
		at := time.Date(2026, 11, 5, 17, 0, 0, 123456789, time.UTC)

		// 1. The hook fires after the progress upsert and the attempt insert,
		// before the first review_log row.
		injected := errors.New("injected between progress and review_log")
		commitHook = func(stage string) error {
			if stage == "attempt" {
				return injected
			}
			return nil
		}
		_, err = repo.CommitAnswer(rvCommit(sid, sess.ID, cid, sess.ID+":r1", "MASTERED", at))
		commitHook = nil
		if !errors.Is(err, injected) {
			t.Fatalf("[%s] err %v, want the injected one", name, err)
		}
		if got := rvTake(t, repo, sid, cid); got != before {
			t.Fatalf("[%s] after a failure before the log: %+v, want %+v", name, got, before)
		}

		// 2. A real failure after the log row: the state names a row that
		// does not exist. The log row and the progress change go too.
		c := rvCommit(sid, sess.ID, cid, sess.ID+":r2", "MASTERED", at)
		c.FSRS.State.ConceptID = "no.such.row"
		if _, err := repo.CommitAnswer(c); !errors.Is(err, ErrNoProgressRow) {
			t.Fatalf("[%s] err %v, want ErrNoProgressRow", name, err)
		}
		if got := rvTake(t, repo, sid, cid); got != before {
			t.Fatalf("[%s] after a failure after the log: %+v, want %+v", name, got, before)
		}
		// The key of the failed commit is free: the answer can be committed.
		if res, err := repo.CommitAnswer(rvCommit(sid, sess.ID, cid, sess.ID+":r2", "MASTERED", at)); err != nil || res.Duplicate {
			t.Fatalf("[%s] commit after failures: %+v %v", name, res, err)
		}
		if got := rvTake(t, repo, sid, cid); got.status != "MASTERED" || got.rows != before.rows+1 || got.logs != 1 || got.fsrs != "review" {
			t.Fatalf("[%s] after the good commit: %+v", name, got)
		}
	}
}

func TestReview2_RetrySameKeyWritesNothing(t *testing.T) {
	for name, repo := range fsrsStores(t) {
		sid := commitFixture(t, repo)
		sess, err := repo.CreateSession(sid)
		if err != nil {
			t.Fatal(err)
		}
		const cid = "calc.deriv.partial"
		at := time.Date(2026, 11, 5, 17, 0, 0, 0, time.UTC)
		key := sess.ID + ":r1"
		first, err := repo.CommitAnswer(rvCommit(sid, sess.ID, cid, key, "MASTERED", at))
		if err != nil || first.Duplicate || first.AttemptID == 0 {
			t.Fatalf("[%s] first: %+v %v", name, first, err)
		}
		after1 := rvTake(t, repo, sid, cid)
		// The retry carries different values (a re-grade): none may land.
		retry := rvCommit(sid, sess.ID, cid, key, "LEARNING", at.Add(3*time.Second))
		retry.Progress.Attempts = 99
		*retry.FSRS.State.FSRSStability = 0.5
		res, err := repo.CommitAnswer(retry)
		if err != nil || !res.Duplicate || res.AttemptID != 0 {
			t.Fatalf("[%s] retry: %+v %v, want Duplicate", name, res, err)
		}
		if got := rvTake(t, repo, sid, cid); got != after1 {
			t.Errorf("[%s] the retry changed rows: %+v, want %+v", name, got, after1)
		}
		p, _ := repo.GetProgress(sid, cid)
		if p.FSRSStability == nil || *p.FSRSStability != 2.3065 {
			t.Errorf("[%s] the retry moved S: %v", name, p.FSRSStability)
		}
		l, _ := repo.ReviewLogFor(sid, cid)
		if len(l) != 1 || !l[0].At.Equal(at) {
			t.Errorf("[%s] log %+v", name, l)
		}
	}
}

func TestReview2_SameSecondRowsReadBackInTimeOrder(t *testing.T) {
	for name, repo := range fsrsStores(t) {
		sid := commitFixture(t, repo)
		sess, err := repo.CreateSession(sid)
		if err != nil {
			t.Fatal(err)
		}
		const cid = "calc.deriv.partial"
		if err := repo.UpsertProgress(fsrsRow(sid, cid)); err != nil {
			t.Fatal(err)
		}
		base := time.Date(2026, 11, 5, 17, 0, 0, 0, time.UTC)
		// Inserted out of time order, mixing a whole second and fractions,
		// then two at the identical instant.
		ins := []struct {
			off time.Duration
			r   int
		}{{700 * time.Millisecond, 1}, {0, 2}, {200 * time.Millisecond, 3}, {900 * time.Millisecond, 4}, {900 * time.Millisecond, 1}}
		for i, x := range ins {
			e := ReviewLogEntry{StudentID: sid, ConceptID: cid, At: base.Add(x.off), Source: "review", Rating: x.r,
				SAfter: 1, DAfter: 5, IdempotencyKey: sess.ID + ":o" + string(rune('a'+i)), EngineMode: "shadow"}
			if _, err := repo.CommitAnswer(AnswerCommit{FSRS: &FSRSCommit{Log: []ReviewLogEntry{e}}}); err != nil {
				t.Fatalf("[%s] %v", name, err)
			}
		}
		l, err := repo.ReviewLogFor(sid, cid)
		if err != nil {
			t.Fatal(err)
		}
		want := []int{2, 3, 1, 4, 1} // 0, .2, .7, .9 (first inserted), .9
		if len(l) != len(want) {
			t.Fatalf("[%s] %d rows", name, len(l))
		}
		for i, e := range l {
			if e.Rating != want[i] {
				t.Errorf("[%s] row %d: rating %d at %s, want rating %d", name, i, e.Rating, e.At.Format(time.RFC3339Nano), want[i])
			}
		}
	}
}
