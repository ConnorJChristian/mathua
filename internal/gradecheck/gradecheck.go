// Package gradecheck grades every question's own answer key through the path
// a learner's answer takes, and reports the keys that do not grade correct,
// the plausible equivalent spellings that are refused, and the wrong answers
// that are accepted (fork change).
//
// A question bank's item carries its answer; a generator computes one. Either
// is only a question if submitting that answer grades correct, if the other
// spellings a learner reasonably types grade correct too, and if a wrong
// answer grades wrong. The server never checks that: a generator checks one
// problem of itself at registration, and scripts/python_bank.py runs code
// tasks, but a typed-math key written as "\frac{1}{2}" that the numeric
// grader cannot read, or a multiple-choice key equal to one of its wrong
// options, is only found when a learner meets it.
//
// The check runs every bank item of the loaded modules (internal/modules,
// so MATHUA_PUBLIC_MODULES_ONLY and MATHUA_ROOT apply) and a sample of every
// generator through engine.GradeWith with grader.ResolveChoiceLetter in
// front, exactly as internal/server submits an answer.
//
//	go test ./internal/gradecheck          (the committed modules)
//	go run ./scripts/gradecheck -local     (your modules-local/ too)
package gradecheck

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/engine"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
	"github.com/chuma-beep/mathua/internal/modules"
	"github.com/chuma-beep/mathua/internal/pytrack"
)

// Failure is one answer that graded the wrong way.
type Failure struct {
	Module  string
	Bank    string // bank file without .json, or "generator"
	Concept string
	Item    string
	Kind    string
	// Check names what was submitted: "key", "key letter", "variant",
	// "wrong", "empty", "leak", "asked" (the key rounded as the prompt asks).
	Check    string
	Answer   string
	Feedback string
	// Expected is the answer key as served (without its bank prefix).
	Expected string
	// Custom marks a generator with its own grader (generator.GradedGenerator),
	// whose answer format is its own: a variant it refuses is a note, not a
	// defect, where a shared-grader refusal is.
	Custom bool
}

func (f Failure) String() string {
	where := f.Concept
	if f.Item != "" {
		where += "#" + f.Item
	}
	tag := ""
	if f.Custom {
		tag = " (custom grader)"
	}
	return fmt.Sprintf("%s/%s %s [%s %s] %s%s: answer %q (key %q): %s",
		f.Module, f.Bank, where, f.Kind, f.Check, f.Verdict(), tag, f.Answer, f.Expected, f.Feedback)
}

// Verdict says which way the grade went against what was wanted.
func (f Failure) Verdict() string {
	switch f.Check {
	case "wrong", "empty", "wrong letter":
		return "ACCEPTED"
	case "serve":
		return "DEFECT"
	case "leak":
		// A prompt that holds its key may mean to ("which of these two
		// verbs", "convert e^{1.5} to log form"): a note for the author.
		return "NOTE"
	}
	return "REFUSED"
}

// Options select what the check submits.
type Options struct {
	// Code runs every code task's reference solution and starter through the
	// Python runner. Slow: one interpreter per task.
	Code bool
	// Variants submits equivalent spellings of each typed-math key.
	Variants bool
	// Wrong submits plausible wrong answers.
	Wrong bool
	// Seeds is how many problems to sample per generator concept.
	Seeds int
	// Modules limits the bank check to these module ids (empty: all).
	Modules []string
	// Concepts limits the check to these concept ids (empty: all).
	Concepts []string
	// Editor maps a key's LaTeX (Part.Tex) to what the typed-LaTeX box
	// submits for it (web/next-app/scripts/latex-plain.mjs: MathLive's
	// converter, then plainAnswer.ts). When set, that form is submitted
	// as the "editor" check.
	Editor map[string]string
	// Log receives progress lines; nil is silent.
	Log func(format string, args ...any)
}

