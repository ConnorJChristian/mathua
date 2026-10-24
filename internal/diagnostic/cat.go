package diagnostic

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/generator"
)

type State string

const (
	StateProbing State = "probing"
	StateDone    State = "done"
)

const (
	// minTotalQuestions is the point at which convergence is *considered*, not a
	// floor it must reach. shouldStop below requires the cover to be finished AND
	// this count, but NextQuestion ends the session outright when the covering set
	// runs out, and that path does not consult this constant. Every cover concept
	// needs two probes and no concept is asked twice, so a scope whose closure is
	// smaller than 13 concepts simply cannot reach 25 — and nothing pads the gap,
	// because repeating settled concepts would invent evidence.
	//
	// It was described as "the floor" here and in docs/architecture.md until a
	// five-concept scope was found asking ten questions while reporting
	// min_total: 25. Both now say what the code does.
	minTotalQuestions = 25
	// maxTotalQuestions hard caps an adaptive session (supplemental included).
	maxTotalQuestions = 45
	// coverSize is the target size of the compressed covering set.
	coverSize = 30
	// beliefThreshold is the frontier cutoff: belief >= threshold is "known".
	beliefThreshold = 0.6
	// confidenceThreshold gates supplemental re-probes: cover concepts
	// settled below this confidence get one extra probe each.
	confidenceThreshold = 0.7
	// Bayes evidence weights (conflict resolution): a correct answer is
	// +evidenceCorrectWeight toward prerequisites, an incorrect answer is
	// −evidenceIncorrectWeight toward dependents; conflicting prior
	// evidence discounts both directions by conflictDiscount.
	evidenceCorrectWeight   = 0.3
	evidenceIncorrectWeight = 0.3
	conflictDiscount        = 0.5
	// conditionalStripFactor scales back "conditionally completed"
	// prerequisites (belief in [threshold, threshold+0.15)) on a miss —
	// the frontier falls backwards immediately on struggle.
	conditionalStripFactor = 0.8
	// maxSupplementalProbes caps extra re-probes per concept.
	maxSupplementalProbes = 1
	// inferKnown / inferUnknown: a cover concept never asked whose belief
	// evidence elsewhere has pushed past these is settled by inference
	// (fork change).
	inferKnown   = 0.85
	inferUnknown = 0.15
	// ancestorEvidenceWeight: a correct answer is evidence for every
	// prerequisite below it, not just the direct ones (fork change: one
	// level of propagation at 0.15 left arithmetic at 0.5 under a correct
	// calculus answer, so it was asked).
	ancestorEvidenceWeight = 0.35
	// dependentEvidenceWeight: a miss is evidence against everything built
	// on it, decaying with distance (fork change).
	dependentEvidenceWeight = 0.3
)

type Session struct {
	sync.Mutex

	ID        string
	StudentID string
	State     State
	Attempts  []Attempt
	order     []*concepts.Concept

	// LastProblem stores the current problem the student saw.
	LastProblem     *generator.Problem
	LastConceptID   string
	LastConceptName string
	// prevProblem tracks the superseded pending question so a "silly
	// mistake" retry can take the student back to it (answer responses
	// always carry the next staged question; the voidable attempt belongs
	// to the previous one).
	prevProblem     *generator.Problem
	prevConceptID   string
	prevConceptName string

	// Per-concept student model (belief 0-1 + confidence 0-1).
	beliefs      map[string]float64
	confidence   map[string]float64
	probeCounts  map[string]int
	correctCount map[string]int
	totalCount   map[string]int
	totalAsked   int
	doneSet      map[string]bool
	// supplementalCount tracks extra re-probes per concept (cap enforced).
	supplementalCount map[string]int
	// retryUsed marks that the pending question's "silly mistake" retry was
	// consumed; reset whenever NextQuestion serves a new problem.
	retryUsed bool
	// targets are the goal concepts of a placement (a course's plan topics),
	// fork change: the cover samples them so the test probes the material
	// the learner is placing into, not just the foundations under it.
	targets map[string]bool
}

type Attempt struct {
	ConceptID      string
	Correct        bool
	Fast           bool
	ElapsedSeconds float64
	Timestamp      time.Time
	// DontKnow marks an admitted unknown ("I don't know" button): clean
	// negative evidence, as opposed to a failed attempt which may be a slip.
	DontKnow bool
	// BeliefSnapshot captures concept beliefs before this attempt's update
	// so a voided "silly mistake" retry can restore exact prior state.
	BeliefSnapshot map[string]float64
}

