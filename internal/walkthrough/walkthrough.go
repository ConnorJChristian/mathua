// Package walkthrough is the one feedback contract a graded answer comes back
// with (fork change, FSRS_PORT_PLAN 0c): the key as shown, the worked steps
// to it, the one-line reason, where the learner's answer left the path when
// that can be told, and the topic's reading.
//
// The steps travel inside the explanation string. Every path that already
// carries an explanation (the study anchor, the active session, the quiz and
// diagnostic sessions, the attempts table) then carries the steps too, with no
// change to storage: a problem's explanation is Encode(explain, steps), and
// the response boundary calls Split to take it apart. A client that reads the
// explanation whole still sees the steps, as HTML comments and lines, so an
// older history row renders as before.
package walkthrough

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// StepsMarker opens the steps block at the end of an explanation. StepMarker
// separates steps. Both are HTML comments, invisible to a Markdown renderer
// and impossible in authored prose or LaTeX.
const (
	StepsMarker = "<!--steps-->"
	StepMarker  = "<!--step-->"
)

// Reading is the topic's reading, as a link: a module's own file at a page,
// or a web page.
type Reading struct {
	Label string `json:"label"`
	URL   string `json:"url"`
	Page  int    `json:"page,omitempty"`
}

// Divergence says where a wrong answer left the worked path. Step is the
// 1-based step whose value the learner's answer matches, 0 when the step is
// not known and only the reason is. Reason is one line: the grader's own
// diagnosis when it has one, else what the match says.
type Divergence struct {
	Step   int    `json:"step"`
	Reason string `json:"reason"`
}

// Walkthrough is what every graded answer returns, right or wrong.
type Walkthrough struct {
	// Answer is the key as it is shown to the learner: a tidied value, a
	// choice's text, a printed output, or the typed-math key in LaTeX.
	Answer string `json:"answer"`
	// Steps are the worked path, in the item's notation. Empty when the item
	// has none (a definition, a term, a flashcard without working).
	Steps []string `json:"steps"`
	// Explain is the one-line reason the key is right.
	Explain string `json:"explain"`
	// DivergedAt is set for a wrong answer when the step it left the path at
	// can be told, or when the grader diagnosed the mistake.
	DivergedAt *Divergence `json:"diverged_at,omitempty"`
	// Reading is the topic's reading, when it has one.
	Reading *Reading `json:"reading,omitempty"`
}

// Encode writes an explanation that carries steps. With no steps it is the
// explanation itself.
func Encode(explain string, steps []string) string {
	var kept []string
	for _, s := range steps {
		if s = strings.TrimSpace(s); s != "" {
			kept = append(kept, s)
		}
	}
	explain = strings.TrimSpace(explain)
	if len(kept) == 0 {
		return explain
	}
	var b strings.Builder
	b.WriteString(explain)
	if explain != "" {
		b.WriteString("\n\n")
	}
	b.WriteString(StepsMarker)
	for i, s := range kept {
		if i > 0 {
			b.WriteString("\n" + StepMarker)
		}
		b.WriteString("\n")
		b.WriteString(s)
	}
	return b.String()
}

// Split takes an encoded explanation apart: the prose before the steps
// block, and the steps. An explanation without the block is returned whole
// with no steps.
func Split(explanation string) (explain string, steps []string) {
	i := strings.Index(explanation, StepsMarker)
	if i < 0 {
		return strings.TrimSpace(explanation), nil
	}
	explain = strings.TrimSpace(explanation[:i])
	for _, s := range strings.Split(explanation[i+len(StepsMarker):], StepMarker) {
		if s = strings.TrimSpace(s); s != "" {
			steps = append(steps, s)
		}
	}
	return explain, steps
}

// HasSteps reports whether an explanation carries a steps block.
func HasSteps(explanation string) bool {
	return strings.Contains(explanation, StepsMarker)
}

// answerLineRe is the key line a bank item's explanation opens with
// (internal/pytrack: "**Answer:** \(x = 3\)").
var answerLineRe = regexp.MustCompile(`(?m)^\*\*Answer:\*\*[ \t]*(.*)$`)

// readingLineRe is the reading line a bank item's explanation ends with:
// "Read more: [Book §1.2, Title](url)".
var readingLineRe = regexp.MustCompile(`(?m)^Read more: \[([^\]]*)\]\(([^)]*)\)[ \t]*$`)

// sourceLineRe is the source line of a bank item with no link.
var sourceLineRe = regexp.MustCompile(`(?m)^Source: .*$`)

