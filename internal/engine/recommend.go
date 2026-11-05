package engine

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/chuma-beep/mathua/internal/mastery"
	"github.com/chuma-beep/mathua/internal/review"
	"github.com/chuma-beep/mathua/internal/scheduler"
	"github.com/chuma-beep/mathua/internal/storage"
)

// conceptSnapshots assembles the scheduler's view of a learner.
//
// This was written out three times — NextQuestion, NextReviewQuestion and the recommender —
// which is how they came to disagree. The recommender in particular could not reuse either: it
// needs the whole DAG including unseen concepts, and it needs to run outside a session.
//
// One definition, so "what should this learner do next" cannot answer differently depending on
// which caller asked.
// ConceptSnapshots is the scheduler's view of a learner's progress, for
// surfaces that render eligibility themselves (the course view). progress may
// be nil for a guest.
func (e *Engine) ConceptSnapshots(studentID string, progress map[string]*storage.ConceptProgress) map[string]*scheduler.ConceptSnapshot {
	return e.conceptSnapshotsFrom(studentID, progress)
}

func (e *Engine) conceptSnapshotsFrom(studentID string, progress map[string]*storage.ConceptProgress) map[string]*scheduler.ConceptSnapshot {
	snapshots := make(map[string]*scheduler.ConceptSnapshot, len(progress))
	for cid, p := range progress {
		reqStreak := 0
		timeThresh := 0.0
		if c := e.dag.Concept(cid); c != nil {
			reqStreak = c.MasteryThreshold.Streak
			timeThresh = c.MasteryThreshold.AvgTimeSeconds
		}
		snapshots[cid] = &scheduler.ConceptSnapshot{
			Status:         mastery.Status(p.Status),
			Streak:         p.Streak,
			LastAttempted:  timeOrZero(p.LastAttempted),
			LastReviewed:   timeOrZero(p.LastReviewed),
			NextReviewDue:  p.NextReviewDue,
			RequiredStreak: reqStreak,
			TimeThreshold:  timeThresh,
			WeaknessScore:  p.WeaknessScore,
			Placement:      p.PlacementSeeded,
			Attempts:       p.Attempts,
		}
	}
	// Concepts never attempted still need a snapshot: prereqsMet returns false for a missing
	// one, so an absent concept reads as blocked rather than as available-and-unstarted.
	for _, c := range e.dag.Order() {
		if _, ok := snapshots[c.ID]; !ok {
			snapshots[c.ID] = &scheduler.ConceptSnapshot{
				Status:         mastery.StatusUnseen,
				RequiredStreak: c.MasteryThreshold.Streak,
				TimeThreshold:  c.MasteryThreshold.AvgTimeSeconds,
			}
		}
	}
	// A topic below the learner's floor counts as a met prerequisite, as a
	// placement does (reachability, never mastery): the floor hides it, so
	// it must not lock the course topics that rest on it (floor.go).
	for cid := range e.floorMet(studentID) {
		if s := snapshots[cid]; s != nil {
			s.Placement = true
		}
	}
	return snapshots
}

// conceptSnapshots is the loading wrapper, for the question-serving paths.
func (e *Engine) conceptSnapshots(studentID string) (map[string]*scheduler.ConceptSnapshot, error) {
	progress, err := e.repo.GetAllProgress(studentID)
	if err != nil {
		return nil, fmt.Errorf("load progress: %w", err)
	}
	return e.conceptSnapshotsFrom(studentID, progress), nil
}

// ReviewsDue counts the topics due for review. Moved here from the server,
// which computed it inline and which the recommender needs it from.
func (e *Engine) ReviewsDue(studentID string) (int, error) {
	return e.ReviewsDueTrack(studentID, "")
}

// ReviewsDueTrack counts due reviews in one track ("" is the Math track):
// the length of DueReviewIDs, so the count, the Study banner and the
// sidebar badge (/api/reviews/due) agree with the review session.
func (e *Engine) ReviewsDueTrack(studentID, track string) (int, error) {
	progress, err := e.repo.GetAllProgress(studentID)
	if err != nil {
		return 0, fmt.Errorf("load progress: %w", err)
	}
	return len(e.DueReviewIDs(studentID, track, progress, time.Now())), nil
}

