package engine

import (
	"fmt"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/diagnostic"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/mastery"
	"github.com/chuma-beep/mathua/internal/storage"
)

// Fork change: per-course placement and review tests.
//
// A course offers two tests. The placement test is the adaptive diagnostic
// scoped to one course: its own topics and everything they rest on, so a
// learner can skip in-course material they already know as well as the
// foundations under it. The review test is the mastery check scoped to one
// course: topics that course has taught the learner, weakest first, spread
// across its sections.
//
// Both ask only questions the server grades by itself and that are quick to
// answer (generator.GeneratorContext.ObjectiveOnly): generated questions, and
// from a question bank only multiple choice, "what does this print" and typed
// math. Recall cards are graded by the learner, code tasks take minutes and a
// sandbox, and proofs are neither quick nor, by default, evidence.
//
// Viability. A test built from three topics measures little and reads as
// broken, so a course offers a test only when it has at least
// MinGradableTopics topics with such questions:
//
//   - review: the course's own topics (plan, targets and module topics) that
//     are objectively gradable. A course without a prerequisite chain, such
//     as a reading course or a handbook with a multiple-choice bank, can be
//     reviewed: review needs only questions, not structure.
//   - placement: the objectively gradable topics on the course's path (its
//     own topics plus their prerequisites), and the path must have at least
//     one prerequisite edge. Placement works by opening what a right answer
//     rests on; on a path without edges every topic is already open, so a
//     placement test could unlock nothing.

// MinGradableTopics is how many objectively gradable topics a course needs
// before it offers a test.
const MinGradableTopics = 8

// CoursePlacementMaxQuestions caps one course placement test.
const CoursePlacementMaxQuestions = 30

// ReviewTestLimit is how many topics one course review test covers, and
// FullReviewTestLimit the cap of a full-course review (final-exam prep).
const (
	ReviewTestLimit     = 15
	FullReviewTestLimit = 25
)

// objectiveItemsOf is a bank generator's ObjectiveItems, or 0 for a
// generator that does not count them. Only consulted for bank topics.
func objectiveItemsOf(g generator.Generator) int {
	if oc, ok := g.(generator.ObjectiveItemCounter); ok {
		return oc.ObjectiveItems()
	}
	return 0
}

// isBankTopic reports a question-bank topic (internal/pytrack).
func isBankTopic(c *concepts.Concept) bool {
	return c.GradingType == "bank" || c.GradingType == "python"
}

// ObjectivelyGradable reports whether a topic can serve a test question: a
// generated topic, or a bank topic with at least one item a test may serve.
func (e *Engine) ObjectivelyGradable(c *concepts.Concept) bool {
	if c == nil || e.registry == nil {
		return false
	}
	g, err := e.registry.Get(c.ID)
	if err != nil {
		return false
	}
	if isBankTopic(c) {
		return objectiveItemsOf(g) > 0
	}
	return true
}

// CourseSection is one teaching unit of a course: a module, the plan topics
// in no module, or (Prereq) the path's prerequisite fill-ins.
type CourseSection struct {
	Title  string
	Topics []*concepts.Concept
	Prereq bool
}

// CourseSections splits a course's path into its sections, in course order:
// its modules, then plan topics in no module ("Also on the plan", or
// "Topics" for a course without modules), then the prerequisites. Every
// topic on the path appears once. This is the course page's layout
// (server/courseview.go).
func (e *Engine) CourseSections(courseID string) ([]CourseSection, error) {
	if e.planner == nil {
		return nil, fmt.Errorf("no planner configured")
	}
	c := e.planner.Course(courseID)
	if c == nil {
		return nil, fmt.Errorf("course %q not found", courseID)
	}
	path, err := e.planner.PathForCourse(courseID)
	if err != nil {
		return nil, err
	}
	own := map[string]bool{}
	for _, id := range c.Plan {
		own[id] = true
	}
	for _, id := range c.Targets {
		own[id] = true
	}
	placed := map[string]bool{}
	var out []CourseSection
	for _, m := range c.Modules {
		sec := CourseSection{Title: m.Title}
		for _, id := range m.Topics {
			if cc := e.dag.Concept(id); cc != nil && !placed[id] {
				placed[id] = true
				sec.Topics = append(sec.Topics, cc)
			}
		}
		out = append(out, sec)
	}
	restTitle := "Topics"
	if len(c.Modules) > 0 {
		restTitle = "Also on the plan"
	}
	rest := CourseSection{Title: restTitle}
	pre := CourseSection{Title: "Prerequisites", Prereq: true}
	for _, cc := range path.Concepts {
		if placed[cc.ID] {
			continue
		}
		placed[cc.ID] = true
		if own[cc.ID] {
			rest.Topics = append(rest.Topics, cc)
		} else {
			pre.Topics = append(pre.Topics, cc)
		}
	}
	if len(rest.Topics) > 0 {
		out = append(out, rest)
	}
	if len(pre.Topics) > 0 {
		out = append(out, pre)
	}
	return out, nil
}

