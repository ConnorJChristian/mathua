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
// About a quarter of the imported flashcards ask for a proof, a derivation or a
// justification ("use the total differential to show that the MRS is a/b").
// Marking such a card yourself after reading a one-line back is easy to get
// wrong in your own favour. When a language-model judge is configured
// (internal/proof), such a card is served with a written-answer box instead of
// the reveal button and graded against a marking scheme generated from the
// card. The learner can still reveal the back and mark it themselves.
//
// The card stays a recall card throughout: its answer key keeps the "recall:"
// prefix, so the engine records it exactly as a self-graded card (practice,
// never mastery evidence, never a miss toward a halt; engine.practiceOnly),
// with the judge's verdict in place of the learner's own mark. Placement and
// course tests never serve a recall card (GeneratorContext.ObjectiveOnly), and
// a quiz serves this one as a plain flashcard (NoProofs; PlainRecall).

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

// WrittenGrader grades written answers to recall cards. internal/recallproof
// installs one (SetWrittenGrader) at startup.
type WrittenGrader interface {
	// On reports that written answers can be graded at all (a key, and
	// MATHUA_LLM_RECALL not 0).
	On() bool
	// ServeWritten reports whether the card should be served for a written
	// answer now: grading is on for its bank, the month's budget is not spent
	// and its marking scheme has not failed to generate. It must be cheap and
	// must not call the model: cards are rendered at startup too.
	ServeWritten(card RecallCard) bool
	// GradeWritten grades a written answer. A result it cannot give is
	// Unavailable, never a miss.
	GradeWritten(card RecallCard, answer string) grader.Result
}

var writtenGrader WrittenGrader

// SetWrittenGrader installs the written-answer grader; nil uninstalls it.
func SetWrittenGrader(w WrittenGrader) { writtenGrader = w }

// WrittenGradingOn reports an installed written-answer grader that is on, so
// the server can allow a written answer the time its grading takes.
func WrittenGradingOn() bool { return writtenGrader != nil && writtenGrader.On() }

// notGradedNow is the reason a written answer was not graded when no grader
// can take it: the learner can still reveal the back and mark it.
const notGradedNow = "Written answers are not being graded now. Use “Just show me the answer” and mark it yourself."

// gradeWritten grades a written answer to the recall card key. Anything that
// cannot be graded is Unavailable: nothing is recorded and the learner can
// still self-grade.
func (t *Track) gradeWritten(key, answer string) grader.Result {
	it := t.Bank.Item(key)
	var c *Concept
	if i := strings.LastIndex(key, "#"); i > 0 {
		c = t.Bank.Concept(key[:i])
	}
	w := writtenGrader
	if it == nil || c == nil || it.Kind != KindRecall || !Classify(it).Written || w == nil {
		return grader.Result{Unavailable: true, Feedback: notGradedNow}
	}
	return w.GradeWritten(t.card(c, it), answer)
}

// Classification is why a recall card is, or is not, a written-answer card.
type Classification struct {
	Written bool
	// Reason is the keyword that made it one ("prove", "show that"), the
	// override ("grading: llm"), or what excluded it ("show the graph").
	Reason string
}

var (
	tagRe      = regexp.MustCompile(`(?s)<[^>]*>`)
	mathRe     = regexp.MustCompile(`(?s)\\\(.*?\\\)|\\\[.*?\\\]|\$\$.*?\$\$|\$[^$\n]+\$`)
	quotedRe   = regexp.MustCompile(`“[^”]*”|"[^"]*"`)
	cropNoteRe = regexp.MustCompile(`(?i)⚠?\s*graph auto-extracted[^.]*\.?`)
	spaceRe    = regexp.MustCompile(`\s+`)

	// clozeRe is a cloze deletion: the card asks for the missing words.
	clozeRe = regexp.MustCompile(`\[\.\.\.\]|\[…\]|\{\{c\d+::`)
	// figureRe is a prompt that leans on a picture the judge cannot see.
	figureRe = regexp.MustCompile(`\b(the|this|these) (figures?|diagrams?|pictures?|charts?|plots?)( below| above)? (shows?|below|above|depicts?)\b|\bshown in the (figure|diagram)\b|\bthe figures? (labeled|labelled)\b`)
	// computerRe is a computer exercise: its answers are numbers read off
	// output the learner produced, not an argument.
	computerRe = regexp.MustCompile(`\bwoo\.data\b|\bsm\.ols\b|\bnp\.|\bpd\.|\bplt\.|\brng\b|\bimport \w|\bcomputer exercise\b|\bpython\b|\bdataframe\b`)
	// visualRe is a task carried out on a graph, sketch, table or diagram.
	// "graph" alone is not one: in graph theory it is the object of proof.
	visualRe = regexp.MustCompile(`\b(sketch|draw|plot|plotting|graphically|illustrate (it |this |them )?(on|with|in) (a|the) (graph|diagram|figure)|schedule|table of values|diagram|on (a|the) (graph|number line|same axes)|use (a|the) graph|graph (it|them|the function|both|each|this|these))\b|^graph\b|[.;:] graph\b`)

	// Phrases that mention a keyword without asking for an argument.
	notAskedRe = regexp.MustCompile(strings.Join([]string{
		// "show your work" on a numeric answer
		`\bshow (all )?(your |the |each |all )?(work|working|steps|calculations?)\b`,
		// a subject that shows: "the results show", "what does this show"
		`\b(figures?|graphs?|tables?|diagrams?|results?|data|outputs?|plots?|charts?|panels?|screens?|they|which|this|these|it|will|would|does|do|did|can|may|might|could) shows?\b`,
		`\bwhat (would|do|does|did|must|should|can) [^.?!]{0,40}\b(show|prove|demonstrate|verify|confirm)\b`,
		// a method question: "how do you prove", "what is needed to show"
		`\bhow (do|would|can|could|should|might|does|did|to) (you |we |one |i )?(\w+ ){0,2}(prove|show|verify|derive|justify|demonstrate|confirm)\b`,
		`\bhow (you|we|one|i) (would|could|can|might|should|will) (\w+ ){0,2}(prove|show|verify|derive|justify|demonstrate|confirm)\b`,
		`\b(is|are|was|were|be|been|need|needs|have|has|had|must|helpful|useful|used|enough|suffices?|ought|trying|try|tries|want|wants|easier|hard|harder|easy|way|ways|order|able|attempt|attempts|strategy|technique|method) to (prove|show|verify|derive|justify|demonstrate|confirm)\b`,
		`\b(showing|proving|deriving|verifying)\b`,
		// "compute it, or show that it does not exist": the task is the computation
		`\bor (to )?(show|prove|explain)\b[^.?!;]*`,
		// checking a computed answer
		`\b(verify|confirm|check) (your|the|this|that|each) (answer|result|solution|value)s?\b`,
		`\bverify (it |this |them )?(numerically|graphically|with a (graph|calculator|plot)|on a (graph|calculator)|by (graph|calculator|substitution))\b`,
		`\bverify (the )?crop\b`,
	}, "|"))

	proveRe   = regexp.MustCompile(`\b(dis)?prove\b`)
	argueRe   = regexp.MustCompile(`\b(derive|justify|demonstrate|explain why|argue that|establish that)\b`)
	verifyRe  = regexp.MustCompile(`\b(verify|confirm)\b`)
	showThat  = regexp.MustCompile(`\bshow\b[^.?!;:]{0,80}?\bthat\b`)
	showClaim = regexp.MustCompile(`\bshow\b[^.?!;:]{0,80}?\b(is|are|has|have|equals?|satisf(y|ies)|converges?|diverges?|exists?|holds?|impl(y|ies)|lies|lie|forms?|spans?|defines?|cannot|must|gives?|yields?|reduces?|becomes?|remains?|belongs?|commutes?|preserves?)\b`)
	// showFormula is "show \(f'(x) > 0\)." with the claim all mathematics.
	showFormula = regexp.MustCompile(`\bshow math( and math)?\s*([.?!;]|$)`)
	showHow     = regexp.MustCompile(`\bshow,|\bshow (algebraically|analytically|mathematically|formally|rigorously|directly|explicitly|carefully|how|why|by|using|from|without|whether)\b`)
)

