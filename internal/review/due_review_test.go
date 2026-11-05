package review

import (
	"math/rand"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/mastery"
)

// Independent review of FSRS Phase 0 (plan 6.1): adversarial cases for the
// one due rule that the author's table (testdata/due_cases.json) may not
// cover, plus a monotonicity property.

func ptr(t time.Time) *time.Time { return &t }

// A mastered row whose SM-2 date is far in the future is NOT due, however
// long ago it was last reviewed: the date decides, decay is pacing only.
func TestReview_FarFutureDateNotDueDespiteOldReview(t *testing.T) {
	now := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	last := now.AddDate(0, 0, -20)
	r := Row{Status: mastery.StatusMastered, LastReviewed: last, LastAttempted: last, NextReviewDue: ptr(now.AddDate(0, 0, 37))}
	if IsDue(r, now) {
		t.Fatalf("mastered row reviewed 20 days ago with a 37-day date is due; the decay rule leaked into IsDue")
	}
	if Overdue(r, now) != 0 {
		t.Fatalf("overdue = %v, want 0", Overdue(r, now))
	}
	if at, ok := DueAt(r); !ok || !at.Equal(*r.NextReviewDue) {
		t.Fatalf("DueAt = %v %v, want the date", at, ok)
	}
}

// A LEARNING or PRACTICING row with a past next_review_due (a practice
// date) is not due, not even if it has a long history.
func TestReview_NonMasteredWithPastDateNeverDue(t *testing.T) {
	now := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	for _, st := range []mastery.Status{mastery.StatusLearning, mastery.StatusPracticing, mastery.StatusUnseen, mastery.Status("DECAYING")} {
		r := Row{Status: st, LastReviewed: now.AddDate(0, 0, -40), LastAttempted: now.AddDate(0, 0, -40), NextReviewDue: ptr(now.AddDate(0, 0, -30))}
		if IsDue(r, now) {
			t.Errorf("%s row with a past date is due", st)
		}
		if _, ok := DueAt(r); ok {
			t.Errorf("%s row has a DueAt", st)
		}
		if Overdue(r, now) != 0 {
			t.Errorf("%s row has overdue %v", st, Overdue(r, now))
		}
	}
}

// Exactly at the date: due (not After). One nanosecond before: not.
func TestReview_BoundaryAtTheDate(t *testing.T) {
	due := time.Date(2026, 11, 6, 4, 49, 0, 0, time.UTC)
	r := Row{Status: mastery.StatusMastered, LastReviewed: due.AddDate(0, 0, -1), NextReviewDue: ptr(due)}
	if !IsDue(r, due) {
		t.Error("not due at its own date")
	}
	if IsDue(r, due.Add(-time.Nanosecond)) {
		t.Error("due a nanosecond before its date")
	}
	// Undated: the horizon is exactly 14*24h after the anchor.
	u := Row{Status: mastery.StatusMastered, LastReviewed: due.Add(-14 * 24 * time.Hour)}
	if !IsDue(u, due) {
		t.Error("undated row not due at its 14-day horizon")
	}
	if IsDue(u, due.Add(-time.Nanosecond)) {
		t.Error("undated row due before its 14-day horizon")
	}
}

// Day boundaries across the 2026-11-01 fall-back in America/Los_Angeles.
// The rule is in absolute time: a topic due at local midnight on Nov 1
// (PDT) is due at every later instant, including after the clocks go
// back, and the day count never goes backwards.
func TestReview_DSTFallBackLosAngeles(t *testing.T) {
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Skip("no tz database: " + err.Error())
	}
	due := time.Date(2026, 11, 1, 0, 0, 0, 0, la) // PDT, before the 02:00 fall-back
	if _, off := due.Zone(); off != -7*3600 {
		t.Fatalf("expected PDT at local midnight Nov 1, got offset %d", off)
	}
	r := Row{Status: mastery.StatusMastered, LastReviewed: due.AddDate(0, 0, -6), NextReviewDue: ptr(due)}
	if IsDue(r, due.Add(-time.Minute)) {
		t.Error("due before local midnight")
	}
	prev := -1.0
	for h := 0; h <= 72; h++ {
		now := due.Add(time.Duration(h) * time.Hour)
		if !IsDue(r, now) {
			t.Errorf("not due at %s", now.In(la))
		}
		od := Overdue(r, now)
		if od < prev {
			t.Errorf("overdue went backwards at %s: %v < %v", now.In(la), od, prev)
		}
		prev = od
	}
	// Local midnight on Nov 2 (PST) is 25 hours after local midnight on
	// Nov 1 (PDT): overdue is 25/24 days by the absolute rule. A reader that
	// wants calendar days must convert; the rule does not promise them.
	nov2 := time.Date(2026, 11, 2, 0, 0, 0, 0, la)
	if got := Overdue(r, nov2); got < 1 || got > 1.05 {
		t.Errorf("overdue at local midnight Nov 2 = %v, want about 25/24", got)
	}
	// An undated row anchored just before the fall-back: its horizon is
	// 14*24h later in absolute time, which lands one hour EARLIER on the
	// local clock (23:00 on Nov 14 for a 00:00 Nov 1 anchor).
	u := Row{Status: mastery.StatusMastered, LastReviewed: due}
	at, _ := DueAt(u)
	if at.In(la).Hour() != 23 || at.In(la).Day() != 14 {
		t.Logf("note: undated horizon for a 00:00 PDT Nov 1 anchor lands at %s local", at.In(la))
	}
	if !IsDue(u, time.Date(2026, 11, 15, 0, 0, 0, 0, la)) {
		t.Error("undated row not due at local midnight Nov 15")
	}
}

// Property: once due, a row stays due as time advances (no surface can see
// a topic drop out of the queue without a review), and a row that is not
// due has a DueAt in the future (or is not mastered).
func TestReview_MonotoneInTime(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	base := time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)
	statuses := []mastery.Status{mastery.StatusMastered, mastery.StatusLearning, mastery.StatusPracticing, mastery.StatusUnseen}
	for i := 0; i < 20000; i++ {
		r := Row{Status: statuses[rng.Intn(len(statuses))]}
		if rng.Intn(3) > 0 {
			r.LastReviewed = base.AddDate(0, 0, -rng.Intn(60))
		}
		if rng.Intn(3) > 0 {
			r.LastAttempted = base.AddDate(0, 0, -rng.Intn(60))
		}
		if rng.Intn(2) == 0 {
			r.NextReviewDue = ptr(base.AddDate(0, 0, rng.Intn(80)-40))
		}
		was := false
		for d := 0; d < 90; d++ {
			now := base.AddDate(0, 0, d)
			is := IsDue(r, now)
			if was && !is {
				t.Fatalf("row %+v was due on day %d and not on day %d", r, d-1, d)
			}
			if r.Status != mastery.StatusMastered && is {
				t.Fatalf("row %+v is due but not mastered", r)
			}
			if !is {
				if at, ok := DueAt(r); r.Status == mastery.StatusMastered && (!ok || !at.After(now)) {
					t.Fatalf("row %+v not due at %s but DueAt=%v ok=%v", r, now, at, ok)
				}
			}
			was = is
		}
	}
}