// CourseTestInfo says which tests a course offers a learner and how long
// they are. The course view serves it as "tests".
type CourseTestInfo struct {
	Placement bool `json:"placement"`
	Review    bool `json:"review"`
	// GradableTopics: the course's own topics with objectively graded
	// questions (the review viability count).
	GradableTopics int `json:"gradable_topics"`
	// PlacementTopics: objectively gradable topics on the course's path,
	// prerequisites included (the placement viability count).
	PlacementTopics int `json:"placement_topics"`
	// PlacementQuestions: the most questions the placement test asks this
	// learner (topics they already have progress on are never asked).
	PlacementQuestions int `json:"placement_questions"`
	// ReviewTopics: gradable course topics the learner has studied, which is
	// what a review test can cover now.
	ReviewTopics int `json:"review_topics"`
	// ReviewQuestions and FullReviewQuestions: one question per topic.
	ReviewQuestions     int `json:"review_questions"`
	FullReviewQuestions int `json:"full_review_questions"`
}

// studiedForReview is the CourseQuizConcepts rule: worked on, not merely
// placed.
func studiedForReview(p *storage.ConceptProgress) bool {
	return p != nil && p.Attempts > 0 && !p.PlacementSeeded
}

// hasProgress reports a topic placement cannot change: ApplyGoalResults
// leaves every row above UNSEEN alone, so asking about it is wasted.
func hasProgress(p *storage.ConceptProgress) bool {
	return p != nil && p.Status != "" && p.Status != string(mastery.StatusUnseen)
}

