package review

import (
	"sync"
	"testing"
	"time"
)

// Learner-local days across midnight and the 2026-11-01 fall-back in
// America/Los_Angeles (plan 4.8). Stored instants are UTC throughout.
func TestLocalDay_MidnightAndDST(t *testing.T) {
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Skip("no tz database")
	}
	// 2026-10-31 23:30 PDT = 2026-11-01 06:30 UTC; 00:30 PDT the next day = 07:30 UTC.
	lateSat := time.Date(2026, 11, 1, 6, 30, 0, 0, time.UTC)
	earlySun := time.Date(2026, 11, 1, 7, 30, 0, 0, time.UTC)
	if LocalDay(lateSat, la) != "2026-10-31" || LocalDay(earlySun, la) != "2026-11-01" {
		t.Fatalf("local days %s, %s", LocalDay(lateSat, la), LocalDay(earlySun, la))
	}
	if SameLocalDay(lateSat, earlySun, la) {
		t.Fatal("23:30 and 00:30 local read as the same day")
	}
	if !SameLocalDay(lateSat, lateSat.Add(-20*time.Hour), la) {
		t.Fatal("03:30 and 23:30 local Saturday are the same day")
	}
	// The formula's notion differs: an hour apart is "same day" to it.
	if ElapsedWholeDays(lateSat, earlySun) != 0 {
		t.Fatal("an hour apart is not under a whole day")
	}
	// UTC midnight is mid-afternoon in LA: not a day boundary there.
	utcMidnight := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	if !SameLocalDay(utcMidnight.Add(-time.Minute), utcMidnight.Add(time.Minute), la) {
		t.Fatal("UTC midnight split a local day")
	}

	// Fall back: Sunday 2026-11-01 has 25 hours. 01:30 PDT (08:30 UTC) and
	// 01:30 PST (09:30 UTC) are the same local day; the day starts at
	// 07:00 UTC and the next at 08:00 UTC (PST).
	firstOneThirty := time.Date(2026, 11, 1, 8, 30, 0, 0, time.UTC)
	secondOneThirty := time.Date(2026, 11, 1, 9, 30, 0, 0, time.UTC)
	if !SameLocalDay(firstOneThirty, secondOneThirty, la) {
		t.Fatal("the repeated 01:30 is not the same day")
	}
	start := StartOfLocalDay(firstOneThirty, la)
	next := NextLocalDay(firstOneThirty, la)
	if !start.Equal(time.Date(2026, 11, 1, 7, 0, 0, 0, time.UTC)) || !next.Equal(time.Date(2026, 11, 2, 8, 0, 0, 0, time.UTC)) {
		t.Fatalf("Nov 1 starts %v, next %v", start, next)
	}
	if next.Sub(start) != 25*time.Hour {
		t.Fatalf("the fall-back day is %v long", next.Sub(start))
	}
	// Calendar days between: the 25-hour day counts as one.
	if d := LocalDaysBetween(lateSat, time.Date(2026, 11, 3, 6, 0, 0, 0, time.UTC), la); d != 2 {
		t.Fatalf("Oct 31 -> Nov 2 (22:00 PST) = %d days, want 2 (49 hours, two calendar days)", d)
	}
	if d := LocalDaysBetween(earlySun, lateSat, la); d != -1 {
		t.Fatalf("backwards = %d, want -1", d)
	}
	if d := LocalDaysBetween(firstOneThirty, secondOneThirty, la); d != 0 {
		t.Fatalf("within the repeated hour = %d", d)
	}
	// Spring forward 2026-03-08: a 23-hour day.
	mar := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	if NextLocalDay(mar, la).Sub(StartOfLocalDay(mar, la)) != 23*time.Hour {
		t.Fatal("spring-forward day is not 23 hours")
	}
	if d := LocalDaysBetween(mar.Add(-24*time.Hour), mar.Add(24*time.Hour), la); d != 2 {
		t.Fatalf("across spring forward = %d, want 2", d)
	}
	// Results are UTC instants whatever the zone, and the day string is the
	// same whether the input is given in UTC or local time.
	if start.Location() != time.UTC || LocalDay(lateSat.In(la), la) != LocalDay(lateSat, la) {
		t.Fatal("zone leaked")
	}
}

func TestLearnerZone_DefaultAndOverride(t *testing.T) {
	zoneOnce = sync.Once{}
	t.Setenv("MATHUA_TZ", "")
	loc, err := LearnerZone()
	if err != nil || loc.String() != DefaultZone {
		t.Fatalf("default zone %v %v", loc, err)
	}
	zoneOnce = sync.Once{}
	t.Setenv("MATHUA_TZ", "No/Such_Zone")
	loc, err = LearnerZone()
	if err == nil || loc != time.UTC {
		t.Fatalf("bad zone: %v %v", loc, err)
	}
	zoneOnce = sync.Once{}
	t.Setenv("MATHUA_TZ", "Europe/London")
	if loc, err := LearnerZone(); err != nil || loc.String() != "Europe/London" {
		t.Fatalf("override: %v %v", loc, err)
	}
	zoneOnce = sync.Once{}
}