// DueReviewIDs is the one answer to "what is due for review" (fork change,
// FSRS plan Phase 0): every topic the learner is reviewed on whose row is
// due (review.IsDue), most overdue first. It is what the due count, the
// Study banner, the course page's due marks, the Up next review card and
// its reason, the review session and the mixed session's review serves all
// read, so they cannot disagree about a topic.
//
// Not due, whatever the row says: a topic the learner is not reviewed on
// (notReviewed: below their floor, or a credit-only foundation no followed
// course lists, unless pinned), a topic outside the track, and a row for a
// topic the graph no longer has (the session could not serve it).
func (e *Engine) DueReviewIDs(studentID, track string, progress map[string]*storage.ConceptProgress, now time.Time) []string {
	skip := e.notReviewed(studentID, progress)
	type due struct {
		id      string
		overdue float64
	}
	var out []due
	for cid, p := range progress {
		if p == nil || skip[cid] || e.dag.Concept(cid) == nil || !e.InTrack(cid, track) {
			continue
		}
		if r := progressRow(p); review.IsDue(r, now) {
			out = append(out, due{cid, review.Overdue(r, now)})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].overdue != out[j].overdue {
			return out[i].overdue > out[j].overdue
		}
		return out[i].id < out[j].id
	})
	ids := make([]string, len(out))
	for i, d := range out {
		ids[i] = d.id
	}
	return ids
}

// ReviewCandidates is what the review session could serve at now (the
// scheduler's review candidates minus notReviewed), read without a
// session. It must equal DueReviewIDs(track ""); fsrs_doctor checks it.
func (e *Engine) ReviewCandidates(studentID string, progress map[string]*storage.ConceptProgress, now time.Time) []string {
	snapshots := e.conceptSnapshotsFrom(studentID, progress)
	return e.trackScheduler("").ReviewCandidates(snapshots, now, e.notReviewed(studentID, progress))
}

// DueReviewSet is DueReviewIDs as a set, for surfaces that mark topics.
func (e *Engine) DueReviewSet(studentID string, progress map[string]*storage.ConceptProgress, now time.Time) map[string]bool {
	out := map[string]bool{}
	for _, id := range e.DueReviewIDs(studentID, "", progress, now) {
		out[id] = true
	}
	return out
}

// progressRow is what the due rule reads of a stored row.
func progressRow(p *storage.ConceptProgress) review.Row {
	return review.Row{
		Status:        mastery.Status(p.Status),
		LastReviewed:  timeOrZero(p.LastReviewed),
		LastAttempted: timeOrZero(p.LastAttempted),
		NextReviewDue: p.NextReviewDue,
	}
}

// RecommendNext is the learner's head plus their alternatives. It is the answer to "what
// should I do now?", and it is answered once, here, rather than recomputed in the browser from
// five API calls and a bundled copy of the corpus.
func (e *Engine) RecommendNext(studentID string, exclude []string) (scheduler.RecommendationResponse, error) {
	return e.RecommendNextTrack(studentID, exclude, "")
}

