package engine

// FSRS Phase 2: fixes for the independent review's findings (M2, L1, L5,
// I2, I3, L2). Each test failed before its fix.

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
