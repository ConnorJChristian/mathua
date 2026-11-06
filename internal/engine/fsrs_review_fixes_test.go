package engine

// FSRS Phase 2: fixes for the independent review's findings (M2, L1, L5,
// I2, I3, L2). Each test failed before its fix.

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/clock"
	"github.com/chuma-beep/mathua/internal/fsrs"
	"github.com/chuma-beep/mathua/internal/mastery"
	"github.com/chuma-beep/mathua/internal/review"
	"github.com/chuma-beep/mathua/internal/storage"
)

// logFailRepo fails ReviewLogFor, so the shadow event cannot be computed.
type logFailRepo struct{ storage.Repository }

func (r *logFailRepo) ReviewLogFor(string, string) ([]storage.ReviewLogEntry, error) {
	return nil, errors.New("injected: review_log read failed")
}

// M2: a dropped shadow event leaves an fsrs_errors row, on both failure
// paths (not computed, not written), naming the learner, the topic and the
// committed attempt; the answer itself is committed.
func TestFix2_DroppedEventIsRecorded(t *testing.T) {
	for _, tc := range []struct {
		stage string
		wrap  func(storage.Repository) storage.Repository
	}{
		{"written", func(r storage.Repository) storage.Repository { return &commitFailRepo{Repository: r} }},
		{"computed", func(r storage.Repository) storage.Repository { return &logFailRepo{Repository: r} }},
	} {
		e, sid, sim := rvEngine(t, review.ModeShadow, rvMonday)
		answerCorrectly(t, e, sid, 6)
		sim.Advance(3 * 24 * time.Hour)
		n0 := rvAttempts(t, e, sid)
		inner := e.repo
		e.repo = tc.wrap(inner)
		res, _, err := rvReview(t, e, sid, "a", "42", "7", "r1")
		e.repo = inner
		if err != nil || res == nil {
			t.Fatalf("%s: the answer failed: %v", tc.stage, err)
		}
		atts, _ := e.repo.GetAttemptsForStudent(sid)
		if len(atts) != n0+1 {
			t.Fatalf("%s: attempts %d -> %d", tc.stage, n0, len(atts))
		}
		rows, err := e.repo.FSRSErrors(sid)
		if err != nil {
			t.Fatal(err)
		}
		last := atts[len(atts)-1]
		for _, a := range atts {
			if a.ID > last.ID {
				last = a
			}
		}
		if len(rows) != 1 || rows[0].Stage != tc.stage || rows[0].ConceptID != "a" || rows[0].StudentID != sid || rows[0].AttemptID != last.ID || rows[0].Error == "" {
			t.Errorf("%s: fsrs_errors %+v (attempt %d)", tc.stage, rows, last.ID)
		}
		if n := len(rvLog(t, e, sid, "a")); n != 1 {
			t.Errorf("%s: %d log rows, want the first mastery only", tc.stage, n)
		}
	}
	// A clean answer records nothing.
	e, sid, sim := rvEngine(t, review.ModeShadow, rvMonday)
	answerCorrectly(t, e, sid, 6)
	sim.Advance(3 * 24 * time.Hour)
	mustOK3(t)(rvReview(t, e, sid, "a", "42", "7", "r1"))
	if rows, _ := e.repo.FSRSErrors(""); len(rows) != 0 {
		t.Errorf("a clean answer wrote fsrs_errors %+v", rows)
	}
}

// L1: one rule for credited rows. The first FSRS event on a credited row
// with no state initialises it from the reference first review for that
// event's rating (no lapse threshold, no seed from attempts), labelled
// "credit", whatever right answers came after the credit; a real credit
// sets PlacementSeeded too and is still "credit". fsrs_seed leaves such a
// row alone. A row the learner mastered (MasteredAt) is seeded as before.
func TestFix_L1_CreditedRowsOneRule(t *testing.T) {
	p := review.DefaultParams()
	for _, tc := range []struct {
		name      string
		rightsAft int // right practice answers after the credit
		review    bool
		correct   bool
	}{
		{"miss straight after the credit", 0, false, false},
		{"one right answer, then a miss", 1, false, false},
		{"three right answers, then a miss", 3, false, false},
		{"a right review", 1, true, true},
		{"a wrong review", 2, true, false},
	} {
		e, sid, sim := rvEngine(t, review.ModeShadow, rvMonday)
		credit := clock.Now()
		// What creditPrerequisites writes: both flags, no MasteredAt.
		if err := e.repo.UpsertProgress(&storage.ConceptProgress{StudentID: sid, ConceptID: "a", Status: string(mastery.StatusMastered),
			ImplicitCredit: 1, PlacementSeeded: true, LastReviewed: &credit, SM2EFactor: 2.5}); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < tc.rightsAft; i++ {
			sim.Advance(time.Minute)
			e.SetStudyAnchor(sid, "a", "42")
			mustOK(t)(e.SubmitStudyAnswer(sid, "a", "42", 5, ""))
		}
		if seeds, none, _ := e.FSRSSeedPlan(sid, clock.Now()); len(seeds) != 0 || len(none) != 1 {
			t.Errorf("%s: fsrs_seed would seed %d, leave %v; want the credited row left", tc.name, len(seeds), none)
		}
		sim.Advance(2 * 24 * time.Hour)
		rating := fsrs.Again
		if tc.review {
			ans := "7"
			if tc.correct {
				ans, rating = "42", fsrs.Good
			}
			mustOK3(t)(rvReview(t, e, sid, "a", "42", ans, "r1"))
		} else {
			e.SetStudyAnchor(sid, "a", "42")
			mustOK(t)(e.SubmitStudyAnswer(sid, "a", "7", 5, ""))
		}
		l := rvLog(t, e, sid, "a")
		row := rvRow(t, e, sid, "a")
		want := p.Initial(rating)
		if len(l) != 1 || l[0].Rating != int(rating) || l[0].Held != "" || l[0].SBefore != 0 || l[0].ElapsedDays != 0 {
			t.Errorf("%s: log %+v", tc.name, l)
			continue
		}
		if row.FSRSSeed != "credit" || *row.FSRSStability != want.Stability || *row.FSRSDifficulty != want.Difficulty || !row.FSRSLastReview.Equal(l[0].At) {
			t.Errorf("%s: row seed %q S %v D %v, want credit %v %v", tc.name, row.FSRSSeed, *row.FSRSStability, *row.FSRSDifficulty, want.Stability, want.Difficulty)
		}
		replayAgrees(t, e, sid)
	}
}

