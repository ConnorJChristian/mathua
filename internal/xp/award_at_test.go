package xp

import "testing"

// XP follows the difficulty a question was served at: in a two-minute topic an
// easy question is worth less than a medium one and a hard one more; unknown
// difficulty keeps the medium price; a wrong answer is worth nothing, and a
// sub-minute drill is still worth the 1-XP floor.
func TestAwardValueScalesWithDifficulty(t *testing.T) {
	d := func(v float64) *float64 { return &v }
	const threshold = 120 // seconds; answered within it, so x1.1
	easy := AwardValue(true, 60, threshold, d(0.3))
	medium := AwardValue(true, 60, threshold, nil)
	hard := AwardValue(true, 60, threshold, d(1.0))
	if !(easy < medium && medium < hard) {
		t.Errorf("easy %.2f, medium %.2f, hard %.2f: want strictly increasing", easy, medium, hard)
	}
	if float64(Award(true, 60, threshold, 0, "lesson")) != float64(int(medium)) {
		t.Errorf("unknown difficulty %.2f is not the medium price %d", medium, Award(true, 60, threshold, 0, "lesson"))
	}
	if AwardValue(false, 60, threshold, d(1.0)) != 0 {
		t.Error("a wrong answer is worth XP")
	}
	if got := AwardValue(true, 5, 10, d(0.3)); got < 1 || got > 1.1 {
		t.Errorf("a sub-minute drill is worth %.2f, want the 1-XP floor", got)
	}
}
