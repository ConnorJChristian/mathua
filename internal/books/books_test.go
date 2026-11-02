package books

import (
	"reflect"
	"testing"
)

func TestIDResolvesFilesTagsNamesAndDecks(t *testing.T) {
	cases := map[string]string{
		"ma-program/hoy-3e.pdf":                      "hoy",
		"ma-program/hoy-supplement.pdf":              "hoy",
		"Hoy et al., Mathematics for Economics (3e)": "hoy",
		"Hoy_Mathematics_for_Economics":              "hoy",
		"Schaums_Calc_Econ":                          "schaums-calc-econ",
		"ma-program/schaums-math-methods-2e.pdf":     "schaums-math-methods",
		"Schaums_Calculus_Mendelson":                 "schaums-calculus",
		"Schaums_Linear_Algebra":                     "schaums-linear-algebra",
		"Axler_Linear_Algebra_Done_Right":            "axler",
		"ma-program/axler-ladr-4e.pdf":               "axler",
		"Abbott, Understanding Analysis (2e)":        "abbott",
		"ma-program/casella-berger.pdf":              "casella-berger",
		"Wooldridge - Introductory Econometrics 7e":  "wooldridge",
		"web:Wooldridge":                             "wooldridge",
		"ma-program/heiss-python-econometrics.pdf":   "heiss",
		"Python Data Science Handbook":               "pdsh",
		"web:Statistical Learning":                   "islp",
		"ma-program/esl-2e.pdf":                      "esl",
		"ma-program/hansen-prob-stats.pdf":           "hansen-prob-stats",
		"Some Local Notes":                           "some-local-notes",
		"":                                           "",
	}
	for in, want := range cases {
		if got := ID(in); got != want {
			t.Errorf("ID(%q) = %q, want %q", in, got, want)
		}
	}
	// A deck's course name is no book.
	if got := Known("Mathematical Economics I (Hoy et al., Schaum's, Axler)"); got != "" {
		t.Errorf("Known(course deck) = %q, want none", got)
	}
}

func TestOrderCourseTextsThenPreferenceThenSchaums(t *testing.T) {
	t.Setenv(PreferenceEnv, "")
	got := Order([]string{"leithold", "axler", "schaums-linear-algebra", "schaums-calc-econ", "wooldridge"}, []string{"wooldridge"})
	want := []string{"wooldridge", "schaums-calc-econ", "schaums-linear-algebra", "axler", "leithold"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Order = %v, want %v", got, want)
	}
	t.Setenv(PreferenceEnv, "Leithold, axler")
	got = Order([]string{"schaums-calc-econ", "axler", "leithold"}, nil)
	want = []string{"leithold", "axler", "schaums-calc-econ"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Order with %s = %v, want %v", PreferenceEnv, got, want)
	}
}

func TestRegisterAddsALocalBook(t *testing.T) {
	Register("my-notes", "My notes", "notes/week1.pdf", "Week One Notes")
	if got := ID("Week One Notes"); got != "my-notes" {
		t.Fatalf("ID = %q", got)
	}
	if Label("my-notes") != "My notes" || Label("hoy") != "Hoy" {
		t.Fatalf("labels %q %q", Label("my-notes"), Label("hoy"))
	}
}
