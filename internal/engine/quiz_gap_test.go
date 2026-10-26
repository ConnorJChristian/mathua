package engine

import (
	"testing"
	"time"
)

// An install can space mastery checks out to weekly with
// MATHUA_QUIZ_GAP_HOURS; without it the nightly default holds, and a value
// that is not a number is ignored rather than turning the gate off.
func TestQuizGateGapFromEnv(t *testing.T) {
	t.Setenv("MATHUA_QUIZ_GAP_HOURS", "")
	if got := quizGateGap(); got != QuizGateGap {
		t.Errorf("default gap = %v, want %v", got, QuizGateGap)
	}
	t.Setenv("MATHUA_QUIZ_GAP_HOURS", "168")
	if got := quizGateGap(); got != 168*time.Hour {
		t.Errorf("weekly gap = %v, want 168h", got)
	}
	t.Setenv("MATHUA_QUIZ_GAP_HOURS", "weekly")
	if got := quizGateGap(); got != QuizGateGap {
		t.Errorf("unparseable value gave %v, want the default", got)
	}
}
