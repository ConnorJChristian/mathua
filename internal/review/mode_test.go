package review

import (
	"strings"
	"testing"
)

func TestParseMode_DefaultsOffAndRejectsTypos(t *testing.T) {
	for in, want := range map[string]Mode{"": ModeOff, "off": ModeOff, "OFF": ModeOff, " shadow ": ModeShadow, "Active": ModeActive} {
		got, err := ParseMode(in)
		if err != nil || got != want {
			t.Errorf("ParseMode(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, in := range []string{"on", "shadw", "1", "true", "active "} {
		got, err := ParseMode(in)
		if in == "active " {
			if err != nil || got != ModeActive {
				t.Errorf("surrounding space should be trimmed: %v %v", got, err)
			}
			continue
		}
		if err == nil || got != ModeOff {
			t.Errorf("ParseMode(%q) = %v, %v; want off with an error", in, got, err)
		}
	}
	if ModeOff.Writes() || ModeOff.Decides() || !ModeShadow.Writes() || ModeShadow.Decides() || !ModeActive.Writes() || !ModeActive.Decides() {
		t.Error("mode predicates")
	}
}

// FSRS Phase 2 review, I5: active parses (it is the Phase 3 value) but this
// build refuses to run it: the server exits at startup with a message that
// names what to set instead.
func TestCheckBuilt_ActiveIsRefusedUntilPhase3(t *testing.T) {
	for _, m := range []Mode{ModeOff, ModeShadow} {
		if err := CheckBuilt(m); err != nil {
			t.Errorf("%s: %v", m, err)
		}
	}
	err := CheckBuilt(ModeActive)
	if err == nil {
		t.Fatal("active accepted")
	}
	for _, want := range []string{"MATHUA_FSRS=active", "Phase 3", "MATHUA_FSRS=shadow"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("message %q lacks %q", err, want)
		}
	}
}
