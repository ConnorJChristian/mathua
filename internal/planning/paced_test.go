package planning

import (
	"os"
	"path/filepath"
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
