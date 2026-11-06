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
//
// The replay is deterministic (FSRS Phase 2): -seed fixes every random
// source, and the engine reads a simulated clock (internal/clock) that
// stands at -now at the start of every session and moves a fixed step
// before each answer; between sessions the copy is aged one day. Two runs
// with the same snapshot, seed and -now print the same report and write the
// same -json, -surfaces and -dump files, byte for byte. -fsrs off|shadow
// runs the engine in that FSRS mode (default: MATHUA_FSRS); -surfaces
// writes, per session, what every surface shows (the review queue, the due
// count, Up next and its reasons, the mixed session, pacing, plan order,
// the gate, the mastery check's picks, XP and the progress view);
// -dump writes the final database as sorted text (-dump-fsrs=false leaves
// out the fsrs_* columns and review_log, for comparing off with shadow);
// -out-db keeps the replayed copy.
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

	"github.com/chuma-beep/mathua/internal/books"
	"github.com/chuma-beep/mathua/internal/clock"
	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/diagnostic"
	"github.com/chuma-beep/mathua/internal/engine"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/generator/all"
	"github.com/chuma-beep/mathua/internal/mastery"
	"github.com/chuma-beep/mathua/internal/modules"
	"github.com/chuma-beep/mathua/internal/planning"
	"github.com/chuma-beep/mathua/internal/pytrack"
	"github.com/chuma-beep/mathua/internal/review"
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
	surfw    io.Writer // -surfaces: one JSON line per session, or nil

	// sim is the simulated clock (nil: the wall clock, as the tests use):
	// it stands at base when a session starts and moves answerStep before
	// each answer.
	sim  *clock.Sim
	base time.Time

	// The served-order check (FORK.md, "The order of practice inside a
	// topic"): lessons checked and out of order.
	orderChecked, orderBad int
}

// tierRank orders a step's parts: the course's own book, parallel books,
// the practice set. A missed book exercise coming back (retry) and a
// question served outside the scaffold rank nowhere.
func tierRank(t string) int {
	switch t {
	case generator.TierMain:
		return 0
	case generator.TierParallel:
		return 1
	case generator.TierPractice:
		return 2
	}
	return -1
}

// servedOrderError checks questions served in order (one lesson batch, or
// a climb through a topic): within a run at one step, main book, then
// parallel books, then the practice set, never back. "" when in order.
func servedOrderError(probs []generator.Problem) string {
	step, top := 0, -1
	for i, p := range probs {
		if p.Step != step {
			step, top = p.Step, -1
		}
		r := tierRank(p.Tier)
		if r < 0 {
			continue
		}
		if r < top {
			return fmt.Sprintf("question %d at step %d is %s after %s", i+1, p.Step, p.Tier, []string{"main", "parallel", "practice"}[top])
		}
		top = r
	}
	return ""
}

// checkOrder records the served-order check of one lesson; a batch out of
// order is a bad serve.
func (r *replay) checkOrder(session int, cid string, probs []generator.Problem) {
	r.orderChecked++
	if msg := servedOrderError(probs); msg != "" {
		r.orderBad++
		r.serves = append(r.serves, Serve{Session: session, Surface: "study", Concept: cid, Why: "served order",
			Flags: []string{"served out of order: " + msg}})
	}
}

