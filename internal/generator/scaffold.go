package generator

import (
	"math/rand"
	"os"
	"strconv"

	"github.com/chuma-beep/mathua/internal/books"
)

// Fork change: the order a lesson serves its questions in (FORK.md, "The
// order of practice inside a topic").
//
// A topic's questions climb three scaffold steps, as Math Academy's do:
//
//  1. apply the rule: one step, one idea (bank level 1, or a generated
//     question at the easy end of the ladder);
//  2. worked cases: the book's examples and exercises, several steps
//     (level 2);
//  3. applications: word problems, economic settings, proofs (level 3).
//
// The learner works the lowest step not yet done; a step is done once one
// of its questions has a right answer and the learner has tried the step's
// book exercises, up to the book step cap (BookStepCap, default 3): a
// topic with nine book exercises at a step moves on after three, and the
// rest are served later, in the sampling once every step is done. Within a step the book's own exercises come first, in book
// order, then the bank's other items (untried ones first, at random, so
// variants interleave) and the topic's generated questions. An item that
// unlocks after another is never in the list until it is unlocked. The
// step comes from the learner's record, not from the Learn ladder's
// difficulty, which starts over at every visit: a learner who left off at
// step 3 resumes there, and one who misses stays at the step until a right
// answer (two misses in a row send them to the prerequisites as before).
// Generated questions are served at their step's difficulty. Once every
// step is done, practice and reviews sample across the steps by
// difficulty, as before.

// StepOfLevel is the scaffold step of a bank item's level.
func StepOfLevel(level int) int {
	switch {
	case level == 1:
		return 1
	case level >= 3:
		return 3
	default:
		return 2 // level 2, or unset
	}
}

// StepOfDifficulty is the step a generator difficulty (0.3-1.0) asks for.
// The Learn ladder starts at 0.4 (step 1) and climbs 0.15 per right answer.
func StepOfDifficulty(d float64) int {
	switch {
	case d <= 0.45:
		return 1
	case d < 0.75:
		return 2
	default:
		return 3
	}
}

// DifficultyOfStep is the generator difficulty a step serves at.
func DifficultyOfStep(step int) float64 {
	switch step {
	case 1:
		return 0.35
	case 3:
		return 0.9
	default:
		return 0.6
	}
}

// StepHistory is implemented by a History that knows the steps at which
// the learner has answered the topic's generated questions right.
type StepHistory interface {
	// GeneratedCorrectAt reports a right answer to a generated question of
	// this topic at step or above.
	GeneratedCorrectAt(step int) bool
}

// DefaultBookStepCap is how many book exercises a learner tries at a
// scaffold step before the step can be done, when no course sets one.
const DefaultBookStepCap = 3

// BookStepCapEnv overrides DefaultBookStepCap for every topic; 0 or a
// negative number switches the cap off (every book exercise at a step comes
// before the next step).
const BookStepCapEnv = "MATHUA_BOOK_STEP_CAP"

// EnvBookStepCap is the cap MATHUA_BOOK_STEP_CAP sets, or the default. A
// value of 0 or below means no cap and is returned as 0.
func EnvBookStepCap() int {
	if v, err := strconv.Atoi(os.Getenv(BookStepCapEnv)); err == nil {
		if v < 0 {
			return 0
		}
		return v
	}
	return DefaultBookStepCap
}

// BookStepCapper is implemented by a History that knows the cap for its
// topic (a course's book_step_cap, engine/history.go).
type BookStepCapper interface {
	BookStepCap() int
}

// BookStepCapOf is the cap for the topic h is about: its own, when it
// knows one, else MATHUA_BOOK_STEP_CAP or the default. 0 means no cap.
func BookStepCapOf(h History) int {
	if c, ok := h.(BookStepCapper); ok {
		return c.BookStepCap()
	}
	return EnvBookStepCap()
}

// ScaffoldSource is a bank generator that lists its servable items for a
// learner as scaffold steps (internal/pytrack).
type ScaffoldSource interface {
	ScaffoldItems(ctx GeneratorContext) []ScaffoldItem
}

// ScaffoldItemSource is a bank generator that describes one of its items,
// by id, as a scaffold step: for an item filed under another topic.
type ScaffoldItemSource interface {
	ItemScaffold(itemID string, ctx GeneratorContext) (ScaffoldItem, bool)
}

// ScaffoldItem is one question a topic can serve.
type ScaffoldItem struct {
	Step      int
	Book      bool
	Attempted bool
	Correct   bool
	// Source is the item's book (internal/books id), "" for an unsourced
	// item.
	Source string
	// Linked marks an item of a topic tied to this one by `encompasses`:
	// it is served only as a parallel book's item (ArrangeBooks drops it
	// otherwise), never in the sampling once the steps are done.
	Linked bool
	Serve  func() (Problem, bool)
}

