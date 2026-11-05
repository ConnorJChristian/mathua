package repair

import (
	"bytes"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/engine"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/storage"
)

func testRepo(t *testing.T) (*storage.SQLiteStore, *concepts.DAG, string, string) {
	t.Helper()
	dag, err := concepts.Build([]concepts.Concept{{
		ID: "t", Label: "T", Domain: "d", GradingType: "numeric",
		MasteryThreshold: concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 10},
	}})
	if err != nil {
		t.Fatalf("build dag: %v", err)
	}
	repo, err := storage.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { repo.Close() })
	stu, err := repo.CreateStudent("S")
	if err != nil {
		t.Fatalf("student: %v", err)
	}
	sess, err := repo.CreateSession(stu.ID)
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	return repo, dag, stu.ID, sess.ID
}

func TestRepair_FlipsFalseNegative(t *testing.T) {
	repo, dag, sid, sess := testRepo(t)
	base := time.Now().UTC().Add(-time.Hour)

	// False negative: the old numeric grader rejected the spoken mixed number.
	must(t, func() error {
		_, err := repo.RecordAttempt(storage.AttemptEntry{
			SessionID: sess, StudentID: sid, ConceptID: "t",
			Answer: "4 and 1/10", Expected: "4 1/10", Correct: false,
			ElapsedSeconds: 5, Timestamp: base,
		})
		return err
	}())
	// Genuine miss: must stay incorrect after repair.
	must(t, func() error {
		_, err := repo.RecordAttempt(storage.AttemptEntry{
			SessionID: sess, StudentID: sid, ConceptID: "t",
			Answer: "5", Expected: "4 1/10", Correct: false,
			ElapsedSeconds: 9, Timestamp: base.Add(time.Minute),
		})
		return err
	}())

	reg := generator.NewRegistry()
	eng := engine.New(repo, dag, reg, nil, nil)
	defer eng.Close()

	var buf bytes.Buffer
	sum, err := Run(eng, repo, dag, true, &buf)
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if sum.Repaired != 1 {
		t.Fatalf("dry repaired=%d, want 1\n%s", sum.Repaired, buf.String())
	}
	if all, _ := repo.GetAllAttempts(); all[0].Correct {
		t.Fatal("dry run must not write")
	}

	buf.Reset()
	sum, err = Run(eng, repo, dag, false, &buf)
	if err != nil {
		t.Fatalf("repair: %v", err)
	}
	if sum.Repaired != 1 || sum.ProgressWrites != 1 {
		t.Fatalf("repair repaired=%d writes=%d, want 1/1\n%s", sum.Repaired, sum.ProgressWrites, buf.String())
	}

	all, err := repo.GetAllAttempts()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	var flipped, wrong int
	for _, a := range all {
		if a.Answer == "4 and 1/10" && a.Correct {
			flipped++
		}
		if a.Answer == "5" && !a.Correct {
			wrong++
		}
	}
	if flipped != 1 {
		t.Errorf("false negative not flipped: %+v", all)
	}
	if wrong != 1 {
		t.Errorf("genuine miss was changed: %+v", all)
	}

	// Rebuilt progress reflects the corrected history (attempts counted).
	p, err := repo.GetProgress(sid, "t")
	if err != nil {
		t.Fatalf("progress: %v", err)
	}
	if p == nil || p.Attempts != 2 {
		t.Fatalf("progress not rebuilt: %+v", p)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
}

// A repair rebuilds the row from its attempts, whole. The FSRS state has its
// own history (the review_log) and must come through unchanged (FSRS plan
// Phase 1 round-trip, the "third copy of the state machine").
func TestRepair_KeepsFSRSState(t *testing.T) {
	repo, dag, sid, sess := testRepo(t)
	base := time.Now().UTC().Add(-time.Hour)
	must(t, func() error {
		_, err := repo.RecordAttempt(storage.AttemptEntry{
			SessionID: sess, StudentID: sid, ConceptID: "t",
			Answer: "4 and 1/10", Expected: "4 1/10", Correct: false,
			ElapsedSeconds: 5, Timestamp: base,
		})
		return err
	}())
	s, d := 21.5, 3.25
	last := base.Add(-24 * time.Hour).Truncate(time.Second) // RFC3339 storage
	must(t, repo.UpsertProgress(&storage.ConceptProgress{
		StudentID: sid, ConceptID: "t", Status: "LEARNING", Attempts: 1,
		FSRSStability: &s, FSRSDifficulty: &d, FSRSLastReview: &last, FSRSReps: 3, FSRSLapses: 1, FSRSState: "relearning", FSRSSeed: "mastery",
	}))
	eng := engine.New(repo, dag, generator.NewRegistry(), nil, nil)
	defer eng.Close()
	var buf bytes.Buffer
	sum, err := Run(eng, repo, dag, false, &buf)
	if err != nil || sum.ProgressWrites != 1 {
		t.Fatalf("repair: %v %+v\n%s", err, sum, buf.String())
	}
	got, _ := repo.GetProgress(sid, "t")
	if got == nil || got.FSRSStability == nil || *got.FSRSStability != s || got.FSRSDifficulty == nil || *got.FSRSDifficulty != d ||
		got.FSRSLastReview == nil || !got.FSRSLastReview.Equal(last) || got.FSRSReps != 3 || got.FSRSLapses != 1 || got.FSRSState != "relearning" || got.FSRSSeed != "mastery" {
		t.Fatalf("repair dropped FSRS state: %+v", got)
	}
	if got.Attempts != 1 || got.Streak != 1 {
		t.Errorf("repair did not rebuild the SM-2 side: %+v", got)
	}
}
