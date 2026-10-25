package grader

import (
	"regexp"
	"strings"
)

// A plain multi-word phrase, the only shape where the answer field can lose a
// word boundary without changing meaning.
var wordPhraseRe = regexp.MustCompile(`^[A-Za-z]+(?:\s+[A-Za-z]+)+$`)

// samePhrase accepts a phrase whose spaces were lost on the way out of the
// math field.
//
// MathLive serialises `line segment` as `l i n e s e g m e n t`: it treats
// adjacent letters in math mode as a product and spaces every one. The client
// rejoins single-letter runs, which turns the phrase into `linesegment`, so an
// exact comparison against `line segment` marks a correct answer wrong.
//
// Two guards keep this narrow. The expected answer must be letters and spaces
// only, so `5 R 3` (remainder notation) and `2pi r` (a product) never come
// here. The answer must be letters only, so a collapsed phrase cannot absorb
// digits, operators or a fraction.
func samePhrase(expected, answer string) bool {
	e := strings.ToLower(strings.TrimSpace(expected))
	a := strings.ToLower(strings.TrimSpace(answer))
	if !wordPhraseRe.MatchString(e) || a == "" {
		return false
	}
	for _, r := range a {
		if (r < 'a' || r > 'z') && r != ' ' {
			return false
		}
	}
	joined := strings.Join(strings.Fields(e), "")
	return strings.Join(strings.Fields(a), "") == joined
}

type choiceGrader struct{}

func (g *choiceGrader) grade(expected, answer string) Result {
	e := strings.TrimSpace(expected)
	a := strings.TrimSpace(answer)
	if e == "" || a == "" {
		return emptyResult()
	}
	if strings.EqualFold(e, a) {
		return Result{Correct: true, Score: 1}
	}
	if samePhrase(e, a) {
		return Result{Correct: true, Score: 1}
	}
	if isWordKey(e) && wordAnswerMatches(e, a) {
		return Result{Correct: true, Score: 1}
	}

	// A typed option letter is resolved against the question's options before
	// grading (ResolveChoiceLetter); matching a letter against the words of
	// the key accepted "A" for any key containing the word "a".

	return Result{Correct: false, Score: 0, Feedback: "Incorrect choice"}
}

var choiceOptionRe = regexp.MustCompile(`(?m)^([A-E])\) (.+)$`)

// ResolveChoiceLetter turns a typed option letter ("b") into that option's
// text, read from the question the learner saw. A multiple-choice key is the
// correct option's text, so a bare letter can only be graded against the
// options. The grader used to accept any letter that appeared as a word in
// the key: "A" was right for every key containing the word "a". When the key
// is itself a letter, or the question has no such option, the answer is
// returned unchanged.
func ResolveChoiceLetter(question, expected, answer string) string {
	a := strings.TrimSpace(answer)
	e := strings.TrimSpace(expected)
	opts := choiceOptionRe.FindAllStringSubmatch(question, -1)
	keyIsOption := false
	for _, m := range opts {
		if strings.EqualFold(strings.TrimSpace(m[2]), e) {
			keyIsOption = true
		}
	}
	// The key is the option's text ("4", "Z", "a line through two points").
	// A learner who typed that text is answered as typed; only a bare letter
	// that is not itself an option's text is resolved to its option. A
	// one-character key is option text when an option reads that way: "4"
	// is not the letter D (Q44, a regression of the first version of this).
	if keyIsOption || len(opts) == 0 {
		if len(a) != 1 {
			return answer
		}
		for _, m := range opts {
			if strings.EqualFold(strings.TrimSpace(m[2]), a) {
				return answer
			}
		}
		l := strings.ToUpper(a)
		for _, m := range opts {
			if m[1] == l {
				return strings.TrimSpace(m[2])
			}
		}
		return answer
	}
	// The key is a letter ("B") that names an option: a clicked option's
	// text becomes its letter.
	if len(e) == 1 {
		for _, m := range opts {
			if strings.EqualFold(strings.TrimSpace(m[2]), a) {
				return m[1]
			}
		}
	}
	return answer
}
