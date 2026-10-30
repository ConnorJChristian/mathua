package pytrack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
	"github.com/chuma-beep/mathua/internal/modules"
)

// Answer prefixes. The served answer says which kind of question it was, so
// one grading type can cover all three and every serve path (Learn, review,
// quiz) grades the same way without carrying extra fields.
const (
	prefixChoice = "mc:"
	prefixOutput = "out:"
	prefixCode   = "code:"
	prefixRecall = "recall:"
	prefixMath   = "math:"
	prefixTerm   = "term:"
)

// TypedMarker introduces a math item's answer fields in the served question:
// the prompt, then the marker followed by a JSON list of the fields, each
// {"label": LaTeX shown before the input, "grading": the grader}, then "-->".
// The client draws one math input per field and submits the answers one per
// line. The expected answers stay on the server.
const TypedMarker = "<!--typed-answer "

// TermMarker ends a term item's served question (fork change): the client
// draws one text input for the term. The expected answer stays on the server.
const TermMarker = "<!--term-answer-->"

// mathRouter grades math items with the graders Mathua's own questions use.
var mathRouter = grader.NewRouter()

// RecallMarker separates a recall card's front from its back in the served
// question. The client shows the front, reveals the back on request, and
// submits the learner's own verdict: "right" or "wrong". A recall card is a
// flashcard, so the back necessarily reaches the browser.
const RecallMarker = "<!--recall-back-->"

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

