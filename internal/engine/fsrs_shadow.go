package engine

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/chuma-beep/mathua/internal/fsrs"
	"github.com/chuma-beep/mathua/internal/grader"
	"github.com/chuma-beep/mathua/internal/mastery"
	"github.com/chuma-beep/mathua/internal/review"
	"github.com/chuma-beep/mathua/internal/storage"
)

// SetFSRSMode sets the engine's FSRS mode (MATHUA_FSRS by default). The
// replay and tests use it to run off and shadow in one process.
func (e *Engine) SetFSRSMode(m review.Mode) { e.fsrsMode = m }

// FSRSMode is the engine's FSRS mode.
func (e *Engine) FSRSMode() review.Mode { return e.fsrsMode }

// SetFSRSRules sets which events move FSRS (MATHUA_FSRS_RULES by default).
func (e *Engine) SetFSRSRules(r review.Rules) { e.fsrsRules = r }

// FSRSRules is the rules the engine applies in shadow and active mode.
func (e *Engine) FSRSRules() review.Rules { return e.fsrsRules }

// PacedPlanOrder is the order new topics are offered in (pacedOrder), as a
// list: read-only, for the replay's surface comparison.
func (e *Engine) PacedPlanOrder(studentID string) []string {
	progress, err := e.repo.GetAllProgress(studentID)
	if err != nil {
		return nil
	}
	order := e.pacedOrder(studentID, progress)
	ids := make([]string, 0, len(order))
	for id := range order {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		if order[ids[i]] != order[ids[j]] {
			return order[ids[i]] < order[ids[j]]
		}
		return ids[i] < ids[j]
	})
	return ids
}

// ---- shadow events (FSRS Phase 2, plan 3.3) ----

// The sources an answer is graded under, and the review_log sources an
// event is written under.
const (
	fsrsSrcReview       = "review"        // a review-session answer
	fsrsSrcMasteryCheck = "mastery_check" // a weekly mastery check item
	fsrsSrcCourseTest   = "course_test"   // a course review test item
	fsrsSrcPractice     = "practice"      // Learn, the mixed session's new work
	fsrsLogFirstMastery = "first_mastery" // the answer that first masters a topic
	fsrsLogPracticeLaps = "practice_lapse"
)

// fsrsAnswer is what the shadow event needs to know about one graded answer
// beyond the progress rows.
type fsrsAnswer struct {
	source   string // fsrsSrc*
	expected string // the answer key: tells recall, proof and bank items apart
	graded   bool   // a proof answer the judge graded (Result.Proof set)
	key      string // idempotency key; "" is the attempt's own
}

// FSRSMemory is a progress row's FSRS state and the time of its last
// review (zero when it has none).
func FSRSMemory(p *storage.ConceptProgress) (review.Memory, time.Time) {
	if p == nil {
		return review.Memory{}, time.Time{}
	}
	m := review.Memory{Reps: p.FSRSReps, Lapses: p.FSRSLapses, Phase: review.Phase(p.FSRSState)}
	if p.FSRSStability != nil {
		m.Stability = *p.FSRSStability
	}
	if p.FSRSDifficulty != nil {
		m.Difficulty = *p.FSRSDifficulty
	}
	if p.FSRSStability == nil {
		m.Phase = review.PhaseNone
	}
	var last time.Time
	if p.FSRSLastReview != nil {
		last = *p.FSRSLastReview
	}
	return m, last
}

func setFSRS(p *storage.ConceptProgress, m review.Memory, last time.Time, seed string) {
	s, d, l := m.Stability, m.Difficulty, last.UTC()
	p.FSRSStability, p.FSRSDifficulty, p.FSRSLastReview = &s, &d, &l
	p.FSRSReps, p.FSRSLapses, p.FSRSState, p.FSRSSeed = m.Reps, m.Lapses, string(m.Phase), seed
}

// fsrsItemID is the bank item an answer key names, "" for a generated one.
func fsrsItemID(expected string) string {
	for _, pre := range []string{"math:", "term:", "proof:"} {
		if strings.HasPrefix(expected, pre) {
			return expected
		}
	}
	return ""
}

// fsrsSeedOf is how a stateless mastered row's first state is labelled
// (fsrs_seed): its own attempts, placement or credit.
func fsrsSeedOf(p *storage.ConceptProgress, demonstrated bool) string {
	switch {
	case demonstrated:
		return "mastery"
	case p.PlacementSeeded:
		return "placement"
	case p.ImplicitCredit > 0:
		return "credit"
	}
	return "mastery"
}

// raisedOn reports an applied, stability-raising log row for the topic on
// at's learner-local day.
func raisedOn(log []storage.ReviewLogEntry, at time.Time, loc *time.Location) bool {
	for _, e := range log {
		if e.Held == "" && e.Source != "seed" && e.SAfter > e.SBefore && review.SameLocalDay(e.At, at, loc) {
			return true
		}
	}
	return false
}

