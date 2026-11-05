package review

// Phase 1 independent reviewer: learner-local day helpers across midnight
// and both 2026 DST changes in America/Los_Angeles (spring forward Mar 8,
// fall back Nov 1), plus a sweep of the whole year. Every returned instant
// must be UTC (storage stays UTC).

import (
	"sync"
	"testing"
	"time"
)

func la(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Skip("no tzdata:", err)
	}
	return loc
}

func TestReviewer_DayHelpersSweep2026(t *testing.T) {
	loc := la(t)
	lengths := map[time.Duration][]Day{}
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2027, 1, 2, 0, 0, 0, 0, time.UTC)
	for at := start; at.Before(end); at = at.Add(7*time.Minute + 13*time.Second) {
		s, n := StartOfLocalDay(at, loc), NextLocalDay(at, loc)
		if s.Location() != time.UTC || n.Location() != time.UTC {
			t.Fatalf("%v: non-UTC result %v / %v", at, s.Location(), n.Location())
		}
		if at.Before(s) || !at.Before(n) {
			t.Fatalf("%v not in [%v, %v)", at, s, n)
		}
		if LocalDay(s, loc) != LocalDay(at, loc) || LocalDay(n, loc) == LocalDay(at, loc) {
			t.Fatalf("%v: day bounds disagree with LocalDay", at)
		}
		if l := s.In(loc); l.Hour() != 0 || l.Minute() != 0 {
			t.Fatalf("%v: start %v is not local midnight", at, l)
		}
		if got := LocalDaysBetween(at, n, loc); got != 1 {
			t.Fatalf("%v: LocalDaysBetween(t, next) = %d", at, got)
		}
		if got := LocalDaysBetween(n, at, loc); got != -1 {
			t.Fatalf("%v: LocalDaysBetween(next, t) = %d", at, got)
		}
		if !SameLocalDay(at, s, loc) || SameLocalDay(at, n, loc) {
			t.Fatalf("%v: SameLocalDay disagrees", at)
		}
		d := n.Sub(s)
		if len(lengths[d]) == 0 || lengths[d][len(lengths[d])-1] != LocalDay(at, loc) {
			lengths[d] = append(lengths[d], LocalDay(at, loc))
		}
	}
	if got := lengths[23*time.Hour]; len(got) != 1 || got[0] != "2026-03-08" {
		t.Errorf("23-hour days: %v, want [2026-03-08]", got)
	}
	if got := lengths[25*time.Hour]; len(got) != 1 || got[0] != "2026-11-01" {
		t.Errorf("25-hour days: %v, want [2026-11-01]", got)
	}
	if len(lengths) != 3 {
		t.Errorf("day lengths seen: %d kinds", len(lengths))
	}
	// Whole year: Jan 1 -> Jan 1 is 365 calendar days despite both changes.
	if got := LocalDaysBetween(time.Date(2026, 1, 1, 20, 0, 0, 0, time.UTC), time.Date(2027, 1, 1, 20, 0, 0, 0, time.UTC), loc); got != 365 {
		t.Errorf("2026 has %d local days", got)
	}
}

