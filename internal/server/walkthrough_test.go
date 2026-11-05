package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/auth"
	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/engine"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/mastery"
	"github.com/chuma-beep/mathua/internal/planning"
	"github.com/chuma-beep/mathua/internal/storage"
	"github.com/chuma-beep/mathua/internal/walkthrough"
)

// The one feedback contract (fork change, FSRS_PORT_PLAN 0c): every graded
// answer, on every surface, comes back with `walkthrough`: the key, the steps,
// the one-line reason, where a wrong answer left the path, and the reading.
// These tests serve a generator whose explanation carries two steps and
// check the contract on Learn, review, the mastery check and the placement
// results, and that a wrong answer and a right one differ in the one way they
// should: only a wrong answer gets `diverged_at`.

const (
	stepsExplain = "Two and two make 4."
	stepOne      = "Start from the first operand, 2."
	stepTwo      = "Add the second: 2 + 2 = 4."
)

// stepsGen serves 2+2 with a worked path of two steps.
type stepsGen struct{}

func (g *stepsGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	return generator.Problem{Question: "2+2=?", Answer: "4",
		Explanation: walkthrough.Encode(stepsExplain, []string{stepOne, stepTwo})}
}

// walkthroughServer is a two-concept server with auth, both concepts served
// by stepsGen.
func walkthroughServer(t *testing.T) (*http.ServeMux, storage.Repository, string, string) {
	t.Helper()
	d, _ := concepts.Build([]concepts.Concept{
		{ID: "a", Label: "A", Domain: "d", GradingType: "numeric", Prerequisites: []string{},
			MasteryThreshold: concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 60}},
		{ID: "b", Label: "B", Domain: "d", GradingType: "numeric", Prerequisites: []string{"a"},
			MasteryThreshold: concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 60}},
	})
	store, _ := storage.NewSQLiteStore(":memory:")
	t.Cleanup(func() { store.Close() })
	reg := generator.NewRegistry()
	reg.Register("a", &stepsGen{})
	reg.Register("b", &stepsGen{})
	s := New(engine.New(store, d, reg, nil, planning.New(d)), store, auth.New(store))
	mux := http.NewServeMux()
	s.Register(mux)
	token, studentID := quizGuest(t, mux)
	return mux, store, token, studentID
}

type wireWalkthrough struct {
	Answer     string   `json:"answer"`
	Steps      []string `json:"steps"`
	Explain    string   `json:"explain"`
	DivergedAt *struct {
		Step   int    `json:"step"`
		Reason string `json:"reason"`
	} `json:"diverged_at"`
	Reading *struct {
		Label string `json:"label"`
		URL   string `json:"url"`
	} `json:"reading"`
}

// assertWalkthrough checks the contract's shape for the served 2+2 question.
func assertWalkthrough(t *testing.T, what string, w *wireWalkthrough, wrong bool) {
	t.Helper()
	if w == nil {
		t.Fatalf("%s: no walkthrough", what)
	}
	if w.Answer != "4" {
		t.Errorf("%s: answer = %q, want 4", what, w.Answer)
	}
	if len(w.Steps) != 2 || w.Steps[0] != stepOne || w.Steps[1] != stepTwo {
		t.Errorf("%s: steps = %q", what, w.Steps)
	}
	if w.Explain != stepsExplain {
		t.Errorf("%s: explain = %q", what, w.Explain)
	}
	if wrong && w.DivergedAt == nil {
		t.Errorf("%s: a wrong answer equal to step 1's value must say so", what)
	}
	if !wrong && w.DivergedAt != nil {
		t.Errorf("%s: a right answer has no divergence: %+v", what, w.DivergedAt)
	}
	if wrong && w.DivergedAt != nil && w.DivergedAt.Step != 1 {
		t.Errorf("%s: diverged_at.step = %d, want 1", what, w.DivergedAt.Step)
	}
}

