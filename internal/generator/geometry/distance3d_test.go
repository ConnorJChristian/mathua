package geometry

import (
	"math/rand"
	"regexp"
	"strconv"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
)

// The key is recomputed from the two printed points, and the pool is wide:
// it used to be five fixed points, all measured from the origin.
func TestDistance3DKeyAndVariety(t *testing.T) {
	pt := regexp.MustCompile(`\((-?\d+),(-?\d+),(-?\d+)\)`)
	seen := map[string]bool{}
	offOrigin := 0
	for i := 0; i < 300; i++ {
		rand.Seed(int64(i + 1))
		p := (&distance3DGen{}).Generate(generator.GeneratorContext{Difficulty: 0.9})
		m := pt.FindAllStringSubmatch(p.Question, -1)
		if len(m) != 2 {
			t.Fatalf("want two points in %q", p.Question)
		}
		sum := 0
		for k := 1; k <= 3; k++ {
			a, _ := strconv.Atoi(m[0][k])
			b, _ := strconv.Atoi(m[1][k])
			sum += (b - a) * (b - a)
		}
		d, _ := strconv.Atoi(p.Answer)
		if d*d != sum {
			t.Fatalf("%q: key %d but the squared distance is %d", p.Question, d, sum)
		}
		if m[0][0] != "(0,0,0)" {
			offOrigin++
		}
		seen[p.Question] = true
	}
	if len(seen) < 200 || offOrigin < 250 {
		t.Errorf("%d distinct of 300, %d off the origin", len(seen), offOrigin)
	}
}
