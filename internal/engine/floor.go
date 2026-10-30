package engine

import (
	"os"
	"sort"
	"strings"

	"github.com/chuma-beep/mathua/internal/mastery"
	"github.com/chuma-beep/mathua/internal/planning"
	"github.com/chuma-beep/mathua/internal/storage"
)

// Fork change: the learner's floor.
//
// A graduate learner kept being served grade-school topics: two-digit
// addition as a review, "What is a variable" in the mixed session, fraction
// topics on the mastery check, a placement test opening on the distance
// between two points. Each surface had its own guard, or none, and a guard
// that hung on one environment variable came back the day it was not set.
//
// The floor is one rule every surface asks: the domains a learner is past.
// For a learner who follows courses it comes from the courses themselves
// (planning.Course.Floor: a course above school grades is past the
// grade-school domains), so it needs no setting. A topic below the floor is
// never served to them as a review, as new work in Up next or the mixed
// session, as remediation, on the mastery check or in a placement test,
// unless a followed course lists it as its own topic or the learner pins it.
// It still counts as met where a topic above it needs it: a floor never
// locks course material.
//
// MATHUA_REVIEW_FLOOR_DOMAINS overrides the course floor with its own list
// ("none" or "off" switches the floor off). For a learner who follows no
// course it keeps its older, narrower meaning: their mastered topics in
// those domains are not reviewed, and nothing else changes.

// FloorScope says how far a floor reaches.
const (
	// FloorAll: nothing below the floor is served (a learner on courses).
	FloorAll = "all"
	// FloorReviews: mastered topics below the floor are not reviewed (an
	// environment floor for a learner on no course).
	FloorReviews = "reviews"
	// FloorNone: no floor.
	FloorNone = "none"
)

// FloorInfo is the learner's active floor, for the API and the startup log.
type FloorInfo struct {
	Domains []string `json:"domains"`
	// Source is "env" (MATHUA_REVIEW_FLOOR_DOMAINS), "courses" (the followed
	// courses' floors) or "none".
	Source string `json:"source"`
	Scope  string `json:"scope"`
}

// EnvFloor reads MATHUA_REVIEW_FLOOR_DOMAINS: the domains, and whether the
// variable is set at all ("none" and "off" set it to no domains).
func EnvFloor() ([]string, bool) {
	raw := strings.TrimSpace(os.Getenv("MATHUA_REVIEW_FLOOR_DOMAINS"))
	if raw == "" {
		return nil, false
	}
	if l := strings.ToLower(raw); l == "none" || l == "off" {
		return []string{}, true
	}
	var out []string
	for _, d := range strings.Split(raw, ",") {
		if d = strings.TrimSpace(d); d != "" {
			out = append(out, d)
		}
	}
	sort.Strings(out)
	return out, true
}

// FloorStartupLine is the server's startup log line about the floor, so the
// floor in force is visible without a learner (cmd/mathua).
func FloorStartupLine() string {
	if env, set := EnvFloor(); set {
		if len(env) == 0 {
			return "off (MATHUA_REVIEW_FLOOR_DOMAINS=none)"
		}
		return strings.Join(env, ",") + " (MATHUA_REVIEW_FLOOR_DOMAINS; for a learner on no course, reviews only)"
	}
	return "per learner from their courses: a course above school grades is past " +
		strings.Join(planning.GradeSchoolDomains, ",") + " (GET /api/floor)"
}

