package server

import (
	"strings"

	"github.com/chuma-beep/mathua/internal/engine"
	"github.com/chuma-beep/mathua/internal/grader"
	"github.com/chuma-beep/mathua/internal/walkthrough"
)

// Fork change (FSRS_PORT_PLAN 0c, internal/walkthrough): every graded answer
// comes back with one feedback contract, whatever surface asked. The engine
// carries the served explanation; this file turns it into the contract on
// the way out, with the topic's reading attached.

// walkthroughOf builds the contract for one graded question. expected is the
// served answer key, explanation the served explanation (steps included),
// learnerAnswer what was typed (empty for "I don't know"), diagnosis the
// grader's one-line diagnosis when it has one.
func (s *Server) walkthroughOf(conceptID, expected, explanation, learnerAnswer, diagnosis string) *walkthrough.Walkthrough {
	w := walkthrough.Build(shownKey(expected), explanation, learnerAnswer, diagnosis, s.readingFor(conceptID))
	return &w
}

// attachWalkthrough fills res.Walkthrough from the result's own fields.
func (s *Server) attachWalkthrough(res *engine.AnswerResult, learnerAnswer string, dontKnow bool) {
	if res == nil {
		return
	}
	if dontKnow {
		learnerAnswer = ""
	}
	res.Walkthrough = s.walkthroughOf(res.ConceptID, res.ExpectedAnswer, res.Explanation, learnerAnswer, res.Diagnosis)
}

// shownKey is the answer key as the learner reads it: a choice's text, a
// printed output, a tidied value. A typed-math, term, recall, code or proof
// key names the item, and its shown form is the explanation's own answer
// line, which Build lifts out.
func shownKey(expected string) string {
	switch {
	case strings.HasPrefix(expected, "mc:"):
		return strings.TrimPrefix(expected, "mc:")
	case strings.HasPrefix(expected, "out:"):
		return strings.TrimPrefix(expected, "out:")
	case grader.IsBankAnswer(expected):
		return ""
	}
	return displayAnswer(expected)
}

// readingFor is the topic's first reading as a link, or nil. A bank item's
// own "Read more" line stands in when the module gives the topic no reading.
func (s *Server) readingFor(conceptID string) *walkthrough.Reading {
	if conceptID == "" {
		return nil
	}
	for _, o := range s.readingOptions(conceptID) {
		if o.URL == "" {
			continue
		}
		label := o.Book
		if o.Source != "" {
			if label != "" {
				label += " · "
			}
			label += o.Source
		}
		return &walkthrough.Reading{Label: label, URL: o.URL, Page: o.Page}
	}
	return nil
}
