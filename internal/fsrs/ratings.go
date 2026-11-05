package fsrs

// Rating is what a scheduled review earned: how well the learner recalled the
// concept at the moment it came due.
//
// The four values are FSRS's. Only two of them can be produced from Mathua
// today, and that is not an accident of incomplete code: a correct answer is
// Good, an incorrect answer is Again. Hard would claim the learner recalled it
// with serious difficulty and Easy would claim it was effortless, and Mathua
// has no calibrated signal for either. A timing ratio is not that signal, a
// fast wrong answer is common, and a slow correct answer on a hard concept is
// not evidence of Easy recall.
type Rating int

const (
	// Again means the learner could not recall the concept.
	Again Rating = 1
	// Hard means recall was difficult. No call site produces it yet.
	Hard Rating = 2
	// Good means the learner recalled the concept.
	Good Rating = 3
	// Easy means recall was effortless. No call site produces it yet.
	Easy Rating = 4
)

// Valid reports whether r is one of the four ratings.
func (r Rating) Valid() bool { return r >= Again && r <= Easy }

// NativeRating maps one graded answer to the only rating it can justify.
//
// It exists so that the mapping is a single decision in one place rather than
// an if/else repeated at four call sites, and so that a review of the code can
// see exactly which ratings Mathua can award.
func NativeRating(correct bool) Rating {
	if correct {
		return Good
	}
	return Again
}

// LegacySM2 is the pre-FSRS scheduling state of one learner and concept.
//
// It is a distinct type from Rating on purpose. A legacy compatibility
// transition, the correct-but-streak-unmet case, which the old SM-2 code
// treats as a failed repetition and schedules one day out, is not a review
// rating and must never be recorded as one. The compiler will not let a
// LegacySM2 be passed where a Rating is expected, which is a cheaper guarantee
// than a comment saying not to.
type LegacySM2 struct {
	// Repetitions is the SM-2 repetition count.
	Repetitions int
	// EFactor is the easiness factor, conventionally 2.5 at the start.
	EFactor float64
	// IntervalDays is the scheduled gap in days.
	IntervalDays int
}

// LegacyStreakUnmet is the legacy transition for a correct answer that has not
// yet met the concept's required streak.
//
// It reproduces ComputeSM2WithSpeed's quality-below-3 branch exactly: the
// repetition count resets, the interval becomes one day and the easiness factor
// is left alone. That reset is real behaviour today and is preserved, but it is
// compatibility behaviour, not a memory measurement. It must not touch FSRS
// state: recording it as Again would both fabricate a rating and leave the
// recorded state disagreeing with the due date that the compatibility path
// wrote.
func LegacyStreakUnmet(prev LegacySM2) LegacySM2 {
	out := prev
	out.Repetitions = 0
	out.IntervalDays = 1
	return out
}