// Defects are the failures that are not a custom grader's own format
// refusing an alternative spelling, and not a leak note.
func (r Report) Defects() []Failure {
	var out []Failure
	for _, f := range r.Failures {
		if (f.Custom && f.Check == "variant") || f.Check == "leak" {
			continue
		}
		out = append(out, f)
	}
	return out
}

// Report is the outcome of one run.
type Report struct {
	Items    int // bank items checked
	Problems int // generated problems checked
	Checks   int // answers submitted
	Failures []Failure
	// Skipped counts items left out (code tasks without Options.Code,
	// recall cards' written answers without a judge).
	Skipped int
}

// Checker holds the loaded content and the grading path.
type Checker struct {
	Mods   *modules.Set
	DAG    *concepts.DAG
	Reg    *generator.Registry
	Track  *pytrack.Track
	Router *grader.Router
	// moduleOf maps a bank dir to the module id.
	moduleOf map[string]string
}

// New loads the enabled modules' banks and the concept graph. reg must have
// every generator registered, the bank included (internal/generator/all).
func New(mods *modules.Set, dag *concepts.DAG, reg *generator.Registry, t *pytrack.Track) *Checker {
	c := &Checker{Mods: mods, DAG: dag, Reg: reg, Track: t, Router: grader.NewRouter(), moduleOf: map[string]string{}}
	for _, m := range mods.Modules {
		c.moduleOf[m.ID] = m.Dir
	}
	return c
}

// grade is the server's path: the choice letter resolved against the served
// question, then engine.GradeWith.
func (c *Checker) grade(conceptID, question, expected, answer string) grader.Result {
	return engine.GradeWith(c.DAG, c.Reg, c.Router, conceptID, expected, grader.ResolveChoiceLetter(question, expected, answer))
}

func (c *Checker) moduleFor(bank string) string {
	for _, m := range c.Mods.Modules {
		if _, err := os.Stat(filepath.Join(m.Dir, "bank", bank+".json")); err == nil {
			return m.ID
		}
	}
	return "?"
}

