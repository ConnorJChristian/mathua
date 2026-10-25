package proof

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/grader"
	"github.com/chuma-beep/mathua/internal/modules"
)

func testProblem() Problem {
	return Problem{
		ID:        "t.sqrt2",
		Concept:   "ra.order.completeness",
		Title:     "sqrt 2 is irrational",
		Level:     1,
		Statement: "Prove that there is no rational number whose square is 2.",
		Reference: "Suppose (p/q)^2 = 2 in lowest terms. Then p is even, so q is even. Contradiction.",
		Milestones: []Milestone{
			{ID: "m1", Points: 2, Desc: "sets up a contradiction with a fraction in lowest terms"},
			{ID: "m2", Points: 3, Desc: "shows p is even"},
			{ID: "m3", Points: 2, Desc: "shows q is even and concludes"},
		},
		Caps: []Cap{{Error: "assumes the conclusion", Max: 1}},
	}
}

// fakeAPI speaks enough of the Messages API for the grader: each call is
// answered by reply(n) with n the 0-based call number.
type fakeAPI struct {
	srv      *httptest.Server
	calls    atomic.Int32
	mu       sync.Mutex
	bodies   []map[string]any
	headers  []http.Header
	reply    func(n int) (status int, body string)
	inFlight atomic.Int32
	maxSeen  atomic.Int32
	hold     time.Duration
}