// RecommendNextTrack recommends within one track ("" is the math track).
func (e *Engine) RecommendNextTrack(studentID string, exclude []string, track string) (scheduler.RecommendationResponse, error) {
	track = e.NormalizeTrack(track)
	// A track whose module says it has no placement (the Python track's first
	// topics are open from the start) is never offered the diagnostic.
	noPlacement := track != AllTracks && !e.Modules().HasPlacement(track)
	// The offer view: the track's concepts plus the outside concepts they
	// directly need, so sequencing crosses tracks (see trackViews).
	sched := e.offerScheduler(track)
	if studentID == "" {
		return sched.Recommend(scheduler.RecommendInput{
			Now: time.Now(), Weakness: map[string]float64{},
			DiagnosticCompleted: noPlacement,
		}), nil
	}
	progress, err := e.repo.GetAllProgress(studentID)
	if err != nil {
		return scheduler.RecommendationResponse{Alternatives: []scheduler.Recommendation{}}, fmt.Errorf("load progress: %w", err)
	}
	snapshots := e.conceptSnapshotsFrom(studentID, progress)
	weakness := e.weaknessMapFromProgress(progress)
	now := time.Now()
	dueIDs := e.DueReviewIDs(studentID, track, progress, now)
	due := len(dueIDs)
	dueSet := make(map[string]bool, due)
	for _, id := range dueIDs {
		dueSet[id] = true
	}
	quizDue, _ := e.QuizDue(studentID)
	quizXP, _ := e.QuizXPSince(studentID)
	quizTaken := false
	if last, err := e.repo.LastQuizCompletion(studentID); err == nil && last != nil {
		quizTaken = true
	}

	// How far along the learner is, in the two terms the recommender needs. Mastered is
	// effective MASTERED — decay applied, so a stale concept has left the count rather than
	// inflating it, and *not* "anything not unseen", which counted a concept the learner had
	// merely started. Only the new-learner gate consumes this today and AttemptedDays already
	// guarded it, so the miscount was unobservable; it is fixed because the field is named
	// after a fact it did not report, and the next caller inherits discrepancies rather than
	// reading the comment.
	mastered, attemptedDays := 0, map[string]bool{}
	for _, p := range progress {
		if !e.InTrack(p.ConceptID, track) {
			continue
		}
		if mastery.EffectiveStatus(mastery.Status(p.Status), daysSinceReview(p.LastReviewed), mastery.DecayDays) == mastery.StatusMastered {
			mastered++
		}
		if p.LastAttempted != nil {
			attemptedDays[p.LastAttempted.UTC().Format("2006-01-02")] = true
		}
	}

	// Topics outside the learner's course paths are not offered. They are
	// excluded, not dropped from the snapshots: a dropped topic had no
	// snapshot and read as never seen, so one whose prerequisite the
	// placement had credited was offered as "Ready to learn" ("Addition word
	// problems" after a calculus placement; Q27).
	// A due review is never off the path (FORK.md: reviews come most
	// overdue first "across every course and book"): the review card
	// stands for the whole due set, as the count does.
	e.ensureActivePath(studentID)
	e.mu.Lock()
	if path := e.activePath[studentID]; len(path) > 0 {
		for cid := range snapshots {
			if !path[cid] && !dueSet[cid] {
				exclude = append(exclude, cid)
			}
		}
	}
	e.mu.Unlock()
	// Credit-only foundations no followed course asks for are neither
	// reviewed nor offered as new (creditOnlyOffCourse), and mastered topics
	// below the review floor are not served again (belowReviewFloor).
	for cid := range e.notReviewed(studentID, progress) {
		exclude = append(exclude, cid)
	}

	// The server's record that placement happened. This is the authoritative
	// guard against offering the diagnostic back as an active task; the
	// AttemptedDays check alone is not enough, because placement deliberately
	// seeds reachability without mastery.
	diagnosticCompleted := noPlacement
	if st, err := e.repo.GetStudent(studentID); err == nil && st != nil && st.DiagnosticCompleted {
		diagnosticCompleted = true
	}
	pathOrder := e.pacedOrder(studentID, progress)
	pinned := map[string]bool{}
	for _, pin := range e.Pins(studentID) {
		pinned[pin.Concept] = true
	}

	remedial := map[string]bool{}
	for _, id := range e.QuizRemedial(studentID) {
		remedial[id] = true
	}
	urgent := e.urgentTopics(studentID, progress, now)
	res := sched.Recommend(scheduler.RecommendInput{
		Urgent:              urgent,
		Remedial:            remedial,
		Snapshots:           snapshots,
		Now:                 now,
		Weakness:            weakness,
		DueReviews:          due,
		QuizDue:             quizDue,
		QuizXPSince:         quizXP,
		QuizTaken:           quizTaken,
		ConceptsMastered:    mastered,
		AttemptedDays:       len(attemptedDays),
		ExcludeConceptIds:   exclude,
		DiagnosticCompleted: diagnosticCompleted,
		PathOrder:           pathOrder,
		Pinned:              pinned,
	})
	e.labelBridges(&res, track)
	if res.Primary != nil && res.Primary.ID == "graph" {
		// "Nothing to do" is wrong when the track's topics are only waiting on
		// another track. Offer its entry points: a learner may practise a
		// locked topic, and mastering it credits what it builds on.
		if jump := e.jumpIns(track, e.conceptSnapshotsFrom(studentID, progress), 3); len(jump) > 0 {
			res.Primary = &jump[0]
			res.Alternatives = append(jump[1:], res.Alternatives...)
		}
	}
	e.explainRecommendations(&res, studentID, progress, remedial, urgent, dueIDs, now)
	return res, nil
}

// pacedOrder is the followed courses' plans merged by pace
// (planning.PlansOrderPaced): the order new topics are offered in, in Up
// next and in the mixed session alike. Nil for a learner on no planned course.
func (e *Engine) pacedOrder(studentID string, progress map[string]*storage.ConceptProgress) map[string]int {
	if e.planner == nil || studentID == "" {
		return nil
	}
	st, err := e.repo.GetStudent(studentID)
	if err != nil || st == nil || st.CourseID == "" {
		return nil
	}
	done := func(id string) bool {
		p := progress[id]
		return p != nil && mastery.EffectiveStatus(mastery.Status(p.Status), daysSinceReview(p.LastReviewed), mastery.DecayDays) == mastery.StatusMastered
	}
	return e.planner.PlansOrderPaced(ActiveCourses(st.CourseID), done, time.Now())
}

// courseShortName drops the ": study plan" a course name ends with.
func courseShortName(name string) string {
	name = strings.TrimSpace(name)
	for _, suf := range []string{": study plan", ": directed reading"} {
		name = strings.TrimSuffix(name, suf)
	}
	return name
}

