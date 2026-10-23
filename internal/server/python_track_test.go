package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/auth"
	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/engine"
	"github.com/chuma-beep/mathua/internal/generator/all"
	"github.com/chuma-beep/mathua/internal/modules"
	"github.com/chuma-beep/mathua/internal/planning"
	"github.com/chuma-beep/mathua/internal/pytrack"
	"github.com/chuma-beep/mathua/internal/storage"
)

// pythonServer is the real corpus (mathematics and Python) behind the real
// handlers, so these tests see exactly what a learner on either track sees.
func pythonServer(t *testing.T) *http.ServeMux {
	t.Helper()
	mods, err := modules.Load("../..")
	if err != nil {
		t.Fatalf("modules: %v", err)
	}
	dag, err := concepts.LoadDirs("../../data/concepts", mods.ConceptDirs()...)
	if err != nil {
		t.Fatalf("corpus: %v", err)
	}
	store, err := storage.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	ll := miniLessons(t, map[string]string{"arith.add.basic": "Adding."})
	eng := engine.New(store, dag, all.Registry(), ll, planning.New(dag))
	eng.SetModules(mods)
	s := New(eng, store, auth.New(store))
	mux := http.NewServeMux()
	s.Register(mux)
	return mux
}

func getBody(t *testing.T, mux *http.ServeMux, token, path string) (int, map[string]interface{}) {
	t.Helper()
	rec := authedGet(t, mux, path, token)
	var res map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	return rec.Code, res
}

func TestPythonTrack_DomainsAreSeparated(t *testing.T) {
	mux := pythonServer(t)
	token, _ := guestTokenAndID(t, mux)

	for _, tc := range []struct {
		query      string
		wantPython bool
	}{{"", false}, {"?track=math", false}, {"?track=python", true}} {
		code, body := getBody(t, mux, token, "/api/curriculum/domains"+tc.query)
		if code != 200 {
			t.Fatalf("domains%s: %d", tc.query, code)
		}
		ds, _ := body["domains"].([]interface{})
		if len(ds) == 0 {
			t.Fatalf("domains%s: none", tc.query)
		}
		for _, d := range ds {
			id, _ := d.(map[string]interface{})["id"].(string)
			if strings.HasPrefix(id, "python_") != tc.wantPython {
				t.Errorf("domains%s lists %q", tc.query, id)
			}
		}
	}
}

func TestPythonTrack_NextStaysInTrack(t *testing.T) {
	mux := pythonServer(t)
	token, _ := guestTokenAndID(t, mux)

	hrefs := func(body map[string]interface{}) []string {
		var out []string
		if p, _ := body["primary"].(map[string]interface{}); p != nil {
			a, _ := p["action"].(map[string]interface{})
			h, _ := a["href"].(string)
			out = append(out, h)
		}
		alts, _ := body["alternatives"].([]interface{})
		for _, x := range alts {
			a, _ := x.(map[string]interface{})["action"].(map[string]interface{})
			h, _ := a["href"].(string)
			out = append(out, h)
		}
		return out
	}

	_, py := getBody(t, mux, token, "/api/next?track=python")
	got := hrefs(py)
	if len(got) == 0 {
		t.Fatalf("python next is empty: %v", py)
	}
	for _, h := range got {
		if !strings.HasPrefix(h, "/learn?concept=py.") {
			t.Errorf("python track recommended %q (the Python track has no placement diagnostic)", h)
		}
	}
	_, math := getBody(t, mux, token, "/api/next")
	for _, h := range hrefs(math) {
		if strings.Contains(h, "concept=py.") {
			t.Errorf("math track recommended Python concept %q", h)
		}
	}
}

func TestPythonTrack_TeachingStepLinksTheBook(t *testing.T) {
	mux := pythonServer(t)
	code, body := getBody(t, mux, "", "/api/lessons/py.np.slicing/kp")
	if code != 200 {
		t.Fatalf("kp: %d %v", code, body)
	}
	kps, _ := body["kps"].([]interface{})
	if len(kps) != 1 {
		t.Fatalf("want one teaching step, got %v", body)
	}
	we, _ := kps[0].(map[string]interface{})["worked_example"].(string)
	if !strings.Contains(we, "https://jakevdp.github.io/PythonDataScienceHandbook/02.02-the-basics-of-numpy-arrays.html") {
		t.Errorf("teaching step does not link its book section:\n%s", we)
	}
}

