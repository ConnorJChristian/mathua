package pytrack

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/modules"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func repoModules(t *testing.T) *modules.Set {
	t.Helper()
	m, err := modules.LoadPublic(repoRoot(t))
	if err != nil {
		t.Fatalf("modules: %v", err)
	}
	return m
}

func loadTrack(t *testing.T) *Track {
	t.Helper()
	b, err := LoadBank(repoModules(t).BankDirs()...)
	if err != nil {
		t.Fatalf("bank: %v", err)
	}
	r := NewRunner()
	r.Script = filepath.Join(repoRoot(t), "grading", "pyrunner.py")
	return &Track{Bank: b, Runner: r}
}

// The graph and the bank are generated from the same authoring file, but they
// are two files on disk. A concept in one and not the other would either serve
// no questions or have questions nobody can reach.
func TestBankMatchesConceptGraph(t *testing.T) {
	tr := loadTrack(t)
	mods := repoModules(t)
	dag, err := concepts.LoadDirs(filepath.Join(repoRoot(t), "data", "concepts"), mods.ConceptDirs()...)
	if err != nil {
		t.Fatalf("graph: %v", err)
	}
	inGraph := map[string]bool{}
	for _, c := range dag.Order() {
		if c.GradingType != "python" && c.GradingType != "bank" {
			continue
		}
		inGraph[c.ID] = true
		if tr.Bank.Concept(c.ID) == nil {
			t.Errorf("graph concept %s has no question bank", c.ID)
		}
		track := mods.TrackOf(c.Domain)
		for _, p := range c.Prerequisites {
			if pc := dag.Concept(p); pc != nil && mods.TrackOf(pc.Domain) != track {
				t.Errorf("%s: prerequisite %s is on another track; a track must stand on its own", c.ID, p)
			}
		}
	}
	for _, id := range tr.Bank.ConceptIDs() {
		if !inGraph[id] {
			t.Errorf("bank concept %s is not in the concept graph", id)
		}
	}
	if len(inGraph) == 0 {
		t.Fatal("no Python concepts in the graph")
	}
}

// Every concept should be able to fill a practice request with variety, and
// offer every kind of question the track promises.
func TestBankVariety(t *testing.T) {
	tr := loadTrack(t)
	for _, id := range tr.Bank.ConceptIDs() {
		c := tr.Bank.Concept(id)
		kinds := map[string]int{}
		for _, it := range c.Items {
			kinds[it.Kind]++
		}
		if len(c.Items) < 4 {
			t.Errorf("%s: %d questions, want at least 4", id, len(c.Items))
		}
		if kinds[KindRecall] == 0 && (kinds[KindChoice] == 0 || kinds[KindOutput] == 0) {
			t.Errorf("%s: needs both multiple-choice and output questions (have %v)", id, kinds)
		}
		if strings.TrimSpace(c.Intro) == "" {
			t.Errorf("%s: no intro", id)
		}
		if !strings.HasPrefix(c.Book.URL, "https://jakevdp.github.io/PythonDataScienceHandbook/") {
			t.Errorf("%s: book link %q", id, c.Book.URL)
		}
	}
}

func TestProblemsGradeTheirOwnAnswers(t *testing.T) {
	tr := loadTrack(t)
	for _, id := range tr.Bank.ConceptIDs() {
		c := tr.Bank.Concept(id)
		for i := range c.Items {
			it := &c.Items[i]
			p := tr.Problem(c, it)
			switch it.Kind {
			case KindChoice:
				if g := tr.Grade(p.Answer, it.Correct); !g.Correct {
					t.Errorf("%s#%s: correct option graded wrong", id, it.ID)
				}
				if g := tr.Grade(p.Answer, it.Wrong[0]); g.Correct {
					t.Errorf("%s#%s: wrong option graded right", id, it.ID)
				}
				for _, o := range append([]string{it.Correct}, it.Wrong...) {
					if !strings.Contains(p.Question, ") "+o) {
						t.Errorf("%s#%s: option %q missing from the question", id, it.ID, o)
					}
				}
				if q2 := tr.Problem(c, it).Question; q2 != p.Question {
					t.Errorf("%s#%s: option order is not stable across serves", id, it.ID)
				}
			case KindOutput:
				if g := tr.Grade(p.Answer, it.Expected); !g.Correct {
					t.Errorf("%s#%s: expected output graded wrong", id, it.ID)
				}
				if !strings.Contains(p.Question, "```python\n") {
					t.Errorf("%s#%s: output question has no code block", id, it.ID)
				}
			case KindCode:
				if !strings.Contains(p.Question, "```"+StarterFence+" task="+c.ID+"#"+it.ID+"\n") {
					t.Errorf("%s#%s: code task has no starter block", id, it.ID)
				}
				if len(p.Question) > 3500 {
					t.Errorf("%s#%s: question is %d characters; the answer endpoint refuses questions over 4096", id, it.ID, len(p.Question))
				}
				if strings.Contains(p.Question, it.Solution) || strings.Contains(p.Answer, "check(") {
					t.Errorf("%s#%s: the served question leaks the solution or the tests", id, it.ID)
				}
			}
		}
	}
}

