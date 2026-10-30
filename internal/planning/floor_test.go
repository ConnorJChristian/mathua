package planning

import (
	"encoding/json"
	"testing"
)

// A course above school grades is past the grade-school domains by default;
// a school-grade course is not; floor_domains overrides either, and an empty
// list switches the floor off.
func TestCourseFloor(t *testing.T) {
	cases := []struct {
		json string
		want int
	}{
		{`{"id":"a","grade":"MA"}`, 4},
		{`{"id":"a","grade":"college"}`, 4},
		{`{"id":"a","grade":"12th-college"}`, 4},
		{`{"id":"a","grade":"Pro"}`, 4},
		{`{"id":"a","grade":"4th"}`, 0},
		{`{"id":"a","grade":"8th-9th"}`, 0},
		{`{"id":"a","grade":"K"}`, 0},
		{`{"id":"a","grade":"MA","floor_domains":[]}`, 0},
		{`{"id":"a","grade":"6th","floor_domains":["arithmetic"]}`, 1},
	}
	for _, c := range cases {
		var co Course
		if err := json.Unmarshal([]byte(c.json), &co); err != nil {
			t.Fatal(err)
		}
		if got := len(co.Floor()); got != c.want {
			t.Errorf("%s: %d floor domains, want %d", c.json, got, c.want)
		}
	}
}
