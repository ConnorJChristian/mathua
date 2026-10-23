package pytrack

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
)

// Track names. A concept belongs to the Python track when its domain starts
// with DomainPrefix; everything else is the Math track.
const (
	TrackMath    = "math"
	TrackPython  = "python"
	DomainPrefix = "python_"
)

// IsPythonDomain reports whether a concept domain belongs to the Python track.
func IsPythonDomain(domain string) bool { return strings.HasPrefix(domain, DomainPrefix) }

// TrackOfDomain names the track a domain belongs to.
func TrackOfDomain(domain string) string {
	if IsPythonDomain(domain) {
		return TrackPython
	}
	return TrackMath
}

// NormalizeTrack maps a request parameter onto a track. Anything that is not
// "python" is the Math track, so a client that never sends the parameter keeps
// the behaviour it had before the Python track existed.
func NormalizeTrack(s string) string {
	if strings.EqualFold(strings.TrimSpace(s), TrackPython) {
		return TrackPython
	}
	return TrackMath
}

// Answer prefixes. The served answer says which kind of question it was, so
// one grading type can cover all three and every serve path (Learn, review,
// quiz) grades the same way without carrying extra fields.
const (
	prefixChoice = "mc:"
	prefixOutput = "out:"
	prefixCode   = "code:"
)

// StarterFence is the info string that marks a code task's starter code in
// the question Markdown. The client turns that block into the code box.
const StarterFence = "python starter"

// Track ties the bank to a runner.
type Track struct {
	Bank   *Bank
	Runner *Runner
}

var (
	defaultOnce  sync.Once
	defaultTrack *Track
	defaultErr   error
)

// Default is the process-wide track: the embedded bank and a runner configured
// from the environment.
func Default() (*Track, error) {
	defaultOnce.Do(func() {
		b, err := LoadBank()
		if err != nil {
			defaultErr = err
			return
		}
		defaultTrack = &Track{Bank: b, Runner: NewRunner()}
	})
	return defaultTrack, defaultErr
}

// Register adds one generator per bank concept and installs the grader.
func (t *Track) Register(reg *generator.Registry) error {
	for _, id := range t.Bank.ConceptIDs() {
		if err := reg.Register(id, &conceptGen{track: t, c: t.Bank.Concept(id)}); err != nil {
			return err
		}
	}
	grader.SetPythonGrader(t.Grade)
	return nil
}

// conceptGen serves one concept's bank.
type conceptGen struct {
	track *Track
	c     *Concept
}

func (g *conceptGen) PlainText() bool { return true }

func (g *conceptGen) AnswerFormat() generator.AnswerFormat {
	return generator.AnswerFormat{
		Hint:      "Pick an option, type exactly what the code prints, or write code in the box",
		InputMode: "text",
	}
}

// levelDifficulty places a bank level on the engine's 0.3-1.0 difficulty scale.
func levelDifficulty(level int) float64 {
	switch level {
	case 1:
		return 0.35
	case 3:
		return 0.9
	default:
		return 0.6
	}
}

func (g *conceptGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	items := g.c.Items
	// Prefer questions near the requested difficulty, but never starve: a
	// concept whose bank has nothing near it serves from the whole bank.
	var near []int
	for i, it := range items {
		if math.Abs(levelDifficulty(it.Level)-ctx.Difficulty) <= 0.3 {
			near = append(near, i)
		}
	}
	idx := 0
	if len(near) > 0 && rand.Intn(5) > 0 { // one in five draws from the whole bank for variety
		idx = near[rand.Intn(len(near))]
	} else {
		idx = rand.Intn(len(items))
	}
	return g.track.Problem(g.c, &items[idx])
}

