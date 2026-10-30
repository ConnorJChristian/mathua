// Command replay_audit replays study sessions for one learner against a copy
// of the app's database and reports every topic the engine served and why,
// classified as appropriate or not for a learner following courses.
//
// It drives the engine the way the app does (Up next, the review session,
// a lesson on the head of the queue, the mixed /session feed, the mastery
// check and the course placement test), answering as a learner who knows
// most of the material would, and ages the database by one day between
// sessions so reviews come due. Nothing is written anywhere but the copy.
//
//	MATHUA_ROOT=/path/to/install go run ./scripts/replay_audit -db snapshot.db -student ID -sessions 30
//
// -db is never written: the tool copies it (without any -wal or -shm beside
// it) into a fresh temporary directory and replays against the copy, so a
// stale write-ahead log from an earlier run can never be applied to it.
//
// Flags: -placement also replays the course placement tests and the
// onboarding diagnostic for the followed courses on a throwaway copy; -json
// writes every serve as a JSON line to the named file; -seed fixes the
// learner's luck. MATHUA_REVIEW_FLOOR_DOMAINS applies as it does in the server.
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/diagnostic"
	"github.com/chuma-beep/mathua/internal/engine"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/generator/all"
	"github.com/chuma-beep/mathua/internal/mastery"
	"github.com/chuma-beep/mathua/internal/modules"
	"github.com/chuma-beep/mathua/internal/planning"
	"github.com/chuma-beep/mathua/internal/pytrack"
	"github.com/chuma-beep/mathua/internal/scheduler"
	"github.com/chuma-beep/mathua/internal/storage"
)

// Serve is one topic the engine put in front of the learner.
type Serve struct {
	Session int      `json:"session"`
	Surface string   `json:"surface"` // upnext, review, study, feed, mastery_check, remedial, placement, onboarding
	Concept string   `json:"concept"`
	Label   string   `json:"label"`
	Domain  string   `json:"domain"`
	Why     string   `json:"why"` // review due / plan order / pin / remedial / weakness / placement / new
	Flags   []string `json:"flags,omitempty"`
}

// floorDomains is the grade-school floor a graduate learner is past.
var floorDomains = map[string]bool{"arithmetic": true, "fractions": true, "prealgebra": true, "geometry": true}

type replay struct {
	eng      *engine.Engine
	repo     *storage.SQLiteStore
	db       *sql.DB
	dag      *concepts.DAG
	reg      *generator.Registry
	planner  *planning.Planner
	bank     *pytrack.Bank
	sid      string
	rng      *rand.Rand
	own      map[string]bool // followed courses' own topics
	pinned   map[string]bool
	afterDue map[string]map[string]bool // course -> after-due topics
	dueOf    map[string][]string        // course -> plan topics due by the date, in plan order
	dueSome  map[string]bool            // topics some followed course needs by its date (or has no date)
	depth    map[string]int             // links below the nearest followed-course topic
	path     map[string]bool            // active path (every followed course's path)
	missed   map[string]bool            // topics missed on the last mastery check
	serves   []Serve
	logw     io.Writer
}

func main() {
	dbPath := flag.String("db", "", "database snapshot to replay (read only: the replay runs on a fresh copy)")
	sid := flag.String("student", "", "student id")
	n := flag.Int("sessions", 30, "sessions to replay, one per day")
	seed := flag.Int64("seed", 1, "random seed")
	placement := flag.Bool("placement", true, "also replay placement and the onboarding diagnostic")
	jsonOut := flag.String("json", "", "write every serve as a JSON line here")
	flag.Parse()
	if *dbPath == "" || *sid == "" {
		fmt.Fprintln(os.Stderr, "usage: replay_audit -db copy.db -student ID [-sessions N]")
		os.Exit(2)
	}
	work, err := freshCopy(*dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer os.RemoveAll(filepath.Dir(work))
	r, err := open(work, *sid, *seed)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *jsonOut != "" {
		f, err := os.Create(*jsonOut)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer f.Close()
		r.logw = f
	}
	if *placement {
		r.replayPlacement()
	}
	for s := 1; s <= *n; s++ {
		r.session(s)
		if err := r.ageOneDay(); err != nil {
			fmt.Fprintln(os.Stderr, "age:", err)
			os.Exit(1)
		}
	}
	r.report()
}

// freshCopy copies a snapshot into a new temporary directory. A snapshot is
// expected to be checkpointed (no -wal beside it, or an empty one); one with a
// non-empty -wal is refused
// rather than copied without its log.
func freshCopy(src string) (string, error) {
	if fi, err := os.Stat(src + "-wal"); err == nil && fi.Size() > 0 {
		return "", fmt.Errorf("%s has a -wal beside it: checkpoint it first", src)
	}
	dir, err := os.MkdirTemp("", "replay_audit-")
	if err != nil {
		return "", err
	}
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()
	dst := filepath.Join(dir, "replay.db")
	out, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return "", err
	}
	return dst, out.Close()
}