func (o Options) wants(module, concept string) bool {
	if len(o.Modules) > 0 && !contains(o.Modules, module) {
		return false
	}
	if len(o.Concepts) > 0 && !contains(o.Concepts, concept) {
		return false
	}
	return true
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// CheckBank grades every bank item's key, and with the options its variants
// and wrong answers.
func (c *Checker) CheckBank(opts Options) Report {
	var rep Report
	logf := opts.Log
	if logf == nil {
		logf = func(string, ...any) {}
	}
	for _, id := range c.Track.Bank.ConceptIDs() {
		con := c.Track.Bank.Concept(id)
		bank := c.Track.Bank.BankOf(id)
		module := c.moduleFor(bank)
		if !opts.wants(module, id) {
			continue
		}
		for i := range con.Items {
			it := &con.Items[i]
			rep.Items++
			p := c.Track.Problem(con, it)
			expected := strings.TrimPrefix(p.Answer, kindPrefix(it.Kind))
			if it.Kind == pytrack.KindMath {
				var ks []string
				for _, pt := range it.Parts {
					ks = append(ks, pt.Answer)
				}
				expected = strings.Join(ks, "\n")
			}
			at := func(check, answer, feedback string) {
				rep.Failures = append(rep.Failures, Failure{Module: module, Bank: bank, Concept: id, Item: it.ID, Kind: it.Kind,
					Check: check, Answer: answer, Feedback: feedback, Expected: expected})
			}
			right := func(check, answer string) {
				rep.Checks++
				res := c.grade(id, p.Question, p.Answer, answer)
				if res.Unavailable {
					at(check, answer, "ungraded: "+res.Feedback)
					return
				}
				if !res.Correct {
					at(check, answer, res.Feedback)
				}
			}
			wrong := func(check, answer string) {
				if !opts.Wrong {
					return
				}
				rep.Checks++
				res := c.grade(id, p.Question, p.Answer, answer)
				if res.Correct {
					at(check, answer, "graded correct")
				}
			}
			switch it.Kind {
			case pytrack.KindChoice:
				c.checkChoice(it, p, opts, right, wrong, at)
			case pytrack.KindOutput:
				right("key", it.Expected)
				right("variant", strings.Join(strings.Fields(it.Expected), " "))
				wrong("empty", "")
				wrong("wrong", perturbOutput(it.Expected))
				wrong("wrong", it.Prompt)
			case pytrack.KindTerm:
				c.checkTerm(it, p, opts, right, wrong, at)
			case pytrack.KindMath:
				c.checkMath(it, p, opts, right, wrong, at)
			case pytrack.KindRecall:
				right("key", "right")
				wrong("wrong", "wrong")
				wrong("empty", "")
				// A written answer without a judge is ungraded, never a miss
				// and never a pass.
				rep.Checks++
				if res := c.grade(id, p.Question, p.Answer, "A written answer."); res.Correct || !res.Unavailable {
					at("written", "A written answer.", fmt.Sprintf("a written answer with no judge must be ungraded, got correct=%v unavailable=%v", res.Correct, res.Unavailable))
				}
			case pytrack.KindCode:
				if !opts.Code {
					rep.Skipped++
					continue
				}
				right("key", it.Solution)
				wrong("empty", "")
				starter := it.Starter
				if strings.TrimSpace(starter) == "" {
					starter = "pass"
				}
				wrong("wrong", starter)
			}
		}
		logf("%s/%s %s: %d items", module, bank, id, len(con.Items))
	}
	return rep
}

func kindPrefix(kind string) string {
	switch kind {
	case pytrack.KindChoice:
		return "mc:"
	case pytrack.KindOutput:
		return "out:"
	case pytrack.KindCode:
		return "code:"
	case pytrack.KindRecall:
		return "recall:"
	case pytrack.KindMath:
		return "math:"
	case pytrack.KindTerm:
		return "term:"
	}
	return ""
}

var optionRe = regexp.MustCompile(`(?m)^([A-E])\) (.+)$`)

func (c *Checker) checkChoice(it *pytrack.Item, p generator.Problem, opts Options, right, wrong func(check, answer string), at func(check, answer, feedback string)) {
	right("key", it.Correct)
	// The served options: the key must be among them, and its letter must
	// grade right as a typed letter.
	opts2 := optionRe.FindAllStringSubmatch(p.Question, -1)
	letter := ""
	for _, m := range opts2 {
		if strings.TrimSpace(m[2]) == strings.TrimSpace(it.Correct) {
			letter = m[1]
		}
	}
	if letter == "" {
		at("serve", it.Correct, "the correct option is not among the served options")
	} else {
		right("key letter", letter)
		right("key letter", strings.ToLower(letter))
	}
	right("variant", " "+it.Correct+" ")
	wrong("empty", "")
	for _, m := range opts2 {
		if strings.TrimSpace(m[2]) == strings.TrimSpace(it.Correct) {
			continue
		}
		wrong("wrong", m[2])
		wrong("wrong letter", m[1])
	}
	for _, w := range it.Wrong {
		wrong("wrong", w)
	}
	if !opts.Wrong {
		return
	}
	// A prompt that states its own answer is a leak: the stem contains the
	// key as a whole phrase (not a one-word key like "Yes").
	if k := strings.ToLower(strings.TrimSpace(it.Correct)); len(strings.Fields(k)) >= 3 && strings.Contains(strings.ToLower(it.Prompt), k) {
		at("leak", it.Correct, "the prompt contains the correct option's text")
	}
}

func (c *Checker) checkTerm(it *pytrack.Item, p generator.Problem, opts Options, right, wrong func(check, answer string), at func(check, answer, feedback string)) {
	right("key", it.Correct)
	right("variant", strings.ToUpper(it.Correct))
	right("variant", " "+it.Correct+".")
	for _, a := range it.Accept {
		right("key", a)
	}
	wrong("empty", "")
	wrong("wrong", "x")
	wrong("wrong", it.Correct+" zz")
	if !opts.Wrong {
		return
	}
	// A term the prompt itself names is given away.
	if k := strings.ToLower(strings.TrimSpace(it.Correct)); len(k) >= 4 && strings.Contains(strings.ToLower(it.Prompt), k) {
		at("leak", it.Correct, "the prompt contains the term")
	}
}

func (c *Checker) checkMath(it *pytrack.Item, p generator.Problem, opts Options, right, wrong func(check, answer string), at func(check, answer, feedback string)) {
	keys := make([]string, len(it.Parts))
	for i, pt := range it.Parts {
		keys[i] = pt.Answer
	}
	join := func(parts []string) string { return strings.Join(parts, "\n") }
	right("key", join(keys))
	// The LaTeX the source wrote, as a learner who copies the book's
	// notation types it (NormalizeTypedMath reads it).
	texs := make([]string, len(it.Parts))
	texOK := true
	for i, pt := range it.Parts {
		texs[i] = strings.TrimSpace(pt.Tex)
		if texs[i] == "" || strings.ContainsAny(texs[i], "=<>,") || strings.Contains(texs[i], "\\text") || strings.Contains(texs[i], "\\begin") {
			texOK = false
		}
	}
	if texOK && join(texs) != join(keys) {
		right("tex", join(texs))
	}
	if opts.Editor != nil {
		// What the typed-LaTeX box sends for the source's LaTeX.
		ed := make([]string, len(it.Parts))
		ok := true
		for i, pt := range it.Parts {
			v, found := opts.Editor[strings.TrimSpace(pt.Tex)]
			if !found || strings.TrimSpace(v) == "" {
				ok = false
			}
			ed[i] = v
		}
		if ok && join(ed) != join(keys) {
			right("editor", join(ed))
		}
	}
	if opts.Variants {
		// Each part's spellings in turn, the other parts at their key.
		for i, pt := range it.Parts {
			for _, v := range variants(pt) {
				sub := append([]string(nil), keys...)
				sub[i] = v
				right("variant", join(sub))
			}
			// The answer the prompt asks for ("to two decimal places"): the
			// key rounded to those places must grade right, and one unit
			// off in the last place must not.
			if n := grader.AskedPlaces(it.Prompt); n > 0 && pt.Grading != "set" && pt.Grading != "interval" {
				if v, ok := constEval(strings.TrimSpace(pt.Answer)); ok {
					r := roundHalfAway(v, n)
					sub := append([]string(nil), keys...)
					sub[i] = strconv.FormatFloat(r, 'f', n, 64)
					right("asked", join(sub))
					// A key written with fewer places than the prompt names
					// ("99.93" under "round the coefficients to three
					// decimals") is graded at its own places, and often the
					// instruction is about another quantity: only a key
					// that carries the places asked, or an exact one, says
					// what one unit off is.
					a := strings.TrimSpace(pt.Answer)
					if dot := strings.IndexByte(a, '.'); dot < 0 || len(a)-dot-1 >= n {
						off := append([]string(nil), keys...)
						off[i] = strconv.FormatFloat(r+math.Pow(10, -float64(n)), 'f', n, 64)
						wrong("wrong", join(off))
					}
				}
			}
		}
	}
	wrong("empty", "")
	if len(it.Parts) > 1 {
		// Only the first part right is not the item right.
		sub := make([]string, len(keys))
		sub[0] = keys[0]
		wrong("wrong", join(sub))
	}
	for i, pt := range it.Parts {
		for _, w := range wrongs(pt) {
			if w == strings.TrimSpace(pt.Answer) {
				continue // a key that is itself "x"
			}
			sub := append([]string(nil), keys...)
			sub[i] = w
			wrong("wrong", join(sub))
		}
	}
	if !opts.Wrong {
		return
	}
	// A key written into the prompt (a "show that" with the value in the
	// stem) is a leak, for a key a learner could not guess.
	for _, pt := range it.Parts {
		k := strings.TrimSpace(pt.Answer)
		if len(k) >= 4 && !strings.HasPrefix(k, "0.") && strings.Contains(strings.ReplaceAll(it.Prompt, " ", ""), k) {
			at("leak", k, "the prompt contains the answer")
		}
	}
}

// variants are spellings of a typed-math key a learner reasonably submits.
func variants(pt pytrack.Part) []string {
	a := strings.TrimSpace(pt.Answer)
	var out []string
	add := func(s string) {
		if s != a && s != "" {
			out = append(out, s)
		}
	}
	v, isConst := constEval(a)
	switch pt.Grading {
	case "numeric", "expression":
		if isConst {
			if strings.Contains(a, "/") && !strings.ContainsAny(a, "()^ ") {
				// A fraction: the same decimal to three places and to more.
				if v != math.Trunc(v) {
					// Rounded half away from zero, as a learner rounds; a
					// rounding that is zero is no answer.
					for _, n := range []int{3, 6} {
						if r := roundHalfAway(v, n); r != 0 {
							add(strconv.FormatFloat(r, 'f', n, 64))
						}
					}
				}
			} else if strings.Contains(a, ".") {
				add(a + "0")                                           // a trailing zero
				add(strings.TrimRight(strings.TrimRight(a, "0"), ".")) // without trailing zeros
				if strings.HasPrefix(a, "0.") {
					add(a[1:]) // ".5"
				} else if strings.HasPrefix(a, "-0.") {
					add("-" + a[2:])
				}
				if f, ok := asFraction(a); ok {
					add(f)
				}
			} else if v == math.Trunc(v) && math.Abs(v) < 1e9 {
				add(a + ".0")
				if n := int64(v); n%1 == 0 && n > 999 && n < 1e7 {
					add(withThousands(n))
				}
			}
			if strings.HasPrefix(a, "-") {
				add("−" + a[1:]) // a typeset minus
			} else if !strings.HasPrefix(a, "+") {
				add("+" + a)
			}
			if !strings.ContainsAny(a, " ") {
				add(" " + a + " ")
			}
		}
		if wordKey(a) {
			add(strings.ToUpper(a))
			if !yesNo(a) {
				add("the " + a)
			}
		} else if pt.Grading == "expression" || !isConst {
			add(texSpelling(a))
			add(strings.ReplaceAll(a, "*", "\\cdot "))
			add(strings.ReplaceAll(strings.ReplaceAll(a, "pi", "π"), "**", "^"))
			add(strings.ReplaceAll(a, "^", "**"))
			add(expToCaret(a))
			add(sup2Re.ReplaceAllString(a, "²"))
			if !strings.Contains(a, "^") && !wordKey(a) {
				// Dropping a product sign after a number: "2*x" as "2x". Not
				// next to a power, where "10^x*ln(10)" would become
				// 10^(x ln 10), and not between letters ("pi*i").
				add(numStarRe.ReplaceAllString(strings.ReplaceAll(a, " ", ""), "$1$2"))
			}
		}
	case "set":
		s := a
		add(strings.ReplaceAll(s, " ", ""))
		if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
			add("\\{" + s[1:len(s)-1] + "\\}")
			inner := strings.Split(s[1:len(s)-1], ",")
			if len(inner) > 1 {
				rev := make([]string, len(inner))
				for i, x := range inner {
					rev[len(inner)-1-i] = strings.TrimSpace(x)
				}
				add("{" + strings.Join(rev, ", ") + "}")
			}
		}
		if s == "∅" || s == "{}" {
			add("{}")
			add("∅")
			add("O/")
			add("\\emptyset")
		}
	case "interval":
		add(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(a, "∞", "oo"), "∪", "U"), "ℝ", "RR"))
		add(strings.ReplaceAll(strings.ReplaceAll(a, "∞", "\\infty"), "∪", "\\cup"))
		add(strings.ReplaceAll(a, " ", ""))
	}
	return dedupe(out)
}

