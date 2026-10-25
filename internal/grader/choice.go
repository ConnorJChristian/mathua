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
	if len(strings.TrimSpace(expected)) == 1 {
		// The key is a letter: a clicked option's text becomes its letter.
		for _, m := range choiceOptionRe.FindAllStringSubmatch(question, -1) {
			if strings.EqualFold(strings.TrimSpace(m[2]), a) {
				return m[1]
			}
		}
		return answer
	}
	if len(a) != 1 {
		return answer
	}
	l := strings.ToUpper(a)
	for _, m := range choiceOptionRe.FindAllStringSubmatch(question, -1) {
		if m[1] == l {
			return strings.TrimSpace(m[2])
		}
	}
	return answer
}
