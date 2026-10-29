package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/auth"
	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/engine"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/grader"
	"github.com/chuma-beep/mathua/internal/planning"
	"github.com/chuma-beep/mathua/internal/storage"
)

// downGen is a topic whose grader goes down once registered (the registry
// self-checks a grader at registration): every answer is then ungraded.
type downGen struct{ down *bool }

func (downGen) Generate(generator.GeneratorContext) generator.Problem {
	return generator.Problem{Question: "What is 1?", Answer: "1"}
}
func (g downGen) Grade(expected, answer string) grader.Result {
	if *g.down {
		return grader.Result{Feedback: "Grading service timed out", Unavailable: true}
	}
	return grader.Result{Correct: answer == expected}
}

// courseTestServer has two courses of eight chained generated topics:
// plan.ok (c1..c8, graded normally, in two units) and plan.down (d1..d8,
// whose grader is down).
func courseTestServer(t *testing.T) (*http.ServeMux, *storage.SQLiteStore) {
	t.Helper()
	th := concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 30}
	var raw []concepts.Concept
	reg := generator.NewRegistry()
	down := false
	t.Cleanup(func() { down = false })
	for _, p := range []string{"c", "d"} {
		for i := 1; i <= 8; i++ {
			id := fmt.Sprintf("%s%d", p, i)
			var pre []string
			if i > 1 {
				pre = []string{fmt.Sprintf("%s%d", p, i-1)}
			}
			raw = append(raw, concepts.Concept{ID: id, Label: "Topic " + id, Domain: "algebra", Subdomain: "s" + id, GradingType: "numeric", Prerequisites: pre, MasteryThreshold: th})
			if p == "c" {
				reg.Register(id, &testGen{})
			} else {
				if err := reg.Register(id, downGen{&down}); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	down = true
	d, err := concepts.Build(raw)
	if err != nil {
		t.Fatal(err)
	}
	cp := filepath.Join(t.TempDir(), "courses.json")
	os.WriteFile(cp, []byte(`[
	 {"id":"plan.ok","name":"Course OK","plan":["c1","c2","c3","c4","c5","c6","c7","c8"],
	  "modules":[{"title":"First half","topics":["c1","c2","c3","c4"]},{"title":"Second half","topics":["c5","c6","c7","c8"]}]},
	 {"id":"plan.down","name":"Course down","plan":["d1","d2","d3","d4","d5","d6","d7","d8"]}
	]`), 0o644)
	pl, err := planning.Load(cp, d)
	if err != nil {
		t.Fatal(err)
	}
	store, err := storage.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	s := New(engine.New(store, d, reg, nil, pl), store, auth.New(store))
	mux := http.NewServeMux()
	s.Register(mux)
	return mux, store
}

func decodeMap(t *testing.T, body string) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("decode %q: %v", body, err)
	}
	return m
}

func studyAll(t *testing.T, store *storage.SQLiteStore, sid string, ids ...string) {
	t.Helper()
	now := time.Now().UTC()
	for _, id := range ids {
		if err := store.UpsertProgress(&storage.ConceptProgress{StudentID: sid, ConceptID: id, Status: "LEARNING", Attempts: 2, WeaknessScore: 0.5, LastAttempted: &now, SM2EFactor: 2.5}); err != nil {
			t.Fatal(err)
		}
	}
}

// The course view says which tests a course offers, and both tests start
// from it: placement with a course id, review over studied topics, with a
// per-section summary at the end.
func TestCourseTests_ViewPlacementAndReview(t *testing.T) {
	mux, store := courseTestServer(t)
	token, sid := guestTokenAndID(t, mux)

	rec := authedGet(t, mux, "/api/courses/view?id=plan.ok", token)
	if rec.Code != 200 {
		t.Fatalf("view: %d %s", rec.Code, rec.Body.String())
	}
	view := decodeMap(t, rec.Body.String())
	tests, _ := view["tests"].(map[string]interface{})
	if tests == nil || tests["placement"] != true || tests["review"] != true || tests["review_topics"].(float64) != 0 {
		t.Fatalf("tests = %v", view["tests"])
	}

	code, body := postJSONRaw(t, mux, "/api/goal/diagnostic", `{"course_id":"plan.ok"}`, token)
	if code != 200 {
		t.Fatalf("placement start: %d %s", code, body)
	}
	if m := decodeMap(t, body); m["course_id"] != "plan.ok" || m["question"] == nil {
		t.Fatalf("placement start = %v", m)
	}

	// Nothing studied: the review test says so instead of testing.
	code, body = postJSONRaw(t, mux, "/api/quiz/session", `{"course_id":"plan.ok"}`, token)
	if m := decodeMap(t, body); code != 200 || m["empty"] != true {
		t.Fatalf("empty review: %d %s", code, body)
	}

	studyAll(t, store, sid, "c1", "c2", "c6")
	code, body = postJSONRaw(t, mux, "/api/quiz/session", `{"course_id":"plan.ok"}`, token)
	m := decodeMap(t, body)
	if code != 200 || m["questions_total"].(float64) != 3 || m["course_id"] != "plan.ok" || m["test"] != "review" {
		t.Fatalf("review start: %d %s", code, body)
	}
	sess := m["session_id"].(string)
	cid := m["concept_id"].(string)
	var last map[string]interface{}
	for i := 0; i < 3; i++ {
		answer := "42" // testGen's answer is "" so "42" is a miss
		code, body = postJSONRaw(t, mux, "/api/quiz/answer", fmt.Sprintf(`{"session_id":%q,"concept_id":%q,"answer":%q,"elapsed":3}`, sess, cid, answer), token)
		if code != 200 {
			t.Fatalf("answer %d: %d %s", i, code, body)
		}
		last = decodeMap(t, body)
		if last["done"] == true {
			break
		}
		cid = last["concept_id"].(string)
	}
	if last["done"] != true {
		t.Fatalf("review did not finish: %v", last)
	}
	secs, _ := last["sections"].([]interface{})
	if len(secs) != 2 {
		t.Fatalf("want two sections in the summary, got %v", last["sections"])
	}
	missed := 0
	for _, s := range secs {
		missed += len(s.(map[string]interface{})["missed"].([]interface{}))
	}
	if missed != 3 {
		t.Errorf("missed = %d, sections %v", missed, secs)
	}
	// Finishing it restarts the mastery check's gate and gap, and the end
	// screen has to be told so.
	if last["counted_as_mastery_check"] != true {
		t.Errorf("a finished course test should say it counted as the mastery check: %v", last)
	}
}