func open(dbPath, sid string, seed int64) (*replay, error) {
	root := os.Getenv("MATHUA_ROOT")
	if root == "" {
		return nil, fmt.Errorf("MATHUA_ROOT must name the install (its data/ and modules)")
	}
	mods, err := modules.Default()
	if err != nil {
		return nil, fmt.Errorf("modules: %w", err)
	}
	dag, err := concepts.LoadDirs(filepath.Join(root, "data", "concepts"), mods.ConceptDirs()...)
	if err != nil {
		return nil, fmt.Errorf("concepts: %w", err)
	}
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	reg := all.Registry()
	planner, err := planning.Load(filepath.Join(root, "data", "courses.json"), dag, mods.CourseFiles()...)
	if err != nil {
		return nil, fmt.Errorf("courses: %w", err)
	}
	if ds, err := planning.LoadDestinations(filepath.Join(root, "data", "destinations.json"), mods.DestinationFiles()...); err == nil {
		planner.SetDestinations(ds)
	}
	eng := engine.New(store, dag, reg, nil, planner)
	eng.SetModules(mods)
	tr, err := pytrack.Default()
	if err != nil {
		return nil, err
	}
	st, err := store.GetStudent(sid)
	if err != nil || st == nil {
		return nil, fmt.Errorf("student %s: %v", sid, err)
	}
	r := &replay{eng: eng, repo: store, db: db, dag: dag, reg: reg, planner: planner, bank: tr.Bank, sid: sid,
		rng: rand.New(rand.NewSource(seed)), own: map[string]bool{}, pinned: map[string]bool{},
		afterDue: map[string]map[string]bool{}, dueOf: map[string][]string{}, dueSome: map[string]bool{},
		depth: map[string]int{}, missed: map[string]bool{}, logw: io.Discard}
	for _, id := range engine.ActiveCourses(st.CourseID) {
		c := planner.Course(id)
		if c == nil {
			continue
		}
		for _, t := range c.Plan {
			r.own[t] = true
		}
		for _, t := range c.Targets {
			r.own[t] = true
		}
		r.afterDue[id] = c.AfterDueTopics()
		for _, t := range c.Plan {
			if !r.afterDue[id][t] {
				r.dueOf[id] = append(r.dueOf[id], t)
				r.dueSome[t] = true
			}
		}
		// A course with no after-due module needs its whole path.
		if len(r.afterDue[id]) == 0 {
			if p, err := planner.PathForCourse(id); err == nil {
				for _, cc := range p.Concepts {
					r.dueSome[cc.ID] = true
				}
			}
		}
	}
	// Depth below the followed courses' own topics, for placement.
	queue := []string{}
	for t := range r.own {
		r.depth[t] = 0
		queue = append(queue, t)
	}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		for _, p := range dag.PrereqsOf(id) {
			if _, ok := r.depth[p.ID]; !ok {
				r.depth[p.ID] = r.depth[id] + 1
				queue = append(queue, p.ID)
			}
		}
	}
	for _, p := range eng.Pins(sid) {
		r.pinned[p.Concept] = true
	}
	// The engine builds the active path lazily; a recommendation does it.
	eng.RecommendNext(sid, nil)
	r.path = eng.ActivePath(sid)
	fmt.Printf("learner %s follows %s; %d own topics, %d on path, %d pinned, floor=%q\n",
		st.Name, st.CourseID, len(r.own), len(r.path), len(r.pinned), os.Getenv("MATHUA_REVIEW_FLOOR_DOMAINS"))
	return r, nil
}

// ---- one session ----