// Build assembles the contract from a graded answer: the shown key (empty
// when the explanation carries it), the served explanation, the learner's
// answer and the grader's diagnosis (both empty for "I don't know" or a
// right answer), and the topic's reading when the caller has one.
//
// The explanation's own key line and reading line are lifted into Answer and
// Reading rather than repeated in Explain, so a surface renders each once.
func Build(shownAnswer, explanation, learnerAnswer, diagnosis string, reading *Reading) Walkthrough {
	explain, steps := Split(explanation)
	w := Walkthrough{Answer: strings.TrimSpace(shownAnswer), Steps: steps, Reading: reading}
	if m := answerLineRe.FindStringSubmatch(explain); m != nil {
		if w.Answer == "" {
			w.Answer = strings.TrimSpace(m[1])
		}
		explain = strings.Replace(explain, m[0], "", 1)
	}
	if m := readingLineRe.FindStringSubmatch(explain); m != nil {
		if w.Reading == nil {
			w.Reading = &Reading{Label: m[1], URL: m[2]}
		}
		explain = strings.Replace(explain, m[0], "", 1)
	} else if m := sourceLineRe.FindString(explain); m != "" && w.Reading == nil {
		w.Reading = &Reading{Label: strings.TrimSpace(strings.TrimPrefix(m, "Source:"))}
		explain = strings.Replace(explain, m, "", 1)
	}
	w.Explain = collapseBlankLines(explain)
	if w.Steps == nil {
		w.Steps = []string{}
	}
	w.DivergedAt = diverge(w.Steps, w.Answer, learnerAnswer, diagnosis)
	return w
}

// diverge finds the step a wrong answer stopped at: the first step whose
// working ends on the learner's number, when that number is not the key.
// Only numbers are matched, and only whole tokens, so "8" does not match
// inside "0.8" or "18".
func diverge(steps []string, key, answer, diagnosis string) *Divergence {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		if diagnosis != "" {
			return &Divergence{Reason: diagnosis}
		}
		return nil
	}
	if v, ok := parseNumber(answer); ok && !numberEquals(key, v) {
		for i, s := range steps {
			if stepEndsWith(s, v) {
				reason := diagnosis
				if reason == "" {
					reason = "Your answer is the value at this step: the path was left here."
				}
				return &Divergence{Step: i + 1, Reason: reason}
			}
		}
	}
	if diagnosis != "" {
		return &Divergence{Reason: diagnosis}
	}
	return nil
}

var numberRe = regexp.MustCompile(`-?[0-9]+(?:\.[0-9]+)?(?:/[0-9]+)?`)

func parseNumber(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if !numberRe.MatchString(s) || numberRe.FindString(s) != s {
		return 0, false
	}
	if i := strings.Index(s, "/"); i > 0 {
		n, err1 := strconv.ParseFloat(s[:i], 64)
		d, err2 := strconv.ParseFloat(s[i+1:], 64)
		if err1 != nil || err2 != nil || d == 0 {
			return 0, false
		}
		return n / d, true
	}
	v, err := strconv.ParseFloat(s, 64)
	return v, err == nil
}

func numberEquals(key string, v float64) bool {
	k, ok := parseNumber(key)
	return ok && close(k, v)
}

func close(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= 1e-9*(1+abs(a))
}

func abs(a float64) float64 {
	if a < 0 {
		return -a
	}
	return a
}

// trailingNumberRe is a number that closes a step, before any closing
// punctuation or math delimiter: "..., so f_x = 8.", "... = \(8\)".
var trailingNumberRe = regexp.MustCompile(`(-?[0-9]+(?:\.[0-9]+)?(?:/[0-9]+)?)[ \t.$)\\]*$`)

// stepEndsWith reports whether a step closes on the number v: the value a
// step works out is written last ("f_x = 6x - 2y, which at (1,-1) is 8."),
// and a number elsewhere in the step (an exponent, a point) is not a result.
func stepEndsWith(step string, v float64) bool {
	m := trailingNumberRe.FindStringSubmatch(strings.TrimSpace(step))
	if m == nil {
		return false
	}
	last, ok := parseNumber(m[1])
	return ok && close(last, v)
}

var blankLinesRe = regexp.MustCompile(`\n{3,}`)

func collapseBlankLines(s string) string {
	return strings.TrimSpace(blankLinesRe.ReplaceAllString(s, "\n\n"))
}

// TrimCovered returns text without the part the steps already say: when the
// text opens with the steps' sentences in order (a worked solution the bank
// build split into steps), what follows them is returned, typically the
// source line; otherwise text is returned whole. Whitespace, semicolons and
// commas are not compared: a step split off a clause lost its separator.
func TrimCovered(text string, steps []string) string {
	if len(steps) == 0 {
		return text
	}
	skip := func(r rune) bool { return r == ' ' || r == '\n' || r == '\t' || r == '\r' || r == ';' || r == ',' }
	want := []rune(strings.Join(steps, " "))
	w := 0
	for w < len(want) && skip(want[w]) {
		w++
	}
	if w == len(want) {
		return text
	}
	runes := []rune(text)
	for i, r := range runes {
		if skip(r) {
			continue
		}
		if r != want[w] {
			return text
		}
		w++
		for w < len(want) && skip(want[w]) {
			w++
		}
		if w == len(want) {
			return strings.TrimSpace(string(runes[i+1:]))
		}
	}
	return text
}