// Progress is the backend truth for the progress bar (MA parity: no exact
// total is promised — the Diagnostic is adaptive). CoverDone/CoverSize is
// the monotonic coverage fraction (never decreases); Answered counts
// questions answered; EstimatedTotal is a legacy hint kept for compat and
// must NOT be displayed as a promise.
type Progress struct {
	Answered       int  `json:"answered"`
	EstimatedTotal int  `json:"estimated_total"`
	MinTotal       int  `json:"min_total"`
	MaxTotal       int  `json:"max_total"`
	CoverDone      int  `json:"cover_done"`
	CoverSize      int  `json:"cover_size"`
	Done           bool `json:"done"`
}

type Engine struct {
	dag      *concepts.DAG
	registry *generator.Registry
}

func NewEngine(dag *concepts.DAG, reg *generator.Registry) *Engine {
	return &Engine{dag: dag, registry: reg}
}

// Snapshot is the durable form of a Session.
//
// It exists because the adaptive model is private: encoding a Session directly
// would persist its id and attempts and silently drop every unexported field,
// so a restored run would believe all concepts sit at the 0.5 prior and would
// re-ask questions the learner had already answered. The snapshot names each
// field, and Decode rebuilds the path through the DAG so the session keeps
// pointing at real concepts rather than at strings.
type Snapshot struct {
	ID        string    `json:"id"`
	StudentID string    `json:"student_id"`
	State     State     `json:"state"`
	Attempts  []Attempt `json:"attempts"`

	// Order is the session's frozen path, by concept id.
	Order []string `json:"order"`

	LastProblem     *generator.Problem `json:"last_problem,omitempty"`
	LastConceptID   string             `json:"last_concept_id"`
	LastConceptName string             `json:"last_concept_name"`
	PrevProblem     *generator.Problem `json:"prev_problem,omitempty"`
	PrevConceptID   string             `json:"prev_concept_id"`
	PrevConceptName string             `json:"prev_concept_name"`

	Beliefs           map[string]float64 `json:"beliefs"`
	Confidence        map[string]float64 `json:"confidence"`
	ProbeCounts       map[string]int     `json:"probe_counts"`
	CorrectCount      map[string]int     `json:"correct_count"`
	TotalCount        map[string]int     `json:"total_count"`
	DoneSet           map[string]bool    `json:"done_set"`
	SupplementalCount map[string]int     `json:"supplemental_count"`
	TotalAsked        int                `json:"total_asked"`
	RetryUsed         bool               `json:"retry_used"`
	Targets           []string           `json:"targets,omitempty"`
}

// Encode renders the session for storage. The lock is held across marshaling,
// not merely while copying: an answer arriving mid-encode must not produce a
// snapshot that pairs a new attempt count with old beliefs.
//
// Only the newest attempt keeps its belief snapshot. RetryServe restores the
// last miss, and nothing reads an older one, so carrying one per attempt would
// write a full belief map for every question answered — a megabyte or more by
// the end of a run, rewritten on every answer.
func (s *Session) Encode() ([]byte, error) {
	s.Lock()
	defer s.Unlock()
	order := make([]string, 0, len(s.order))
	for _, c := range s.order {
		order = append(order, c.ID)
	}
	attempts := make([]Attempt, len(s.Attempts))
	copy(attempts, s.Attempts)
	for i := 0; i < len(attempts)-1; i++ {
		attempts[i].BeliefSnapshot = nil
	}
	snap := Snapshot{
		ID:                s.ID,
		StudentID:         s.StudentID,
		State:             s.State,
		Attempts:          attempts,
		Order:             order,
		LastProblem:       s.LastProblem,
		LastConceptID:     s.LastConceptID,
		LastConceptName:   s.LastConceptName,
		PrevProblem:       s.prevProblem,
		PrevConceptID:     s.prevConceptID,
		PrevConceptName:   s.prevConceptName,
		Beliefs:           s.beliefs,
		Confidence:        s.confidence,
		ProbeCounts:       s.probeCounts,
		CorrectCount:      s.correctCount,
		TotalCount:        s.totalCount,
		DoneSet:           s.doneSet,
		SupplementalCount: s.supplementalCount,
		TotalAsked:        s.totalAsked,
		RetryUsed:         s.retryUsed,
	}
	for id := range s.targets {
		snap.Targets = append(snap.Targets, id)
	}
	sort.Strings(snap.Targets)
	return json.Marshal(snap)
}

