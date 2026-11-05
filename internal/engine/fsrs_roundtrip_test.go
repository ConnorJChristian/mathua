package engine

import (
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/diagnostic"
	"github.com/chuma-beep/mathua/internal/storage"
)

// Every whole-row write path keeps the FSRS columns (FSRS plan 3.2, 4.7:
// the release blocker). Since the Phase 1 review the progress upserts never
// write them (only WriteFSRSState does), so "keeps" means the stored state
// at write time, not the snapshot's. The graph is a <- b. A row on "a" carries FSRS
// state; each path rewrites rows and the state must come through. The
// store-level statements are tested on SQLite and Postgres in
// internal/storage/fsrs_roundtrip_test.go; these are the engine paths that
// build the batches.

func seedFSRS(t *testing.T, e *Engine, sid, cid, status string) *storage.ConceptProgress {
	t.Helper()
	s, d := 15.25, 3.5
	last := time.Now().UTC().Add(-48 * time.Hour).Truncate(time.Second)
	rev := last
	row := &storage.ConceptProgress{
		StudentID: sid, ConceptID: cid, Status: status, Streak: 3, Attempts: 5, LastReviewed: &rev, LastAttempted: &rev,
		FSRSStability: &s, FSRSDifficulty: &d, FSRSLastReview: &last, FSRSReps: 2, FSRSLapses: 0, FSRSState: "review", FSRSSeed: "mastery",
	}
	if err := e.repo.UpsertProgress(row); err != nil {
		t.Fatal(err)
	}
	if err := e.repo.WriteFSRSState(row); err != nil {
		t.Fatal(err)
	}
	return row
}

func assertFSRSKept(t *testing.T, path string, e *Engine, sid, cid string, want *storage.ConceptProgress) {
	t.Helper()
	got, err := e.repo.GetProgress(sid, cid)
	if err != nil || got == nil {
		t.Fatalf("%s: row missing: %v", path, err)
	}
	if got.FSRSStability == nil || *got.FSRSStability != *want.FSRSStability ||
		got.FSRSDifficulty == nil || *got.FSRSDifficulty != *want.FSRSDifficulty ||
		got.FSRSLastReview == nil || !got.FSRSLastReview.Equal(*want.FSRSLastReview) ||
		got.FSRSReps != want.FSRSReps || got.FSRSLapses != want.FSRSLapses || got.FSRSState != want.FSRSState || got.FSRSSeed != want.FSRSSeed {
		t.Fatalf("%s dropped FSRS state: %+v", path, got)
	}
}

func TestFSRSColumns_SurvivePropagateWeakness(t *testing.T) {
	e := testEngine(t)
	sid := creditStudent(t, e)
	want := seedFSRS(t, e, sid, "b", "MASTERED")
	// a weak enough to propagate onto its dependent b (w > 0.3, b below 0.3w).
	if err := e.repo.UpsertProgress(&storage.ConceptProgress{StudentID: sid, ConceptID: "a", Status: "LEARNING", WeaknessScore: 0.9}); err != nil {
		t.Fatal(err)
	}
	e.PropagateWeakness(sid)
	got, _ := e.repo.GetProgress(sid, "b")
	if got.WeaknessScore <= 0 {
		t.Fatalf("weakness did not propagate onto b: %+v (the path under test did not run)", got)
	}
	assertFSRSKept(t, "PropagateWeakness", e, sid, "b", want)
}

