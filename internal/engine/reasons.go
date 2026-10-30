package engine

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/chuma-beep/mathua/internal/mastery"
	"github.com/chuma-beep/mathua/internal/scheduler"
	"github.com/chuma-beep/mathua/internal/storage"
)

// Fork change: every recommendation and every served question says why it
// was chosen, in words a learner can check: "review due 3 days ago", "next
// in ECON 5153 · 7 Systems of Linear Equations", "pinned", "missed in the
// mastery check". The badge says what kind of task it is; the reason says
// what put it here. The replay audit (scripts/replay_audit) logs it too.

// planReasons maps each topic on a followed course's path to why the plan
// wants it: "next in <course> · <module>" for a plan topic, "needed for
// <plan topic> in <course>" for a prerequisite the plan pulls in. The first
// course to name a topic gives its reason.
func (e *Engine) planReasons(studentID string) map[string]string {
	out := map[string]string{}
	if e.planner == nil || studentID == "" {
		return out
	}
	st, err := e.repo.GetStudent(studentID)
	if err != nil || st == nil {
		return out
	}
	for _, id := range ActiveCourses(st.CourseID) {
		c := e.planner.Course(id)
		if c == nil {
			continue
		}
		name := courseShortName(c.Name)
		module := map[string]string{}
		for _, m := range c.Modules {
			for _, t := range m.Topics {
				if _, ok := module[t]; !ok {
					module[t] = m.Title
				}
			}
		}
		inPlan := map[string]bool{}
		for _, t := range c.Plan {
			inPlan[t] = true
		}
		for _, t := range c.Targets {
			inPlan[t] = true
		}
		path, err := e.planner.PathForCourse(id)
		if err != nil || path == nil {
			continue
		}
		var pending []string
		for _, cc := range path.Concepts {
			if !inPlan[cc.ID] {
				pending = append(pending, cc.ID)
				continue
			}
			for _, p := range pending {
				if _, ok := out[p]; !ok {
					out[p] = "needed for " + cc.Label + " in " + name
				}
			}
			pending = pending[:0]
			if _, ok := out[cc.ID]; !ok {
				if m := module[cc.ID]; m != "" {
					out[cc.ID] = "next in " + name + " · " + m
				} else {
					out[cc.ID] = "next in " + name
				}
			}
		}
		for _, p := range pending {
			if _, ok := out[p]; !ok {
				out[p] = "on the path of " + name
			}
		}
	}
	return out
}

// reviewDueReason words how overdue a review is.
func reviewDueReason(due *time.Time, now time.Time) string {
	if due == nil {
		return "review due: not practised for a while"
	}
	days := int(math.Floor(now.Sub(*due).Hours() / 24))
	switch {
	case days <= 0:
		return "review due today"
	case days == 1:
		return "review due 1 day ago"
	default:
		return fmt.Sprintf("review due %d days ago", days)
	}
}

// mostOverdue is the due review the review session serves first: the one
// furthest past its date, among the topics the learner is reviewed on.
func (e *Engine) mostOverdue(progress map[string]*storage.ConceptProgress, skip map[string]bool, now time.Time) (*storage.ConceptProgress, int) {
	var best *storage.ConceptProgress
	n := 0
	for _, p := range progress {
		if p == nil || skip[p.ConceptID] || p.NextReviewDue == nil || p.NextReviewDue.After(now) || mastery.Status(p.Status) != mastery.StatusMastered {
			continue
		}
		n++
		if best == nil || p.NextReviewDue.Before(*best.NextReviewDue) {
			best = p
		}
	}
	return best, n
}

// explainRecommendations fills Why on every recommendation.
func (e *Engine) explainRecommendations(res *scheduler.RecommendationResponse, studentID string, progress map[string]*storage.ConceptProgress, remedial map[string]bool, urgent map[string]string) {
	now := time.Now()
	plan := e.planReasons(studentID)
	skip := e.notReviewed(studentID, progress)
	explain := func(r *scheduler.Recommendation) {
		if r == nil || r.Why != "" {
			return
		}
		switch {
		case r.Kind == scheduler.KindReview:
			if p, n := e.mostOverdue(progress, skip, now); p != nil {
				r.Why = reviewDueReason(p.NextReviewDue, now)
				if n > 1 {
					r.Why += fmt.Sprintf(" (%d due)", n)
				}
			} else {
				r.Why = "review due"
			}
		case r.Kind == scheduler.KindMasteryCheck:
			r.Why = "mastery check due"
		case r.ID == "diagnostic":
			r.Why = "no placement yet"
		case r.Pinned && r.PinnedFor != "":
			r.Why = "pinned: needed for " + r.PinnedFor
		case r.Pinned:
			r.Why = "pinned"
		case urgent[r.ConceptID] != "" && r.Kind != scheduler.KindReview:
			r.Why = urgent[r.ConceptID]
			if p := plan[r.ConceptID]; p != "" {
				r.Why += " · " + p
			}
		case remedial[r.ConceptID]:
			r.Why = "missed in the mastery check"
		case r.Badge == "Needed next":
			r.Why = strings.ToLower(r.Detail[:1]) + r.Detail[1:]
		case r.Badge == "Jump in":
			r.Why = "jump in: its prerequisites are in another track"
		case plan[r.ConceptID] != "":
			r.Why = plan[r.ConceptID]
			if r.Kind == scheduler.KindPractice {
				if p := progress[r.ConceptID]; p != nil && p.Attempts > 0 {
					r.Why = "needs practice · " + r.Why
				}
			}
		case r.Kind == scheduler.KindPractice:
			r.Why = "needs practice"
		case r.ConceptID != "":
			r.Why = "ready to learn"
		}
	}
	explain(res.Primary)
	for i := range res.Alternatives {
		explain(&res.Alternatives[i])
	}
}

// questionReason says why a question was served in the mixed session or the
// review session.
func (e *Engine) questionReason(studentID, conceptID string, isReview bool, progress map[string]*storage.ConceptProgress) string {
	if isReview {
		if p := progress[conceptID]; p != nil {
			return reviewDueReason(p.NextReviewDue, time.Now())
		}
		return "review due"
	}
	for _, p := range e.Pins(studentID) {
		if p.Concept == conceptID {
			return "pinned"
		}
	}
	if r := e.planReasons(studentID)[conceptID]; r != "" {
		return r
	}
	return "ready to learn"
}
