package storage

import (
	"errors"
	"testing"
	"time"
)

// CommitAnswer (FSRS Phase 2, step 2): one transaction writes the progress
// row, the attempt and, in shadow or active mode, the review_log rows and
// the fsrs_* columns. They land together, roll back together, and a retry
// (a key already stored) writes nothing at all. Both stores.

func commitFixture(t *testing.T, repo Repository) (sid string) {
	t.Helper()
	st, err := repo.CreateStudent("commit")
	if err != nil {
		t.Fatal(err)
	}
	return st.ID
}

func shadowCommit(sid, sess, key string) AnswerCommit {
	at := time.Date(2026, 11, 5, 17, 0, 20, 0, time.UTC)
	p := fsrsRow(sid, "calc.deriv.partial")
	p.FSRSStability, p.FSRSDifficulty, p.FSRSLastReview = nil, nil, nil
	p.FSRSReps, p.FSRSLapses, p.FSRSState, p.FSRSSeed = 0, 0, "none", ""
	s, d := 2.3065, 2.118103970459016
	state := *p
	state.FSRSStability, state.FSRSDifficulty, state.FSRSLastReview = &s, &d, &at
	state.FSRSReps, state.FSRSState, state.FSRSSeed = 1, "review", "mastery"
	return AnswerCommit{
		Progress: p,
		Attempt: &AttemptEntry{SessionID: sess, StudentID: sid, ConceptID: p.ConceptID, Answer: "4", Expected: "4",
			Correct: true, ElapsedSeconds: 12, Timestamp: at, Source: "practice"},
		FSRS: &FSRSCommit{
			Log: []ReviewLogEntry{{StudentID: sid, ConceptID: p.ConceptID, At: at, Source: "first_mastery", Rating: 3,
				SAfter: s, DAfter: d, IdempotencyKey: key, EngineMode: "shadow"}},
			State: &state,
		},
	}
}

func counts(t *testing.T, repo Repository, sid string) (progress, attempts, logs int, state string) {
	t.Helper()
	all, err := repo.GetAllProgress(sid)
	if err != nil {
		t.Fatal(err)
	}
	if p := all["calc.deriv.partial"]; p != nil {
		state = p.FSRSState
	}
	atts, err := repo.GetAttemptsForStudent(sid)
	if err != nil {
		t.Fatal(err)
	}
	log, err := repo.ReviewLogFor(sid, "")
	if err != nil {
		t.Fatal(err)
	}
	return len(all), len(atts), len(log), state
}

func TestCommitAnswer_WritesAllTogether(t *testing.T) {
	for name, repo := range fsrsStores(t) {
		sid := commitFixture(t, repo)
		sess, err := repo.CreateSession(sid)
		if err != nil {
			t.Fatal(err)
		}
		c := shadowCommit(sid, sess.ID, sess.ID+":attempt-1")
		res, err := repo.CommitAnswer(c)
		if err != nil || res.Duplicate || res.AttemptID == 0 {
			t.Fatalf("[%s] commit: %+v %v", name, res, err)
		}
		if p, a, l, st := counts(t, repo, sid); p != 1 || a != 1 || l != 1 || st != "review" {
			t.Fatalf("[%s] after commit: %d rows, %d attempts, %d log, state %q", name, p, a, l, st)
		}
		got, _ := repo.GetProgress(sid, "calc.deriv.partial")
		sameFSRS(t, name, c.FSRS.State, got)
		if got.Status != "MASTERED" || got.Attempts != 9 {
			t.Errorf("[%s] progress columns not written: %+v", name, got)
		}

		// A retry with the same key writes nothing: no second attempt, no
		// progress change, no log row.
		again := shadowCommit(sid, sess.ID, sess.ID+":attempt-1")
		again.Progress.Attempts = 99
		res, err = repo.CommitAnswer(again)
		if err != nil || !res.Duplicate || res.AttemptID != 0 {
			t.Fatalf("[%s] retry: %+v %v, want Duplicate", name, res, err)
		}
		if p, a, l, _ := counts(t, repo, sid); p != 1 || a != 1 || l != 1 {
			t.Fatalf("[%s] after retry: %d rows, %d attempts, %d log", name, p, a, l)
		}
		if got, _ := repo.GetProgress(sid, "calc.deriv.partial"); got.Attempts != 9 {
			t.Errorf("[%s] a retry rewrote progress: attempts %d", name, got.Attempts)
		}

		// An empty key is the attempt's own: <session>:a<attempt id>.
		third := shadowCommit(sid, sess.ID, "")
		res, err = repo.CommitAnswer(third)
		if err != nil || res.Duplicate {
			t.Fatalf("[%s] commit with the attempt's key: %+v %v", name, res, err)
		}
		log, _ := repo.ReviewLogFor(sid, "")
		if want := sess.ID + ":a" + itoa64(res.AttemptID); log[len(log)-1].IdempotencyKey != want {
			t.Errorf("[%s] key %q, want %q", name, log[len(log)-1].IdempotencyKey, want)
		}
	}
}

