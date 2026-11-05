package gradecheck

// Phase 1 independent reviewer (Phase 0 F3): consecutive daily runs draw
// different problems from each other, not only from the fixed source, and
// a rerun on the same day repeats. Covers the first few generator
// concepts, across the UTC midnight that separates two LA evenings.

import (
	"testing"
	"time"
)

func TestReviewer_ConsecutiveDaysDrawDifferentProblems(t *testing.T) {
	if testing.Short() {
		t.Skip("generates against SymPy")
	}
	c := load(t)
	ids := c.Reg.Concepts()
	if len(ids) < 3 {
		t.Fatalf("only %d generator concepts", len(ids))
	}
	ids = ids[:3]
	draw := func(at time.Time) []string {
		var got []string
		c.CheckGenerators(Options{Seeds: 3, Source: DaySeed(at), Concepts: ids, probe: func(q string) { got = append(got, q) }})
		return got
	}
	// 15:59 and 16:01 PST on Nov 4 straddle 00:00 UTC Nov 5: two UTC days.
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Skip(err)
	}
	before := time.Date(2026, 11, 4, 15, 59, 0, 0, la)
	after := time.Date(2026, 11, 4, 16, 1, 0, 0, la)
	if DaySeed(before) == DaySeed(after) {
		t.Fatalf("UTC midnight did not change the seed: %d", DaySeed(before))
	}
	a, a2, b := draw(before), draw(before.Add(-6*time.Hour)), draw(after)
	if len(a) == 0 {
		t.Fatal("no problems drawn")
	}
	if !equalStrings(a, a2) {
		t.Fatal("two runs on the same UTC day drew different problems")
	}
	if equalStrings(a, b) {
		t.Fatalf("consecutive days drew the same problems: %v", a)
	}
	// And a week of days is not one repeating sample.
	seen := map[string]bool{}
	for d := 0; d < 7; d++ {
		seen[joinQ(draw(before.AddDate(0, 0, d)))] = true
	}
	if len(seen) < 6 {
		t.Errorf("7 consecutive days gave only %d distinct samples", len(seen))
	}
}

func joinQ(qs []string) string {
	out := ""
	for _, q := range qs {
		out += q + "\x00"
	}
	return out
}
