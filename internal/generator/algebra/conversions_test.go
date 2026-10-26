package algebra

// Tests for the seven converted concepts.
//
// Each recomputes the answer from the published facts instead of trusting the
// generator's arithmetic, and each is checked against its shipped grading type
// so a corpus row that drifts back to a table-driven type fails here. The seven
// were fixed questions before this batch; the exemptions that let them ship
// without a schema are gone, so a regression would also fail the coverage test.

import (
	"strconv"
	"strings"
	"testing"
)

var convertedIDs = []string{
	"alg.eq.extraneous_roots",
	"alg.func.domain",
	"alg.func.graph_analysis",
	"alg.func.injectivity",
	"alg.func.irrational",
	"alg.func.power",
	"alg.systems.concept",
}

func convFactInt(t *testing.T, facts map[string]string, key string) int {
	t.Helper()
	raw, ok := facts[key]
	if !ok {
		t.Fatalf("missing fact %q", key)
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		t.Fatalf("fact %q is not an integer: %q", key, raw)
	}
	return v
}

func convPow(base, exp int) int {
	out := 1
	for i := 0; i < exp; i++ {
		out *= base
	}
	return out
}

func TestConvertedConceptsAreNumericInTheCorpus(t *testing.T) {
	types := shippedTypes(t)
	for _, id := range convertedIDs {
		if types[id] != "numeric" {
			t.Errorf("%s: corpus says %q, want numeric", id, types[id])
		}
	}
}

func TestConvertedAnswersRecomputeFromFacts(t *testing.T) {
	reg := batchRegistry(t)

	// alg.func.domain answers an interval now; functions_sets_test.go checks
	// it point by point against the formula.

	for i := 0; i < 100; i++ {
		p := sample(t, reg, "alg.func.irrational", i)
		radicand := convFactInt(t, p.Facts, "radicand")
		answer := convFactInt(t, p.Facts, "answer")
		switch p.Facts["root"] {
		case "square":
			if answer*answer != radicand {
				t.Fatalf("irrational: %d^2 != %d", answer, radicand)
			}
		case "cube":
			if answer*answer*answer != radicand {
				t.Fatalf("irrational: %d^3 != %d", answer, radicand)
			}
		default:
			t.Fatalf("irrational: unexpected root %q", p.Facts["root"])
		}
	}

	for i := 0; i < 100; i++ {
		p := sample(t, reg, "alg.func.power", i)
		base := convFactInt(t, p.Facts, "base")
		exponent := convFactInt(t, p.Facts, "exponent")
		if p.Answer != strconv.Itoa(convPow(base, exponent)) {
			t.Fatalf("power: %d^%d answer=%s", base, exponent, p.Answer)
		}
	}

	for i := 0; i < 100; i++ {
		p := sample(t, reg, "alg.func.graph_analysis", i)
		slope := convFactInt(t, p.Facts, "slope")
		intercept := convFactInt(t, p.Facts, "intercept")
		x := convFactInt(t, p.Facts, "answer")
		if slope*x+intercept != 0 {
			t.Fatalf("graph: %dx + %d is not zero at x=%d", slope, intercept, x)
		}
	}

	for i := 0; i < 100; i++ {
		p := sample(t, reg, "alg.eq.extraneous_roots", i)
		a := convFactInt(t, p.Facts, "a")
		b := convFactInt(t, p.Facts, "b")
		parts := strings.Split(p.Facts["candidates"], ",")
		if len(parts) != 2 {
			t.Fatalf("extraneous: candidates %q", p.Facts["candidates"])
		}
		p0, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		q0, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
		answer := convFactInt(t, p.Facts, "answer")
		other := p0
		if answer == p0 {
			other = q0
		}
		if answer != p0 && answer != q0 {
			t.Fatalf("extraneous: answer %d is not a candidate (%d, %d)", answer, p0, q0)
		}
		if answer < b {
			t.Fatalf("extraneous: answer %d is below b=%d and cannot satisfy the original", answer, b)
		}
		if other >= b {
			t.Fatalf("extraneous: the other candidate %d is not extraneous (b=%d)", other, b)
		}
		if (answer-b)*(answer-b) != answer+a {
			t.Fatalf("extraneous: %d does not solve the squared equation", answer)
		}
	}

	for i := 0; i < 100; i++ {
		p := sample(t, reg, "alg.systems.concept", i)
		s := convFactInt(t, p.Facts, "s")
		d := convFactInt(t, p.Facts, "d")
		x := convFactInt(t, p.Facts, "answer")
		if 2*x-s != d {
			t.Fatalf("systems: x=%d does not give x-y=%d when x+y=%d", x, d, s)
		}
	}

	injective := map[string]bool{
		"2x + 3": true, "-4x + 1": true, "5x - 2": true, "x^3": true,
		"x^2": false, "|x|": false, "x^2 + 1": false, "x^4": false,
	}
	for i := 0; i < 100; i++ {
		p := sample(t, reg, "alg.func.injectivity", i)
		segments := strings.Split(p.Facts["functions"], ";")
		if len(segments) != 3 {
			t.Fatalf("injectivity: functions %q", p.Facts["functions"])
		}
		count := 0
		for _, seg := range segments {
			eq := strings.Index(seg, "=")
			if eq < 0 {
				t.Fatalf("injectivity: segment %q has no equals sign", seg)
			}
			body := strings.TrimSpace(seg[eq+1:])
			yes, ok := injective[body]
			if !ok {
				t.Fatalf("injectivity: unknown function body %q", body)
			}
			if yes {
				count++
			}
		}
		if p.Answer != strconv.Itoa(count) {
			t.Fatalf("injectivity: answer=%s want %d from %q", p.Answer, count, p.Facts["functions"])
		}
	}
}
