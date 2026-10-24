package planning

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/mastery"
	"github.com/chuma-beep/mathua/internal/storage"
)

type Course struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Grade       string   `json:"grade"`
	Description string   `json:"description"`
	Targets     []string `json:"targets"`
	// Plan, when set, is the course's study plan: its topics in the order
	// they are taught. The course's path keeps that order, with each topic's
	// unmet prerequisites slotted in just before it, and the recommender
	// offers new topics in path order. A course with a plan and no targets
	// targets its plan.
	Plan []string `json:"plan,omitempty"`
	// Texts are the course's authority texts, in priority order, as
	// "<module>/<file>" (fork change). A topic reached through the course
	// opens its reading in the first of these that covers it.
	Texts []string `json:"texts,omitempty"`
	// TopicTexts puts particular texts first for particular topics (fork
	// change): analysis topics read Abbott even inside the economics course.
	TopicTexts map[string][]string `json:"topic_texts,omitempty"`
	// Term names when the course runs ("Fall 2026"); shown on its card.
	Term string `json:"term,omitempty"`
	// Due is the date the course must be done by ("2026-12-06": the final,
	// the test date), fork change. Courses with a due date share the study
	// queue in proportion to the pace each needs (topics left per day left).
	Due string `json:"due,omitempty"`
	// Modules split the plan into the units the course is taught in (weeks,
	// chapters), each naming a slice of plan topics (fork change). A course
	// without modules is shown as one unit.
	Modules []Module `json:"modules,omitempty"`
}

// TextsFor is the course's texts in priority order for one topic: its
// topic-specific texts first, then the course's texts.
func (c *Course) TextsFor(conceptID string) []string {
	if c == nil {
		return nil
	}
	if extra := c.TopicTexts[conceptID]; len(extra) > 0 {
		out := append([]string(nil), extra...)
		return append(out, c.Texts...)
	}
	return c.Texts
}

// Module is one teaching unit of a course: a title and the plan topics it
// covers, in order. Topics is a subset of the course's plan.
type Module struct {
	Title  string   `json:"title"`
	Topics []string `json:"topics"`
}

type Path struct {
	Course   *Course
	Concepts []*concepts.Concept
}

type Planner struct {
	dag      *concepts.DAG
	courses  []*Course
	byID     map[string]*Course
	destByID map[string]*Destination
}

func Load(coursesPath string, dag *concepts.DAG, extra ...string) (*Planner, error) {
	var courses []*Course
	for i, path := range append([]string{coursesPath}, extra...) {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read courses: %w", err)
		}
		var batch []*Course
		if err := json.Unmarshal(data, &batch); err != nil {
			return nil, fmt.Errorf("parse courses %s: %w", path, err)
		}
		for _, c := range batch {
			if len(c.Targets) == 0 {
				c.Targets = append([]string(nil), c.Plan...)
			}
		}
		if i > 0 {
			// A module's course must stand on the graph this install has; a
			// target from a disabled module would otherwise plan nothing.
			for _, c := range batch {
				for _, t := range append(append([]string(nil), c.Targets...), c.Plan...) {
					if dag != nil && dag.Concept(t) == nil {
						return nil, fmt.Errorf("course %q in %s targets unknown concept %q", c.ID, path, t)
					}
				}
			}
		}
		courses = append(courses, batch...)
	}
	byID := make(map[string]*Course, len(courses))
	for _, c := range courses {
		if _, dup := byID[c.ID]; dup {
			return nil, fmt.Errorf("course id %q is defined twice", c.ID)
		}
		byID[c.ID] = c
	}
	return &Planner{dag: dag, courses: courses, byID: byID}, nil
}

// New returns a Planner over the DAG without a course catalog.
// PrerequisitesOf works catalog-free; course lookups stay empty.
func New(dag *concepts.DAG) *Planner {
	return &Planner{dag: dag, byID: map[string]*Course{}}
}

func (p *Planner) Courses() []*Course {
	return p.courses
}

func (p *Planner) Course(id string) *Course {
	return p.byID[id]
}

