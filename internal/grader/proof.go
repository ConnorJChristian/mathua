package grader

import "strings"

// GradingProof is a written proof graded against an authored marking scheme
// by a language-model judge (internal/proof). The judge is installed at
// startup with SetProofGrader; this package only routes to it, the same way
// it routes question-bank answers to internal/pytrack.
const GradingProof GradingType = "proof"

// ProofPrefix marks a served proof question's answer key: "proof:<problem id>".
// The key names the problem; the rubric and reference proof stay on the server.
const ProofPrefix = "proof:"

// IsProofAnswer reports a served answer key that belongs to a proof problem.
func IsProofAnswer(expected string) bool { return strings.HasPrefix(expected, ProofPrefix) }

// ProofMilestone is one line of a marking scheme with the judge's verdict on it.
type ProofMilestone struct {
	ID     string `json:"id"`
	Desc   string `json:"desc"`
	Points int    `json:"points"`
	// Met is "yes", "partial" or "no".
	Met  string `json:"met"`
	Note string `json:"note,omitempty"`
}

// ProofReport is the detail of a graded proof, carried on Result.Proof so the
// answer hosts can show the score, the milestone checklist and the feedback.
type ProofReport struct {
	// Score is the median of the judge runs, 0 to Max.
	Score int `json:"score"`
	Max   int `json:"max"`
	// Runs are the individual run scores, in the order they finished.
	Runs []int `json:"runs"`
	// FullyCorrect is true only when every run scored at least the pass mark.
	FullyCorrect bool             `json:"fully_correct"`
	Milestones   []ProofMilestone `json:"milestones"`
	FirstError   string           `json:"first_error,omitempty"`
	Feedback     string           `json:"feedback"`
	// Cached is true when an identical proof of the same problem was graded
	// before and this result cost nothing.
	Cached bool `json:"cached,omitempty"`
	// CountsForMastery says whether this result moved the mastery ladder
	// (MATHUA_PROOF_COUNTS_FOR_MASTERY); when false it is feedback only.
	CountsForMastery bool   `json:"counts_for_mastery"`
	Model            string `json:"model,omitempty"`
}

// proofGrader is installed by internal/proof at startup. It is a hook rather
// than an import because internal/proof depends on this package for Result.
var proofGrader func(expected, answer string) Result

// SetProofGrader installs the proof grader. Call it once at startup; nil
// uninstalls it (tests).
func SetProofGrader(fn func(expected, answer string) Result) { proofGrader = fn }

// gradeProof routes a proof answer to the installed grader. Without one the
// answer is not graded at all: "no judge configured" is not "you were wrong".
func gradeProof(expected, answer string) Result {
	if proofGrader == nil {
		return Result{Feedback: "Proof grading is not configured on this server", Unavailable: true}
	}
	return proofGrader(expected, answer)
}