// Decode rebuilds a session from a snapshot written by Encode.
//
// A concept the current corpus no longer knows is dropped from the path rather
// than restored as a nil pointer; if none survive, the caller must treat the
// session as gone instead of resuming a run that can never serve a question.
func (e *Engine) Decode(data []byte) (*Session, error) {
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("decode diagnostic snapshot: %w", err)
	}
	if snap.ID == "" {
		return nil, fmt.Errorf("diagnostic snapshot has no id")
	}
	order := make([]*concepts.Concept, 0, len(snap.Order))
	for _, id := range snap.Order {
		c := e.dag.Concept(id)
		if c == nil {
			continue
		}
		order = append(order, c)
	}
	if len(order) == 0 {
		return nil, fmt.Errorf("diagnostic snapshot %s has no concepts the corpus still knows", snap.ID)
	}
	state := snap.State
	if state == "" {
		state = StateProbing
	}
	s := &Session{
		ID:                snap.ID,
		StudentID:         snap.StudentID,
		State:             state,
		Attempts:          snap.Attempts,
		order:             order,
		LastProblem:       snap.LastProblem,
		LastConceptID:     snap.LastConceptID,
		LastConceptName:   snap.LastConceptName,
		prevProblem:       snap.PrevProblem,
		prevConceptID:     snap.PrevConceptID,
		prevConceptName:   snap.PrevConceptName,
		beliefs:           orZero(snap.Beliefs),
		confidence:        orZero(snap.Confidence),
		probeCounts:       orZeroInt(snap.ProbeCounts),
		correctCount:      orZeroInt(snap.CorrectCount),
		totalCount:        orZeroInt(snap.TotalCount),
		doneSet:           orFalse(snap.DoneSet),
		supplementalCount: orZeroInt(snap.SupplementalCount),
		totalAsked:        snap.TotalAsked,
		retryUsed:         snap.RetryUsed,
	}
	if len(snap.Targets) > 0 {
		s.targets = make(map[string]bool, len(snap.Targets))
		for _, id := range snap.Targets {
			s.targets[id] = true
		}
	}
	return s, nil
}

func orZero(m map[string]float64) map[string]float64 {
	if m == nil {
		return map[string]float64{}
	}
	return m
}

func orZeroInt(m map[string]int) map[string]int {
	if m == nil {
		return map[string]int{}
	}
	return m
}

func orFalse(m map[string]bool) map[string]bool {
	if m == nil {
		return map[string]bool{}
	}
	return m
}

func (e *Engine) Start() *Session {
	order := e.dag.Order()
	return e.StartWithPath(order)
}

// StartWithPath begins an adaptive session over the given path (topo order).
// It compresses the path into a minimal covering set and picks the first
// question by info gain.
func (e *Engine) StartWithPath(path []*concepts.Concept) *Session {
	if len(path) == 0 {
		return &Session{State: StateDone, order: path}
	}
	s := &Session{
		State:             StateProbing,
		order:             path,
		beliefs:           make(map[string]float64, len(path)),
		confidence:        make(map[string]float64, len(path)),
		probeCounts:       make(map[string]int, len(path)),
		correctCount:      make(map[string]int, len(path)),
		totalCount:        make(map[string]int, len(path)),
		doneSet:           make(map[string]bool, len(path)),
		supplementalCount: make(map[string]int),
	}
	for _, c := range path {
		s.beliefs[c.ID] = 0.5
	}
	return s
}

// StartWithGoal begins a placement over path toward goal concepts (a
// course's plan topics), fork change. The goal topics are sampled into the
// covering set so the test probes the material being placed into; without
// them the cover was every root of the path plus evenly spaced concepts, and
// a graduate course's path has dozens of roots (single-digit addition, the
// number line), which is where a quarter of the questions went.
func (e *Engine) StartWithGoal(path []*concepts.Concept, goal []string) *Session {
	s := e.StartWithPath(path)
	if s.State == StateDone || len(goal) == 0 {
		return s
	}
	inPath := make(map[string]bool, len(path))
	for _, c := range path {
		inPath[c.ID] = true
	}
	s.targets = map[string]bool{}
	for _, id := range goal {
		if inPath[id] {
			s.targets[id] = true
		}
	}
	if len(s.targets) == 0 {
		s.targets = nil
	}
	return s
}

// compressedCover returns the smallest covering set (see coverOf).
func (e *Engine) compressedCover(s *Session) []*concepts.Concept {
	return s.compressedCoverSafe()
}