// abbrevRe is an abbreviation or an initial a sentence does not end on.
var abbrevRe = regexp.MustCompile(`(?:\b(?:e\.g|i\.e|cf|vs|approx|eq|Eq|fig|Fig|ch|Ch|sec|Sec|p|pp|no|No|Prof|Dr|Thm|Def|Prop|Cor|Lem|Ex|resp)|\b[A-Z])\.$`)

// workingRe says a sentence works something out: an equals sign or a
// comparison, an operator between two numbers, a power, or a colon
// introducing a value. A year or a count on its own is prose.
var workingRe = regexp.MustCompile(`[=≈→⇒≤≥<>∂∫∑√^]|\d\s*[-+*/−×·±]\s*\d|\d\s*[a-zA-Zα-ωπλ]\b|:\s*[-−(\\]{0,2}\d`)

var sentenceEndRe = regexp.MustCompile(`(?:[.!?])\s+`)

// SplitSentences is a paragraph's sentences: a full stop, question or
// exclamation mark followed by white space ends one unless it closes an
// abbreviation or an initial. A sentence whose clauses are joined by
// semicolons and each work something out is one step per clause.
// (scripts/python_bank.py split_sentences is the same rule for the bank
// build.)
func SplitSentences(text string) []string {
	var out []string
	// Mathematics is never split: each span becomes a placeholder while the
	// sentence boundaries are found, and is put back after.
	var spans []string
	text = mathSpanRe.ReplaceAllStringFunc(strings.TrimSpace(text), func(m string) string {
		spans = append(spans, m)
		return fmt.Sprintf("\x00%d\x01", len(spans)-1)
	})
	restore := func(s string) string {
		return spanRefRe.ReplaceAllStringFunc(s, func(ref string) string {
			n, _ := strconv.Atoi(ref[1 : len(ref)-1])
			return spans[n]
		})
	}
	for text != "" {
		loc := sentenceEndRe.FindStringIndex(text)
		var chunk string
		if loc == nil {
			chunk, text = text, ""
		} else {
			chunk, text = text[:loc[0]+1], text[loc[1]:]
		}
		if len(out) > 0 && abbrevRe.MatchString(out[len(out)-1]) {
			out[len(out)-1] += " " + chunk
		} else {
			out = append(out, chunk)
		}
	}
	var steps []string
	for _, sent := range out {
		clauses := semicolonRe.Split(sent, -1)

		all := len(clauses) > 1
		for _, c := range clauses {
			if !workingRe.MatchString(restore(c)) {
				all = false
			}
		}
		if !all {
			// A chain of assignments, "a_2 = 3, a_3 = 5, a_4 = 7": one step
			// each, when every clause opens as an assignment.
			clauses = commaRe.Split(sent, -1)
			all = len(clauses) > 1
			for _, c := range clauses {
				if !assignRe.MatchString(restore(c)) {
					all = false
				}
			}
		}
		if all {
			for _, c := range clauses {
				steps = append(steps, restore(strings.TrimRight(c, ";,")))
			}
		} else {
			steps = append(steps, restore(sent))
		}
	}
	return steps
}

var (
	mathSpanRe  = regexp.MustCompile(`(?s)\\\[.*?\\\]|\\\(.*?\\\)|\$\$.*?\$\$|\$[^$\n]*\$`)
	spanRefRe   = regexp.MustCompile("\x00[0-9]+\x01")
	semicolonRe = regexp.MustCompile(`;\s+`)
	commaRe     = regexp.MustCompile(`,\s+`)
	brParaRe    = regexp.MustCompile(`(?i)(?:<br\s*/?>\s*){2,}`)
	brRe        = regexp.MustCompile(`(?i)<br\s*/?>`)
	paraRe      = regexp.MustCompile(`\n\s*\n`)
	assignRe    = regexp.MustCompile(`^[$\\(]*[A-Za-z][A-Za-z0-9_{}^']*(?:\([^()]*\))?\s*=`)
)

// WorkedSteps is the steps a worked solution works through: its sentences in
// order, without a closing "Source:" paragraph, when at least two of them
// carry working. A one-sentence reason, or prose with no working, gives no
// steps and is shown whole as the reason. The bank build does the same for
// authored items; this is for generated explanations and imported backs.
func WorkedSteps(text string) []string {
	var steps []string
	// An imported card's back breaks lines with HTML.
	text = brParaRe.ReplaceAllString(text, "\n\n")
	text = brRe.ReplaceAllString(text, "\n")
	for _, para := range paraRe.Split(text, -1) {
		para = strings.Join(strings.Fields(para), " ")
		if para == "" || strings.HasPrefix(para, "Source:") || strings.HasPrefix(para, "```") || strings.HasPrefix(para, "Read more:") {
			continue
		}
		for _, st := range SplitSentences(para) {
			// A source written into the last sentence is a citation, not a step.
			if strings.HasPrefix(st, "Source:") || strings.HasPrefix(st, "Read more:") {
				continue
			}
			steps = append(steps, st)
		}
	}
	working := 0
	for _, s := range steps {
		if workingRe.MatchString(s) {
			working++
		}
	}
	if len(steps) < 2 || working < 2 {
		return nil
	}
	return steps
}
