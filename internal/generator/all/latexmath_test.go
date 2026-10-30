package all

// Regression tests for the LaTeX over-escape defect found in the curriculum
// audit: a math span whose entire content is a number canonicalises to
// literal dollars, so a learner reads "$16$" instead of 16.
//
// The classifier test pins the rule itself. The generated-math test is the
// corpus-wide ratchet: every registered generator is sampled at three
// difficulties and both fields are canonicalised exactly as the server does.
// A single regenerated instance that writes a numeral inside math fails the
// build, which is the point — the audit found 59 such generators and the
// fix must not quietly come back.

import (
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/latex"
	"github.com/chuma-beep/mathua/internal/solutions"
)

// overescapedNumber matches a paired escaped number, which is the undefined
// rendering: \$16\$ is rendered as the characters $16$ rather than as 16.
var overescapedNumber = regexp.MustCompile(`\\\$[0-9][0-9.,]*\\\$`)

// overescapedHit returns the first paired escaped number in s, or "". A
// closer that is the first character of a display opener (\$5$$) is not a
// pair, which is the same exception the canonicaliser uses.
func overescapedHit(s string) string {
	for _, loc := range overescapedNumber.FindAllStringIndex(s, -1) {
		if loc[1] < len(s) && s[loc[1]] == '$' {
			continue
		}
		return s[loc[0]:loc[1]]
	}
	return ""
}

func TestOverescapedNumberClassifier(t *testing.T) {
	cases := []struct{ in, want string }{
		{`\(1\)`, `\$1\$`},
		{`\(0\)`, `\$0\$`},
		{`$1$`, `\$1\$`},
		{`\(16\) points`, `\$16\$`},
		// A minus sign is another token, so a negative number is not over-escaped.
		{`\(-3\)`, ""},
		{`price \$5 and \$7`, ""},
		{`\(\frac{1}{2}\)`, ""},
		{`\(4p = 3\)`, ""},
		{`\(x = 1\)`, ""},
		{`\(360^{\circ}\)`, ""},
		{`about 50% of the time`, ""},
		{`\(50\%\)`, ""},
		{`\(12.5\%\)`, ""},
		{`The answer is 16.`, ""},
	}
	for _, tc := range cases {
		out := latex.Canonicalize(tc.in, latex.Generators)
		if got := overescapedHit(out); got != tc.want {
			t.Errorf("Canonicalize(%q) = %q, flagged %q, want %q", tc.in, out, got, tc.want)
		}
	}
}

func TestGeneratedMathAvoidsOverescapedNumbers(t *testing.T) {
	const perDifficulty = 5
	reg := Registry()
	// The server assembles explanations from the worked solutions
	// (data/lessons/solutions); a schema that wraps a lone number in math
	// (Q66: "roots \\([[r1]]\\)" -> "\\$3\\$") is caught only on that path.
	l, err := solutions.Load("../../../data/lessons/solutions")
	if err != nil || len(l.Concepts()) == 0 {
		t.Fatalf("load worked solutions: %v", err)
	}
	reg.SetSolutions(l)
	ids := reg.Concepts()
	sort.Strings(ids)

	var offenders []string
	for i, id := range ids {
		for di, difficulty := range []float64{0.12, 0.5, 0.9} {
			for k := 0; k < perDifficulty; k++ {
				rand.Seed(int64(i*1000 + di*100 + k))
				p, err := reg.Generate(id, difficulty)
				if err != nil {
					t.Fatalf("%s: %v", id, err)
				}
				for _, field := range []string{p.Question, p.Explanation} {
					out := latex.Canonicalize(field, latex.Generators)
					if hit := overescapedHit(out); hit != "" {
						offenders = append(offenders, fmt.Sprintf("%s (%s)", id, hit))
					}
				}
			}
		}
	}
	if len(offenders) > 0 {
		shown := offenders
		if len(shown) > 10 {
			shown = shown[:10]
		}
		t.Fatalf("%d generated strings canonicalise a lone number into literal dollars; write bare numerals as text: %s",
			len(offenders), strings.Join(shown, "; "))
	}
}