// Problem renders one item as a served question.
func (t *Track) Problem(c *Concept, it *Item) generator.Problem {
	book := fmt.Sprintf("Read more: [Python Data Science Handbook §%s, %s](%s)", c.Book.Section, c.Book.Title, c.Book.URL)
	switch it.Kind {
	case KindChoice:
		q := strings.TrimSpace(it.Prompt)
		for i, o := range stableOptions(c.ID, it) {
			q += fmt.Sprintf("\n\n%c) %s", 'A'+i, o)
		}
		return generator.Problem{Question: q, Answer: prefixChoice + it.Correct, Explanation: strings.TrimSpace(it.Explain) + "\n\n" + book}
	case KindOutput:
		q := strings.TrimSpace(it.Prompt) + "\n\n```python\n" + strings.TrimRight(it.Code, "\n") + "\n```"
		exp := strings.TrimSpace(it.Explain) + "\n\nIt prints:\n\n```text\n" + strings.TrimRight(it.Expected, "\n") + "\n```\n\n" + book
		return generator.Problem{Question: q, Answer: prefixOutput + it.Expected, Explanation: exp}
	default: // KindCode
		starter := strings.TrimRight(it.Starter, "\n")
		if starter == "" {
			starter = "# Write your code here"
		}
		// The fence names the task so the code box's Run button can ask for the
		// task's fixture on every surface (Learn, review, quiz), including those
		// that do not carry the served answer. The key is an identifier, not a
		// secret: the tests it points at never leave the server.
		key := itemKey(c.ID, it.ID)
		q := strings.TrimSpace(it.Prompt) + "\n\n```" + StarterFence + " task=" + key + "\n" + starter + "\n```"
		exp := strings.TrimSpace(it.Explain) + "\n\nOne solution:\n\n```python\n" + strings.TrimRight(it.Solution, "\n") + "\n```\n\n" + book
		return generator.Problem{Question: q, Answer: prefixCode + key, Explanation: exp}
	}
}

// stableOptions orders an item's options the same way every time it is served.
//
// The engine records each served question by its text, and mastery evidence
// counts distinct texts as distinct problem instances. Shuffling per serve
// would turn one question into several "instances" and credit variety the
// learner never showed, so the order is a fixed function of the item instead.
// At most five options are kept, which is what the client can label.
func stableOptions(conceptID string, it *Item) []string {
	opts := append([]string{it.Correct}, it.Wrong...)
	if len(opts) > 5 {
		opts = opts[:5]
	}
	h := fnv.New64a()
	h.Write([]byte(conceptID + "#" + it.ID))
	r := rand.New(rand.NewSource(int64(h.Sum64())))
	r.Shuffle(len(opts), func(i, j int) { opts[i], opts[j] = opts[j], opts[i] })
	return opts
}

// Grade grades any Python-track answer. It is installed as the grader for the
// "python" grading type.
func (t *Track) Grade(expected, answer string) grader.Result {
	switch {
	case strings.HasPrefix(expected, prefixChoice):
		if strings.TrimSpace(answer) == "" {
			return grader.Result{Feedback: "No answer given"}
		}
		if normalizeChoice(answer) == normalizeChoice(strings.TrimPrefix(expected, prefixChoice)) {
			return grader.Result{Correct: true, Score: 1}
		}
		return grader.Result{Feedback: "Incorrect choice"}
	case strings.HasPrefix(expected, prefixOutput):
		if strings.TrimSpace(answer) == "" {
			return grader.Result{Feedback: "No answer given"}
		}
		if NormalizeOutput(answer) == NormalizeOutput(strings.TrimPrefix(expected, prefixOutput)) {
			return grader.Result{Correct: true, Score: 1}
		}
		return grader.Result{Feedback: "That is not what the code prints"}
	case strings.HasPrefix(expected, prefixCode):
		return t.gradeCode(strings.TrimPrefix(expected, prefixCode), answer)
	default:
		return grader.Result{Feedback: "Unknown Python question type", Unavailable: true}
	}
}

