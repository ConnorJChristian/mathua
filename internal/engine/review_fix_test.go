package engine

import (
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/mastery"
	"github.com/chuma-beep/mathua/internal/storage"
)

// A mastered topic whose review is due is served in a review session, and a
// correct review answer reschedules it (fork fix).
func TestDueReviewOfMasteredTopicIsServedAndRescheduled(t *testing.T) {
	e := testEngine(t)
	st, _ := e.CreateStudent("r")
	past := time.Now().Add(-48 * time.Hour)
	due := time.Now().Add(-time.Hour)
	for _, id := range []string{"a", "b"} {
		if err := e.repo.UpsertProgress(&storage.ConceptProgress{
			StudentID: st.ID, ConceptID: id, Status: string(mastery.StatusMastered), Attempts: 6, Streak: 3,
			LastAttempted: &past, LastReviewed: &past, MasteredAt: &past, NextReviewDue: &due, SM2EFactor: 2.5, SM2Interval: 1,
		}); err != nil {
			t.Fatal(err)
		}
	}
	n, err := e.ReviewsDue(st.ID)
	if err != nil || n != 2 {
		t.Fatalf("due = %d (%v), want 2", n, err)
	}
	sess, err := e.repo.CreateSession(st.ID)
	if err != nil {
		t.Fatal(err)
	}
	q, err := e.NextReviewQuestion(sess.ID, st.ID)
	if err != nil || q == nil {
		t.Fatalf("no review served for a due mastered topic: %v %v", q, err)
	}
	answer := "42"
	if q.ConceptID == "b" {
		answer = "99"
	}
	res, err := e.SubmitAnswer(sess.ID, st.ID, q.AttemptID, answer, 5)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Correct {
		t.Fatalf("the right answer graded wrong: %+v", res)
	}
	p, _ := e.repo.GetProgress(st.ID, q.ConceptID)
	if res.Correct {
		if p.NextReviewDue == nil || !p.NextReviewDue.After(time.Now()) {
			t.Fatalf("a correct review must push the next review into the future: %v", p.NextReviewDue)
		}
		if p.Status != string(mastery.StatusMastered) {
			t.Fatalf("status changed: %s", p.Status)
		}
	}
	n, _ = e.ReviewsDue(st.ID)
	if n != 1 {
		t.Fatalf("after one review, due = %d, want 1", n)
	}
}

// A self-graded card never moves the ladder.
func TestSelfGradedCardDoesNotAdvanceMastery(t *testing.T) {
	e := testEngine(t)
	st, _ := e.CreateStudent("c")
	p := &storage.ConceptProgress{StudentID: st.ID, ConceptID: "a", Status: string(mastery.StatusPracticing), Streak: 2, Attempts: 4, SM2EFactor: 2.5}
	for i := 0; i < 6; i++ {
		if _, _, err := e.applyGradedAttempt(p, gradedAttempt{studentID: st.ID, conceptID: "a", requiredStreak: 3, timeThreshold: 60, elapsed: 3, correct: true, selfGraded: true}); err != nil {
			t.Fatal(err)
		}
	}
	if p.Status != string(mastery.StatusPracticing) || p.Streak != 2 {
		t.Fatalf("self-graded cards moved the ladder: status=%s streak=%d", p.Status, p.Streak)
	}
	if p.Attempts != 10 {
		t.Fatalf("attempts should still count: %d", p.Attempts)
	}
}

// The mastery test needs new ground and a night between tests, not XP alone.
func TestQuizGateNeedsTopicsAndAGap(t *testing.T) {
	e := testEngine(t)
	st, _ := e.CreateStudent("g")
	if err := e.repo.AwardXP(st.ID, QuizGateXP, "test", "k1"); err != nil {
		t.Fatal(err)
	}
	if due, _ := e.QuizDue(st.ID); !due {
		t.Fatal("first test: XP alone opens the gate")
	}
	if err := e.RecordQuizCompletion(st.ID); err != nil {
		t.Fatal(err)
	}
	if err := e.repo.AwardXP(st.ID, QuizGateXP, "test", "k2"); err != nil {
		t.Fatal(err)
	}
	if due, _ := e.QuizDue(st.ID); due {
		t.Fatal("a second test the same hour with no new topics must not be due")
	}
}
