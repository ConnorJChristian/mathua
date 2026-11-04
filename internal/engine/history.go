package engine

import (
	"sync"

	"github.com/chuma-beep/mathua/internal/books"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
	"github.com/chuma-beep/mathua/internal/storage"
)

// Fork change: what a learner has done with a topic's bank items and proofs,
// for items that unlock after another is answered correctly and for serving
// book exercises first (generator.History).
//
// The attempts table records every graded answer with the served answer key
// (expected) and question. A bank item's key names the item ("math:<concept>#
// <item>", "proof:<id>"), so it identifies the item across serves and across
// the topics a card is filed under; a multiple-choice key names the option,
// so for those the question text is compared too (generator.QuestionKey).
// The rows are read once per served question and only when the topic has a
// gated or book item to decide about: Repository.ItemOutcomes groups them in
// the database over the (student, concept) index.

// itemHistory loads one learner's outcomes on one topic on first use.
type itemHistory struct {
	repo      storage.Repository
	studentID string
	conceptID string

	once    sync.Once
	byKey   map[string]bool // answer key -> any correct
	byQuest map[string]bool // mc answer key + "\x00" + question key -> any correct

	stepOnce  sync.Once
	stepTop   int     // highest scaffold step with a right generated answer
	stepTried [4]bool // steps with any generated attempt

	bookCap int // generator.BookStepCapper

	// The topic's book policy (generator.BookPolicyHistory): from courses
	// when set (a lesson opened from a course), else from the learner's
	// followed courses that list the topic, read on first use.
	eng        *Engine
	courses    []string
	policyOnce sync.Once
	policy     generator.BookPolicy
}

// BookPolicy implements generator.BookPolicyHistory.
func (h *itemHistory) BookPolicy() generator.BookPolicy {
	h.policyOnce.Do(func() {
		ids := h.courses
		if ids == nil && h.eng != nil {
			ids = h.eng.followedCoursesListing(h.studentID, h.conceptID)
		}
		h.policy = h.eng.bookPolicy(h.conceptID, ids)
	})
	return h.policy
}

// followedCoursesListing is the learner's followed courses that list a
// topic, in the order they follow them.
func (e *Engine) followedCoursesListing(studentID, conceptID string) []string {
	if e == nil || e.planner == nil || e.repo == nil {
		return nil
	}
	st, err := e.repo.GetStudent(studentID)
	if err != nil || st == nil {
		return nil
	}
	var out []string
	for _, id := range ActiveCourses(st.CourseID) {
		if c := e.planner.Course(id); c.Lists(conceptID) {
			out = append(out, id)
		}
	}
	return out
}

// bookPolicy is what courses (the first is the one that brought the
// learner here) say about a topic's books: their texts for it in order,
// as book ids; the smallest parallel_book_cap they set, else
// MATHUA_PARALLEL_BOOK_CAP; and the books they exclude, less their texts.
func (e *Engine) bookPolicy(conceptID string, courseIDs []string) generator.BookPolicy {
	p := generator.BookPolicy{ParallelCap: generator.EnvParallelBookCap()}
	if e == nil || e.planner == nil {
		return p
	}
	seen := map[string]bool{}
	capSet := false
	for _, id := range courseIDs {
		c := e.planner.Course(id)
		if c == nil {
			continue
		}
		for _, tx := range c.TextsFor(conceptID) {
			if b := books.ID(tx); b != "" && !seen[b] {
				seen[b] = true
				p.Texts = append(p.Texts, b)
			}
		}
		if c.ParallelBookCap != nil {
			v := *c.ParallelBookCap
			if v < 0 {
				v = 0
			}
			if !capSet || (v > 0 && (p.ParallelCap == 0 || v < p.ParallelCap)) {
				p.ParallelCap = v
			}
			capSet = true
		}
		for _, x := range c.ParallelExcludeBooks {
			if b := books.ID(x); b != "" {
				if p.Exclude == nil {
					p.Exclude = map[string]bool{}
				}
				p.Exclude[b] = true
			}
		}
	}
	for b := range seen {
		delete(p.Exclude, b)
	}
	return p
}