func (s *Session) index(id string) int {
	for i, c := range s.order {
		if c.ID == id {
			return i
		}
	}
	return len(s.order)
}

// entropy is the binary entropy of a belief p (uncertainty).
func entropy(p float64) float64 {
	if p <= 0 || p >= 1 {
		return 0
	}
	return -(p*math.Log2(p) + (1-p)*math.Log2(1-p))
}

// infoGain scores a candidate concept: uncertainty times coverage influence.
// Coverage = how many other cover concepts it reaches through prereqs/dependents.
func (e *Engine) infoGain(s *Session, c *concepts.Concept, coverSet map[string]bool) float64 {
	unc := entropy(s.beliefs[c.ID])
	influence := 1.0
	for _, dep := range e.dag.DependentsOf(c.ID) {
		if coverSet[dep.ID] {
			influence += 0.25
		}
	}
	// A right answer settles everything below it, so a concept with many
	// unsettled cover concepts beneath it is worth asking first (fork change).
	seen := map[string]bool{c.ID: true}
	stack := e.dag.PrereqsOf(c.ID)
	for len(stack) > 0 {
		pr := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[pr.ID] {
			continue
		}
		seen[pr.ID] = true
		if coverSet[pr.ID] && !s.doneSet[pr.ID] {
			influence += 0.15
		}
		stack = append(stack, e.dag.PrereqsOf(pr.ID)...)
	}
	if s.targets[c.ID] {
		influence += 0.25
	}
	return unc * influence
}

// pickByInfoGain selects the highest-gain cover concept not yet settled.
func (e *Engine) pickByInfoGain(s *Session) string {
	cover := e.compressedCover(s)
	coverSet := make(map[string]bool, len(cover))
	for _, c := range cover {
		coverSet[c.ID] = true
	}
	best := ""
	bestScore := -1.0
	for _, c := range cover {
		if s.doneSet[c.ID] {
			continue
		}
		score := e.infoGain(s, c, coverSet)
		if score > bestScore {
			bestScore = score
			best = c.ID
		}
	}
	return best
}

func (e *Engine) NextQuestion(s *Session) (*generator.Problem, string, error) {
	s.Lock()
	defer s.Unlock()
	if s.State == StateDone {
		s.LastProblem = nil
		return nil, "", nil
	}
	cid := e.pickByInfoGain(s)
	if cid == "" {
		// Cover exhausted: run the supplemental diagnostic for
		// low-confidence concepts before declaring done.
		cid = e.supplementalDiagnostic(s)
	}
	if cid == "" {
		s.State = StateDone
		s.LastProblem = nil
		return nil, "", nil
	}
	concept := s.order[s.index(cid)]
	s.probeCounts[cid]++
	// Vary difficulty: start low, increase on later probes.
	difficulty := 0.3
	if s.probeCounts[cid] >= 2 {
		difficulty = 0.6
	}
	if s.probeCounts[cid] >= 3 {
		difficulty = 0.8
	}
	p, err := e.registry.GenerateContext(cid, generator.GeneratorContext{Difficulty: clampDifficulty(difficulty), NoCards: true})
	if err != nil {
		return nil, "", err
	}
	// Keep the superseded pending question: a retry voids its attempt and
	// takes the student back to it (responses always stage the next one).
	s.prevProblem, s.prevConceptID, s.prevConceptName = s.LastProblem, s.LastConceptID, s.LastConceptName
	s.LastProblem = &p
	s.LastConceptID = cid
	s.LastConceptName = concept.Label
	s.retryUsed = false
	return s.LastProblem, cid, nil
}

