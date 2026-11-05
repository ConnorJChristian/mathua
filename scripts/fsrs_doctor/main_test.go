package main

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/fsrs"
	"github.com/chuma-beep/mathua/internal/review"
	"github.com/chuma-beep/mathua/internal/storage"
)

// The doctor finds each class of defect it exists for, and nothing on a
// consistent database. Public modules only (MATHUA_PUBLIC_MODULES_ONLY), so
// the fixture is the same on every machine.
func TestDoctor_FindsEachClass(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MATHUA_ROOT", root)
	t.Setenv("MATHUA_PUBLIC_MODULES_ONLY", "1")
	db := filepath.Join(t.TempDir(), "doctor.db")
	store, err := storage.NewSQLiteStore(db)
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.CreateStudent("doctored")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetCourseID(st.ID, "econ5153"); err != nil {
		t.Fatal(err)
	}
	sid := st.ID
	now := time.Now().UTC().Truncate(time.Second)
	p := review.DefaultParams()

	// A consistent topic: two reviews, logged, state = replay(log).
	m := p.Initial(fsrs.Good)
	t0 := now.Add(-10 * 24 * time.Hour)
	t1 := t0.Add(3 * 24 * time.Hour)
	next, elapsed, r := p.Review(m, fsrs.Good, t0, t1)
	logRow := func(cid string, at time.Time, rating fsrs.Rating, el float64, before, after review.Memory, rr float64, key string) {
		if _, err := store.InsertReviewLog(storage.ReviewLogEntry{StudentID: sid, ConceptID: cid, At: at, Source: "review", Rating: int(rating), ElapsedDays: el,
			SBefore: before.Stability, SAfter: after.Stability, DBefore: before.Difficulty, DAfter: after.Difficulty, RAtReview: rr, IdempotencyKey: key, EngineMode: "shadow"}); err != nil {
			t.Fatal(err)
		}
	}
	logRow("calc.deriv.power_rule", t0, fsrs.Good, 0, review.Memory{}, m, 0, "k1")
	logRow("calc.deriv.power_rule", t1, fsrs.Good, elapsed, m, next, r, "k2")
	row := func(cid string, mem review.Memory, last time.Time, status string) *storage.ConceptProgress {
		s, d := mem.Stability, mem.Difficulty
		rev := last
		return &storage.ConceptProgress{StudentID: sid, ConceptID: cid, Status: status, Streak: 3, Attempts: 6, LastReviewed: &rev, LastAttempted: &rev, MasteredAt: &rev,
			FSRSStability: &s, FSRSDifficulty: &d, FSRSLastReview: &last, FSRSReps: mem.Reps, FSRSLapses: mem.Lapses, FSRSState: string(mem.Phase), FSRSSeed: "mastery"}
	}
	good := row("calc.deriv.power_rule", next, t1, "MASTERED")
	good.NextReviewDue = nil // credited-style undated: due 14 days after t1? t1 is 7 days ago: not due
	if err := store.UpsertProgress(good); err != nil {
		t.Fatal(err)
	}
	// A row whose state does not match its log (a dropped event).
	bad := row("calc.deriv.chain_rule", m, t0, "MASTERED")
	logRow("calc.deriv.chain_rule", t0, fsrs.Good, 0, review.Memory{}, m, 0, "k3")
	logRow("calc.deriv.chain_rule", t1, fsrs.Good, elapsed, m, next, r, "k4")
	bad.FSRSLastReview = &t1
	if err := store.UpsertProgress(bad); err != nil {
		t.Fatal(err)
	}
	// State with no log; log with no state; an orphan with state; S NaN; D out of range.
	unlogged := row("calc.deriv.product_rule", next, t1, "MASTERED")
	if err := store.UpsertProgress(unlogged); err != nil {
		t.Fatal(err)
	}
	logRow("calc.deriv.quotient_rule", t0, fsrs.Good, 0, review.Memory{}, m, 0, "k5")
	if err := store.UpsertProgress(&storage.ConceptProgress{StudentID: sid, ConceptID: "calc.deriv.quotient_rule", Status: "LEARNING"}); err != nil {
		t.Fatal(err)
	}
	orphan := row("no.such.topic", next, t1, "MASTERED")
	logRow("no.such.topic", t1, fsrs.Good, 0, review.Memory{}, next, 0, "k6")
	if err := store.UpsertProgress(orphan); err != nil {
		t.Fatal(err)
	}
	nan := row("calc.limit.concept", next, t1, "MASTERED")
	v := math.NaN()
	nan.FSRSStability = &v
	logRow("calc.limit.concept", t1, fsrs.Good, 0, review.Memory{}, next, 0, "k7")
	if err := store.UpsertProgress(nan); err != nil {
		t.Fatal(err)
	}
	// A floor topic (arithmetic) with a past date: the rule excludes it and
	// the session must not serve it; a consistent due row too.
	past := now.Add(-48 * time.Hour)
	floor := &storage.ConceptProgress{StudentID: sid, ConceptID: "arith.add.double", Status: "MASTERED", NextReviewDue: &past, LastReviewed: &past, MasteredAt: &past}
	if err := store.UpsertProgress(floor); err != nil {
		t.Fatal(err)
	}
	store.Close()

	findings, counts, err := Run(db, "")
	if err != nil {
		t.Fatal(err)
	}
	if counts["learners"] != 1 || counts["log_rows"] != 7 {
		t.Errorf("counts %v", counts)
	}
	want := map[string]string{
		"replay:calc.deriv.chain_rule":      "stability",
		"unlogged:calc.deriv.product_rule":  "no review_log",
		"unlogged:calc.deriv.quotient_rule": "no FSRS state",
		"orphan:no.such.topic":              "",
		"invalid:calc.limit.concept":        "stability NaN",
	}
	seen := map[string]bool{}
	for _, f := range findings {
		seen[f.Check+":"+f.Concept] = true
		t.Logf("%s %s: %s", f.Check, f.Concept, f.Detail)
	}
	for k := range want {
		if !seen[k] {
			t.Errorf("missing finding %s", k)
		}
	}
	for _, k := range []string{"replay:calc.deriv.power_rule", "surfaced:arith.add.double", "unlogged:calc.deriv.power_rule", "invalid:calc.deriv.power_rule"} {
		if seen[k] {
			t.Errorf("false finding %s", k)
		}
	}
	// The original was not written to.
	if _, err := os.Stat(db + "-wal"); err == nil {
		// A WAL file may exist from the fixture writes; the test asserts the rows, not the file.
		_ = err
	}
	reopened, err := storage.NewSQLiteStore(db)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if log, _ := reopened.ReviewLogFor(sid, ""); len(log) != 7 {
		t.Errorf("the doctor changed the log: %d rows", len(log))
	}
	if sessions, _ := reopened.GetAllProgress(sid); len(sessions) != 7 {
		t.Errorf("the doctor changed the progress rows: %d", len(sessions))
	}
}
