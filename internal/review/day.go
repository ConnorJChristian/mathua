package review

import (
	"os"
	"sync"
	"time"
)

// Learner-local days (FSRS plan 4.8). Timestamps are stored in UTC; the
// day a review belongs to, the daily review cap and the "one
// stability-raising event per topic per day" rule (Phase 2 and 3) are
// counted in the learner's local day. The zone is MATHUA_TZ, default
// America/Los_Angeles; an unknown name falls back to UTC and is reported
// by the server at startup.
//
// Note the two notions of "same day": the FSRS short-term formula applies
// when less than a whole day (24 hours) has passed since the last review,
// as the reference computes it (ElapsedWholeDays); SameLocalDay is the
// calendar rule for caps and reports. A review at 23:30 and one at 00:30
// are the same day to the formula and different days to the cap.

// DefaultZone is the learner's zone when MATHUA_TZ is unset.
const DefaultZone = "America/Los_Angeles"

var (
	zoneOnce sync.Once
	zone     *time.Location
	zoneErr  error
)

// LearnerZone is the configured zone. The error names a MATHUA_TZ that
// did not load; the zone returned is then UTC.
func LearnerZone() (*time.Location, error) {
	zoneOnce.Do(func() {
		name := os.Getenv("MATHUA_TZ")
		if name == "" {
			name = DefaultZone
		}
		zone, zoneErr = time.LoadLocation(name)
		if zoneErr != nil {
			zone = time.UTC
		}
	})
	return zone, zoneErr
}

// Day is a calendar day in the learner's zone, as "2026-11-01".
type Day string

// LocalDay is the learner-local calendar day t falls in.
func LocalDay(t time.Time, loc *time.Location) Day {
	return Day(t.In(loc).Format("2006-01-02"))
}

// SameLocalDay reports whether a and b fall in the same learner-local day.
func SameLocalDay(a, b time.Time, loc *time.Location) bool {
	return LocalDay(a, loc) == LocalDay(b, loc)
}

// StartOfLocalDay is the UTC instant the learner's day containing t
// begins (local midnight; on a DST change day that is still 00:00 local).
func StartOfLocalDay(t time.Time, loc *time.Location) time.Time {
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc).UTC()
}

// NextLocalDay is the UTC instant the learner's next day begins: the
// start of the day after t's. Across the fall-back change it is 25 hours
// after the previous start, across spring-forward 23.
func NextLocalDay(t time.Time, loc *time.Location) time.Time {
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day()+1, 0, 0, 0, 0, loc).UTC()
}

// LocalDaysBetween is how many learner-local calendar days from a's day
// to b's day (0 on the same day, negative when b is earlier). It counts
// calendar days, not 24-hour periods, so the DST change does not shift it.
func LocalDaysBetween(a, b time.Time, loc *time.Location) int {
	sa, sb := StartOfLocalDay(a, loc), StartOfLocalDay(b, loc)
	// Day starts are 23, 24 or 25 hours apart; rounding to the nearest day
	// recovers the calendar count.
	return int(float64(sb.Sub(sa))/float64(24*time.Hour) + 0.5*sign(sb.Sub(sa)))
}

func sign(d time.Duration) float64 {
	if d < 0 {
		return -1
	}
	return 1
}