// retryEligibleForLocked reports whether a recorded miss looks like a slip
// worth offering back. Any one of three triggers qualifies:
//  1. established knowledge — pre-answer belief at or above the confidence
//     threshold (the model is surprised);
//  2. level drop — the staged next concept sits earlier in path order than
//     the missed one (the miss demoted the student; heuristic: path order is
//     topological, foundations first, not a strict difficulty scale);
//  3. really simple — the missed question was served at base difficulty
//     (first probe, 0.3 tier).
//
// Admits ("I don't know") never qualify: an admit is not a slip.
// Caller must hold s.Lock. Genuine unknowns on hard material get no offer.
func (e *Engine) retryEligibleForLocked(s *Session, missedCID, nextCID string) bool {
	if s.State == StateDone || s.retryUsed {
		return false
	}
	n := len(s.Attempts)
	if n == 0 {
		return false
	}
	last := s.Attempts[n-1]
	if last.ConceptID != missedCID || last.Correct || last.DontKnow {
		return false
	}
	if prior, ok := last.BeliefSnapshot[last.ConceptID]; ok && prior >= confidenceThreshold {
		return true
	}
	// A level drop counts only when the model leaned towards "known": after
	// a miss the placement now descends by design (fork change), so a drop
	// after a miss on a concept it had no opinion about is not a surprise.
	if nextCID != "" && s.index(nextCID) < s.index(missedCID) && last.BeliefSnapshot[last.ConceptID] > 0.5 {
		return true
	}
	// When the staged next question re-probes the missed concept, staging
	// already counted that probe; the missed one is the probe before it
	// (fork fix: a first miss re-probed at once never qualified).
	probes := s.probeCounts[missedCID]
	if nextCID == missedCID {
		probes--
	}
	return probes <= 1
}

// RetryAvailableFor reports whether the missed concept's latest miss may be
// retried ("silly mistake" offer), given the staged next concept ("" when
// none is staged yet). Safe for concurrent use.
func (e *Engine) RetryAvailableFor(s *Session, missedCID, nextCID string) bool {
	s.Lock()
	defer s.Unlock()
	return e.retryEligibleForLocked(s, missedCID, nextCID)
}

// RetryServe voids the just-superseded question's recorded miss for a
// "silly mistake" retry and restores that question as pending: the slip is
// popped, counts decremented, beliefs restored from the pre-answer snapshot,
// and doneSet recomputed. Only the superseded pending question can come
// back, only when its miss passes the methodology gate, and only once
// (retryUsed). Returns the restored problem, concept ID and name.
func (e *Engine) RetryServe(s *Session, conceptID string) (*generator.Problem, string, string, error) {
	s.Lock()
	defer s.Unlock()
	if s.State == StateDone {
		return nil, "", "", fmt.Errorf("diagnostic session already complete")
	}
	if conceptID == "" {
		return nil, "", "", fmt.Errorf("concept_id required")
	}
	if s.retryUsed {
		return nil, "", "", fmt.Errorf("retry already used for this question")
	}
	n := len(s.Attempts)
	if n == 0 || s.prevProblem == nil {
		return nil, "", "", fmt.Errorf("nothing to retry")
	}
	last := s.Attempts[n-1]
	if last.ConceptID != conceptID || last.ConceptID != s.prevConceptID {
		return nil, "", "", fmt.Errorf("latest attempt does not match the retryable question")
	}
	if last.Correct {
		return nil, "", "", fmt.Errorf("correct answers cannot be retried")
	}
	if !e.retryEligibleForLocked(s, last.ConceptID, s.LastConceptID) {
		return nil, "", "", fmt.Errorf("retry not available: the miss does not look like a slip")
	}
	s.Attempts = s.Attempts[:n-1]
	s.totalCount[last.ConceptID]--
	s.totalAsked--
	s.beliefs = last.BeliefSnapshot
	s.doneSet[last.ConceptID] = s.totalCount[last.ConceptID] >= 2
	s.LastProblem, s.LastConceptID, s.LastConceptName = s.prevProblem, s.prevConceptID, s.prevConceptName
	s.prevProblem, s.prevConceptID, s.prevConceptName = nil, "", ""
	s.retryUsed = true
	return s.LastProblem, s.LastConceptID, s.LastConceptName, nil
}

// SettleCurrent marks the current question's concept as settled without
// recording an answer — the skip escape hatch for questions that can't be
// answered (bad content, repeated submit failures). No belief update, no
// attempt, no XP: settling (rather than merely re-asking) is what moves CAT
// past a poisoned transition, since NextQuestion is deterministic on state.
// Returns the settled concept ID ("" if nothing pending).
func (e *Engine) SettleCurrent(s *Session) string {
	s.Lock()
	defer s.Unlock()
	cid := s.LastConceptID
	if cid == "" || s.LastProblem == nil {
		return ""
	}
	s.doneSet[cid] = true
	return cid
}

// RecordAnswer processes an answer: belief update + evidence propagation +
// confidence update + supplemental + stop conditions.
func (e *Engine) RecordAnswer(s *Session, conceptID string, correct, fast bool) {
	elapsed := 0.0
	if fast {
		elapsed = -1 // marker: fast without measured elapsed (legacy callers/tests)
	} else {
		elapsed = -2 // marker: not-fast without measured elapsed
	}
	e.RecordAnswerTimed(s, conceptID, correct, elapsed, 0)
}

