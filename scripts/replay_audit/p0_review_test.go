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

// Independent review of FSRS Phase 0 (plan 6.1): engine-level adversarial
// cases on a learner who follows econ5153, with rows the author's fixtures
// may not pin down one by one. Every surface is read through dueSurfaces,
// so a disagreement anywhere fails the case.

type p0Learner struct {
	r                   *replay
	own, prereqs, floor []string
	offPath             []string
	sid                 string
	at                  func(days float64) *time.Time
}

func newP0Learner(t *testing.T, seed int64) *p0Learner {
	t.Helper()
	root := os.Getenv("MATHUA_ROOT")
	if root == "" {
		abs, err := filepath.Abs("../..")
		if err != nil {
			t.Fatal(err)
		}
		root = abs
		t.Setenv("MATHUA_ROOT", root)
	}
	mods, err := modules.LoadPublic(root)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	db := filepath.Join(dir, "p0.db")
	store, err := storage.NewSQLiteStore(db)
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.CreateStudent("p0-review")
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
	r, err := openWith(db, st.ID, seed, mods)
	if err != nil {
		t.Fatal(err)
	}
	l := &p0Learner{r: r, sid: st.ID}
	now := time.Now().UTC()
	l.at = func(d float64) *time.Time { v := now.Add(time.Duration(d * 24 * float64(time.Hour))); return &v }
	path, err := r.planner.PathForCourse("econ5153")
	if err != nil {
		t.Fatal(err)
	}
	onPath := map[string]bool{}
	for _, c := range path.Concepts {
		onPath[c.ID] = true
		switch {
		case r.own[c.ID]:
			l.own = append(l.own, c.ID)
		case floorDomains[c.Domain]:
			l.floor = append(l.floor, c.ID)
		default:
			l.prereqs = append(l.prereqs, c.ID)
		}
	}
	for _, c := range r.dag.Order() {
		if !onPath[c.ID] && !floorDomains[c.Domain] {
			l.offPath = append(l.offPath, c.ID)
		}
	}
	sort.Strings(l.own)
	sort.Strings(l.prereqs)
	sort.Strings(l.floor)
	sort.Strings(l.offPath)
	if len(l.own) < 3 || len(l.prereqs) < 3 || len(l.floor) < 1 || len(l.offPath) < 3 {
		t.Fatalf("fixture: own=%d prereqs=%d floor=%d offPath=%d", len(l.own), len(l.prereqs), len(l.floor), len(l.offPath))
	}
	return l
}

func (l *p0Learner) upsert(t *testing.T, p *storage.ConceptProgress) {
	t.Helper()
	p.StudentID = l.sid
	if p.SM2EFactor == 0 {
		p.SM2EFactor = 2.5
	}
	if err := l.r.repo.UpsertProgress(p); err != nil {
		t.Fatal(err)
	}
}

func (l *p0Learner) due(t *testing.T) dueReport {
	t.Helper()
	rep := l.r.dueSurfaces()
	if len(rep.Disagreements) > 0 {
		t.Errorf("surfaces disagree: %s\n  %s", rep, strings.Join(rep.Disagreements, "\n  "))
	}
	return rep
}

