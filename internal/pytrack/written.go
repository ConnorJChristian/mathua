package pytrack

import (
	"html"
	"regexp"
	"strings"

	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
)

// Written-answer recall cards (fork change).
//
// About a third of the imported flashcards ask for a proof, a derivation or a
// justification ("use the total differential to show that the MRS is a/b").
// Marking such a card yourself after reading a one-line back is easy to get
// wrong in your own favour. When a language-model judge is configured
// (internal/proof), such a card is served with a written-answer box instead of
// the reveal button and graded against a marking scheme generated from the
// card. The learner can still reveal the back and mark it themselves.
//
// The card stays a recall card throughout: its answer key keeps the "recall:"
// prefix, so it is practice exactly as a self-graded card is (never served in
// placement or a course test, and evidence only on the terms the proof grader
// sets; see engine.practiceOnly).

// Grading overrides on a recall item ("grading" in the bank file).
const (
	GradingLLM  = "llm"  // always a written-answer card
	GradingSelf = "self" // never: always self-graded
)

// WrittenMarker is placed in a written-answer card between the front and the
// recall marker. It is the proof grader's answer marker (internal/proof
// Marker), so every answer host shows its proof box, its longer timeout and
// its grade report; the recall marker after it tells the host that the back
// can be revealed and self-graded instead (web/next-app/lib/pythonQuestion.ts
// isWrittenRecall).
const WrittenMarker = "<!--proof-answer-->"

// RecallCard is what a written-answer grader needs to know about a card.
type RecallCard struct {
	// Key is "<concept>#<item>", the item's id in the bank.
	Key     string
	Concept string
	// Bank is the bank file the card is in, without ".json" ("anki_calculus").
	Bank   string
	Level  int
	Prompt string
	Back   string
}

// WrittenGrader grades written answers to recall cards. internal/proof
// installs one (SetWrittenGrader) when a judge is configured.
type WrittenGrader interface {
	// ServeWritten reports whether the card should be served for a written
	// answer now: grading is on for its bank, the month's budget is not spent
	// and its marking scheme has not failed to generate.
	ServeWritten(card RecallCard) bool
	// GradeWritten grades a written answer. A result it cannot give is
	// Unavailable, never a miss.
	GradeWritten(card RecallCard, answer string) grader.Result
}

var writtenGrader WrittenGrader

// SetWrittenGrader installs the written-answer grader; nil uninstalls it.
func SetWrittenGrader(w WrittenGrader) { writtenGrader = w }

// Classification is why a recall card is, or is not, a written-answer card.
type Classification struct {
	Written bool
	// Reason is the keyword that made it one ("prove", "show that"), the
	// override ("grading: llm"), or what excluded it ("show the graph").
	Reason string
}

var (
	tagRe       = regexp.MustCompile(`(?s)<[^>]*>`)
	mathRe      = regexp.MustCompile(`(?s)\\\(.*?\\\)|\\\[.*?\\\]|\$\$.*?\$\$`)
	quotedRe    = regexp.MustCompile(`“[^”]*”|"[^"]*"`)
	cropNoteRe  = regexp.MustCompile(`(?i)⚠?\s*graph auto-extracted[^.]*?(p\.\s*\d+)?`)
	spaceRe     = regexp.MustCompile(`\s+`)
	figureRe    = regexp.MustCompile(`\b(the|this|these) (figures?|diagrams?|graphs?|pictures?|charts?|plots?)( below| above)? (shows?|below|above|depicts?)\b|\bshown in the (figure|diagram|graph)\b|\bthe figures? (labeled|labelled)\b`)
	visualTask  = regexp.MustCompile(`\b(sketch|draw|graph|plot|schedule|table of values|phase diagram)\b`)
	showSubject = regexp.MustCompile(`\b(figures?|graphs?|tables?|diagrams?|results?|data|outputs?|plots?|charts?|panels?|screens?|they|which|will|would|does|do|can|may|might|to)\s+shows?\b`)
	showModal   = regexp.MustCompile(`\b(have|has|need|needs|must|would|should|ought) to show\b|\bwhat (would|do|does|must|should) [^.?!]{0,40}\bshow\b`)
	showThatRe  = regexp.MustCompile(`\bshow\b[^.?!;:]{0,80}?\bthat\b`)
	showBareRe  = regexp.MustCompile(`\bshow\s+(\S+)`)
	showVisual  = regexp.MustCompile(`\bshow\s+(your|all|the|each|its|their)?\s*(work|working|steps|calculations?|graphs?|plots?|diagrams?|figures?|pictures?|sketch|tables?|on (a|the) (graph|diagram|number line))\b`)
	orShowRe    = regexp.MustCompile(`\bor show\b`)
	proveRe     = regexp.MustCompile(`\b(dis)?prove\b`)
	verifyRe    = regexp.MustCompile(`\bverify\b`)
	verifyThat  = regexp.MustCompile(`\bverify that\b`)
	confirmRe   = regexp.MustCompile(`\bconfirm\b`)
	argueRe     = regexp.MustCompile(`\b(derive|justify|demonstrate|explain why)\b`)
	verifyNoise = regexp.MustCompile(`\bverify (the )?crop\b|\bverify (it |this |your answer )?(numerically|graphically|with a (graph|calculator|plot)|on a (graph|calculator))\b`)
)