// A grader that is down never records a miss, in either test: the answer is
// refused with 503, nothing is recorded, and the same question stays pending.
func TestCourseTests_UngradedIsNeverAMiss(t *testing.T) {
	mux, store := courseTestServer(t)
	token, sid := guestTokenAndID(t, mux)

	// Placement.
	code, body := postJSONRaw(t, mux, "/api/goal/diagnostic", `{"course_id":"plan.down"}`, token)
	if code != 200 {
		t.Fatalf("placement start: %d %s", code, body)
	}
	m := decodeMap(t, body)
	if m["concept_id"] == nil {
		t.Fatalf("placement start: %s", body)
	}
	sess, cid := m["session_id"].(string), m["concept_id"].(string)
	for i := 0; i < 2; i++ {
		code, body = postJSONRaw(t, mux, "/api/goal/diagnostic/answer", fmt.Sprintf(`{"session_id":%q,"concept_id":%q,"answer":"1","elapsed":3}`, sess, cid), token)
		if code != http.StatusServiceUnavailable {
			t.Fatalf("placement answer %d: want 503, got %d %s", i, code, body)
		}
	}
	rec := authedGet(t, mux, "/api/goal/diagnostic/resume?session_id="+sess, token)
	if r := decodeMap(t, rec.Body.String()); r["concept_id"] != cid || r["progress"].(map[string]interface{})["answered"].(float64) != 0 {
		t.Fatalf("placement moved on after an ungraded answer: %v", r)
	}

	// Review.
	studyAll(t, store, sid, "d1", "d2")
	code, body = postJSONRaw(t, mux, "/api/quiz/session", `{"course_id":"plan.down"}`, token)
	if code != 200 {
		t.Fatalf("review start: %d %s", code, body)
	}
	m = decodeMap(t, body)
	sess, cid = m["session_id"].(string), m["concept_id"].(string)
	for i := 0; i < 2; i++ {
		code, body = postJSONRaw(t, mux, "/api/quiz/answer", fmt.Sprintf(`{"session_id":%q,"concept_id":%q,"answer":"1","elapsed":3}`, sess, cid), token)
		if code != http.StatusServiceUnavailable {
			t.Fatalf("review answer %d: want 503, got %d %s", i, code, body)
		}
	}
	p, err := store.GetProgress(sid, cid)
	if err != nil || p == nil || p.Attempts != 2 {
		t.Fatalf("progress changed by an ungraded answer: %+v %v", p, err)
	}
}

// A course placement's results agree with the course page: "Start here" is the
// course page's next topic, and the XP completion estimate gives way to the
// course's own pace.
func TestCoursePlacementResults_MatchTheCoursePage(t *testing.T) {
	mux, _ := courseTestServer(t)
	token, _ := guestTokenAndID(t, mux)
	code, body := postJSONRaw(t, mux, "/api/goal/diagnostic", `{"course_id":"plan.ok"}`, token)
	if code != 200 {
		t.Fatalf("placement start: %d %s", code, body)
	}
	m := decodeMap(t, body)
	sess, cid := m["session_id"].(string), m["concept_id"].(string)
	for i := 0; i < 60; i++ {
		code, body = postJSONRaw(t, mux, "/api/goal/diagnostic/answer", fmt.Sprintf(`{"session_id":%q,"concept_id":%q,"answer":"42","elapsed":3}`, sess, cid), token)
		if code != 200 {
			t.Fatalf("answer %d: %d %s", i, code, body)
		}
		m = decodeMap(t, body)
		if m["done"] == true {
			break
		}
		cid, _ = m["concept_id"].(string)
	}
	code, body = postJSONRaw(t, mux, "/api/goal/plan", fmt.Sprintf(`{"session_id":%q}`, sess), token)
	if code != 200 {
		t.Fatalf("plan: %d %s", code, body)
	}
	plan := decodeMap(t, body)
	view := decodeMap(t, authedGet(t, mux, "/api/courses/view?id=plan.ok", token).Body.String())
	next, _ := view["next_topic"].(map[string]interface{})
	if next == nil {
		t.Fatalf("the course page names no next topic: %v", view)
	}
	if plan["frontier_label"] != next["label"] || plan["frontier_id"] != next["id"] {
		t.Errorf("Start here = %v (%v), the course page says %v (%v)", plan["frontier_label"], plan["frontier_id"], next["label"], next["id"])
	}
	if plan["completion_estimates"] != nil {
		t.Errorf("completion_estimates = %v; a course placement reports the course's pace instead", plan["completion_estimates"])
	}
	pace, _ := plan["course_pace"].(map[string]interface{})
	if pace == nil || pace["topics_left"] == nil {
		t.Errorf("course_pace = %v", plan["course_pace"])
	}
}
