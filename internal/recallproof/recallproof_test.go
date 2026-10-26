package recallproof

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/proof"
	"github.com/chuma-beep/mathua/internal/pytrack"
)

// fakeJudge passes an answer containing "CORRECT" with 7 and fails anything
// else with 2. Every run bills 1,000 input and 400 output tokens.
type fakeJudge struct {
	calls atomic.Int32
	fail  error
}

var judgeUsage = proof.Usage{InputTokens: 1000, OutputTokens: 400}

func (j *fakeJudge) Judge(_ context.Context, p *proof.Problem, answer string) (proof.Verdict, proof.Usage, error) {
	j.calls.Add(1)
	if j.fail != nil {
		return proof.Verdict{}, judgeUsage, j.fail
	}
	v := proof.Verdict{Score: 2, FirstError: "the key step is asserted", Feedback: "Justify the key step."}
	met := "no"
	if strings.Contains(answer, "CORRECT") {
		v = proof.Verdict{Score: 7, Feedback: "Complete."}
		met = "yes"
	}
	for _, m := range p.Milestones {
		v.Milestones = append(v.Milestones, proof.MilestoneVerdict{ID: m.ID, Met: met})
	}
	return v, judgeUsage, nil
}

// fakeWriter writes a two-milestone scheme, recording the cards it was asked
// for. Every scheme bills 2,000 input and 1,000 output tokens.
type fakeWriter struct {
	mu    sync.Mutex
	cards []string
	err   error
	hold  time.Duration
}

var writerUsage = proof.Usage{InputTokens: 2000, OutputTokens: 1000}

func (w *fakeWriter) WriteScheme(ctx context.Context, c pytrack.RecallCard) (*Scheme, proof.Usage, error) {
	w.mu.Lock()
	w.cards = append(w.cards, c.Key+"|"+c.Back)
	err := w.err
	w.mu.Unlock()
	if w.hold > 0 {
		time.Sleep(w.hold)
	}
	if err != nil {
		return nil, writerUsage, err
	}
	return &Scheme{
		Reference:  "Full argument: " + c.Back,
		Milestones: []proof.Milestone{{ID: "m1", Points: 4, Desc: "sets up"}, {ID: "m2", Points: 3, Desc: "concludes"}},
		Caps:       []proof.Cap{{Error: "assumes the conclusion", Max: 1}},
	}, writerUsage, nil
}

func (w *fakeWriter) calls() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.cards)
}

func newProof(t *testing.T, store proof.Store, key string) (*proof.Grader, *fakeJudge) {
	t.Helper()
	p := proof.New(proof.Config{APIKey: key, Runs: 3, MonthlyUSD: 60, Timeout: 10 * time.Second}, nil, store)
	j := &fakeJudge{}
	if key != "" {
		p.Judge = j
	}
	return p, j
}

func card() pytrack.RecallCard {
	return pytrack.RecallCard{
		Key: "anki.x.a#w", Concept: "anki.x.a", Bank: "anki_linear_algebra", Level: 2,
		Prompt: "Prove that the sum of two odd numbers is even.",
		Back:   "(2a+1)+(2b+1) = 2(a+b+1).",
	}
}

func on() Config { return Config{On: true, RetryAfter: time.Hour} }

