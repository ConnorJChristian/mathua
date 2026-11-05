package engine

import (
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/mastery"
	"github.com/chuma-beep/mathua/internal/storage"
)

// The progress payload carries due-ness from the one rule (review.IsDue,
// exclusions applied), separate from the DECAYING pacing status. The web
// used to label DECAYING as "Due for review" (Phase 0 review, F1): a
// topic with a long interval reviewed over DecayDays ago is decaying for
// pacing and not due, and must say so.
func TestProgressWithMasteryPct_DueIsTheOneRuleNotDecay(t *testing.T) {
	e := testEngine(t)
	st, _ := e.CreateStudent("due_vs_decay")
	now := time.Now().UTC()
	old := now.Add(-(mastery.DecayDays + 1) * 24 * time.Hour)
	future := now.AddDate(0, 0, 30)
	past := now.Add(-time.Hour)
	fresh := now.Add(-time.Hour)
	rows := []*storage.ConceptProgress{
		// a: decayed for pacing, scheduled 30 days out: not due.
		{StudentID: st.ID, ConceptID: "a", Status: "MASTERED", LastReviewed: &old, NextReviewDue: &future, MasteredAt: &old},
		// b: reviewed an hour ago, date passed: due and not decaying.
		{StudentID: st.ID, ConceptID: "b", Status: "MASTERED", LastReviewed: &fresh, NextReviewDue: &past, MasteredAt: &fresh},
	}
	if err := e.repo.UpsertProgressBatch(rows); err != nil {
		t.Fatal(err)
	}
	views, err := e.ProgressWithMasteryPct(st.ID)
	if err != nil {
		t.Fatal(err)
	}
	a, b := views["a"], views["b"]
	if a.Status != "DECAYING" || a.Due {
		t.Errorf("a: status %q due %v; want DECAYING and not due (future date)", a.Status, a.Due)
	}
	if b.Status != "MASTERED" || !b.Due {
		t.Errorf("b: status %q due %v; want MASTERED and due (past date)", b.Status, b.Due)
	}
	// The same answer the due count gives.
	stored, _ := e.repo.GetAllProgress(st.ID)
	want := e.DueReviewSet(st.ID, stored, time.Now().UTC())
	for id, v := range views {
		if v.Due != want[id] {
			t.Errorf("%s: payload due=%v, DueReviewSet=%v", id, v.Due, want[id])
		}
	}
}
