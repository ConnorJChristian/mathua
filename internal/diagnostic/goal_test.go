package diagnostic

import (
	"fmt"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/generator"
)

// goalDAG has the shape of a graduate course's placement path: twenty
// elementary roots, a forty-step foundation chain resting on them, and
// fifteen plan topics on top.
func goalDAG(t *testing.T) (*concepts.DAG, *generator.Registry, []*concepts.Concept, []string) {
	t.Helper()
	var raw []concepts.Concept
	add := func(id string, prereqs []string) {
		raw = append(raw, concepts.Concept{ID: id, Label: id, Domain: "d", Subdomain: "s-" + id,
			Prerequisites: prereqs, MasteryThreshold: concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 10}})
	}
	for i := 0; i < 20; i++ {
		add(fmt.Sprintf("root%02d", i), nil)
	}
	for i := 1; i <= 40; i++ {
		pre := []string{fmt.Sprintf("root%02d", i%20)}
		if i > 1 {
			pre = append(pre, fmt.Sprintf("f%02d", i-1))
		}
		add(fmt.Sprintf("f%02d", i), pre)
	}
	var goal []string
	for i := 1; i <= 15; i++ {
		pre := []string{"f40"}
		if i > 1 {
			pre = append(pre, fmt.Sprintf("t%02d", i-1))
		}
		id := fmt.Sprintf("t%02d", i)
		add(id, pre)
		goal = append(goal, id)
	}
	d, err := concepts.Build(raw)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	reg := generator.NewRegistry()
	for _, c := range raw {
		reg.Register(c.ID, &fakeGen{})
	}
	return d, reg, d.Order(), goal
}

func runPlacement(t *testing.T, knows func(string) bool) (asked []string, rep *DiagnosticReport) {
	d, reg, path, goal := goalDAG(t)
	e := NewEngine(d, reg)
	s := e.StartWithGoal(path, goal)
	for i := 0; i < 60 && !e.IsComplete(s); i++ {
		_, cid, err := e.NextQuestion(s)
		if err != nil {
			t.Fatalf("next: %v", err)
		}
		if cid == "" {
			break
		}
		asked = append(asked, cid)
		e.RecordAnswerTimed(s, cid, knows(cid), 5, 30)
	}
	return asked, e.Report(s)
}

// Fork fix: a course placement probes the course, not its roots, and does not
// re-ask what was answered right.
func TestStartWithGoal_StrongLearnerSkipsFoundations(t *testing.T) {
	asked, rep := runPlacement(t, func(string) bool { return true })
	seen := map[string]int{}
	roots, targets := 0, 0
	for _, id := range asked {
		seen[id]++
		if seen[id] == 2 {
			t.Errorf("%s was asked again after a right answer", id)
		}
		if strings.HasPrefix(id, "root") {
			roots++
		}
		if strings.HasPrefix(id, "t") {
			targets++
		}
	}
	if roots > 0 {
		t.Errorf("a learner who answers everything right was asked %d root questions: %v", roots, asked)
	}
	if targets < len(asked)/2 {
		t.Errorf("want most questions on plan topics, got %d of %d: %v", targets, len(asked), asked)
	}
	if !strings.HasPrefix(rep.FrontierLabel, "t") {
		t.Errorf("frontier should be a plan topic, got %q", rep.FrontierLabel)
	}
	if len(asked) >= maxTotalQuestions {
		t.Errorf("a strong learner should finish under the cap, used %d", len(asked))
	}
}

// The other side: a learner who knows only the roots and the first five
// foundation steps is placed there, not above, and nothing they don't know
// is credited.
func TestStartWithGoal_WeakLearnerFoundAtTheBottom(t *testing.T) {
	knows := func(id string) bool {
		if strings.HasPrefix(id, "root") {
			return true
		}
		var n int
		if _, err := fmt.Sscanf(id, "f%02d", &n); err == nil {
			return n <= 5
		}
		return false
	}
	asked, rep := runPlacement(t, knows)
	for id, b := range rep.MasteryLevels {
		if b >= beliefThreshold && !knows(id) {
			t.Errorf("%s placed as known (%.2f) but the learner does not know it", id, b)
		}
	}
	if strings.HasPrefix(rep.FrontierLabel, "t") || rep.FrontierLabel == "" {
		t.Errorf("frontier should be among the foundations, got %q after %v", rep.FrontierLabel, asked)
	}
	if len(asked) >= maxTotalQuestions {
		t.Errorf("a weak learner should be found under the cap, used %d: %v", len(asked), asked)
	}
}