func TestSchemeWrittenOnceThenReused(t *testing.T) {
	store := proof.NewMemStore()
	p, j := newProof(t, store, "k")
	w := &fakeWriter{}
	g := New(p, on(), w)
	c := card()
	if !g.ServeWritten(c) {
		t.Fatal("a card with a key and budget should be served for a written answer")
	}
	r := g.GradeWritten(c, "CORRECT: write them as 2a+1 and 2b+1 ...")
	if r.Unavailable || !r.Correct || r.Proof == nil || r.Proof.Score != 7 {
		t.Fatalf("grade: %+v", r)
	}
	if r.Proof.CountsForMastery {
		t.Error("a flashcard's grade is never reported as mastery evidence")
	}
	r = g.GradeWritten(c, "it is obvious")
	if r.Unavailable || r.Correct || r.Diagnosis == "" || !strings.HasPrefix(r.Feedback, "Score 2/7.") {
		t.Fatalf("a flawed answer should be graded wrong with its first error: %+v", r)
	}
	if w.calls() != 1 {
		t.Fatalf("the scheme should be written once, then reused: %d writes", w.calls())
	}
	if j.calls.Load() != 6 {
		t.Errorf("each grading runs the judge 3 times: %d", j.calls.Load())
	}
	// Stored under the card's key and a hash of its front and back.
	raw, ok, _ := store.ProofGrade(context.Background(), "recall-scheme:v1:anki.x.a#w:"+CardHash(c))
	if !ok || !strings.Contains(raw, `"m1"`) {
		t.Fatalf("scheme not stored: %q", raw)
	}
	// A restart: a new grader on the same store reuses it.
	p2, _ := newProof(t, store, "k")
	g2 := New(p2, on(), w)
	if r := g2.GradeWritten(c, "CORRECT again"); !r.Correct || w.calls() != 1 {
		t.Fatalf("a stored scheme should survive a restart: %+v, %d writes", r, w.calls())
	}
	// Editing the card's back (or front) gets it a new scheme.
	c.Back = "Their sum is 2(a+b+1), which is even."
	if r := g2.GradeWritten(c, "CORRECT"); !r.Correct || w.calls() != 2 {
		t.Fatalf("an edited card should get a new scheme: %d writes", w.calls())
	}
	c2 := card()
	c2.Prompt += " Use the definition."
	g2.GradeWritten(c2, "CORRECT")
	if w.calls() != 3 {
		t.Fatalf("an edited front should get a new scheme: %d writes", w.calls())
	}
}

func TestConcurrentGradingsShareOneScheme(t *testing.T) {
	p, _ := newProof(t, proof.NewMemStore(), "k")
	w := &fakeWriter{hold: 50 * time.Millisecond}
	g := New(p, on(), w)
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			g.GradeWritten(card(), fmt.Sprintf("CORRECT %d", i))
		}(i)
	}
	wg.Wait()
	if w.calls() != 1 {
		t.Errorf("five gradings at once should write one scheme: %d", w.calls())
	}
}

func TestSchemeFailureFallsBackToSelfGrading(t *testing.T) {
	store := proof.NewMemStore()
	p, j := newProof(t, store, "k")
	w := &fakeWriter{err: errors.New("HTTP 529: overloaded")}
	g := New(p, on(), w)
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	g.now = func() time.Time { return now }
	c := card()
	r := g.GradeWritten(c, "CORRECT")
	if !r.Unavailable || r.Correct || !strings.Contains(r.Feedback, "Just show me the answer") {
		t.Fatalf("a failed scheme leaves the answer ungraded, pointing to self-grading: %+v", r)
	}
	if j.calls.Load() != 0 {
		t.Error("no scheme, no judge")
	}
	if g.ServeWritten(c) {
		t.Error("after a failed scheme the card is served as plain recall")
	}
	// A passing failure is retried after RetryAfter.
	now = now.Add(2 * time.Hour)
	w.err = nil
	if !g.ServeWritten(c) {
		t.Fatal("a passing failure should be retried after RetryAfter")
	}
	if r := g.GradeWritten(c, "CORRECT"); !r.Correct {
		t.Fatalf("retry: %+v", r)
	}

	// A card the model judges ungradable stays self-graded until edited,
	// across restarts.
	c2 := card()
	c2.Key = "anki.x.a#v"
	w.err = fmt.Errorf("%w: it needs the figure", ErrNotGradable)
	g.GradeWritten(c2, "CORRECT")
	w.err = nil
	now = now.Add(1000 * time.Hour)
	p2, _ := newProof(t, store, "k")
	g2 := New(p2, on(), w)
	g2.now = g.now
	if g2.ServeWritten(c2) {
		t.Error("an ungradable card should stay self-graded")
	}
	if why, perm := g2.Failure(context.Background(), c2); !perm || !strings.Contains(why, "figure") {
		t.Errorf("failure: %q %v", why, perm)
	}
	c2.Back += " (edited)"
	if !g2.ServeWritten(c2) {
		t.Error("an edited card gets another chance")
	}

	// A scheme that does not add up is a failure too, never a broken rubric.
	bad := &badWriter{}
	g3 := New(p, on(), bad)
	c3 := card()
	c3.Key = "anki.x.a#u"
	if r := g3.GradeWritten(c3, "CORRECT"); !r.Unavailable {
		t.Errorf("an invalid scheme must not be used: %+v", r)
	}
}

