package realanalysis

// Regression tests for the 2026-10-25 QA run (run 5).

import (
	"math/rand"
	"regexp"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
)

func run5Samples(t *testing.T, id string, n int) []generator.Problem {
	t.Helper()
	reg := generator.NewRegistry()
	Register(reg)
	var out []generator.Problem
	for i := 0; i < n; i++ {
		p, err := reg.GenerateContext(id, generator.GeneratorContext{Difficulty: []float64{0.3, 0.6, 0.9}[i%3], Seed: int64(9000 + i)})
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		out = append(out, p)
	}
	return out
}

// The worked answer said the transcendental numbers "contain a copy of an
// interval"; they contain no interval. Each set now carries its own reason,
// and the transcendental one goes through the countable algebraic numbers
// (fails on the old text).
func TestRun5CountabilityReasonFitsTheSet(t *testing.T) {
	seen := false
	for i := 0; i < 4000; i++ {
		rand.Seed(int64(i))
		c := countabilityClaim(i%2 == 0)
		if strings.Contains(c.reason, "copy of an interval or of the infinite 0-1 sequences") {
			t.Fatalf("generic reason: %q", c.reason)
		}
		if c.correct == "the transcendental numbers" {
			seen = true
			if !strings.Contains(c.reason, "algebraic numbers are countable") {
				t.Fatalf("transcendental reason: %q", c.reason)
			}
		}
		if c.correct == "the irrational numbers" && strings.Contains(c.reason, "interval") {
			t.Fatalf("the irrationals contain no interval: %q", c.reason)
		}
	}
	if !seen {
		t.Fatal("transcendental set never drawn")
	}
}

// "The extended real line is... compact" was offered as a distractor, but the
// extended real line is compact (two right options; fails on the old text).
func TestRun5ExtendedRealLineHasOneRightOption(t *testing.T) {
	for _, p := range run5Samples(t, "ra.order.extended", 3000) {
		if strings.HasPrefix(p.Question, "The extended real line is...") && strings.Contains(p.Question, ") compact") {
			t.Fatalf("compact offered as a wrong option: %q", p.Question)
		}
	}
}

// Q35 artefacts in the real-analysis generators (fails on the old text).
func TestRun5NoUnitOrZeroArtefacts(t *testing.T) {
	checks := []struct {
		id  string
		bad *regexp.Regexp
	}{
		{"ra.order.archimedean", regexp.MustCompile(`1/ε = \d+/1\b`)},
		{"ra.seq.convergence", regexp.MustCompile(`\(1\(|1\([nN] \+|\+ 0/n|\d+/1 = `)},
		{"ra.order.nested_intervals", regexp.MustCompile(`\b0 \+ \d+/n|x − 0\b`)},
		{"ra.seq.series", regexp.MustCompile(`\(1 \+ … \+ 1/2\)|r\^1·`)},
		{"ra.seq.tests", regexp.MustCompile(`\)\^1\b`)},
	}
	for _, c := range checks {
		for _, p := range run5Samples(t, c.id, 1500) {
			for _, s := range []string{p.Question, p.Explanation} {
				if c.bad.MatchString(s) {
					t.Fatalf("%s: artefact in %q", c.id, s)
				}
			}
		}
	}
}

// Q60: the floor-function item printed its point as "−1 + 1/2" instead of
// one number, −1/2.
func TestDiscontinuityHalfPointIsOneNumber(t *testing.T) {
	seen := 0
	for i := 0; i < 3000; i++ {
		rand.Seed(int64(i + 1))
		c := discontinuityClassifyClaim()
		if strings.Contains(c.q, "+ 1/2") {
			t.Fatalf("point printed as a sum: %q", c.q)
		}
		if strings.Contains(c.q, "/2?") {
			seen++
		}
	}
	if seen == 0 {
		t.Fatal("no half-integer floor item drawn; the check would pass vacuously")
	}
}
