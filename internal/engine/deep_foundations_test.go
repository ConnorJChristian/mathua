package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/generator/all"
	"github.com/chuma-beep/mathua/internal/planning"
	"github.com/chuma-beep/mathua/internal/storage"
)

// Q57: after an ECON 5153 + Abbott placement, "Power sets and counting
// subsets" pulled the mean of a data set and school probability into Up
// next (stat.data.mean is six links below it). The placement never asks a
// topic more than CoursePlacementDepth links below the course, and statistics
// is not a grade-school domain, so nothing hid them. Once the learner has
// placement evidence on the course, those deep foundations are past.
func TestDeepFoundationsArePastAfterPlacement(t *testing.T) {
	t.Setenv("MATHUA_REVIEW_FLOOR_DOMAINS", "")
	dag, err := concepts.LoadDirs(filepath.Join("..", "..", "data", "concepts"))
	if err != nil {
		t.Fatal(err)
	}
	cp := filepath.Join(t.TempDir(), "courses.json")
	if err := os.WriteFile(cp, []byte(`[{"id":"grad","name":"Mathematics for economists","grade":"MA","description":"",
		"plan":["discrete.sets.powerset","calc.deriv.chain_rule"]}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	pl, err := planning.Load(cp, dag)
	if err != nil {
		t.Fatal(err)
	}
	store, err := storage.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	e := New(store, dag, all.Registry(), nil, pl)
	st, err := e.CreateStudent("graduate")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetActiveCourses(st.ID, []string{"grad"}); err != nil {
		t.Fatal(err)
	}
	if err := store.SetDiagnosticCompleted(st.ID); err != nil {
		t.Fatal(err)
	}

	// Links below Power sets: combinations 1, permutations 2, the counting
	// principle 3, compound events 4, complement 5, basic probability 6,
	// the mean 7.
	deep := []string{"stat.prob.counting", "stat.prob.compound", "stat.prob.complement", "stat.prob.basic", "stat.data.mean"}
	near := []string{"discrete.combinatorics.permutations", "discrete.combinatorics.combinations"}

	// No evidence yet: nothing extra is hidden.
	if below := e.belowFloor(st.ID, nil); below["stat.data.mean"] {
		t.Fatal("stat.data.mean hidden before any placement evidence")
	}

	// The placement credits the chain rule (a course topic) and nothing on
	// the power-set chain.
	if err := store.UpsertProgressBatch([]*storage.ConceptProgress{{StudentID: st.ID, ConceptID: "calc.deriv.chain_rule",
		Status: "PRACTICING", PlacementSeeded: true, SM2EFactor: 2.5}}); err != nil {
		t.Fatal(err)
	}
	below := e.belowFloor(st.ID, nil)
	for _, id := range deep {
		if !below[id] {
			t.Errorf("%s (more than %d links below the course) is still served", id, CoursePlacementDepth)
		}
	}
	for _, id := range append(near, "discrete.sets.powerset", "calc.deriv.chain_rule") {
		if below[id] {
			t.Errorf("%s is within %d links of the course and must stay servable", id, CoursePlacementDepth)
		}
	}
	// Past, so met: the chain under Power sets opens at permutations.
	met := e.FloorMet(st.ID)
	if !met["stat.prob.counting"] {
		t.Error("stat.prob.counting is past but does not count as met")
	}
	// A deep foundation the learner has practised is still theirs.
	if err := store.UpsertProgressBatch([]*storage.ConceptProgress{{StudentID: st.ID, ConceptID: "stat.prob.basic",
		Status: "LEARNING", Attempts: 3, SM2EFactor: 2.5}}); err != nil {
		t.Fatal(err)
	}
	if e.belowFloor(st.ID, nil)["stat.prob.basic"] {
		t.Error("a practised foundation is hidden")
	}
	deep = []string{"stat.prob.counting", "stat.prob.compound", "stat.prob.complement", "stat.data.mean"}
	for i := 0; i < 5; i++ {
		res, err := e.RecommendNext(st.ID, nil)
		if err != nil {
			t.Fatal(err)
		}
		recs := res.Alternatives
		if res.Primary != nil {
			recs = append(recs, *res.Primary)
		}
		for _, r := range recs {
			for _, id := range deep {
				if r.ID == id {
					t.Fatalf("Up next offers %s: %+v", id, r)
				}
			}
		}
	}
}