// Scaffold is everything a topic can serve to one learner now.
type Scaffold struct {
	// Items are the servable bank items, in bank order.
	Items []ScaffoldItem
	// Generated serves the topic's generated question at a step, nil for a
	// topic that is a bank only.
	Generated func(step int) Problem
	// GeneratedDone[s] is a right generated answer at step s or above.
	GeneratedDone [4]bool
	// Share is the chance a bank item is served rather than a generated
	// question when a step has both.
	Share float64
	// BookCap is how many book exercises at a step the learner tries before
	// the step can be done and the rest stop coming first (BookStepCapOf);
	// 0 is no cap: every one of them comes before the next step.
	BookCap int
	// Main is the course's own book (ArrangeBooks); "" takes every book
	// exercise without a source as the main book's, as before books had
	// names.
	Main string
	// Books are the parallel books, in the order they take turns: items of
	// other books on the topic, served after the main book's and before the
	// practice set. Empty: the topic has none and the step is served as it
	// always was.
	Books []string
	// ParallelCap is how many items of each parallel book (and of the main
	// book's items that are not book exercises) a step serves before the
	// practice set; 0 is no cap.
	ParallelCap int
}

// Tiers of a step (Problem.Tier).
const (
	TierMain     = "main"
	TierParallel = "parallel"
	TierPractice = "practice"
	TierRetry    = "retry"
)

// DefaultParallelBookCap is how many items of each parallel book a step
// serves before the practice set, when no course sets one.
const DefaultParallelBookCap = 3

// ParallelBookCapEnv overrides DefaultParallelBookCap; 0 or below is no cap.
const ParallelBookCapEnv = "MATHUA_PARALLEL_BOOK_CAP"

// EnvParallelBookCap is the cap MATHUA_PARALLEL_BOOK_CAP sets, or the
// default; 0 means no cap.
func EnvParallelBookCap() int {
	if v, err := strconv.Atoi(os.Getenv(ParallelBookCapEnv)); err == nil {
		if v < 0 {
			return 0
		}
		return v
	}
	return DefaultParallelBookCap
}

// BookPolicy is what the courses that brought the learner to a topic say
// about its books: their texts in priority order (book ids), the parallel
// book cap and the books whose notation clashes with the course's.
type BookPolicy struct {
	Texts       []string
	ParallelCap int
	Exclude     map[string]bool
}

// BookPolicyHistory is implemented by a History that knows the course
// context of its topic (engine/history.go).
type BookPolicyHistory interface {
	BookPolicy() BookPolicy
}

// BookPolicyOf is h's policy, or no texts, MATHUA_PARALLEL_BOOK_CAP and
// nothing excluded.
func BookPolicyOf(h History) BookPolicy {
	if b, ok := h.(BookPolicyHistory); ok {
		return b.BookPolicy()
	}
	return BookPolicy{ParallelCap: EnvParallelBookCap()}
}

// ArrangeBooks sorts the items into the course's own book, parallel books
// and the rest, by policy p. own is the topic's own book (a bank topic's
// source), the main book when no course text has items here and no book
// exercise names one. It drops excluded books' items (unless the book is
// the main one) and linked items that are not another book's.
func (s *Scaffold) ArrangeBooks(p BookPolicy, own string) {
	have := map[string]bool{}
	firstBook := ""
	for _, it := range s.Items {
		if it.Linked || it.Source == "" {
			continue
		}
		have[it.Source] = true
		if it.Book && firstBook == "" {
			firstBook = it.Source
		}
	}
	s.Main = ""
	for _, t := range p.Texts {
		if have[t] {
			s.Main = t
			break
		}
	}
	if s.Main == "" {
		s.Main = firstBook
	}
	if s.Main == "" && have[own] {
		s.Main = own
	}
	kept := s.Items[:0]
	seen := map[string]bool{}
	var parallel []string
	for _, it := range s.Items {
		other := it.Source != "" && it.Source != s.Main
		if it.Linked && !other {
			continue
		}
		if other && p.Exclude[it.Source] {
			continue
		}
		if other && !seen[it.Source] {
			seen[it.Source] = true
			parallel = append(parallel, it.Source)
		}
		kept = append(kept, it)
	}
	s.Items = kept
	s.Books = books.Order(parallel, p.Texts)
	s.ParallelCap = p.ParallelCap
}

// isMain reports an item of the course's own book: its source, or a book
// exercise with no source (or with no main book known: before ArrangeBooks,
// every book exercise is the main book's, as it always was).
func (s *Scaffold) isMain(it ScaffoldItem) bool {
	if it.Linked {
		return false
	}
	if it.Source == "" || s.Main == "" {
		return it.Book
	}
	return it.Source == s.Main
}