// wordKey reports a key the grader compares as words (grader.IsWordKey).
func wordKey(a string) bool {
	return grader.IsWordKey(strings.TrimSpace(a))
}

func yesNo(a string) bool {
	l := strings.ToLower(strings.TrimSpace(a))
	return l == "yes" || l == "no"
}

// wordWrongs are wrong answers to a word key. A yes/no key accepts an answer
// that starts with its word ("no, it has a cusp"), so its wrong answer is
// the other word.
func wordWrongs(a string) []string {
	if yesNo(a) {
		if strings.EqualFold(a, "yes") {
			return []string{"no", "no, it is not"}
		}
		return []string{"yes", "yes, it is"}
	}
	return []string{a + " zz", "not " + a, "x"}
}

// texSpelling writes a plain key the way its LaTeX would reach the server
// unconverted: x^2 -> x^{2}, sqrt(x) -> \sqrt{x}, a/b -> \frac{a}{b} for
// single tokens.
func texSpelling(a string) string {
	s := supRe.ReplaceAllString(a, "^{$1}")
	s = sqrtTokRe.ReplaceAllString(s, "\\sqrt{$1}")
	s = simpleFracRe.ReplaceAllString(s, "$1\\frac{$2}{$3}$4")
	s = strings.ReplaceAll(s, "pi", "\\pi ")
	return s
}

