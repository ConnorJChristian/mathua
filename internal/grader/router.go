package grader

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
	switch gradingType {
	case GradingNumeric:
		return r.numeric.grade(expected, answer)
	case GradingPolynomial, GradingExpression, GradingSymbolic:
		return gradeSymPy(expected, answer)
	case GradingMultipleChoice:
		return r.choice.grade(expected, answer)
	case GradingComparison:
		return r.comparison.grade(expected, answer)
	case GradingOrdering:
		return r.ordering.grade(expected, answer)
	case GradingTuple:
		return r.tuple.grade(expected, answer)
	case GradingComplex:
		return r.complex.grade(expected, answer)
	case GradingPython:
		return gradePython(expected, answer)
	default:
		return r.tuple.grade(expected, answer)
	}
}

func (r *Router) Close() {}

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
