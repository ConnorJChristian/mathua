package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/clock"
	"github.com/chuma-beep/mathua/internal/engine"
	"github.com/chuma-beep/mathua/internal/fsrs"
	"github.com/chuma-beep/mathua/internal/modules"
	"github.com/chuma-beep/mathua/internal/review"
	"github.com/chuma-beep/mathua/internal/storage"
)

// replayRun is everything one replay produced: the serves (the -json
// lines), the per-session surfaces (-surfaces) and the final database
// (-dump), whole and without the FSRS columns and review_log.
type replayRun struct {
	serves, surfaces, dump, dumpNoFSRS string
	db                                 string // the replayed copy, kept for the caller
}

// testStart is the simulated time the tests' fresh learner starts at: a
// fixed date unlike any wall date the suite runs on.
var testStart = time.Date(2026, 9, 14, 17, 0, 0, 0, time.UTC)

// replayFresh replays a fresh learner placed into econ5153 (the public
// modules), on a simulated clock and a fixed seed, in the given FSRS mode.
func replayFresh(t *testing.T, mode review.Mode, seed int64, sessions int) replayRun {
	t.Helper()
	root := os.Getenv("MATHUA_ROOT")
	if root == "" {
		abs, err := filepath.Abs("../..")
		if err != nil {
			t.Fatal(err)
		}
		root = abs
		t.Setenv("MATHUA_ROOT", root)
	}
	mods, err := modules.LoadPublic(root)
	if err != nil {
		t.Fatal(err)
	}
	engine.SetDeterministicSeed(seed)
	defer engine.SetDeterministicSeed(0)
	sim := clock.NewSim(testStart)
	defer sim.Install()()
	work, sid, err := freshLearner()
	if err != nil {
		t.Fatal(err)
	}
	r, err := openWith(work, sid, seed, mods)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.eng.SetActiveCourses(sid, []string{"econ5153"}); err != nil {
		t.Fatal(err)
	}
	if r, err = openWith(work, sid, seed, mods); err != nil {
		t.Fatal(err)
	}
	r.eng.SetFSRSMode(mode)
	r.sim, r.base = sim, testStart
	var serves, surfaces bytes.Buffer
	r.logw, r.surfw = &serves, &surfaces
	r.replayPlacement()
	if err := r.runSessions(sessions); err != nil {
		t.Fatal(err)
	}
	out := replayRun{serves: serves.String(), surfaces: surfaces.String(), db: work}
	dir := t.TempDir()
	for _, d := range []struct {
		with bool
		into *string
	}{{true, &out.dump}, {false, &out.dumpNoFSRS}} {
		p := filepath.Join(dir, "dump.txt")
		if err := dumpDB(r.db, p, d.with); err != nil {
			t.Fatal(err)
		}
		b, _ := os.ReadFile(p)
		*d.into = string(b)
	}
	return out
}

// firstDiff names the first line where two outputs differ.
func firstDiff(a, b string) string {
	al, bl := strings.Split(a, "\n"), strings.Split(b, "\n")
	for i := 0; i < len(al) && i < len(bl); i++ {
		if al[i] != bl[i] {
			cut := func(s string) string {
				if len(s) > 300 {
					return s[:300] + "..."
				}
				return s
			}
			return "line " + itoa(i+1) + ":\n  " + cut(al[i]) + "\n  " + cut(bl[i])
		}
	}
	return "lengths " + itoa(len(al)) + " and " + itoa(len(bl))
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

// Two same-seed replays produce identical output (FSRS Phase 2, step 1):
// the same serves, the same surfaces every session and the same final
// database. This is the gate "flag off = identical" rests on. It failed
// before the clock: the engine read the wall clock, so whether two writes
// fell in one second changed the order of equal-priority reviews, and
// SQLite's datetime('now') stamped rows with the run's own time.
func TestReplayDeterministic(t *testing.T) {
	if testing.Short() {
		t.Skip("replays sessions")
	}
	a := replayFresh(t, review.ModeOff, 5, 8)
	b := replayFresh(t, review.ModeOff, 5, 8)
	if a.serves == "" || a.surfaces == "" {
		t.Fatal("the replay served nothing")
	}
	if a.serves != b.serves {
		t.Errorf("serves differ: %s", firstDiff(a.serves, b.serves))
	}
	if a.surfaces != b.surfaces {
		t.Errorf("surfaces differ: %s", firstDiff(a.surfaces, b.surfaces))
	}
	if a.dump != b.dump {
		t.Errorf("databases differ: %s", firstDiff(a.dump, b.dump))
	}
	// A different seed is a different learner's luck: the check above is
	// not passing because the replay ignores its inputs.
	c := replayFresh(t, review.ModeOff, 6, 8)
	if c.serves == a.serves {
		t.Error("seed 6 served exactly what seed 5 did; the seed reaches nothing")
	}
}

// The simulated clock is the only time the replay reads: sessions start
// at base, a day apart in the aged copy, and every row the run writes
// carries a time from it, never the wall clock.
func TestReplayWritesOnlyClockTimes(t *testing.T) {
	if testing.Short() {
		t.Skip("replays sessions")
	}
	run := replayFresh(t, review.ModeOff, 5, 3)
	db, err := sql.Open("sqlite3", run.db)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	wall := time.Now().UTC().Format("2006-01-02")
	if wall == testStart.Format("2006-01-02") {
		t.Skip("the wall date is the simulated date; nothing to tell apart")
	}
	for _, q := range []string{
		`select count(*) from attempts where timestamp like '` + wall + `%'`,
		`select count(*) from student_topic_speed where updated_at like '` + wall + `%'`,
		`select count(*) from server_sessions where updated_at like '` + wall + `%'`,
		`select count(*) from quiz_completions where completed_at like '` + wall + `%'`,
	} {
		var n int
		if err := db.QueryRow(q).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("%s: %d rows carry the wall date %s", q, n, wall)
		}
	}
}

