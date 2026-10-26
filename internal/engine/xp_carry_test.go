package engine

import (
	"testing"

	xppolicy "github.com/chuma-beep/mathua/internal/xp"
)

// Whole-XP payouts carry the fraction forward, so ten easy and ten hard answers
// in a two-minute topic pay what they are worth (about 18 and 33) instead of
// both rounding to the same per-answer figure; each answer pays at least 1.
func TestAwardXPCarriesFractions(t *testing.T) {
	e := &Engine{}
	d := func(v float64) *float64 { return &v }
	sum := func(student string, diff float64) int {
		total := 0
		for i := 0; i < 10; i++ {
			paid := e.awardXP(student, xppolicy.AwardValue(true, 60, 120, d(diff)))
			if paid < 1 {
				t.Fatalf("a correct answer paid %d", paid)
			}
			total += paid
		}
		return total
	}
	easy, hard := sum("a", 0.3), sum("b", 1.0)
	if easy < 17 || easy > 18 || hard < 32 || hard > 33 {
		t.Errorf("10 easy paid %d (want ~17.6), 10 hard paid %d (want ~33)", easy, hard)
	}
	if e.awardXP("c", 0) != 0 {
		t.Error("a wrong answer paid XP")
	}
}