func (p *Planner) collectChain(targets []string) (map[string]bool, error) {
	inChain := make(map[string]bool)
	var queue []string
	for _, t := range targets {
		if p.dag.Concept(t) == nil {
			return nil, fmt.Errorf("target concept %q not found in DAG", t)
		}
		queue = append(queue, t)
	}
	for len(queue) > 0 {
		cid := queue[0]
		queue = queue[1:]
		if inChain[cid] {
			continue
		}
		inChain[cid] = true
		for _, prereq := range p.dag.PrereqsOf(cid) {
			if !inChain[prereq.ID] {
				queue = append(queue, prereq.ID)
			}
		}
	}
	return inChain, nil
}

func (p *Planner) PrerequisitesOf(conceptIDs []string) (*Path, error) {
	inChain, err := p.collectChain(conceptIDs)
	if err != nil {
		return nil, err
	}
	var chain []*concepts.Concept
	for _, c := range p.dag.Order() {
		if inChain[c.ID] {
			chain = append(chain, c)
		}
	}
	return &Path{
		Course:   &Course{Targets: conceptIDs},
		Concepts: chain,
	}, nil
}

func (p *Planner) PrerequisiteChain(targets []string) (*Path, error) {
	return p.PrerequisitesOf(targets)
}

func (p *Planner) PathForCourse(courseID string) (*Path, error) {
	c := p.byID[courseID]
	if c == nil {
		return nil, fmt.Errorf("course %q not found", courseID)
	}
	if len(c.Plan) > 0 {
		return p.planPath(c)
	}
	path, err := p.PrerequisitesOf(c.Targets)
	if err != nil {
		return nil, err
	}
	path.Course = c
	return path, nil
}

// planPath is a planned course's path: each plan topic in order, preceded by
// those of its prerequisites (in graph order) not already on the path, then
// anything else the course targets.
func (p *Planner) planPath(c *Course) (*Path, error) {
	order := p.dag.Order()
	seen := make(map[string]bool)
	var out []*concepts.Concept
	add := func(ids []string) error {
		chain, err := p.collectChain(ids)
		if err != nil {
			return err
		}
		for _, cc := range order {
			if chain[cc.ID] && !seen[cc.ID] {
				seen[cc.ID] = true
				out = append(out, cc)
			}
		}
		return nil
	}
	for _, id := range c.Plan {
		if err := add([]string{id}); err != nil {
			return nil, err
		}
	}
	if err := add(c.Targets); err != nil {
		return nil, err
	}
	return &Path{Course: c, Concepts: out}, nil
}

// PlansOrder ranks the paths of several planned courses together. Their
// positions interleave (a course's k-th topic ranks beside every other
// course's k-th), so each followed course keeps getting turns on the shelf
// instead of the first one being finished before the next starts. A concept
// on several paths takes its best rank. Courses without a plan add nothing.
func (p *Planner) PlansOrder(courseIDs []string) map[string]int {
	var orders []map[string]int
	for _, id := range courseIDs {
		if o := p.PlanOrder(id); o != nil {
			orders = append(orders, o)
		}
	}
	if len(orders) == 0 {
		return nil
	}
	if len(orders) == 1 {
		return orders[0]
	}
	rank := map[string]int{}
	for k, o := range orders {
		for id, r := range o {
			v := r*len(orders) + k
			if old, ok := rank[id]; !ok || v < old {
				rank[id] = v
			}
		}
	}
	// Re-number densely so planBonus spreads over the combined length.
	ids := make([]string, 0, len(rank))
	for id := range rank {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return rank[ids[i]] < rank[ids[j]] })
	for i, id := range ids {
		rank[id] = i
	}
	return rank
}

