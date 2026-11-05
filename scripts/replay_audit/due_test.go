package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/mastery"
	"github.com/chuma-beep/mathua/internal/modules"
	"github.com/chuma-beep/mathua/internal/storage"
)

// Every surface agrees on what is due, every day for 30 days, on three
// learners (FSRS plan Phase 0, gate of "0. Fix the due-rule
// disagreement"): the due count (the Study banner, the sidebar badge), the
// course page's due marks, the Up next review card and its reason, the
// review session and the mixed session.
//
// Before the one rule (internal/review), scheduler.effectiveState served
// any mastered topic 14 days after its last review while the count took
// only rows with a past next_review_due: C's six credited course topics
// (mastered through the topics above them, so undated) were served by
// /review from 2026-11-06 while the banner said 0. The heavy learner
// carries the same class of row, so the test fails on that code without
// the snapshot; the snapshot leg (MATHUA_P0_SNAPSHOT and
// MATHUA_P0_STUDENT, with MATHUA_ROOT naming an install whose
// modules-local holds the learner's courses) reproduces it on C's data.
func TestDueSurfacesAgree(t *testing.T) {
	if testing.Short() {
		t.Skip("replays 30 days of sessions")
	}
	root := os.Getenv("MATHUA_ROOT")
	if root == "" {
		abs, err := filepath.Abs("../..")
		if err != nil {
			t.Fatal(err)
		}
		root = abs
		t.Setenv("MATHUA_ROOT", root)
	}
	days := 30

	t.Run("snapshot", func(t *testing.T) {
		src, sid := os.Getenv("MATHUA_P0_SNAPSHOT"), os.Getenv("MATHUA_P0_STUDENT")
		if src == "" || sid == "" {
			t.Skip("set MATHUA_P0_SNAPSHOT and MATHUA_P0_STUDENT to replay a learner's snapshot")
		}
		mods, err := modules.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		work, err := freshCopy(src)
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(filepath.Dir(work))
		r, err := openWith(work, sid, 1, mods)
		if err != nil {
			t.Fatal(err)
		}
		replayDays(t, r, days, true)
	})

	t.Run("fresh placed learner", func(t *testing.T) {
		mods, err := modules.LoadPublic(root)
		if err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		db := filepath.Join(dir, "fresh.db")
		store, err := storage.NewSQLiteStore(db)
		if err != nil {
			t.Fatal(err)
		}
		st, err := store.CreateStudent("fresh")
		if err != nil {
			t.Fatal(err)
		}
		store.Close()
		r, err := openWith(db, st.ID, 7, mods)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := r.eng.SetActiveCourses(st.ID, []string{"econ5153"}); err != nil {
			t.Fatal(err)
		}
		r2, err := openWith(db, st.ID, 7, mods) // reopen with the course followed (own topics, floor)
		if err != nil {
			t.Fatal(err)
		}
		r2.replayPlacement()
		replayDays(t, r2, days, false)
	})

	t.Run("heavy learner", func(t *testing.T) {
		mods, err := modules.LoadPublic(root)
		if err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		db := filepath.Join(dir, "heavy.db")
		store, err := storage.NewSQLiteStore(db)
		if err != nil {
			t.Fatal(err)
		}
		st, err := store.CreateStudent("heavy")
		if err != nil {
			t.Fatal(err)
		}
		if err := store.SetCourseID(st.ID, "econ5153"); err != nil {
			t.Fatal(err)
		}
		if err := store.SetDiagnosticCompleted(st.ID); err != nil {
			t.Fatal(err)
		}
		store.Close()
		r, err := openWith(db, st.ID, 3, mods)
		if err != nil {
			t.Fatal(err)
		}
		seedHeavyLearner(t, r)
		replayDays(t, r, days, true)
	})
}

// replayDays reads every surface, then replays a session and ages the copy
// a day, days times. wantCredited asserts that an undated credited course
// topic comes due and is counted at some point: the row class the old
// count missed.
func replayDays(t *testing.T, r *replay, days int, wantCredited bool) {
	t.Helper()
	creditedDue := false
	for day := 1; day <= days; day++ {
		rep := r.dueSurfaces()
		if len(rep.Disagreements) > 0 {
			t.Errorf("day %d: %s\n  %s", day, rep, strings.Join(rep.Disagreements, "\n  "))
			if day > 3 && t.Failed() {
				t.FailNow()
			}
		}
		progress, _ := r.repo.GetAllProgress(r.sid)
		for _, id := range rep.Due {
			if p := progress[id]; p != nil && p.NextReviewDue == nil && p.Attempts == 0 && p.PlacementSeeded && r.own[id] {
				creditedDue = true
			}
		}
		t.Logf("day %d: %s", day, rep)
		r.session(day)
		if err := r.ageOneDay(); err != nil {
			t.Fatal(err)
		}
	}
	if wantCredited && !creditedDue {
		t.Error("no undated credited course topic came due in the window; the fixture no longer exercises the case")
	}
	for _, s := range r.serves {
		for _, f := range s.Flags {
			if strings.HasPrefix(f, "surfaces disagree") {
				t.Errorf("session %d: %s", s.Session, f)
			}
		}
	}
}

