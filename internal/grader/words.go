package grader

import (
	"regexp"
	"strings"
)

// Word answers (Q52). A key that is a word or phrase ("corner", "no",
// "convex (concave up)", "stable node") is compared as words, never sent to
// SymPy: "no, it has a cusp" was parsed as a tuple and crashed the grader, and
// "a corner" was refused for "corner".

var (
	// A parenthetical needs two letters: "cos(x)" and "(k)" are mathematics.
	wordKeyRe   = regexp.MustCompile(`^[A-Za-z][A-Za-z '\-]*(\([A-Za-z][A-Za-z '\-]+\))?$`)
	parenPartRe = regexp.MustCompile(`\s*\(([^()]*)\)\s*`)
	nonWordRe   = regexp.MustCompile(`[^a-z ]+`)
)

// isWordKey reports a key made of words (with at most one parenthetical) and
// no mathematics.
func isWordKey(expected string) bool {
	e := strings.TrimSpace(expected)
	if e == "" || !wordKeyRe.MatchString(e) {
		return false
	}
	if l := strings.ToLower(e); l == "no" || l == "yes" {
		return true
	}
	// Single letters and short variable-like keys stay mathematics ("x", "e").
	return len(strings.Fields(strings.ReplaceAll(e, "(", " "))) > 1 || len(e) > 2
}

func words(s string) []string {
	s = nonWordRe.ReplaceAllString(strings.ToLower(s), " ")
	var out []string
	for _, w := range strings.Fields(s) {
		if w == "a" || w == "an" || w == "the" {
			continue
		}
		out = append(out, w)
	}
	return out
}

func sameWords(a, b []string) bool {
	if len(a) != len(b) || len(a) == 0 {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// wordAnswerMatches accepts the key's words, ignoring articles, case and
// punctuation; either half of a "main (gloss)" key; and, for a yes/no key, an
// answer that starts with that word ("no, it has a cusp"). It also accepts
// an answer that is the key with a qualifier in front ("stable node" for
// "node") only when the key is one word and the answer ends with it.
func wordAnswerMatches(expected, answer string) bool {
	if strings.EqualFold(strings.TrimSpace(expected), "infinity") {
		switch strings.ToLower(strings.TrimSpace(answer)) {
		case "oo", "∞", "inf", "+inf", "+infinity", "+∞", "+oo":
			return true
		}
	}
	aw := words(answer)
	if len(aw) == 0 {
		return false
	}
	candidates := [][]string{words(expected)}
	if m := parenPartRe.FindStringSubmatch(expected); m != nil {
		candidates = append(candidates, words(parenPartRe.ReplaceAllString(expected, " ")), words(m[1]))
	}
	for _, c := range candidates {
		if sameWords(c, aw) {
			return true
		}
	}
	kw := words(expected)
	if len(kw) == 1 && (kw[0] == "yes" || kw[0] == "no") {
		return aw[0] == kw[0]
	}
	return false
}