// expToCaret spells exp(x) as e^(x) and e^x as exp(x).
func expToCaret(a string) string {
	if strings.Contains(a, "exp(") {
		return expRe.ReplaceAllString(a, "e^($1)")
	}
	if ePowRe.MatchString(a) {
		return ePowRe.ReplaceAllString(a, "exp($1)")
	}
	return a
}

var (
	numStarRe    = regexp.MustCompile(`(\d)\*([a-zA-Z(])`)
	sup2Re       = regexp.MustCompile(`\^2\b`)
	supRe        = regexp.MustCompile(`\^(\d+|[a-zA-Z])\b`)
	sqrtTokRe    = regexp.MustCompile(`sqrt\(([^()]*)\)`)
	simpleFracRe = regexp.MustCompile(`(^|[ (+*-])(\d+|[a-zA-Z])/(\d+|[a-zA-Z])($|[ )+*-])`)
	expRe        = regexp.MustCompile(`exp\(([^()]*)\)`)
	ePowRe       = regexp.MustCompile(`\be\^\(([^()]*)\)`)
)

func asFraction(dec string) (string, bool) {
	neg := strings.HasPrefix(dec, "-")
	d := strings.TrimPrefix(dec, "-")
	i := strings.IndexByte(d, '.')
	if i < 0 {
		return "", false
	}
	places := len(d) - i - 1
	if places == 0 || places > 6 {
		return "", false
	}
	num, err := strconv.ParseInt(strings.Replace(d, ".", "", 1), 10, 64)
	if err != nil {
		return "", false
	}
	den := int64(math.Pow10(places))
	g := gcd(num, den)
	if g == 0 || den/g == 1 {
		return "", false
	}
	s := fmt.Sprintf("%d/%d", num/g, den/g)
	if neg {
		s = "-" + s
	}
	return s, true
}