func newFakeAPI(t *testing.T, reply func(n int) (int, string)) *fakeAPI {
	f := &fakeAPI{reply: reply}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cur := f.inFlight.Add(1)
		defer f.inFlight.Add(-1)
		for {
			m := f.maxSeen.Load()
			if cur <= m || f.maxSeen.CompareAndSwap(m, cur) {
				break
			}
		}
		n := int(f.calls.Add(1)) - 1
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		f.mu.Lock()
		f.bodies = append(f.bodies, body)
		f.headers = append(f.headers, r.Header.Clone())
		f.mu.Unlock()
		if r.URL.Path != "/v1/messages" {
			w.WriteHeader(404)
			return
		}
		if f.hold > 0 {
			time.Sleep(f.hold)
		}
		st, b := f.reply(n)
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(st)
		io.WriteString(w, b)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func message(text string, stop string) string {
	b, _ := json.Marshal(map[string]any{
		"type": "message", "model": "fake", "stop_reason": stop,
		"content": []map[string]any{{"type": "text", "text": text}},
		"usage":   map[string]any{"input_tokens": 100, "output_tokens": 200, "cache_creation_input_tokens": 1000, "cache_read_input_tokens": 0},
	})
	return string(b)
}

func verdictJSON(score int) string {
	met := "yes"
	if score < 6 {
		met = "no"
	}
	return fmt.Sprintf(`{"first_error": "line 2 does not follow", "milestones": [{"id":"m1","met":"yes","note":"ok"},{"id":"m2","met":"%s","note":"x"},{"id":"m3","met":"yes","note":"y"}], "score": %d, "feedback": "score was %d"}`, met, score, score)
}

func scores(ss ...int) func(int) (int, string) {
	return func(n int) (int, string) { return 200, message(verdictJSON(ss[n%len(ss)]), "end_turn") }
}

func newTestGrader(t *testing.T, f *fakeAPI, mod func(*Config)) *Grader {
	t.Helper()
	b, err := NewBank(testProblem())
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{APIKey: "test-key", APIBase: f.srv.URL, Model: "claude-test", Runs: 5, MonthlyUSD: 10, Timeout: 10 * time.Second}
	if mod != nil {
		mod(&cfg)
	}
	return New(cfg, b, NewMemStore())
}

const goodProof = "Suppose p/q in lowest terms squares to 2. Then p^2 = 2q^2 so p is even..."

func TestRequestShapeAndPromptCaching(t *testing.T) {
	f := newFakeAPI(t, scores(7))
	g := newTestGrader(t, f, func(c *Config) { c.Runs = 1 })
	p := g.Bank.Problem("t.sqrt2")
	out := g.GradeProof(context.Background(), p, goodProof)
	if out.Report == nil {
		t.Fatalf("no report: %+v", out)
	}
	h := f.headers[0]
	if h.Get("x-api-key") != "test-key" || h.Get("anthropic-version") != "2023-06-01" || !strings.HasPrefix(h.Get("content-type"), "application/json") {
		t.Fatalf("headers: %v", h)
	}
	b := f.bodies[0]
	if b["model"] != "claude-test" {
		t.Fatalf("model: %v", b["model"])
	}
	sys := b["system"].([]any)
	if len(sys) != 2 {
		t.Fatalf("system blocks: %d", len(sys))
	}
	last := sys[1].(map[string]any)
	cc, ok := last["cache_control"].(map[string]any)
	if !ok || cc["type"] != "ephemeral" {
		t.Fatalf("the rubric block must carry cache_control: %v", last)
	}
	if !strings.Contains(last["text"].(string), "MARKING SCHEME") || !strings.Contains(last["text"].(string), "REFERENCE PROOF") {
		t.Fatal("rubric block lacks the scheme or the reference")
	}
	user := b["messages"].([]any)[0].(map[string]any)
	if !strings.Contains(fmt.Sprint(user["content"]), "p is even") {
		t.Fatal("the student's proof is not in the user message")
	}
	if strings.Contains(fmt.Sprint(sys), "p^2 = 2q^2 so") {
		t.Fatal("the student's proof leaked into the cached system prefix")
	}
}

func TestMedianAndFullyCorrect(t *testing.T) {
	cases := []struct {
		runs   []int
		median int
		full   bool
	}{
		{[]int{7, 7, 6, 7, 5}, 7, false}, // one run below the pass mark: not correct
		{[]int{6, 7, 7, 6, 7}, 7, true},
		{[]int{2, 3, 1, 7, 2}, 2, false},
		{[]int{6, 6, 6, 6, 6}, 6, true},
	}
	for _, c := range cases {
		f := newFakeAPI(t, scores(c.runs...))
		g := newTestGrader(t, f, nil)
		res := g.Grade("proof:t.sqrt2", goodProof)
		if res.Unavailable || res.Proof == nil {
			t.Fatalf("%v: %+v", c.runs, res)
		}
		if res.Proof.Score != c.median || res.Correct != c.full || res.Proof.FullyCorrect != c.full {
			t.Fatalf("%v: got score %d correct %v, want %d %v", c.runs, res.Proof.Score, res.Correct, c.median, c.full)
		}
		if res.Score != float64(c.median)/7 {
			t.Fatalf("partial credit %v", res.Score)
		}
		if len(res.Proof.Runs) != 5 || int(f.calls.Load()) != 5 {
			t.Fatalf("runs %v calls %d", res.Proof.Runs, f.calls.Load())
		}
		// The checklist and feedback come from a run that gave the median.
		if res.Proof.Feedback != fmt.Sprintf("score was %d", c.median) {
			t.Fatalf("feedback from the wrong run: %q", res.Proof.Feedback)
		}
		if len(res.Proof.Milestones) != 3 || res.Proof.Milestones[1].Points != 3 || res.Proof.Milestones[1].Desc == "" {
			t.Fatalf("milestones %+v", res.Proof.Milestones)
		}
	}
}

func TestEvenRunCountTakesLowerMedian(t *testing.T) {
	f := newFakeAPI(t, scores(7, 4, 7, 4))
	g := newTestGrader(t, f, func(c *Config) { c.Runs = 4 })
	res := g.Grade("proof:t.sqrt2", goodProof)
	if res.Proof == nil || res.Proof.Score != 4 {
		t.Fatalf("want the lower median 4, got %+v", res.Proof)
	}
}

func TestRunsAreConcurrent(t *testing.T) {
	f := newFakeAPI(t, scores(7))
	f.hold = 150 * time.Millisecond
	g := newTestGrader(t, f, nil)
	start := time.Now()
	g.Grade("proof:t.sqrt2", goodProof)
	if f.maxSeen.Load() < 5 {
		t.Fatalf("at most %d runs were in flight at once", f.maxSeen.Load())
	}
	if time.Since(start) > 600*time.Millisecond {
		t.Fatalf("runs took %v; they should overlap", time.Since(start))
	}
}

func TestIdenticalProofIsCached(t *testing.T) {
	f := newFakeAPI(t, scores(7))
	g := newTestGrader(t, f, nil)
	first := g.Grade("proof:t.sqrt2", goodProof)
	if first.Proof == nil || first.Proof.Cached {
		t.Fatalf("first grade: %+v", first)
	}
	calls := f.calls.Load()
	again := g.Grade("proof:t.sqrt2", "  "+strings.ReplaceAll(goodProof, "\n", "\r\n")+"  \n")
	if f.calls.Load() != calls {
		t.Fatal("a resubmitted identical proof called the API again")
	}
	if again.Proof == nil || !again.Proof.Cached || again.Proof.Score != 7 {
		t.Fatalf("cached grade: %+v", again.Proof)
	}
	g.Grade("proof:t.sqrt2", goodProof+" Hence the claim.")
	if f.calls.Load() == calls {
		t.Fatal("a different proof must be graded afresh")
	}
}

func TestEditedRubricInvalidatesCache(t *testing.T) {
	f := newFakeAPI(t, scores(7))
	g := newTestGrader(t, f, nil)
	p := g.Bank.Problem("t.sqrt2")
	k1 := g.CacheKey(p, goodProof)
	p.Reference += " Edited."
	if g.CacheKey(p, goodProof) == k1 {
		t.Fatal("cache key ignores the marking scheme")
	}
}

func TestSpendIsRecordedAndBudgetRefuses(t *testing.T) {
	f := newFakeAPI(t, scores(7))
	g := newTestGrader(t, f, func(c *Config) {
		c.Prices = Prices{InputPerMTok: 3, OutputPerMTok: 15, CacheWriteMult: 1.25, CacheReadMult: 0.1}
		c.MaxTokens = 1000
		c.MonthlyUSD = 0.2
	})
	month := time.Now().UTC().Format("2006-01")
	res := g.Grade("proof:t.sqrt2", goodProof)
	if res.Unavailable {
		t.Fatalf("first grade refused: %s", res.Feedback)
	}
	spent, _ := g.Store.ProofSpend(context.Background(), month)
	// 5 runs x (100 in + 1000 cache-write x 1.25) x $3/M + 5 x 200 out x $15/M
	want := 5 * ((100+1250)*3.0 + 200*15.0) / 1e6
	if diff := spent - want; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("spend %v, want %v", spent, want)
	}
	// Push the month to the edge: the next grading would exceed it.
	g.Store.AddProofSpend(context.Background(), month, 0.15, 0, 0)
	calls := f.calls.Load()
	res = g.Grade("proof:t.sqrt2", goodProof+" more")
	if !res.Unavailable || !strings.Contains(res.Feedback, "budget") {
		t.Fatalf("over budget must be ungraded: %+v", res)
	}
	if f.calls.Load() != calls {
		t.Fatal("an over-budget grading called the API")
	}
	if g.Serving() {
		t.Fatal("proofs must not be served once the budget is used up")
	}
}

