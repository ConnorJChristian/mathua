package grader

import (
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
	// A proof is graded as a proof whatever the concept's own grading type:
	// proof problems are served inside generated topics' practice, the way
	// question-bank cards are.
	if gradingType == GradingProof || IsProofAnswer(expected) {
		return gradeProof(expected, answer)
	}
	// A bank answer is graded by the bank (code against hidden tests, a
	// choice against its key): the learner's text must reach it untouched.
	// lenient() used to strip a leading "query = " from SQL tasks (fork fix).
	if !IsBankAnswer(expected) {
		// The math box exports invisible brackets as "{:" and ":}" whatever
		// the concept's type; only the typed-math types used to drop them,
		// so "{:14:}" failed a multiple-choice key of 14. Bank answers are
		// left alone: a code answer may contain "{:" legitimately.
		answer = strings.NewReplacer("{:", "", ":}", "").Replace(answer)
		if res, ok := gradeSetLike(gradingType, expected, answer); ok {
			return res
		}
		// A one-sided inequality key ("x > 17/3") is a solution set (Q55).
		switch gradingType {
		case GradingNumeric, GradingSymbolic, GradingExpression:
			if IsInequalityKey(expected) {
				return GradeInequality(expected, answer)
			}
		}
		answer = lenient(gradingType, expected, answer)
		expected = readAlike(gradingType, expected)
	}
	switch gradingType {
	case GradingNumeric:
		res := r.numeric.grade(expected, answer)
		if !res.Correct && !res.Unavailable && fractionAsDecimal(expected, answer) {
			return Result{Correct: true, Score: 1}
		}
		return res
	case GradingPolynomial, GradingExpression, GradingSymbolic:
		// A word answer ("decreasing") compares case-insensitively (fork fix).
		if strings.EqualFold(strings.TrimSpace(expected), strings.TrimSpace(answer)) || samePhrase(expected, answer) {
			return Result{Correct: true, Score: 1}
		}
		// A list of alternatives ("corner/cusp/vertical tangent/
		// discontinuity") accepts any one of them (Q52).
		if parts := wordListParts(expected); parts != nil {
			if wordListAnswerMatches(parts, answer) {
				return Result{Correct: true, Score: 1}
			}
			return Result{Correct: false, Score: 0, Feedback: "Incorrect"}
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
// An inverse is named too: "f^(-1)(x) =", "f^{-1}(x)=", "f⁻¹(x) =".
var leadingAssignRe = regexp.MustCompile(`^\s*(?:[a-zA-Z]\w*(?:\^\(-1\)|\^\{-1\}|\^-1|⁻¹)?(?:\([a-zA-Z]\))?'*|d[a-zA-Z]/d[a-zA-Z])\s*=\s*`)

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

// readAlike puts the expected answer through the same spelling rules as the
// learner's (NormalizeTypedMath), for the typed-math types. A key written
// with a Greek letter or LaTeX ("α (alpha) and β (beta)", "x^{2}") used to
// be compared with a learner's answer that had been rewritten ("alpha
// (alpha) and beta (beta)"), so the key's own text graded wrong (fork fix).
func readAlike(gradingType GradingType, expected string) string {
	switch gradingType {
	case GradingNumeric, GradingPolynomial, GradingExpression, GradingSymbolic, GradingTuple:
		return NormalizeTypedMath(expected, expected)
	}
	return expected
}

// fractionAsDecimal reports a fraction key answered as the same number
// rounded to three or more decimal places (61/3 as 20.333): the same answer
// (fork fix). The key rounded to the answer's places must be the answer, so
// 0.334 is not 1/3, and the answer must have the key's sign and not be
// zero: a flat window of 0.0006 let -0.00015 pass for 1/6561. The numeric
// and the expression graders both accept it, so a key's grader does not
// decide whether a rounded decimal passes.
func fractionAsDecimal(expected, answer string) bool {
	if !strings.Contains(expected, "/") || !strings.Contains(answer, ".") {
		return false
	}
	ev, ok := constEvalNumeric(expected)
	if !ok {
		return false
	}
	a := normalise(answer)
	av, ok := constEvalNumeric(a)
	if !ok {
		return false
	}
	places, ok := decimalPlaces(a)
	if !ok || places < 3 {
		return false
	}
	if av == 0 || (av < 0) != (ev < 0) {
		return false
	}
	return roundsTo(ev, places, av)
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
	for _, p := range []string{"mc:", "out:", "code:", "recall:", "math:", "term:", ProofPrefix} {
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