// urgentTopics are the plan topics still to do of each followed course due
// within scheduler.UrgentDays (or past its date), with why: they may come
// before due reviews (scheduler.RecommendInput.Urgent).
func (e *Engine) urgentTopics(studentID string, progress map[string]*storage.ConceptProgress, now time.Time) map[string]string {
	if e.planner == nil || studentID == "" {
		return nil
	}
	st, err := e.repo.GetStudent(studentID)
	if err != nil || st == nil {
		return nil
	}
	out := map[string]string{}
	for _, id := range ActiveCourses(st.CourseID) {
		c := e.planner.Course(id)
		due, ok := c.DueDate()
		if !ok {
			continue
		}
		days := int(due.Sub(now).Hours()/24 + 0.999)
		if days > scheduler.UrgentDays {
			continue
		}
		why := courseShortName(c.Name) + " is due today"
		switch {
		case days < 0:
			why = courseShortName(c.Name) + " was due " + c.Due
		case days == 1:
			why = courseShortName(c.Name) + " is due tomorrow"
		case days > 1:
			why = fmt.Sprintf("%s is due in %d days", courseShortName(c.Name), days)
		}
		late := c.AfterDueTopics()
		for _, t := range c.Plan {
			p := progress[t]
			if late[t] || (p != nil && mastery.Status(p.Status) == mastery.StatusMastered) {
				continue
			}
			if _, ok := out[t]; !ok {
				out[t] = why
			}
		}
	}
	return out
}

// jumpIns lists a track's unfinished concepts whose prerequisites inside the
// track are done, so only outside prerequisites hold them back: the places a
// learner can start in this track today.
func (e *Engine) jumpIns(track string, snaps map[string]*scheduler.ConceptSnapshot, max int) []scheduler.Recommendation {
	done := func(id string) bool {
		s := snaps[id]
		return s != nil && (s.Placement || s.Status == mastery.StatusMastered)
	}
	var out []scheduler.Recommendation
	for _, c := range e.TrackDAG(track).Order() {
		if len(out) >= max {
			break
		}
		if done(c.ID) {
			continue
		}
		var missing []string
		ready := true
		for _, pid := range c.Prerequisites {
			if done(pid) {
				continue
			}
			if e.InTrack(pid, track) {
				ready = false
				break
			}
			if p := e.dag.Concept(pid); p != nil {
				missing = append(missing, p.Label)
			}
		}
		if !ready || len(missing) == 0 {
			continue
		}
		if len(missing) > 2 {
			missing = append(missing[:2], fmt.Sprintf("%d more", len(missing)-2))
		}
		out = append(out, scheduler.Recommendation{
			ID:           c.ID,
			ConceptID:    c.ID,
			ConceptTitle: c.Label,
			Kind:         scheduler.KindLearn,
			Reason:       scheduler.ReasonNew,
			Action:       scheduler.Action{Type: scheduler.KindLearn, Href: "/learn?concept=" + c.ID},
			Badge:        "Jump in",
			Detail:       "Builds on " + strings.Join(missing, ", ") + ". You can start here: mastering it credits them.",
			CTA:          "Start →",
		})
	}
	return out
}

// labelBridges says why a concept from another track is on this track's
// shelf: it names the track concept it unlocks.
func (e *Engine) labelBridges(res *scheduler.RecommendationResponse, track string) {
	label := func(r *scheduler.Recommendation) {
		if r == nil || !e.IsBridge(r.ConceptID, track) {
			return
		}
		r.Badge = "Needed next"
		for _, d := range e.dag.DependentsOf(r.ConceptID) {
			if e.InTrack(d.ID, track) {
				r.Detail = "Comes before " + d.Label + " in this track"
				return
			}
		}
	}
	label(res.Primary)
	for i := range res.Alternatives {
		label(&res.Alternatives[i])
	}
}

func daysSinceReview(t *time.Time) float64 {
	if t == nil || t.IsZero() {
		return 999
	}
	return time.Since(*t).Hours() / 24
}

// RecommendPaused is what a paused learner is told. Reviews are hidden while paused (ADR-022
// and handleDueReviews agree on that), so a recommendation pointing at /review would
// contradict the pause the learner just set.
//
// It used to point at `/study` instead, with "Lessons are still open" — true when it was
// written, and a dead end once the library closed: a learner who opened the app on a break was
// sent to a page that could only explain things they had already decided not to practise today.
// `/settings` is where the pause and its end both live, so it is the one destination that can
// actually answer "I am not doing reviews — now what".
func (e *Engine) RecommendPaused(studentID string) scheduler.RecommendationResponse {
	return scheduler.RecommendationResponse{
		Primary: &scheduler.Recommendation{
			ID:     "paused",
			Kind:   scheduler.KindLearn,
			Reason: scheduler.ReasonNew,
			Action: scheduler.Action{Type: scheduler.KindLearn, Href: "/settings"},
			Badge:  "Paused",
			Detail: "Reviews are on hold until you resume — due reviews are hidden while paused",
			CTA:    "Change or resume →",
		},
		Alternatives: []scheduler.Recommendation{},
		GeneratedAt:  time.Now().UTC(),
	}
}
