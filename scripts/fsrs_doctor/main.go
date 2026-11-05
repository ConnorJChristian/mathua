// fsrs_doctor checks the FSRS invariants of a Mathua database (FSRS plan
// 6.1) and changes nothing. It opens the SQLite file read-only and reports:
//
//   - orphan: a progress row or log row for a topic the graph no longer has
//     (kept, never deleted: plan 3.7);
//   - invalid: FSRS state with S <= 0, NaN or infinite, D outside [1, 10],
//     a state name that is not none/review/relearning, or a state with no
//     stability (and the reverse), or a review_log row (other than a seed)
//     whose rating is not 1..4;
//   - surfaced: a topic the review session would serve that the due rule
//     excludes, or one the rule says is due that the session would not
//     serve (the Phase 0 disagreement, read without writing a session);
//   - unlogged: a progress row with FSRS state and no review_log rows, or
//     log rows for a row with no state;
//   - replay: a stored state that differs from replaying its log
//     (review.ReplayLast: held rows are logged and skipped), or an
//     fsrs_last_review that is not the last applied row's time, the
//     event-sourcing check;
//   - error: an fsrs_errors row, an event the app dropped (not computed
//     or not written) while committing its answer without it;
//   - missing: an answer that should have written an event and has no
//     review_log row (missingEvents), the dropped event seen from the
//     attempts.
//
// Usage: MATHUA_ROOT=<install> go run ./scripts/fsrs_doctor -db <path>
// [-student <id>] [-json]. Exit 1 on any finding, 2 on a usage error.
// The daily QA run calls it; the metrics exporter counts its findings.
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/engine"
	"github.com/chuma-beep/mathua/internal/fsrs"
	"github.com/chuma-beep/mathua/internal/generator/all"
	"github.com/chuma-beep/mathua/internal/grader"
	"github.com/chuma-beep/mathua/internal/modules"
	"github.com/chuma-beep/mathua/internal/planning"
	"github.com/chuma-beep/mathua/internal/review"
	"github.com/chuma-beep/mathua/internal/storage"

	_ "github.com/mattn/go-sqlite3"
)

// Finding is one invariant violation.
type Finding struct {
	Check   string `json:"check"`
	Student string `json:"student"`
	Concept string `json:"concept,omitempty"`
	Detail  string `json:"detail"`
}

func main() {
	dbPath := flag.String("db", "", "SQLite database to check (opened read-only)")
	student := flag.String("student", "", "one learner's id (default: every learner)")
	asJSON := flag.Bool("json", false, "print the findings as JSON")
	flag.Parse()
	if *dbPath == "" {
		fmt.Fprintln(os.Stderr, "usage: fsrs_doctor -db <path> [-student <id>] [-json]")
		os.Exit(2)
	}
	findings, counts, err := Run(*dbPath, *student)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fsrs_doctor:", err)
		os.Exit(2)
	}
	if *asJSON {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{"findings": findings, "counts": counts})
	} else {
		for _, f := range findings {
			fmt.Printf("%-9s %s %s: %s\n", f.Check, f.Student, f.Concept, f.Detail)
		}
		fmt.Printf("fsrs_doctor: %d learners, %d progress rows with FSRS state, %d log rows, %d fsrs_errors rows, %d findings\n",
			counts["learners"], counts["with_state"], counts["log_rows"], counts["fsrs_errors"], len(findings))
	}
	if len(findings) > 0 {
		os.Exit(1)
	}
}