// L2: FSRSDueIDs (Phase 3's due rule) and the report count elapsed time as
// the events and py-fsrs do: whole 24-hour periods. Reference values from
// py-fsrs 6.3.2 Scheduler.get_card_retrievability (desired_retention 0.9,
// no learning steps, maximum interval 365, no fuzz). Cases 1 and 3 are due
// by fractional days and not by the reference: the old rule called them
// due.
func TestFix_L2_DueUsesWholeDaysLikeTheReference(t *testing.T) {
	t0 := time.Date(2026, 11, 9, 17, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name    string
		s, d    float64
		last    time.Time
		at      time.Duration
		refR    float64
		wantDue bool
	}{
		{"first Good, 2 d 12 h", 2.3065, 2.118103970459016, t0, 60 * time.Hour, 0.9094932559773545, false},
		{"first Good, 3 d 1 h", 2.3065, 2.118103970459016, t0, 73 * time.Hour, 0.8809479557659419, true},
		{"Good, Good after 3 d; 13 d 20 h", 13.826903694354568, 2.111214235785395, t0.Add(72 * time.Hour), 13*24*time.Hour + 20*time.Hour, 0.9041803379249108, false},
	} {
		e, sid, _ := rvEngine(t, review.ModeShadow, rvMonday)
		s, d, last := tc.s, tc.d, tc.last
		row := &storage.ConceptProgress{StudentID: sid, ConceptID: "a", Status: string(mastery.StatusMastered), MasteredAt: &last, LastReviewed: &last, SM2EFactor: 2.5,
			FSRSStability: &s, FSRSDifficulty: &d, FSRSLastReview: &last, FSRSReps: 2, FSRSState: "review", FSRSSeed: "mastery"}
		now := tc.last.Add(tc.at)
		r, ok := FSRSRetrievabilityAt(row, now)
		if !ok || math.Abs(r-tc.refR) > 1e-12 {
			t.Errorf("%s: R %.16g, py-fsrs %.16g", tc.name, r, tc.refR)
		}
		due := e.FSRSDueIDs(sid, map[string]*storage.ConceptProgress{"a": row}, now, 0.9)
		if got := len(due) == 1; got != tc.wantDue {
			t.Errorf("%s: due %v, want %v (R %.4f)", tc.name, got, tc.wantDue, r)
		}
	}
}