func (r *replay) session(n int) {
	seen := map[string]int{}
	note := func(s Serve) {
		s.Session = n
		if c := r.dag.Concept(s.Concept); c != nil {
			s.Label, s.Domain = c.Label, c.Domain
		}
		s.Flags = r.classify(s, seen)
		seen[s.Surface+"|"+s.Concept]++
		r.serves = append(r.serves, s)
		b, _ := json.Marshal(s)
		fmt.Fprintln(r.logw, string(b))
	}

	// Up next: the Study queue.
	rec, err := r.eng.RecommendNext(r.sid, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "recommend:", err)
		return
	}
	var head *scheduler.Recommendation
	for i, rc := range recsOf(rec) {
		if rc.ConceptID == "" {
			continue
		}
		note(Serve{Surface: "upnext", Concept: rc.ConceptID, Why: whyOf(rc)})
		if head == nil && rc.Kind != scheduler.KindReview && rc.Kind != scheduler.KindMasteryCheck {
			head = &recsOf(rec)[i]
		}
	}

	// The review session: one session, as /review runs it.
	if rs, err := r.repo.CreateSession(r.sid); err == nil {
		for i := 0; i < 8; i++ {
			q, err := r.eng.NextReviewQuestion(rs.ID, r.sid)
			if err != nil || q == nil {
				break
			}
			note(Serve{Surface: "review", Concept: q.ConceptID, Why: whyQ(q, "review due")})
			r.answerSession(rs.ID, q, r.rng.Float64() < 0.85)
		}
	}

	// A lesson on the head of the queue, as the Learn page serves it.
	if head != nil {
		cid := head.ConceptID
		note(Serve{Surface: "study", Concept: cid, Why: whyOf(*head)})
		ctx := generator.GeneratorContext{Difficulty: 0.4, Seed: r.rng.Int63(), History: r.eng.LearnerHistory(r.sid, cid)}
		probs, err := r.reg.BatchGenerateContext(cid, 3, ctx)
		if err == nil {
			anchors := map[string]engine.Anchor{}
			for _, p := range probs {
				anchors[p.Question] = engine.Anchor{Answer: p.Answer, Explanation: p.Explanation, Difficulty: p.Difficulty}
			}
			r.eng.SetStudyAnchorBatch(r.sid, cid, anchors)
			misses := 0
			for _, p := range probs {
				ok := r.rng.Float64() < 0.8
				if !ok {
					// Two misses in a row in Learn queue remediation too.
					if misses++; misses >= 2 {
						r.missed[cid] = true
					}
				} else {
					misses = 0
				}
				r.eng.SubmitStudyAnswer(r.sid, cid, r.answerFor(p.Answer, ok), 20, p.Question)
			}
		}
	}

	// The mixed feed (/session), with enough misses to trigger a halt.
	sess, err := r.repo.CreateSession(r.sid)
	if err == nil {
		missed := ""
		misses := 0
		for i := 0; i < 5; i++ {
			q, err := r.eng.NextQuestion(sess.ID, r.sid)
			if err != nil || q == nil {
				break
			}
			why := "plan order"
			if q.IsReview {
				why = "review due"
			}
			surface := "feed"
			if misses >= 2 && q.ConceptID != missed {
				surface, why = "remedial", "remedial for "+missed
			}
			if r.pinned[q.ConceptID] {
				why = "pin"
			}
			if surface == "feed" {
				why = whyQ(q, why)
			}
			note(Serve{Surface: surface, Concept: q.ConceptID, Why: why})
			ok := r.rng.Float64() < 0.6
			r.answerSession(sess.ID, q, ok)
			if ok {
				misses, missed = 0, ""
			} else if q.ConceptID == missed || missed == "" {
				misses++
				missed = q.ConceptID
			}
		}
	}

	// The mastery check, when due.
	if due, _ := r.eng.QuizDue(r.sid); due {
		picked, _, ok := r.eng.CourseQuizConcepts(r.sid)
		if ok {
			for _, c := range picked {
				note(Serve{Surface: "mastery_check", Concept: c.ID, Why: "mastery check"})
				ctx := generator.GeneratorContext{Difficulty: 0.8, Seed: r.rng.Int63(), ObjectiveOnly: true, NoCards: true, NoProofs: true, History: r.eng.LearnerHistory(r.sid, c.ID)}
				p, err := r.reg.GenerateContext(c.ID, ctx)
				if err != nil || p.Question == "" {
					continue
				}
				ok := r.rng.Float64() < 0.75
				if !ok {
					r.missed[c.ID] = true
				}
				d := 0.8
				r.eng.SubmitQuizAnswer(r.sid, c.ID, r.answerFor(p.Answer, ok), p.Answer, 20, p.Question, &d)
			}
			r.eng.RecordQuizCompletion(r.sid)
			for _, id := range r.eng.QuizRemedial(r.sid) {
				note(Serve{Surface: "remedial", Concept: id, Why: "missed on mastery check"})
			}
		}
	}
}

