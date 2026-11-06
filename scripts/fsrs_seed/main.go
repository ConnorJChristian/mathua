// fsrs_seed gives FSRS state to the mastered topics a learner demonstrated
// before shadow mode existed (FSRS plan 3.4), by replaying each topic's own
// attempts through FSRS in time order: the reference first review (Good)
// at its mastery, then each later review or mastery-check answer (right
// Good, wrong Again) and each practice miss as a lapse under R < 0.95,
// under the one-raise-per-day cap. It is the same replay shadow mode runs
// when an event first touches such a row (Rules.SeedOnTouch), so seeding
// first or letting shadow seed on touch gives the same state.
//
// Credited and placement-only rows (mastered, but not by the learner's own
// answer: no MasteredAt) get no state, even with right answers after the
// credit: their first FSRS event initialises it from the reference first
// review for its rating, labelled credit (FSRS Phase 2 review, L1).
//
// It is a dry run by default: it prints what it would write and changes
// nothing. -apply writes it, one transaction per topic; rerunning is a
// no-op (the log keys are the attempts' own).
//
// Stop the app before -apply. -apply refuses while the database has a
// "-wal" or "-shm" file beside it: SQLite keeps those exactly while a
// connection is open (they go when the last one closes cleanly), so they
// mean the app or another process has the file open, or one crashed.
// -force overrides when you know nothing is writing. (A seed committed
// while the app serves is safe since the review's M1 fix, an answer
// skips seed rows already stored, but the app should not be serving
// while its database is rewritten underneath it.)
//
//	MATHUA_ROOT=<install> go run ./scripts/fsrs_seed -db mathua.db [-student id] [-apply [-force]]
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chuma-beep/mathua/internal/clock"
	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/engine"
	"github.com/chuma-beep/mathua/internal/generator/all"
	"github.com/chuma-beep/mathua/internal/modules"
	"github.com/chuma-beep/mathua/internal/planning"
	"github.com/chuma-beep/mathua/internal/review"
	"github.com/chuma-beep/mathua/internal/storage"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dbPath := flag.String("db", "", "SQLite database (written only with -apply)")
	student := flag.String("student", "", "one learner's id (default: every learner)")
	apply := flag.Bool("apply", false, "write the seeds (default: print them)")
	force := flag.Bool("force", false, "with -apply: write even though the database looks open (a -wal or -shm file)")
	flag.Parse()
	if *dbPath == "" {
		fmt.Fprintln(os.Stderr, "usage: fsrs_seed -db <path> [-student <id>] [-apply [-force]]")
		os.Exit(2)
	}
	if err := run(*dbPath, *student, *apply, *force, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "fsrs_seed:", err)
		os.Exit(1)
	}
}

// liveWriter names the -wal or -shm file beside dbPath, "" when neither
// exists. SQLite creates them when a connection first reads a WAL database
// and removes them when the last connection closes cleanly, so either one
// means the database is open (the app is running) or a process crashed.
func liveWriter(dbPath string) string {
	for _, suffix := range []string{"-wal", "-shm"} {
		if _, err := os.Stat(dbPath + suffix); err == nil {
			return dbPath + suffix
		}
	}
	return ""
}

func run(dbPath, only string, apply, force bool, out *os.File) error {
	if _, err := os.Stat(dbPath); err != nil {
		return err
	}
	if apply && !force {
		if f := liveWriter(dbPath); f != "" {
			return fmt.Errorf("%s exists: the database is open (is the app running?) or a process crashed. "+
				"Stop the app first (docker compose stop mathua) and run again; pass -force only if you know nothing is writing", f)
		}
	}
	root := os.Getenv("MATHUA_ROOT")
	if root == "" {
		return fmt.Errorf("MATHUA_ROOT must name the install (its data/ and modules)")
	}
	mods, err := modules.Default()
	if err != nil {
		return err
	}
	dag, err := concepts.LoadDirs(filepath.Join(root, "data", "concepts"), mods.ConceptDirs()...)
	if err != nil {
		return err
	}
	open := dbPath
	if !apply {
		// A dry run reads a copy, so not even the store's migrations touch
		// the file.
		dir, err := os.MkdirTemp("", "fsrs_seed")
		if err != nil {
			return err
		}
		defer os.RemoveAll(dir)
		open = filepath.Join(dir, "copy.db")
		src, err := sql.Open("sqlite3", "file:"+dbPath+"?mode=ro")
		if err != nil {
			return err
		}
		_, err = src.Exec("VACUUM INTO ?", open)
		src.Close()
		if err != nil {
			return err
		}
	}
	store, err := storage.NewSQLiteStore(open)
	if err != nil {
		return err
	}
	defer store.Close()
	planner, err := planning.Load(filepath.Join(root, "data", "courses.json"), dag, mods.CourseFiles()...)
	if err != nil {
		return err
	}
	eng := engine.New(store, dag, all.Registry(), nil, planner)
	eng.SetModules(mods)
	eng.SetFSRSMode(review.ModeShadow) // the log rows say shadow wrote them
	rules, _ := review.CurrentRules()
	eng.SetFSRSRules(rules)
	db, err := sql.Open("sqlite3", open)
	if err != nil {
		return err
	}
	defer db.Close()
	ids := []string{only}
	if only == "" {
		ids = nil
		rows, err := db.Query(`SELECT id FROM students ORDER BY id`)
		if err != nil {
			return err
		}
		for rows.Next() {
			var id string
			rows.Scan(&id)
			ids = append(ids, id)
		}
		rows.Close()
	}
	verb := "would seed"
	if apply {
		verb = "seeded"
	}
	fmt.Fprintf(out, "fsrs_seed (%s), rules %s\n", map[bool]string{true: "apply", false: "dry run"}[apply], rules)
	total, wrote, left := 0, 0, 0
	for _, sid := range ids {
		seeds, none, err := eng.FSRSSeedPlan(sid, clock.Now())
		if err != nil {
			return fmt.Errorf("%s: %w", sid, err)
		}
		for _, s := range seeds {
			total++
			fmt.Fprintf(out, "%s %s %s: %d events, S %.4f D %.4f reps %d lapses %d, last review %s, R now %.3f\n",
				verb, sid, s.ConceptID, len(s.Log), s.State.Stability, s.State.Difficulty, s.State.Reps, s.State.Lapses,
				s.Last.UTC().Format(time.RFC3339), review.Retrievability(s.State, review.ElapsedWholeDays(s.Last, clock.Now())))
			for _, e := range s.Log {
				held := ""
				if e.Held != "" {
					held = " held " + e.Held
				}
				fmt.Fprintf(out, "    %s %-14s rating %d  S %.4f -> %.4f  R %.3f  %s%s\n", e.At.UTC().Format(time.RFC3339), e.Source, e.Rating, e.SBefore, e.SAfter, e.RAtReview, e.IdempotencyKey, held)
			}
			if apply {
				ok, err := eng.ApplyFSRSSeed(sid, s)
				if err != nil {
					return fmt.Errorf("%s %s: %w", sid, s.ConceptID, err)
				}
				if ok {
					wrote++
				}
			}
		}
		for _, id := range none {
			left++
			fmt.Fprintf(out, "no state %s %s: mastered, not by its own answer (credited or placed): its first event initialises it\n", sid, id)
		}
	}
	fmt.Fprintf(out, "fsrs_seed: %d topics to seed, %d written, %d mastered rows left without state\n", total, wrote, left)
	return nil
}