func TestReviewer_DayHelpersEdges(t *testing.T) {
	loc := la(t)
	cases := []struct {
		name string
		at   time.Time
		day  Day
	}{
		// Midnight edges (PST, UTC-8): 07:59:59.999Z is still the previous day.
		{"jan local 23:59:59.999", time.Date(2026, 1, 15, 7, 59, 59, 999e6, time.UTC), "2026-01-14"},
		{"jan local 00:00", time.Date(2026, 1, 15, 8, 0, 0, 0, time.UTC), "2026-01-15"},
		// UTC midnight is mid-afternoon/evening the day before in LA.
		{"utc midnight", time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), "2026-05-31"},
		// Spring forward Mar 8: 01:59:59 PST = 09:59:59Z, then 03:00 PDT = 10:00Z.
		{"before spring forward", time.Date(2026, 3, 8, 9, 59, 59, 0, time.UTC), "2026-03-08"},
		{"after spring forward", time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC), "2026-03-08"},
		{"mar 8 local 23:59 PDT", time.Date(2026, 3, 9, 6, 59, 0, 0, time.UTC), "2026-03-08"},
		{"mar 9 local 00:00 PDT", time.Date(2026, 3, 9, 7, 0, 0, 0, time.UTC), "2026-03-09"},
		// Fall back Nov 1: 01:30 PDT = 08:30Z and 01:30 PST = 09:30Z, both Nov 1.
		{"first 01:30", time.Date(2026, 11, 1, 8, 30, 0, 0, time.UTC), "2026-11-01"},
		{"second 01:30", time.Date(2026, 11, 1, 9, 30, 0, 0, time.UTC), "2026-11-01"},
		{"oct 31 local 23:59 PDT", time.Date(2026, 11, 1, 6, 59, 0, 0, time.UTC), "2026-10-31"},
		{"nov 1 local 23:59 PST", time.Date(2026, 11, 2, 7, 59, 0, 0, time.UTC), "2026-11-01"},
		{"nov 2 local 00:00 PST", time.Date(2026, 11, 2, 8, 0, 0, 0, time.UTC), "2026-11-02"},
	}
	for _, c := range cases {
		if got := LocalDay(c.at, loc); got != c.day {
			t.Errorf("%s: LocalDay %s, want %s", c.name, got, c.day)
		}
	}
	nov1 := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	if s := StartOfLocalDay(nov1, loc); !s.Equal(time.Date(2026, 11, 1, 7, 0, 0, 0, time.UTC)) {
		t.Errorf("Nov 1 starts %v, want 07:00Z (PDT midnight)", s)
	}
	if n := NextLocalDay(nov1, loc); !n.Equal(time.Date(2026, 11, 2, 8, 0, 0, 0, time.UTC)) {
		t.Errorf("Nov 2 starts %v, want 08:00Z (PST midnight)", n)
	}
	mar8 := time.Date(2026, 3, 8, 18, 0, 0, 0, time.UTC)
	if s := StartOfLocalDay(mar8, loc); !s.Equal(time.Date(2026, 3, 8, 8, 0, 0, 0, time.UTC)) {
		t.Errorf("Mar 8 starts %v, want 08:00Z (PST midnight)", s)
	}
	if n := NextLocalDay(mar8, loc); !n.Equal(time.Date(2026, 3, 9, 7, 0, 0, 0, time.UTC)) {
		t.Errorf("Mar 9 starts %v, want 07:00Z (PDT midnight)", n)
	}
	// The two notions of "same day": 23:30 and 00:30 local are one hour
	// apart, the formula's same day (0 whole days) but different local days.
	a := time.Date(2026, 1, 15, 7, 30, 0, 0, time.UTC)
	b := a.Add(time.Hour)
	if ElapsedWholeDays(a, b) != 0 || SameLocalDay(a, b, loc) || LocalDaysBetween(a, b, loc) != 1 {
		t.Errorf("23:30 -> 00:30: elapsed %v, same local %v", ElapsedWholeDays(a, b), SameLocalDay(a, b, loc))
	}
	// Across fall-back, 24 h after 00:30 PDT Nov 1 is 23:30 PST Nov 1: one
	// whole day to the formula, the same local day to the cap.
	c := time.Date(2026, 11, 1, 7, 30, 0, 0, time.UTC)
	d := c.Add(24 * time.Hour)
	if ElapsedWholeDays(c, d) != 1 || !SameLocalDay(c, d, loc) {
		t.Errorf("fall-back 24h: elapsed %v, same local day %v", ElapsedWholeDays(c, d), SameLocalDay(c, d, loc))
	}
}

func TestReviewer_LearnerZoneDefaultAndBadName(t *testing.T) {
	reset := func() { zoneOnce = sync.Once{}; zone, zoneErr = nil, nil }
	t.Cleanup(reset)
	reset()
	t.Setenv("MATHUA_TZ", "")
	z, err := LearnerZone()
	if err != nil || z.String() != DefaultZone {
		t.Errorf("default zone %v, %v", z, err)
	}
	reset()
	t.Setenv("MATHUA_TZ", "Mars/Olympus_Mons")
	z, err = LearnerZone()
	if err == nil || z != time.UTC {
		t.Errorf("bad zone: %v, %v (want UTC with an error)", z, err)
	}
}