func recsOf(res scheduler.RecommendationResponse) []scheduler.Recommendation {
	var out []scheduler.Recommendation
	if res.Primary != nil {
		out = append(out, *res.Primary)
	}
	return append(out, res.Alternatives...)
}

// whyQ is the engine's reason for a served question when it gives one.
func whyQ(q *engine.Question, fallback string) string {
	if q != nil && q.Reason != "" {
		return q.Reason
	}
	return fallback
}

func whyOf(rc scheduler.Recommendation) string {
	if rc.Why != "" {
		return rc.Why
	}
	switch {
	case rc.Kind == scheduler.KindReview:
		return "review due"
	case rc.Pinned && rc.PinnedFor != "":
		return "pin (prerequisite of " + rc.PinnedFor + ")"
	case rc.Pinned:
		return "pin"
	case rc.Badge == "From your test":
		return "remedial"
	case rc.Kind == scheduler.KindPractice:
		return "weakness"
	case rc.Badge == "Needed next" || rc.Badge == "Jump in":
		return "bridge"
	default:
		return "plan order"
	}
}

func (r *replay) answerSession(sessionID string, q *engine.Question, correct bool) {
	key, ok := r.eng.ExpectedAnswerForReplay(sessionID)
	if !ok {
		return
	}
	r.eng.SubmitAnswer(sessionID, r.sid, q.AttemptID, r.answerFor(key, correct), 15)
}

// answerFor is what a learner types for an answer key, right or wrong.
func (r *replay) answerFor(key string, correct bool) string {
	switch {
	case strings.HasPrefix(key, "recall:"):
		if correct {
			return "right"
		}
		return "wrong"
	case strings.HasPrefix(key, "mc:"):
		if correct {
			return strings.TrimPrefix(key, "mc:")
		}
		return "none of these"
	case strings.HasPrefix(key, "out:"):
		if correct {
			return strings.TrimPrefix(key, "out:")
		}
		return "error"
	case strings.HasPrefix(key, "math:"):
		it := r.bank.Item(strings.TrimPrefix(key, "math:"))
		if it == nil || !correct {
			return "0"
		}
		var parts []string
		for _, p := range it.Parts {
			parts = append(parts, p.Answer)
		}
		return strings.Join(parts, "\n")
	case strings.HasPrefix(key, "term:"):
		it := r.bank.Item(strings.TrimPrefix(key, "term:"))
		if it == nil || !correct {
			return "unknown"
		}
		return it.Correct
	case strings.HasPrefix(key, "code:"), strings.HasPrefix(key, "proof:"):
		return "" // not gradable here: ungraded, nothing recorded
	}
	if correct {
		return key
	}
	return "-9999"
}

// ---- placement ----

func (r *replay) replayPlacement() {
	st, _ := r.repo.GetStudent(r.sid)
	if st == nil {
		return
	}
	note := func(surface, cid, why string) {
		s := Serve{Session: 0, Surface: surface, Concept: cid, Why: why}
		if c := r.dag.Concept(cid); c != nil {
			s.Label, s.Domain = c.Label, c.Domain
		}
		s.Flags = r.classify(s, map[string]int{})
		r.serves = append(r.serves, s)
		b, _ := json.Marshal(s)
		fmt.Fprintln(r.logw, string(b))
	}
	var all []string
	for _, id := range engine.ActiveCourses(st.CourseID) {
		c := r.planner.Course(id)
		if c == nil {
			continue
		}
		if len(c.Plan) > 0 {
			all = append(all, c.Plan...)
		} else {
			all = append(all, c.Targets...)
		}
		info, err := r.eng.CourseTests(r.sid, id)
		if err != nil || !info.Placement {
			continue
		}
		sess, q, err := r.eng.StartCourseDiagnostic(r.sid, id)
		if err != nil {
			continue
		}
		r.runDiagnostic(sess, q, "placement", id, note)
	}
	// The onboarding diagnostic, re-run for the followed courses (the
	// /onboard page's course placement without a single course id).
	if len(all) > 0 {
		sess, q, err := r.eng.StartGoalDiagnostic(r.sid, all)
		if err == nil {
			r.runDiagnostic(sess, q, "onboarding", "all courses", note)
		}
	}
}

