package generator

import "math/rand"

// Fork change: the order a lesson serves its questions in (FORK.md, "The
// order of practice inside a topic").
//
// A topic's questions climb three scaffold steps, as Math Academy's do:
//
//  1. apply the rule: one step, one idea (bank level 1, or a generated
//     question at the easy end of the ladder);
//  2. worked cases: the book's examples and exercises, several steps
//     (level 2);
//  3. applications: word problems, economic settings, proofs (level 3).
//
// The learner works the lowest step not yet done; a step is done once one
// of its questions has a right answer and every book exercise at it has
// been tried. Within a step the book's own exercises come first, in book
// order, then the bank's other items (untried ones first, at random, so
// variants interleave) and the topic's generated questions. An item that
// unlocks after another is never in the list until it is unlocked. The
// step comes from the learner's record, not from the Learn ladder's
// difficulty, which starts over at every visit: a learner who left off at
// step 3 resumes there, and one who misses stays at the step until a right
// answer (two misses in a row send them to the prerequisites as before).
// Generated questions are served at their step's difficulty. Once every
// step is done, practice and reviews sample across the steps by
// difficulty, as before.

// StepOfLevel is the scaffold step of a bank item's level.
func StepOfLevel(level int) int {
	switch {
	case level == 1:
		return 1
	case level >= 3:
		return 3
	default:
		return 2 // level 2, or unset
	}
}

// StepOfDifficulty is the step a generator difficulty (0.3-1.0) asks for.
// The Learn ladder starts at 0.4 (step 1) and climbs 0.15 per right answer.
func StepOfDifficulty(d float64) int {
	switch {
	case d <= 0.45:
		return 1
	case d < 0.75:
		return 2
	default:
		return 3
	}
}

// DifficultyOfStep is the generator difficulty a step serves at.
func DifficultyOfStep(step int) float64 {
	switch step {
	case 1:
		return 0.35
	case 3:
		return 0.9
	default:
		return 0.6
	}
}

// StepHistory is implemented by a History that knows the steps at which
// the learner has answered the topic's generated questions right.
type StepHistory interface {
	// GeneratedCorrectAt reports a right answer to a generated question of
	// this topic at step or above.
	GeneratedCorrectAt(step int) bool
}

// ScaffoldSource is a bank generator that lists its servable items for a
// learner as scaffold steps (internal/pytrack).
type ScaffoldSource interface {
	ScaffoldItems(ctx GeneratorContext) []ScaffoldItem
}

// ScaffoldItemSource is a bank generator that describes one of its items,
// by id, as a scaffold step: for an item filed under another topic.
type ScaffoldItemSource interface {
	ItemScaffold(itemID string, ctx GeneratorContext) (ScaffoldItem, bool)
}

// ScaffoldItem is one question a topic can serve.
type ScaffoldItem struct {
	Step      int
	Book      bool
	Attempted bool
	Correct   bool
	Serve     func() (Problem, bool)
}

// Scaffold is everything a topic can serve to one learner now.
type Scaffold struct {
	// Items are the servable bank items, in bank order.
	Items []ScaffoldItem
	// Generated serves the topic's generated question at a step, nil for a
	// topic that is a bank only.
	Generated func(step int) Problem
	// GeneratedDone[s] is a right generated answer at step s or above.
	GeneratedDone [4]bool
	// Share is the chance a bank item is served rather than a generated
	// question when a step has both.
	Share float64
}

// has reports which steps have something to serve.
func (s *Scaffold) has() [4]bool {
	var h [4]bool
	for _, it := range s.Items {
		h[it.Step] = true
	}
	if s.Generated != nil {
		h[1], h[2], h[3] = true, true, true
	}
	return h
}

// Done reports whether a step is done: a right answer at it or above, and
// every book exercise at it tried.
func (s *Scaffold) Done(step int) bool {
	right := s.GeneratedDone[step]
	for _, it := range s.Items {
		if it.Correct && it.Step >= step {
			right = true
		}
		if it.Book && it.Step == step && !it.Attempted {
			return false
		}
	}
	return right
}

// Frontier is the lowest step with something to serve that is not done,
// or 0 when every step is.
func (s *Scaffold) Frontier() int {
	h := s.has()
	for step := 1; step <= 3; step++ {
		if h[step] && !s.Done(step) {
			return step
		}
	}
	return 0
}

// Pick serves the next question of a lesson at the frontier step. served
// is what this request has already served (a
// lesson asks for several questions at once): nothing is served twice, and
// when a step has nothing left the lesson goes on to the next step. ok is
// false when every step is done: the caller then samples across the steps
// as it always has.
func (s *Scaffold) Pick(served map[string]bool) (Problem, bool) {
	step := s.Frontier()
	if step == 0 {
		return Problem{}, false
	}
	for ; step <= 3; step++ {
		if p, ok := s.pickAt(step, served); ok {
			return p, true
		}
	}
	return Problem{}, false
}

// pickAt serves one question at a step that this request has not served:
// the book's exercises first, in book order (an untried one, or a missed
// one half the time, as generator.PickBook), then the bank's other items,
// untried first and at random, and the generated questions.
func (s *Scaffold) pickAt(step int, served map[string]bool) (Problem, bool) {
	try := func(it ScaffoldItem) (Problem, bool) {
		p, ok := it.Serve()
		if !ok || p.Question == "" || served[p.Question] {
			return Problem{}, false
		}
		return p, true
	}
	var untriedBooks, missedBooks, fresh, others []ScaffoldItem
	for _, it := range s.Items {
		if it.Step != step {
			continue
		}
		switch {
		case it.Book && !it.Correct && !it.Attempted:
			untriedBooks = append(untriedBooks, it)
		case it.Book && !it.Correct:
			// A missed exercise comes back half the time, and is in the
			// pool with the rest the other half.
			missedBooks = append(missedBooks, it)
			others = append(others, it)
		case it.Book:
			others = append(others, it)
		case !it.Attempted:
			fresh = append(fresh, it)
		default:
			others = append(others, it)
		}
	}
	for _, it := range untriedBooks {
		if p, ok := try(it); ok {
			return p, true
		}
	}
	if len(missedBooks) > 0 && rand.Intn(2) == 0 {
		for _, i := range rand.Perm(len(missedBooks)) {
			if p, ok := try(missedBooks[i]); ok {
				return p, true
			}
		}
	}
	gen := func() (Problem, bool) {
		if s.Generated == nil {
			return Problem{}, false
		}
		for i := 0; i < 4; i++ {
			if p := s.Generated(step); p.Question != "" && !served[p.Question] {
				return p, true
			}
		}
		return Problem{}, false
	}
	bankFirst := s.Generated == nil || rand.Float64() < s.Share
	if !bankFirst {
		if p, ok := gen(); ok {
			return p, true
		}
	}
	for _, pool := range [][]ScaffoldItem{fresh, others} {
		for _, i := range rand.Perm(len(pool)) {
			if p, ok := try(pool[i]); ok {
				return p, true
			}
		}
	}
	if p, ok := gen(); ok {
		return p, true
	}
	for _, it := range missedBooks {
		if p, ok := try(it); ok {
			return p, true
		}
	}
	return Problem{}, false
}