// RecordAnswerTimed is the MA-parity path: elapsed seconds and the
// per-concept time threshold drive automaticity weighting. Correct but
// excessively slow answers get diminished weight (MA: "higher likelihood
// the student has not yet learned the topic well enough"), while fast
// correct answers keep the +0.1 automaticity bonus. Incorrect answers are
// unaffected by timing. elapsed < 0 preserves legacy fast/not-fast-only
// behavior (legacy tests; the goal diagnostic is the only route).
func (e *Engine) RecordAnswerTimed(s *Session, conceptID string, correct bool, elapsed, timeThresh float64) {
	e.recordTimed(s, conceptID, correct, elapsed, timeThresh, false)
}

// RecordDontKnow records an admitted unknown as negative evidence through
// the standard incorrect path (belief drop, negative propagation, normal
// 2-probe settle). An admit is never "fast": speed bonuses apply to correct
// answers only. Like skip, it carries no XP; unlike skip, it counts.
func (e *Engine) RecordDontKnow(s *Session, conceptID string, elapsed, timeThresh float64) {
	e.recordTimed(s, conceptID, false, elapsed, timeThresh, true)
}

func (e *Engine) recordTimed(s *Session, conceptID string, correct bool, elapsed, timeThresh float64, dontKnow bool) {
	s.Lock()
	defer s.Unlock()
	fast := false
	if elapsed < 0 {
		fast = elapsed == -1
	} else {
		thresh := timeThresh
		if thresh <= 0 {
			thresh = 10.0
		}
		fast = elapsed < thresh
	}
	if dontKnow {
		fast = false
	}
	snapshot := make(map[string]float64, len(s.beliefs))
	for id, b := range s.beliefs {
		snapshot[id] = b
	}
	s.Attempts = append(s.Attempts, Attempt{
		ConceptID:      conceptID,
		Correct:        correct,
		Fast:           fast,
		ElapsedSeconds: elapsed,
		Timestamp:      time.Now().UTC(),
		DontKnow:       dontKnow,
		BeliefSnapshot: snapshot,
	})
	s.totalCount[conceptID]++
	s.totalAsked++
	if correct {
		s.correctCount[conceptID]++
	}

	// Belief update for the concept itself (MA parity: slow correct answers
	// get diminished weight — "higher likelihood the student has not yet
	// learned the topic well enough to build on it").
	slowFactor := 1.0
	if correct && elapsed >= 0 {
		thresh := timeThresh
		if thresh <= 0 {
			thresh = 10.0
		}
		switch {
		case elapsed > 4*thresh:
			slowFactor = 0.25
		case elapsed > 2*thresh:
			slowFactor = 0.5
		}
	}
	b := s.beliefs[conceptID]
	if correct {
		b += 0.35 * slowFactor * (1 - b)
		if fast {
			b += 0.1 * (1 - b)
		}
	} else {
		b *= 0.55
	}
	if b < 0.05 {
		b = 0.05
	}
	if b > 0.95 {
		b = 0.95
	}
	s.beliefs[conceptID] = b

	// Evidence propagation with Bayes conflict weighting: correct answers
	// lend positive evidence to prerequisites (diminished when slow);
	// incorrect answers lend negative evidence to dependents and strip
	// barely-passing "conditional" credit from prerequisites.
	e.propagateEvidence(s, conceptID, correct, slowFactor)

	// Confidence: 2 probes settle a concept; a probed concept answered right
	// every time is settled with enough confidence to skip the supplemental
	// re-probe (fork change: correct answers were all asked a second time).
	conf := float64(s.probeCounts[conceptID]) * 0.5
	if s.probeCounts[conceptID] >= 1 && s.correctCount[conceptID] == s.totalCount[conceptID] {
		conf += 0.25
	}
	s.confidence[conceptID] = math.Min(1.0, conf)

	// Settle a concept at 2 probes, or at its first answer when that answer
	// agrees with what the model expected: a right answer, or a miss on a
	// concept it did not already believe known. Only a surprising miss (the
	// model believed it known) gets a second probe, since that may be a slip
	// (fork change: every miss used to cost two questions). Low-confidence
	// ones are revisited by supplementalDiagnostic.
	expectedKnown := false
	if n := len(s.Attempts); n > 0 && s.Attempts[n-1].BeliefSnapshot != nil {
		expectedKnown = s.Attempts[n-1].BeliefSnapshot[conceptID] >= beliefThreshold
	}
	s.doneSet[conceptID] = s.totalCount[conceptID] >= 2 ||
		(correct && s.correctCount[conceptID] == s.totalCount[conceptID]) ||
		(!correct && !expectedKnown && s.probeCounts[conceptID] >= 1)
	s.inferredSettled()

	if s.shouldStop() {
		s.State = StateDone
	}
}