// fsrsRun applies Steps to one topic, appending each as a review_log row:
// the one path by which the live event and the seed replay write state.
type fsrsRun struct {
	e        *Engine
	sid, cid string
	mem      review.Memory
	last     time.Time
	log      []storage.ReviewLogEntry // stored rows for the topic, then these
	out      []storage.ReviewLogEntry
	loc      *time.Location
}

func (r *fsrsRun) step(source string, rating fsrs.Rating, at time.Time, lapse bool, key, item string, responseMS int64) error {
	st := review.Step{Rating: rating, At: at, Cap: r.e.fsrsRules.DailyCap, RaisedToday: raisedOn(r.log, at, r.loc)}
	if lapse {
		st.LapseBelowR = r.e.fsrsRules.LapseBelowR
	}
	a, err := review.DefaultParams().Apply(r.mem, r.last, st)
	if err != nil {
		return err
	}
	entry := storage.ReviewLogEntry{
		StudentID: r.sid, ConceptID: r.cid, At: at.UTC(), Source: source, Rating: int(rating),
		ElapsedDays: a.ElapsedDays, SBefore: r.mem.Stability, SAfter: a.Next.Stability,
		DBefore: r.mem.Difficulty, DAfter: a.Next.Difficulty, RAtReview: a.R, ItemID: item,
		ResponseMS: responseMS, IdempotencyKey: key, EngineMode: string(r.e.fsrsMode), Held: a.Held,
	}
	r.out = append(r.out, entry)
	r.log = append(r.log, entry)
	if a.Held == "" {
		r.mem, r.last = a.Next, at
	}
	return nil
}

func (e *Engine) newFSRSRun(before *storage.ConceptProgress) (*fsrsRun, error) {
	loc, _ := review.LearnerZone()
	log, err := e.repo.ReviewLogFor(before.StudentID, before.ConceptID)
	if err != nil {
		return nil, err
	}
	m, last := FSRSMemory(before)
	return &fsrsRun{e: e, sid: before.StudentID, cid: before.ConceptID, mem: m, last: last, log: log, loc: loc}, nil
}