// plainPrompt is the prompt as words: tags, entities, mathematics (as "math")
// and quoted titles removed, lower case.
func plainPrompt(s string) string {
	s = strings.NewReplacer("<br>", " ", "<br/>", " ", "<br />", " ", "</div>", " ", "</p>", " ").Replace(s)
	s = tagRe.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	s = cropNoteRe.ReplaceAllString(s, " ")
	s = mathRe.ReplaceAllString(s, " math ")
	s = quotedRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(strings.ToLower(spaceRe.ReplaceAllString(s, " ")))
}

// Classify reports whether a recall card asks for a written proof,
// derivation or justification. A "grading" override in the bank wins. Only
// recall cards are ever written-answer cards.
//
// The rules, in order:
//   - the override;
//   - exclusions the judge cannot work with: a picture on the front, a back
//     that is only a picture, a prompt that refers to a figure, a cloze
//     deletion, a computer exercise;
//   - phrases that mention a keyword without asking for an argument are
//     removed ("show your work", "what does this show", "how do you prove",
//     "compute it or show it does not exist", "verify your answer");
//   - "prove", "derive", "justify", "demonstrate", "explain why", "argue
//     that" in what is left make a written-answer card;
//   - "show", "verify" and "confirm" do too, unless the task is done on a
//     graph, sketch, table or diagram; "show" must ask for a claim ("show
//     that", "show \(A\) is invertible", "show how", "show, by ...").
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
	if clozeRe.MatchString(it.Prompt) {
		return Classification{Reason: "a cloze card"}
	}
	p := plainPrompt(it.Prompt)
	switch {
	case figureRe.MatchString(p):
		return Classification{Reason: "refers to a figure"}
	case computerRe.MatchString(p):
		return Classification{Reason: "a computer exercise"}
	}
	q := notAskedRe.ReplaceAllString(p, " ")
	if m := proveRe.FindString(q); m != "" {
		return Classification{Written: true, Reason: m}
	}
	if m := argueRe.FindString(q); m != "" {
		return Classification{Written: true, Reason: m}
	}
	visual := visualRe.MatchString(p)
	if m := verifyRe.FindString(q); m != "" {
		if visual {
			return Classification{Reason: m + " on a graph or table"}
		}
		return Classification{Written: true, Reason: m}
	}
	if !strings.Contains(q, "show") {
		if keywordRe.MatchString(p) {
			return Classification{Reason: "keyword not a request"}
		}
		return Classification{}
	}
	if visual {
		return Classification{Reason: "show on a graph"}
	}
	switch {
	case showThat.MatchString(q):
		return Classification{Written: true, Reason: "show that"}
	case showHow.MatchString(q):
		return Classification{Written: true, Reason: "show how"}
	case showClaim.MatchString(q), showFormula.MatchString(q):
		return Classification{Written: true, Reason: "show"}
	}
	return Classification{Reason: "show (no claim)"}
}

// keywordRe finds any of the keywords, for reporting why a card that has one
// is not a written-answer card.
var keywordRe = regexp.MustCompile(`\b(prove|show|verify|justify|derive|confirm|demonstrate|explain why)\b`)

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
// internal/engine strips the marker the same way for a card served in a
// generated topic's quiz (engine.plainRecall).
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