// group is the quota group an item belongs to before the practice set: a
// parallel book's id, "\x00main" for the main book's items that are not
// book exercises (only when the topic has parallel books), or "".
func (s *Scaffold) group(it ScaffoldItem) string {
	if len(s.Books) == 0 {
		return ""
	}
	if s.isMain(it) {
		if it.Book {
			return ""
		}
		return mainGroup
	}
	return it.Source
}

const mainGroup = "\x00main"

// mainBook is a book exercise of the course's own book: the items the
// book step cap counts.
func (s *Scaffold) mainBook(it ScaffoldItem) bool { return it.Book && s.isMain(it) }

// bookTried is how many book exercises at a step have been tried, and how
// many there are.
func (s *Scaffold) bookTried(step int) (tried, total int) {
	for _, it := range s.Items {
		if s.mainBook(it) && it.Step == step {
			total++
			if it.Attempted {
				tried++
			}
		}
	}
	return tried, total
}

// booksMet reports whether the learner has tried enough of a step's book
// exercises: all of them, or BookCap of them when a cap is set; and of each
// parallel book's items, up to ParallelCap of them.
func (s *Scaffold) booksMet(step int) bool {
	tried, total := s.bookTried(step)
	if s.BookCap > 0 && total > s.BookCap {
		if tried < s.BookCap {
			return false
		}
	} else if tried < total {
		return false
	}
	if len(s.Books) == 0 {
		return true
	}
	gt, gn := map[string]int{}, map[string]int{}
	for _, it := range s.Items {
		if it.Step != step {
			continue
		}
		if g := s.group(it); g != "" {
			gn[g]++
			if it.Attempted {
				gt[g]++
			}
		}
	}
	for g, n := range gn {
		need := n
		if s.ParallelCap > 0 && need > s.ParallelCap {
			need = s.ParallelCap
		}
		if gt[g] < need {
			return false
		}
	}
	return true
}

// has reports which steps have something to serve.
func (s *Scaffold) has() [4]bool {
	var h [4]bool
	for _, it := range s.Items {
		h[it.Step] = true
	}
	if s.Generated != nil {
		h[1], h[2], h[3] = true, true, true
	}
	return h
}

// Done reports whether a step is done: a right answer at it or above, the
// step's book exercises tried, up to BookCap of them, and each parallel
// book's items tried, up to ParallelCap of them. A parallel book's right
// answer counts like the main book's.
func (s *Scaffold) Done(step int) bool {
	if !s.booksMet(step) {
		return false
	}
	if s.GeneratedDone[step] {
		return true
	}
	for _, it := range s.Items {
		if it.Correct && it.Step >= step {
			return true
		}
	}
	return false
}

// Frontier is the lowest step with something to serve that is not done,
// or 0 when every step is.
func (s *Scaffold) Frontier() int {
	h := s.has()
	for step := 1; step <= 3; step++ {
		if h[step] && !s.Done(step) {
			return step
		}
	}
	return 0
}

// Path is a step's sources in the order it serves them: the main book (when
// it has items at the step), the parallel books with items at it, then ""
// for the practice set (generated questions and unsourced items).
func (s *Scaffold) Path(step int) []string {
	var out []string
	hasMain, practice := false, s.Generated != nil
	at := map[string]bool{}
	for _, it := range s.Items {
		if it.Step != step {
			continue
		}
		switch {
		case s.isMain(it):
			hasMain = true
		case it.Source != "":
			at[it.Source] = true
		default:
			practice = true
		}
	}
	if hasMain && s.Main != "" {
		out = append(out, s.Main)
	}
	for _, b := range s.Books {
		if at[b] {
			out = append(out, b)
		}
	}
	if practice {
		out = append(out, "")
	}
	return out
}

// Pick serves the next question of a lesson at the frontier step. served
// is what this request has already served (a
// lesson asks for several questions at once): nothing is served twice, and
// when a step has nothing left the lesson goes on to the next step. ok is
// false when every step is done: the caller then samples across the steps
// as it always has.
func (s *Scaffold) Pick(served map[string]bool) (Problem, bool) {
	step := s.Frontier()
	if step == 0 {
		return Problem{}, false
	}
	for ; step <= 3; step++ {
		if p, ok := s.pickAt(step, served); ok {
			p.Step = step
			p.StepPath = s.Path(step)
			return p, true
		}
	}
	return Problem{}, false
}

