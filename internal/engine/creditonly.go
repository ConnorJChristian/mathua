package engine

import "github.com/chuma-beep/mathua/internal/storage"

// creditOnlyOffCourse names the topics a learner holds on credit alone and no
// followed course asks for: no answer of their own on the topic (Attempts is
// zero), only a placement seed or partial credit from the topics above it,
// and the topic is not one of a followed course's own topics (its plan or
// targets). The prerequisites a plan rests on are not its own topics: the
// course page does not list a placed prerequisite as work either.
//
// These are the grade-school foundations an early diagnostic put a graduate
// learner above (two-digit addition under an economics course). They are
// reachability, not something the learner studied, so they are not reviewed
// and not suggested as new topics: review would pad the queue with material
// nobody chose, and decay (a placement-seeded row ages like any other) would
// otherwise bring dozens of them back at once. A pin still brings one back.
//
// The rule applies only to a learner who follows a course. Without one there
// is no course to say what is wanted, and the upstream behaviour stands.
func (e *Engine) creditOnlyOffCourse(studentID string, progress map[string]*storage.ConceptProgress) map[string]bool {
	if e.planner == nil || studentID == "" {
		return nil
	}
	st, err := e.repo.GetStudent(studentID)
	if err != nil || st == nil {
		return nil
	}
	ids := ActiveCourses(st.CourseID)
	if len(ids) == 0 {
		return nil
	}
	own := map[string]bool{}
	for _, id := range ids {
		c := e.planner.Course(id)
		if c == nil {
			continue
		}
		for _, cid := range c.Plan {
			own[cid] = true
		}
		for _, cid := range c.Targets {
			own[cid] = true
		}
	}
	if len(own) == 0 {
		return nil
	}
	pinned := map[string]bool{}
	for _, p := range e.Pins(studentID) {
		pinned[p.Concept] = true
	}
	out := map[string]bool{}
	for cid, p := range progress {
		if p == nil || p.Attempts > 0 || own[cid] || pinned[cid] {
			continue
		}
		if p.PlacementSeeded || p.ImplicitCredit > 0 {
			out[cid] = true
		}
	}
	return out
}