// showFollowers are the words after a bare "show" that make it a request for
// an argument: "show algebraically", "show how", "show, by evaluating".
var showFollowers = map[string]bool{
	"algebraically": true, "analytically": true, "mathematically": true, "formally": true,
	"rigorously": true, "directly": true, "explicitly": true, "carefully": true,
	"how": true, "why": true, "it": true, "there": true, "this": true, "by": true,
	"using": true, "with": true, "from": true, "both": true, "without": true,
	"math": true, "every": true, "any": true, "whether": true,
}

// plainPrompt is the prompt as words: tags, entities and mathematics removed,
// quoted titles dropped, lower case.
func plainPrompt(s string) string {
	s = strings.NewReplacer("<br>", " ", "<br/>", " ", "<br />", " ", "</div>", " ", "</p>", " ").Replace(s)
	s = tagRe.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	s = cropNoteRe.ReplaceAllString(s, " ")
	s = mathRe.ReplaceAllString(s, " MATH ")
	s = quotedRe.ReplaceAllString(s, " ")
	s = strings.ToLower(spaceRe.ReplaceAllString(s, " "))
	return s
}

// Classify reports whether a recall card asks for a written proof,
// derivation or justification. A "grading" override in the bank wins. Only
// recall cards are ever written-answer cards.
func Classify(it *Item) Classification {
	if it.Kind != KindRecall {
		return Classification{Reason: "not a recall card"}
	}
	switch it.Grading {
	case GradingLLM:
		return Classification{Written: true, Reason: "grading: llm"}
	case GradingSelf:
		return Classification{Reason: "grading: self"}
	}
	// The judge reads text only: a front with a picture, or a back that is
	// mostly a picture, cannot be marked from what it can see.
	if strings.Contains(it.Prompt, "<img") {
		return Classification{Reason: "has an image"}
	}
	if strings.Contains(it.Back, "<img") && len(strings.TrimSpace(tagRe.ReplaceAllString(it.Back, ""))) < 40 {
		return Classification{Reason: "the back is a picture"}
	}
	p := plainPrompt(it.Prompt)
	if figureRe.MatchString(p) {
		return Classification{Reason: "refers to a figure"}
	}
	excluded := ""
	note := func(r string) {
		if excluded == "" {
			excluded = r
		}
	}
	if m := proveRe.FindString(p); m != "" {
		return Classification{Written: true, Reason: m}
	}
	if m := argueRe.FindString(p); m != "" {
		return Classification{Written: true, Reason: m}
	}
	if verifyRe.MatchString(p) {
		if verifyNoise.MatchString(p) && !verifyThat.MatchString(p) {
			note("verify graphically")
		} else {
			return Classification{Written: true, Reason: "verify"}
		}
	}
	if confirmRe.MatchString(p) {
		if visualTask.MatchString(p) {
			note("confirm from a graph or table")
		} else {
			return Classification{Written: true, Reason: "confirm"}
		}
	}
	// "show": an imperative asking for an argument, not "the figures show",
	// "what would you have to show", "show your work", "show the graph", or
	// "compute it or show it does not exist".
	if strings.Contains(p, "show") {
		if visualTask.MatchString(p) || strings.Contains(p, "graphically") {
			return Classification{Reason: "show on a graph"}
		}
		q := showSubject.ReplaceAllString(p, " ")
		q = showModal.ReplaceAllString(q, " ")
		if showVisual.MatchString(q) {
			note(strings.TrimSpace(showVisual.FindString(q)))
			q = showVisual.ReplaceAllString(q, " ")
		}
		if orShowRe.MatchString(q) {
			note("or show")
			q = orShowRe.ReplaceAllString(q, " ")
		}
		if showThatRe.MatchString(q) {
			return Classification{Written: true, Reason: "show that"}
		}
		for _, m := range showBareRe.FindAllStringSubmatch(q, -1) {
			w := strings.Trim(m[1], ",.;:")
			if showFollowers[w] || strings.HasPrefix(m[1], ",") || strings.HasSuffix(m[1], ",") {
				return Classification{Written: true, Reason: "show " + w}
			}
		}
		if strings.Contains(q, "show") {
			note("show (not a request for an argument)")
		}
	}
	return Classification{Reason: excluded}
}

