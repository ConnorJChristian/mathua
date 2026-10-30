package planning

import "testing"

// A course's book_step_cap applies to the topics it lists; where two
// courses list a topic the smaller set cap wins, and a course that sets no
// cap leaves its topics to MATHUA_BOOK_STEP_CAP (absent from the map).
func TestBookStepCaps(t *testing.T) {
	two, five, off := 2, 5, 0
	p := &Planner{courses: []*Course{
		{ID: "a", Plan: []string{"t1", "t2"}, BookStepCap: &five},
		{ID: "b", Modules: []Module{{Topics: []string{"t2", "t3"}}}, BookStepCap: &two},
		{ID: "c", Targets: []string{"t4"}, BookStepCap: &off},
		{ID: "d", Plan: []string{"t5"}},
	}}
	got := p.BookStepCaps()
	want := map[string]int{"t1": 5, "t2": 2, "t3": 2, "t4": 0}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: cap %d, want %d", k, got[k], v)
		}
	}
	if _, ok := got["t5"]; ok {
		t.Error("a course without book_step_cap sets nothing")
	}
}
