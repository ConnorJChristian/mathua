package review

import "testing"

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
