package review

import (
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/mastery"
)

type dueCase struct {
	Name          string   `json:"name"`
	Status        string   `json:"status"`
	LastReviewed  *float64 `json:"last_reviewed"`
	LastAttempted *float64 `json:"last_attempted"`
	NextReviewDue *float64 `json:"next_review_due"`
	Due           bool     `json:"due"`
	Overdue       float64  `json:"overdue"`
}

type dueCases struct {
	DecayDays float64   `json:"decay_days"`
	Cases     []dueCase `json:"cases"`
}

func loadCases(t *testing.T) dueCases {
	t.Helper()
	raw, err := os.ReadFile("testdata/due_cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cs dueCases
	if err := json.Unmarshal(raw, &cs); err != nil {
		t.Fatal(err)
	}
	if len(cs.Cases) == 0 {
		t.Fatal("no cases")
	}
	return cs
}

// The fixture is shared with scripts/learning_report_test.py, so the
// Python reader applies the same rule. Its decay window must be the Go
// constant's, not a copy that can drift (AGENTS.md section 7).
func TestDueCasesFixture(t *testing.T) {
	cs := loadCases(t)
	if cs.DecayDays != mastery.DecayDays {
		t.Fatalf("fixture decay_days = %v, mastery.DecayDays = %v", cs.DecayDays, mastery.DecayDays)
	}
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	at := func(d *float64) time.Time {
		if d == nil {
			return time.Time{}
		}
		return now.Add(time.Duration(*d * 24 * float64(time.Hour)))
	}
	for _, c := range cs.Cases {
		r := Row{Status: mastery.Status(c.Status), LastReviewed: at(c.LastReviewed), LastAttempted: at(c.LastAttempted)}
		if c.NextReviewDue != nil {
			d := at(c.NextReviewDue)
			r.NextReviewDue = &d
		}
		if got := IsDue(r, now); got != c.Due {
			t.Errorf("%s: IsDue = %v, want %v", c.Name, got, c.Due)
		}
		if got := Overdue(r, now); math.Abs(got-c.Overdue) > 1e-6 {
			t.Errorf("%s: Overdue = %v, want %v", c.Name, got, c.Overdue)
		}
	}
}

// Due comes no later than the scheduled date and never before it; without a
// date, exactly at the decay horizon, which is when mastery.EffectiveStatus
// reads DECAYING: an undated row is due on the day it decays, so the count
// and the review session cannot disagree about it.
func TestUndatedRowIsDueWhenItDecays(t *testing.T) {
	now := time.Now().UTC()
	for days := 0.0; days <= 40; days += 0.5 {
		r := Row{Status: mastery.StatusMastered, LastReviewed: now.Add(-time.Duration(days * 24 * float64(time.Hour)))}
		decayed := mastery.EffectiveStatus(r.Status, days, mastery.DecayDays) == "DECAYING"
		if got := IsDue(r, now); got != decayed {
			t.Fatalf("%.1f days: IsDue = %v, decayed = %v", days, got, decayed)
		}
	}
}

func TestDueAt(t *testing.T) {
	now := time.Now().UTC()
	past := now.Add(-time.Hour)
	if at, ok := DueAt(Row{Status: mastery.StatusMastered, NextReviewDue: &past, LastReviewed: now.AddDate(0, 0, -30)}); !ok || !at.Equal(past) {
		t.Errorf("a dated row is due at its date: %v %v", at, ok)
	}
	lr := now.AddDate(0, 0, -5)
	if at, ok := DueAt(Row{Status: mastery.StatusMastered, LastReviewed: lr}); !ok || !at.Equal(lr.AddDate(0, 0, mastery.DecayDays)) {
		t.Errorf("an undated row is due DecayDays after its review: %v %v", at, ok)
	}
	if _, ok := DueAt(Row{Status: mastery.StatusLearning, NextReviewDue: &past}); ok {
		t.Error("a learning row has no review date")
	}
}
