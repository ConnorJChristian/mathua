package grader

import (
	"math"
	"regexp"
	"strings"
)

type Router struct {
	numeric    *numericGrader
	choice     *choiceGrader
	comparison *comparisonGrader
	ordering   *orderingGrader
	tuple      *tupleGrader
	complex    *complexGrader
}

func NewRouter() *Router {
	return &Router{
		numeric:    &numericGrader{},
		choice:     &choiceGrader{},
		comparison: &comparisonGrader{},
		ordering:   &orderingGrader{},
		tuple:      &tupleGrader{},
		complex:    &complexGrader{},
	}
}

func (r *Router) Grade(gradingType GradingType, expected, answer string) Result {
	// A bank answer is graded by the bank (code against hidden tests, a
	// choice against its key): the learner's text must reach it untouched.
	// lenient() used to strip a leading "query = " from SQL tasks (fork fix).
	if !IsBankAnswer(expected) {
		answer = lenient(gradingType, expected, answer)
	}
	switch gradingType {
	case GradingNumeric:
		res := r.numeric.grade(expected, answer)
		if !res.Correct && !res.Unavailable {
			// A fraction answered as a rounded decimal (61/3 as 20.333): the
			// same number to three places is the same answer (fork fix).
			if ev, ok := constEvalNumeric(expected); ok {
				if av, ok2 := constEvalNumeric(answer); ok2 && strings.Contains(expected, "/") && strings.Contains(answer, ".") {
					if math.Abs(ev-av) <= 0.0006+1e-9 {
						return Result{Correct: true, Score: 1}
					}
				}
			}
		}
		return res
	case GradingPolynomial, GradingExpression, GradingSymbolic:
		// A word answer ("decreasing") compares case-insensitively (fork fix).
		if strings.EqualFold(strings.TrimSpace(expected), strings.TrimSpace(answer)) || samePhrase(expected, answer) {
			return Result{Correct: true, Score: 1}
		}
		// A word key is graded as words, never by SymPy (Q52).
		if isWordKey(expected) {
			if wordAnswerMatches(expected, answer) {
				return Result{Correct: true, Score: 1}
			}
			return Result{Correct: false, Score: 0, Feedback: "Incorrect"}
		}
		return gradeSymPy(expected, answer)
	case GradingMultipleChoice:
		res := r.choice.grade(expected, answer)
		if !res.Correct && strings.HasPrefix(strings.TrimSpace(expected), "[") &&
			strings.ReplaceAll(expected, " ", "") == strings.ReplaceAll(answer, " ", "") {
			return Result{Correct: true, Score: 1}
		}
		if !res.Correct && looksLikeMath(expected) {
			// A free-form expression declared as a choice (an ODE's general
			// solution, a set, a log identity) is compared as mathematics,
			// not as text: spaces, "y =" and the names of the arbitrary
			// constants do not matter (fork fix).
			if sym := gradeSymPy(canonConstants(expected), canonConstants(answer)); sym.Correct {
				return sym
			}
		}
		return res
	case GradingComparison:
		return r.comparison.grade(expected, answer)
	case GradingOrdering:
		return r.ordering.grade(expected, answer)
	case GradingTuple:
		return r.tuple.grade(expected, answer)
	case GradingComplex:
		return r.complex.grade(expected, answer)
	case GradingBank, GradingPython:
		return gradePython(expected, answer)
	default:
		return r.tuple.grade(expected, answer)
	}
}

func (r *Router) Close() {}

// leadingAssignRe matches a learner's "x = ", "y=", "f'(x) =", "dy/dx =" in
// front of an answer whose expected form has no equals sign.
var leadingAssignRe = regexp.MustCompile(`^\s*(?:[a-zA-Z]\w*(?:\([a-zA-Z]\))?'*|d[a-zA-Z]/d[a-zA-Z])\s*=\s*`)

// lenient normalises what a learner typed in the ways that never change the
// answer: a leading "x =" when the expected answer has none, and spaces inside
// a bracketed tuple or matrix (fork fix: "x = 6" and "[[9, 6], [0, 0]]" used to
// grade wrong where "6" and "[[9,6],[0,0]]" graded right).
func lenient(gradingType GradingType, expected, answer string) string {
	a := strings.TrimSpace(answer)
	e := strings.TrimSpace(expected)
	if !strings.Contains(e, "=") && leadingAssignRe.MatchString(a) {
		if rest := strings.TrimSpace(leadingAssignRe.ReplaceAllString(a, "")); rest != "" {
			a = rest
		}
	}
	if gradingType == GradingTuple && strings.HasPrefix(e, "[") && strings.HasPrefix(a, "[") {
		a = strings.ReplaceAll(a, " ", "")
	}
	// The notation the prompts are written in ("8y^{2} + 4", "\\frac{1}{3}",
	// "\\sqrt{x}", a typeset minus sign) is read as the plain form the
	// graders take (Q49; Q41 had covered flashcards only).
	switch gradingType {
	case GradingNumeric, GradingPolynomial, GradingExpression, GradingSymbolic, GradingTuple:
		a = NormalizeTypedMath(a, e)
	}
	return a
}

// looksLikeMath reports an expected "choice" that is an expression rather
// than an option label or a word.
func looksLikeMath(expected string) bool {
	e := strings.TrimSpace(expected)
	if len(e) <= 1 || wordPhraseRe.MatchString(e) {
		return false
	}
	return strings.ContainsAny(e, "=^+/{}\\")
}

// canonConstants strips a leading "y =" and renames the arbitrary constants
// of a general solution (C, C1, C2, c, K) to one spelling, so SymPy sees two
// general solutions that differ only in constant names as equal.
func canonConstants(s string) string {
	s = strings.TrimSpace(s)
	s = leadingAssignRe.ReplaceAllString(s, "")
	s = regexp.MustCompile(`\b[cCkK]_?(\d)\b`).ReplaceAllString(s, "C$1")
	s = regexp.MustCompile(`\b[cCkK]\b`).ReplaceAllString(s, "C1")
	return s
}

// IsBankAnswer reports whether an expected answer came from a question bank
// (internal/pytrack): its kind prefix says so. A bank item can be served on a
// generated concept (your flashcards in its practice), and must then be graded
// as the bank item it is, whatever the concept's own grading type.
func IsBankAnswer(expected string) bool {
	for _, p := range []string{"mc:", "out:", "code:", "recall:", "math:"} {
		if strings.HasPrefix(expected, p) {
			return true
		}
	}
	return false
}

// pythonGrader is installed by internal/pytrack at startup. It is a hook rather
// than an import because pytrack depends on this package for Result.
var pythonGrader func(expected, answer string) Result

// SetPythonGrader installs the Python-track grader. Call it once at startup.
func SetPythonGrader(fn func(expected, answer string) Result) { pythonGrader = fn }

func gradePython(expected, answer string) Result {
	if pythonGrader == nil {
		return Result{Feedback: "Python grading is not configured", Unavailable: true}
	}
	return pythonGrader(expected, answer)
}
