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
	"github.com/chuma-beep/mathua/internal/modules"
	"github.com/chuma-beep/mathua/internal/review"
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