func (t *Track) gradeCode(key, code string) grader.Result {
	it := t.Bank.Item(key)
	if it == nil || it.Kind != KindCode {
		return grader.Result{Feedback: "This task is no longer in the question bank; load a new question", Unavailable: true}
	}
	if strings.TrimSpace(code) == "" {
		return grader.Result{Feedback: "Write some code first", Diagnosis: "The code box was empty."}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(t.Runner.TimeLimit())*time.Second+t.Runner.WallExtra+5*time.Second)
	defer cancel()
	res, err := t.Runner.Run(ctx, Request{Mode: "grade", Code: code, Setup: it.Setup, Tests: it.Tests})
	if err != nil {
		if errors.Is(err, ErrUnavailable) {
			return grader.Result{Feedback: "The Python runner is unavailable: " + err.Error(), Unavailable: true}
		}
		return grader.Result{Feedback: err.Error(), Unavailable: true}
	}
	return t.verdict(res)
}

func (t *Track) verdict(res *Result) grader.Result {
	if res.SetupError != "" {
		// The hidden fixture failed, which is the bank's fault, not the learner's.
		return grader.Result{Feedback: "This task's setup failed: " + res.SetupError, Unavailable: true}
	}
	if res.TimedOut {
		msg := fmt.Sprintf("Your code ran past the %d-second time limit. Look for a loop that never ends.", t.Runner.TimeLimit())
		return grader.Result{Feedback: msg, Diagnosis: msg}
	}
	if res.Error != "" {
		last := lastLine(res.Error)
		return grader.Result{
			Feedback:  "Your code raised an error:\n" + res.Error,
			Diagnosis: "Your code raised an error: " + last,
		}
	}
	total := len(res.Tests)
	var failed []TestResult
	for _, tr := range res.Tests {
		if !tr.Passed {
			failed = append(failed, tr)
		}
	}
	if res.Passed && len(failed) == 0 {
		return grader.Result{Correct: true, Score: 1}
	}
	if total == 0 {
		return grader.Result{Feedback: "No checks ran", Unavailable: true}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d of %d hidden checks failed:", len(failed), total)
	for _, f := range failed {
		b.WriteString("\n- " + f.Message)
	}
	diag := fmt.Sprintf("%d of %d checks failed. %s", len(failed), total, failed[0].Message)
	return grader.Result{
		Score:     float64(total-len(failed)) / float64(total),
		Feedback:  b.String(),
		Diagnosis: diag,
	}
}

func lastLine(s string) string {
	s = strings.TrimRight(s, "\n")
	if i := strings.LastIndexByte(s, '\n'); i >= 0 {
		return s[i+1:]
	}
	return s
}

// RunCode runs learner code for the Run button: the task's fixture, then the
// code, with output and figures returned and no tests. key is the served
// answer ("code:<concept>#<item>") or empty for a scratch run.
func (t *Track) RunCode(ctx context.Context, key, code string) (*Result, error) {
	setup := ""
	key = strings.TrimPrefix(key, prefixCode)
	if key != "" {
		if it := t.Bank.Item(key); it != nil && it.Kind == KindCode {
			setup = it.Setup
		}
	}
	return t.Runner.Run(ctx, Request{Mode: "run", Code: code, Setup: setup, Figures: true})
}

var spaceRun = regexp.MustCompile(`\s+`)
var spaceNearPunct = regexp.MustCompile(`\s*([\[\](){},:])\s*`)

// NormalizeOutput is the comparison form of printed output.
//
// Whitespace is the one thing a learner cannot reasonably reproduce: NumPy pads
// columns to a common width and wraps long arrays, and a learner typing
// `[1 2 3]` for `[ 1  2  3]` has read the output correctly. So runs of
// whitespace, including line breaks, compare as one space, and whitespace next
// to brackets, commas and colons is ignored. Everything else — digits, quotes,
// dtype names, the order of values — must match exactly.
func NormalizeOutput(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.TrimSpace(s)
	s = spaceRun.ReplaceAllString(s, " ")
	s = spaceNearPunct.ReplaceAllString(s, "$1")
	return s
}

func normalizeChoice(s string) string {
	s = strings.TrimSpace(s)
	s = spaceRun.ReplaceAllString(s, " ")
	return s
}
