package quiz

import (
	"sort"
	"sync"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/generator"
)

// Session is an actionable quiz every ~150 XP, own grading path (attemptID).
// Guest unlimited retake: each POST /api/quiz/session creates a new session.
type Session struct {
	sync.Mutex
	ID            string
	StudentID     string
	Order         []*concepts.Concept
	Index         int
	Attempts      []Attempt
	LastProblem   *generator.Problem
	LastConceptID string
	// Difficulties holds weakness→difficulty per concept (80% target, 0.3-1.0).
	Difficulties map[string]float64
	// Avoid is question text served in the learner's recent tests; a
	// regeneration is tried before serving one of them again (fork fix: the
	// same code task came up in three tests running).
	Avoid map[string]bool
	// Objective marks a course test (fork change): every question grades
	// itself and is quick (generator.GeneratorContext.ObjectiveOnly), so no
	// flashcard, recall card, code task or proof is served.
	Objective bool
	// CourseID and Kind name a course test ("review" or "full"); empty for
	// the mastery check.
	CourseID string
	Kind     string
}

type Attempt struct {
	ConceptID string
	Correct   bool
}

type Engine struct {
	dag      *concepts.DAG
	registry *generator.Registry
}

func NewEngine(dag *concepts.DAG, reg *generator.Registry) *Engine {
	return &Engine{dag: dag, registry: reg}
}

// Start picks diverse recent weak concepts via weakness hint, fallback to DAG order.
// Caller provides weakness-filtered concepts; we cap to 5 for stub.
func (e *Engine) Start(concepts []*concepts.Concept) *Session {
	return e.StartLimit(concepts, DefaultLimit)
}

// DefaultLimit is the mastery check's length.
const DefaultLimit = 5

// StartLimit is Start with a topic cap other than five (fork change: a
// course review test covers up to fifteen, a full-course review more). The
// empty fallback stays five topics.
func (e *Engine) StartLimit(concepts []*concepts.Concept, limit int) *Session {
	if limit <= 0 {
		limit = DefaultLimit
	}
	if len(concepts) == 0 {
		concepts = e.dag.Order()
		if len(concepts) > DefaultLimit {
			concepts = concepts[:DefaultLimit]
		}
	}
	if len(concepts) > limit {
		concepts = concepts[:limit]
	}
	return &Session{
		Order:        concepts,
		Difficulties: make(map[string]float64, len(concepts)),
	}
}

func (e *Engine) NextQuestion(s *Session) (*generator.Problem, string, error) {
	s.Lock()
	defer s.Unlock()
	for s.Index < len(s.Order) {
		c := s.Order[s.Index]
		// 80% difficulty target via weakness map (engine.DifficultyFor); fallback 0.8.
		difficulty := 0.8
		if d, ok := s.Difficulties[c.ID]; ok && d > 0 {
			difficulty = d
		}
		gctx := generator.GeneratorContext{Difficulty: difficulty, NoProofs: true}
		if s.Objective {
			gctx.NoCards, gctx.ObjectiveOnly = true, true
		}
		p, err := e.registry.GenerateContext(c.ID, gctx)
		if err != nil {
			return nil, "", err
		}
		for i := 0; i < 6 && s.Avoid[p.Question]; i++ {
			if p2, err2 := e.registry.GenerateContext(c.ID, gctx); err2 == nil && p2.Question != "" {
				p = p2
			}
		}
		if p.Question == "" {
			// Nothing this topic may serve in a course test: pass it by,
			// untested, as a skip would.
			s.Index++
			continue
		}
		s.LastProblem = &p
		s.LastConceptID = c.ID
		return s.LastProblem, c.ID, nil
	}
	s.LastProblem = nil
	return nil, "", nil
}

func (e *Engine) RecordAnswer(s *Session, conceptID string, correct bool) {
	s.Lock()
	defer s.Unlock()
	s.Attempts = append(s.Attempts, Attempt{ConceptID: conceptID, Correct: correct})
	// Advance index after answer
	if s.Index < len(s.Order) && s.Order[s.Index].ID == conceptID {
		s.Index++
	}
}

func (e *Engine) IsComplete(s *Session) bool {
	s.Lock()
	defer s.Unlock()
	return s.Index >= len(s.Order)
}

// HasAttempts reports whether any answer was recorded this session —
// used to avoid resetting the 150 XP gate baseline on an all-skipped quiz.
func (e *Engine) HasAttempts(s *Session) bool {
	s.Lock()
	defer s.Unlock()
	return len(s.Attempts) > 0
}

// SkipQuestion advances past the current question without recording an
// attempt — the skip escape hatch for questions that can't be answered.
// No grade, no XP, no remedial: the concept simply goes untested this quiz.
func (e *Engine) SkipQuestion(s *Session) {
	s.Lock()
	defer s.Unlock()
	if s.Index < len(s.Order) {
		s.Index++
	}
}

// PickQuizConcepts selects diverse recent weak concepts for quiz.
// Stub: sorts by weakness desc, picks top 5 distinct subdomains if possible.
func PickQuizConcepts(dag *concepts.DAG, weakness map[string]float64) []*concepts.Concept {
	type scored struct {
		c *concepts.Concept
		w float64
	}
	var list []scored
	for _, c := range dag.Order() {
		w := weakness[c.ID]
		if w > 0.3 {
			list = append(list, scored{c, w})
		}
	}
	if len(list) == 0 {
		// fallback to 5 available concepts
		order := dag.Order()
		if len(order) > 5 {
			order = order[:5]
		}
		return order
	}
	sort.Slice(list, func(i, j int) bool { return list[i].w > list[j].w })
	// distinct subdomain pick
	seen := map[string]bool{}
	var out []*concepts.Concept
	for _, s := range list {
		if !seen[s.c.Subdomain] {
			out = append(out, s.c)
			seen[s.c.Subdomain] = true
		}
		if len(out) >= 5 {
			break
		}
	}
	// fill up to 5 if still short
	for _, s := range list {
		if len(out) >= 5 {
			break
		}
		found := false
		for _, c := range out {
			if c.ID == s.c.ID {
				found = true
				break
			}
		}
		if !found {
			out = append(out, s.c)
		}
	}
	return out
}
