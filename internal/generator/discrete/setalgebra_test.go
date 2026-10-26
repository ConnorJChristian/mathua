package discrete

import (
	"math/rand"
	"regexp"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
)

// A roster inside math must escape its braces, or LaTeX reads them as
// grouping and "A = {1, 2, 3}" renders as "A = 1, 2, 3".
var bareBraceRoster = regexp.MustCompile(`(^|[^\\])\{-?\d+(, ?-?\d+)*\}`)

func TestSetRostersRenderWithBraces(t *testing.T) {
	for i := 0; i < 300; i++ {
		rand.Seed(int64(i))
		d := []float64{0.3, 0.6, 0.9}[i%3]
		p := (&setOpsGen{}).Generate(generator.GeneratorContext{Difficulty: d})
		for _, m := range regexp.MustCompile(`\\\((.*?)\\\)`).FindAllStringSubmatch(p.Question, -1) {
			if bareBraceRoster.MatchString(m[1]) {
				t.Fatalf("unescaped roster in math: %q", p.Question)
			}
		}
	}
}

// Difficulty changes the kind of question: one operation on two rosters at
// the bottom, compound expressions, three sets or products at the top.
func TestSetOpsDifficultyChangesTheShape(t *testing.T) {
	compound := regexp.MustCompile(`\\cup \(|\\cap \(|\\bar\{A\} \\cap|\(A - B\) \\cup|\\overline\{A \\cap|\\times|\\mathcal\{P\}`)
	count := func(d float64) (n int, distinct int) {
		seen := map[string]bool{}
		for i := 0; i < 200; i++ {
			rand.Seed(int64(1000 + i))
			p := (&setOpsGen{}).Generate(generator.GeneratorContext{Difficulty: d})
			seen[p.Question] = true
			if compound.MatchString(p.Question) {
				n++
			}
		}
		return n, len(seen)
	}
	low, lowDistinct := count(0.3)
	high, highDistinct := count(0.95)
	if low != 0 {
		t.Errorf("%d compound questions at difficulty 0.3, want none", low)
	}
	if high < 150 {
		t.Errorf("only %d of 200 compound questions at difficulty 0.95", high)
	}
	if lowDistinct < 100 || highDistinct < 100 {
		t.Errorf("distinct stems: %d at 0.3, %d at 0.95; want at least 100 of 200", lowDistinct, highDistinct)
	}
}

// The key is a set: a reordered roster is right, a roster missing an
// element is wrong, and {} is the empty set.
func TestSetOpsGradesRostersAsSets(t *testing.T) {
	g := &setOpsGen{}
	for i := 0; i < 200; i++ {
		rand.Seed(int64(i))
		p := g.Generate(generator.GeneratorContext{Difficulty: []float64{0.3, 0.6, 0.9}[i%3]})
		if p.Answer == "∅" {
			if !g.Grade(p.Answer, "{}").Correct {
				t.Fatalf("{} rejected for ∅ in %q", p.Question)
			}
			continue
		}
		inner := strings.TrimSuffix(strings.TrimPrefix(p.Answer, "{"), "}")
		parts := splitTopLevel(inner)
		rev := make([]string, len(parts))
		for j := range parts {
			rev[j] = parts[len(parts)-1-j]
		}
		if !g.Grade(p.Answer, "{"+strings.Join(rev, ", ")+"}").Correct {
			t.Fatalf("reordered roster rejected for %q", p.Answer)
		}
		if len(parts) > 1 && g.Grade(p.Answer, "{"+strings.Join(parts[1:], ",")+"}").Correct {
			t.Fatalf("roster missing %s accepted for %q", parts[0], p.Answer)
		}
	}
}

func splitTopLevel(s string) []string {
	var out []string
	depth, start := 0, 0
	for i, r := range s {
		switch r {
		case '(', '{':
			depth++
		case ')', '}':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, s[start:i])
				start = i + 1
			}
		}
	}
	return append(out, s[start:])
}

// The two-set counts are consistent: recompute every key from the numbers
// in the question.
func TestVennKeysFollowFromTheQuestion(t *testing.T) {
	nums := regexp.MustCompile(`\d+`)
	for i := 0; i < 300; i++ {
		rand.Seed(int64(i))
		p := (&vennGen{}).Generate(generator.GeneratorContext{Difficulty: []float64{0.3, 0.6, 0.9}[i%3]})
		head := p.Question[:strings.Index(p.Question, "How many")]
		var v []int
		for _, s := range nums.FindAllString(head, -1) {
			n := 0
			for _, c := range s {
				n = n*10 + int(c-'0')
			}
			v = append(v, n)
		}
		total, a, b := v[0], v[1], v[2]
		var both, none int
		if strings.Contains(p.Question, "do neither") {
			none = v[3]
			both = a + b - (total - none)
		} else {
			both = v[3]
			none = total - (a + b - both)
		}
		if both < 0 || none < 0 || both > a || both > b {
			t.Fatalf("inconsistent counts in %q", p.Question)
		}
		var want int
		switch {
		case strings.Contains(p.Question, "A \\cup B\\)?"):
			want = a + b - both
		case strings.Contains(p.Question, "A - B"):
			want = a - both
		case strings.Contains(p.Question, "neither set"):
			want = none
		case strings.Contains(p.Question, "exactly one"):
			want = a + b - 2*both
		case strings.Contains(p.Question, "A \\cap B"):
			want = both
		default:
			t.Fatalf("unrecognised question %q", p.Question)
		}
		if p.Answer != itoa(want) {
			t.Fatalf("key %s, want %d: %q", p.Answer, want, p.Question)
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// Relation, partition and function keys are re-derived from the printed
// pairs and blocks, and the pool is wide: it used to be nine fixed items.
func TestRelationKeysFollowFromTheQuestion(t *testing.T) {
	pairRe := regexp.MustCompile(`\((\d), (\d)\)`)
	seen := map[string]bool{}
	for i := 0; i < 300; i++ {
		rand.Seed(int64(i + 7))
		p := (&relationsGen{}).Generate(generator.GeneratorContext{Difficulty: 0.3})
		seen[p.Question] = true
		has := map[[2]int]bool{}
		var rel [][2]int
		for _, m := range pairRe.FindAllStringSubmatch(p.Question, -1) {
			pr := [2]int{int(m[1][0] - '0'), int(m[2][0] - '0')}
			has[pr] = true
			rel = append(rel, pr)
		}
		want := true
		switch {
		case strings.Contains(p.Question, "reflexive"):
			for a := 1; a <= 3; a++ {
				want = want && has[[2]int{a, a}]
			}
		case strings.Contains(p.Question, "symmetric"):
			for _, r := range rel {
				want = want && has[[2]int{r[1], r[0]}]
			}
		default:
			for _, r := range rel {
				for _, s := range rel {
					if r[1] == s[0] {
						want = want && has[[2]int{r[0], s[1]}]
					}
				}
			}
		}
		if p.Answer != map[bool]string{true: "yes", false: "no"}[want] {
			t.Fatalf("key %s for %q", p.Answer, p.Question)
		}
	}
	if len(seen) < 100 {
		t.Errorf("only %d distinct relation questions of 300", len(seen))
	}
}