func TestFailuresAreUnavailableNotMisses(t *testing.T) {
	cases := map[string]func(int) (int, string){
		"http error": func(int) (int, string) {
			return 529, `{"type":"error","error":{"type":"overloaded_error","message":"busy"}}`
		},
		"refusal": func(int) (int, string) { return 200, message(verdictJSON(7), "refusal") },
		"prose":   func(int) (int, string) { return 200, message("This proof looks good to me.", "end_turn") },
		"bad score": func(int) (int, string) {
			return 200, message(strings.Replace(verdictJSON(7), `"score": 7`, `"score": 9`, 1), "end_turn")
		},
		"bad mileset": func(int) (int, string) {
			return 200, message(strings.Replace(verdictJSON(7), `"m3"`, `"m9"`, 1), "end_turn")
		},
		"minority ok": func(n int) (int, string) {
			if n < 2 {
				return 200, message(verdictJSON(7), "end_turn")
			}
			return 500, `{}`
		},
	}
	for name, reply := range cases {
		f := newFakeAPI(t, reply)
		g := newTestGrader(t, f, nil)
		res := g.Grade("proof:t.sqrt2", goodProof)
		if !res.Unavailable || res.Correct || res.Proof != nil {
			t.Fatalf("%s: want ungraded, got %+v", name, res)
		}
		// Nothing is cached, so a retry is graded afresh.
		calls := f.calls.Load()
		g.Grade("proof:t.sqrt2", goodProof)
		if f.calls.Load() == calls {
			t.Fatalf("%s: a failed grading was cached", name)
		}
	}
}

func TestMajorityOfRunsIsEnough(t *testing.T) {
	f := newFakeAPI(t, func(n int) (int, string) {
		if n == 0 {
			return 500, `{}`
		}
		return 200, message(verdictJSON(7), "end_turn")
	})
	g := newTestGrader(t, f, nil)
	res := g.Grade("proof:t.sqrt2", goodProof)
	if res.Unavailable || !res.Correct || len(res.Proof.Runs) != 4 {
		t.Fatalf("%+v", res)
	}
}

