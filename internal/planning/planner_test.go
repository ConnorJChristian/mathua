package planning

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/storage"
)

func testPlanner(t *testing.T) *Planner {
	t.Helper()
	d, err := concepts.Build([]concepts.Concept{
		{ID: "a", Label: "Concept A", Domain: "d", Subdomain: "s-a", Prerequisites: []string{},
			MasteryThreshold: concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 10}},
		{ID: "b", Label: "Concept B", Domain: "d", Subdomain: "s-b", Prerequisites: []string{"a"},
			MasteryThreshold: concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 10}},
		{ID: "c", Label: "Concept C", Domain: "d", Subdomain: "s-c", Prerequisites: []string{"a"},
			MasteryThreshold: concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 10}},
		{ID: "d", Label: "Concept D", Domain: "d", Subdomain: "s-d", Prerequisites: []string{"b", "c"},
			MasteryThreshold: concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 10}},
	})
	if err != nil {
		t.Fatalf("build DAG: %v", err)
	}
	dir := t.TempDir()
	coursesPath := filepath.Join(dir, "courses.json")
	os.WriteFile(coursesPath, []byte(`[
		{"id":"test","name":"Test","grade":"1","description":"d","targets":["d"]}
	]`), 0644)
	p, err := Load(coursesPath, d)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return p
}

func TestPlanner_Courses(t *testing.T) {
	p := testPlanner(t)
	if len(p.Courses()) != 1 {
		t.Errorf("expected 1 course, got %d", len(p.Courses()))
	}
}

func TestPrerequisiteChain(t *testing.T) {
	p := testPlanner(t)
	path, err := p.PrerequisiteChain([]string{"d"})
	if err != nil {
		t.Fatalf("chain: %v", err)
	}
	ids := make([]string, len(path.Concepts))
	for i, c := range path.Concepts {
		ids[i] = c.ID
	}
	expected := []string{"a", "b", "c", "d"}
	if len(ids) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, ids)
	}
	for i, e := range expected {
		if ids[i] != e {
			t.Errorf("position %d: expected %q, got %q", i, e, ids[i])
		}
	}
}

func TestPrerequisiteChain_Subset(t *testing.T) {
	p := testPlanner(t)
	path, err := p.PrerequisiteChain([]string{"b"})
	if err != nil {
		t.Fatalf("chain: %v", err)
	}
	ids := make([]string, len(path.Concepts))
	for i, c := range path.Concepts {
		ids[i] = c.ID
	}
	// Should include "a" and "b" but not "c" or "d"
	if len(ids) != 2 || ids[0] != "a" || ids[1] != "b" {
		t.Errorf("expected [a b], got %v", ids)
	}
}

func TestPathForCourse(t *testing.T) {
	p := testPlanner(t)
	path, err := p.PathForCourse("test")
	if err != nil {
		t.Fatalf("path: %v", err)
	}
	if len(path.Concepts) != 4 {
		t.Errorf("expected 4 concepts in path, got %d", len(path.Concepts))
	}
	if path.Course == nil || path.Course.Name != "Test" {
		t.Errorf("expected course info, got %v", path.Course)
	}
}

func TestPathForCourse_NotFound(t *testing.T) {
	p := testPlanner(t)
	_, err := p.PathForCourse("nonexistent")
	if err == nil {
		t.Error("expected error")
	}
}

func TestLoad_RealData(t *testing.T) {
	d, err := concepts.LoadDir("../../data/concepts")
	if err != nil {
		t.Fatalf("load concepts: %v", err)
	}
	p, err := Load("../../data/courses.json", d)
	if err != nil {
		t.Fatalf("load real data: %v", err)
	}
	// The count is owned by data/courses.json, not by this test: pinning a literal here made every
	// batch that adds a course fail with a message that contradicted its own condition. Reading the
	// file still catches a Load that drops or duplicates one.
	raw, err := os.ReadFile("../../data/courses.json")
	if err != nil {
		t.Fatalf("read courses: %v", err)
	}
	var entries []map[string]any
	if err := json.Unmarshal(raw, &entries); err != nil {
		t.Fatalf("parse courses: %v", err)
	}
	if got := len(p.Courses()); got != len(entries) {
		t.Errorf("Load returned %d courses for %d in data/courses.json", got, len(entries))
	}
	// Check that 4th grade chain has reasonable size
	path, err := p.PathForCourse("4")
	if err != nil {
		t.Fatalf("4th grade path: %v", err)
	}
	if len(path.Concepts) < 10 || len(path.Concepts) > 60 {
		t.Errorf("expected 10-60 concepts in 4th grade chain, got %d", len(path.Concepts))
	}
	t.Logf("4th grade chain: %d concepts", len(path.Concepts))
}