func itoa64(n int64) string {
	b := []byte{}
	if n == 0 {
		return "0"
	}
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// Off mode: no FSRS half, so nothing FSRS is written.
func TestCommitAnswer_OffWritesNothingFSRS(t *testing.T) {
	for name, repo := range fsrsStores(t) {
		sid := commitFixture(t, repo)
		sess, _ := repo.CreateSession(sid)
		c := shadowCommit(sid, sess.ID, "k-off")
		c.FSRS = nil
		res, err := repo.CommitAnswer(c)
		if err != nil || res.AttemptID == 0 {
			t.Fatalf("[%s] %+v %v", name, res, err)
		}
		if p, a, l, st := counts(t, repo, sid); p != 1 || a != 1 || l != 0 || st != "none" {
			t.Fatalf("[%s] off: %d rows, %d attempts, %d log, state %q", name, p, a, l, st)
		}
	}
}

// A failure at any stage rolls the whole answer back: no progress row, no
// attempt, no log row, no state.
func TestCommitAnswer_RollsBackTogether(t *testing.T) {
	boom := errors.New("injected")
	for _, stage := range []string{"progress", "attempt", "log", "state"} {
		for name, repo := range fsrsStores(t) {
			sid := commitFixture(t, repo)
			sess, _ := repo.CreateSession(sid)
			commitHook = func(s string) error {
				if s == stage {
					return boom
				}
				return nil
			}
			_, err := repo.CommitAnswer(shadowCommit(sid, sess.ID, "k-"+stage+"-"+name+"-"+sid))
			commitHook = nil
			if !errors.Is(err, boom) {
				t.Fatalf("[%s] stage %s: err %v", name, stage, err)
			}
			if p, a, l, _ := counts(t, repo, sid); p != 0 || a != 0 || l != 0 {
				t.Errorf("[%s] failure at %s left %d rows, %d attempts, %d log rows", name, stage, p, a, l)
			}
		}
	}
}

// A held event is stored with its reason and read back.
func TestReviewLog_HeldRoundTrips(t *testing.T) {
	for name, repo := range fsrsStores(t) {
		sid := commitFixture(t, repo)
		at := time.Date(2026, 11, 5, 17, 0, 0, 0, time.UTC)
		if _, err := repo.InsertReviewLog(ReviewLogEntry{StudentID: sid, ConceptID: "x", At: at, Source: "review",
			Rating: 3, IdempotencyKey: "held-" + sid, EngineMode: "shadow", Held: "daily_cap"}); err != nil {
			t.Fatal(err)
		}
		log, err := repo.ReviewLogFor(sid, "x")
		if err != nil || len(log) != 1 || log[0].Held != "daily_cap" {
			t.Errorf("[%s] %+v %v", name, log, err)
		}
	}
}

// FSRS Phase 2 review, M1: a seed row already stored (scripts/fsrs_seed
// -apply ran between the engine's read and its commit) is skipped on its
// own; the answer, whose own key is new, commits in full. A commit of seed
// rows alone stays all-or-nothing: any stored key makes it a duplicate.
func TestCommitAnswer_StoredSeedRowIsSkippedNotARetry(t *testing.T) {
	for name, repo := range fsrsStores(t) {
		sid := commitFixture(t, repo)
		sess, err := repo.CreateSession(sid)
		if err != nil {
			t.Fatal(err)
		}
		c := shadowCommit(sid, sess.ID, sess.ID+":attempt-1")
		seed := c.FSRS.Log[0]
		seed.IdempotencyKey = "seed:a17"
		seed.At = seed.At.Add(-time.Hour)
		// The script's commit: the seed row alone.
		if res, err := repo.CommitAnswer(AnswerCommit{FSRS: &FSRSCommit{Log: []ReviewLogEntry{seed}}}); err != nil || res.Duplicate {
			t.Fatalf("[%s] seed commit: %+v %v", name, res, err)
		}
		// The engine's commit carries the same seed row, then the answer's.
		c.FSRS.Log = append([]ReviewLogEntry{seed}, c.FSRS.Log...)
		res, err := repo.CommitAnswer(c)
		if err != nil || res.Duplicate || res.AttemptID == 0 || res.SkippedSeedRows != 1 {
			t.Fatalf("[%s] answer after the seed: %+v %v; want committed with 1 seed row skipped", name, res, err)
		}
		if p, a, l, st := counts(t, repo, sid); p != 1 || a != 1 || l != 2 || st != "review" {
			t.Errorf("[%s] after: %d rows, %d attempts, %d log, state %q", name, p, a, l, st)
		}
		// The seed again on its own: a duplicate, nothing written.
		res, err = repo.CommitAnswer(AnswerCommit{FSRS: &FSRSCommit{Log: []ReviewLogEntry{seed}, State: c.FSRS.State}})
		if err != nil || !res.Duplicate {
			t.Errorf("[%s] seed-only recommit: %+v %v, want Duplicate", name, res, err)
		}
		// The answer's own key again: a retry.
		res, err = repo.CommitAnswer(c)
		if err != nil || !res.Duplicate {
			t.Errorf("[%s] answer retry: %+v %v, want Duplicate", name, res, err)
		}
		if _, a, l, _ := counts(t, repo, sid); a != 1 || l != 2 {
			t.Errorf("[%s] retries wrote: %d attempts, %d log", name, a, l)
		}
	}
}