// L5: the answer that first masters a topic creates its state wherever it
// is given: a mastery check or a course review test gives the same
// first_mastery row (the reference first Good, dated at that answer) as
// practice. Reaching mastery is the event, so the quiz kinds' own rules do
// not hold it; first_mastery=off does.
func TestFix_L5_FirstMasteryThroughAQuiz(t *testing.T) {
	for _, tc := range []struct {
		name   string
		course bool
		rules  func(*review.Rules)
		want   bool
	}{
		{"mastery check", false, func(*review.Rules) {}, true},
		{"course review test", true, func(*review.Rules) {}, true},
		{"mastery check, mastery_check=off", false, func(r *review.Rules) { r.MasteryCheck = false }, true},
		{"course test, course_tests=off", true, func(r *review.Rules) { r.CourseTests = false }, true},
		{"mastery check, first_mastery=off", false, func(r *review.Rules) { r.FirstMastery = false }, false},
	} {
		e, sid, sim := rvEngine(t, review.ModeShadow, rvMonday)
		r := review.DefaultRules()
		tc.rules(&r)
		e.SetFSRSRules(r)
		var masteredAt time.Time
		for i := 0; i < 12 && masteredAt.IsZero(); i++ {
			sim.Advance(30 * time.Second)
			var err error
			if tc.course {
				_, err = e.SubmitCourseTestAnswer(sid, "a", "42", "42", 5, "q", nil)
			} else {
				_, err = e.SubmitQuizAnswer(sid, "a", "42", "42", 5, "q", nil)
			}
			if err != nil {
				t.Fatal(err)
			}
			if p := rvRow(t, e, sid, "a"); p.Status == string(mastery.StatusMastered) {
				masteredAt = clock.Now()
			}
		}
		if masteredAt.IsZero() {
			t.Fatalf("%s: never mastered", tc.name)
		}
		l := rvLog(t, e, sid, "a")
		p := rvRow(t, e, sid, "a")
		if !tc.want {
			if len(l) != 0 || p.FSRSStability != nil {
				t.Errorf("%s: log %+v", tc.name, l)
			}
			continue
		}
		if len(l) != 1 || l[0].Source != "first_mastery" || l[0].Rating != int(fsrs.Good) || !l[0].At.Equal(masteredAt) || l[0].SAfter != refFirstGoodS {
			t.Errorf("%s: log %+v", tc.name, l)
		}
		if p.FSRSStability == nil || *p.FSRSStability != refFirstGoodS || p.FSRSSeed != "mastery" || p.FSRSReps != 1 {
			t.Errorf("%s: row %+v", tc.name, p)
		}
		replayAgrees(t, e, sid)
	}
}

// I2: a second right review the same learner-local day is held whole, even
// with S large enough that the short-term formula raises nothing: no D,
// reps or last-review movement.
func TestFix_I2_SecondSameDayCorrectHeldWhole(t *testing.T) {
	e, sid, sim := rvEngine(t, review.ModeShadow, rvMonday)
	answerCorrectly(t, e, sid, 6)
	sim.Advance(24 * time.Hour)
	mustOK3(t)(rvReview(t, e, sid, "a", "42", "42", "r1")) // the day's raise: 2.31 -> 7.3
	before := *rvRow(t, e, sid, "a")
	sim.Advance(2 * time.Hour)
	mustOK3(t)(rvReview(t, e, sid, "a", "42", "42", "r2"))
	l := rvLog(t, e, sid, "a")
	after := rvRow(t, e, sid, "a")
	if len(l) != 3 || l[2].Held != review.HeldDailyCap || l[2].SAfter != l[2].SBefore || l[2].DAfter != l[2].DBefore {
		t.Fatalf("log %+v", l)
	}
	if *after.FSRSStability != *before.FSRSStability || *after.FSRSDifficulty != *before.FSRSDifficulty ||
		after.FSRSReps != before.FSRSReps || !after.FSRSLastReview.Equal(*before.FSRSLastReview) {
		t.Errorf("the held event moved the state: %+v -> %+v", before, after)
	}
	// The next local day it applies.
	sim.Advance(22 * time.Hour)
	mustOK3(t)(rvReview(t, e, sid, "a", "42", "42", "r3"))
	if l := rvLog(t, e, sid, "a"); l[3].Held != "" || rvRow(t, e, sid, "a").FSRSReps != before.FSRSReps+1 {
		t.Errorf("next day: %+v", l[3])
	}
	replayAgrees(t, e, sid)
}

// I3: the daily cap counts the learner-local calendar day. Nov 1 2026 in
// America/Los_Angeles is 25 hours long. A review 24 h after a first
// mastery at 01:30 PDT (before the fall-back) lands at 00:30 PST on Nov 2,
// the next local day: it is applied, not held. One at 00:30 PDT lands at
// 23:30 PST the same Nov 1: held.
func TestFix_I3_DailyCapByLearnerLocalDayAcrossDST(t *testing.T) {
	if loc, _ := review.LearnerZone(); loc.String() != "America/Los_Angeles" {
		t.Skip("learner zone is not America/Los_Angeles")
	}
	for _, tc := range []struct {
		name    string
		mastery time.Time
		held    string
	}{
		{"01:30 PDT + 24 h = Nov 2 00:30 PST", time.Date(2026, 11, 1, 8, 30, 0, 0, time.UTC), ""},
		{"00:30 PDT + 24 h = Nov 1 23:30 PST", time.Date(2026, 11, 1, 7, 30, 0, 0, time.UTC), review.HeldDailyCap},
		{"Oct 31 23:30 PDT + 24 h = Nov 1 22:30 PST", time.Date(2026, 11, 1, 6, 30, 0, 0, time.UTC), ""},
	} {
		e, sid, sim := rvEngine(t, review.ModeShadow, tc.mastery)
		answerCorrectly(t, e, sid, 6)
		sim.Advance(24 * time.Hour)
		mustOK3(t)(rvReview(t, e, sid, "a", "42", "42", "r1"))
		l := rvLog(t, e, sid, "a")
		if len(l) != 2 || l[1].ElapsedDays != 1 || l[1].Held != tc.held {
			t.Errorf("%s: log %+v, want held %q", tc.name, l, tc.held)
		}
		replayAgrees(t, e, sid)
	}
}
