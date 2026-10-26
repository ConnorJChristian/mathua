package pytrack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Term items (type the word) and Spanish Sí/No questions, as the marx
// module's Harnecker bank writes them.
func TestTermItems(t *testing.T) {
	dir := t.TempDir()
	bank := `{"chapter":"x","title":"X","concepts":[{"id":"mpe.x.t","book":{"name":"Harnecker","section":"cap. 3","title":"Las fuerzas productivas","url":""},"intro":"","items":[
{"id":"t1","kind":"term","level":1,"prompt":"¿Cómo se llama el excedente de valor?","correct":"plusvalía","accept":["plusvalor"],"explain":"Según el manual (cap. 3, p. 84)."},
{"id":"t2","kind":"term","level":1,"prompt":"¿Y la clausura de empresas contra la huelga?","correct":"lock-out","explain":"Cap. 4, p. 99."},
{"id":"s1","kind":"mc","level":1,"prompt":"¿Existió siempre el Estado?","correct":"No","wrong":["Sí"],"explain":"Cap. 7, pp. 85–86."}]}]}`
	if err := os.WriteFile(filepath.Join(dir, "x.json"), []byte(bank), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := LoadBank(dir)
	if err != nil {
		t.Fatal(err)
	}
	tr := &Track{Bank: b, Runner: NewRunner()}
	c := b.Concept("mpe.x.t")

	p := tr.Problem(c, &c.Items[0])
	if p.Question != "¿Cómo se llama el excedente de valor?\n\n"+TermMarker {
		t.Fatalf("question: %q", p.Question)
	}
	if strings.Contains(p.Question, "plusval") || strings.Contains(p.Answer, "plusval") {
		t.Error("the expected term leaked into the served question or answer key")
	}
	if !strings.HasPrefix(p.Explanation, "**Answer:** plusvalía") || !strings.Contains(p.Explanation, "Source: Las fuerzas productivas") {
		t.Errorf("explanation: %q", p.Explanation)
	}
	for _, tc := range []struct {
		answer string
		ok     bool
	}{
		{"plusvalía", true},
		{"Plusvalia", true},
		{"  la plusvalía. ", true},
		{"«PLUSVALÍA»", true},
		{"plusvalor", true},
		{"plusvalía relativa", false},
		{"valor", false},
		{"", false},
	} {
		g := tr.Grade(p.Answer, tc.answer)
		if g.Correct != tc.ok || g.Unavailable {
			t.Errorf("%q: got %+v, want correct=%v", tc.answer, g, tc.ok)
		}
	}
	lock := tr.Problem(c, &c.Items[1])
	for _, a := range []string{"lock-out", "Lockout", "lock out", "LOCK–OUT"} {
		if g := tr.Grade(lock.Answer, a); !g.Correct {
			t.Errorf("%q should match lock-out: %+v", a, g)
		}
	}

	// Sí/No is a binary choice served Sí first, whichever is correct.
	sn := tr.Problem(c, &c.Items[2])
	if !strings.HasSuffix(sn.Question, "A) Sí\n\nB) No") {
		t.Errorf("sí/no order: %q", sn.Question)
	}
	if g := tr.Grade(sn.Answer, "No"); !g.Correct {
		t.Error("No should be correct")
	}
	if g := tr.Grade(sn.Answer, "Sí"); g.Correct {
		t.Error("Sí should be wrong")
	}

	// A term item and a Sí/No item are objective: a test may serve them.
	if n := (&conceptGen{track: tr, c: c}).ObjectiveItems(); n != 3 {
		t.Errorf("objective items = %d, want 3", n)
	}

	for name, bad := range map[string]string{
		"empty term":      `"correct":"plusvalía","accept":["plusvalor"]` + "|" + `"correct":" ","accept":["plusvalor"]`,
		"two-line term":   `"correct":"plusvalía","accept":["plusvalor"]` + "|" + `"correct":"plus\nvalía","accept":["plusvalor"]`,
		"empty accept":    `"accept":["plusvalor"]` + "|" + `"accept":[""]`,
		"term with wrong": `"accept":["plusvalor"]` + "|" + `"accept":["plusvalor"],"wrong":["valor"]`,
		"no explanation":  `"explain":"Según el manual (cap. 3, p. 84)."` + "|" + `"explain":""`,
		"sí with no pair": `"correct":"No","wrong":["Sí"]` + "|" + `"correct":"No","wrong":["Quizás"]`,
	} {
		parts := strings.SplitN(bad, "|", 2)
		s := strings.Replace(bank, parts[0], parts[1], 1)
		if s == bank {
			t.Fatalf("%s: the test edit did not apply", name)
		}
		if err := os.WriteFile(filepath.Join(dir, "x.json"), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadBank(dir); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestNormalizeTerm(t *testing.T) {
	for in, want := range map[string]string{
		"Plusvalía":                    "plusvalia",
		"  la Composición  Orgánica. ": "composicion organica",
		"lock-out":                     "lock out",
		"¿Fuerza de trabajo?":          "fuerza de trabajo",
		"el":                           "el",
		"":                             "",
	} {
		if got := normalizeTerm(in); got != want {
			t.Errorf("normalizeTerm(%q) = %q, want %q", in, got, want)
		}
	}
}