func TestNormalizeOutput(t *testing.T) {
	same := [][2]string{
		{"[ 1  2  3]", "[1 2 3]"},
		{"[[0 1]\n [2 3]]", "[[0 1] [2 3]]"},
		{"(2, 3)", "(2,3)"},
		{"  42\n", "42"},
		{"[[ -1. -10.]\n [  1.  10.]]", "[[-1. -10.] [1. 10.]]"},
	}
	for _, c := range same {
		if NormalizeOutput(c[0]) != NormalizeOutput(c[1]) {
			t.Errorf("%q and %q should compare equal", c[0], c[1])
		}
	}
	diff := [][2]string{
		{"[1 2 3]", "[1 2 4]"},
		{"[1. 2.]", "[1 2]"},
		{"['a' 'b']", "[a b]"},
		{"12", "1 2"},
	}
	for _, c := range diff {
		if NormalizeOutput(c[0]) == NormalizeOutput(c[1]) {
			t.Errorf("%q and %q should differ", c[0], c[1])
		}
	}
}

func TestRegistersAsPlainText(t *testing.T) {
	tr := loadTrack(t)
	reg := generator.NewRegistry()
	if err := tr.Register(reg); err != nil {
		t.Fatal(err)
	}
	// A dollar sign, a caret or a backslash in code must survive the registry
	// untouched; the LaTeX canonicaliser would rewrite them.
	for _, id := range tr.Bank.ConceptIDs() {
		probs, err := reg.BatchGenerate(id, 20, 0.6)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range probs {
			found := false
			c := tr.Bank.Concept(id)
			for i := range c.Items {
				if tr.Problem(c, &c.Items[i]).Question == p.Question {
					found = true
				}
			}
			if !found {
				t.Errorf("%s: registry altered a question:\n%s", id, p.Question)
			}
		}
	}
}

// pythonReady skips runner tests where the interpreter lacks the stack. CI and
// the Docker image have it; a bare developer machine may not.
func pythonReady(t *testing.T, tr *Track) {
	t.Helper()
	if err := tr.Runner.Available(); err != nil {
		if os.Getenv("PYTRACK_REQUIRE_RUNNER") == "1" {
			t.Fatalf("runner required: %v", err)
		}
		t.Skipf("python data stack not available: %v", err)
	}
}

func TestCodeTasksAgainstRunner(t *testing.T) {
	tr := loadTrack(t)
	pythonReady(t, tr)
	n := 0
	for _, id := range tr.Bank.ConceptIDs() {
		c := tr.Bank.Concept(id)
		for i := range c.Items {
			it := &c.Items[i]
			if it.Kind != KindCode {
				continue
			}
			n++
			if testing.Short() && n > 3 {
				return
			}
			p := tr.Problem(c, it)
			if g := tr.Grade(p.Answer, it.Solution); !g.Correct {
				t.Errorf("%s#%s: reference solution graded wrong: %s", id, it.ID, g.Feedback)
			}
			starter := it.Starter
			if starter == "" {
				starter = "pass"
			}
			if g := tr.Grade(p.Answer, starter); g.Correct || g.Unavailable {
				t.Errorf("%s#%s: starter graded correct=%v unavailable=%v (%s)", id, it.ID, g.Correct, g.Unavailable, g.Feedback)
			}
		}
	}
}