// learnerFloor is the learner's floor and the topics it never hides: the
// followed courses' own topics (plan and targets) and the learner's pins.
func (e *Engine) learnerFloor(studentID string) (FloorInfo, map[string]bool) {
	keep := map[string]bool{}
	var courses []*planning.Course
	if e.repo != nil && studentID != "" {
		for _, p := range e.Pins(studentID) {
			keep[p.Concept] = true
		}
		if e.planner != nil {
			if st, err := e.repo.GetStudent(studentID); err == nil && st != nil {
				for _, id := range ActiveCourses(st.CourseID) {
					if c := e.planner.Course(id); c != nil {
						courses = append(courses, c)
						for _, cid := range c.Plan {
							keep[cid] = true
						}
						for _, cid := range c.Targets {
							keep[cid] = true
						}
					}
				}
			}
		}
	}
	scope := FloorReviews
	if len(courses) > 0 {
		scope = FloorAll
	}
	if env, set := EnvFloor(); set {
		if len(env) == 0 {
			return FloorInfo{Domains: []string{}, Source: "env", Scope: FloorNone}, keep
		}
		return FloorInfo{Domains: env, Source: "env", Scope: scope}, keep
	}
	if len(courses) == 0 {
		return FloorInfo{Domains: []string{}, Source: "none", Scope: FloorNone}, keep
	}
	// Several courses: a domain is below the floor only when every followed
	// course is past it (a school-grade course alongside a graduate one
	// keeps its domains servable).
	count := map[string]int{}
	for _, c := range courses {
		seen := map[string]bool{}
		for _, d := range c.Floor() {
			if !seen[d] {
				seen[d] = true
				count[d]++
			}
		}
	}
	var domains []string
	for d, n := range count {
		if n == len(courses) {
			domains = append(domains, d)
		}
	}
	sort.Strings(domains)
	if len(domains) == 0 {
		return FloorInfo{Domains: []string{}, Source: "courses", Scope: FloorNone}, keep
	}
	return FloorInfo{Domains: domains, Source: "courses", Scope: FloorAll}, keep
}

// Floor is the learner's active floor (GET /api/floor, the Settings page).
func (e *Engine) Floor(studentID string) FloorInfo {
	f, _ := e.learnerFloor(studentID)
	return f
}

// belowFloor names the topics the floor hides from a learner. With scope
// FloorAll that is every topic of a floor domain the learner's courses do
// not list and they have not pinned, whatever its status; with
// FloorReviews, only the mastered ones (they are not reviewed).
func (e *Engine) belowFloor(studentID string, progress map[string]*storage.ConceptProgress) map[string]bool {
	f, keep := e.learnerFloor(studentID)
	if f.Scope == FloorNone || len(f.Domains) == 0 {
		return nil
	}
	floor := map[string]bool{}
	for _, d := range f.Domains {
		floor[d] = true
	}
	out := map[string]bool{}
	if f.Scope == FloorAll {
		for _, c := range e.dag.Order() {
			if floor[c.Domain] && !keep[c.ID] {
				out[c.ID] = true
			}
		}
		for id := range e.deepFoundations(studentID, progress, keep) {
			out[id] = true
		}
		return out
	}
	for cid, p := range progress {
		if p == nil || keep[cid] || mastery.Status(p.Status) != mastery.StatusMastered {
			continue
		}
		if c := e.dag.Concept(cid); c != nil && floor[c.Domain] {
			out[cid] = true
		}
	}
	return out
}

