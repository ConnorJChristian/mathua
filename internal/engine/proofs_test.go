package engine

import (
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
	"github.com/chuma-beep/mathua/internal/mastery"
)

type proofSrcStub struct {
	serving bool
	share   float64
}

func (s proofSrcStub) Serving() bool  { return s.serving }
func (s proofSrcStub) Share() float64 { return s.share }
func (s proofSrcStub) ByConcept() map[string][]string {
	return map[string][]string{"a": {"p1"}, "ghost": {"p2"}}
}
func (s proofSrcStub) ProblemFor(id string) (generator.Problem, bool) {
	return generator.Problem{Question: "**Prove.** something <!--proof-answer-->", Answer: "proof:" + id, Raw: true}, true
}

func proofDAG(t *testing.T) *concepts.DAG {
	th := concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 60}
	d, err := concepts.Build([]concepts.Concept{{ID: "a", Label: "A", Domain: "d", GradingType: "numeric", MasteryThreshold: th}})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// A topic with proof problems serves them in its practice, marked as cards,
// but never in placement or a quiz, and never while the grader cannot grade.
func TestProofsServedInTopicPractice(t *testing.T) {
	reg := generator.NewRegistry()
	reg.Register("a", &testGen{answer: "42"})
	attachProofsFrom(proofDAG(t), reg, proofSrcStub{serving: true, share: 1})
	attachProofsFrom(proofDAG(t), reg, proofSrcStub{serving: true, share: 1}) // wrapped once

	p, _ := reg.GenerateContext("a", generator.GeneratorContext{Difficulty: 0.5})
	if p.Answer != "proof:p1" || !strings.HasPrefix(p.Question, CardMarker+"\n") || strings.Count(p.Question, CardMarker) != 1 {
		t.Fatalf("want a proof in practice, got %q / %q", p.Question, p.Answer)
	}
	for _, ctx := range []generator.GeneratorContext{{Difficulty: 0.5, NoCards: true}, {Difficulty: 0.5, NoProofs: true}} {
		if p, _ := reg.GenerateContext("a", ctx); p.Answer != "42" {
			t.Fatalf("%+v must get the topic's own question, got %q", ctx, p.Answer)
		}
	}

	reg2 := generator.NewRegistry()
	reg2.Register("a", &testGen{answer: "42"})
	attachProofsFrom(proofDAG(t), reg2, proofSrcStub{serving: false, share: 1})
	if p, _ := reg2.GenerateContext("a", generator.GeneratorContext{Difficulty: 0.5}); p.Answer != "42" {
		t.Fatal("a proof was served while the grader cannot grade")
	}
	// Cards attach on top without wrapping the topic again.
	attachCardsFrom(proofDAG(t), reg, nil)
	if g, _ := reg.Get("a"); g.(*proofMixGen) == nil {
		t.Fatal("wrapper replaced")
	}
}

// A proof is graded as a proof, and while MATHUA_PROOF_COUNTS_FOR_MASTERY is
// off a correct one moves nothing on the mastery ladder and a wrong one is no
// miss toward a halt. With the flag on it is ordinary evidence.
func TestProofResultsAreFeedbackOnlyUntilCalibrated(t *testing.T) {
	var verdict grader.Result
	// Installed after each engine is built: building one loads the default
	// proof grader, which installs itself.
	install := func() { grader.SetProofGrader(func(e, a string) grader.Result { return verdict }) }
	t.Cleanup(func() { grader.SetProofGrader(nil) })

	run := func(counts bool) (streak int, status string, halted bool, proofSeen bool) {
		old := proofsCountForMastery
		proofsCountForMastery = func() bool { return counts }
		defer func() { proofsCountForMastery = old }()
		e := testEngine(t)
		st, _ := e.CreateStudent("prover")
		install()
		verdict = grader.Result{Correct: true, Score: 1, Proof: &grader.ProofReport{Score: 7, Max: 7, FullyCorrect: true}}
		for i := 0; i < 3; i++ {
			res, err := e.SubmitQuizAnswer(st.ID, "a", "a long proof", "proof:p1", 300, "Prove it", nil)
			if err != nil {
				t.Fatal(err)
			}
			proofSeen = res.Proof != nil && res.Proof.Score == 7
		}
		p, _ := e.repo.GetProgress(st.ID, "a")
		streak, status = p.Streak, p.Status
		verdict = grader.Result{Score: 2.0 / 7, Proof: &grader.ProofReport{Score: 2, Max: 7}}
		var res *AnswerResult
		for i := 0; i < 2; i++ {
			res, _ = e.SubmitQuizAnswer(st.ID, "a", "a wrong proof", "proof:p1", 300, "Prove it", nil)
		}
		return streak, status, res.Halted, proofSeen
	}
	streak, status, halted, seen := run(false)
	if !seen {
		t.Fatal("the proof report must reach the answer result")
	}
	if streak != 0 || status != string(mastery.StatusUnseen) || halted {
		t.Fatalf("flag off: a proof moved mastery (streak %d, status %s, halted %v)", streak, status, halted)
	}
	streak, _, halted, _ = run(true)
	if streak == 0 || !halted {
		t.Fatalf("flag on: a proof must count (streak %d, halted %v)", streak, halted)
	}
}

func TestUngradedProofRecordsNothing(t *testing.T) {
	e := testEngine(t)
	grader.SetProofGrader(func(e, a string) grader.Result {
		return grader.Result{Unavailable: true, Feedback: "budget used up"}
	})
	t.Cleanup(func() { grader.SetProofGrader(nil) })
	st, _ := e.CreateStudent("prover2")
	res, err := e.SubmitQuizAnswer(st.ID, "a", "proof", "proof:p1", 300, "Prove it", nil)
	if err != nil || !res.Ungraded || res.Correct {
		t.Fatalf("%+v %v", res, err)
	}
	if p, _ := e.repo.GetProgress(st.ID, "a"); p != nil && p.Attempts != 0 {
		t.Fatalf("an ungraded proof was recorded: %+v", p)
	}
}