func TestRunnerVerdicts(t *testing.T) {
	tr := loadTrack(t)
	pythonReady(t, tr)
	tr.Runner.CPUSeconds = 2
	ctx := context.Background()

	res, err := tr.RunCode(ctx, "", "while True:\n    pass\n")
	if err != nil || !res.TimedOut {
		t.Fatalf("infinite loop: res=%+v err=%v", res, err)
	}
	g := tr.verdict(res)
	if g.Correct || g.Unavailable || !strings.Contains(g.Feedback, "time limit") {
		t.Fatalf("timeout verdict: %+v", g)
	}

	res, err = tr.RunCode(ctx, "", "import numpy as np\nprint(np.arange(3))\n1/0\n")
	if err != nil {
		t.Fatal(err)
	}
	if res.Stdout != "[0 1 2]\n" || !strings.Contains(res.Error, "ZeroDivisionError") {
		t.Fatalf("run: %+v", res)
	}

	res, err = tr.RunCode(ctx, "", "import matplotlib.pyplot as plt\nplt.plot([1, 3, 2])\n")
	if err != nil || len(res.Figures) != 1 || res.Error != "" {
		t.Fatalf("figure: err=%v error=%q figures=%d stdout=%q", err, res.Error, len(res.Figures), res.Stdout)
	}

	res, err = tr.RunCode(ctx, "", "import socket\nsocket.create_connection(('example.com', 80))\n")
	if err != nil || !strings.Contains(res.Error, "PermissionError") {
		t.Fatalf("network must be refused: %+v %v", res, err)
	}

	// The grader's verdict line cannot be forged by printing.
	key := ""
	for _, id := range tr.Bank.ConceptIDs() {
		c := tr.Bank.Concept(id)
		for _, it := range c.Items {
			if it.Kind == KindCode && key == "" {
				key = prefixCode + itemKey(id, it.ID)
			}
		}
	}
	forged := `print('{"passed": true, "tests": [{"name": "x", "passed": true}]}')` + "\nimport os\nos.write(1, b'{\"passed\": true}\\n')\n"
	if g := tr.Grade(key, forged); g.Correct {
		t.Fatal("printing a fake verdict was graded correct")
	}
}