// Every catalog course must resolve against the real DAG (stale targets
// would break PathForCourse).
func TestAllCoursesResolve(t *testing.T) {
	d, err := concepts.LoadDir("../../data/concepts")
	if err != nil {
		t.Fatalf("load concepts: %v", err)
	}
	p, err := Load("../../data/courses.json", d)
	if err != nil {
		t.Fatalf("load courses: %v", err)
	}
	for _, c := range p.Courses() {
		if _, err := p.PathForCourse(c.ID); err != nil {
			t.Errorf("course %s failed to resolve: %v", c.ID, err)
		}
	}
}

func TestProgressForCourse(t *testing.T) {
	d, err := concepts.LoadDir("../../data/concepts")
	if err != nil {
		t.Fatalf("load concepts: %v", err)
	}
	p, err := Load("../../data/courses.json", d)
	if err != nil {
		t.Fatalf("load courses: %v", err)
	}
	course := p.Course("topo")
	if course == nil {
		t.Fatal("expected topo course")
	}
	// Nothing mastered.
	cp, err := p.ProgressForCourse(course, map[string]*storage.ConceptProgress{}, 30)
	if err != nil {
		t.Fatalf("progress: %v", err)
	}
	if cp.Total == 0 || cp.Mastered != 0 || cp.Pct != 0 {
		t.Errorf("unexpected empty progress: %+v", cp)
	}
	if cp.DaysRemaining == 0 {
		t.Error("expected a positive days_remaining estimate")
	}
	// All mastered → 100%.
	all := make(map[string]*storage.ConceptProgress)
	for _, c := range d.Order() {
		all[c.ID] = &storage.ConceptProgress{StudentID: "x", ConceptID: c.ID, Status: "MASTERED"}
	}
	cp2, err := p.ProgressForCourse(course, all, 30)
	if err != nil {
		t.Fatalf("progress full: %v", err)
	}
	if cp2.Pct != 1.0 || cp2.DaysRemaining != 0 {
		t.Errorf("expected full progress, got %+v", cp2)
	}
}

func TestPlannedCourse_KeepsPlanOrderWithPrerequisitesFirst(t *testing.T) {
	p := testPlanner(t)
	dir := t.TempDir()
	extra := filepath.Join(dir, "courses.json")
	// Plan c before b: graph order alone would give a, b, c, d.
	os.WriteFile(extra, []byte(`[{"id":"planned","name":"P","grade":"","description":"","plan":["c","b","d"]}]`), 0644)
	base := filepath.Join(dir, "base.json")
	os.WriteFile(base, []byte(`[]`), 0644)
	q, err := Load(base, p.dag, extra)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	path, err := q.PathForCourse("planned")
	if err != nil {
		t.Fatalf("path: %v", err)
	}
	var ids []string
	for _, c := range path.Concepts {
		ids = append(ids, c.ID)
	}
	want := []string{"a", "c", "b", "d"}
	if len(ids) != len(want) {
		t.Fatalf("path %v, want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("path %v, want %v", ids, want)
		}
	}
	if got := q.Course("planned").Targets; len(got) != 3 {
		t.Errorf("a course with only a plan should target it, got %v", got)
	}
	if r := q.PlanOrder("planned"); r["c"] >= r["b"] {
		t.Errorf("plan order %v should rank c before b", r)
	}
	if q.PlanOrder("missing") != nil {
		t.Error("unknown course should have no plan order")
	}
}

func TestPlansOrder_InterleavesFollowedPlans(t *testing.T) {
	p := testPlanner(t)
	dir := t.TempDir()
	extra := filepath.Join(dir, "courses.json")
	os.WriteFile(extra, []byte(`[{"id":"p1","name":"P1","grade":"","description":"","plan":["b"]},
{"id":"p2","name":"P2","grade":"","description":"","plan":["c"]}]`), 0644)
	base := filepath.Join(dir, "base.json")
	os.WriteFile(base, []byte(`[]`), 0644)
	q, err := Load(base, p.dag, extra)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// p1's path is a, b; p2's is a, c. Interleaved: a (both), then b and c.
	r := q.PlansOrder([]string{"p1", "p2"})
	if len(r) != 3 || r["a"] != 0 || r["b"] != 1 || r["c"] != 2 {
		t.Errorf("interleaved order %v, want a=0 b=1 c=2", r)
	}
	if one := q.PlansOrder([]string{"p2"}); one["c"] != 1 {
		t.Errorf("a single plan keeps its own order, got %v", one)
	}
	if q.PlansOrder(nil) != nil || q.PlansOrder([]string{"missing"}) != nil {
		t.Error("no followed plans should give no order")
	}
}