// propagateEvidence applies ±evidence weights along DAG edges with
// conflictDiscount damping both directions when prior evidence conflicts.
func (e *Engine) propagateEvidence(s *Session, conceptID string, correct bool, slowFactor float64) {
	if correct {
		// Every prerequisite below a correct answer is evidence-weighted,
		// the direct ones most, decaying with distance (fork change).
		seen := map[string]bool{conceptID: true}
		level := e.dag.PrereqsOf(conceptID)
		w := ancestorEvidenceWeight * slowFactor
		for len(level) > 0 {
			var next []*concepts.Concept
			for _, pr := range level {
				if seen[pr.ID] {
					continue
				}
				seen[pr.ID] = true
				if _, onPath := s.beliefs[pr.ID]; onPath {
					s.beliefs[pr.ID] += w * (1 - s.beliefs[pr.ID])
				}
				next = append(next, e.dag.PrereqsOf(pr.ID)...)
			}
			level = next
			w *= 0.9
		}
		return
	}
	// A miss is evidence against everything built on it, the direct
	// dependents most, decaying with distance (fork change: one level at
	// 0.15 left a weak learner's whole course unsettled, so the test spent
	// its budget failing plan topics instead of finding where they stand).
	seen := map[string]bool{conceptID: true}
	level := e.dag.DependentsOf(conceptID)
	w := dependentEvidenceWeight
	for len(level) > 0 {
		var next []*concepts.Concept
		for _, dep := range level {
			if seen[dep.ID] {
				continue
			}
			seen[dep.ID] = true
			if _, onPath := s.beliefs[dep.ID]; onPath {
				s.beliefs[dep.ID] *= (1 - w)
			}
			next = append(next, e.dag.DependentsOf(dep.ID)...)
		}
		level = next
		w *= 0.85
	}
	// MA fall-back: a miss also strips "conditionally completed" credit
	// from barely-passed prerequisites — the frontier falls backwards
	// along the learning path immediately on struggle.
	for _, pr := range e.dag.PrereqsOf(conceptID) {
		if pr.ID == conceptID {
			continue
		}
		if pb := s.beliefs[pr.ID]; pb >= beliefThreshold && pb < beliefThreshold+0.15 {
			s.beliefs[pr.ID] = pb * conditionalStripFactor
		}
	}
}

// supplementalDiagnostic re-probes settled cover concepts whose confidence
// never reached confidenceThreshold (e.g. answers recorded without question
// probes on the legacy API path). Each concept gets at most
// maxSupplementalProbes extra probes; "" means supplemental is complete.
// Caller must hold s.Lock (NextQuestion does).
func (e *Engine) supplementalDiagnostic(s *Session) string {
	cover := e.compressedCover(s)
	for _, c := range cover {
		if !s.doneSet[c.ID] {
			continue
		}
		if s.totalCount[c.ID] == 0 {
			continue // settled by inference, never asked
		}
		if s.confidence[c.ID] >= confidenceThreshold {
			continue
		}
		if s.supplementalCount[c.ID] >= maxSupplementalProbes {
			continue
		}
		s.supplementalCount[c.ID]++
		delete(s.doneSet, c.ID)
		return c.ID
	}
	return ""
}

func (s *Session) shouldStop() bool {
	if s.totalAsked >= maxTotalQuestions {
		return true
	}
	cover := s.compressedCoverSafe()
	coverDone := true
	for _, c := range cover {
		if !s.doneSet[c.ID] {
			coverDone = false
			break
		}
	}
	if coverDone && s.totalAsked >= minTotalQuestions {
		return true
	}
	return false
}