// seedHeavyLearner writes a learner with 200 mastered topics: dated rows
// at every stage of SM-2 (intervals of 1, 6, 15 and 37 days, some past,
// some to come, some longer than the decay window), undated credited rows
// on course topics that fall due over the next fortnight, credited
// foundations no course lists (never due), mastered topics off every
// course's path (reviewed anyway), mastered floor topics (never served)
// and learning rows with practice dates (not reviews).
func seedHeavyLearner(t *testing.T, r *replay) {
	t.Helper()
	now := time.Now().UTC()
	at := func(d float64) *time.Time { v := now.Add(time.Duration(d * 24 * float64(time.Hour))); return &v }
	path, err := r.planner.PathForCourse("econ5153")
	if err != nil {
		t.Fatal(err)
	}
	onPath := map[string]bool{}
	var own, prereqs, floor, offPath []string
	for _, c := range path.Concepts {
		onPath[c.ID] = true
		switch {
		case r.own[c.ID]:
			own = append(own, c.ID)
		case floorDomains[c.Domain]:
			floor = append(floor, c.ID)
		default:
			prereqs = append(prereqs, c.ID)
		}
	}
	// Topics off every course's path, studied anyway (C's z-scores and
	// arithmetic series): reviewed like any other mastered topic.
	for _, c := range r.dag.Order() {
		if !onPath[c.ID] && !floorDomains[c.Domain] {
			offPath = append(offPath, c.ID)
		}
	}
	sort.Strings(own)
	sort.Strings(prereqs)
	sort.Strings(floor)
	sort.Strings(offPath)
	if len(own) < 30 || len(prereqs) < 50 || len(offPath) < 100 {
		t.Fatalf("path has %d own topics, %d prerequisites and %d topics off it; the fixture wants more", len(own), len(prereqs), len(offPath))
	}
	var rows []*storage.ConceptProgress
	mastered := 0
	// Own topics: half demonstrated with a date, half credited without one.
	for i, id := range own {
		p := &storage.ConceptProgress{StudentID: r.sid, ConceptID: id, Status: string(mastery.StatusMastered), SM2EFactor: 2.5}
		if i%2 == 0 {
			interval := []int{1, 6, 15}[i/2%3]
			reviewed := -float64(i%20) - 1 // 1 to 20 days ago
			p.Attempts, p.Streak, p.SM2Interval, p.SM2Repetitions = 6, 3, interval, i/2%3
			p.LastAttempted, p.LastReviewed, p.MasteredAt = at(reviewed), at(reviewed), at(reviewed)
			p.NextReviewDue = at(reviewed + float64(interval))
		} else {
			p.PlacementSeeded = true
			p.LastReviewed = at(-float64(i % 14)) // due from day 1 to day 14
		}
		rows = append(rows, p)
		mastered++
	}
	// Prerequisites no course lists: credited (never due), and demonstrated
	// (reviewed like any other), up to 200 mastered in all.
	for i, id := range prereqs {
		if mastered >= 200 {
			// The rest are learning rows with a practice date, not reviews.
			if i%3 == 0 {
				rows = append(rows, &storage.ConceptProgress{StudentID: r.sid, ConceptID: id, Status: string(mastery.StatusLearning), Attempts: 2, Streak: 1, LastAttempted: at(-2), NextReviewDue: at(-1), SM2EFactor: 2.5})
			}
			continue
		}
		p := &storage.ConceptProgress{StudentID: r.sid, ConceptID: id, Status: string(mastery.StatusMastered), SM2EFactor: 2.5}
		if i%3 == 0 {
			reviewed := -float64(i % 25)
			p.Attempts, p.Streak, p.SM2Interval = 5, 3, 6
			p.LastAttempted, p.LastReviewed, p.MasteredAt = at(reviewed), at(reviewed), at(reviewed)
			p.NextReviewDue = at(reviewed + 6)
		} else {
			p.PlacementSeeded = true
			p.ImplicitCredit = float64(i % 2)
			p.LastReviewed = at(-float64(i % 20))
		}
		rows = append(rows, p)
		mastered++
	}
	for i, id := range offPath {
		if mastered >= 200 {
			break
		}
		reviewed := -float64(i % 30)
		interval := []int{1, 6, 15, 37}[i%4]
		rows = append(rows, &storage.ConceptProgress{StudentID: r.sid, ConceptID: id, Status: string(mastery.StatusMastered), Attempts: 7, Streak: 3, LastAttempted: at(reviewed), LastReviewed: at(reviewed), MasteredAt: at(reviewed - 1), NextReviewDue: at(reviewed + float64(interval)), SM2EFactor: 2.5, SM2Interval: interval, SM2Repetitions: i % 4})
		mastered++
	}
	for _, id := range floor {
		rows = append(rows, &storage.ConceptProgress{StudentID: r.sid, ConceptID: id, Status: string(mastery.StatusMastered), Attempts: 4, Streak: 3, LastAttempted: at(-10), LastReviewed: at(-10), MasteredAt: at(-10), NextReviewDue: at(-9), SM2EFactor: 2.5, SM2Interval: 1})
	}
	for _, p := range rows {
		if err := r.repo.UpsertProgress(p); err != nil {
			t.Fatal(err)
		}
	}
	if mastered < 200 {
		t.Fatalf("seeded %d mastered topics, want 200", mastered)
	}
	t.Logf("heavy learner: %d rows, %d mastered (%d own, %d prerequisites, %d off the path, %d floor)", len(rows), mastered, len(own), len(prereqs), len(offPath), len(floor))
}
