package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/clock"
	"github.com/chuma-beep/mathua/internal/fsrs"
	"github.com/chuma-beep/mathua/internal/review"
	"github.com/chuma-beep/mathua/internal/storage"
)

// The report reads a copy, and on a learner with one reviewed topic says
// what each section is for: the backlog, the per-day comparison, the
// calibration of the one prediction, the forecast.
func TestShadowReport(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MATHUA_ROOT", root)
	t.Setenv("MATHUA_PUBLIC_MODULES_ONLY", "1")
	db := filepath.Join(t.TempDir(), "report.db")
	store, err := storage.NewSQLiteStore(db)
	if err != nil {
		t.Fatal(err)
	}
	st, _ := store.CreateStudent("reported")
	store.SetCourseID(st.ID, "econ5153")
	p := review.DefaultParams()
	now := time.Date(2026, 11, 20, 17, 0, 0, 0, time.UTC)
	t0 := now.AddDate(0, 0, -25)
	t1 := t0.AddDate(0, 0, 2)
	m0 := p.Initial(fsrs.Good)
	m1, el, r := p.Review(m0, fsrs.Good, t0, t1)
	cid := "calc.deriv.power_rule"
	store.InsertReviewLog(storage.ReviewLogEntry{StudentID: st.ID, ConceptID: cid, At: t0, Source: "first_mastery", Rating: 3, SAfter: m0.Stability, DAfter: m0.Difficulty, IdempotencyKey: "k0", EngineMode: "shadow"})
	store.InsertReviewLog(storage.ReviewLogEntry{StudentID: st.ID, ConceptID: cid, At: t1, Source: "review", Rating: 3, ElapsedDays: el, SBefore: m0.Stability, SAfter: m1.Stability, DBefore: m0.Difficulty, DAfter: m1.Difficulty, RAtReview: r, IdempotencyKey: "k1", EngineMode: "shadow"})
	s, d := m1.Stability, m1.Difficulty
	row := &storage.ConceptProgress{StudentID: st.ID, ConceptID: cid, Status: "MASTERED", Attempts: 5, LastReviewed: &t1, LastAttempted: &t1, MasteredAt: &t0, SM2EFactor: 2.5}
	store.UpsertProgress(row)
	row.FSRSStability, row.FSRSDifficulty, row.FSRSLastReview, row.FSRSReps, row.FSRSState, row.FSRSSeed = &s, &d, &t1, 2, "review", "mastery"
	if err := store.WriteFSRSState(row); err != nil {
		t.Fatal(err)
	}
	if err := store.RecordFSRSError(storage.FSRSError{At: t1, StudentID: st.ID, ConceptID: cid, AttemptID: 9, Stage: "written", Error: "injected"}); err != nil {
		t.Fatal(err)
	}
	store.Close()
	before, _ := os.ReadFile(db)

	out := filepath.Join(t.TempDir(), "r.md")
	f, _ := os.Create(out)
	if err := run(f, db, "", now.Format(time.RFC3339), 10); err != nil {
		t.Fatal(err)
	}
	f.Close()
	b, _ := os.ReadFile(out)
	md := string(b)
	for _, want := range []string{
		"**fsrs_errors: 1 rows**", "1 fsrs_errors rows.",
		"### First-day backlog", "| FSRS, R < 0.90, state as stored | 1 |",
		"### Per day", "### Calibration", "| 90-100% | 1 |", "### Workload forecast, next 10 days",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("report lacks %q:\n%s", want, md)
		}
	}
	if after, _ := os.ReadFile(db); string(after) != string(before) {
		t.Error("the report changed the database")
	}
}

// L3: without -now the report is about the current time (the clock), not
// the day after the last log row; and when every event is same-day it says
// there is nothing to calibrate instead of printing a log loss.
func TestShadowReport_NowAndSameDayCalibration(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MATHUA_ROOT", root)
	t.Setenv("MATHUA_PUBLIC_MODULES_ONLY", "1")
	db := filepath.Join(t.TempDir(), "report.db")
	store, err := storage.NewSQLiteStore(db)
	if err != nil {
		t.Fatal(err)
	}
	st, _ := store.CreateStudent("sameday")
	p := review.DefaultParams()
	t0 := time.Date(2026, 10, 20, 17, 0, 0, 0, time.UTC)
	m0 := p.Initial(fsrs.Good)
	cid := "calc.deriv.power_rule"
	store.InsertReviewLog(storage.ReviewLogEntry{StudentID: st.ID, ConceptID: cid, At: t0, Source: "first_mastery", Rating: 3, SAfter: m0.Stability, DAfter: m0.Difficulty, IdempotencyKey: "k0", EngineMode: "shadow"})
	// A practice miss 2 minutes later, held at R = 1.
	store.InsertReviewLog(storage.ReviewLogEntry{StudentID: st.ID, ConceptID: cid, At: t0.Add(2 * time.Minute), Source: "practice_lapse", Rating: 1, SBefore: m0.Stability, SAfter: m0.Stability,
		DBefore: m0.Difficulty, DAfter: m0.Difficulty, RAtReview: 1, IdempotencyKey: "k1", EngineMode: "shadow", Held: review.HeldLapseRHigh})
	s, d := m0.Stability, m0.Difficulty
	row := &storage.ConceptProgress{StudentID: st.ID, ConceptID: cid, Status: "MASTERED", Attempts: 5, LastReviewed: &t0, LastAttempted: &t0, MasteredAt: &t0, SM2EFactor: 2.5}
	store.UpsertProgress(row)
	row.FSRSStability, row.FSRSDifficulty, row.FSRSLastReview, row.FSRSReps, row.FSRSState, row.FSRSSeed = &s, &d, &t0, 1, "review", "mastery"
	if err := store.WriteFSRSState(row); err != nil {
		t.Fatal(err)
	}
	store.Close()

	now := time.Date(2026, 11, 5, 21, 13, 0, 0, time.UTC) // "the current time"
	defer clock.NewSim(now).Install()()
	var b strings.Builder
	if err := run(&b, db, "", "", 5); err != nil {
		t.Fatal(err)
	}
	md := b.String()
	if !strings.Contains(md, "Now: 2026-11-05T21:13:00Z") {
		t.Errorf("default now is not the clock's time:\n%s", md)
	}
	if !strings.Contains(md, "Not enough between-day reviews to calibrate") || strings.Contains(md, "Log loss") {
		t.Errorf("same-day-only calibration:\n%s", md)
	}
	// -now still overrides.
	b.Reset()
	if err := run(&b, db, "", "2026-10-25T17:00:00Z", 5); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "Now: 2026-10-25T17:00:00Z") {
		t.Errorf("-now ignored")
	}
}