func TestNetworkErrorAndNoKey(t *testing.T) {
	b, _ := NewBank(testProblem())
	g := New(Config{APIKey: "k", APIBase: "http://127.0.0.1:1", Runs: 3, MonthlyUSD: 5, Timeout: 2 * time.Second}, b, NewMemStore())
	if res := g.Grade("proof:t.sqrt2", goodProof); !res.Unavailable {
		t.Fatalf("network error must be ungraded: %+v", res)
	}
	g = New(Config{Runs: 3, MonthlyUSD: 5}, b, NewMemStore())
	if res := g.Grade("proof:t.sqrt2", goodProof); !res.Unavailable || !strings.Contains(res.Feedback, "ANTHROPIC_API_KEY") {
		t.Fatalf("no key must be ungraded: %+v", res)
	}
	if g.Enabled() || g.Serving() {
		t.Fatal("a grader without a key must not serve proofs")
	}
	if res := g.Grade("proof:t.nope", goodProof); !res.Unavailable {
		t.Fatal("unknown problem must be ungraded")
	}
	if res := g.Grade("proof:t.sqrt2", "   "); !res.Unavailable {
		t.Fatal("an empty proof must not be graded")
	}
}

func TestRouterDispatchesProofs(t *testing.T) {
	grader.SetProofGrader(nil)
	r := grader.NewRouter()
	if res := r.Grade(grader.GradingMultipleChoice, "proof:x", "some proof"); !res.Unavailable {
		t.Fatalf("no proof grader installed must be ungraded: %+v", res)
	}
	var got string
	grader.SetProofGrader(func(e, a string) grader.Result { got = e + "|" + a; return grader.Result{Correct: true} })
	defer grader.SetProofGrader(nil)
	if res := r.Grade(grader.GradingNumeric, "proof:x", "x = 1 so"); !res.Correct || got != "proof:x|x = 1 so" {
		t.Fatalf("router did not hand the proof over untouched: %q", got)
	}
	if res := r.Grade(grader.GradingBank, "proof:x", "p"); !res.Correct {
		t.Fatal("bank-typed proof answer not routed")
	}
	if !grader.IsBankAnswer("proof:x") {
		t.Fatal("a proof key is a bank answer (served inside generated topics)")
	}
}

func TestParseVerdict(t *testing.T) {
	p := testProblem()
	v, err := ParseVerdict("```json\n"+verdictJSON(5)+"\n```", &p)
	if err != nil || v.Score != 5 || len(v.Milestones) != 3 {
		t.Fatalf("%+v %v", v, err)
	}
	v, err = ParseVerdict(strings.Replace(verdictJSON(7), "line 2 does not follow", "none", 1), &p)
	if err != nil || v.FirstError != "" {
		t.Fatalf("none should clear first_error: %+v %v", v, err)
	}
	for _, bad := range []string{
		`{"score": 3.5, "milestones": []}`,
		`{"milestones": []}`,
		strings.Replace(verdictJSON(7), `"met":"yes"`, `"met":"maybe"`, 1),
		strings.Replace(verdictJSON(7), `{"id":"m3","met":"yes","note":"y"}`, `{"id":"m2","met":"yes","note":"y"}`, 1),
	} {
		if _, err := ParseVerdict(bad, &p); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
}

func TestLoadBankValidates(t *testing.T) {
	dir := t.TempDir()
	p := testProblem()
	write := func(ps ...Problem) {
		raw, _ := json.Marshal(File{Chapter: "c", Title: "t", Problems: ps})
		os.WriteFile(filepath.Join(dir, "a.json"), raw, 0o644)
	}
	write(p)
	b, err := LoadBank(dir)
	if err != nil || b.Len() != 1 || b.ByConcept()["ra.order.completeness"][0] != "t.sqrt2" {
		t.Fatalf("%v %v", b, err)
	}
	bad := p
	bad.Milestones = append([]Milestone(nil), p.Milestones...)
	bad.Milestones[0].Points = 3
	write(bad)
	if _, err := LoadBank(dir); err == nil || !strings.Contains(err.Error(), "sum to 8") {
		t.Fatalf("points must sum to 7: %v", err)
	}
	write(p, p)
	if _, err := LoadBank(dir); err == nil {
		t.Fatal("duplicate id accepted")
	}
	os.WriteFile(filepath.Join(dir, "a.json"), []byte(`{"chapter":"c","title":"t","problems":[],"extra":1}`), 0o644)
	if _, err := LoadBank(dir); err == nil {
		t.Fatal("unknown field accepted")
	}
}

func TestServedQuestion(t *testing.T) {
	b, _ := NewBank(testProblem())
	sp, ok := b.ProblemFor("t.sqrt2")
	if !ok || sp.Answer != "proof:t.sqrt2" || !strings.HasSuffix(sp.Question, Marker) || !sp.Raw {
		t.Fatalf("%+v", sp)
	}
	if strings.Contains(sp.Question, "MARKING") || strings.Contains(sp.Question, "lowest terms. Then p is even") {
		t.Fatal("the rubric or reference reached the question")
	}
	if !strings.Contains(sp.Explanation, "p is even") {
		t.Fatal("the reference proof is the explanation")
	}
}

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "abc")
	t.Setenv("MATHUA_PROOF_MODEL", "claude-opus-5-5")
	t.Setenv("MATHUA_PROOF_RUNS", "3")
	t.Setenv("MATHUA_PROOF_MONTHLY_USD", "12.5")
	t.Setenv("MATHUA_PROOF_COUNTS_FOR_MASTERY", "")
	c := ConfigFromEnv()
	if c.APIKey != "abc" || c.Model != "claude-opus-5-5" || c.Runs != 3 || c.MonthlyUSD != 12.5 || c.CountsForMastery {
		t.Fatalf("%+v", c)
	}
	if c.Prices.OutputPerMTok <= defaultPrices("claude-sonnet-5-5").OutputPerMTok {
		t.Fatal("opus default prices should be higher")
	}
	t.Setenv("MATHUA_PROOF_MODEL", "")
	t.Setenv("MATHUA_PROOF_RUNS", "")
	t.Setenv("MATHUA_PROOF_MONTHLY_USD", "")
	t.Setenv("MATHUA_PROOF_COUNTS_FOR_MASTERY", "1")
	c = ConfigFromEnv()
	if c.Model != DefaultModel || c.Runs != DefaultRuns || c.MonthlyUSD != DefaultMonthlyUSD || !c.CountsForMastery {
		t.Fatalf("defaults: %+v", c)
	}
}