// BookStepCap implements generator.BookStepCapper: the topic's book step
// cap, from the courses that list it or MATHUA_BOOK_STEP_CAP.
func (h *itemHistory) BookStepCap() int { return h.bookCap }

// bookStepCap is a topic's book step cap: the smallest book_step_cap of the
// courses that list it, else MATHUA_BOOK_STEP_CAP or its default.
func (e *Engine) bookStepCap(conceptID string) int {
	e.bookCapsOnce.Do(func() { e.bookCaps = e.planner.BookStepCaps() })
	if v, ok := e.bookCaps[conceptID]; ok {
		return v
	}
	return generator.EnvBookStepCap()
}

// stepWindow is how many recent attempts on a topic the scaffold reads.
const stepWindow = 200

// GeneratedCorrectAt implements generator.StepHistory: a right answer to
// one of the topic's generated questions (not a bank item or a proof) at
// this scaffold step or above, by the difficulty it was served at.
func (h *itemHistory) GeneratedCorrectAt(step int) bool {
	h.loadSteps()
	return h.stepTop >= step
}

// GeneratedAttemptedAt implements generator.StepAttemptHistory: any answer,
// right or wrong, to one of the topic's generated questions at this step.
func (h *itemHistory) GeneratedAttemptedAt(step int) bool {
	h.loadSteps()
	return step >= 0 && step < len(h.stepTried) && h.stepTried[step]
}

func (h *itemHistory) loadSteps() {
	h.stepOnce.Do(func() {
		rows, err := h.repo.GetRecentAttemptsForConcept(h.studentID, h.conceptID, stepWindow)
		if err != nil {
			return
		}
		for _, r := range rows {
			if r.Difficulty == nil || grader.IsBankAnswer(r.Expected) || grader.IsProofAnswer(r.Expected) {
				continue
			}
			s := generator.StepOfDifficulty(*r.Difficulty)
			if s >= 0 && s < len(h.stepTried) {
				h.stepTried[s] = true
			}
			if r.Correct && s > h.stepTop {
				h.stepTop = s
			}
		}
	})
}

// LearnerHistory is the generator.History of one learner on one topic, or
// nil without a learner or a store.
func (e *Engine) LearnerHistory(studentID, conceptID string) generator.History {
	return e.LearnerHistoryIn(studentID, conceptID, nil)
}

// LearnerHistoryIn is LearnerHistory reached through courses (a lesson
// opened from a course page): their texts decide the topic's main book.
// Nil courses take the learner's followed courses that list the topic.
func (e *Engine) LearnerHistoryIn(studentID, conceptID string, courses []string) generator.History {
	h := newItemHistory(e.repo, studentID, conceptID)
	if ih, ok := h.(*itemHistory); ok {
		ih.bookCap = e.bookStepCap(conceptID)
		ih.eng, ih.courses = e, courses
	}
	return h
}

func newItemHistory(repo storage.Repository, studentID, conceptID string) generator.History {
	if repo == nil || studentID == "" || conceptID == "" {
		return nil
	}
	return &itemHistory{repo: repo, studentID: studentID, conceptID: conceptID, bookCap: generator.EnvBookStepCap()}
}

func (h *itemHistory) load() {
	h.byKey, h.byQuest = map[string]bool{}, map[string]bool{}
	rows, err := h.repo.ItemOutcomes(h.studentID, h.conceptID)
	if err != nil {
		return // nothing known: gated items stay hidden, book items come first
	}
	for _, r := range rows {
		if r.Question != "" {
			k := r.Expected + "\x00" + generator.QuestionKey(r.Question)
			h.byQuest[k] = h.byQuest[k] || r.Correct
			continue
		}
		h.byKey[r.Expected] = h.byKey[r.Expected] || r.Correct
	}
}

// Outcome implements generator.History.
func (h *itemHistory) Outcome(answer, question string) (attempted, correct bool) {
	h.once.Do(h.load)
	if question != "" {
		correct, attempted = h.byQuest[answer+"\x00"+generator.QuestionKey(question)]
		return attempted, correct
	}
	correct, attempted = h.byKey[answer]
	return attempted, correct
}
