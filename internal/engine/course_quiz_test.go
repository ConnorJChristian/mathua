package engine

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/planning"
	"github.com/chuma-beep/mathua/internal/storage"
)

// The mastery test of a learner following courses covers only topics on those
// courses that they have worked on: not untouched course topics, not studied
// topics from elsewhere, not placement-only credit.
func TestCourseQuizConcepts_ScopedToFollowedCoursesAndStudiedTopics(t *testing.T) {
	th := concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 60}
	d, err := concepts.Build([]concepts.Concept{
		{ID: "a", Label: "A", Domain: "d", Subdomain: "s1", GradingType: "numeric", MasteryThreshold: th},
		{ID: "b", Label: "B", Domain: "d", Subdomain: "s1", GradingType: "numeric", Prerequisites: []string{"a"}, MasteryThreshold: th},
		{ID: "c", Label: "C", Domain: "d", Subdomain: "s2", GradingType: "numeric", Prerequisites: []string{"b"}, MasteryThreshold: th},
		{ID: "p", Label: "P", Domain: "d", Subdomain: "s3", GradingType: "numeric", MasteryThreshold: th},
		{ID: "x", Label: "X (another course)", Domain: "other", Subdomain: "s9", GradingType: "numeric", MasteryThreshold: th},
	})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	cp := filepath.Join(dir, "courses.json")
	if err := os.WriteFile(cp, []byte(`[{"id":"plan.one","name":"Course one","plan":["c","p"]},{"id":"plan.other","name":"Other","plan":["x"]}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	pl, err := planning.Load(cp, d)
	if err != nil {
		t.Fatal(err)
	}
	store, err := storage.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	e := New(store, d, generator.NewRegistry(), nil, pl)
	st, err := e.CreateStudent("c")
	if err != nil {
		t.Fatal(err)
	}

	// Following no course: the caller keeps the track-wide quiz.
	if _, _, ok := e.CourseQuizConcepts(st.ID); ok {
		t.Fatal("no followed course should leave the quiz track-wide")
	}
	if _, err := e.SetActiveCourses(st.ID, []string{"plan.one"}); err != nil {
		t.Fatal(err)
	}
	// Nothing studied yet: an empty, scoped quiz.
	if got, _, ok := e.CourseQuizConcepts(st.ID); !ok || len(got) != 0 {
		t.Fatalf("nothing studied: want empty scoped quiz, got %v ok=%v", got, ok)
	}

	now := time.Now()
	for _, p := range []*storage.ConceptProgress{
		{StudentID: st.ID, ConceptID: "a", Status: "PRACTICING", Attempts: 4, WeaknessScore: 0.2, LastAttempted: &now},
		{StudentID: st.ID, ConceptID: "b", Status: "LEARNING", Attempts: 2, WeaknessScore: 0.7, LastAttempted: &now},
		{StudentID: st.ID, ConceptID: "p", Status: "LEARNING", PlacementSeeded: true},
		{StudentID: st.ID, ConceptID: "x", Status: "LEARNING", Attempts: 5, WeaknessScore: 0.9, LastAttempted: &now},
	} {
		if err := store.UpsertProgress(p); err != nil {
			t.Fatal(err)
		}
	}
	got, courses, ok := e.CourseQuizConcepts(st.ID)
	if !ok {
		t.Fatal("want a scoped quiz")
	}
	ids := map[string]bool{}
	for _, c := range got {
		ids[c.ID] = true
	}
	if len(got) != 2 || !ids["a"] || !ids["b"] {
		t.Fatalf("want exactly the studied course topics a and b, got %v", ids)
	}
	if got[0].ID != "b" {
		t.Errorf("the weakest studied topic should lead, got %s", got[0].ID)
	}
	if len(courses) != 1 || courses[0] != "Course one" {
		t.Errorf("courses = %v", courses)
	}
}
