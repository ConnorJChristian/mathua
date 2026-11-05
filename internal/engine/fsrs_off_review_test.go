package engine

// Phase 1 independent reviewer: with MATHUA_FSRS off (unset), no answer
// path writes FSRS state or a review_log row. Drives practice (to mastery
// and a miss on the mastered topic), study answers and "I don't know",
// the quiz path, a review-session answer, the diagnostic placement and
// prerequisite credit, then inspects every row and the log.

import (
	"testing"

	"github.com/chuma-beep/mathua/internal/diagnostic"
)

func TestReviewer_FSRSOffWritesNothing(t *testing.T) {
	t.Setenv("MATHUA_FSRS", "")
	e := testEngine(t)
	sid := creditStudent(t, e)

	answerCorrectly(t, e, sid, 6) // masters a, credits nothing above it yet
	e.SetStudyAnchor(sid, "a", "42")
	if _, err := e.SubmitStudyAnswer(sid, "a", "7", 2.0, ""); err != nil { // miss on a mastered topic
		t.Fatal(err)
	}
	e.SetStudyAnchor(sid, "b", "99")
	if _, err := e.SubmitStudyAnswer(sid, "b", "99", 2.0, ""); err != nil {
		t.Fatal(err)
	}
	e.SetStudyAnchor(sid, "b", "99")
	if _, err := e.SubmitStudyDontKnow(sid, "b", 2.0, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SubmitQuizAnswer(sid, "a", "42", "42", 2.0, "", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SubmitQuizDontKnow(sid, "b", "99", 2.0, "", nil); err != nil {
		t.Fatal(err)
	}
	// A review-session answer, when the session offers one.
	if sess, err := e.repo.CreateSession(sid); err == nil {
		e.mu.Lock()
		e.sessions[sess.ID] = &activeSession{}
		e.mu.Unlock()
		if q, err := e.NextReviewQuestion(sess.ID, sid); err == nil && q != nil {
			e.mu.Lock()
			exp := e.sessions[sess.ID].expectedAnswer
			e.mu.Unlock()
			_, _ = e.SubmitAnswer(sess.ID, sid, q.AttemptID, exp, 2.0)
		}
	}
	other := creditStudent(t, e)
	if err := e.ApplyGoalResults(other, &diagnostic.Session{StudentID: other, Attempts: []diagnostic.Attempt{
		{ConceptID: "a", Correct: true, Fast: true}, {ConceptID: "b", Correct: false},
	}}); err != nil {
		t.Fatal(err)
	}
	e.creditPrerequisites(other, "b", true)
	e.PropagateWeakness(other)

	rows := 0
	for _, s := range []string{sid, other} {
		all, err := e.repo.GetAllProgress(s)
		if err != nil {
			t.Fatal(err)
		}
		for cid, p := range all {
			rows++
			if p.FSRSStability != nil || p.FSRSDifficulty != nil || p.FSRSLastReview != nil ||
				p.FSRSReps != 0 || p.FSRSLapses != 0 || p.FSRSState != "none" || p.FSRSSeed != "" {
				t.Errorf("%s/%s: FSRS written with the flag off: %+v", s, cid, p)
			}
		}
		log, err := e.repo.ReviewLogFor(s, "")
		if err != nil {
			t.Fatal(err)
		}
		if len(log) != 0 {
			t.Errorf("%s: %d review_log rows with the flag off", s, len(log))
		}
	}
	if rows < 3 {
		t.Fatalf("only %d progress rows: the paths under test did not run", rows)
	}
}
