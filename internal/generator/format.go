package generator

import (
	"strconv"
	"strings"
)

// AnswerFormat is what the learner should type and which keyboard the field
// should offer.
//
// It exists because grading_type describes the *grader*, not the answer. A
// concept declared numeric may be graded by its own Grade method that wants
// "5 R 3", and a symbolic concept may want the word "yes". Deriving the input
// from the type alone put a digits-only keypad in front of answers that could
// not express, and learners reported the questions as broken formatting rather
// than as an input problem, which is exactly what they looked like.
type AnswerFormat struct {
	// Hint is one line under the field naming the syntax.
	Hint string
	// InputMode is "numeric" or "text". Text is the safe default: a text field
	// can accept a number, a numeric keypad cannot accept a letter.
	InputMode string
}

// FormattedGenerator is a Generator that knows the syntax of its own answers.
//
// Optional. A generator that does not implement it gets the default for its
// concept's grading type, which is right for the overwhelming majority. The
// ones that need it are those whose Grade method accepts something the type
// does not describe.
type FormattedGenerator interface {
	Generator
	AnswerFormat() AnswerFormat
}

// DefaultAnswerFormat is the type-based default. These strings are the single
// source for the hint the learner reads: the client no longer keeps its own
// copy, it renders what the server sends and falls back to text.
func DefaultAnswerFormat(gradingType string) AnswerFormat {
	switch gradingType {
	case "numeric":
		return AnswerFormat{
			Hint:      "Answer with a number, fraction, or mixed number (e.g. 4 1/10)",
			InputMode: "numeric",
		}
	case "expression", "polynomial":
		return AnswerFormat{Hint: "Answer as an expression", InputMode: "text"}
	case "symbolic":
		return AnswerFormat{Hint: "Answer in simplest form", InputMode: "text"}
	case "multiple_choice":
		return AnswerFormat{Hint: "Type the exact answer", InputMode: "text"}
	case "comparison":
		return AnswerFormat{Hint: "Answer <, >, or =", InputMode: "text"}
	case "ordering":
		return AnswerFormat{Hint: "Answer in order, separated by commas", InputMode: "text"}
	case "tuple":
		return AnswerFormat{Hint: "Separate values with commas, e.g. 2, 3 or (2, 3)", InputMode: "text"}
	case "complex":
		return AnswerFormat{Hint: "Answer as a + bi", InputMode: "text"}
	default:
		return AnswerFormat{Hint: "Answer in the form shown", InputMode: "text"}
	}
}

// AnswerFormatFor merges a generator's own declaration over the default for its
// type. An empty field in the declaration means "keep the default".
func AnswerFormatFor(gen Generator, gradingType string) AnswerFormat {
	f := DefaultAnswerFormat(gradingType)
	formatted, ok := gen.(FormattedGenerator)
	if !ok {
		return f
	}
	own := formatted.AnswerFormat()
	if strings.TrimSpace(own.Hint) != "" {
		f.Hint = own.Hint
	}
	if own.InputMode != "" {
		f.InputMode = own.InputMode
	}
	return f
}

// Sq writes n squared for an explanation: "3^2", or "(-3)^2" for a negative
// n. "-3^2" reads as -(3^2) = -9, so a distance worked as sqrt(-3^2 + -4^2)
// was wrong as written (Q29).
func Sq(n int) string {
	if n < 0 {
		return "(" + strconv.Itoa(n) + ")^2"
	}
	return strconv.Itoa(n) + "^2"
}