// Run opens the database read-only, checks every learner (or one) and
// returns the findings and some counts.
func Run(dbPath, onlyStudent string) ([]Finding, map[string]int, error) {
	root := os.Getenv("MATHUA_ROOT")
	if root == "" {
		return nil, nil, fmt.Errorf("MATHUA_ROOT must name the install (its data/ and modules)")
	}
	mods, err := modules.Default()
	if err != nil {
		return nil, nil, fmt.Errorf("modules: %w", err)
	}
	dag, err := concepts.LoadDirs(filepath.Join(root, "data", "concepts"), mods.ConceptDirs()...)
	if err != nil {
		return nil, nil, fmt.Errorf("concepts: %w", err)
	}
	// Read-only: the file is copied (VACUUM INTO, which reads a consistent
	// snapshot through the WAL) and the copy is what the store opens, so
	// the store's migrations and the engine's session bookkeeping touch
	// nothing of the original.
	copyPath, err := snapshotCopy(dbPath)
	if err != nil {
		return nil, nil, err
	}
	defer os.RemoveAll(filepath.Dir(copyPath))
	store, err := storage.NewSQLiteStore(copyPath)
	if err != nil {
		return nil, nil, err
	}
	defer store.Close()
	db, err := sql.Open("sqlite3", copyPath)
	if err != nil {
		return nil, nil, err
	}
	defer db.Close()
	planner, err := planning.Load(filepath.Join(root, "data", "courses.json"), dag, mods.CourseFiles()...)
	if err != nil {
		return nil, nil, fmt.Errorf("courses: %w", err)
	}
	eng := engine.New(store, dag, all.Registry(), nil, planner)
	eng.SetModules(mods)
	defer eng.Close()

	students, err := listStudents(db, onlyStudent)
	if err != nil {
		return nil, nil, err
	}
	counts := map[string]int{"learners": len(students)}
	var findings []Finding
	now := time.Now().UTC()
	params := review.DefaultParams()
	rules, _ := review.CurrentRules()
	for _, sid := range students {
		errs, err := store.FSRSErrors(sid)
		if err != nil {
			return nil, nil, fmt.Errorf("fsrs_errors %s: %w", sid, err)
		}
		counts["fsrs_errors"] += len(errs)
		for _, e := range errs {
			findings = append(findings, Finding{"error", sid, e.ConceptID, fmt.Sprintf("fsrs_errors row %d at %s: event not %s (attempt %d): %s",
				e.ID, e.At.UTC().Format(time.RFC3339), e.Stage, e.AttemptID, e.Error)})
		}
		progress, err := store.GetAllProgress(sid)
		if err != nil {
			return nil, nil, fmt.Errorf("progress %s: %w", sid, err)
		}
		log, err := store.ReviewLogFor(sid, "")
		if err != nil {
			return nil, nil, fmt.Errorf("review_log %s: %w", sid, err)
		}
		counts["log_rows"] += len(log)
		byTopic := map[string][]storage.ReviewLogEntry{}
		for _, e := range log {
			byTopic[e.ConceptID] = append(byTopic[e.ConceptID], e)
			if dag.Concept(e.ConceptID) == nil {
				findings = append(findings, Finding{"orphan", sid, e.ConceptID, "review_log row for a topic the graph no longer has"})
			}
			// A review row's rating must be Again..Easy; a seed row sets
			// the state outright and carries no rating of its own.
			if e.Held != "" && e.Held != review.HeldDailyCap && e.Held != review.HeldLapseRHigh {
				findings = append(findings, Finding{"invalid", sid, e.ConceptID, fmt.Sprintf("review_log row %d: held %q is not a known reason", e.ID, e.Held)})
			}
			if e.Source != "seed" {
				if err := review.CheckRating(fsrs.Rating(e.Rating)); err != nil {
					findings = append(findings, Finding{"invalid", sid, e.ConceptID, fmt.Sprintf("review_log row %d (%s): %v", e.ID, e.Source, err)})
				}
			}
		}
		for cid, p := range progress {
			m := memoryOf(p)
			hasState := p.FSRSStability != nil || !m.IsZero()
			if hasState {
				counts["with_state"]++
			}
			if dag.Concept(cid) == nil && hasState {
				findings = append(findings, Finding{"orphan", sid, cid, "FSRS state for a topic the graph no longer has (kept)"})
			}
			if d := invalid(p); d != "" {
				findings = append(findings, Finding{"invalid", sid, cid, d})
			}
			entries := byTopic[cid]
			if hasState && len(entries) == 0 && p.FSRSSeed != "anki" {
				findings = append(findings, Finding{"unlogged", sid, cid, "FSRS state with no review_log rows"})
			}
			if !hasState && len(entries) > 0 {
				findings = append(findings, Finding{"unlogged", sid, cid, fmt.Sprintf("%d review_log rows for a row with no FSRS state", len(entries))})
			}
			if hasState && len(entries) > 0 {
				events := make([]review.Event, len(entries))
				for i, e := range entries {
					events[i] = review.Event{At: e.At, Rating: fsrs.Rating(e.Rating), ElapsedDays: e.ElapsedDays, Source: e.Source, SAfter: e.SAfter, DAfter: e.DAfter, Held: e.Held}
				}
				replayed, last, _ := review.ReplayLast(params, events)
				if d := review.Diff(m, replayed); d != "" {
					findings = append(findings, Finding{"replay", sid, cid, d})
				}
				// fsrs_last_review is the last event that changed the state:
				// a held event (daily cap, a practice miss at high R) is
				// logged without moving it.
				if p.FSRSLastReview == nil || !p.FSRSLastReview.Truncate(time.Second).Equal(last.Truncate(time.Second)) {
					findings = append(findings, Finding{"replay", sid, cid, fmt.Sprintf("fsrs_last_review %v is not the last applied log row's at %v", p.FSRSLastReview, last)})
				}
			}
		}
		for cid := range byTopic {
			if progress[cid] == nil {
				findings = append(findings, Finding{"unlogged", sid, cid, "review_log rows with no progress row"})
			}
		}
		atts, err := store.GetAttemptsForStudent(sid)
		if err != nil {
			return nil, nil, fmt.Errorf("attempts %s: %w", sid, err)
		}
		for _, a := range missingEvents(atts, byTopic, rules) {
			findings = append(findings, Finding{"missing", sid, a.ConceptID, fmt.Sprintf("%s attempt %d at %s has no review_log row in its second (a dropped event; see the error check)",
				a.Source, a.ID, a.Timestamp.UTC().Format(time.RFC3339))})
		}
		// Due rule versus what the session would surface, read-only.
		due := eng.DueReviewIDs(sid, "", progress, now)
		dueSet := map[string]bool{}
		for _, id := range due {
			dueSet[id] = true
		}
		served := eng.ReviewCandidates(sid, progress, now)
		servedSet := map[string]bool{}
		for _, id := range served {
			servedSet[id] = true
			if !dueSet[id] {
				findings = append(findings, Finding{"surfaced", sid, id, "the review session would serve it; the due rule excludes it"})
			}
		}
		for _, id := range due {
			if !servedSet[id] {
				findings = append(findings, Finding{"surfaced", sid, id, "due by the rule; the review session would not serve it"})
			}
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.Check != b.Check {
			return a.Check < b.Check
		}
		if a.Student != b.Student {
			return a.Student < b.Student
		}
		return a.Concept < b.Concept
	})
	return findings, counts, nil
}