// Default is the process-wide track: the enabled modules' banks and a runner
// configured from the environment.
func Default() (*Track, error) {
	defaultOnce.Do(func() {
		mods, err := modules.Default()
		if err != nil {
			defaultErr = err
			return
		}
		b, err := LoadBank(mods.BankDirs()...)
		if err != nil {
			defaultErr = err
			return
		}
		b.FileExists = func(module, file string) bool {
			p := mods.FilePath(module, file)
			if p == "" {
				return false
			}
			st, err := os.Stat(p)
			return err == nil && !st.IsDir()
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

// ProblemFor serves one item of this concept's bank by its id (fork change:
// a module's cards.json files single cards under the topics they are about).
func (g *conceptGen) ProblemFor(itemID string) (generator.Problem, bool) {
	for i := range g.c.Items {
		if g.c.Items[i].ID == itemID {
			return g.track.Problem(g.c, &g.c.Items[i]), true
		}
	}
	return generator.Problem{}, false
}

// objectiveKind reports an item the server grades by itself and a learner
// answers in well under a minute: multiple choice, "what does this print" and
// typed math. Recall cards are self-graded and code tasks are long, so a test
// never serves them.
func objectiveKind(kind string) bool {
	return kind == KindChoice || kind == KindOutput || kind == KindMath || kind == KindTerm
}

// ObjectiveItems is how many of this topic's items a test may serve. A book
// reference (placement never serves one) and an item that unlocks after
// another (a new learner has not unlocked it) are not counted, so the count
// holds for every test and every learner.
func (g *conceptGen) ObjectiveItems() int {
	n := 0
	for _, it := range g.c.Items {
		if objectiveKind(it.Kind) && it.BookRef == nil && it.UnlocksAfter == "" {
			n++
		}
	}
	return n
}

// item finds one of this concept's items by id.
func (g *conceptGen) item(id string) *Item {
	for i := range g.c.Items {
		if g.c.Items[i].ID == id {
			return &g.c.Items[i]
		}
	}
	return nil
}

// outcome reports what the learner in ctx has done with one of this
// concept's items: whether they tried it and whether they got it right.
func (g *conceptGen) outcome(it *Item, ctx generator.GeneratorContext) (attempted, correct bool) {
	if ctx.History == nil {
		return false, false
	}
	p := g.track.problem(g.c, it) // the header does not change the key
	q := ""
	if it.Kind == KindChoice {
		q = generator.QuestionKey(p.Question)
	}
	return ctx.History.Outcome(p.Answer, q)
}

// ItemServable implements generator.GatedItems.
func (g *conceptGen) ItemServable(itemID string, ctx generator.GeneratorContext) bool {
	it := g.item(itemID)
	return it != nil && g.servable(it, ctx)
}

func (g *conceptGen) servable(it *Item, ctx generator.GeneratorContext) bool {
	if ctx.ObjectiveOnly && !objectiveKind(it.Kind) {
		return false
	}
	if it.BookRef != nil && ctx.NoBookRefs {
		return false
	}
	if it.UnlocksAfter != "" {
		prev := g.item(it.UnlocksAfter)
		if prev == nil {
			return false
		}
		if _, correct := g.outcome(prev, ctx); !correct {
			return false
		}
	}
	return true
}

// ItemBookPending implements generator.GatedItems.
func (g *conceptGen) ItemBookPending(itemID string, ctx generator.GeneratorContext) (pending, attempted bool) {
	it := g.item(itemID)
	if it == nil || it.BookRef == nil || ctx.NoBookRefs {
		return false, false
	}
	attempted, correct := g.outcome(it, ctx)
	return !correct, attempted
}

// pendingBooks lists the servable book items the learner has not answered
// correctly, in bank order, with whether each was tried.
func (g *conceptGen) pendingBooks(items []Item, ctx generator.GeneratorContext) ([]int, []bool) {
	var idx []int
	var tried []bool
	for i := range items {
		if items[i].BookRef == nil {
			continue
		}
		attempted, correct := g.outcome(&items[i], ctx)
		if !correct {
			idx = append(idx, i)
			tried = append(tried, attempted)
		}
	}
	return idx, tried
}

// BookPending implements generator.BookPender: this topic still has a book
// exercise the learner has not answered correctly. Tests never prefer them.
func (g *conceptGen) BookPending(ctx generator.GeneratorContext) bool {
	if ctx.NoBookRefs || ctx.ObjectiveOnly {
		return false
	}
	for i := range g.c.Items {
		if it := &g.c.Items[i]; it.BookRef != nil {
			if _, correct := g.outcome(it, ctx); !correct {
				return true
			}
		}
	}
	return false
}

func (g *conceptGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	// Fork change: drop what this learner may not be served (an item that
	// unlocks after another not yet answered correctly, a book page in
	// placement, anything slow or self-graded in a test).
	items := g.c.Items
	gated := false
	for i := range items {
		if items[i].UnlocksAfter != "" || items[i].BookRef != nil || ctx.ObjectiveOnly {
			gated = true
			break
		}
	}
	if gated {
		var kept []Item
		for i := range items {
			if g.servable(&items[i], ctx) {
				kept = append(kept, items[i])
			}
		}
		if len(kept) == 0 {
			return generator.Problem{}
		}
		items = kept
	}
	// Book exercises come first in practice until each is answered correctly
	// (generator.PickBook); tests mix them in with the rest.
	if !ctx.ObjectiveOnly && !ctx.NoBookRefs {
		if idx, tried := g.pendingBooks(items, ctx); len(idx) > 0 {
			if k := generator.PickBook(tried); k >= 0 {
				p := g.track.Problem(g.c, &items[idx[k]])
				if ctx.NoProofs || ctx.NoCards {
					p = PlainRecall(p)
				}
				return p
			}
		}
	}
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
	p := g.track.Problem(g.c, &items[idx])
	if ctx.NoProofs || ctx.NoCards || ctx.ObjectiveOnly {
		// A quiz, placement or a test asks for no written answer.
		p = PlainRecall(p)
	}
	return p
}

// Problem renders one item as a served question. A book-reference item
// opens with the book header (generator.BookHeader).
func (t *Track) Problem(c *Concept, it *Item) generator.Problem {
	p := t.problem(c, it)
	if it.BookRef != nil {
		p.Question = generator.BookHeader(*it.BookRef, t.Bank.fileExists(it.BookRef.Module, it.BookRef.File)) + "\n\n" + p.Question
	}
	return p
}

func (t *Track) problem(c *Concept, it *Item) generator.Problem {
	book := ""
	if c.Book.URL != "" {
		name := c.Book.Name
		if name == "" {
			name = "Python Data Science Handbook"
		}
		book = fmt.Sprintf("Read more: [%s §%s, %s](%s)", name, c.Book.Section, c.Book.Title, c.Book.URL)
	}
	switch it.Kind {
	case KindChoice:
		q := strings.TrimSpace(it.Prompt)
		for i, o := range stableOptions(c.ID, it) {
			q += fmt.Sprintf("\n\n%c) %s", 'A'+i, o)
		}
		return generator.Problem{Question: q, Answer: prefixChoice + it.Correct, Explanation: joinNonEmpty(strings.TrimSpace(it.Explain), book)}
	case KindOutput:
		q := strings.TrimSpace(it.Prompt) + "\n\n```python\n" + strings.TrimRight(it.Code, "\n") + "\n```"
		exp := joinNonEmpty(strings.TrimSpace(it.Explain)+"\n\nIt prints:\n\n```text\n"+strings.TrimRight(it.Expected, "\n")+"\n```", book)
		return generator.Problem{Question: q, Answer: prefixOutput + it.Expected, Explanation: exp}
	case KindRecall:
		q := strings.TrimSpace(it.Prompt) + "\n\n" + RecallMarker + "\n\n" + strings.TrimSpace(it.Back)
		exp := strings.TrimSpace(it.Back)
		if strings.TrimSpace(it.Explain) != "" {
			exp += "\n\n" + strings.TrimSpace(it.Explain)
		}
		if c.Book.URL != "" {
			exp += "\n\n" + book
		} else if c.Book.Title != "" {
			exp += "\n\nSource: " + c.Book.Title
		}
		p := generator.Problem{Question: q, Answer: prefixRecall + itemKey(c.ID, it.ID), Explanation: exp}
		if t.servesWritten(c, it) {
			p = writtenProblem(p, it)
		}
		return p
	case KindMath:
		type field struct {
			Label   string `json:"label"`
			Grading string `json:"grading"`
		}
		fields := make([]field, len(it.Parts))
		shown := make([]string, len(it.Parts))
		for i, p := range it.Parts {
			fields[i] = field{Label: p.Label, Grading: p.Grading}
			if p.Label != "" {
				shown[i] = "\\(" + p.Label + " = " + p.Tex + "\\)"
			} else {
				shown[i] = "\\(" + p.Tex + "\\)"
			}
		}
		js, _ := json.Marshal(fields)
		q := strings.TrimSpace(it.Prompt) + "\n\n" + TypedMarker + string(js) + "-->"
		exp := "**Answer:** " + strings.Join(shown, ", ")
		if b := strings.TrimSpace(it.Back); b != "" {
			exp += "\n\n" + b
		}
		if strings.TrimSpace(it.Explain) != "" {
			exp += "\n\n" + strings.TrimSpace(it.Explain)
		}
		if c.Book.URL != "" {
			exp += "\n\n" + book
		} else if c.Book.Title != "" {
			exp += "\n\nSource: " + c.Book.Title
		}
		return generator.Problem{Question: q, Answer: prefixMath + itemKey(c.ID, it.ID), Explanation: exp}
	case KindTerm:
		q := strings.TrimSpace(it.Prompt) + "\n\n" + TermMarker
		exp := "**Answer:** " + strings.TrimSpace(it.Correct)
		if strings.TrimSpace(it.Explain) != "" {
			exp += "\n\n" + strings.TrimSpace(it.Explain)
		}
		if book != "" {
			exp += "\n\n" + book
		} else if c.Book.Title != "" {
			exp += "\n\nSource: " + c.Book.Title
		}
		return generator.Problem{Question: q, Answer: prefixTerm + itemKey(c.ID, it.ID), Explanation: exp}
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
		exp := joinNonEmpty(strings.TrimSpace(it.Explain)+"\n\nOne solution:\n\n```python\n"+strings.TrimRight(it.Solution, "\n")+"\n```", book)
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
	// True/False, Yes/No and Sí/No read in their usual order.
	if binaryChoice(*it) {
		if negativeOption(opts[0]) {
			opts[0], opts[1] = opts[1], opts[0]
		}
		return opts
	}
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
	case strings.HasPrefix(expected, prefixRecall):
		switch strings.ToLower(strings.TrimSpace(answer)) {
		case "right":
			return grader.Result{Correct: true, Score: 1}
		case "wrong":
			return grader.Result{Feedback: "Marked as missed"}
		case "":
			return grader.Result{Feedback: "Reveal the answer, then mark whether you had it"}
		default:
			// A written answer to a written-answer card (written.go).
			return t.gradeWritten(strings.TrimPrefix(expected, prefixRecall), answer)
		}
	case strings.HasPrefix(expected, prefixCode):
		return t.gradeCode(strings.TrimPrefix(expected, prefixCode), answer)
	case strings.HasPrefix(expected, prefixMath):
		return t.gradeMath(strings.TrimPrefix(expected, prefixMath), answer)
	case strings.HasPrefix(expected, prefixTerm):
		return t.gradeTerm(strings.TrimPrefix(expected, prefixTerm), answer)
	default:
		return grader.Result{Feedback: "Unknown Python question type", Unavailable: true}
	}
}

// gradeTerm grades a term item: right when the answer matches the term or one
// of its accepted spellings after normalizeTerm.
func (t *Track) gradeTerm(key, answer string) grader.Result {
	it := t.Bank.Item(key)
	if it == nil || it.Kind != KindTerm {
		return grader.Result{Feedback: "This question is no longer in the question bank; load a new question", Unavailable: true}
	}
	got := normalizeTerm(answer)
	if got == "" {
		return grader.Result{Feedback: "No answer given"}
	}
	for _, want := range append([]string{it.Correct}, it.Accept...) {
		w := normalizeTerm(want)
		if got == w || strings.ReplaceAll(got, " ", "") == strings.ReplaceAll(w, " ", "") {
			return grader.Result{Correct: true, Score: 1}
		}
	}
	return grader.Result{Feedback: "That is not the term"}
}

// termFold maps accented Latin letters to their base letter, so a learner
// without an accent key is not marked wrong for "plusvalia".
var termFold = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ä", "a", "ã", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "ö", "o", "õ", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ñ", "n", "ç", "c",
)

// termArticles are leading articles a typed term may carry ("la plusvalía").
var termArticles = []string{"el ", "la ", "los ", "las ", "lo ", "un ", "una ", "the ", "a ", "an "}

// normalizeTerm puts a typed term in a comparable form: lower case, accents
// folded, hyphens and runs of space as one space, surrounding punctuation and
// quotes trimmed, and one leading article dropped.
func normalizeTerm(s string) string {
	s = termFold.Replace(strings.ToLower(s))
	s = strings.NewReplacer("-", " ", "\u2013", " ", "\u2014", " ").Replace(s)
	s = strings.TrimFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	s = spaceRun.ReplaceAllString(s, " ")
	for _, a := range termArticles {
		if strings.HasPrefix(s, a) && len(s) > len(a) {
			s = s[len(a):]
			break
		}
	}
	return s
}

// gradeMath grades a math item: one answer per line, one line per part, each
// with the part's own grader. The item is right when every part is.
func (t *Track) gradeMath(key, answer string) grader.Result {
	it := t.Bank.Item(key)
	if it == nil || it.Kind != KindMath {
		return grader.Result{Feedback: "This question is no longer in the question bank; load a new question", Unavailable: true}
	}
	lines := strings.Split(strings.ReplaceAll(answer, "\r\n", "\n"), "\n")
	right := 0
	var missed []string
	for i, p := range it.Parts {
		a := ""
		if i < len(lines) {
			a = strings.TrimSpace(lines[i])
		}
		name := p.Label
		if name == "" {
			name = "the answer"
		} else {
			name = "\\(" + name + "\\)"
		}
		if a == "" {
			missed = append(missed, name+" is empty")
			continue
		}
		key, typed := grader.NormalizeTypedMath(p.Answer, p.Answer), grader.NormalizeTypedMath(a, p.Answer)
		if p.Grading == "set" || p.Grading == "interval" {
			// Braces are the notation here, not LaTeX grouping.
			key, typed = p.Answer, a
		}
		res := mathRouter.Grade(grader.GradingType(p.Grading), key, typed)
		if res.Unavailable {
			return res
		}
		// The answer the prompt asked for: the key rounded to the places
		// it names (fork fix).
		if !res.Correct && p.Grading != "set" && p.Grading != "interval" && grader.RoundedAsAsked(key, typed, grader.AskedPlaces(it.Prompt)) {
			res = grader.Result{Correct: true, Score: 1}
		}
		if res.Correct {
			right++
		} else {
			missed = append(missed, name+" is not right")
		}
	}
	n := len(it.Parts)
	if right == n {
		return grader.Result{Correct: true, Score: 1}
	}
	fb := "Not equivalent to the expected answer"
	if n > 1 {
		fb = fmt.Sprintf("%d of %d right: %s", right, n, strings.Join(missed, ", "))
	}
	return grader.Result{Score: float64(right) / float64(n), Feedback: fb}
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

func joinNonEmpty(parts ...string) string {
	var out []string
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, "\n\n")
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