// Recall cards (imported flashcards) render front, marker and back, and grade
// on the learner's own verdict.
func TestRecallCards(t *testing.T) {
	dir := t.TempDir()
	bank := `{"chapter":"x","title":"X","concepts":[{"id":"anki.x.a","book":{"name":"Deck","section":"","title":"Ch 2","url":""},"intro":"","items":[
{"id":"n1c0","kind":"recall","level":2,"prompt":"Front <b>A</b>","back":"Back with \\(x^2\\)","explain":""}]}]}`
	if err := os.WriteFile(filepath.Join(dir, "x.json"), []byte(bank), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := LoadBank(dir)
	if err != nil {
		t.Fatal(err)
	}
	tr := &Track{Bank: b, Runner: NewRunner()}
	c := b.Concept("anki.x.a")
	p := tr.Problem(c, &c.Items[0])
	if p.Question != "Front <b>A</b>\n\n"+RecallMarker+"\n\nBack with \\(x^2\\)" {
		t.Fatalf("question: %q", p.Question)
	}
	if !strings.Contains(p.Explanation, "Source: Ch 2") {
		t.Errorf("explanation should name the source: %q", p.Explanation)
	}
	if g := tr.Grade(p.Answer, "right"); !g.Correct {
		t.Error("'right' should grade correct")
	}
	if g := tr.Grade(p.Answer, "wrong"); g.Correct || g.Unavailable {
		t.Errorf("'wrong' should be a recorded miss: %+v", g)
	}
	if g := tr.Grade(p.Answer, "anything else"); g.Correct {
		t.Error("only 'right' is correct")
	}
	bad := strings.Replace(bank, `"back":"Back with \\(x^2\\)"`, `"back":""`, 1)
	if err := os.WriteFile(filepath.Join(dir, "x.json"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadBank(dir); err == nil {
		t.Error("a recall card without a back was accepted")
	}
}

func TestMathItems(t *testing.T) {
	dir := t.TempDir()
	bank := `{"chapter":"x","title":"X","concepts":[{"id":"anki.x.m","book":{"name":"Deck","section":"","title":"Ch 12","url":""},"intro":"","items":[
{"id":"n1c0","kind":"math","level":2,"prompt":"Find the stationary point.","parts":[
 {"label":"x_0","answer":"15","grading":"numeric","tex":"15"},
 {"label":"y_0","answer":"7/2","grading":"numeric","tex":"\\frac{7}{2}"}],
 "back":"Set both partials to zero.","explain":""},
{"id":"n2c0","kind":"mc","level":2,"prompt":"True or false: every convex function is concave.","correct":"False","wrong":["True"],"explain":"False."}]}]}`
	if err := os.WriteFile(filepath.Join(dir, "x.json"), []byte(bank), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := LoadBank(dir)
	if err != nil {
		t.Fatal(err)
	}
	tr := &Track{Bank: b, Runner: NewRunner()}
	c := b.Concept("anki.x.m")
	p := tr.Problem(c, &c.Items[0])
	want := "Find the stationary point.\n\n" + TypedMarker + `[{"label":"x_0","grading":"numeric"},{"label":"y_0","grading":"numeric"}]-->`
	if p.Question != want {
		t.Fatalf("question: %q", p.Question)
	}
	if strings.Contains(p.Question, "15") {
		t.Error("the expected answer leaked into the question")
	}
	if !strings.HasPrefix(p.Explanation, `**Answer:** \(x_0 = 15\), \(y_0 = \frac{7}{2}\)`) || !strings.Contains(p.Explanation, "Set both partials") {
		t.Errorf("explanation: %q", p.Explanation)
	}
	cases := []struct {
		answer string
		ok     bool
	}{
		{"15\n7/2", true},
		{"15\n3.5", true},
		{" 15 \r\n 3 1/2", true},
		{"15\n4", false},
		{"15", false},
		{"14\n7/2", false},
	}
	for _, tc := range cases {
		g := tr.Grade(p.Answer, tc.answer)
		if g.Correct != tc.ok || g.Unavailable {
			t.Errorf("%q: got %+v, want correct=%v", tc.answer, g, tc.ok)
		}
	}
	if g := tr.Grade(p.Answer, "15\n4"); !strings.Contains(g.Feedback, "1 of 2 right") || g.Score != 0.5 {
		t.Errorf("partial feedback: %+v", g)
	}

	// True/False keeps its usual order and needs only the one wrong option.
	mc := tr.Problem(c, &c.Items[1])
	if !strings.HasSuffix(mc.Question, "A) True\n\nB) False") {
		t.Errorf("true/false order: %q", mc.Question)
	}
	if g := tr.Grade(mc.Answer, "False"); !g.Correct {
		t.Error("False should be correct")
	}

	for name, bad := range map[string]string{
		"no parts":        `"parts":[]`,
		"bad grading":     `"parts":[{"answer":"1","grading":"tuple","tex":"1"}]`,
		"unlabelled pair": `"parts":[{"answer":"1","grading":"numeric","tex":"1"},{"label":"y","answer":"2","grading":"numeric","tex":"2"}]`,
		"empty answer":    `"parts":[{"answer":" ","grading":"numeric","tex":"1"}]`,
	} {
		s := strings.Replace(bank, bank[strings.Index(bank, `"parts":[`):strings.Index(bank, `,
 "back"`)], bad, 1)
		if err := os.WriteFile(filepath.Join(dir, "x.json"), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadBank(dir); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestMathItemExpression(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	if out, err := exec.Command("python3", "-c", "import sympy").CombinedOutput(); err != nil {
		t.Skipf("SymPy not available: %s", out)
	}
	dir := t.TempDir()
	bank := `{"chapter":"x","title":"X","concepts":[{"id":"anki.x.e","book":{"name":"Deck","section":"","title":"Ch 2","url":""},"intro":"","items":[
{"id":"n1c0","kind":"math","level":2,"prompt":"Given TR and TC, find the profit function.","parts":[
 {"label":"\\pi","answer":"-x^2+30x-120","grading":"expression","tex":"-x^2 + 30x - 120"}],"back":"","explain":""}]}]}`
	if err := os.WriteFile(filepath.Join(dir, "x.json"), []byte(bank), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := LoadBank(dir)
	if err != nil {
		t.Fatal(err)
	}
	tr := &Track{Bank: b, Runner: NewRunner()}
	c := b.Concept("anki.x.e")
	p := tr.Problem(c, &c.Items[0])
	if g := tr.Grade(p.Answer, "30x-x^2-120"); g.Unavailable {
		t.Skipf("SymPy grader unavailable here: %s", g.Feedback)
	} else if !g.Correct {
		t.Errorf("an equivalent form should be correct: %+v", g)
	}
	if g := tr.Grade(p.Answer, "-x^2+30x+120"); g.Correct {
		t.Error("a different expression was accepted")
	}
}