// CourseTests computes a course's test offer for a learner ("" for a guest:
// viability only, nothing studied).
func (e *Engine) CourseTests(studentID, courseID string) (*CourseTestInfo, error) {
	secs, err := e.CourseSections(courseID)
	if err != nil {
		return nil, err
	}
	var progress map[string]*storage.ConceptProgress
	if studentID != "" {
		if p, err := e.repo.GetAllProgress(studentID); err == nil {
			progress = p
		}
	}
	info := &CourseTestInfo{}
	onPath := map[string]bool{}
	for _, sec := range secs {
		for _, c := range sec.Topics {
			onPath[c.ID] = true
		}
	}
	edges := false
	unknown := 0
	for _, sec := range secs {
		for _, c := range sec.Topics {
			for _, p := range c.Prerequisites {
				if onPath[p] {
					edges = true
				}
			}
			if !e.ObjectivelyGradable(c) {
				continue
			}
			info.PlacementTopics++
			if !hasProgress(progress[c.ID]) {
				unknown++
			}
			if sec.Prereq {
				continue
			}
			info.GradableTopics++
			if studiedForReview(progress[c.ID]) {
				info.ReviewTopics++
			}
		}
	}
	info.Placement = edges && info.PlacementTopics >= MinGradableTopics
	info.Review = info.GradableTopics >= MinGradableTopics
	if info.Placement {
		info.PlacementQuestions = minInt(unknown, CoursePlacementMaxQuestions)
	}
	if info.Review {
		info.ReviewQuestions = minInt(info.ReviewTopics, ReviewTestLimit)
		info.FullReviewQuestions = minInt(info.ReviewTopics, FullReviewTestLimit)
	}
	return info, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// StartCourseDiagnostic starts a course placement test: the diagnostic over
// the objectively gradable topics on the course's path, sampling the
// course's own topics into the cover, at most CoursePlacementMaxQuestions
// questions, never asking a topic the learner already has progress on.
// Results go through ApplyGoalResults like every placement.
func (e *Engine) StartCourseDiagnostic(studentID, courseID string) (*diagnostic.Session, *Question, error) {
	secs, err := e.CourseSections(courseID)
	if err != nil {
		return nil, nil, err
	}
	var progress map[string]*storage.ConceptProgress
	if studentID != "" {
		if p, err := e.repo.GetAllProgress(studentID); err == nil {
			progress = p
		}
	}
	var path []*concepts.Concept
	var own []string
	known := map[string]bool{}
	for _, sec := range secs {
		for _, c := range sec.Topics {
			if !e.ObjectivelyGradable(c) {
				continue
			}
			path = append(path, c)
			if !sec.Prereq {
				own = append(own, c.ID)
			}
			if hasProgress(progress[c.ID]) {
				known[c.ID] = true
			}
		}
	}
	// The diagnostic walks its path in graph order (StartWithPath).
	rank := map[string]int{}
	for i, c := range e.dag.Order() {
		rank[c.ID] = i
	}
	sortConcepts(path, rank)
	session := e.diag.StartCourse(path, own, known, CoursePlacementMaxQuestions)
	session.CourseID = courseID
	if session.State == diagnostic.StateDone {
		return session, nil, nil
	}
	return e.firstDiagnosticQuestion(session)
}

func sortConcepts(list []*concepts.Concept, rank map[string]int) {
	for i := 1; i < len(list); i++ {
		for j := i; j > 0 && rank[list[j].ID] < rank[list[j-1].ID]; j-- {
			list[j], list[j-1] = list[j-1], list[j]
		}
	}
}

// firstDiagnosticQuestion stages a new session's first question.
func (e *Engine) firstDiagnosticQuestion(session *diagnostic.Session) (*diagnostic.Session, *Question, error) {
	prob, cid, err := e.diag.NextQuestion(session)
	if err != nil {
		return nil, nil, err
	}
	if prob == nil || cid == "" {
		return session, nil, nil
	}
	label := cid
	if c := e.dag.Concept(cid); c != nil {
		label = c.Label
	}
	return session, &Question{
		ConceptID:   cid,
		ConceptName: label,
		Question:    prob.Question,
		GradingType: e.gradingTypeFor(cid),
	}, nil
}

// CourseReviewConcepts picks a course review test: studied, objectively
// gradable topics of the course itself, spread across its sections, at most
// ReviewTestLimit (FullReviewTestLimit for a full-course review).
func (e *Engine) CourseReviewConcepts(studentID, courseID string, full bool) ([]*concepts.Concept, error) {
	if e.planner == nil || e.planner.Course(courseID) == nil {
		return nil, fmt.Errorf("course %q not found", courseID)
	}
	limit := ReviewTestLimit
	if full {
		limit = FullReviewTestLimit
	}
	picked, _, _ := e.CourseQuizConceptsFor(studentID, QuizScope{
		Courses:   []string{courseID},
		Limit:     limit,
		BySection: true,
		Gradable:  true,
	})
	return picked, nil
}

// TopicRef names a topic in a test summary.
type TopicRef struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// SectionResult is one course section's line in a test summary. Passed are
// topics answered right (placement: placed; review: held), Missed those
// answered wrong (placement: start here; review: queued for review), and
// Inferred, placement only, topics placed without being asked because a
// right answer rested on them.
type SectionResult struct {
	Title    string     `json:"title"`
	Total    int        `json:"total"`
	Passed   []TopicRef `json:"passed"`
	Missed   []TopicRef `json:"missed"`
	Inferred []TopicRef `json:"inferred,omitempty"`
}

// CourseTestSummary groups a test's outcomes by course section. results is
// each tested topic's last verdict; inferred are placed topics not asked.
// Sections with nothing to report are left out.
func (e *Engine) CourseTestSummary(courseID string, results map[string]bool, inferred map[string]bool) []SectionResult {
	secs, err := e.CourseSections(courseID)
	if err != nil {
		return nil
	}
	out := []SectionResult{}
	for _, sec := range secs {
		r := SectionResult{Title: sec.Title, Total: len(sec.Topics), Passed: []TopicRef{}, Missed: []TopicRef{}}
		for _, c := range sec.Topics {
			ref := TopicRef{ID: c.ID, Label: c.Label}
			if ok, tested := results[c.ID]; tested {
				if ok {
					r.Passed = append(r.Passed, ref)
				} else {
					r.Missed = append(r.Missed, ref)
				}
			} else if inferred[c.ID] {
				r.Inferred = append(r.Inferred, ref)
			}
		}
		if len(r.Passed)+len(r.Missed)+len(r.Inferred) > 0 {
			out = append(out, r)
		}
	}
	return out
}

// ApplyCoursePlacement applies a course placement test through
// ApplyGoalResults and reports it per course section: what was placed by a
// right answer, what was placed below one, and what was missed.
func (e *Engine) ApplyCoursePlacement(studentID string, session *diagnostic.Session) ([]SectionResult, error) {
	session.Lock()
	courseID := session.CourseID
	attempts := make([]diagnostic.Attempt, len(session.Attempts))
	copy(attempts, session.Attempts)
	session.Unlock()
	before, err := e.repo.GetAllProgress(studentID)
	if err != nil {
		return nil, fmt.Errorf("load progress: %w", err)
	}
	if err := e.ApplyGoalResults(studentID, session); err != nil {
		return nil, err
	}
	if courseID == "" {
		return nil, nil
	}
	after, err := e.repo.GetAllProgress(studentID)
	if err != nil {
		return nil, fmt.Errorf("load progress: %w", err)
	}
	results := map[string]bool{}
	for _, att := range attempts {
		results[att.ConceptID] = att.Correct
	}
	inferred := map[string]bool{}
	for id, p := range after {
		if _, tested := results[id]; tested || p == nil || !p.PlacementSeeded {
			continue
		}
		if b := before[id]; b == nil || !b.PlacementSeeded {
			inferred[id] = true
		}
	}
	return e.CourseTestSummary(courseID, results, inferred), nil
}