func (r *replay) runDiagnostic(sess *diagnostic.Session, q *engine.Question, surface, scope string, note func(string, string, string)) {
	if q != nil {
		note(surface, q.ConceptID, "placement "+scope)
	}
	for i := 0; i < 80 && q != nil; i++ {
		c := r.dag.Concept(q.ConceptID)
		thresh := 30.0
		if c != nil {
			thresh = c.MasteryThreshold.AvgTimeSeconds
		}
		r.eng.SubmitDiagnosticAnswerTimed(sess, q.ConceptID, true, thresh/2, thresh)
		if r.eng.IsDiagnosticComplete(sess) {
			break
		}
		prob, cid, err := r.eng.NextDiagnosticQuestion(sess)
		if err != nil || prob == nil || cid == "" {
			break
		}
		q = &engine.Question{ConceptID: cid}
		note(surface, cid, "placement "+scope)
	}
}

// ---- classification ----

func (r *replay) classify(s Serve, seen map[string]int) []string {
	var flags []string
	c := r.dag.Concept(s.Concept)
	if c == nil {
		return []string{"unknown concept"}
	}
	progress, _ := r.repo.GetAllProgress(r.sid)
	p := progress[s.Concept]
	pinned := r.pinned[s.Concept]
	own := r.own[s.Concept]
	if floorDomains[c.Domain] && !pinned && !own {
		flags = append(flags, "floor")
	}
	if p != nil && p.Attempts == 0 && (p.PlacementSeeded || p.ImplicitCredit > 0) && !own && !pinned && s.Surface != "placement" && s.Surface != "onboarding" && !r.directPrereqOfMissed(s.Concept) {
		flags = append(flags, "credit-only off course")
	}
	if len(r.path) > 0 && !r.path[s.Concept] && !pinned && s.Surface != "review" && !strings.HasPrefix(s.Why, "review") && s.Surface != "placement" && s.Surface != "onboarding" {
		flags = append(flags, "off course")
	}
	// The same topic twice in one session is wrong for a review (a missed
	// review waits for the next session), a recommendation list or a test.
	// Practice on a topic being learned repeats it by design: mastery takes
	// a streak of answers.
	if seen[s.Surface+"|"+s.Concept] > 0 && (s.Surface == "review" || s.Surface == "upnext" || s.Surface == "mastery_check" || (s.Surface == "feed" && strings.HasPrefix(s.Why, "review"))) {
		flags = append(flags, "twice in session")
	}
	if (s.Surface == "placement" || s.Surface == "onboarding") && !pinned {
		if d, ok := r.depth[s.Concept]; ok && d > engine.CoursePlacementDepth {
			flags = append(flags, fmt.Sprintf("placement deeper than %d", engine.CoursePlacementDepth))
		}
	}
	if s.Surface == "remedial" && s.Why == "missed on mastery check" && !r.missed[s.Concept] && !r.directPrereqOfMissed(s.Concept) {
		flags = append(flags, "remedial not a direct prerequisite")
	}
	if s.Surface != "review" && !strings.HasPrefix(s.Why, "review") && s.Surface != "placement" && s.Surface != "onboarding" && s.Surface != "mastery_check" && s.Surface != "remedial" {
		if !r.prereqsMet(c, progress) {
			flags = append(flags, "prerequisites unmet")
		}
		for course, late := range r.afterDue {
			if late[s.Concept] && !r.dueSome[s.Concept] {
				for _, due := range r.dueOf[course] {
					if dp := progress[due]; (dp == nil || mastery.Status(dp.Status) != mastery.StatusMastered) && r.prereqsMet(r.dag.Concept(due), progress) {
						flags = append(flags, "after-due before due ("+due+")")
						break
					}
				}
			}
		}
	}
	if s.Surface == "remedial" && strings.HasPrefix(s.Why, "remedial for ") {
		missed := strings.TrimPrefix(s.Why, "remedial for ")
		mc := r.dag.Concept(missed)
		direct := false
		if mc != nil {
			for _, pid := range mc.Prerequisites {
				if pid == s.Concept {
					direct = true
				}
			}
			for _, pid := range mc.KeyPrerequisites {
				if pid == s.Concept {
					direct = true
				}
			}
		}
		if !direct {
			flags = append(flags, "remedial not a direct prerequisite")
		}
	}
	return flags
}

// directPrereqOfMissed reports a direct (or key) prerequisite of a topic the
// learner missed on the mastery check or twice in Learn: remediation sends
// them there on purpose, whatever credit it holds.
func (r *replay) directPrereqOfMissed(id string) bool {
	for m := range r.missed {
		if mc := r.dag.Concept(m); mc != nil {
			for _, pid := range append(append([]string(nil), mc.Prerequisites...), mc.KeyPrerequisites...) {
				if pid == id {
					return true
				}
			}
		}
	}
	return false
}

