package gradecheck

import (
	"testing"
	"time"
)

// The daily QA run must draw new generator problems each day (Phase 0
// review, F3: with seeds live again, a fixed source grades the same three
// problems per generator forever), while `go test` keeps a fixed sample.
func TestDaySeed_ChangesByDayAndNeverZero(t *testing.T) {
	d1 := time.Date(2026, 11, 4, 23, 30, 0, 0, time.UTC)
	d2 := d1.Add(time.Hour) // 2026-11-05 00:30 UTC
	if DaySeed(d1) == DaySeed(d2) {
		t.Fatalf("DaySeed is the same across midnight: %d", DaySeed(d1))
	}
	if DaySeed(d1) != 20261104 {
		t.Fatalf("DaySeed(2026-11-04) = %d, want 20261104", DaySeed(d1))
	}
	// The local day does not matter: the source is the UTC date.
	la, _ := time.LoadLocation("America/Los_Angeles")
	if DaySeed(d1.In(la)) != DaySeed(d1) {
		t.Fatalf("DaySeed depends on the location of its argument")
	}
	if DaySeed(d2) == 0 {
		t.Fatalf("DaySeed is 0, which means the fixed source")
	}
}

// Two runs with the same source draw the same problems; a different source
// draws different ones. One generator concept keeps the test quick.
func TestCheckGenerators_SourceDecidesTheSample(t *testing.T) {
	if testing.Short() {
		t.Skip("generates against SymPy")
	}
	c := load(t)
	ids := c.Reg.Concepts()
	if len(ids) == 0 {
		t.Fatal("no generator concepts")
	}
	questions := func(source int64) []string {
		var got []string
		c.CheckGenerators(Options{Seeds: 3, Source: source, Concepts: ids[:1], Log: nil, probe: func(q string) { got = append(got, q) }})
		return got
	}
	a := questions(0)
	b := questions(0)
	if len(a) == 0 {
		t.Fatalf("no problems generated for %s", ids[0])
	}
	if !equalStrings(a, b) {
		t.Fatalf("the fixed source drew different problems on two runs:\n%v\n%v", a, b)
	}
	var differs bool
	for _, src := range []int64{20261104, 20261105, 20261106, 20261107, 20261108} {
		if !equalStrings(a, questions(src)) {
			differs = true
			break
		}
	}
	if !differs {
		t.Fatalf("five day sources all drew the fixed source's problems for %s: %v", ids[0], a)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
