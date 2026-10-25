package generator

import (
	"fmt"
	"hash/fnv"
	"math/rand"
	"strings"
)

// choiceLetters are the option labels the client recognises; `web/next-app/lib/choices.ts` parses
// trailing "A) …" blocks and refuses a list whose letters are not consecutive from A.
var choiceLetters = []string{"A", "B", "C", "D", "E"}

// MultipleChoice builds a recognition question from one correct answer and its distractors.
//
// The options are deduplicated, shuffled (in an order fixed by the stem) and written into the stem as an "A) …" block. The answer is
// the correct option's text rather than its letter, because a tap submits the text and the
// multiple_choice grader compares whole strings; it also accepts a bare letter, so a learner who
// types "B" is graded right.
//
// At most five options are kept (the letters the client parses). When more are supplied the correct
// one is moved into the fifth slot before truncation, so the answer never disappears.
func MultipleChoice(question, correct string, distractors ...string) Problem {
	opts := []string{correct}
	seen := map[string]bool{strings.TrimSpace(correct): true}
	for _, d := range distractors {
		key := strings.TrimSpace(d)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		opts = append(opts, d)
	}
	// The order is fixed by the question itself (Q30). A fresh shuffle on
	// every serve made the same question a "new" text each time: the
	// practice filter that skips what the learner just answered missed it,
	// and mastery counted one question seen four times as four instances.
	h := fnv.New64a()
	h.Write([]byte(question + "\x00" + correct))
	rng := rand.New(rand.NewSource(int64(h.Sum64())))
	rng.Shuffle(len(opts), func(i, j int) { opts[i], opts[j] = opts[j], opts[i] })
	if len(opts) > len(choiceLetters) {
		for i, o := range opts {
			if o == correct {
				opts[len(choiceLetters)-1], opts[i] = opts[i], opts[len(choiceLetters)-1]
				break
			}
		}
		opts = opts[:len(choiceLetters)]
	}
	var b strings.Builder
	b.WriteString(question)
	for i, o := range opts {
		fmt.Fprintf(&b, "\n\n%s) %s", choiceLetters[i], o)
	}
	return Problem{Question: b.String(), Answer: correct}
}