func main() {
	dbPath := flag.String("db", "", "database snapshot to replay (read only: the replay runs on a fresh copy)")
	sid := flag.String("student", "", "student id")
	n := flag.Int("sessions", 30, "sessions to replay, one per day")
	seed := flag.Int64("seed", 1, "random seed")
	placement := flag.Bool("placement", true, "also replay placement and the onboarding diagnostic")
	jsonOut := flag.String("json", "", "write every serve as a JSON line here")
	orderTopics := flag.String("order-topics", "", "comma-separated topics: climb each as a fresh learner on the followed courses and print the served sources per step")
	orderCourse := flag.String("order-course", "", "with -order-topics: the course the lessons are opened from (default: the followed courses that list the topic)")
	orderRounds := flag.Int("order-rounds", 8, "with -order-topics: lesson batches of 3 per topic")
	coverage := flag.String("coverage", "", "write a Markdown report of each course's topics with parallel-book items, and from which books, here")
	nowFlag := flag.String("now", "", "the simulated time each session starts at (RFC3339; default: 17:00 UTC on the day after the learner's last attempt)")
	fsrsFlag := flag.String("fsrs", "", "FSRS mode for the replay: off, shadow or active (default: MATHUA_FSRS)")
	surfaces := flag.String("surfaces", "", "write what every surface shows, one JSON line per session, here")
	dump := flag.String("dump", "", "write the final database as sorted text here")
	dumpFSRS := flag.Bool("dump-fsrs", true, "with -dump: include the fsrs_* columns and review_log")
	outDB := flag.String("out-db", "", "keep the replayed database copy here")
	fresh := flag.String("fresh", "", "replay a fresh learner placed into these courses (comma-separated ids) on an empty database instead of -db/-student")
	flag.Parse()
	if (*dbPath == "" || *sid == "") && *fresh == "" {
		fmt.Fprintln(os.Stderr, "usage: replay_audit -db copy.db -student ID [-sessions N] | -fresh course[,course]")
		os.Exit(2)
	}
	var work string
	var err error
	if *fresh != "" {
		// A fresh learner, created under the seed and the clock so the run
		// repeats: placed into the courses (placement and the onboarding
		// diagnostic), then the sessions.
		engine.SetDeterministicSeed(*seed)
		at := freshStart
		if *nowFlag != "" {
			if at, err = time.Parse(time.RFC3339, *nowFlag); err != nil {
				fmt.Fprintln(os.Stderr, "-now:", err)
				os.Exit(2)
			}
		}
		restore := clock.NewSim(at).Install()
		work, *sid, err = freshLearner()
		restore()
		*placement = true
	} else {
		work, err = freshCopy(*dbPath)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer os.RemoveAll(filepath.Dir(work))
	base, err := startTime(work, *sid, *nowFlag)
	if *fresh != "" && *nowFlag == "" {
		base = freshStart
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// The clock is installed before the engine is built: nothing the
	// replay does reads the wall clock.
	sim := clock.NewSim(base)
	defer sim.Install()()
	mode, merr := review.CurrentMode()
	if *fsrsFlag != "" {
		mode, merr = review.ParseMode(*fsrsFlag)
	}
	if merr == nil {
		merr = review.CheckBuilt(mode)
	}
	if merr != nil {
		fmt.Fprintln(os.Stderr, merr)
		os.Exit(2)
	}
	r, err := open(work, *sid, *seed)
	if err == nil && *fresh != "" {
		// Follow the courses, then reopen with them followed (own topics,
		// floor), as the fresh-learner test does.
		if _, err = r.eng.SetActiveCourses(*sid, strings.Split(*fresh, ",")); err == nil {
			r, err = open(work, *sid, *seed)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	r.sim, r.base = sim, base
	r.eng.SetFSRSMode(mode)
	fmt.Printf("clock %s, fsrs %s\n", base.Format(time.RFC3339), mode)
	if *surfaces != "" {
		f, err := os.Create(*surfaces)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer f.Close()
		r.surfw = f
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
	if *coverage != "" {
		if err := r.writeCoverage(*coverage); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *orderTopics != "" {
		bad := r.replayOrder(strings.Split(*orderTopics, ","), *orderCourse, *orderRounds)
		if bad > 0 {
			os.Exit(1)
		}
		return
	}
	if *placement {
		r.replayPlacement()
	}
	if err := r.runSessions(*n); err != nil {
		fmt.Fprintln(os.Stderr, "age:", err)
		os.Exit(1)
	}
	r.report()
	if *dump != "" {
		if err := dumpDB(r.db, *dump, *dumpFSRS); err != nil {
			fmt.Fprintln(os.Stderr, "dump:", err)
			os.Exit(1)
		}
	}
	if *outDB != "" {
		os.Remove(*outDB)
		if _, err := r.db.Exec("VACUUM INTO ?", *outDB); err != nil {
			fmt.Fprintln(os.Stderr, "out-db:", err)
			os.Exit(1)
		}
	}
}

// answerStep is how far the simulated clock moves before each answer.
const answerStep = 20 * time.Second

// tick moves the simulated clock one answer on (nothing on the wall clock).
func (r *replay) tick() {
	if r.sim != nil {
		r.sim.Advance(answerStep)
	}
}

// runSessions replays n sessions, one a day: each starts with the clock at
// base, and the copy is aged a day after it, so the next is tomorrow.
func (r *replay) runSessions(n int) error {
	for s := 1; s <= n; s++ {
		if r.sim != nil {
			r.sim.Set(r.base)
		}
		r.session(s)
		if err := r.ageOneDay(); err != nil {
			return err
		}
	}
	return nil
}

// freshStart is when a -fresh learner is created and placed.
var freshStart = time.Date(2026, 11, 5, 17, 0, 0, 0, time.UTC)

// freshLearner creates an empty database in a new temporary directory with
// one learner, and returns its path and the id; main has the engine set the
// courses (SetActiveCourses) and reopens.
func freshLearner() (string, string, error) {
	dir, err := os.MkdirTemp("", "replay_audit-")
	if err != nil {
		return "", "", err
	}
	db := filepath.Join(dir, "replay.db")
	store, err := storage.NewSQLiteStore(db)
	if err != nil {
		return "", "", err
	}
	defer store.Close()
	st, err := store.CreateStudent("fresh")
	if err != nil {
		return "", "", err
	}
	return db, st.ID, nil
}

// startTime is the simulated time sessions start at: -now when given, else
// 17:00 UTC (morning in the learner's zone) on the day after the learner's
// last attempt, so a snapshot replays from where it stopped and the run
// does not depend on the day it is made.
func startTime(dbPath, sid, flagNow string) (time.Time, error) {
	if flagNow != "" {
		t, err := time.Parse(time.RFC3339, flagNow)
		if err != nil {
			return time.Time{}, fmt.Errorf("-now: %w", err)
		}
		return t.UTC(), nil
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return time.Time{}, err
	}
	defer db.Close()
	var last sql.NullString
	if err := db.QueryRow(`select max(timestamp) from attempts where student_id = ?`, sid).Scan(&last); err != nil {
		return time.Time{}, fmt.Errorf("last attempt: %w", err)
	}
	day := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if last.Valid && last.String != "" {
		for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05"} {
			if t, err := time.Parse(layout, last.String); err == nil {
				day = t.UTC().Truncate(24 * time.Hour)
				break
			}
		}
	}
	return day.Add(24*time.Hour + 17*time.Hour), nil
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
	mods, err := modules.Default()
	if err != nil {
		return nil, fmt.Errorf("modules: %w", err)
	}
	return openWith(dbPath, sid, seed, mods)
}

// openWith opens the replay on one module set (the test fixtures load
// theirs explicitly, so MATHUA_PUBLIC_MODULES_ONLY does not decide).
func openWith(dbPath, sid string, seed int64, mods *modules.Set) (*replay, error) {
	root := os.Getenv("MATHUA_ROOT")
	if root == "" {
		return nil, fmt.Errorf("MATHUA_ROOT must name the install (its data/ and modules)")
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
	// One seed fixes every random source the engine draws from (attempt
	// ids that seed the generators, scheduler tie-breaks, the global
	// math/rand stream), so a same-seed replay is repeatable.
	engine.SetDeterministicSeed(seed)
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

	// What is due, read from every surface (surfaces.go): a disagreement
	// is a bad serve of its own.
	rep := r.dueSurfaces()
	if len(rep.Disagreements) > 0 {
		cid := ""
		if len(rep.Due) > 0 {
			cid = rep.Due[0]
		}
		note(Serve{Surface: "due", Concept: cid, Why: rep.String(), Flags: []string{"surfaces disagree on due: " + strings.Join(rep.Disagreements, "; ")}})
	}
	r.writeSurfaces(n, rep)

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
			r.checkOrder(n, cid, probs)
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
				r.tick()
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
				r.tick()
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
	r.tick()
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
		r.tick()
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
	if s.Surface == "due" {
		return s.Flags // classified by dueSurfaces
	}
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
	} else if d, ok := r.depth[s.Concept]; ok && d > engine.CoursePlacementDepth && !pinned && !own &&
		s.Surface != "placement" && s.Surface != "onboarding" {
		// A foundation more links below every followed course's topics than
		// a placement reaches (Q57: the mean of a data set, school
		// probability, six links under "Power sets").
		flags = append(flags, fmt.Sprintf("foundation deeper than %d", engine.CoursePlacementDepth))
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
	met := r.eng.FloorMet(r.sid)
	for _, pid := range c.Prerequisites {
		if met[pid] {
			continue // below the learner's floor: met, as the engine counts it
		}
		p := progress[pid]
		if p == nil {
			return false
		}
		if p.PlacementSeeded {
			continue
		}
		days := 999.0
		if p.LastReviewed != nil {
			days = clock.Since(*p.LastReviewed).Hours() / 24
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
// The FSRS state moves with it (plan 4.9): fsrs_last_review and
// review_log.at shift by the same day, so the next event's elapsed time,
// the log's time order and the doctor's replay check stay right. The log
// keeps its fixed-width nine-digit form (storage.ReviewLogAtLayout); the
// elapsed_days a row recorded is what it read and does not change.
func (r *replay) ageOneDay() error {
	for _, stmt := range ageStatements {
		if _, err := r.db.Exec(stmt, r.sid); err != nil {
			return err
		}
	}
	return nil
}

// ageStatements shift one learner's record a day into the past (the daily
// QA playbook runs the same statements on its copy: PLAYBOOK_ADDENDUM_FSRS_SHADOW.md).
var ageStatements = []string{
	`update concept_progress set
		   last_attempted = case when last_attempted is null or last_attempted='' then last_attempted else strftime('%Y-%m-%dT%H:%M:%SZ', last_attempted, '-1 day') end,
		   last_reviewed = case when last_reviewed is null or last_reviewed='' then last_reviewed else strftime('%Y-%m-%dT%H:%M:%SZ', last_reviewed, '-1 day') end,
		   next_review_due = case when next_review_due is null or next_review_due='' then next_review_due else strftime('%Y-%m-%dT%H:%M:%SZ', next_review_due, '-1 day') end,
		   mastered_at = case when mastered_at is null or mastered_at='' then mastered_at else strftime('%Y-%m-%dT%H:%M:%SZ', mastered_at, '-1 day') end,
		   fsrs_last_review = case when fsrs_last_review is null or fsrs_last_review='' then fsrs_last_review else strftime('%Y-%m-%dT%H:%M:%SZ', fsrs_last_review, '-1 day') end
		 where student_id = ?`,
	`update review_log set at = substr(strftime('%Y-%m-%dT%H:%M:%S', at, '-1 day'), 1, 19) || substr(at, 20) where student_id = ?`,
	`update attempts set timestamp = strftime('%Y-%m-%dT%H:%M:%SZ', timestamp, '-1 day') where student_id = ?`,
	`update quiz_completions set completed_at = strftime('%Y-%m-%d %H:%M:%S', completed_at, '-1 day') where student_id = ?`,
}

// ---- report ----

func (r *replay) report() {
	fmt.Printf("\nserved order (main book -> parallel books -> practice set): %d lessons checked, %d out of order\n", r.orderChecked, r.orderBad)
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

// replayOrder climbs each topic as a fresh learner following the same
// courses: lesson batches of three, answered right four times in five,
// printing each question's step, tier and source, and checks the climb
// serves main book -> parallel books -> practice set at every step. It
// returns the topics out of order.
func (r *replay) replayOrder(topics []string, course string, rounds int) int {
	st, _ := r.repo.GetStudent(r.sid)
	bad := 0
	for i, cid := range topics {
		cid = strings.TrimSpace(cid)
		if cid == "" || r.dag.Concept(cid) == nil {
			fmt.Printf("\n%s: unknown topic\n", cid)
			continue
		}
		fresh, err := r.repo.CreateStudent(fmt.Sprintf("order-%d", i))
		if err != nil {
			fmt.Println("student:", err)
			return 1
		}
		if st != nil {
			r.repo.SetCourseID(fresh.ID, st.CourseID)
		}
		var courses []string
		if course != "" {
			courses = engine.ActiveCourses(course)
		}
		fmt.Printf("\n%s (%s)\n", cid, r.dag.Concept(cid).Label)
		var all []generator.Problem
		for round := 0; round < rounds; round++ {
			ctx := generator.GeneratorContext{Difficulty: 0.4, Seed: r.rng.Int63(), History: r.eng.LearnerHistoryIn(fresh.ID, cid, courses)}
			probs, err := r.reg.BatchGenerateContext(cid, 3, ctx)
			if err != nil || len(probs) == 0 {
				break
			}
			anchors := map[string]engine.Anchor{}
			for _, p := range probs {
				anchors[p.Question] = engine.Anchor{Answer: p.Answer, Explanation: p.Explanation, Difficulty: p.Difficulty}
			}
			r.eng.SetStudyAnchorBatch(fresh.ID, cid, anchors)
			for _, p := range probs {
				src := books.Label(p.SourceBook)
				if src == "" {
					src = "-"
				}
				tier := p.Tier
				if tier == "" {
					tier = "sampled (steps done)"
				}
				path := make([]string, len(p.StepPath))
				for k, b := range p.StepPath {
					path[k] = books.Label(b)
					if b == "" {
						path[k] = "practice set"
					}
				}
				q := strings.Join(strings.Fields(p.Question), " ")
				if len(q) > 48 {
					q = q[:48] + "..."
				}
				ok := r.rng.Float64() < 0.8
				mark := "right"
				if !ok {
					mark = "wrong"
				}
				fmt.Printf("  r%d step %d %-9s %-32s [%s] %s: %s\n", round+1, p.Step, tier, src, strings.Join(path, " -> "), mark, q)
				r.eng.SubmitStudyAnswer(fresh.ID, cid, r.answerFor(p.Answer, ok), 20, p.Question)
			}
			all = append(all, probs...)
		}
		if msg := servedOrderError(all); msg != "" {
			bad++
			fmt.Println("  OUT OF ORDER:", msg)
		} else {
			fmt.Println("  in order")
		}
	}
	fmt.Printf("\n%d topics, %d out of order\n", len(topics), bad)
	return bad
}

// writeCoverage reports, per course, the topics whose scaffold has
// parallel-book items for a fresh learner opening them from the course, and
// from which books.
func (r *replay) writeCoverage(path string) error {
	fresh, err := r.repo.CreateStudent("coverage")
	if err != nil {
		return err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Parallel books by course (%s)\n\n", time.Now().Format("2006-01-02"))
	b.WriteString("For a fresh learner opening each topic from the course: the course's own book on the topic (main), and the other books whose items the topic serves after it and before the practice set (parallel), with their item counts. Generated by `go run ./scripts/replay_audit -coverage`.\n\n")
	b.WriteString("| Course | Topics | With a main book | With parallel books | Parallel books (topics) |\n|---|---|---|---|---|\n")
	var detail strings.Builder
	for _, c := range r.planner.Courses() {
		seen := map[string]bool{}
		var topics []string
		add := func(ids []string) {
			for _, id := range ids {
				if !seen[id] && r.dag.Concept(id) != nil {
					seen[id] = true
					topics = append(topics, id)
				}
			}
		}
		add(c.Plan)
		add(c.Targets)
		for _, m := range c.Modules {
			add(m.Topics)
		}
		if len(topics) == 0 {
			continue
		}
		withMain, withPar := 0, 0
		bookTopics := map[string]int{}
		var rows []string
		for _, cid := range topics {
			g, err := r.reg.Get(cid)
			if err != nil {
				continue
			}
			rep, ok := g.(generator.ScaffoldReporter)
			if !ok {
				continue
			}
			sc := rep.ScaffoldFor(generator.GeneratorContext{Difficulty: 0.4, History: r.eng.LearnerHistoryIn(fresh.ID, cid, []string{c.ID})})
			if sc == nil {
				continue
			}
			n := map[string]int{}
			for _, it := range sc.Items {
				n[it.Source]++
			}
			if sc.Main != "" {
				withMain++
			}
			if len(sc.Books) == 0 {
				continue
			}
			withPar++
			var parts []string
			for _, bk := range sc.Books {
				bookTopics[bk]++
				parts = append(parts, fmt.Sprintf("%s %d", books.Label(bk), n[bk]))
			}
			main := "-"
			if sc.Main != "" {
				main = fmt.Sprintf("%s %d", books.Label(sc.Main), n[sc.Main])
			}
			rows = append(rows, fmt.Sprintf("| `%s` | %s | %s | %s |", cid, r.dag.Concept(cid).Label, main, strings.Join(parts, ", ")))
		}
		bks := make([]string, 0, len(bookTopics))
		for bk := range bookTopics {
			bks = append(bks, bk)
		}
		sort.Slice(bks, func(i, j int) bool {
			if bookTopics[bks[i]] != bookTopics[bks[j]] {
				return bookTopics[bks[i]] > bookTopics[bks[j]]
			}
			return bks[i] < bks[j]
		})
		var list []string
		for _, bk := range bks {
			list = append(list, fmt.Sprintf("%s (%d)", books.Label(bk), bookTopics[bk]))
		}
		if len(list) == 0 {
			list = []string{"none"}
		}
		fmt.Fprintf(&b, "| %s (`%s`) | %d | %d | %d | %s |\n", c.Name, c.ID, len(topics), withMain, withPar, strings.Join(list, ", "))
		if len(rows) > 0 {
			fmt.Fprintf(&detail, "\n## %s (`%s`)\n\nTexts: %s\n\n| Topic | Label | Main book (items) | Parallel books (items), in serving order |\n|---|---|---|---|\n%s\n", c.Name, c.ID, strings.Join(c.Texts, ", "), strings.Join(rows, "\n"))
		}
	}
	b.WriteString(detail.String())
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