func (r *replay) prereqsMet(c *concepts.Concept, progress map[string]*storage.ConceptProgress) bool {
	if c == nil {
		return true
	}
	for _, pid := range c.Prerequisites {
		p := progress[pid]
		if p == nil {
			return false
		}
		if p.PlacementSeeded {
			continue
		}
		days := 999.0
		if p.LastReviewed != nil {
			days = time.Since(*p.LastReviewed).Hours() / 24
		}
		st := mastery.EffectiveStatus(mastery.Status(p.Status), days, mastery.DecayDays)
		if st != mastery.StatusMastered && st != "DECAYING" {
			return false
		}
	}
	return true
}

// ---- aging ----

// ageOneDay moves every timestamp of the learner's record one day into the
// past, so the next session is tomorrow: reviews fall due and mastery decays.
func (r *replay) ageOneDay() error {
	for _, stmt := range []string{
		`update concept_progress set
		   last_attempted = case when last_attempted is null or last_attempted='' then last_attempted else strftime('%Y-%m-%dT%H:%M:%SZ', last_attempted, '-1 day') end,
		   last_reviewed = case when last_reviewed is null or last_reviewed='' then last_reviewed else strftime('%Y-%m-%dT%H:%M:%SZ', last_reviewed, '-1 day') end,
		   next_review_due = case when next_review_due is null or next_review_due='' then next_review_due else strftime('%Y-%m-%dT%H:%M:%SZ', next_review_due, '-1 day') end,
		   mastered_at = case when mastered_at is null or mastered_at='' then mastered_at else strftime('%Y-%m-%dT%H:%M:%SZ', mastered_at, '-1 day') end
		 where student_id = ?`,
		`update attempts set timestamp = strftime('%Y-%m-%dT%H:%M:%SZ', timestamp, '-1 day') where student_id = ?`,
		`update quiz_completions set completed_at = strftime('%Y-%m-%d %H:%M:%S', completed_at, '-1 day') where student_id = ?`,
	} {
		if _, err := r.db.Exec(stmt, r.sid); err != nil {
			return err
		}
	}
	return nil
}

// ---- report ----

func (r *replay) report() {
	type key struct{ surface, flag string }
	counts := map[key]int{}
	total := map[string]int{}
	bad := map[string]int{}
	examples := map[string][]string{}
	for _, s := range r.serves {
		total[s.Surface]++
		if len(s.Flags) > 0 {
			bad[s.Surface]++
		}
		for _, f := range s.Flags {
			counts[key{s.Surface, f}]++
			if len(examples[f]) < 6 {
				examples[f] = append(examples[f], fmt.Sprintf("s%d %s %s (%s) %s", s.Session, s.Surface, s.Concept, s.Label, s.Why))
			}
		}
	}
	surfaces := make([]string, 0, len(total))
	for s := range total {
		surfaces = append(surfaces, s)
	}
	sort.Strings(surfaces)
	fmt.Println()
	fmt.Printf("%-14s %6s %6s %6s\n", "surface", "served", "ok", "bad")
	for _, s := range surfaces {
		fmt.Printf("%-14s %6d %6d %6d\n", s, total[s], total[s]-bad[s], bad[s])
	}
	fmt.Println()
	keys := make([]key, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].flag != keys[j].flag {
			return keys[i].flag < keys[j].flag
		}
		return keys[i].surface < keys[j].surface
	})
	for _, k := range keys {
		fmt.Printf("%-40s %-14s %d\n", k.flag, k.surface, counts[k])
	}
	fmt.Println()
	flags := make([]string, 0, len(examples))
	for f := range examples {
		flags = append(flags, f)
	}
	sort.Strings(flags)
	for _, f := range flags {
		fmt.Println(f + ":")
		for _, ex := range examples[f] {
			fmt.Println("  " + ex)
		}
	}
	// Topics served, by surface and why, for reading.
	fmt.Println()
	byWhy := map[string]int{}
	for _, s := range r.serves {
		byWhy[s.Surface+" / "+s.Why]++
	}
	ws := make([]string, 0, len(byWhy))
	for w := range byWhy {
		ws = append(ws, w)
	}
	sort.Strings(ws)
	for _, w := range ws {
		fmt.Printf("%-60s %d\n", w, byWhy[w])
	}
}
