package engine

import (
	"strings"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/mastery"
	"github.com/chuma-beep/mathua/internal/storage"
)

func TestReviewDueReason(t *testing.T) {
	now := time.Now()
	at := func(h int) *time.Time { v := now.Add(time.Duration(h) * time.Hour); return &v }
	decayed := func(days int) *time.Time { v := now.AddDate(0, 0, -mastery.DecayDays-days); return &v }
	for _, c := range []struct {
		p    *storage.ConceptProgress
		want string
	}{
		{&storage.ConceptProgress{Status: "MASTERED", NextReviewDue: at(-2)}, "review due today"},
		{&storage.ConceptProgress{Status: "MASTERED", NextReviewDue: at(-30)}, "review due 1 day ago"},
		{&storage.ConceptProgress{Status: "MASTERED", NextReviewDue: at(-75)}, "review due 3 days ago"},
		// A credited topic has no date: its due day is the decay horizon.
		{&storage.ConceptProgress{Status: "MASTERED", LastReviewed: decayed(0)}, "review due today"},
		{&storage.ConceptProgress{Status: "MASTERED", LastReviewed: decayed(3)}, "review due 3 days ago"},
		{&storage.ConceptProgress{Status: "MASTERED"}, "review due: not practised for a while"},
		{nil, "review due"},
	} {
		if got := reviewDueReason(c.p, now); got != c.want {
			t.Errorf("reviewDueReason(%+v) = %q, want %q", c.p, got, c.want)
		}
	}
}

// Every recommendation says why it is there: the plan topic names its
// course and module, a prerequisite the topic it is needed for, a pin and a
// mastery-check miss say so, and the review card how overdue the most
// overdue review is.
func TestRecommendations_CarryReasons(t *testing.T) {
	t.Setenv("MATHUA_REVIEW_FLOOR_DOMAINS", "")
	e, sid := seqFixture(t, seqCourse(time.Now().Add(40*24*time.Hour)))
	e.SetActiveCourses(sid, []string{"c"})

	plan := e.planReasons(sid)
	if got := plan["due1"]; got != "next in Math for economists · 1 Due" {
		t.Errorf("plan reason for due1 = %q", got)
	}
	if got := plan["fl"]; got != "needed for Due 1 in Math for economists" {
		t.Errorf("prerequisite reason for fl = %q", got)
	}

	threeDays := time.Now().Add(-75 * time.Hour)
	e.repo.UpsertProgress(&storage.ConceptProgress{StudentID: sid, ConceptID: "r", Status: "MASTERED", Attempts: 5, SM2EFactor: 2.5, LastReviewed: &threeDays, LastAttempted: &threeDays, NextReviewDue: &threeDays})
	if _, err := e.SetPins(sid, []string{"due3"}); err != nil {
		t.Fatal(err)
	}
	d := 0.8
	e.SubmitQuizAnswer(sid, "due2", "wrong", "1", 20, "q", &d)

	res, err := e.RecommendNext(sid, nil)
	if err != nil {
		t.Fatal(err)
	}
	why := map[string]string{}
	for _, r := range recs(res) {
		if r.Why == "" {
			t.Errorf("%s (%s) carries no reason", r.ConceptID, r.Kind)
		}
		key := r.ConceptID
		if r.Kind == "review" {
			key = "review"
		}
		why[key] = r.Why
	}
	if !strings.HasPrefix(why["review"], "review due 3 days ago") {
		t.Errorf("review reason = %q", why["review"])
	}
	if why["due3"] != "pinned" {
		t.Errorf("pin reason = %q", why["due3"])
	}
	if why["due2"] != "missed in the mastery check" {
		t.Errorf("mastery-check reason = %q (all: %v)", why["due2"], why)
	}
}
