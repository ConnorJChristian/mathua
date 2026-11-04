package walkthrough

import (
	"reflect"
	"testing"
)

func TestEncodeSplitRoundTrip(t *testing.T) {
	steps := []string{"Differentiate: $C'(x) = 4x + 2$.", "At $x = 2$: $4 \\cdot 2 + 2 = 10$."}
	enc := Encode("So the marginal cost is 10.", steps)
	explain, got := Split(enc)
	if explain != "So the marginal cost is 10." {
		t.Fatalf("explain = %q", explain)
	}
	if !reflect.DeepEqual(got, steps) {
		t.Fatalf("steps = %q, want %q", got, steps)
	}
	if !HasSteps(enc) || HasSteps("plain") {
		t.Fatal("HasSteps")
	}
	// No steps: the explanation is itself, untouched.
	if Encode("one line", nil) != "one line" {
		t.Fatal("encode with no steps must be the explanation")
	}
	if e, s := Split("one line"); e != "one line" || s != nil {
		t.Fatalf("split plain = %q %v", e, s)
	}
	// Empty steps are dropped, so a blank step never renders.
	if _, s := Split(Encode("x", []string{" ", "a", ""})); !reflect.DeepEqual(s, []string{"a"}) {
		t.Fatalf("blank steps kept: %v", s)
	}
}

func TestBuildWrongAnswerFindsTheStep(t *testing.T) {
	exp := Encode("$df = f_x\\,dx + f_y\\,dy = -1$.", []string{
		"$f_x = 6x - 2y$, which at $(1,-1)$ is 8.",
		"$f_y = -2x + 6y + 1$, which at $(1,-1)$ is -7.",
		"$dx = 0.05$ and $dy = 0.2$, so $df = 8 \\cdot 0.05 + (-7) \\cdot 0.2 = -1$.",
	})
	w := Build("-1", exp, "8", "", nil)
	if w.Answer != "-1" || len(w.Steps) != 3 || w.Explain != "$df = f_x\\,dx + f_y\\,dy = -1$." {
		t.Fatalf("shape: %+v", w)
	}
	if w.DivergedAt == nil || w.DivergedAt.Step != 1 {
		t.Fatalf("an answer equal to step 1's value diverges at step 1: %+v", w.DivergedAt)
	}
	// The grader's diagnosis is the reason when it has one.
	w = Build("-1", exp, "1", "You have the right number with the sign flipped.", nil)
	if w.DivergedAt == nil || w.DivergedAt.Step != 0 || w.DivergedAt.Reason != "You have the right number with the sign flipped." {
		t.Fatalf("diagnosis without a step match: %+v", w.DivergedAt)
	}
	// A number inside a step (an exponent, a point) is not that step's result.
	w = Build("-1", exp, "6", "", nil)
	if w.DivergedAt != nil {
		t.Fatalf("6 is a coefficient, not a step's value: %+v", w.DivergedAt)
	}
}

func TestBuildRightAnswerAndDontKnowShapes(t *testing.T) {
	exp := Encode("why", []string{"s1", "s2 = 4"})
	right := Build("4", exp, "4", "", nil)
	if right.DivergedAt != nil {
		t.Fatalf("a right answer has no divergence: %+v", right.DivergedAt)
	}
	dontKnow := Build("4", exp, "", "", nil)
	if dontKnow.DivergedAt != nil || len(dontKnow.Steps) != 2 || dontKnow.Answer != "4" {
		t.Fatalf("I don't know keeps the steps and no divergence: %+v", dontKnow)
	}
	// Steps is always a list, never null, so a client can read its length.
	none := Build("4", "why", "", "", nil)
	if none.Steps == nil || len(none.Steps) != 0 {
		t.Fatalf("no steps must be an empty list: %#v", none.Steps)
	}
}

func TestBuildLiftsBankAnswerLineAndReading(t *testing.T) {
	exp := "**Answer:** \\(x_0 = 60\\), \\(y_0 = 40\\)\n\nSet the two first-order conditions to zero.\n\nRead more: [Hoy §12.2, Constrained optimisation](https://example.test/hoy)"
	w := Build("", exp, "", "", nil)
	if w.Answer != "\\(x_0 = 60\\), \\(y_0 = 40\\)" {
		t.Fatalf("answer line not lifted: %q", w.Answer)
	}
	if w.Reading == nil || w.Reading.Label != "Hoy §12.2, Constrained optimisation" || w.Reading.URL != "https://example.test/hoy" {
		t.Fatalf("reading not lifted: %+v", w.Reading)
	}
	if w.Explain != "Set the two first-order conditions to zero." {
		t.Fatalf("explain keeps only the prose: %q", w.Explain)
	}
	// A reading the caller supplies wins over the bank's line, which is
	// still removed from the prose.
	w = Build("", exp, "", "", &Reading{Label: "Hoy 12.2, pp. 474-484", URL: "/api/modules/ma-program/files/hoy-3e.pdf", Page: 488})
	if w.Reading.Page != 488 || w.Explain != "Set the two first-order conditions to zero." {
		t.Fatalf("module reading: %+v %q", w.Reading, w.Explain)
	}
	// A source without a link becomes a reading with a label only.
	w = Build("", "**Answer:** 3\n\nBecause.\n\nSource: Hoy 2.1", "", "", nil)
	if w.Reading == nil || w.Reading.Label != "Hoy 2.1" || w.Reading.URL != "" || w.Explain != "Because." {
		t.Fatalf("source line: %+v %q", w.Reading, w.Explain)
	}
}
