package engine

import (
	"errors"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/clock"
	"github.com/chuma-beep/mathua/internal/fsrs"
	"github.com/chuma-beep/mathua/internal/mastery"
	"github.com/chuma-beep/mathua/internal/review"
	"github.com/chuma-beep/mathua/internal/storage"
)

// Shadow scheduling (FSRS Phase 2, plan 3.3): which answers write an FSRS
// event, with what rating, and that the log replays to the stored state.

// shadowEngine is testEngine in shadow mode with the default rules, on a
// simulated clock at 09:00 learner time.
func shadowEngine(t *testing.T) (*Engine, string, *clock.Sim) {
	t.Helper()
	sim := clock.NewSim(time.Date(2026, 11, 9, 17, 0, 0, 0, time.UTC))
	t.Cleanup(sim.Install())
	e := testEngine(t)
	e.SetFSRSMode(review.ModeShadow)
	e.SetFSRSRules(review.DefaultRules())
	return e, creditStudent(t, e), sim
}

// reviewAnswer answers one review-session question on cid.
func reviewAnswer(t *testing.T, e *Engine, sid, cid, answer, attemptID string) *AnswerResult {
	t.Helper()
	sess, err := e.repo.CreateSession(sid)
	if err != nil {
		t.Fatal(err)
	}
	e.mu.Lock()
	e.sessions[sess.ID] = &activeSession{studentID: sid, conceptID: cid, expectedAnswer: map[string]string{"a": "42", "b": "99"}[cid],
		attemptID: attemptID, isReview: true, requiredStreak: 3, timeThreshold: 60, questionText: "What is the answer?"}
	e.mu.Unlock()
	res, err := e.SubmitAnswer(sess.ID, sid, attemptID, answer, 5)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func logOf(t *testing.T, e *Engine, sid, cid string) []storage.ReviewLogEntry {
	t.Helper()
	l, err := e.repo.ReviewLogFor(sid, cid)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

// replayAgrees is the doctor's event-sourcing check on every row.
func replayAgrees(t *testing.T, e *Engine, sid string) {
	t.Helper()
	all, _ := e.repo.GetAllProgress(sid)
	for cid, p := range all {
		m, last := FSRSMemory(p)
		var ev []review.Event
		for _, x := range logOf(t, e, sid, cid) {
			ev = append(ev, review.Event{At: x.At, Rating: fsrs.Rating(x.Rating), ElapsedDays: x.ElapsedDays, Source: x.Source, SAfter: x.SAfter, DAfter: x.DAfter, Held: x.Held})
		}
		got, gotLast, err := review.ReplayLast(review.DefaultParams(), ev)
		if err != nil {
			t.Fatalf("%s: %v", cid, err)
		}
		if d := review.Diff(m, got); d != "" {
			t.Errorf("%s: stored state is not the replay of its log: %s", cid, d)
		}
		if !m.IsZero() && !gotLast.Truncate(time.Second).Equal(last.Truncate(time.Second)) {
			t.Errorf("%s: fsrs_last_review %v, replay %v", cid, last, gotLast)
		}
	}
}

func TestShadow_FirstMasteryInitialisesWithTheReference(t *testing.T) {
	e, sid, _ := shadowEngine(t)
	answerCorrectly(t, e, sid, 6)
	p, _ := e.repo.GetProgress(sid, "a")
	if p.Status != string(mastery.StatusMastered) {
		t.Fatalf("a not mastered: %s", p.Status)
	}
	log := logOf(t, e, sid, "a")
	if len(log) != 1 || log[0].Source != "first_mastery" || log[0].Rating != int(fsrs.Good) || log[0].EngineMode != "shadow" {
		t.Fatalf("log %+v, want one first_mastery Good", log)
	}
	m, _ := FSRSMemory(p)
	if m != review.Initial(fsrs.Good) || p.FSRSSeed != "mastery" || m.Stability != fsrs.DefaultParameters()[2] {
		t.Errorf("state %+v seed %q, want the reference first Good (S = w[2])", m, p.FSRSSeed)
	}
	// Practice right on the mastered topic records evidence only.
	for i := 0; i < 2; i++ {
		e.SetStudyAnchor(sid, "a", "42")
		if _, err := e.SubmitStudyAnswer(sid, "a", "42", 5, ""); err != nil {
			t.Fatal(err)
		}
	}
	if n := len(logOf(t, e, sid, "a")); n != 1 {
		t.Errorf("practice right on a mastered topic wrote %d log rows, want 1", n)
	}
	replayAgrees(t, e, sid)
}

func TestShadow_ReviewAnswersAndKeys(t *testing.T) {
	e, sid, sim := shadowEngine(t)
	answerCorrectly(t, e, sid, 6)
	sim.Advance(3 * 24 * time.Hour)
	reviewAnswer(t, e, sid, "a", "42", "r1")
	sim.Advance(4 * 24 * time.Hour)
	reviewAnswer(t, e, sid, "a", "7", "r2")
	log := logOf(t, e, sid, "a")
	if len(log) != 3 {
		t.Fatalf("log %+v", log)
	}
	right, wrong := log[1], log[2]
	if right.Source != "review" || right.Rating != int(fsrs.Good) || right.ElapsedDays != 3 || right.SAfter <= right.SBefore || right.RAtReview <= 0 || right.RAtReview >= 1 {
		t.Errorf("right review %+v", right)
	}
	if wrong.Rating != int(fsrs.Again) || wrong.SAfter >= wrong.SBefore || wrong.Held != "" {
		t.Errorf("wrong review %+v", wrong)
	}
	if right.IdempotencyKey[len(right.IdempotencyKey)-3:] != ":r1" {
		t.Errorf("key %q, want <session>:r1", right.IdempotencyKey)
	}
	p, _ := e.repo.GetProgress(sid, "a")
	if p.FSRSLapses != 1 || p.FSRSState != "relearning" || p.FSRSReps != 3 {
		t.Errorf("after a lapse: %+v", p)
	}
	replayAgrees(t, e, sid)
}

func TestShadow_PracticeLapseThreshold(t *testing.T) {
	e, sid, sim := shadowEngine(t)
	answerCorrectly(t, e, sid, 6)
	// A miss an hour later: R is 1, at or above 0.95, so held.
	sim.Advance(time.Hour)
	e.SetStudyAnchor(sid, "a", "42")
	if _, err := e.SubmitStudyAnswer(sid, "a", "7", 5, ""); err != nil {
		t.Fatal(err)
	}
	log := logOf(t, e, sid, "a")
	if len(log) != 2 || log[1].Source != "practice_lapse" || log[1].Held != review.HeldLapseRHigh || log[1].SAfter != log[1].SBefore {
		t.Fatalf("an early miss: %+v", log)
	}
	// Six days on, R is well below 0.95: a lapse.
	sim.Advance(6 * 24 * time.Hour)
	e.SetStudyAnchor(sid, "a", "42")
	if _, err := e.SubmitStudyAnswer(sid, "a", "7", 5, ""); err != nil {
		t.Fatal(err)
	}
	log = logOf(t, e, sid, "a")
	if p, _ := e.repo.GetProgress(sid, "a"); mastery.Status(p.Status) == mastery.StatusMastered {
		if len(log) != 3 || log[2].Held != "" || log[2].Rating != int(fsrs.Again) || log[2].RAtReview >= 0.95 {
			t.Fatalf("a late miss: %+v", log)
		}
	}
	// Off by rule: no event at all.
	e.SetFSRSRules(func() review.Rules { r := review.DefaultRules(); r.PracticeLapse = false; return r }())
	sim.Advance(6 * 24 * time.Hour)
	n := len(logOf(t, e, sid, "a"))
	e.SetStudyAnchor(sid, "a", "42")
	e.SubmitStudyAnswer(sid, "a", "7", 5, "")
	if m := len(logOf(t, e, sid, "a")); m != n {
		t.Errorf("practice_lapse=off still wrote %d rows", m-n)
	}
	replayAgrees(t, e, sid)
}

func TestShadow_QuizSourcesAndRules(t *testing.T) {
	e, sid, sim := shadowEngine(t)
	answerCorrectly(t, e, sid, 6)
	sim.Advance(2 * 24 * time.Hour)
	if _, err := e.SubmitQuizAnswer(sid, "a", "42", "42", 5, "q", nil); err != nil {
		t.Fatal(err)
	}
	sim.Advance(2 * 24 * time.Hour)
	if _, err := e.SubmitCourseTestAnswer(sid, "a", "42", "42", 5, "q", nil); err != nil {
		t.Fatal(err)
	}
	sim.Advance(2 * 24 * time.Hour)
	if _, err := e.SubmitCourseTestDontKnow(sid, "a", "42", 5, "q", nil); err != nil {
		t.Fatal(err)
	}
	log := logOf(t, e, sid, "a")
	if len(log) != 4 || log[1].Source != "mastery_check" || log[2].Source != "course_test" || log[3].Source != "course_test" || log[3].Rating != int(fsrs.Again) {
		t.Fatalf("log %+v", log)
	}
	// course_tests=off and mastery_check=off: no events.
	r := review.DefaultRules()
	r.CourseTests, r.MasteryCheck = false, false
	e.SetFSRSRules(r)
	sim.Advance(2 * 24 * time.Hour)
	e.SubmitCourseTestAnswer(sid, "a", "42", "42", 5, "q", nil)
	e.SubmitQuizAnswer(sid, "a", "42", "42", 5, "q", nil)
	if n := len(logOf(t, e, sid, "a")); n != 4 {
		t.Errorf("rules off still wrote %d rows", n-4)
	}
	replayAgrees(t, e, sid)
}

func TestShadow_SelfGradedRecallIsNoEvent(t *testing.T) {
	e, sid, sim := shadowEngine(t)
	answerCorrectly(t, e, sid, 6)
	sim.Advance(3 * 24 * time.Hour)
	if _, err := e.SubmitQuizAnswer(sid, "a", "wrong", "recall:card-1", 5, "q", nil); err != nil {
		t.Fatal(err)
	}
	if n := len(logOf(t, e, sid, "a")); n != 1 {
		t.Errorf("a self-graded card wrote %d FSRS rows", n-1)
	}
}

// One stability-raising event per topic per learner-local day: after the
// first mastery (which raised S), a mastery-check miss then a right answer
// the same day relearns nothing more; the next day it does.
func TestShadow_DailyCap(t *testing.T) {
	e, sid, sim := shadowEngine(t)
	answerCorrectly(t, e, sid, 6)
	sim.Advance(time.Hour)
	e.SubmitQuizAnswer(sid, "a", "7", "42", 5, "q", nil)  // Again: lowers S, never held
	e.SubmitQuizAnswer(sid, "a", "42", "42", 5, "q", nil) // Good after Again: would raise
	log := logOf(t, e, sid, "a")
	if len(log) != 3 || log[1].Held != "" || log[2].Held != review.HeldDailyCap || log[2].SAfter != log[2].SBefore {
		t.Fatalf("same day: %+v", log)
	}
	sim.Advance(24 * time.Hour)
	e.SubmitQuizAnswer(sid, "a", "42", "42", 5, "q", nil)
	log = logOf(t, e, sid, "a")
	if last := log[len(log)-1]; last.Held != "" || last.SAfter <= last.SBefore {
		t.Errorf("next day: %+v", last)
	}
	replayAgrees(t, e, sid)
}

// A row mastered before shadow mode (no state, attempts of its own) is
// seeded from those attempts at its first event; a credited row with no
// attempts gets its state from this first answer.
func TestShadow_SeedOnFirstTouch(t *testing.T) {
	e, sid, sim := shadowEngine(t)
	e.SetFSRSMode(review.ModeOff)
	answerCorrectly(t, e, sid, 6) // mastered in off mode: no state
	sim.Advance(2 * 24 * time.Hour)
	e.SetStudyAnchor(sid, "a", "42")
	e.SubmitStudyAnswer(sid, "a", "7", 5, "") // an off-mode miss, two days on
	if n := len(logOf(t, e, sid, "a")); n != 0 {
		t.Fatalf("off wrote %d rows", n)
	}
	e.SetFSRSMode(review.ModeShadow)
	sim.Advance(5 * 24 * time.Hour)
	reviewAnswer(t, e, sid, "a", "42", "r1")
	log := logOf(t, e, sid, "a")
	if len(log) < 2 || log[0].Source != "first_mastery" || log[0].IdempotencyKey[:6] != "seed:a" || log[len(log)-1].Source != "review" {
		t.Fatalf("seeded log %+v", log)
	}
	p, _ := e.repo.GetProgress(sid, "a")
	if p.FSRSSeed != "mastery" || p.FSRSReps < 2 {
		t.Errorf("seeded row %+v", p)
	}
	replayAgrees(t, e, sid)

	// A credited row: no attempts, mastered by placement.
	last := clock.Now().AddDate(0, 0, -20)
	if err := e.repo.UpsertProgress(&storage.ConceptProgress{StudentID: sid, ConceptID: "b", Status: string(mastery.StatusMastered),
		PlacementSeeded: true, LastReviewed: &last, SM2EFactor: 2.5}); err != nil {
		t.Fatal(err)
	}
	reviewAnswer(t, e, sid, "b", "1", "r2")
	lb := logOf(t, e, sid, "b")
	pb, _ := e.repo.GetProgress(sid, "b")
	if len(lb) != 1 || lb[0].Rating != int(fsrs.Again) || pb.FSRSSeed != "placement" || pb.FSRSState != "relearning" {
		t.Errorf("credited row's first answer: %+v %+v", lb, pb)
	}
	replayAgrees(t, e, sid)
}

func TestShadow_RetryIsDuplicate(t *testing.T) {
	e, sid, _ := shadowEngine(t)
	answerCorrectly(t, e, sid, 6)
	p, _ := e.repo.GetProgress(sid, "a")
	before := *p
	sess, _ := e.repo.CreateSession(sid)
	att := &storage.AttemptEntry{SessionID: sess.ID, StudentID: sid, ConceptID: "a", Answer: "42", Expected: "42", Correct: true, Timestamp: clock.Now(), Source: "review"}
	o := gradedAttempt{studentID: sid, conceptID: "a", correct: true, attempt: att, fsrs: fsrsAnswer{source: fsrsSrcReview, expected: "42", key: sess.ID + ":x"}}
	if _, err := e.commitAnswer(p, &before, o); err != nil {
		t.Fatal(err)
	}
	if _, err := e.commitAnswer(p, &before, o); !errors.Is(err, ErrDuplicateAnswer) {
		t.Fatalf("retry: %v, want ErrDuplicateAnswer", err)
	}
	atts, _ := e.repo.GetAttemptsForStudent(sid)
	n := 0
	for _, a := range atts {
		if a.SessionID == sess.ID {
			n++
		}
	}
	if n != 1 {
		t.Errorf("the retry recorded %d attempts in the session, want 1", n)
	}
}

// scripts/fsrs_seed's plan is the first-touch seed: the same rows, the same
// state; applying it twice writes once; a credited row is left alone.
func TestFSRSSeedPlan(t *testing.T) {
	e, sid, sim := shadowEngine(t)
	e.SetFSRSMode(review.ModeOff)
	answerCorrectly(t, e, sid, 6)
	last := clock.Now().AddDate(0, 0, -20)
	e.repo.UpsertProgress(&storage.ConceptProgress{StudentID: sid, ConceptID: "b", Status: string(mastery.StatusMastered), PlacementSeeded: true, LastReviewed: &last, SM2EFactor: 2.5})
	sim.Advance(4 * 24 * time.Hour)
	e.SetFSRSMode(review.ModeShadow)
	seeds, none, err := e.FSRSSeedPlan(sid, clock.Now())
	if err != nil || len(seeds) != 1 || seeds[0].ConceptID != "a" || len(none) != 1 || none[0] != "b" {
		t.Fatalf("plan %+v none %v err %v", seeds, none, err)
	}
	if n := len(logOf(t, e, sid, "a")); n != 0 {
		t.Fatalf("the plan wrote %d rows", n)
	}
	if seeds[0].State != review.Initial(fsrs.Good) {
		t.Errorf("seeded state %+v, want the reference first Good", seeds[0].State)
	}
	if ok, err := e.ApplyFSRSSeed(sid, seeds[0]); !ok || err != nil {
		t.Fatalf("apply: %v %v", ok, err)
	}
	if ok, err := e.ApplyFSRSSeed(sid, seeds[0]); ok || err != nil {
		t.Fatalf("second apply: %v %v, want a no-op", ok, err)
	}
	p, _ := e.repo.GetProgress(sid, "a")
	if p.FSRSSeed != "mastery" || p.FSRSState != "review" {
		t.Errorf("row %+v", p)
	}
	if pb, _ := e.repo.GetProgress(sid, "b"); pb.FSRSState != "none" {
		t.Errorf("credited row seeded: %+v", pb)
	}
	replayAgrees(t, e, sid)
}