// pickAt serves one question at a step that this request has not served:
// the main book's exercises first, in book order (an untried one, or a
// missed one half the time, as generator.PickBook); then, when the topic has
// parallel books, the main book's other items and each parallel book's
// items, taking turns, up to ParallelCap of each; then the practice set:
// the bank's other items, untried first and at random, and the generated
// questions. Once BookCap of the step's book exercises (or ParallelCap of a
// book's items) have been tried, the untried rest no longer come first:
// they join the step's untried items.
func (s *Scaffold) pickAt(step int, served map[string]bool) (Problem, bool) {
	try := func(it ScaffoldItem, tier string) (Problem, bool) {
		p, ok := it.Serve()
		if !ok || p.Question == "" || served[p.Question] {
			return Problem{}, false
		}
		p.SourceBook, p.Tier = it.Source, tier
		return p, true
	}
	var untriedBooks, missedBooks, fresh, others []ScaffoldItem
	capped := s.BookCap > 0
	if capped {
		tried, _ := s.bookTried(step)
		capped = tried >= s.BookCap
	}
	// Quota groups: the main book's other items and each parallel book.
	type quota struct {
		count   int // tried, or served in this request
		untried []ScaffoldItem
	}
	groups := map[string]*quota{}
	var grouped []ScaffoldItem
	for _, it := range s.Items {
		if it.Step != step {
			continue
		}
		if g := s.group(it); g != "" {
			q := groups[g]
			if q == nil {
				q = &quota{}
				groups[g] = q
			}
			switch {
			case it.Attempted:
				q.count++
				others = append(others, it)
			default:
				if p, ok := it.Serve(); ok && served[p.Question] {
					q.count++
					continue
				}
				q.untried = append(q.untried, it)
				grouped = append(grouped, it)
			}
			continue
		}
		switch {
		case it.Book && !it.Attempted && capped:
			fresh = append(fresh, it)
		case it.Book && !it.Correct && !it.Attempted:
			untriedBooks = append(untriedBooks, it)
		case it.Book && !it.Correct:
			// A missed exercise comes back half the time, and is in the
			// pool with the rest the other half.
			missedBooks = append(missedBooks, it)
			others = append(others, it)
		case it.Book:
			others = append(others, it)
		case !it.Attempted:
			fresh = append(fresh, it)
		default:
			others = append(others, it)
		}
	}
	for _, it := range untriedBooks {
		if p, ok := try(it, TierMain); ok {
			return p, true
		}
	}
	if len(missedBooks) > 0 && rand.Intn(2) == 0 {
		for _, i := range rand.Perm(len(missedBooks)) {
			if p, ok := try(missedBooks[i], TierRetry); ok {
				return p, true
			}
		}
	}
	if len(groups) > 0 {
		open := func(g string) bool {
			q := groups[g]
			return q != nil && len(q.untried) > 0 && (s.ParallelCap <= 0 || q.count < s.ParallelCap)
		}
		serveGroup := func(g, tier string) (Problem, bool) {
			q := groups[g]
			// A book's exercises in book order, its other items at random.
			var books, rest []ScaffoldItem
			for _, it := range q.untried {
				if it.Book {
					books = append(books, it)
				} else {
					rest = append(rest, it)
				}
			}
			for _, it := range books {
				if p, ok := try(it, tier); ok {
					return p, true
				}
			}
			for _, i := range rand.Perm(len(rest)) {
				if p, ok := try(rest[i], tier); ok {
					return p, true
				}
			}
			return Problem{}, false
		}
		if open(mainGroup) {
			if p, ok := serveGroup(mainGroup, TierMain); ok {
				return p, true
			}
		}
		// The parallel books take turns: the open book with the fewest
		// served at this step, the earlier in Books on a tie.
		tried := map[string]bool{}
		for {
			best := ""
			for _, b := range s.Books {
				if tried[b] || !open(b) {
					continue
				}
				if best == "" || groups[b].count < groups[best].count {
					best = b
				}
			}
			if best == "" {
				break
			}
			tried[best] = true
			if p, ok := serveGroup(best, TierParallel); ok {
				return p, true
			}
		}
		// Past their quotas, a book's untried items join the step's.
		fresh = append(fresh, grouped...)
	}
	gen := func() (Problem, bool) {
		if s.Generated == nil {
			return Problem{}, false
		}
		for i := 0; i < 4; i++ {
			if p := s.Generated(step); p.Question != "" && !served[p.Question] {
				p.Tier = TierPractice
				return p, true
			}
		}
		return Problem{}, false
	}
	bankFirst := s.Generated == nil || rand.Float64() < s.Share
	if !bankFirst {
		if p, ok := gen(); ok {
			return p, true
		}
	}
	for _, pool := range [][]ScaffoldItem{fresh, others} {
		for _, i := range rand.Perm(len(pool)) {
			if p, ok := try(pool[i], TierPractice); ok {
				return p, true
			}
		}
	}
	if p, ok := gen(); ok {
		return p, true
	}
	for _, it := range missedBooks {
		if p, ok := try(it, TierRetry); ok {
			return p, true
		}
	}
	return Problem{}, false
}
