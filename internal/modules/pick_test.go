package modules

import "testing"

func TestPickReadingPrefersCourseTexts(t *testing.T) {
	s := &Set{readings: map[string][]Reading{
		"c": {
			{Source: "Hoy", File: "hoy.pdf", Module: "m"},
			{Source: "Schaum", File: "schaum.pdf", Module: "m"},
		},
	}}
	if r, _ := s.PickReading("c", nil); r.Source != "Hoy" {
		t.Errorf("no course: first reading, got %s", r.Source)
	}
	if r, _ := s.PickReading("c", []string{"m/schaum.pdf", "m/hoy.pdf"}); r.Source != "Schaum" {
		t.Errorf("course texts in priority order, got %s", r.Source)
	}
	if r, _ := s.PickReading("c", []string{"x/none.pdf"}); r.Source != "Hoy" {
		t.Errorf("unmatched texts fall back to the first reading, got %s", r.Source)
	}
	if _, ok := s.PickReading("missing", nil); ok {
		t.Error("a concept without readings has none")
	}
}