func servePractice(t *testing.T, mux *http.ServeMux, token, studentID, concept string) string {
	t.Helper()
	rec := httptest.NewRecorder()
	get := httptest.NewRequest("GET", "/api/lessons/"+concept+"/practice?student_id="+studentID+"&count=1", nil)
	get.Header.Set("Authorization", "Bearer "+token)
	mux.ServeHTTP(rec, get)
	if rec.Code != 200 {
		t.Fatalf("practice: %d %s", rec.Code, rec.Body.String())
	}
	var set struct {
		Questions []struct{ Question string } `json:"questions"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &set); err != nil || len(set.Questions) == 0 {
		t.Fatalf("decode practice: %v %s", err, rec.Body.String())
	}
	return set.Questions[0].Question
}

// Learn: wrong, right and "I don't know".
func TestWalkthrough_LearnWrongRightAndDontKnow(t *testing.T) {
	mux, _, token, studentID := walkthroughServer(t)
	for _, tc := range []struct {
		name, body string
		wrong      bool
		dontKnow   bool
	}{
		{"wrong", `"answer":"2"`, true, false},
		{"right", `"answer":"4"`, false, false},
		{"dont know", `"answer":"","dont_know":true`, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := servePractice(t, mux, token, studentID, "a")
			code, body := postJSONRaw(t, mux, "/api/study/answer",
				`{"concept_id":"a",`+tc.body+`,"elapsed":5.0,"question":`+quoteJSON(q)+`,"student_id":"`+studentID+`"}`, token)
			if code != 200 {
				t.Fatalf("study answer: %d %s", code, body)
			}
			var res struct {
				Correct     bool             `json:"correct"`
				Walkthrough *wireWalkthrough `json:"walkthrough"`
			}
			if err := json.Unmarshal([]byte(body), &res); err != nil {
				t.Fatal(err)
			}
			if res.Correct == (tc.wrong || tc.dontKnow) {
				t.Fatalf("correct = %v: %s", res.Correct, body)
			}
			assertWalkthrough(t, "learn "+tc.name, res.Walkthrough, tc.wrong)
			if tc.dontKnow && res.Walkthrough.DivergedAt != nil {
				t.Errorf("I don't know has nothing to diverge from: %+v", res.Walkthrough.DivergedAt)
			}
		})
	}
}

// Review: the result of /api/reviews/answer carries the contract.
func TestWalkthrough_Review(t *testing.T) {
	mux, store, token, studentID := walkthroughServer(t)
	past := time.Now().Add(-48 * time.Hour)
	due := time.Now().Add(-time.Hour)
	if err := store.UpsertProgress(&storage.ConceptProgress{
		StudentID: studentID, ConceptID: "a", Status: string(mastery.StatusMastered), Attempts: 6, Streak: 3,
		LastAttempted: &past, LastReviewed: &past, MasteredAt: &past, NextReviewDue: &due, SM2EFactor: 2.5, SM2Interval: 1,
	}); err != nil {
		t.Fatal(err)
	}
	sess := quizPost(t, mux, "/api/reviews/session", `{}`, token)
	q, _ := sess["question"].(map[string]interface{})
	if q == nil {
		t.Fatalf("no review question: %v", sess)
	}
	code, body := postJSONRaw(t, mux, "/api/reviews/answer",
		`{"session_id":"`+sess["session_id"].(string)+`","attempt_id":"`+q["attempt_id"].(string)+`","answer":"2","elapsed":5}`, token)
	if code != 200 {
		t.Fatalf("review answer: %d %s", code, body)
	}
	var res struct {
		Result struct {
			Correct     bool             `json:"correct"`
			Walkthrough *wireWalkthrough `json:"walkthrough"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(body), &res); err != nil {
		t.Fatal(err)
	}
	if res.Result.Correct {
		t.Fatalf("2 graded right: %s", body)
	}
	assertWalkthrough(t, "review", res.Result.Walkthrough, true)
}