// needsEvent reports whether an attempt, under the rules in force, must
// have written a review_log row once its topic is in the memory model (the
// plan 3.3 table): a review-session answer always; a quiz answer when the
// rules count both quiz kinds (the attempt does not say whether it was a
// mastery check or a course review test, so with only one counting it is
// left out); never a self-graded recall card; a proof or judged item only
// with judged=on (an ungraded one writes no attempt). Practice is left
// out: whether a miss was a lapse depends on the status at that moment.
func needsEvent(a storage.AttemptEntry, rules review.Rules) bool {
	if strings.HasPrefix(a.Expected, "recall:") {
		return false
	}
	if grader.IsProofAnswer(a.Expected) && !rules.Judged {
		return false
	}
	switch a.Source {
	case "review":
		return true
	case "quiz":
		return rules.MasteryCheck && rules.CourseTests
	}
	return false
}

// missingEvents is the attempts that needed an event (needsEvent), at or
// after their topic's first review_log row, with no row for that topic in
// the same second (attempts are stored to the second; an event's at is
// its attempt's time). The FSRS Phase 2 reviewer's proposed check: a
// dropped shadow write leaves the state consistent with its log, so only
// the attempts show it.
func missingEvents(atts []storage.AttemptEntry, byTopic map[string][]storage.ReviewLogEntry, rules review.Rules) []storage.AttemptEntry {
	seconds := map[string]map[int64]bool{}
	for cid, entries := range byTopic {
		set := map[int64]bool{}
		for _, e := range entries {
			set[e.At.Unix()] = true
		}
		seconds[cid] = set
	}
	var out []storage.AttemptEntry
	for _, a := range atts {
		entries := byTopic[a.ConceptID]
		if len(entries) == 0 || !needsEvent(a, rules) || a.Timestamp.Unix() < entries[0].At.Unix() {
			continue
		}
		if !seconds[a.ConceptID][a.Timestamp.Unix()] {
			out = append(out, a)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func memoryOf(p *storage.ConceptProgress) review.Memory {
	m := review.Memory{Reps: p.FSRSReps, Lapses: p.FSRSLapses, Phase: review.Phase(p.FSRSState)}
	if p.FSRSStability != nil {
		m.Stability = *p.FSRSStability
	}
	if p.FSRSDifficulty != nil {
		m.Difficulty = *p.FSRSDifficulty
	}
	return m
}

func invalid(p *storage.ConceptProgress) string {
	switch review.Phase(p.FSRSState) {
	case review.PhaseNone, "":
		if p.FSRSStability != nil || p.FSRSDifficulty != nil || p.FSRSLastReview != nil || p.FSRSReps != 0 || p.FSRSLapses != 0 {
			return "fsrs_state none with stability, difficulty, last review or counts set"
		}
		return ""
	case review.PhaseReview, review.PhaseRelearning:
	default:
		return fmt.Sprintf("fsrs_state %q is not none, review or relearning", p.FSRSState)
	}
	if p.FSRSStability == nil || p.FSRSDifficulty == nil {
		return "state " + p.FSRSState + " with NULL stability or difficulty"
	}
	if s := *p.FSRSStability; !(s > 0) || math.IsNaN(s) || math.IsInf(s, 0) {
		return fmt.Sprintf("stability %v", s)
	}
	if d := *p.FSRSDifficulty; d < 1 || d > 10 || math.IsNaN(d) {
		return fmt.Sprintf("difficulty %v outside [1, 10]", d)
	}
	if p.FSRSLastReview == nil || p.FSRSReps < 1 || p.FSRSLapses < 0 || p.FSRSLapses > p.FSRSReps {
		return fmt.Sprintf("last review %v, reps %d, lapses %d", p.FSRSLastReview, p.FSRSReps, p.FSRSLapses)
	}
	return ""
}

// snapshotCopy copies the database into a temporary directory with VACUUM
// INTO, from a read-only connection.
func snapshotCopy(dbPath string) (string, error) {
	if _, err := os.Stat(dbPath); err != nil {
		return "", err
	}
	src, err := sql.Open("sqlite3", "file:"+dbPath+"?mode=ro")
	if err != nil {
		return "", err
	}
	defer src.Close()
	dir, err := os.MkdirTemp("", "fsrs_doctor")
	if err != nil {
		return "", err
	}
	dst := filepath.Join(dir, "copy.db")
	if _, err := src.Exec("VACUUM INTO ?", dst); err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("copy database: %w", err)
	}
	return dst, nil
}

func listStudents(db *sql.DB, only string) ([]string, error) {
	if only != "" {
		return []string{only}, nil
	}
	rows, err := db.Query(`SELECT id FROM students ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