func TestFSRSColumns_SurviveCreditPrerequisites(t *testing.T) {
	e := testEngine(t)
	sid := creditStudent(t, e)
	// A credited row with FSRS state (the strong-link refresh rewrites it).
	want := seedFSRS(t, e, sid, "a", "MASTERED")
	want.PlacementSeeded = true
	want.ImplicitCredit = 1
	if err := e.repo.UpsertProgressBatch([]*storage.ConceptProgress{want}); err != nil {
		t.Fatal(err)
	}
	e.creditPrerequisites(sid, "b", false)
	got, _ := e.repo.GetProgress(sid, "a")
	if got.LastReviewed == nil || !got.LastReviewed.After(*want.LastReviewed) {
		t.Fatalf("the credit refresh did not run: %+v", got)
	}
	assertFSRSKept(t, "creditPrerequisites (refresh)", e, sid, "a", want)
	// And a row that gains credit (partial, LEARNING with state from a lapse).
	e2 := testEngine(t)
	sid2 := creditStudent(t, e2)
	want2 := seedFSRS(t, e2, sid2, "a", "LEARNING")
	e2.creditPrerequisites(sid2, "b", true)
	got2, _ := e2.repo.GetProgress(sid2, "a")
	if got2.ImplicitCredit <= 0 {
		t.Fatalf("no credit given: %+v", got2)
	}
	assertFSRSKept(t, "creditPrerequisites (credit)", e2, sid2, "a", want2)
}

func TestFSRSColumns_SurviveApplyGoalResults(t *testing.T) {
	e := testEngine(t)
	sid := creditStudent(t, e)
	// The diagnostic rewrites only rows still UNSEEN (it never destroys
	// mastery), so an UNSEEN row carrying state (a lapse-seeded or
	// Anki-seeded topic never practised here) is the case.
	want := seedFSRS(t, e, sid, "a", "UNSEEN")
	session := &diagnostic.Session{StudentID: sid, Attempts: []diagnostic.Attempt{{ConceptID: "a", Correct: true, Fast: true}}}
	if err := e.ApplyGoalResults(sid, session); err != nil {
		t.Fatal(err)
	}
	got, _ := e.repo.GetProgress(sid, "a")
	if !got.PlacementSeeded || got.Status != "PRACTICING" {
		t.Fatalf("placement did not write a: %+v", got)
	}
	assertFSRSKept(t, "ApplyGoalResults", e, sid, "a", want)
}

func TestFSRSColumns_SurviveGradedAnswer(t *testing.T) {
	e := testEngine(t)
	sid := creditStudent(t, e)
	want := seedFSRS(t, e, sid, "a", "LEARNING")
	answerCorrectly(t, e, sid, 2)
	got, _ := e.repo.GetProgress(sid, "a")
	if got.Attempts != want.Attempts+2 {
		t.Fatalf("answers not recorded: %+v", got)
	}
	// With MATHUA_FSRS off nothing moves the state, so it reads as seeded.
	assertFSRSKept(t, "applyGradedAttempt", e, sid, "a", want)
}

// interleavingRepo runs before() just ahead of the first batch write: a
// review commit that lands between a path's snapshot read and its write.
type interleavingRepo struct {
	storage.Repository
	before func()
}

func (r *interleavingRepo) UpsertProgressBatch(ps []*storage.ConceptProgress) error {
	if r.before != nil {
		r.before()
		r.before = nil
	}
	return r.Repository.UpsertProgressBatch(ps)
}

// PropagateWeakness reads every row, then writes the batch. An FSRS write
// between the two survives the batch (FSRS Phase 1 review, Phase 2 note:
// lost updates).
func TestFSRSColumns_InterleavedReviewSurvivesPropagateWeakness(t *testing.T) {
	e := testEngine(t)
	sid := creditStudent(t, e)
	seedFSRS(t, e, sid, "b", "MASTERED")
	if err := e.repo.UpsertProgress(&storage.ConceptProgress{StudentID: sid, ConceptID: "a", Status: "LEARNING", WeaknessScore: 0.9}); err != nil {
		t.Fatal(err)
	}
	base := e.repo
	newer, _ := base.GetProgress(sid, "b")
	s2 := 99.5
	newer.FSRSStability, newer.FSRSReps = &s2, 3
	e.repo = &interleavingRepo{Repository: base, before: func() {
		if err := base.WriteFSRSState(newer); err != nil {
			t.Error(err)
		}
	}}
	defer func() { e.repo = base }()
	e.PropagateWeakness(sid)
	got, _ := base.GetProgress(sid, "b")
	if got.WeaknessScore <= 0 {
		t.Fatalf("weakness did not propagate onto b: %+v (the path under test did not run)", got)
	}
	assertFSRSKept(t, "PropagateWeakness after an interleaved review", e, sid, "b", newer)
}