// deepFoundations names the topics a course learner is past by the
// evidence of their placement (Q57): prerequisites more than
// CoursePlacementDepth links below every topic of every followed course,
// which no followed course lists and the learner has never practised, once
// the learner has placement credit or mastery on a topic of a followed
// course.
//
// A course placement never asks these (placementKeep), and a correct answer
// credits only what lies below it, so a prerequisite chain under a course
// topic the test did not reach stayed open as new work: after an ECON 5153 +
// Abbott placement, "Power sets and counting subsets" pulled in the mean of
// a data set and school probability (stat.data.mean, stat.prob.basic,
// complement, compound), six links down, and they took a quarter of the
// first hundred answers. The domain floor does not reach them (statistics
// and algebra are not grade-school domains). Like the floor, they still count
// as met, so they lock nothing, and a pin or a course that lists one serves
// it as usual.
func (e *Engine) deepFoundations(studentID string, progress map[string]*storage.ConceptProgress, keep map[string]bool) map[string]bool {
	if e.planner == nil || e.repo == nil || studentID == "" {
		return nil
	}
	st, err := e.repo.GetStudent(studentID)
	if err != nil || st == nil {
		return nil
	}
	var own []string
	for _, id := range ActiveCourses(st.CourseID) {
		if c := e.planner.Course(id); c != nil {
			own = append(own, c.Plan...)
			own = append(own, c.Targets...)
		}
	}
	if len(own) == 0 {
		return nil
	}
	if progress == nil {
		if progress, err = e.repo.GetAllProgress(studentID); err != nil {
			return nil
		}
	}
	evidence := false
	for _, id := range own {
		if p := progress[id]; p != nil && (mastery.Status(p.Status) == mastery.StatusMastered ||
			(p.PlacementSeeded && mastery.Status(p.Status) != mastery.StatusUnseen)) {
			evidence = true
			break
		}
	}
	if !evidence {
		return nil
	}
	depth := map[string]int{}
	queue := make([]string, 0, len(own))
	for _, id := range own {
		if _, ok := depth[id]; !ok && e.dag.Concept(id) != nil {
			depth[id] = 0
			queue = append(queue, id)
		}
	}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		for _, p := range e.dag.PrereqsOf(id) {
			if _, ok := depth[p.ID]; !ok {
				depth[p.ID] = depth[id] + 1
				queue = append(queue, p.ID)
			}
		}
	}
	out := map[string]bool{}
	for id, d := range depth {
		if d <= CoursePlacementDepth || keep[id] {
			continue
		}
		// A foundation the learner has practised stays theirs: it is
		// reviewed and offered as before (a deep prerequisite of one course
		// can be a topic the learner studied for another: the central limit
		// theorem under a data course). Only untouched ones are past.
		if p := progress[id]; p != nil && p.Attempts > 0 {
			continue
		}
		out[id] = true
	}
	return out
}

// FloorMet is floorMet for tools (scripts/replay_audit): the topics that
// count as met prerequisites because the learner is past them.
func (e *Engine) FloorMet(studentID string) map[string]bool { return e.floorMet(studentID) }

// floorMet names the topics that count as met prerequisites because the
// learner is past them (scope FloorAll only): a floor never locks what is
// above it.
func (e *Engine) floorMet(studentID string) map[string]bool {
	f, _ := e.learnerFloor(studentID)
	if f.Scope != FloorAll {
		return nil
	}
	return e.belowFloor(studentID, nil)
}

// CoursePlacementDepth bounds how far below a course's own topics a course
// placement reaches (fork change): its own topics (depth 0), their direct
// prerequisites (1) and the prerequisites of those (2). A correct answer
// still seeds everything below it (ApplyGoalResults), so nothing deeper
// needs asking; asking it is how a course placement opened on grade-school
// geometry.
const CoursePlacementDepth = 2

// placementKeep says which topics a placement for these goal topics may
// ask: within CoursePlacementDepth links below the goal and not below the
// learner's floor. depthBound false keeps the depth unbounded (the
// onboarding placement of a learner on no course).
func (e *Engine) placementKeep(studentID string, goal []string, depthBound bool) func(id string) bool {
	below := e.belowFloor(studentID, nil)
	f, _ := e.learnerFloor(studentID)
	if f.Scope != FloorAll {
		below = nil // an environment floor on no course only limits reviews
	}
	var depth map[string]int
	if depthBound {
		depth = map[string]int{}
		queue := make([]string, 0, len(goal))
		for _, id := range goal {
			if _, ok := depth[id]; !ok {
				depth[id] = 0
				queue = append(queue, id)
			}
		}
		for len(queue) > 0 {
			id := queue[0]
			queue = queue[1:]
			if depth[id] >= CoursePlacementDepth {
				continue
			}
			for _, p := range e.dag.PrereqsOf(id) {
				if _, ok := depth[p.ID]; !ok {
					depth[p.ID] = depth[id] + 1
					queue = append(queue, p.ID)
				}
			}
		}
	}
	return func(id string) bool {
		if below[id] {
			return false
		}
		if depth != nil {
			if _, ok := depth[id]; !ok {
				return false
			}
		}
		return true
	}
}