type badWriter struct{}

func (badWriter) WriteScheme(context.Context, pytrack.RecallCard) (*Scheme, proof.Usage, error) {
	return &Scheme{Reference: "R", Milestones: []proof.Milestone{{ID: "m1", Points: 5, Desc: "d"}}}, proof.Usage{}, nil
}

func TestNoKeyMeansPlainRecall(t *testing.T) {
	p, _ := newProof(t, proof.NewMemStore(), "")
	w := &fakeWriter{}
	g := New(p, on(), w)
	if g.On() || g.ServeWritten(card()) {
		t.Fatal("without an API key nothing is served for a written answer")
	}
	if r := g.GradeWritten(card(), "CORRECT"); !r.Unavailable {
		t.Errorf("without a key a written answer is ungraded: %+v", r)
	}
	if w.calls() != 0 {
		t.Error("without a key no scheme is written")
	}
	// MATHUA_LLM_RECALL=0 turns it off with a key.
	p2, _ := newProof(t, proof.NewMemStore(), "k")
	if New(p2, Config{On: false}, w).ServeWritten(card()) {
		t.Error("MATHUA_LLM_RECALL=0 should serve plain recall")
	}

	// Installed in the track: the card renders as an ordinary flashcard.
	dir := t.TempDir()
	bank := `{"chapter":"x","title":"X","concepts":[{"id":"anki.x.a","book":{"name":"D","section":"","title":"Ch 1","url":""},"intro":"","items":[
{"id":"w","kind":"recall","level":2,"prompt":"Prove that the sum of two odd numbers is even.","back":"(2a+1)+(2b+1) = 2(a+b+1)."}]}]}`
	if err := os.WriteFile(filepath.Join(dir, "anki_x.json"), []byte(bank), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := pytrack.LoadBank(dir)
	if err != nil {
		t.Fatal(err)
	}
	tr := &pytrack.Track{Bank: b}
	cpt := b.Concept("anki.x.a")
	pytrack.SetWrittenGrader(g)
	defer pytrack.SetWrittenGrader(nil)
	if q := tr.Problem(cpt, &cpt.Items[0]).Question; strings.Contains(q, pytrack.WrittenMarker) {
		t.Errorf("no key: the card should be plain recall: %q", q)
	}
	g2 := New(p2, on(), w)
	pytrack.SetWrittenGrader(g2)
	if q := tr.Problem(cpt, &cpt.Items[0]).Question; !strings.Contains(q, pytrack.WrittenMarker) {
		t.Errorf("with a key the card should take a written answer: %q", q)
	}
}

func TestSpendIsCounted(t *testing.T) {
	store := proof.NewMemStore()
	p, _ := newProof(t, store, "k")
	g := New(p, on(), &fakeWriter{})
	month := time.Now().UTC().Format("2006-01")
	g.GradeWritten(card(), "CORRECT")
	want := p.Cfg.Prices.Cost(writerUsage) + 3*p.Cfg.Prices.Cost(judgeUsage)
	got, _ := store.ProofSpend(context.Background(), month)
	if got < want-1e-9 || got > want+1e-9 {
		t.Fatalf("spend = %.6f, want scheme + 3 runs = %.6f", got, want)
	}

	// A spent budget: nothing served for a written answer, nothing written.
	store2 := proof.NewMemStore()
	store2.AddProofSpend(context.Background(), month, 59.99, 0, 0)
	p2, j2 := newProof(t, store2, "k")
	w2 := &fakeWriter{}
	g2 := New(p2, on(), w2)
	if g2.ServeWritten(card()) {
		t.Error("a spent budget serves plain recall")
	}
	if r := g2.GradeWritten(card(), "CORRECT"); !r.Unavailable || !strings.Contains(r.Feedback, "budget") {
		t.Errorf("a spent budget leaves the answer ungraded: %+v", r)
	}
	if w2.calls() != 0 || j2.calls.Load() != 0 {
		t.Error("a spent budget calls nothing")
	}
}

func TestJudgeFailureIsUngradedNotAMiss(t *testing.T) {
	p, j := newProof(t, proof.NewMemStore(), "k")
	j.fail = errors.New("HTTP 500")
	g := New(p, on(), &fakeWriter{})
	if r := g.GradeWritten(card(), "CORRECT"); !r.Unavailable || r.Correct {
		t.Errorf("a failed judge leaves the answer ungraded: %+v", r)
	}
	if r := g.GradeWritten(card(), "   "); !r.Unavailable {
		t.Errorf("an empty answer is ungraded: %+v", r)
	}
}

func TestBankAllowlistAndExclusions(t *testing.T) {
	t.Setenv("MATHUA_LLM_RECALL", "")
	t.Setenv("MATHUA_LLM_RECALL_BANKS", "")
	t.Setenv("MATHUA_LLM_RECALL_EXCLUDE_BANKS", "linear_algebra, anki_real_analysis.json")
	c := ConfigFromEnv(true)
	if !c.On {
		t.Error("on by default with a key")
	}
	if ConfigFromEnv(false).On {
		t.Error("off without a key")
	}
	for bank, want := range map[string]bool{"anki_linear_algebra": false, "anki_real_analysis": false, "anki_calculus": true} {
		if c.BankAllowed(bank) != want {
			t.Errorf("BankAllowed(%s) = %v", bank, !want)
		}
	}
	t.Setenv("MATHUA_LLM_RECALL_BANKS", "calculus,anki_econometrics")
	t.Setenv("MATHUA_LLM_RECALL_EXCLUDE_BANKS", "")
	c = ConfigFromEnv(true)
	for bank, want := range map[string]bool{"anki_calculus": true, "anki_econometrics": true, "anki_linear_algebra": false} {
		if c.BankAllowed(bank) != want {
			t.Errorf("allowlist: BankAllowed(%s) = %v", bank, !want)
		}
	}
	t.Setenv("MATHUA_LLM_RECALL", "0")
	if ConfigFromEnv(true).On {
		t.Error("MATHUA_LLM_RECALL=0 turns it off")
	}

	// An excluded bank's cards are neither served nor graded.
	p, _ := newProof(t, proof.NewMemStore(), "k")
	w := &fakeWriter{}
	g := New(p, Config{On: true, Exclude: []string{"linear_algebra"}}, w)
	if g.ServeWritten(card()) || !g.GradeWritten(card(), "CORRECT").Unavailable || w.calls() != 0 {
		t.Error("an excluded bank should stay self-graded")
	}
}

func TestParseScheme(t *testing.T) {
	c := card()
	good := "```json\n" + `{"gradable": true, "note": "", "reference": "R", "given": "", "milestones": [{"id": "m1", "points": 3, "desc": "a"}, {"id": "m2", "points": 4, "desc": "b"}], "caps": [{"error": "circular", "max": 1}], "alternatives": []}` + "\n```"
	s, err := ParseScheme(good, c)
	if err != nil || len(s.Milestones) != 2 || s.Caps[0].Max != 1 {
		t.Fatalf("good scheme: %v %+v", err, s)
	}
	if _, err := ParseScheme(`{"gradable": false, "note": "needs the figure"}`, c); !errors.Is(err, ErrNotGradable) {
		t.Errorf("ungradable: %v", err)
	}
	for _, bad := range []string{
		`no json here`,
		`{"gradable": true, "reference": "R", "milestones": [{"id": "m1", "points": 6, "desc": "a"}]}`,
		`{"gradable": true, "reference": "", "milestones": [{"id": "m1", "points": 7, "desc": "a"}]}`,
		`{"gradable": true, "reference": "R", "milestones": [{"id": "m1", "points": 7, "desc": "a"}], "caps": [{"error": "x", "max": 7}]}`,
	} {
		if _, err := ParseScheme(bad, c); !errors.Is(err, ErrUnparseable) {
			t.Errorf("should be unparseable: %s (%v)", bad, err)
		}
	}
}

func TestCardText(t *testing.T) {
	got := CardText("Let <b>A</b> be&nbsp;invertible.<br>Show \\(A^{-1}\\) is unique.<div>Hint</div>")
	if got != "Let A be invertible.\nShow \\(A^{-1}\\) is unique.Hint" {
		t.Errorf("%q", got)
	}
}
