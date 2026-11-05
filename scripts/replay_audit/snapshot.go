package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/chuma-beep/mathua/internal/clock"
)

// surfaceSnapshot is what every surface shows at the start of one session
// (FSRS Phase 2, gate 4: with MATHUA_FSRS=shadow each must be byte-identical
// to off). The fsrs_* fields of the progress view are the stored columns
// themselves, which shadow writes; they are left out here and compared in
// the -dump instead.
type surfaceSnapshot struct {
	Session    int             `json:"session"`
	Now        string          `json:"now"`
	Due        dueReport       `json:"due"`         // the due set, the count, the course marks, Up next's review card, the review queue, the mixed session's reviews
	UpNext     json.RawMessage `json:"up_next"`     // the Study queue with its reasons and badges
	Courses    json.RawMessage `json:"courses"`     // each course's progress and pacing ("N topics/week")
	PlanOrder  []string        `json:"plan_order"`  // the order new topics are offered in
	Domains    json.RawMessage `json:"domains"`     // the Domains page: per-topic states
	Gate       json.RawMessage `json:"gate"`        // the readiness payload of the head of the queue and of each due topic
	QuizDue    bool            `json:"quiz_due"`    // the mastery-check gate
	QuizPicked []string        `json:"quiz_picked"` // the mastery check's topic picks
	XPTotal    int             `json:"xp_total"`
	XPToday    int             `json:"xp_today"`
	Progress   json.RawMessage `json:"progress"` // the progress view (/api/progress) without the fsrs_* fields
	Pace       json.RawMessage `json:"pace"`
}

func rawJSON(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		b, _ = json.Marshal(map[string]string{"error": err.Error()})
	}
	return b
}

// writeSurfaces writes the session's snapshot as one JSON line to -surfaces.
func (r *replay) writeSurfaces(n int, due dueReport) {
	if r.surfw == nil {
		return
	}
	snap := surfaceSnapshot{Session: n, Now: clock.Now().UTC().Format("2006-01-02T15:04:05Z"), Due: due}
	rec, err := r.eng.RecommendNext(r.sid, nil)
	if err == nil {
		snap.UpNext = rawJSON(rec)
	} else {
		snap.UpNext = rawJSON(err.Error())
	}
	cat, _ := r.eng.CourseCatalog(r.sid)
	snap.Courses = rawJSON(cat)
	snap.PlanOrder = r.eng.PacedPlanOrder(r.sid)
	doms, _ := r.eng.CurriculumDomains(r.sid)
	snap.Domains = rawJSON(doms)
	gateTopics := append([]string(nil), due.Due...)
	for _, rc := range recsOf(rec) {
		if rc.ConceptID != "" {
			gateTopics = append(gateTopics, rc.ConceptID)
		}
	}
	gate := map[string]interface{}{}
	for _, id := range gateTopics {
		if el, err := r.eng.EligibilityFor(r.sid, id); err == nil {
			gate[id] = el
		}
	}
	snap.Gate = rawJSON(gate)
	snap.QuizDue, _ = r.eng.QuizDue(r.sid)
	if picked, _, ok := r.eng.CourseQuizConcepts(r.sid); ok {
		for _, c := range picked {
			snap.QuizPicked = append(snap.QuizPicked, c.ID)
		}
	}
	snap.XPTotal, snap.XPToday, _ = r.repo.GetXP(r.sid)
	if pv, err := r.eng.ProgressWithMasteryPct(r.sid); err == nil {
		snap.Progress = withoutFSRS(rawJSON(pv))
	}
	pace, _ := r.eng.PaceFor(r.sid, 30)
	snap.Pace = rawJSON(pace)
	b, _ := json.Marshal(snap)
	fmt.Fprintln(r.surfw, string(b))
}

// withoutFSRS drops every "fsrs_*" key from a JSON object of objects (the
// progress view), keeping the rest byte-stable (encoding/json sorts map
// keys).
func withoutFSRS(raw json.RawMessage) json.RawMessage {
	var m map[string]map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return raw
	}
	for _, row := range m {
		for k := range row {
			if strings.HasPrefix(k, "fsrs_") {
				delete(row, k)
			}
		}
	}
	return rawJSON(m)
}

// dumpDB writes every table of the database as sorted text, one row per
// line: "table|col=value|...". With withFSRS false the fsrs_* columns and the
// review_log table are left out, so an off run and a shadow run of the same
// replay can be compared for "only the FSRS columns and review_log differ".
func dumpDB(db *sql.DB, path string, withFSRS bool) error {
	rows, err := db.Query(`select name from sqlite_master where type = 'table' and name not like 'sqlite_%' order by name`)
	if err != nil {
		return err
	}
	var tables []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			rows.Close()
			return err
		}
		tables = append(tables, t)
	}
	rows.Close()
	var out []string
	for _, t := range tables {
		if !withFSRS && t == "review_log" {
			continue
		}
		lines, err := dumpTable(db, t, withFSRS)
		if err != nil {
			return fmt.Errorf("%s: %w", t, err)
		}
		out = append(out, lines...)
	}
	return os.WriteFile(path, []byte(strings.Join(out, "\n")+"\n"), 0o644)
}

func dumpTable(db *sql.DB, table string, withFSRS bool) ([]string, error) {
	rows, err := db.Query(`select * from "` + table + `"`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var lines []string
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		parts := []string{table}
		for i, c := range cols {
			if !withFSRS && strings.HasPrefix(c, "fsrs_") {
				continue
			}
			if table == "schema_migrations" && c == "applied_at" {
				continue // when the copy was opened: bookkeeping, not replay output
			}
			v := vals[i]
			if b, ok := v.([]byte); ok {
				v = string(b)
			}
			parts = append(parts, fmt.Sprintf("%s=%v", c, v))
		}
		lines = append(lines, strings.Join(parts, "|"))
	}
	sort.Strings(lines)
	return lines, rows.Err()
}