func TestCalibrationStats(t *testing.T) {
	rs := []CalResult{
		{Row: CalRow{Human: 7}, Judge: 7, Pass: true},
		{Row: CalRow{Human: 2}, Judge: 6, Pass: true}, // a false pass, off by 4
		{Row: CalRow{Human: 1}, Judge: 1, Pass: false},
		{Row: CalRow{Human: 6}, Judge: 5, Pass: false}, // a false fail, off by 1
		{Row: CalRow{Human: 3}, Err: "budget"},
	}
	s := Stats(rs)
	if s.N != 4 || s.Ungraded != 1 || s.MAE != 5.0/4 || s.Bias != 3.0/4 || s.Within1 != 0.75 {
		t.Fatalf("%+v", s)
	}
	if s.FalsePassDenom != 2 || s.FalsePass != 0.5 || s.FalseFailDenom != 2 || s.FalseFail != 0.5 {
		t.Fatalf("%+v", s)
	}
}

func TestCalibrateEndToEnd(t *testing.T) {
	f := newFakeAPI(t, scores(7))
	g := newTestGrader(t, f, func(c *Config) { c.Runs = 3 })
	in := "problem_id,proof_text,human_score\n" +
		"t.sqrt2,\"Suppose p/q, in lowest terms...\",7\n" +
		"t.sqrt2,circular,1\n"
	rows, err := ReadCalibrationCSV(strings.NewReader(in))
	if err != nil || len(rows) != 2 || rows[0].Proof != "Suppose p/q, in lowest terms..." {
		t.Fatalf("%v %v", rows, err)
	}
	_, s := Calibrate(context.Background(), g, rows, nil)
	if s.N != 2 || s.FalsePass != 1 || s.Bias != 3 {
		t.Fatalf("a judge that passes everything must show a false pass: %+v", s)
	}
	if _, err := ReadCalibrationCSV(strings.NewReader("problem_id,proof_text,human_score\nx,y,9\n")); err == nil {
		t.Fatal("score 9 accepted")
	}
	var buf strings.Builder
	p := testProblem()
	p.Samples = []Sample{{Kind: "correct", Proof: "a, \"quoted\"\nline", Score: 7}}
	b, _ := NewBank(p)
	WriteCalibrationCSV(&buf, b.SampleRows())
	back, err := ReadCalibrationCSV(strings.NewReader(buf.String()))
	if err != nil || len(back) != 1 || back[0].Proof != "a, \"quoted\"\nline" || back[0].Label != "correct" {
		t.Fatalf("round trip: %v %v", back, err)
	}
}

// The committed modules carry no proof problems: marking schemes follow a
// textbook, so they live only in a git-ignored modules-local/ module.
func TestPublicModulesShipNoProofs(t *testing.T) {
	mods, err := modules.LoadPublic("../..")
	if err != nil {
		t.Fatal(err)
	}
	if len(mods.Modules) == 0 {
		t.Fatal("no public modules found; the check would pass vacuously")
	}
	if dirs := mods.ProofDirs(); len(dirs) != 0 {
		t.Fatalf("committed modules contain proofs/: %v", dirs)
	}
}