// Shadow mode reads nothing it writes (FSRS Phase 2, gate 4): the same
// replay in off and in shadow serves the same topics with the same reasons,
// shows the same surfaces every session (the review queue, the due count,
// Up next, the mixed session, pacing, plan order, the gate, the mastery
// check's picks, XP, the progress view) and leaves the same database but
// for the fsrs_* columns and review_log. Shadow did write: the log is not
// empty, and every stored state is the replay of its log.
func TestShadowEqualsOff(t *testing.T) {
	if testing.Short() {
		t.Skip("replays sessions")
	}
	off := replayFresh(t, review.ModeOff, 5, 10)
	sh := replayFresh(t, review.ModeShadow, 5, 10)
	if off.serves != sh.serves {
		t.Errorf("serves differ: %s", firstDiff(off.serves, sh.serves))
	}
	if off.surfaces != sh.surfaces {
		t.Errorf("surfaces differ: %s", firstDiff(off.surfaces, sh.surfaces))
	}
	if off.dumpNoFSRS != sh.dumpNoFSRS {
		t.Errorf("databases differ outside FSRS: %s", firstDiff(off.dumpNoFSRS, sh.dumpNoFSRS))
	}
	if off.dump == sh.dump {
		t.Error("shadow wrote nothing: the comparison proves nothing")
	}
	store, err := storage.NewSQLiteStore(sh.db)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	db, _ := sql.Open("sqlite3", sh.db)
	defer db.Close()
	var sid string
	db.QueryRow(`select id from students limit 1`).Scan(&sid)
	all, _ := store.GetAllProgress(sid)
	log, _ := store.ReviewLogFor(sid, "")
	if len(log) == 0 {
		t.Fatal("shadow wrote no review_log rows")
	}
	byTopic := map[string][]review.Event{}
	for _, e := range log {
		byTopic[e.ConceptID] = append(byTopic[e.ConceptID], review.Event{At: e.At, Rating: fsrs.Rating(e.Rating), ElapsedDays: e.ElapsedDays, Source: e.Source, SAfter: e.SAfter, DAfter: e.DAfter, Held: e.Held})
	}
	states := 0
	for cid, p := range all {
		m, last := engine.FSRSMemory(p)
		if m.IsZero() {
			if len(byTopic[cid]) > 0 {
				t.Errorf("%s: %d log rows, no state", cid, len(byTopic[cid]))
			}
			continue
		}
		states++
		got, gotLast, err := review.ReplayLast(review.DefaultParams(), byTopic[cid])
		if d := review.Diff(m, got); d != "" || err != nil || !gotLast.Truncate(time.Second).Equal(last.Truncate(time.Second)) {
			t.Errorf("%s: replay %s %v; last %v vs %v", cid, d, err, gotLast, last)
		}
	}
	t.Logf("shadow: %d review_log rows, %d topics with state", len(log), states)
}

// The one-day aging moves the FSRS state too (plan 4.9): fsrs_last_review
// and review_log.at go back exactly a day, the log keeps its fixed width.
func TestAgeOneDayShiftsFSRS(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "age.db")
	store, err := storage.NewSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	st, _ := store.CreateStudent("age")
	last := time.Date(2026, 11, 1, 8, 30, 0, 0, time.UTC) // the DST day in the learner's zone
	s, d := 2.3065, 2.1
	p := &storage.ConceptProgress{StudentID: st.ID, ConceptID: "x", Status: "MASTERED", LastReviewed: &last, SM2EFactor: 2.5}
	store.UpsertProgress(p)
	p.FSRSStability, p.FSRSDifficulty, p.FSRSLastReview, p.FSRSReps, p.FSRSState = &s, &d, &last, 1, "review"
	if err := store.WriteFSRSState(p); err != nil {
		t.Fatal(err)
	}
	at := last.Add(123456789 * time.Nanosecond)
	store.InsertReviewLog(storage.ReviewLogEntry{StudentID: st.ID, ConceptID: "x", At: at, Source: "review", Rating: 3, IdempotencyKey: "k", EngineMode: "shadow"})
	store.Close()
	db, _ := sql.Open("sqlite3", path)
	defer db.Close()
	r := &replay{db: db, sid: st.ID}
	if err := r.ageOneDay(); err != nil {
		t.Fatal(err)
	}
	var lr, la string
	db.QueryRow(`select fsrs_last_review from concept_progress where student_id = ?`, st.ID).Scan(&lr)
	db.QueryRow(`select at from review_log where student_id = ?`, st.ID).Scan(&la)
	if lr != "2026-10-31T08:30:00Z" {
		t.Errorf("fsrs_last_review %q", lr)
	}
	if want := at.AddDate(0, 0, -1).UTC().Format(storage.ReviewLogAtLayout); la != want {
		t.Errorf("review_log.at %q, want %q", la, want)
	}
}