// Mastery check and course tests (one handler): the verdict carries the
// walkthrough of the question just graded, and nothing else that teaches.
func TestWalkthrough_QuizVerdict(t *testing.T) {
	mux, _, token, _ := walkthroughServer(t)
	for _, tc := range []struct {
		name, answer string
		wrong        bool
	}{{"wrong", "2", true}, {"right", "4", false}} {
		t.Run(tc.name, func(t *testing.T) {
			sess := quizPost(t, mux, "/api/quiz/session", `{}`, token)
			first, _ := sess["concept_id"].(string)
			code, body := postJSONRaw(t, mux, "/api/quiz/answer",
				`{"session_id":"`+sess["session_id"].(string)+`","concept_id":"`+first+`","answer":"`+tc.answer+`","elapsed":5}`, token)
			if code != 200 {
				t.Fatalf("quiz answer: %d %s", code, body)
			}
			var res struct {
				Done        bool             `json:"done"`
				Correct     bool             `json:"correct"`
				Walkthrough *wireWalkthrough `json:"walkthrough"`
			}
			if err := json.Unmarshal([]byte(body), &res); err != nil {
				t.Fatal(err)
			}
			if res.Correct == tc.wrong {
				t.Fatalf("correct=%v: %s", res.Correct, body)
			}
			assertWalkthrough(t, "quiz "+tc.name, res.Walkthrough, tc.wrong)
			var raw map[string]interface{}
			_ = json.Unmarshal([]byte(body), &raw)
			for _, k := range []string{"explanation", "diagnosis"} {
				if _, ok := raw[k]; ok {
					t.Errorf("the verdict must carry the solution under walkthrough only, found %q: %s", k, body)
				}
			}
		})
	}
}

// Placement: no walkthrough mid-test; the results list every answered
// question with its walkthrough once the run is over.
func TestWalkthrough_PlacementResultsOnly(t *testing.T) {
	mux, _, token, _ := walkthroughServer(t)
	code, start := postJSONRaw(t, mux, "/api/goal/diagnostic", `{"concept_ids":["a","b"]}`, token)
	if code != 200 {
		t.Fatalf("start: %d %s", code, start)
	}
	var st struct {
		SessionID string `json:"session_id"`
		ConceptID string `json:"concept_id"`
	}
	_ = json.Unmarshal([]byte(start), &st)
	// Every answer in order: the first wrong, the rest right (the CAT may
	// ask a concept again).
	answers := []string{"2"}
	code, body := postJSONRaw(t, mux, "/api/goal/diagnostic/answer",
		`{"session_id":"`+st.SessionID+`","concept_id":"`+st.ConceptID+`","answer":"2","elapsed":5}`, token)
	if code != 200 {
		t.Fatalf("answer: %d %s", code, body)
	}
	if strings.Contains(body, "walkthrough") || strings.Contains(body, stepOne) {
		t.Fatalf("a placement verdict must not teach mid-test: %s", body)
	}
	var mid struct {
		Done      bool   `json:"done"`
		ConceptID string `json:"concept_id"`
	}
	_ = json.Unmarshal([]byte(body), &mid)
	for !mid.Done {
		answers = append(answers, "4")
		code, body = postJSONRaw(t, mux, "/api/goal/diagnostic/answer",
			`{"session_id":"`+st.SessionID+`","concept_id":"`+mid.ConceptID+`","answer":"4","elapsed":5}`, token)
		if code != 200 {
			t.Fatalf("answer: %d %s", code, body)
		}
		if strings.Contains(body, "walkthrough") {
			t.Fatalf("a placement verdict must not teach mid-test: %s", body)
		}
		_ = json.Unmarshal([]byte(body), &mid)
	}
	code, body = postJSONRaw(t, mux, "/api/goal/plan", `{"session_id":"`+st.SessionID+`"}`, token)
	if code != 200 {
		t.Fatalf("plan: %d %s", code, body)
	}
	var plan struct {
		Questions []struct {
			ConceptID   string           `json:"concept_id"`
			Correct     bool             `json:"correct"`
			Answer      string           `json:"answer"`
			Walkthrough *wireWalkthrough `json:"walkthrough"`
		} `json:"questions"`
	}
	if err := json.Unmarshal([]byte(body), &plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.Questions) != len(answers) {
		t.Fatalf("results list %d questions, want %d: %s", len(plan.Questions), len(answers), body)
	}
	for i, q := range plan.Questions {
		wrong := answers[i] == "2"
		if q.Correct == wrong || q.Answer != answers[i] {
			t.Errorf("question %d (%s): correct=%v answer=%q", i, q.ConceptID, q.Correct, q.Answer)
		}
		assertWalkthrough(t, "placement "+q.ConceptID, q.Walkthrough, wrong)
	}
}
