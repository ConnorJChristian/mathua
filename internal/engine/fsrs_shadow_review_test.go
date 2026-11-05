package engine

// FSRS Phase 2 independent reviewer. Adversarial tests of shadow
// scheduling: off writes nothing on any answer path; the plan 3.3 rule
// table with its edge cases; retries; a dropped shadow write; events in
// the same second; the daily cap across the DST change. Values pinned
// from py-fsrs 6.3.2 (tools/fsrs-reference), not from the Go code.

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/clock"
	"github.com/chuma-beep/mathua/internal/diagnostic"
	"github.com/chuma-beep/mathua/internal/fsrs"
	"github.com/chuma-beep/mathua/internal/grader"
	"github.com/chuma-beep/mathua/internal/mastery"
	"github.com/chuma-beep/mathua/internal/review"
	"github.com/chuma-beep/mathua/internal/storage"
)

// py-fsrs 6.3.2, Scheduler(desired_retention=0.9, learning_steps=(),
// relearning_steps=(), maximum_interval=365, enable_fuzzing=False): the
// first review of a new card rated Good.
const (
	refFirstGoodS = 2.3065
	refFirstGoodD = 2.118103970459016
)

func rvEngine(t *testing.T, mode review.Mode, at time.Time) (*Engine, string, *clock.Sim) {
	t.Helper()
	sim := clock.NewSim(at)
	t.Cleanup(sim.Install())
	e := testEngine(t)
	e.SetFSRSMode(mode)
	e.SetFSRSRules(review.DefaultRules())
	return e, creditStudent(t, e), sim
}

var rvMonday = time.Date(2026, 11, 9, 17, 0, 0, 0, time.UTC) // 09:00 PST

// rvReview serves one review-session question on cid with the given
// answer key and submits answer. It returns the session id too.
func rvReview(t *testing.T, e *Engine, sid, cid, expected, answer, attemptID string) (*AnswerResult, string, error) {
	t.Helper()
	sess, err := e.repo.CreateSession(sid)
	if err != nil {
		t.Fatal(err)
	}
	e.mu.Lock()
	e.sessions[sess.ID] = &activeSession{studentID: sid, conceptID: cid, expectedAnswer: expected,
		attemptID: attemptID, isReview: true, requiredStreak: 3, timeThreshold: 60, questionText: "Q?"}
	e.mu.Unlock()
	res, err := e.SubmitAnswer(sess.ID, sid, attemptID, answer, 5)
	return res, sess.ID, err
}

