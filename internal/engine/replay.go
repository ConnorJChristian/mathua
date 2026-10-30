package engine

// ExpectedAnswerForReplay returns the answer key of the question a session
// is waiting on, so a replay (scripts/replay_audit) can answer it the way
// a learner who knows the material would. It is an engine method only: no
// route serves it, and the key stays on the server.
func (e *Engine) ExpectedAnswerForReplay(sessionID string) (string, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	as := e.sessions[sessionID]
	if as == nil || as.answered || as.attemptID == "" {
		return "", false
	}
	return as.expectedAnswer, true
}