// compressedCoverSafe is the covering set, a session-only computation
// (no Engine needed, for the stop condition and progress).
//
// Without goal topics: roots of the path plus evenly spaced concepts across
// its topological order (upstream behaviour). With goal topics (a course
// placement, fork change): half the cover is the goal topics, evenly sampled
// in path order, and half is evenly spaced across the path; roots are not
// forced in, because a correct answer above them settles them by inference.
func (s *Session) compressedCoverSafe() []*concepts.Concept {
	if len(s.order) <= coverSize {
		return s.order
	}
	var out []*concepts.Concept
	seen := make(map[string]bool)
	add := func(c *concepts.Concept) {
		if !seen[c.ID] {
			seen[c.ID] = true
			out = append(out, c)
		}
	}
	if len(s.targets) > 0 {
		var goal []*concepts.Concept
		for _, c := range s.order {
			if s.targets[c.ID] {
				goal = append(goal, c)
			}
		}
		half := coverSize / 2
		if len(goal) <= half {
			for _, c := range goal {
				add(c)
			}
		} else {
			for i := 0; i < half; i++ {
				add(goal[i*len(goal)/half])
			}
		}
		step := int(math.Ceil(float64(len(s.order)) / float64(coverSize-len(out))))
		for i := step / 2; i < len(s.order); i += step {
			add(s.order[i])
		}
		sort.Slice(out, func(i, j int) bool { return s.index(out[i].ID) < s.index(out[j].ID) })
		return out
	}
	step := int(math.Ceil(float64(len(s.order)) / float64(coverSize)))
	for i := 0; i < len(s.order); i += step {
		add(s.order[i])
	}
	for _, c := range s.order {
		if len(c.Prerequisites) == 0 {
			add(c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return s.index(out[i].ID) < s.index(out[j].ID) })
	return out
}

// inferredSettled marks cover concepts whose belief evidence elsewhere has
// made decisive (fork change): a correct answer two levels above single-digit
// addition settles it without asking. They count as covered and are never
// picked; they carry no probe, so the supplemental pass leaves them alone.
// Caller holds s.Lock.
func (s *Session) inferredSettled() {
	for _, c := range s.compressedCoverSafe() {
		if s.doneSet[c.ID] || s.totalCount[c.ID] > 0 {
			continue
		}
		if b := s.beliefs[c.ID]; b >= inferKnown || b <= inferUnknown {
			s.doneSet[c.ID] = true
		}
	}
}

func (e *Engine) IsComplete(s *Session) bool {
	s.Lock()
	defer s.Unlock()
	return s.State == StateDone
}

// Progress returns the backend truth for the progress bar (MA parity).
// CoverDone/CoverSize is monotonic (doneSet only grows); display the bar
// from cover fraction, never from the moving estimated_total denominator.
func (e *Engine) Progress(s *Session) Progress {
	s.Lock()
	defer s.Unlock()
	answered := s.totalAsked
	cover := s.compressedCoverSafeLocked()
	coverDone := 0
	for _, c := range cover {
		if s.doneSet[c.ID] {
			coverDone++
		}
	}
	remaining := len(cover) - coverDone
	est := answered + remaining
	if est < minTotalQuestions {
		est = minTotalQuestions
	}
	if est > maxTotalQuestions {
		est = maxTotalQuestions
	}
	if est < answered {
		est = answered
	}
	return Progress{
		Answered:       answered,
		EstimatedTotal: est,
		MinTotal:       minTotalQuestions,
		MaxTotal:       maxTotalQuestions,
		CoverDone:      coverDone,
		CoverSize:      len(cover),
		Done:           s.State == StateDone,
	}
}

// compressedCoverSafeLocked is compressedCoverSafe without locking
// (caller must hold s.Lock).
func (s *Session) compressedCoverSafeLocked() []*concepts.Concept {
	return s.compressedCoverSafe()
}

// FrontierEstimate returns the highest order index whose belief is known.
func (e *Engine) FrontierEstimate(s *Session) int {
	s.Lock()
	defer s.Unlock()
	idx := -1
	for i, c := range s.order {
		if s.beliefs[c.ID] >= beliefThreshold {
			idx = i
		}
	}
	return idx
}

// KnowledgeConfidence returns per-concept confidence 0-1 (copy, lock-safe).
func (e *Engine) KnowledgeConfidence(s *Session) map[string]float64 {
	s.Lock()
	defer s.Unlock()
	out := make(map[string]float64, len(s.confidence))
	for k, v := range s.confidence {
		out[k] = v
	}
	return out
}

// clampDifficulty keeps a requested difficulty in the registry's 0-1 range,
// as registry.Generate does.
func clampDifficulty(d float64) float64 {
	if d < 0 {
		return 0
	}
	if d > 1 {
		return 1
	}
	return d
}