// The full Learn loop for each question kind: serve, answer, grade, record.
func TestPythonTrack_LearnLoopGradesEveryKind(t *testing.T) {
	tr, err := pytrack.Default()
	if err != nil {
		t.Fatal(err)
	}
	runner := tr.Runner.Available() == nil
	mux := pythonServer(t)
	token, _ := guestTokenAndID(t, mux)

	const cid = "py.np.dtypes" // a root concept: open without prerequisites
	code, body := getBody(t, mux, token, "/api/lessons/"+cid+"/practice?count=20&difficulty=0.6")
	if code != 200 {
		t.Fatalf("practice: %d %v", code, body)
	}
	qs, _ := body["questions"].([]interface{})
	bank := tr.Bank.Concept(cid)
	seen := map[string]bool{}
	for _, raw := range qs {
		q := raw.(map[string]interface{})
		question, _ := q["question"].(string)
		answer, _ := q["answer"].(string)
		if strings.Contains(question, "check(") {
			t.Fatalf("served question contains hidden test code")
		}
		var right, wrong string
		kind := strings.SplitN(answer, ":", 2)[0]
		switch kind {
		case "mc":
			right = strings.TrimPrefix(answer, "mc:")
			wrong = "definitely not an option"
		case "out":
			right = strings.TrimPrefix(answer, "out:")
			wrong = "[]"
		case "code":
			if !runner {
				continue
			}
			key := strings.TrimPrefix(answer, "code:")
			for _, it := range bank.Items {
				if cid+"#"+it.ID == key {
					right = it.Solution
				}
			}
			wrong = "x = 1"
		default:
			t.Fatalf("unexpected served answer %q", answer)
		}
		if seen[kind] {
			continue
		}
		seen[kind] = true
		for _, tc := range []struct {
			answer string
			want   bool
		}{{right, true}, {wrong, false}} {
			payload, _ := json.Marshal(map[string]interface{}{
				"concept_id": cid, "answer": tc.answer, "elapsed": 30.0, "question": question,
			})
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/api/study/answer", strings.NewReader(string(payload)))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			mux.ServeHTTP(rec, req)
			var res map[string]interface{}
			_ = json.Unmarshal(rec.Body.Bytes(), &res)
			if rec.Code != 200 {
				t.Fatalf("%s answer: %d %s", kind, rec.Code, rec.Body.String())
			}
			if ung, _ := res["ungraded"].(bool); ung {
				t.Fatalf("%s answer ungraded: %v", kind, res)
			}
			if got, _ := res["correct"].(bool); got != tc.want {
				t.Errorf("%s: answer %q graded correct=%v, want %v (%v)", kind, tc.answer, got, tc.want, res)
			}
			if kind == "code" && !tc.want {
				if d, _ := res["diagnosis"].(string); d == "" {
					t.Errorf("a failed code task should name the failing check: %v", res)
				}
			}
		}
	}
	for _, k := range []string{"mc", "out"} {
		if !seen[k] {
			t.Errorf("practice for %s served no %q question in 20 draws", cid, k)
		}
	}
	if runner && !seen["code"] {
		t.Errorf("practice for %s served no code task in 20 draws", cid)
	}
}

func TestPythonTrack_RunEndpoint(t *testing.T) {
	tr, _ := pytrack.Default()
	if err := tr.Runner.Available(); err != nil {
		if os.Getenv("PYTRACK_REQUIRE_RUNNER") == "1" {
			t.Fatal(err)
		}
		t.Skip(err)
	}
	mux := pythonServer(t)
	code, res := postJSON(t, mux, "/api/python/run", `{"code":"import numpy as np\nprint(np.arange(4).sum())"}`)
	if code != 200 || res["stdout"] != "6\n" {
		t.Fatalf("run: %d %v", code, res)
	}
	// A task key brings the task's fixture with it, so the variables the prompt
	// promises exist when the learner presses Run.
	code, res = postJSON(t, mux, "/api/python/run", `{"code":"print(raw)","task":"code:py.np.dtypes#c2"}`)
	if code != 200 || !strings.Contains(res["stdout"].(string), "7.9") {
		t.Fatalf("run with fixture: %d %v", code, res)
	}
}

func TestModuleFilesStayInsideFiles(t *testing.T) {
	mux := pythonServer(t)
	for _, p := range []string{
		"/api/modules/python-dsh/files/../module.json",
		"/api/modules/python-dsh/files/%2e%2e/module.json",
		"/api/modules/python-dsh/module.json",
		"/api/modules/nope/files/x.pdf",
	} {
		rec := authedGet(t, mux, p, "")
		if rec.Code == 200 {
			t.Errorf("%s served a file outside files/", p)
		}
	}
}