func rvLog(t *testing.T, e *Engine, sid, cid string) []storage.ReviewLogEntry {
	t.Helper()
	l, err := e.repo.ReviewLogFor(sid, cid)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func rvAttempts(t *testing.T, e *Engine, sid string) int {
	t.Helper()
	a, err := e.repo.GetAttemptsForStudent(sid)
	if err != nil {
		t.Fatal(err)
	}
	return len(a)
}

func rvRow(t *testing.T, e *Engine, sid, cid string) *storage.ConceptProgress {
	t.Helper()
	p, err := e.repo.GetProgress(sid, cid)
	if err != nil || p == nil {
		t.Fatalf("progress %s: %v", cid, err)
	}
	return p
}

// rvProof installs a proof grader that returns r for every proof.
func rvProof(t *testing.T, r grader.Result) {
	grader.SetProofGrader(func(expected, answer string) grader.Result { return r })
	t.Cleanup(func() { grader.SetProofGrader(nil) })
}

func rvGraded(full bool) grader.Result {
	score := 2
	if full {
		score = 4
	}
	return grader.Result{Correct: full, Score: float64(score) / 4, Proof: &grader.ProofReport{Score: score, Max: 4, FullyCorrect: full}}
}

// everyPath drives every answer path once: practice to mastery, a practice
// miss, Learn answers and "I don't know", review right / wrong / "I don't
// know" / a self-graded card / a judged proof, the mastery check (answer
// and "I don't know"), a course review test (answer and "I don't know"),
// placement and credit.
func everyPath(t *testing.T, e *Engine, sid string, sim *clock.Sim) {
	t.Helper()
	rvProof(t, rvGraded(true))
	answerCorrectly(t, e, sid, 6) // practice: masters a
	sim.Advance(3 * 24 * time.Hour)
	e.SetStudyAnchor(sid, "a", "42")
	mustOK(t)(e.SubmitStudyAnswer(sid, "a", "7", 5, "")) // practice miss, mastered
	e.SetStudyAnchor(sid, "b", "99")
	mustOK(t)(e.SubmitStudyAnswer(sid, "b", "99", 5, ""))
	e.SetStudyAnchor(sid, "b", "99")
	mustOK(t)(e.SubmitStudyDontKnow(sid, "b", 5, ""))
	sim.Advance(24 * time.Hour)
	for i, x := range []struct{ exp, ans string }{{"42", "42"}, {"42", "7"}, {"42", "I don't know"}, {"recall:card-1", "anything"}, {"proof:p1", "a proof"}} {
		if _, _, err := rvReview(t, e, sid, "a", x.exp, x.ans, "rv"+string(rune('0'+i))); err != nil {
			t.Fatalf("review %d: %v", i, err)
		}
		sim.Advance(time.Hour)
	}
	sim.Advance(24 * time.Hour)
	mustOK(t)(e.SubmitQuizAnswer(sid, "a", "42", "42", 5, "q", nil))
	mustOK(t)(e.SubmitQuizDontKnow(sid, "a", "42", 5, "q", nil))
	sim.Advance(24 * time.Hour)
	mustOK(t)(e.SubmitCourseTestAnswer(sid, "a", "7", "42", 5, "q", nil))
	mustOK(t)(e.SubmitCourseTestDontKnow(sid, "a", "42", 5, "q", nil))
	if err := e.ApplyGoalResults(sid, &diagnostic.Session{StudentID: sid, Attempts: []diagnostic.Attempt{
		{ConceptID: "a", Correct: true, Fast: true}, {ConceptID: "b", Correct: false}}}); err != nil {
		t.Fatal(err)
	}
	e.creditPrerequisites(sid, "b", true)
	e.PropagateWeakness(sid)
}

func mustOK(t *testing.T) func(*AnswerResult, error) {
	return func(_ *AnswerResult, err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
}

// Gate: off mode writes nothing FSRS on every answer path. The same drive in
// shadow mode writes rows, so the check is not vacuous.
func TestReview2_OffWritesNothingOnEveryPath(t *testing.T) {
	e, sid, sim := rvEngine(t, review.ModeOff, rvMonday)
	everyPath(t, e, sid, sim)
	if l := rvLog(t, e, sid, ""); len(l) != 0 {
		t.Fatalf("off wrote %d review_log rows: %+v", len(l), l)
	}
	all, _ := e.repo.GetAllProgress(sid)
	if len(all) == 0 {
		t.Fatal("no progress rows: the drive did nothing")
	}
	for cid, p := range all {
		if p.FSRSStability != nil || p.FSRSDifficulty != nil || p.FSRSLastReview != nil || p.FSRSReps != 0 ||
			p.FSRSLapses != 0 || (p.FSRSState != "none" && p.FSRSState != "") || p.FSRSSeed != "" {
			t.Errorf("off wrote FSRS state on %s: %+v", cid, p)
		}
	}
	offAttempts := rvAttempts(t, e, sid)

	es, ssid, ssim := rvEngine(t, review.ModeShadow, rvMonday)
	everyPath(t, es, ssid, ssim)
	if n := len(rvLog(t, es, ssid, "")); n < 8 {
		t.Errorf("shadow wrote only %d rows on the same drive", n)
	}
	if n := rvAttempts(t, es, ssid); n != offAttempts {
		t.Errorf("shadow recorded %d attempts, off %d", n, offAttempts)
	}
}

// Plan 3.3, row by row, with the edge cases.
func TestReview2_RuleTable(t *testing.T) {
	t.Run("first mastery is the reference first Good", func(t *testing.T) {
		e, sid, _ := rvEngine(t, review.ModeShadow, rvMonday)
		answerCorrectly(t, e, sid, 6)
		l := rvLog(t, e, sid, "a")
		if len(l) != 1 {
			t.Fatalf("log %+v", l)
		}
		x := l[0]
		if x.Source != "first_mastery" || x.Rating != int(fsrs.Good) || x.Held != "" || x.SBefore != 0 || x.ElapsedDays != 0 || x.RAtReview != 0 {
			t.Errorf("row %+v", x)
		}
		if math.Abs(x.SAfter-refFirstGoodS) > 1e-12 || math.Abs(x.DAfter-refFirstGoodD) > 1e-12 {
			t.Errorf("S/D %.15g/%.15g, reference %.15g/%.15g", x.SAfter, x.DAfter, refFirstGoodS, refFirstGoodD)
		}
		p := rvRow(t, e, sid, "a")
		if *p.FSRSStability != x.SAfter || p.FSRSReps != 1 || p.FSRSState != "review" || p.FSRSSeed != "mastery" || !p.FSRSLastReview.Equal(clock.Now()) {
			t.Errorf("row %+v", p)
		}
	})

	t.Run("I don't know in a review is Again", func(t *testing.T) {
		e, sid, sim := rvEngine(t, review.ModeShadow, rvMonday)
		answerCorrectly(t, e, sid, 6)
		sim.Advance(3 * 24 * time.Hour)
		if _, _, err := rvReview(t, e, sid, "a", "42", "I don't know", "r1"); err != nil {
			t.Fatal(err)
		}
		l := rvLog(t, e, sid, "a")
		if len(l) != 2 || l[1].Source != "review" || l[1].Rating != int(fsrs.Again) || l[1].Held != "" || l[1].SAfter >= l[1].SBefore {
			t.Fatalf("log %+v", l)
		}
		if p := rvRow(t, e, sid, "a"); p.FSRSLapses != 1 || p.FSRSState != "relearning" {
			t.Errorf("row %+v", p)
		}
	})

	t.Run("Learn I don't know on a mastered topic is a practice lapse", func(t *testing.T) {
		e, sid, sim := rvEngine(t, review.ModeShadow, rvMonday)
		answerCorrectly(t, e, sid, 6)
		sim.Advance(5 * 24 * time.Hour)
		e.SetStudyAnchor(sid, "a", "42")
		mustOK(t)(e.SubmitStudyDontKnow(sid, "a", 5, ""))
		l := rvLog(t, e, sid, "a")
		if len(l) != 2 || l[1].Source != "practice_lapse" || l[1].Rating != int(fsrs.Again) || l[1].Held != "" || l[1].RAtReview >= 0.95 {
			t.Fatalf("log %+v", l)
		}
	})

	t.Run("a second right review the same local day does not raise S", func(t *testing.T) {
		e, sid, sim := rvEngine(t, review.ModeShadow, rvMonday)
		answerCorrectly(t, e, sid, 6)
		sim.Advance(24 * time.Hour)
		mustOK3(t)(rvReview(t, e, sid, "a", "42", "42", "r1")) // raises 2.31 -> 7.3
		sim.Advance(2 * time.Hour)
		mustOK3(t)(rvReview(t, e, sid, "a", "42", "42", "r2"))
		l := rvLog(t, e, sid, "a")
		if len(l) != 3 || l[1].SAfter <= l[1].SBefore {
			t.Fatalf("log %+v", l)
		}
		if second := l[2]; second.SAfter > second.SBefore {
			t.Errorf("second same-day review raised S %.4f -> %.4f (held %q)", second.SBefore, second.SAfter, second.Held)
		}
		t.Logf("large S: second same-day Good held=%q, S %.4f -> %.4f, D %.4f -> %.4f",
			l[2].Held, l[2].SBefore, l[2].SAfter, l[2].DBefore, l[2].DAfter)
	})

	t.Run("a second right answer the same day after a relearn is held", func(t *testing.T) {
		e, sid, sim := rvEngine(t, review.ModeShadow, rvMonday)
		answerCorrectly(t, e, sid, 6)
		sim.Advance(3 * 24 * time.Hour)
		mustOK3(t)(rvReview(t, e, sid, "a", "42", "7", "r1")) // Again: S 0.64, never held
		sim.Advance(time.Hour)
		mustOK3(t)(rvReview(t, e, sid, "a", "42", "42", "r2")) // the day's first raise
		sim.Advance(time.Hour)
		mustOK3(t)(rvReview(t, e, sid, "a", "42", "42", "r3")) // would raise again: held
		l := rvLog(t, e, sid, "a")
		if len(l) != 4 || l[1].Held != "" || l[2].Held != "" || l[2].SAfter <= l[2].SBefore || l[3].Held != review.HeldDailyCap || l[3].SAfter != l[3].SBefore {
			t.Fatalf("log %+v", l)
		}
		p := rvRow(t, e, sid, "a")
		if *p.FSRSStability != l[2].SAfter || !p.FSRSLastReview.Equal(l[2].At) || p.FSRSReps != 3 {
			t.Errorf("held event moved the state: %+v", p)
		}
	})

	t.Run("practice miss on a mastered topic: held while R >= 0.95 (under a day), Again after", func(t *testing.T) {
		e, sid, sim := rvEngine(t, review.ModeShadow, rvMonday)
		answerCorrectly(t, e, sid, 6)
		sim.Advance(23*time.Hour + 59*time.Minute)
		e.SetStudyAnchor(sid, "a", "42")
		mustOK(t)(e.SubmitStudyAnswer(sid, "a", "7", 5, ""))
		l := rvLog(t, e, sid, "a")
		if len(l) != 2 || l[1].Source != "practice_lapse" || l[1].Held != review.HeldLapseRHigh || l[1].RAtReview != 1 || l[1].SAfter != l[1].SBefore {
			t.Fatalf("23h59m: %+v", l)
		}

		e2, sid2, sim2 := rvEngine(t, review.ModeShadow, rvMonday)
		answerCorrectly(t, e2, sid2, 6)
		sim2.Advance(24 * time.Hour)
		e2.SetStudyAnchor(sid2, "a", "42")
		mustOK(t)(e2.SubmitStudyAnswer(sid2, "a", "7", 5, ""))
		l2 := rvLog(t, e2, sid2, "a")
		// py-fsrs: R after 1 whole day at S = w[2] is 0.946847 < 0.95.
		if len(l2) != 2 || l2[1].Held != "" || l2[1].Rating != int(fsrs.Again) || math.Abs(l2[1].RAtReview-0.946847) > 1e-6 {
			t.Fatalf("24h: %+v", l2)
		}
	})

	t.Run("practice right on a mastered topic is no event", func(t *testing.T) {
		e, sid, sim := rvEngine(t, review.ModeShadow, rvMonday)
		answerCorrectly(t, e, sid, 6)
		sim.Advance(9 * 24 * time.Hour)
		e.SetStudyAnchor(sid, "a", "42")
		mustOK(t)(e.SubmitStudyAnswer(sid, "a", "42", 5, ""))
		if n := len(rvLog(t, e, sid, "a")); n != 1 {
			t.Errorf("%d rows", n)
		}
	})

	t.Run("judge API failure is no event and no attempt", func(t *testing.T) {
		e, sid, sim := rvEngine(t, review.ModeShadow, rvMonday)
		answerCorrectly(t, e, sid, 6)
		n0 := rvAttempts(t, e, sid)
		rvProof(t, grader.Result{Unavailable: true, Feedback: "judge down"})
		sim.Advance(3 * 24 * time.Hour)
		res, _, err := rvReview(t, e, sid, "a", "proof:p1", "my proof", "r1")
		if err != nil || res == nil || !res.Ungraded {
			t.Fatalf("res %+v err %v", res, err)
		}
		if n := len(rvLog(t, e, sid, "a")); n != 1 {
			t.Errorf("judge failure wrote %d rows", n-1)
		}
		if n := rvAttempts(t, e, sid); n != n0 {
			t.Errorf("judge failure recorded %d attempts", n-n0)
		}
	})

	t.Run("judged proof: graded Good / Again; judged=off nothing", func(t *testing.T) {
		e, sid, sim := rvEngine(t, review.ModeShadow, rvMonday)
		answerCorrectly(t, e, sid, 6)
		sim.Advance(3 * 24 * time.Hour)
		rvProof(t, rvGraded(true))
		mustOK3(t)(rvReview(t, e, sid, "a", "proof:p1", "good proof", "r1"))
		sim.Advance(3 * 24 * time.Hour)
		rvProof(t, rvGraded(false))
		mustOK3(t)(rvReview(t, e, sid, "a", "proof:p1", "weak proof", "r2"))
		l := rvLog(t, e, sid, "a")
		if len(l) != 3 || l[1].Rating != int(fsrs.Good) || l[2].Rating != int(fsrs.Again) || l[1].ItemID != "proof:p1" {
			t.Fatalf("log %+v", l)
		}
		r := review.DefaultRules()
		r.Judged = false
		e.SetFSRSRules(r)
		sim.Advance(3 * 24 * time.Hour)
		mustOK3(t)(rvReview(t, e, sid, "a", "proof:p1", "weak proof", "r3"))
		if n := len(rvLog(t, e, sid, "a")); n != 3 {
			t.Errorf("judged=off wrote %d rows", n-3)
		}
	})

	t.Run("self-graded recall in a review is no event", func(t *testing.T) {
		e, sid, sim := rvEngine(t, review.ModeShadow, rvMonday)
		answerCorrectly(t, e, sid, 6)
		sim.Advance(3 * 24 * time.Hour)
		mustOK3(t)(rvReview(t, e, sid, "a", "recall:card-9", "whatever", "r1"))
		mustOK(t)(e.SubmitQuizAnswer(sid, "a", "x", "recall:card-9", 5, "q", nil))
		if n := len(rvLog(t, e, sid, "a")); n != 1 {
			t.Errorf("recall wrote %d rows", n-1)
		}
	})

	t.Run("placement is no event and leaves state alone", func(t *testing.T) {
		e, sid, sim := rvEngine(t, review.ModeShadow, rvMonday)
		answerCorrectly(t, e, sid, 6)
		before := *rvRow(t, e, sid, "a")
		sim.Advance(2 * 24 * time.Hour)
		if err := e.ApplyGoalResults(sid, &diagnostic.Session{StudentID: sid, Attempts: []diagnostic.Attempt{
			{ConceptID: "a", Correct: false}, {ConceptID: "b", Correct: true, Fast: true}}}); err != nil {
			t.Fatal(err)
		}
		if n := len(rvLog(t, e, sid, "")); n != 1 {
			t.Errorf("placement wrote %d rows", n-1)
		}
		after := rvRow(t, e, sid, "a")
		if *after.FSRSStability != *before.FSRSStability || after.FSRSReps != before.FSRSReps || !after.FSRSLastReview.Equal(*before.FSRSLastReview) {
			t.Errorf("placement moved a's state: %+v -> %+v", before, after)
		}
		if pb, _ := e.repo.GetProgress(sid, "b"); pb != nil && pb.FSRSStability != nil {
			t.Errorf("placement created state on b: %+v", pb)
		}
	})

	// A credited row (mastered, no attempts of its own) has no state. Two
	// histories that differ by one right practice answer end far apart
	// (reviewer finding L-credit): a miss with no own answer before it
	// creates an Again state (seed "credit", no R threshold); a right
	// answer and then a miss seeds a first-mastery Good dated at the
	// credit (LastReviewed), labelled "mastery", and holds the miss.
	t.Run("credited row: first practice miss creates an Again state", func(t *testing.T) {
		e, sid, sim := rvEngine(t, review.ModeShadow, rvMonday)
		now := clock.Now()
		if err := e.repo.UpsertProgress(&storage.ConceptProgress{StudentID: sid, ConceptID: "a", Status: string(mastery.StatusMastered),
			ImplicitCredit: 1, LastReviewed: &now, SM2EFactor: 2.5}); err != nil {
			t.Fatal(err)
		}
		sim.Advance(2 * time.Minute)
		e.SetStudyAnchor(sid, "a", "42")
		mustOK(t)(e.SubmitStudyAnswer(sid, "a", "7", 5, ""))
		l := rvLog(t, e, sid, "a")
		p := rvRow(t, e, sid, "a")
		if len(l) != 1 || l[0].Source != "practice_lapse" || l[0].Rating != int(fsrs.Again) || l[0].Held != "" || p.FSRSSeed != "credit" || p.FSRSStability == nil {
			t.Fatalf("credited miss: log %+v row %+v", l, p)
		}
		t.Logf("credited, miss 2 min after credit: S %.4f state %s seed %s", *p.FSRSStability, p.FSRSState, p.FSRSSeed)
	})
	t.Run("credited row: one right answer then a miss seeds a mastery at the credit time", func(t *testing.T) {
		e, sid, sim := rvEngine(t, review.ModeShadow, rvMonday)
		credit := clock.Now()
		if err := e.repo.UpsertProgress(&storage.ConceptProgress{StudentID: sid, ConceptID: "a", Status: string(mastery.StatusMastered),
			ImplicitCredit: 1, LastReviewed: &credit, SM2EFactor: 2.5}); err != nil {
			t.Fatal(err)
		}
		sim.Advance(time.Minute)
		e.SetStudyAnchor(sid, "a", "42")
		mustOK(t)(e.SubmitStudyAnswer(sid, "a", "42", 5, ""))
		if n := len(rvLog(t, e, sid, "a")); n != 0 {
			t.Fatalf("a right answer on a credited row wrote %d rows", n)
		}
		sim.Advance(time.Minute)
		e.SetStudyAnchor(sid, "a", "42")
		mustOK(t)(e.SubmitStudyAnswer(sid, "a", "7", 5, ""))
		l := rvLog(t, e, sid, "a")
		p := rvRow(t, e, sid, "a")
		if len(l) != 2 || l[0].Source != "first_mastery" || !l[0].At.Equal(credit) || l[1].Held != review.HeldLapseRHigh || p.FSRSSeed != "mastery" || *p.FSRSStability != refFirstGoodS {
			t.Fatalf("log %+v row %+v", l, p)
		}
	})

	// The cap's day is the learner-local day. Nov 1 2026 (America/Los_Angeles)
	// is 25 hours long: a review a full 24 h after the first mastery is
	// still the same local day, so its raise is held.
	t.Run("daily cap on the DST day holds a review 24 h later", func(t *testing.T) {
		if loc, _ := review.LearnerZone(); loc.String() != "America/Los_Angeles" {
			t.Skip("learner zone is not America/Los_Angeles")
		}
		e, sid, sim := rvEngine(t, review.ModeShadow, time.Date(2026, 11, 1, 7, 30, 0, 0, time.UTC)) // 00:30 PDT
		answerCorrectly(t, e, sid, 6)
		sim.Advance(24 * time.Hour) // 23:30 PST, Nov 1
		mustOK3(t)(rvReview(t, e, sid, "a", "42", "42", "r1"))
		l := rvLog(t, e, sid, "a")
		if len(l) != 2 || l[1].ElapsedDays != 1 || l[1].Held != review.HeldDailyCap {
			t.Fatalf("log %+v", l)
		}
	})
}

func mustOK3(t *testing.T) func(*AnswerResult, string, error) {
	return func(_ *AnswerResult, _ string, err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
}

// A retried review answer (same session and attempt id) returns the first
// result and writes nothing.
func TestReview2_RetryReturnsSameResultWritesNothing(t *testing.T) {
	e, sid, sim := rvEngine(t, review.ModeShadow, rvMonday)
	answerCorrectly(t, e, sid, 6)
	sim.Advance(3 * 24 * time.Hour)
	res1, sess, err := rvReview(t, e, sid, "a", "42", "42", "r1")
	if err != nil {
		t.Fatal(err)
	}
	n0, l0 := rvAttempts(t, e, sid), len(rvLog(t, e, sid, ""))
	xp0, _, _ := e.repo.GetXP(sid)
	p0 := *rvRow(t, e, sid, "a")
	sim.Advance(10 * time.Second)
	res2, err := e.SubmitAnswer(sess, sid, "r1", "42", 5)
	if err != nil {
		t.Fatal(err)
	}
	if res2.Correct != res1.Correct || res2.XP != res1.XP || res2.NewStatus != res1.NewStatus || res2.Feedback != res1.Feedback {
		t.Errorf("retry result %+v, first %+v", res2, res1)
	}
	xp1, _, _ := e.repo.GetXP(sid)
	p1 := rvRow(t, e, sid, "a")
	if rvAttempts(t, e, sid) != n0 || len(rvLog(t, e, sid, "")) != l0 || xp1 != xp0 || p1.Attempts != p0.Attempts || *p1.FSRSStability != *p0.FSRSStability {
		t.Errorf("the retry wrote something: attempts %d->%d log %d->%d xp %d->%d", n0, rvAttempts(t, e, sid), l0, len(rvLog(t, e, sid, "")), xp0, xp1)
	}
}

// After a crash between the commit and the session tombstone, the client's
// retry reaches the grader again. Shadow refuses it (the review_log key is
// stored: ErrDuplicateAnswer); off records it a second time. Both are
// pinned here: it is the one place shadow decides something off does not
// (reviewer finding I-dup).
func TestReview2_CrashWindowRetry_ShadowRefusesOffRecords(t *testing.T) {
	for _, mode := range []review.Mode{review.ModeShadow, review.ModeOff} {
		e, sid, sim := rvEngine(t, mode, rvMonday)
		answerCorrectly(t, e, sid, 6)
		sim.Advance(3 * 24 * time.Hour)
		_, sess, err := rvReview(t, e, sid, "a", "42", "42", "r1")
		if err != nil {
			t.Fatal(err)
		}
		n0 := rvAttempts(t, e, sid)
		// The process died before the tombstone: the persisted session
		// still holds the unanswered question.
		if err := e.repo.UpsertActiveSession(&storage.ActiveSession{SessionID: sess, StudentID: sid, ConceptID: "a", ExpectedAnswer: "42",
			AttemptID: "r1", Question: "Q?", IsReview: true}); err != nil {
			t.Fatal(err)
		}
		e2 := New(e.repo, e.dag, e.registry, nil, nil)
		e2.SetFSRSMode(mode)
		e2.SetFSRSRules(review.DefaultRules())
		_, err = e2.SubmitAnswer(sess, sid, "r1", "42", 5)
		n1 := rvAttempts(t, e2, sid)
		switch mode {
		case review.ModeShadow:
			if !errors.Is(err, ErrDuplicateAnswer) || n1 != n0 {
				t.Errorf("shadow: err %v, attempts %d -> %d; want ErrDuplicateAnswer and nothing written", err, n0, n1)
			}
		case review.ModeOff:
			if err != nil || n1 != n0+1 {
				t.Errorf("off: err %v, attempts %d -> %d; want the retry recorded again", err, n0, n1)
			}
		}
	}
}

// commitFailRepo fails every commit that carries an FSRS half.
type commitFailRepo struct {
	storage.Repository
	failed int
}

func (r *commitFailRepo) CommitAnswer(c storage.AnswerCommit) (storage.CommitResult, error) {
	if c.FSRS != nil {
		r.failed++
		return storage.CommitResult{}, errors.New("injected: review_log write failed")
	}
	return r.Repository.CommitAnswer(c)
}

// A failed shadow write never costs the answer, and leaves no trace the
// doctor's checks can see (the stored state still replays from its log). A
// cross-check of attempts against the log does see it (the reviewer's
// proposed doctor check).
func TestReview2_DroppedShadowWriteIsInvisibleToTheReplayCheck(t *testing.T) {
	e, sid, sim := rvEngine(t, review.ModeShadow, rvMonday)
	answerCorrectly(t, e, sid, 6)
	sim.Advance(3 * 24 * time.Hour)
	fr := &commitFailRepo{Repository: e.repo}
	e.repo = fr
	res, _, err := rvReview(t, e, sid, "a", "42", "7", "r1")
	e.repo = fr.Repository
	if err != nil || res == nil || res.Correct {
		t.Fatalf("answer lost: %+v %v", res, err)
	}
	if fr.failed != 1 {
		t.Fatalf("injected %d failures", fr.failed)
	}
	l := rvLog(t, e, sid, "a")
	if len(l) != 1 {
		t.Fatalf("log %+v", l)
	}
	replayAgrees(t, e, sid) // the doctor's replay: consistent, so silent
	// The proposed check: a review answer after the topic's first event
	// with no log row at its second.
	atts, _ := e.repo.GetAttemptsForStudent(sid)
	missing := 0
	for _, a := range atts {
		if a.ConceptID != "a" || a.Source != "review" || a.Timestamp.Before(l[0].At.Truncate(time.Second)) {
			continue
		}
		found := false
		for _, x := range l {
			if x.At.Truncate(time.Second).Equal(a.Timestamp.Truncate(time.Second)) {
				found = true
			}
		}
		if !found {
			missing++
		}
	}
	if missing != 1 {
		t.Errorf("attempts-vs-log check found %d missing events, want 1", missing)
	}
}

// Events in the same second keep their order: the log reads back in answer
// order and replays to the stored state.
func TestReview2_SameSecondEventsKeepOrder(t *testing.T) {
	e, sid, sim := rvEngine(t, review.ModeShadow, rvMonday)
	answerCorrectly(t, e, sid, 6)
	sim.Advance(3 * 24 * time.Hour)
	mustOK(t)(e.SubmitQuizAnswer(sid, "a", "7", "42", 5, "q", nil)) // Again
	sim.Advance(300 * time.Millisecond)
	mustOK(t)(e.SubmitQuizAnswer(sid, "a", "42", "42", 5, "q", nil)) // Good (raise: held)
	sim.Advance(300 * time.Millisecond)
	mustOK3(t)(rvReview(t, e, sid, "a", "42", "7", "r1")) // Again
	// And two at the identical instant.
	mustOK3(t)(rvReview(t, e, sid, "a", "42", "42", "r2"))
	l := rvLog(t, e, sid, "a")
	want := []int{int(fsrs.Good), int(fsrs.Again), int(fsrs.Good), int(fsrs.Again), int(fsrs.Good)}
	if len(l) != len(want) {
		t.Fatalf("log %+v", l)
	}
	for i, x := range l {
		if x.Rating != want[i] {
			t.Errorf("row %d rating %d, want %d (%+v)", i, x.Rating, want[i], x)
		}
		if i > 0 && x.At.Before(l[i-1].At) {
			t.Errorf("row %d out of time order", i)
		}
	}
	replayAgrees(t, e, sid)
}
