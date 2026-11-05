package review

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Rules says which events move FSRS in shadow and active mode (FSRS plan
// 3.3). C has not decided the open rows yet, so each is a switch whose
// default is the plan's proposal; MATHUA_FSRS_RULES overrides them
// ("course_tests=off,lapse_below_r=0.9"), the server logs the rules in
// force at startup, and every review_log row is written under them. The
// rows that are not switches are fixed by the plan:
//
//   - a review answer on an objective item: right is Good, wrong or "I don't
//     know" is Again;
//   - a self-graded recall card: never an event;
//   - placement and the onboarding diagnostic: never an event (seeds only,
//     plan 3.4: a placed row has no state);
//   - implicit and encompasses credit: never an event in Phase 2.
type Rules struct {
	// MasteryCheck: the weekly mastery check's items are reviews.
	MasteryCheck bool
	// CourseTests: a course review test's items are reviews ([C] 2;
	// proposed yes).
	CourseTests bool
	// PracticeLapse: a wrong practice answer on a MASTERED topic is a lapse
	// (Again) when R < LapseBelowR; at or above it the answer is logged and
	// held (no state change) ([C] 4; proposed yes, 0.95).
	PracticeLapse bool
	LapseBelowR   float64
	// Judged: an LLM-graded proof or written recall card is Good when
	// graded fully correct and Again otherwise, only when it was graded (a
	// failed call or no key is no event) ([C] 3; proposed yes).
	Judged bool
	// FirstMastery: the answer that first takes a topic to MASTERED creates
	// its state with the reference first-review state for Good (S = w[2]).
	FirstMastery bool
	// DailyCap: at most one stability-raising event per topic per
	// learner-local day; a later one the same day is logged and held.
	DailyCap bool
	// SeedOnTouch: a demonstrated-mastered row with no state (mastered
	// before shadow mode existed) gets its state by replaying its own
	// attempts the first time an event touches it (plan 3.4; the same
	// replay as scripts/fsrs_seed).
	SeedOnTouch bool
}

// DefaultRules is the plan's proposal for every switch.
func DefaultRules() Rules {
	return Rules{
		MasteryCheck:  true,
		CourseTests:   true,
		PracticeLapse: true,
		LapseBelowR:   0.95,
		Judged:        true,
		FirstMastery:  true,
		DailyCap:      true,
		SeedOnTouch:   true,
	}
}

func (r *Rules) bools() map[string]*bool {
	return map[string]*bool{
		"mastery_check":  &r.MasteryCheck,
		"course_tests":   &r.CourseTests,
		"practice_lapse": &r.PracticeLapse,
		"judged":         &r.Judged,
		"first_mastery":  &r.FirstMastery,
		"daily_cap":      &r.DailyCap,
		"seed_on_touch":  &r.SeedOnTouch,
	}
}

// String is the canonical form, every switch named, sorted:
// "course_tests=on,daily_cap=on,...,lapse_below_r=0.95,...".
func (r Rules) String() string {
	var parts []string
	for k, v := range r.bools() {
		parts = append(parts, k+"="+map[bool]string{true: "on", false: "off"}[*v])
	}
	parts = append(parts, "lapse_below_r="+strconv.FormatFloat(r.LapseBelowR, 'g', -1, 64))
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

// ParseRules applies "name=on|off,lapse_below_r=0.9" to the defaults. An
// unknown name or a bad value is an error, and the defaults are returned, so
// a typo never changes what counts as a review.
func ParseRules(s string) (Rules, error) {
	r := DefaultRules()
	s = strings.TrimSpace(s)
	if s == "" {
		return r, nil
	}
	out := DefaultRules()
	for _, kv := range strings.Split(s, ",") {
		kv = strings.TrimSpace(kv)
		if kv == "" {
			continue
		}
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			return r, fmt.Errorf("MATHUA_FSRS_RULES %q: want name=value", kv)
		}
		k, v = strings.ToLower(strings.TrimSpace(k)), strings.ToLower(strings.TrimSpace(v))
		if k == "lapse_below_r" {
			f, err := strconv.ParseFloat(v, 64)
			if err != nil || f <= 0 || f > 1 {
				return r, fmt.Errorf("MATHUA_FSRS_RULES lapse_below_r=%q: want a number in (0, 1]", v)
			}
			out.LapseBelowR = f
			continue
		}
		p, known := out.bools()[k]
		if !known {
			return r, fmt.Errorf("MATHUA_FSRS_RULES: unknown rule %q", k)
		}
		switch v {
		case "on", "true", "1", "yes":
			*p = true
		case "off", "false", "0", "no":
			*p = false
		default:
			return r, fmt.Errorf("MATHUA_FSRS_RULES %s=%q: want on or off", k, v)
		}
	}
	return out, nil
}

var (
	rulesOnce sync.Once
	rules     Rules
	rulesErr  error
)

// CurrentRules is the process's rules, read once from MATHUA_FSRS_RULES.
func CurrentRules() (Rules, error) {
	rulesOnce.Do(func() { rules, rulesErr = ParseRules(os.Getenv("MATHUA_FSRS_RULES")) })
	return rules, rulesErr
}
