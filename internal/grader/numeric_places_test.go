package grader

import "testing"

// Q50: places decide, not a window around the key.
func TestNumericDecimalPlaces(t *testing.T) {
	r := NewRouter()
	cases := []struct {
		key, typed string
		want       bool
	}{
		{"12.61", "12.61", true},
		{"12.61", "12.610001", true},
		{"12.61", "12.62", false},
		{"0.2", "0.25", false},
		{"0.2", "0.20", true},
		{"0.01", "0.015", false},
		{"3.5", "3.55", false},
		{"2.666667", "2.67", true},
		{"2.666667", "2.667", true},
		{"2.666667", "8/3", true},
		{"2.666667", "2.6", false},
		{"2.666667", "2.7", false},
		{"5", "5.0", true},
		{"5", "5.1", false},
		{"1/3", "0.333", true},
	}
	for _, c := range cases {
		if got := r.Grade(GradingNumeric, c.key, c.typed).Correct; got != c.want {
			t.Errorf("key %s, typed %s: correct=%v, want %v", c.key, c.typed, got, c.want)
		}
	}
}