func roundHalfAway(v float64, places int) float64 {
	f := math.Pow(10, float64(places))
	return math.Round(v*f) / f
}

func gcd(a, b int64) int64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func withThousands(n int64) string {
	s := strconv.FormatInt(n, 10)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// wrongs are answers near the key that must grade wrong.
func wrongs(pt pytrack.Part) []string {
	a := strings.TrimSpace(pt.Answer)
	var out []string
	switch pt.Grading {
	case "numeric":
		if v, ok := constEval(a); ok {
			out = append(out, fmtNum(v+1))
			if v != 0 {
				out = append(out, fmtNum(-v), fmtNum(v*10))
			}
			if v != math.Trunc(v) {
				out = append(out, fmtNum(math.Round(v*10)/10+0.1))
			}
		} else {
			out = append(out, a+"+1")
		}
	case "expression":
		if wordKey(a) {
			out = append(out, wordWrongs(a)...)
		} else if v, ok := constEval(a); ok {
			out = append(out, fmtNum(v+1))
			if v != 0 {
				out = append(out, fmtNum(-v))
			}
		} else {
			out = append(out, "("+a+")+1", "2*("+a+")")
			if !strings.HasPrefix(a, "-") {
				out = append(out, "-("+a+")")
			}
		}
		out = append(out, "x")
	case "set":
		out = append(out, "{}", "{0, 999}")
		if strings.HasPrefix(a, "{") && strings.Count(a, ",") >= 1 {
			out = append(out, a[:strings.LastIndex(a, ",")]+"}")
		}
	case "interval":
		out = append(out, "(0, 999)", "ℝ")
		if strings.HasPrefix(a, "[") {
			out = append(out, "("+a[1:])
		} else if strings.HasPrefix(a, "(") && !strings.HasPrefix(a, "(-∞") && !strings.HasPrefix(a, "(-oo") {
			out = append(out, "["+a[1:])
		}
	}
	return dedupe(out)
}

func fmtNum(v float64) string {
	if v == math.Trunc(v) && math.Abs(v) < 1e15 {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func constEval(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	if m := mixedNumRe.FindStringSubmatch(s); m != nil {
		w, _ := strconv.ParseFloat(m[1], 64)
		n, _ := strconv.ParseFloat(m[2], 64)
		d, _ := strconv.ParseFloat(m[3], 64)
		if d == 0 {
			return 0, false
		}
		if w < 0 {
			return w - n/d, true
		}
		return w + n/d, true
	}
	for _, r := range s {
		if !strings.ContainsRune("0123456789.+-*/^() ", r) {
			return 0, false
		}
	}
	if strings.ContainsAny(s, "+-*/^()") && !fracRe.MatchString(s) && !negRe.MatchString(s) {
		return 0, false // left to the grader's own evaluator
	}
	if m := fracRe.FindStringSubmatch(s); m != nil {
		n, _ := strconv.ParseFloat(m[1], 64)
		d, _ := strconv.ParseFloat(m[2], 64)
		if d == 0 {
			return 0, false
		}
		return n / d, true
	}
	v, err := strconv.ParseFloat(s, 64)
	return v, err == nil
}

var (
	mixedNumRe = regexp.MustCompile(`^(-?\d+) (\d+)/(\d+)$`)
	fracRe     = regexp.MustCompile(`^(-?\d+)/(\d+)$`)
	negRe      = regexp.MustCompile(`^-?\d+(\.\d+)?$`)
)

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// perturbOutput changes one digit of a printed output, or appends a token.
func perturbOutput(s string) string {
	for i, r := range s {
		if r >= '0' && r <= '8' {
			return s[:i] + string(r+1) + s[i+1:]
		}
	}
	return s + " x"
}

// CheckGenerators samples every generator concept: the problem's own answer
// grades right as clicked or typed, an empty answer grades wrong, and a
// nearby wrong answer grades wrong.
func (c *Checker) CheckGenerators(opts Options) Report {
	var rep Report
	n := opts.Seeds
	if n <= 0 {
		n = 3
	}
	rng := rand.New(rand.NewSource(1))
	ids := c.Reg.Concepts()
	sort.Strings(ids)
	for _, id := range ids {
		if len(opts.Concepts) > 0 && !contains(opts.Concepts, id) {
			continue
		}
		con := c.DAG.Concept(id)
		gt := ""
		if con != nil {
			gt = con.GradingType
		}
		if gt == "bank" || gt == "python" {
			continue // the bank check covers it
		}
		custom := false
		if gen, err := c.Reg.Get(id); err == nil {
			_, custom = gen.(generator.GradedGenerator)
		}
		for k := 0; k < n; k++ {
			diff := []float64{0.3, 0.6, 0.9}[k%3]
			p, err := c.Reg.GenerateContext(id, generator.GeneratorContext{Difficulty: diff, Seed: rng.Int63(), NoCards: true, NoProofs: true})
			if err != nil || p.Question == "" || grader.IsBankAnswer(p.Answer) || grader.IsProofAnswer(p.Answer) {
				break
			}
			rep.Problems++
			at := func(check, answer, feedback string) {
				rep.Failures = append(rep.Failures, Failure{Module: "data", Bank: "generator", Concept: id, Kind: gt,
					Check: check, Answer: answer, Feedback: feedback, Expected: p.Answer, Custom: custom})
			}
			rep.Checks++
			if res := c.grade(id, p.Question, p.Answer, p.Answer); !res.Correct {
				at("key", p.Answer, res.Feedback)
				break
			}
			if opts.Wrong {
				rep.Checks++
				if res := c.grade(id, p.Question, p.Answer, ""); res.Correct {
					at("empty", "", "graded correct")
				}
				for _, w := range generatorWrongs(gt, p) {
					rep.Checks++
					if res := c.grade(id, p.Question, p.Answer, w); res.Correct {
						at("wrong", w, "graded correct")
					}
				}
			}
			if opts.Variants {
				for _, v := range generatorVariants(gt, p.Answer) {
					rep.Checks++
					if res := c.grade(id, p.Question, p.Answer, v); !res.Correct && !res.Unavailable {
						at("variant", v, res.Feedback)
					}
				}
			}
		}
	}
	return rep
}

func generatorWrongs(gt string, p generator.Problem) []string {
	a := strings.TrimSpace(p.Answer)
	switch grader.GradingType(gt) {
	case grader.GradingMultipleChoice:
		var out []string
		for _, m := range optionRe.FindAllStringSubmatch(p.Question, -1) {
			if strings.TrimSpace(m[2]) != a {
				out = append(out, strings.TrimSpace(m[2]))
			}
		}
		return out
	case grader.GradingNumeric:
		if v, ok := constEval(a); ok {
			out := []string{fmtNum(v + 1)}
			if v != 0 {
				out = append(out, fmtNum(-v))
			}
			return out
		}
	case grader.GradingExpression, grader.GradingSymbolic, grader.GradingPolynomial:
		if wordKey(a) {
			return wordWrongs(a)
		}
		if v, ok := constEval(a); ok {
			return []string{fmtNum(v + 1)}
		}
		if strings.Contains(a, "^") && !strings.ContainsAny(a, "xyzt") {
			// A large constant power (8^48): +1 is below floating-point
			// resolution, so double it instead.
			return []string{"2*(" + a + ")"}
		}
		if !strings.ContainsAny(a, "=,<>") && !strings.Contains(a, "\\") && len(a) < 80 {
			return []string{"(" + a + ")+1"}
		}
	}
	return nil
}

func generatorVariants(gt string, a string) []string {
	a = strings.TrimSpace(a)
	switch grader.GradingType(gt) {
	case grader.GradingNumeric:
		return variants(pytrack.Part{Answer: a, Grading: "numeric"})
	case grader.GradingExpression, grader.GradingSymbolic, grader.GradingPolynomial:
		if strings.ContainsAny(a, "=,<>|") || strings.Contains(a, "\\") {
			return nil
		}
		return variants(pytrack.Part{Answer: a, Grading: "expression"})
	}
	return nil
}

// Load builds a Checker from the enabled modules and the generator registry
// reg (internal/generator/all.Registry()). conceptsDir is data/concepts.
func Load(conceptsDir string, reg *generator.Registry) (*Checker, error) {
	mods, err := modules.Default()
	if err != nil {
		return nil, err
	}
	dag, err := concepts.LoadDirs(conceptsDir, mods.ConceptDirs()...)
	if err != nil {
		return nil, err
	}
	t, err := pytrack.Default()
	if err != nil {
		return nil, err
	}
	return New(mods, dag, reg, t), nil
}

// RunnerReady reports whether code tasks can be run.
func (c *Checker) RunnerReady() error { return c.Track.Runner.Available() }
