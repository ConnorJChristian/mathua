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
	// The gloss may list alternatives with "/" ("stable spiral (spiral
	// sink/stable focus)").
	wordKeyRe   = regexp.MustCompile(`^[A-Za-z][A-Za-z '\-]*(\([A-Za-z][A-Za-z '\-/]+\))?$`)
	// A list key names alternatives, any one of which is the answer:
	// "corner/cusp/vertical tangent/discontinuity" (Q52).
	wordListPartRe = regexp.MustCompile(`^[A-Za-z][A-Za-z '\-]*$`)
	parenPartRe = regexp.MustCompile(`\s*\(([^()]*)\)\s*`)
	// A minus sign is kept: "-sinh x" is not "sinh x".
	nonWordRe = regexp.MustCompile(`[^a-z \-]+`)
)

// IsWordKey is isWordKey for the key self-check (internal/gradecheck), so
// the check spells a key as words exactly when the grader reads it as words.
func IsWordKey(expected string) bool { return isWordKey(expected) }

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
	// A difference of variable names ("x-y", "wz-y x") is mathematics, not
	// a hyphenated phrase: every piece is one or two letters (fork fix:
	// "wz-yx" was refused for the determinant "wz-y x").
	if strings.Contains(e, "-") {
		short := true
		for _, w := range strings.FieldsFunc(e, func(r rune) bool { return r == ' ' || r == '-' }) {
			if len(w) > 2 {
				short = false
				break
			}
		}
		if short {
			return false
		}
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
	aw = dropFiller(aw)
	candidates := [][]string{words(expected)}
	if m := parenPartRe.FindStringSubmatch(expected); m != nil {
		candidates = append(candidates, words(parenPartRe.ReplaceAllString(expected, " ")))
		for _, g := range strings.Split(m[1], "/") {
			candidates = append(candidates, words(g))
		}
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

// dropFiller removes a leading "it is" / "the function is" from a word
// answer: "it is convex" says "convex".
func dropFiller(aw []string) []string {
	for _, pre := range [][]string{{"it", "is"}, {"it", "s"}, {"function", "is"}, {"f", "is"}, {"this", "is"}} {
		if len(aw) > len(pre) {
			match := true
			for i := range pre {
				if aw[i] != pre[i] {
					match = false
					break
				}
			}
			if match {
				return aw[len(pre):]
			}
		}
	}
	return aw
}

// wordListParts returns the alternatives of a list key ("corner/cusp/
// vertical tangent/discontinuity"), or nil when the key is not one. Every
// part must be a word or phrase of at least three letters, so "x/y" and
// "1/2" stay mathematics. A gloss in brackets after the list is kept as a
// further alternative.
func wordListParts(expected string) []string {
	e := strings.TrimSpace(expected)
	main, gloss := e, ""
	if m := parenPartRe.FindStringSubmatch(e); m != nil {
		main = strings.TrimSpace(parenPartRe.ReplaceAllString(e, " "))
		gloss = m[1]
	}
	if !strings.Contains(main, "/") {
		return nil
	}
	parts := strings.Split(main, "/")
	for i, p := range parts {
		p = strings.TrimSpace(p)
		if !wordListPartRe.MatchString(p) || len(strings.ReplaceAll(p, " ", "")) < 3 {
			return nil
		}
		parts[i] = p
	}
	if gloss != "" {
		parts = append(parts, strings.Split(gloss, "/")...)
	}
	return parts
}

// wordListAnswerMatches accepts any one alternative of a list key, compared
// as words (articles, case and punctuation ignored).
func wordListAnswerMatches(parts []string, answer string) bool {
	for _, p := range parts {
		if wordAnswerMatches(p, answer) {
			return true
		}
	}
	return false
}
