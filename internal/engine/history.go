package engine

import (
	"sync"

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

	stepOnce sync.Once
	stepTop  int // highest scaffold step with a right generated answer
}

// stepWindow is how many recent attempts on a topic the scaffold reads.
const stepWindow = 200

// GeneratedCorrectAt implements generator.StepHistory: a right answer to
// one of the topic's generated questions (not a bank item or a proof) at
// this scaffold step or above, by the difficulty it was served at.
func (h *itemHistory) GeneratedCorrectAt(step int) bool {
	h.stepOnce.Do(func() {
		rows, err := h.repo.GetRecentAttemptsForConcept(h.studentID, h.conceptID, stepWindow)
		if err != nil {
			return
		}
		for _, r := range rows {
			if !r.Correct || r.Difficulty == nil || grader.IsBankAnswer(r.Expected) || grader.IsProofAnswer(r.Expected) {
				continue
			}
			if s := generator.StepOfDifficulty(*r.Difficulty); s > h.stepTop {
				h.stepTop = s
			}
		}
	})
	return h.stepTop >= step
}

// LearnerHistory is the generator.History of one learner on one topic, or
// nil without a learner or a store.
func (e *Engine) LearnerHistory(studentID, conceptID string) generator.History {
	return newItemHistory(e.repo, studentID, conceptID)
}

func newItemHistory(repo storage.Repository, studentID, conceptID string) generator.History {
	if repo == nil || studentID == "" || conceptID == "" {
		return nil
	}
	return &itemHistory{repo: repo, studentID: studentID, conceptID: conceptID}
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
