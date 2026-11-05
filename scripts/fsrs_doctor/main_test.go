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
	if err := store.WriteFSRSState(good); err != nil {
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
	if err := store.WriteFSRSState(bad); err != nil {
		t.Fatal(err)
	}
	// State with no log; log with no state; an orphan with state; S NaN; D out of range.
	unlogged := row("calc.deriv.product_rule", next, t1, "MASTERED")
	if err := store.UpsertProgress(unlogged); err != nil {
		t.Fatal(err)
	}
	if err := store.WriteFSRSState(unlogged); err != nil {
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
	if err := store.WriteFSRSState(orphan); err != nil {
		t.Fatal(err)
	}
	nan := row("calc.limit.concept", next, t1, "MASTERED")
	v := math.NaN()
	nan.FSRSStability = &v
	logRow("calc.limit.concept", t1, fsrs.Good, 0, review.Memory{}, next, 0, "k7")
	if err := store.UpsertProgress(nan); err != nil {
		t.Fatal(err)
	}
	if err := store.WriteFSRSState(nan); err != nil {
		t.Fatal(err)
	}
	// A log row with rating 0 (an event-to-rating bug): Replay skips it, so
	// the state replays consistently, and the doctor still flags the row.
	zeroRated := row("calc.deriv.concept", m, t0, "MASTERED")
	logRow("calc.deriv.concept", t0, fsrs.Good, 0, review.Memory{}, m, 0, "k8")
	logRow("calc.deriv.concept", t0, fsrs.Rating(0), 0, m, m, 0, "k9")
	if err := store.UpsertProgress(zeroRated); err != nil {
		t.Fatal(err)
	}
	if err := store.WriteFSRSState(zeroRated); err != nil {
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
	if counts["learners"] != 1 || counts["log_rows"] != 9 {
		t.Errorf("counts %v", counts)
	}
	want := map[string]string{
		"replay:calc.deriv.chain_rule":      "stability",
		"unlogged:calc.deriv.product_rule":  "no review_log",
		"unlogged:calc.deriv.quotient_rule": "no FSRS state",
		"orphan:no.such.topic":              "",
		"invalid:calc.limit.concept":        "stability NaN",
		"invalid:calc.deriv.concept":        "invalid rating",
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
	for _, k := range []string{"replay:calc.deriv.power_rule", "surfaced:arith.add.double", "unlogged:calc.deriv.power_rule", "invalid:calc.deriv.power_rule", "replay:calc.deriv.concept"} {
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
	if log, _ := reopened.ReviewLogFor(sid, ""); len(log) != 9 {
		t.Errorf("the doctor changed the log: %d rows", len(log))
	}
	if sessions, _ := reopened.GetAllProgress(sid); len(sessions) != 8 {
		t.Errorf("the doctor changed the progress rows: %d", len(sessions))
	}
}

// doctorFixture is a store on a temp file with one learner and one
// consistent topic (first mastery at t0, a review at t1), for the checks
// added after the review.
func doctorFixture(t *testing.T) (db string, store *storage.SQLiteStore, sid string, t0, t1 time.Time) {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MATHUA_ROOT", root)
	t.Setenv("MATHUA_PUBLIC_MODULES_ONLY", "1")
	db = filepath.Join(t.TempDir(), "doctor.db")
	store, err = storage.NewSQLiteStore(db)
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.CreateStudent("doctored")
	if err != nil {
		t.Fatal(err)
	}
	sid = st.ID
	p := review.DefaultParams()
	t0 = time.Now().UTC().Truncate(time.Second).Add(-10 * 24 * time.Hour)
	t1 = t0.Add(3*24*time.Hour + 500*time.Millisecond)
	m := p.Initial(fsrs.Good)
	next, elapsed, r := p.Review(m, fsrs.Good, t0, t1)
	cid := "calc.deriv.power_rule"
	for i, x := range []storage.ReviewLogEntry{
		{At: t0, Source: "first_mastery", Rating: 3, SAfter: m.Stability, DAfter: m.Difficulty},
		{At: t1, Source: "review", Rating: 3, ElapsedDays: elapsed, SBefore: m.Stability, SAfter: next.Stability, DBefore: m.Difficulty, DAfter: next.Difficulty, RAtReview: r},
	} {
		x.StudentID, x.ConceptID, x.EngineMode, x.IdempotencyKey = sid, cid, "shadow", "k"+string(rune('0'+i))
		if _, err := store.InsertReviewLog(x); err != nil {
			t.Fatal(err)
		}
	}
	s, d, last := next.Stability, next.Difficulty, t1
	row := &storage.ConceptProgress{StudentID: sid, ConceptID: cid, Status: "MASTERED", Streak: 3, Attempts: 7, LastReviewed: &t1, LastAttempted: &t1, MasteredAt: &t0,
		FSRSStability: &s, FSRSDifficulty: &d, FSRSLastReview: &last, FSRSReps: next.Reps, FSRSLapses: next.Lapses, FSRSState: string(next.Phase), FSRSSeed: "mastery"}
	if err := store.UpsertProgress(row); err != nil {
		t.Fatal(err)
	}
	if err := store.WriteFSRSState(row); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"s1", "s2", "s3"} {
		if err := store.EnsureSession(id, sid); err != nil {
			t.Fatal(err)
		}
	}
	attempt := func(at time.Time, source, expected string, correct bool) {
		if _, err := store.RecordAttempt(storage.AttemptEntry{SessionID: "s1", StudentID: sid, ConceptID: cid, Answer: "x", Expected: expected,
			Correct: correct, Timestamp: at, Source: source}); err != nil {
			t.Fatal(err)
		}
	}
	attempt(t0, "practice", "42", true)                         // the mastering answer
	attempt(t1, "review", "42", true)                           // logged at t1 (the same second)
	attempt(t0.Add(-time.Hour), "review", "42", true)           // before the topic's first event: not in the model yet
	attempt(t1.Add(time.Hour), "review", "recall:card-1", true) // self-graded: no event
	attempt(t1.Add(2*time.Hour), "practice", "42", false)       // practice: left out
	return db, store, sid, t0, t1
}

// M2: an event the app dropped is visible. The fsrs_errors row is an
// "error" finding and the answer with no log row a "missing" one; the
// doctor exits non-zero (findings). The consistent fixture has none.
func TestDoctor_DroppedEventIsFound(t *testing.T) {
	db, store, _, _, _ := doctorFixture(t)
	store.Close()
	if findings, _, err := Run(db, ""); err != nil || len(findings) != 0 {
		t.Fatalf("clean fixture: %v %+v", err, findings)
	}

	db, store, sid, _, t1 := doctorFixture(t)
	dropped := t1.Add(26 * time.Hour)
	id, err := store.RecordAttempt(storage.AttemptEntry{SessionID: "s2", StudentID: sid, ConceptID: "calc.deriv.power_rule", Answer: "7", Expected: "42",
		Timestamp: dropped, Source: "review"})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RecordFSRSError(storage.FSRSError{At: dropped, StudentID: sid, ConceptID: "calc.deriv.power_rule", AttemptID: id, Stage: "written", Error: "injected"}); err != nil {
		t.Fatal(err)
	}
	store.Close()
	findings, counts, err := Run(db, "")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	for _, f := range findings {
		seen[f.Check+":"+f.Concept]++
		t.Logf("%s %s: %s", f.Check, f.Concept, f.Detail)
	}
	if seen["error:calc.deriv.power_rule"] != 1 || seen["missing:calc.deriv.power_rule"] != 1 || len(findings) != 2 || counts["fsrs_errors"] != 1 {
		t.Errorf("findings %+v counts %v; want exactly one error and one missing on calc.deriv.power_rule", findings, counts)
	}

	// The rules in force decide: with judged=off a proof review needs no row;
	// with course_tests=off a quiz answer is left out.
	db, store, sid, _, t1 = doctorFixture(t)
	for i, x := range []struct{ source, expected string }{{"review", "proof:p1"}, {"quiz", "42"}} {
		if _, err := store.RecordAttempt(storage.AttemptEntry{SessionID: "s3", StudentID: sid, ConceptID: "calc.deriv.power_rule", Answer: "x", Expected: x.expected,
			Timestamp: t1.Add(time.Duration(30+i) * time.Hour), Source: x.source}); err != nil {
			t.Fatal(err)
		}
	}
	store.Close()
	if findings, _, _ := Run(db, ""); len(findings) != 2 {
		t.Errorf("default rules: %+v, want the proof and the quiz answer missing", findings)
	}
	off := review.DefaultRules()
	off.Judged, off.CourseTests = false, false
	for _, x := range []struct {
		source, expected string
		def, off         bool
	}{
		{"review", "42", true, true}, {"review", "proof:p1", true, false}, {"review", "recall:c", false, false},
		{"quiz", "42", true, false}, {"quiz", "recall:c", false, false}, {"practice", "42", false, false}, {"diagnostic", "42", false, false},
	} {
		a := storage.AttemptEntry{Source: x.source, Expected: x.expected}
		if got := needsEvent(a, review.DefaultRules()); got != x.def {
			t.Errorf("default rules, %s %s: needsEvent %v", x.source, x.expected, got)
		}
		if got := needsEvent(a, off); got != x.off {
			t.Errorf("judged=off,course_tests=off, %s %s: needsEvent %v", x.source, x.expected, got)
		}
	}
}
