package review

import "testing"

func TestRules_DefaultsAreThePlansProposal(t *testing.T) {
	r := DefaultRules()
	if !r.MasteryCheck || !r.CourseTests || !r.PracticeLapse || r.LapseBelowR != 0.95 || !r.Judged || !r.FirstMastery || !r.DailyCap || !r.SeedOnTouch {
		t.Fatalf("defaults %+v", r)
	}
	if got, _ := ParseRules(""); got != r {
		t.Errorf("empty: %+v", got)
	}
	want := "course_tests=on,daily_cap=on,first_mastery=on,judged=on,lapse_below_r=0.95,mastery_check=on,practice_lapse=on,seed_on_touch=on"
	if r.String() != want {
		t.Errorf("String %q", r.String())
	}
}

func TestRules_Parse(t *testing.T) {
	r, err := ParseRules(" course_tests=off, lapse_below_r=0.9 ,judged=no")
	if err != nil || r.CourseTests || r.Judged || r.LapseBelowR != 0.9 || !r.PracticeLapse {
		t.Fatalf("%+v %v", r, err)
	}
	if back, err := ParseRules(r.String()); err != nil || back != r {
		t.Errorf("round trip: %+v %v", back, err)
	}
	for _, bad := range []string{"course_test=off", "judged=maybe", "lapse_below_r=1.5", "daily_cap"} {
		if got, err := ParseRules(bad); err == nil || got != DefaultRules() {
			t.Errorf("%q: %+v %v, want an error and the defaults", bad, got, err)
		}
	}
}
