package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/auth"
	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/engine"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
	"github.com/chuma-beep/mathua/internal/proof"
	"github.com/chuma-beep/mathua/internal/storage"
)

type proofGen struct{}

func (proofGen) Generate(generator.GeneratorContext) generator.Problem {
	return generator.Problem{Question: "**Prove.** the claim\n\n" + proof.Marker, Answer: "proof:p1", Explanation: "reference", Raw: true}
}

func proofServer(t *testing.T) *http.ServeMux {
	t.Helper()
	d, _ := concepts.Build([]concepts.Concept{{ID: "ra.x", Label: "X", Domain: "d", GradingType: "multiple_choice",
		MasteryThreshold: concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 60}}})
	store, _ := storage.NewSQLiteStore(":memory:")
	t.Cleanup(func() { store.Close() })
	reg := generator.NewRegistry()
	reg.Register("ra.x", proofGen{})
	s := New(engine.New(store, d, reg, nil, nil), store, auth.New(store))
	mux := http.NewServeMux()
	s.Register(mux)
	return mux
}

// A proof submitted through Learn is graded by the proof grader, its report
// reaches the client, a long proof is accepted, and an ungraded proof says so.
func TestStudyAnswerCarriesProofReport(t *testing.T) {
	mux := proofServer(t)
	var verdict grader.Result
	grader.SetProofGrader(func(e, a string) grader.Result { return verdict })
	t.Cleanup(func() { grader.SetProofGrader(nil) })
	token, _ := guestTokenAndID(t, mux)
	code, body := getBody(t, mux, token, "/api/lessons/ra.x/practice?count=1&difficulty=0.6")
	if code != 200 {
		t.Fatalf("practice: %d %v", code, body)
	}
	question := body["questions"].([]interface{})[0].(map[string]interface{})["question"].(string)
	long := strings.Repeat("Since x > 0, ", 500) // about 6500 bytes: past the typed-answer bound
	post := func(answer, q string) (int, map[string]interface{}) {
		payload, _ := json.Marshal(map[string]interface{}{"concept_id": "ra.x", "answer": answer, "elapsed": 300.0, "question": q})
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/study/answer", strings.NewReader(string(payload)))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)
		var res map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &res)
		return rec.Code, res
	}
	verdict = grader.Result{Unavailable: true, Feedback: "budget used up"}
	code, res := post(long, question)
	if code != 200 || res["ungraded"] != true {
		t.Fatalf("an ungraded proof must say so: %d %v", code, res)
	}
	verdict = grader.Result{Score: 4.0 / 7, Feedback: "Score 4/7.", Proof: &grader.ProofReport{Score: 4, Max: 7, Feedback: "gap at step 2"}}
	code, res = post(long, question)
	if code != 200 || res["ungraded"] == true || res["correct"] != false {
		t.Fatalf("%d %v", code, res)
	}
	pr, _ := res["proof"].(map[string]interface{})
	if pr == nil || pr["score"] != 4.0 || pr["feedback"] != "gap at step 2" {
		t.Fatalf("proof report missing: %v", res)
	}
	if code, _ := post(long, "What is 2+2?"); code != 400 {
		t.Fatalf("a long answer to an ordinary question must be refused, got %d", code)
	}
}