// card is an item as the written-answer grader sees it.
func (t *Track) card(c *Concept, it *Item) RecallCard {
	return RecallCard{
		Key:     itemKey(c.ID, it.ID),
		Concept: c.ID,
		Bank:    t.Bank.BankOf(c.ID),
		Level:   it.Level,
		Prompt:  strings.TrimSpace(it.Prompt),
		Back:    strings.TrimSpace(it.Back),
	}
}

// Cards lists every recall card in the bank with its classification, in bank
// order, for the operator's tools (mathua proof classify / schemes).
func (b *Bank) Cards() []ClassifiedCard {
	var out []ClassifiedCard
	for _, ch := range b.chapters {
		for ci := range ch.Concepts {
			c := b.concepts[ch.Concepts[ci].ID]
			for i := range c.Items {
				it := &c.Items[i]
				if it.Kind != KindRecall {
					continue
				}
				out = append(out, ClassifiedCard{
					RecallCard: RecallCard{Key: itemKey(c.ID, it.ID), Concept: c.ID, Bank: b.BankOf(c.ID), Level: it.Level,
						Prompt: strings.TrimSpace(it.Prompt), Back: strings.TrimSpace(it.Back)},
					Classification: Classify(it),
				})
			}
		}
	}
	return out
}

// ClassifiedCard is a recall card and its classification.
type ClassifiedCard struct {
	RecallCard
	Classification
}

// writtenProblem renders a written-answer card: the front, the written-answer
// marker, then the recall marker and the back for the self-grade escape.
func writtenProblem(p generator.Problem, it *Item) generator.Problem {
	p.Question = strings.TrimSpace(it.Prompt) + "\n\n" + WrittenMarker + "\n\n" + RecallMarker + "\n\n" + strings.TrimSpace(it.Back)
	return p
}

// PlainRecall turns a written-answer card back into a plain recall card (a
// quiz serves no written answers). Any other problem is returned unchanged.
func PlainRecall(p generator.Problem) generator.Problem {
	if strings.HasPrefix(p.Answer, prefixRecall) && strings.Contains(p.Question, WrittenMarker) {
		p.Question = strings.Replace(p.Question, WrittenMarker+"\n\n", "", 1)
		p.Question = strings.Replace(p.Question, WrittenMarker, "", 1)
	}
	return p
}

// servesWritten reports whether this card is served for a written answer now.
func (t *Track) servesWritten(c *Concept, it *Item) bool {
	if it.Kind != KindRecall || writtenGrader == nil || !Classify(it).Written {
		return false
	}
	return writtenGrader.ServeWritten(t.card(c, it))
}