// ownAttempts is the topic's attempts that are the learner's own work, in
// time order (no diagnostic rows), at or before until.
func (e *Engine) ownAttempts(sid, cid string, until time.Time) ([]storage.AttemptEntry, error) {
	all, err := e.repo.GetAttemptsForStudent(sid)
	if err != nil {
		return nil, err
	}
	var out []storage.AttemptEntry
	for _, a := range all {
		if a.ConceptID == cid && a.Source != "diagnostic" && !a.Timestamp.After(until) {
			out = append(out, a)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].Timestamp.Equal(out[j].Timestamp) {
			return out[i].Timestamp.Before(out[j].Timestamp)
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// seedFromAttempts gives a demonstrated-mastered row with no state its
// state by replaying its own attempts (plan 3.4): the reference first
// review (Good) at its mastery, then each later attempt as the 3.3 table
// says (a review or mastery-check answer, right Good and wrong Again; a
// practice miss as a lapse under the threshold; a practice right answer, a
// self-graded card and an ungraded item nothing), under the daily cap. The
// keys are "seed:a<attempt id>" (or "seed:<learner>:<topic>:mastery" when
// no attempt marks the mastery), so seeding twice writes nothing. It
// reports false when the row has no attempts of its own (credited or
// placed only: no state, plan 3.4).
func (e *Engine) seedFromAttempts(r *fsrsRun, p *storage.ConceptProgress, until time.Time) (bool, error) {
	atts, err := e.ownAttempts(p.StudentID, p.ConceptID, until)
	if err != nil || len(atts) == 0 {
		return false, err
	}
	at := until
	if p.MasteredAt != nil {
		at = *p.MasteredAt
	} else if p.LastReviewed != nil {
		at = *p.LastReviewed
	}
	// The attempt that mastered it: the last correct own answer at or
	// before the mastery time.
	initKey, initIdx := fmt.Sprintf("seed:%s:%s:mastery", p.StudentID, p.ConceptID), -1
	for i, a := range atts {
		if a.Timestamp.After(at.Add(time.Second)) {
			break
		}
		if a.Correct {
			initKey, initIdx = fmt.Sprintf("seed:a%d", a.ID), i
		}
	}
	if err := r.step(fsrsLogFirstMastery, fsrs.Good, at, false, initKey, "", 0); err != nil {
		return false, err
	}
	for i, a := range atts {
		if i <= initIdx || !a.Timestamp.After(at) || isSelfGraded(a.Expected) {
			continue
		}
		if grader.IsProofAnswer(a.Expected) && !e.fsrsRules.Judged {
			continue
		}
		key := fmt.Sprintf("seed:a%d", a.ID)
		rating := fsrs.Again
		if a.Correct {
			rating = fsrs.Good
		}
		ms := int64(a.ElapsedSeconds * 1000)
		switch a.Source {
		case "review":
			err = r.step(fsrsSrcReview, rating, a.Timestamp, false, key, fsrsItemID(a.Expected), ms)
		case "quiz":
			// A quiz row is a mastery check or a course review test; the
			// attempt does not say which, and both count by default.
			if e.fsrsRules.MasteryCheck {
				err = r.step(fsrsSrcMasteryCheck, rating, a.Timestamp, false, key, fsrsItemID(a.Expected), ms)
			}
		default:
			if !a.Correct && e.fsrsRules.PracticeLapse {
				err = r.step(fsrsLogPracticeLaps, fsrs.Again, a.Timestamp, true, key, fsrsItemID(a.Expected), ms)
			}
		}
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

// fsrsCommit is the FSRS half of one graded answer in shadow or active mode
// (plan 3.3, with Rules deciding the open rows), or nil when the answer is
// no event. before is the row as it was, after as the answer leaves it. It
// never decides anything: nothing reads what it writes in Phase 2.
func (e *Engine) fsrsCommit(before, after *storage.ConceptProgress, o gradedAttempt) (*storage.FSRSCommit, error) {
	if !e.fsrsMode.Writes() || o.attempt == nil {
		return nil, nil
	}
	a, rules := o.fsrs, e.fsrsRules
	switch {
	case isSelfGraded(a.expected):
		return nil, nil // a self-graded card is never a review
	case grader.IsProofAnswer(a.expected) && (!rules.Judged || !a.graded):
		return nil, nil // judged only when graded; a failed call is no event
	case a.source == fsrsSrcMasteryCheck && !rules.MasteryCheck,
		a.source == fsrsSrcCourseTest && !rules.CourseTests:
		return nil, nil
	}
	at := o.attempt.Timestamp
	rating := fsrs.Again
	if o.correct {
		rating = fsrs.Good
	}
	wasMastered := mastery.Status(before.Status) == mastery.StatusMastered
	nowMastered := mastery.Status(after.Status) == mastery.StatusMastered
	mem, _ := FSRSMemory(before)
	practice := a.source == fsrsSrcPractice
	source, lapse := a.source, false
	switch {
	case practice && mem.IsZero() && !wasMastered && nowMastered && o.correct:
		if !rules.FirstMastery {
			return nil, nil
		}
		source = fsrsLogFirstMastery
	case !wasMastered && mem.IsZero():
		return nil, nil // not yet in the memory model
	case practice && (o.correct || !wasMastered || !rules.PracticeLapse):
		return nil, nil // practice right, or practice on a topic not mastered: evidence only
	case practice:
		source, lapse = fsrsLogPracticeLaps, true
	}
	r, err := e.newFSRSRun(before)
	if err != nil {
		return nil, err
	}
	seed := before.FSRSSeed
	if mem.IsZero() {
		seed = "mastery"
		if source != fsrsLogFirstMastery {
			// A mastered row with no state: seed it from its own attempts
			// at this first touch, or (credited, placed) let this answer
			// create its state.
			demonstrated := false
			if rules.SeedOnTouch {
				if demonstrated, err = e.seedFromAttempts(r, before, at.Add(-time.Nanosecond)); err != nil {
					return nil, err
				}
			} else if own, err := e.ownAttempts(before.StudentID, before.ConceptID, at.Add(-time.Nanosecond)); err == nil {
				demonstrated = len(own) > 0
			}
			seed = fsrsSeedOf(before, demonstrated)
		}
	}
	if err := r.step(source, rating, at, lapse, a.key, fsrsItemID(a.expected), int64(o.elapsed*1000)); err != nil {
		return nil, err
	}
	state := *after
	if r.mem.IsZero() {
		return &storage.FSRSCommit{Log: r.out}, nil
	}
	setFSRS(&state, r.mem, r.last, seed)
	return &storage.FSRSCommit{Log: r.out, State: &state}, nil
}

// commitAnswer writes one graded answer: the progress row and the attempt,
// and in shadow or active mode the FSRS event, in one transaction
// (storage.CommitAnswer). A shadow failure never costs the learner the
// answer: the event is dropped, logged, and the answer committed without
// it.
func (e *Engine) commitAnswer(progress, before *storage.ConceptProgress, o gradedAttempt) (int64, error) {
	c := storage.AnswerCommit{Progress: progress, Attempt: o.attempt}
	fc, err := e.fsrsCommit(before, progress, o)
	if err != nil {
		log.Printf("fsrs: %s event for %s/%s not computed: %v", e.fsrsMode, o.studentID, o.conceptID, err)
	}
	c.FSRS = fc
	res, err := e.repo.CommitAnswer(c)
	if err == nil && res.Duplicate {
		return 0, ErrDuplicateAnswer
	}
	if err != nil && c.FSRS != nil {
		log.Printf("fsrs: %s event for %s/%s not written: %v", e.fsrsMode, o.studentID, o.conceptID, err)
		c.FSRS = nil
		res, err = e.repo.CommitAnswer(c)
	}
	return res.AttemptID, err
}

// ErrDuplicateAnswer is a retried answer whose event is already stored: the
// first submission's writes stand and nothing of this one is recorded.
var ErrDuplicateAnswer = fmt.Errorf("answer already recorded: %w", ErrNoActiveQuestion)
