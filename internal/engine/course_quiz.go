package engine

import (
	"sort"
	"time"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/mastery"
)

// Fork change: the mastery test follows your courses.
//
// The stock quiz draws from the whole track by weakness, and every concept
// you have never touched counts as weak (0.5), so a quiz mostly sampled
// untaught topics from anywhere in the graph, and its misses queued remedial
// work off your plan. For a learner who follows courses, the quiz instead
// covers what those courses have taught them: topics on the followed courses'
// paths that they have actually worked on (placement alone does not count).
//
// Within that pool the weakest come first, then mastered topics longest since
// they were last seen (a mastery test is also a retention check). The picks
// rotate across the followed courses so each one is represented, and no more
// than two come from one subdomain while others are available.

// QuizConceptLimit is how many topics one mastery test covers.
const QuizConceptLimit = 5

// CourseQuizConcepts returns the topics a mastery test should cover for a
// learner who follows courses. ok is false when they follow none (the caller
// keeps the track-wide quiz). An empty list with ok true means nothing on
// their courses has been studied yet.
func (e *Engine) CourseQuizConcepts(studentID string) (picked []*concepts.Concept, courses []string, ok bool) {
	if e.planner == nil || studentID == "" {
		return nil, nil, false
	}
	st, err := e.repo.GetStudent(studentID)
	if err != nil || st == nil {
		return nil, nil, false
	}
	ids := ActiveCourses(st.CourseID)
	if len(ids) == 0 {
		return nil, nil, false
	}
	progress, err := e.repo.GetAllProgress(studentID)
	if err != nil {
		return nil, nil, false
	}

	type cand struct {
		c      *concepts.Concept
		weak   float64
		master bool
		seen   time.Time
		again  bool // missed on the last check: tested again only after the rest
	}
	lastMissed := map[string]bool{}
	for _, id := range e.QuizRemedial(studentID) {
		lastMissed[id] = true
	}
	var perCourse [][]cand
	taken := map[string]bool{}
	for _, id := range ids {
		path, err := e.planner.PathForCourse(id)
		if err != nil || path == nil {
			continue
		}
		title := id
		if path.Course != nil && path.Course.Name != "" {
			title = path.Course.Name
		}
		courses = append(courses, title)
		var list []cand
		for _, c := range path.Concepts {
			p := progress[c.ID]
			if p == nil || p.Attempts == 0 || p.PlacementSeeded || taken[c.ID] {
				continue
			}
			taken[c.ID] = true
			cd := cand{c: c, weak: p.WeaknessScore, master: p.Status == string(mastery.StatusMastered), again: lastMissed[c.ID]}
			if p.LastAttempted != nil {
				cd.seen = *p.LastAttempted
			}
			list = append(list, cd)
		}
		sort.SliceStable(list, func(i, j int) bool {
			a, b := list[i], list[j]
			// What the last check already caught out waits its turn, so a
			// second check covers what was learned since (fork fix: two
			// checks in a row asked the same five topics).
			if a.again != b.again {
				return !a.again
			}
			if a.master != b.master {
				return !a.master // still-learning topics first
			}
			if !a.master && a.weak != b.weak {
				return a.weak > b.weak
			}
			return a.seen.Before(b.seen) // longest unseen first
		})
		perCourse = append(perCourse, list)
	}

	// Round-robin across courses, at most two per subdomain while the pool
	// still has others to offer.
	sub := map[string]int{}
	pos := make([]int, len(perCourse))
	for pass := 0; pass < 2 && len(picked) < QuizConceptLimit; pass++ {
		progressed := true
		for progressed && len(picked) < QuizConceptLimit {
			progressed = false
			for i := range perCourse {
				for pos[i] < len(perCourse[i]) {
					cd := perCourse[i][pos[i]]
					if pass == 0 && sub[cd.c.Subdomain] >= 2 {
						pos[i]++
						continue
					}
					pos[i]++
					if containsConcept(picked, cd.c.ID) {
						continue
					}
					picked = append(picked, cd.c)
					sub[cd.c.Subdomain]++
					progressed = true
					break
				}
				if len(picked) >= QuizConceptLimit {
					break
				}
			}
		}
		// Second pass: lift the subdomain cap to fill what is left.
		for i := range pos {
			pos[i] = 0
		}
	}
	return picked, courses, true
}

func containsConcept(list []*concepts.Concept, id string) bool {
	for _, c := range list {
		if c.ID == id {
			return true
		}
	}
	return false
}
