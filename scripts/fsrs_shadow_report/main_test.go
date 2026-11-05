package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
