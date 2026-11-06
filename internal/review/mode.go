package review

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// Mode is the FSRS feature flag, MATHUA_FSRS (plan section 5).
//
//   - off (the default): nothing FSRS reads or writes. The SM-2 schedule and
//     the one due rule of Phase 0 decide everything, as before Phase 1.
//   - shadow (Phase 2): every scheduling event computes the FSRS update and
//     writes review_log and the fsrs_* columns; nothing reads them for a
//     decision.
//   - active (Phase 3): "due" and the queue read FSRS.
//
// Phase 1 parses, logs and exposes the flag; shadow and active change no
// behaviour yet. Rollback is MATHUA_FSRS=off; state already written stays.
type Mode string

const (
	ModeOff    Mode = "off"
	ModeShadow Mode = "shadow"
	ModeActive Mode = "active"
)

// ParseMode reads a flag value: empty is off; anything else unknown is
// off with an error, so a typo never turns scheduling on.
func ParseMode(s string) (Mode, error) {
	switch Mode(strings.ToLower(strings.TrimSpace(s))) {
	case "", ModeOff:
		return ModeOff, nil
	case ModeShadow:
		return ModeShadow, nil
	case ModeActive:
		return ModeActive, nil
	}
	return ModeOff, fmt.Errorf("MATHUA_FSRS=%q: want off, shadow or active; using off", s)
}

var (
	modeOnce sync.Once
	mode     Mode
	modeErr  error
)

// CurrentMode is the process's flag, read once from MATHUA_FSRS. The
// error, when set, is the parse failure the server logs at startup.
func CurrentMode() (Mode, error) {
	modeOnce.Do(func() { mode, modeErr = ParseMode(os.Getenv("MATHUA_FSRS")) })
	return mode, modeErr
}

// ErrActiveNotBuilt is MATHUA_FSRS=active before Phase 3 exists. The
// server refuses to start with it (CheckBuilt) rather than run what would
// silently be shadow under an "active" label (FSRS Phase 2 review, I5).
var ErrActiveNotBuilt = fmt.Errorf("MATHUA_FSRS=active: FSRS active scheduling is Phase 3 and is not built in this version; set MATHUA_FSRS=shadow (writes FSRS, decides nothing) or off")

// CheckBuilt reports an error for a mode this build cannot run: active
// until Phase 3.
func CheckBuilt(m Mode) error {
	if m == ModeActive {
		return ErrActiveNotBuilt
	}
	return nil
}

// Writes reports whether the mode writes FSRS state (shadow or active).
func (m Mode) Writes() bool { return m == ModeShadow || m == ModeActive }

// Decides reports whether the mode reads FSRS state for scheduling.
func (m Mode) Decides() bool { return m == ModeActive }