// DueDate parses Due; ok is false when the course has none.
func (c *Course) DueDate() (time.Time, bool) {
	if c == nil || c.Due == "" {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", c.Due)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// DefaultHorizonDays is the pace horizon for a course with no due date: it
// is paced as if due in this many days, so it keeps a share of the queue
// without crowding out a course whose date is near.
const DefaultHorizonDays = 120

// PlansOrderPaced merges the courses' path orders by pace (fork change): each
// course's k-th remaining topic is placed at time k / pace, where pace is the
// topics per day the course needs to finish by its due date, so a course that
// is behind for its date gets proportionally more of the queue. A mastered
// topic keeps its rank but is not counted as remaining. isDone says which
// topics are mastered; now is the pacing date.
func (p *Planner) PlansOrderPaced(courseIDs []string, isDone func(id string) bool, now time.Time) map[string]int {
	type entry struct {
		id string
		t  float64
	}
	var all []entry
	n := 0
	for _, cid := range courseIDs {
		o := p.PlanOrder(cid)
		if o == nil {
			continue
		}
		n++
		c := p.byID[cid]
		ids := make([]string, 0, len(o))
		for id := range o {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return o[ids[i]] < o[ids[j]] })
		remaining := 0
		for _, id := range ids {
			if isDone == nil || !isDone(id) {
				remaining++
			}
		}
		days := float64(DefaultHorizonDays)
		if due, ok := c.DueDate(); ok {
			days = due.Sub(now).Hours() / 24
			if days < 7 {
				days = 7 // a course past or at its date still gets a full-tilt pace, not a division by zero
			}
		}
		pace := float64(remaining) / days
		if pace <= 0 {
			pace = 1.0 / days
		}
		k := 0
		for _, id := range ids {
			if isDone == nil || !isDone(id) {
				k++
			}
			all = append(all, entry{id, float64(k) / pace})
		}
	}
	if n == 0 {
		return nil
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].t < all[j].t })
	rank := make(map[string]int, len(all))
	for _, e := range all {
		if _, ok := rank[e.id]; !ok {
			rank[e.id] = len(rank)
		}
	}
	return rank
}

// PlanOrder ranks a planned course's path: concept id to its position. It is
// nil for an unknown course or one without a plan.
func (p *Planner) PlanOrder(courseID string) map[string]int {
	c := p.byID[courseID]
	if c == nil || len(c.Plan) == 0 {
		return nil
	}
	path, err := p.planPath(c)
	if err != nil {
		return nil
	}
	rank := make(map[string]int, len(path.Concepts))
	for i, cc := range path.Concepts {
		rank[cc.ID] = i
	}
	return rank
}

func (p *Planner) Unmastered(path *Path, progress map[string]*storage.ConceptProgress) []*concepts.Concept {
	var out []*concepts.Concept
	for _, c := range path.Concepts {
		if prog, ok := progress[c.ID]; !ok || prog.Status != string(mastery.StatusMastered) {
			out = append(out, c)
		}
	}
	return out
}

func (p *Planner) WeakAreas(path *Path, progress map[string]*storage.ConceptProgress) []*concepts.Concept {
	return p.Unmastered(path, progress)
}

func (p *Planner) Readiness(path *Path, progress map[string]*storage.ConceptProgress) float64 {
	if len(path.Concepts) == 0 {
		return 1.0
	}
	mastered := 0
	for _, c := range path.Concepts {
		if prog, ok := progress[c.ID]; ok && prog.Status == string(mastery.StatusMastered) {
			mastered++
		}
	}
	return float64(mastered) / float64(len(path.Concepts))
}

// CourseProgress is the accreditation-track view of one course.
type CourseProgress struct {
	Total         int      `json:"total"`
	Mastered      int      `json:"mastered"`
	Pct           float64  `json:"pct"`
	Remaining     []string `json:"remaining,omitempty"`
	DaysRemaining int      `json:"days_remaining"` // estimate at daily_xp_goal
}

// ProgressForCourse computes mastery over the course's transitive path.
func (p *Planner) ProgressForCourse(course *Course, progress map[string]*storage.ConceptProgress, dailyXPGoal int) (*CourseProgress, error) {
	path, err := p.PathForCourse(course.ID)
	if err != nil {
		return nil, err
	}
	total := len(path.Concepts)
	mastered := 0
	var remaining []string
	for _, c := range path.Concepts {
		if prog, ok := progress[c.ID]; ok && prog.Status == string(mastery.StatusMastered) {
			mastered++
		} else {
			remaining = append(remaining, c.ID)
		}
	}
	days := 0
	if goal := dailyXPGoal; goal > 0 && len(remaining) > 0 {
		// Estimate: ~5 concepts per 30 XP of focused work (MA ~1 min/XP).
		perDay := goal / 6
		if perDay < 1 {
			perDay = 1
		}
		days = (len(remaining) + perDay - 1) / perDay
	}
	pct := 0.0
	if total > 0 {
		pct = float64(mastered) / float64(total)
	}
	return &CourseProgress{
		Total:         total,
		Mastered:      mastered,
		Pct:           pct,
		Remaining:     remaining,
		DaysRemaining: days,
	}, nil
}

func (p *Planner) SortCourses(courses []*Course) {
	sort.Slice(courses, func(i, j int) bool {
		return courses[i].ID < courses[j].ID
	})
}
