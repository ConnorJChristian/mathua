package planning

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/concepts"
)

// Fork change: courses share the queue by the pace their due dates need.
func TestPlansOrderPaced_NearerDateGetsMoreOfTheQueue(t *testing.T) {
	var cs []concepts.Concept
	for _, id := range []string{"x1", "x2", "x3", "x4", "y1", "y2", "y3", "y4"} {
		cs = append(cs, concepts.Concept{ID: id, Label: id, Domain: "d", Prerequisites: []string{},
			MasteryThreshold: concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 10}})
	}
	d, err := concepts.Build(cs)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "courses.json")
	os.WriteFile(path, []byte(`[
		{"id":"plan.x","name":"X","grade":"1","description":"","plan":["x1","x2","x3","x4"],"due":"2026-11-01"},
		{"id":"plan.y","name":"Y","grade":"1","description":"","plan":["y1","y2","y3","y4"],"due":"2027-03-01"}
	]`), 0644)
	p, err := Load(path, d)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	rank := p.PlansOrderPaced([]string{"plan.x", "plan.y"}, nil, now)
	if len(rank) != 8 {
		t.Fatalf("rank = %v", rank)
	}
	// X is due in a month, Y in five: all of X comes before the second Y topic.
	for _, x := range []string{"x1", "x2", "x3", "x4"} {
		if rank[x] >= rank["y2"] {
			t.Errorf("%s (rank %d) should come before y2 (rank %d): %v", x, rank[x], rank["y2"], rank)
		}
	}
	if rank["y1"] >= rank["x4"] && rank["y1"] > 4 {
		t.Errorf("y should not be starved entirely: %v", rank)
	}
	// Mastered topics do not count as remaining, so a course nearly done needs less pace.
	done := func(id string) bool { return id == "x1" || id == "x2" || id == "x3" }
	rank2 := p.PlansOrderPaced([]string{"plan.x", "plan.y"}, done, now)
	if rank2["x4"] >= rank2["y2"] {
		t.Errorf("x4 should still precede y2 when X has one topic left: %v", rank2)
	}
}

// Fork fix: prerequisites do not count toward a course's pace. X has 4 plan
// topics due in 20 days (5 days each); its first rests on twelve
// prerequisites, the way a graduate math course rests on a long foundation.
// Y has 10 plan topics due in 100 days (10 days each). Old pacing spread X's
// 16 path concepts over 20 days, so X's first plan topic landed on day 16
// behind half of Y; X should lead, with about two plan topics per Y topic.
func TestPlansOrderPaced_PrerequisitesDoNotInflatePace(t *testing.T) {
	var cs []concepts.Concept
	add := func(id string, prereqs ...string) {
		if prereqs == nil {
			prereqs = []string{}
		}
		cs = append(cs, concepts.Concept{ID: id, Label: id, Domain: "d", Prerequisites: prereqs,
			MasteryThreshold: concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 10}})
	}
	var pre []string
	for j := 1; j <= 12; j++ {
		p := fmt.Sprintf("p%02d", j)
		add(p)
		pre = append(pre, p)
	}
	add("x1", pre...)
	add("x2", "x1")
	add("x3", "x2")
	add("x4", "x3")
	yplan := []string{}
	for i := 1; i <= 10; i++ {
		y := fmt.Sprintf("y%02d", i)
		add(y)
		yplan = append(yplan, `"`+y+`"`)
	}
	d, err := concepts.Build(cs)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "courses.json")
	os.WriteFile(path, []byte(`[
		{"id":"plan.x","name":"X","grade":"1","description":"","plan":["x1","x2","x3","x4"],"due":"2026-10-21"},
		{"id":"plan.y","name":"Y","grade":"1","description":"","plan":[`+strings.Join(yplan, ",")+`],"due":"2027-01-09"}
	]`), 0644)
	p, err := Load(path, d)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	rank := p.PlansOrderPaced([]string{"plan.x", "plan.y"}, nil, now)

	if rank["x1"] > rank["y01"] {
		t.Errorf("X's first plan topic (due in 5 days' pace) should come before Y's first: x1=%d y01=%d", rank["x1"], rank["y01"])
	}
	var plans []string
	for id := range rank {
		if id[0] == 'x' || id[0] == 'y' {
			plans = append(plans, id)
		}
	}
	sort.Slice(plans, func(i, j int) bool { return rank[plans[i]] < rank[plans[j]] })
	xs := 0
	for _, id := range plans[:6] {
		if id[0] == 'x' {
			xs++
		}
	}
	if xs < 3 {
		t.Errorf("first six plan topics should be about 4 X : 2 Y, got %d X: %v", xs, plans[:6])
	}
	for _, pr := range pre {
		if rank[pr] > rank["x1"] {
			t.Errorf("%s (rank %d) should come before x1 (rank %d)", pr, rank[pr], rank["x1"])
		}
	}
}
