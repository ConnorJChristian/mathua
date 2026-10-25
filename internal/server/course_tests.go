package server

import (
	"net/http"
	"time"

	"github.com/chuma-beep/mathua/internal/quiz"
)

// Fork change: course review tests. POST /api/quiz/session with a course_id
// starts one; answers, skips and grading go through the mastery check's own
// handlers (handleQuizAnswer, handleQuizSkip), so a review answer is mastery
// evidence exactly as a mastery-check answer is, and a missed topic is queued
// for remediation the same way. Course placement tests are the goal
// diagnostic started with a course_id (handleGoalDiagnosticStart).

// startCourseReview starts a review test of one course: studied, objectively
// gradable course topics (engine.CourseReviewConcepts), one question each.
func (s *Server) startCourseReview(w http.ResponseWriter, studentID, courseID string, full bool) {
	if studentID == "" {
		writeError(w, "sign in to take a course review test", http.StatusUnauthorized)
		return
	}
	pl := s.eng.GetPlanner()
	if pl == nil || pl.Course(courseID) == nil {
		writeError(w, "course not found", http.StatusNotFound)
		return
	}
	kind := "review"
	if full {
		kind = "full"
	}
	info, err := s.eng.CourseTests(studentID, courseID)
	if err != nil {
		writeError(w, "failed to load course", 500)
		return
	}
	if !info.Review {
		writeJSON(w, map[string]interface{}{
			"done": true, "empty": true, "course_id": courseID, "test": kind,
			"message": "This course does not have enough topics with automatically graded questions for a review test.",
		})
		return
	}
	picked, err := s.eng.CourseReviewConcepts(studentID, courseID, full)
	if err != nil {
		writeError(w, "failed to build the review test", 500)
		return
	}
	if len(picked) == 0 {
		writeJSON(w, map[string]interface{}{
			"done": true, "empty": true, "course_id": courseID, "test": kind,
			"message": "Nothing in this course has been studied yet. The review test covers what the course has taught you, so do a lesson or two first.",
		})
		return
	}
	dag := s.eng.GetDAG()
	qEng := quiz.NewEngine(dag, s.eng.GetGeneratorRegistry())
	sess := qEng.StartLimit(picked, len(picked))
	sess.Objective = true
	sess.CourseID = courseID
	sess.Kind = kind
	for _, c := range picked {
		sess.Difficulties[c.ID] = s.eng.DifficultyFor(studentID, c.ID)
		if recent, err := s.repo.GetRecentAttemptsForConcept(studentID, c.ID, 12); err == nil {
			for _, a := range recent {
				if a.Question != "" {
					if sess.Avoid == nil {
						sess.Avoid = map[string]bool{}
					}
					sess.Avoid[a.Question] = true
				}
			}
		}
	}
	sess.ID = newUUID()
	sess.StudentID = studentID
	s.mu.Lock()
	s.quizSessions[sess.ID] = sess
	s.quizCreated[sess.ID] = time.Now()
	s.mu.Unlock()
	prob, cid, err := qEng.NextQuestion(sess)
	if err != nil {
		writeError(w, "failed to get quiz question", 500)
		return
	}
	name := ""
	if pc := pl.Course(courseID); pc != nil {
		name = pc.Name
	}
	if prob == nil || cid == "" {
		writeJSON(w, map[string]interface{}{"session_id": sess.ID, "done": true, "course_id": courseID, "test": kind})
		return
	}
	label := cid
	if c := dag.Concept(cid); c != nil {
		label = c.Label
	}
	writeJSON(w, s.withAnswerFormat(map[string]interface{}{
		"session_id":         sess.ID,
		"student_id":         studentID,
		"concept_id":         cid,
		"concept_name":       label,
		"question":           prob.Question,
		"grading_type":       s.gradingTypeOf(cid),
		"done":               false,
		"closed_book":        true,
		"time_limit_seconds": s.eng.TimeLimitFor(studentID, cid),
		"questions_total":    len(sess.Order),
		"courses":            []string{name},
		"course_id":          courseID,
		"test":               kind,
	}, cid))
}

// addCourseTestSummary adds a finished course review's per-section summary
// to the final verdict: each tested topic's last result, by section.
func (s *Server) addCourseTestSummary(out map[string]interface{}, sess *quiz.Session) {
	sess.Lock()
	courseID, kind := sess.CourseID, sess.Kind
	results := make(map[string]bool, len(sess.Attempts))
	for _, a := range sess.Attempts {
		results[a.ConceptID] = a.Correct
	}
	sess.Unlock()
	if courseID == "" {
		return
	}
	out["course_id"] = courseID
	out["test"] = kind
	out["sections"] = s.eng.CourseTestSummary(courseID, results, nil)
}