func has(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// 1. A MASTERED own-course topic with a 37-day SM-2 date, last reviewed 20
// days ago: decayed for pacing, NOT due on any surface. Its sibling with a
// date that passed yesterday is due everywhere.
func TestP0Review_FarFutureDateNotDue(t *testing.T) {
	l := newP0Learner(t, 11)
	far, near := l.own[0], l.own[1]
	l.upsert(t, &storage.ConceptProgress{ConceptID: far, Status: string(mastery.StatusMastered), Attempts: 9, Streak: 3, SM2Interval: 37, SM2Repetitions: 4,
		LastAttempted: l.at(-20), LastReviewed: l.at(-20), MasteredAt: l.at(-20), NextReviewDue: l.at(17)})
	l.upsert(t, &storage.ConceptProgress{ConceptID: near, Status: string(mastery.StatusMastered), Attempts: 6, Streak: 3, SM2Interval: 6, SM2Repetitions: 2,
		LastAttempted: l.at(-7), LastReviewed: l.at(-7), MasteredAt: l.at(-7), NextReviewDue: l.at(-1)})
	rep := l.due(t)
	if has(rep.Due, far) || has(rep.Review, far) || has(rep.Feed, far) || has(rep.CourseDue, far) {
		t.Errorf("%s (decayed, date in 17 days) is due somewhere: %+v", far, rep)
	}
	if !has(rep.Due, near) || !has(rep.Review, near) || rep.Count != 1 {
		t.Errorf("%s (date passed yesterday) missing: %+v", near, rep)
	}
	// Pacing still reads it as decayed (plan 3.5: the notions stay apart).
	progress, _ := l.r.repo.GetAllProgress(l.sid)
	snaps := l.r.eng.ConceptSnapshots(l.sid, progress)
	if st := mastery.EffectiveStatus(snaps[far].Status, 20, mastery.DecayDays); st != "DECAYING" {
		t.Errorf("pacing status of %s = %s, want DECAYING", far, st)
	}
}

// 2. Credited rows: an own-course topic credited 15 days ago is due (counted
// and served); a credit-only foundation no course lists, credited 15 days
// ago, is not, until it is pinned, and not again once unpinned.
func TestP0Review_CreditedRowsPinnedAndUnpinned(t *testing.T) {
	l := newP0Learner(t, 12)
	ownCredited, foundation := l.own[0], l.prereqs[0]
	l.upsert(t, &storage.ConceptProgress{ConceptID: ownCredited, Status: string(mastery.StatusMastered), PlacementSeeded: true, LastReviewed: l.at(-15)})
	l.upsert(t, &storage.ConceptProgress{ConceptID: foundation, Status: string(mastery.StatusMastered), PlacementSeeded: true, ImplicitCredit: 1, LastReviewed: l.at(-15)})

	rep := l.due(t)
	if !has(rep.Due, ownCredited) || !has(rep.Review, ownCredited) || !has(rep.CourseDue, ownCredited) {
		t.Errorf("credited own topic %s not due everywhere: %+v", ownCredited, rep)
	}
	if has(rep.Due, foundation) || has(rep.Review, foundation) || has(rep.Feed, foundation) {
		t.Errorf("credit-only foundation %s is due while unpinned: %+v", foundation, rep)
	}
	if rep.Count != 1 {
		t.Errorf("count = %d, want 1", rep.Count)
	}
	if rep.UpNextWhy == "" || !strings.HasPrefix(rep.UpNextWhy, "review due 1 day ago") {
		t.Errorf("Up next reason %q, want 'review due 1 day ago' (credited 15 days ago, 14-day horizon)", rep.UpNextWhy)
	}

	if _, err := l.r.eng.SetPins(l.sid, []string{foundation}); err != nil {
		t.Fatal(err)
	}
	rep = l.due(t)
	if !has(rep.Due, foundation) || !has(rep.Review, foundation) || rep.Count != 2 {
		t.Errorf("pinned foundation %s not due: %+v", foundation, rep)
	}

	if _, err := l.r.eng.SetPins(l.sid, nil); err != nil {
		t.Fatal(err)
	}
	rep = l.due(t)
	if has(rep.Due, foundation) || has(rep.Review, foundation) || rep.Count != 1 {
		t.Errorf("unpinned foundation %s still due: %+v", foundation, rep)
	}
}

//  3. A floor-domain row with a past date is never served or counted.
//  4. An orphan row (a concept the graph no longer has) is never counted,
//     and no surface trips over it.
//  5. A LEARNING row with a past next_review_due is a practice date.
func TestP0Review_FloorOrphanAndLearningRows(t *testing.T) {
	l := newP0Learner(t, 13)
	floor := l.floor[0]
	l.upsert(t, &storage.ConceptProgress{ConceptID: floor, Status: string(mastery.StatusMastered), Attempts: 4, Streak: 3, SM2Interval: 1,
		LastAttempted: l.at(-10), LastReviewed: l.at(-10), MasteredAt: l.at(-10), NextReviewDue: l.at(-9)})
	l.upsert(t, &storage.ConceptProgress{ConceptID: "zz.review.orphan_topic", Status: string(mastery.StatusMastered), Attempts: 4, Streak: 3, SM2Interval: 1,
		LastAttempted: l.at(-10), LastReviewed: l.at(-10), MasteredAt: l.at(-10), NextReviewDue: l.at(-9)})
	l.upsert(t, &storage.ConceptProgress{ConceptID: "zz.review.orphan_credited", Status: string(mastery.StatusMastered), PlacementSeeded: true, LastReviewed: l.at(-30)})
	learning := l.own[2]
	l.upsert(t, &storage.ConceptProgress{ConceptID: learning, Status: string(mastery.StatusLearning), Attempts: 2, Streak: 1,
		LastAttempted: l.at(-2), LastReviewed: l.at(-2), NextReviewDue: l.at(-1)})
	// One genuinely due row so the review session runs at all.
	due := l.own[1]
	l.upsert(t, &storage.ConceptProgress{ConceptID: due, Status: string(mastery.StatusMastered), Attempts: 6, Streak: 3, SM2Interval: 6,
		LastAttempted: l.at(-7), LastReviewed: l.at(-7), MasteredAt: l.at(-7), NextReviewDue: l.at(-1)})

	rep := l.due(t)
	for _, bad := range []string{floor, "zz.review.orphan_topic", "zz.review.orphan_credited", learning} {
		if has(rep.Due, bad) || has(rep.Review, bad) || has(rep.Feed, bad) || has(rep.CourseDue, bad) {
			t.Errorf("%s is due somewhere: %+v", bad, rep)
		}
	}
	if rep.Count != 1 || !has(rep.Review, due) {
		t.Errorf("count = %d, review = %v; want 1 and %s", rep.Count, rep.Review, due)
	}
	// The course view and the profile's reasons do not panic on the orphan.
	if _, err := l.r.eng.RecommendNext(l.sid, nil); err != nil {
		t.Errorf("RecommendNext: %v", err)
	}
}

//  6. The count and the session read the same instant: a row that falls due
//     between two reads is the only way they can differ, and once both are
//     past the date they agree. The window is the time between the two
//     HTTP requests, not a rule difference.
func TestP0Review_CountAndSessionAgreeAroundTheDate(t *testing.T) {
	l := newP0Learner(t, 14)
	id := l.own[0]
	soon := time.Now().UTC().Add(1500 * time.Millisecond)
	l.upsert(t, &storage.ConceptProgress{ConceptID: id, Status: string(mastery.StatusMastered), Attempts: 6, Streak: 3, SM2Interval: 6,
		LastAttempted: l.at(-7), LastReviewed: l.at(-7), MasteredAt: l.at(-7), NextReviewDue: &soon})
	// Before the date: nothing anywhere.
	rep := l.due(t)
	if rep.Count != 0 || len(rep.Review) != 0 || rep.UpNext {
		t.Errorf("before the date: %+v", rep)
	}
	// SQLite stores the date to the second; wait past it with a margin.
	time.Sleep(time.Until(soon) + 1200*time.Millisecond)
	rep = l.due(t)
	if rep.Count != 1 || !has(rep.Review, id) || !rep.UpNext {
		t.Errorf("after the date: %+v", rep)
	}
	if !strings.HasPrefix(rep.UpNextWhy, "review due today") {
		t.Errorf("reason %q, want 'review due today'", rep.UpNextWhy)
	}
}

//  7. Off-path reviews: a mastered topic off every course's path with a
//     past date is counted and served by the review session AND the mixed
//     session (the report's behaviour change), not path-skipped.
func TestP0Review_OffPathDueReviewServedByMixedSession(t *testing.T) {
	l := newP0Learner(t, 15)
	off := l.offPath[0]
	l.upsert(t, &storage.ConceptProgress{ConceptID: off, Status: string(mastery.StatusMastered), Attempts: 7, Streak: 3, SM2Interval: 6,
		LastAttempted: l.at(-9), LastReviewed: l.at(-9), MasteredAt: l.at(-9), NextReviewDue: l.at(-3)})
	rep := l.due(t)
	if !has(rep.Due, off) || !has(rep.Review, off) || !has(rep.Feed, off) || rep.Count != 1 {
		t.Errorf("off-path due review %s: %+v", off, rep)
	}
	if rep.FeedFirst != off {
		t.Errorf("mixed session opened with %s, want the due review %s", rep.FeedFirst, off)
	}
}
