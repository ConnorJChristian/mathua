package engine

import (
	"testing"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/generator"
	"github.com/chuma-beep/mathua/internal/mastery"
	"github.com/chuma-beep/mathua/internal/storage"
)

// Fork change: remediation draws on a topic's prerequisites wherever they
// live. A textbook topic ("hoy.partial", module hoy) rests on the corpus's
// partial derivatives ("calc.partial") and on a pandas topic from another
// module ("py.frame"); it has no key_prerequisites, as no module topic does.
func remedialEngine(t *testing.T) *Engine {
	t.Helper()
	th := concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 60}
	d, err := concepts.Build([]concepts.Concept{
		{ID: "alg.func", Label: "Functions", Domain: "algebra", GradingType: "numeric", Prerequisites: []string{}, MasteryThreshold: th},
		{ID: "calc.partial", Label: "Partial derivatives", Domain: "calculus", GradingType: "numeric", Prerequisites: []string{"alg.func"}, MasteryThreshold: th},
		{ID: "py.frame", Label: "Selecting from a DataFrame", Domain: "python_pandas", GradingType: "numeric", Prerequisites: []string{}, MasteryThreshold: th},
		{ID: "hoy.partial", Label: "Partial derivatives in economics", Domain: "mathematical_economics", GradingType: "numeric",
			Prerequisites: []string{"calc.partial", "py.frame"}, MasteryThreshold: th},
		{ID: "hoy.keyed", Label: "A topic with a key prerequisite", Domain: "mathematical_economics", GradingType: "numeric",
			Prerequisites: []string{"calc.partial", "py.frame"}, KeyPrerequisites: []string{"py.frame"}, MasteryThreshold: th},
		{ID: "hoy.unservable", Label: "A topic whose prerequisite has no generator", Domain: "mathematical_economics", GradingType: "numeric",
			Prerequisites: []string{"no.generator"}, MasteryThreshold: th},
		{ID: "no.generator", Label: "Nothing serves this", Domain: "calculus", GradingType: "numeric", Prerequisites: []string{}, MasteryThreshold: th},
	})
	if err != nil {
		t.Fatal(err)
	}
	store, err := storage.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	reg := generator.NewRegistry()
	for _, id := range []string{"alg.func", "calc.partial", "py.frame", "hoy.partial", "hoy.keyed", "hoy.unservable"} {
		reg.Register(id, &testGen{answer: "1"})
	}
	return New(store, d, reg, nil, nil)
}

func TestRemedial_CrossModulePrerequisitesWeakestFirst(t *testing.T) {
	e := remedialEngine(t)
	st, err := e.CreateStudent("struggles")
	if err != nil {
		t.Fatal(err)
	}
	// Untouched: both direct prerequisites, in declared order, then the topic.
	got := e.remedialPrerequisites(st.ID, "hoy.partial")
	if len(got) != 2 || got[0] != "calc.partial" || got[1] != "py.frame" {
		t.Fatalf("remedial prerequisites of an untouched topic: %v", got)
	}
	// The pandas topic is the weak one: it comes first; the calculus concept
	// is mastered, so it comes last.
	if err := e.repo.UpsertProgressBatch([]*storage.ConceptProgress{
		{StudentID: st.ID, ConceptID: "calc.partial", Status: string(mastery.StatusMastered), SM2EFactor: 2.5},
		{StudentID: st.ID, ConceptID: "py.frame", Status: string(mastery.StatusLearning), WeaknessScore: 0.8, SM2EFactor: 2.5},
	}); err != nil {
		t.Fatal(err)
	}
	got = e.remedialPrerequisites(st.ID, "hoy.partial")
	if len(got) != 2 || got[0] != "py.frame" || got[1] != "calc.partial" {
		t.Fatalf("the unmastered, weak prerequisite should lead: %v", got)
	}
	// key_prerequisites, when set, decide alone (upstream's behaviour).
	if got = e.remedialPrerequisites(st.ID, "hoy.keyed"); len(got) != 1 || got[0] != "py.frame" {
		t.Fatalf("key prerequisites should win: %v", got)
	}
	// A prerequisite nothing can serve is never queued: it would stop the session.
	if got = e.remedialPrerequisites(st.ID, "hoy.unservable"); len(got) != 0 {
		t.Fatalf("an unservable prerequisite must not be queued: %v", got)
	}
}

// A miss on the quiz, or a second miss in a row through Learn, queues the
// topic's prerequisites from the other modules ahead of the topic itself.
func TestRemedial_MissQueuesPrerequisitesFromOtherModules(t *testing.T) {
	e := remedialEngine(t)
	st, err := e.CreateStudent("quiz")
	if err != nil {
		t.Fatal(err)
	}
	res, err := e.SubmitQuizAnswer(st.ID, "hoy.partial", "wrong", "1", 5, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"calc.partial", "py.frame", "hoy.partial"}
	if len(res.Remedial) != 3 || res.Remedial[0] != want[0] || res.Remedial[1] != want[1] || res.Remedial[2] != want[2] {
		t.Fatalf("quiz miss remedial = %v, want %v", res.Remedial, want)
	}
	if got := e.QuizRemedial(st.ID); len(got) != 3 {
		t.Fatalf("queued: %v", got)
	}

	st2, _ := e.CreateStudent("learn")
	e.SetStudyAnchor(st2.ID, "hoy.partial", "1")
	e.SetStudyAnchor(st2.ID, "calc.partial", "1")
	if r1, err := e.SubmitStudyAnswer(st2.ID, "hoy.partial", "wrong", 5, "q"); err != nil || r1.Halted || len(r1.Remedial) != 0 {
		t.Fatalf("one miss through Learn is not a halt: %+v %v", r1, err)
	}
	r2, err := e.SubmitStudyAnswer(st2.ID, "hoy.partial", "wrong", 5, "q")
	if err != nil {
		t.Fatal(err)
	}
	if !r2.Halted || len(r2.Remedial) != 3 || r2.Remedial[0] != "calc.partial" || r2.Remedial[1] != "py.frame" {
		t.Fatalf("a second miss through Learn should halt and queue the prerequisites: %+v", r2)
	}
	// Practising the corpus concept correctly is its remedy, from any module.
	if _, err := e.SubmitStudyAnswer(st2.ID, "calc.partial", "1", 5, "q"); err != nil {
		t.Fatal(err)
	}
	for _, id := range e.QuizRemedial(st2.ID) {
		if id == "calc.partial" {
			t.Fatalf("a correct answer on calc.partial should take it off the queue: %v", e.QuizRemedial(st2.ID))
		}
	}
}

// Practising a locked topic through a session and missing twice serves the
// prerequisites next, in order, before the topic at an easier difficulty.
func TestRemedial_SessionHaltServesCrossModulePrerequisites(t *testing.T) {
	e := remedialEngine(t)
	st, _ := e.CreateStudent("session")
	sess, _ := e.repo.CreateSession(st.ID)
	// Pin the session on hoy.partial the way Learn's "practise anyway" does:
	// the active path holds only that topic.
	e.mu.Lock()
	e.activePath[st.ID] = map[string]bool{"hoy.partial": true, "calc.partial": true, "py.frame": true, "alg.func": true}
	e.sessions[sess.ID] = &activeSession{studentID: st.ID}
	e.mu.Unlock()
	// Force the served concept: two misses in a row on hoy.partial.
	for i := 0; i < 2; i++ {
		e.mu.Lock()
		as := e.sessions[sess.ID]
		as.answered, as.attemptID, as.conceptID, as.expectedAnswer, as.questionText = false, newAttemptID(), "hoy.partial", "1", "q"
		as.requiredStreak, as.timeThreshold = 3, 60
		attempt := as.attemptID
		e.mu.Unlock()
		res, err := e.SubmitAnswer(sess.ID, st.ID, attempt, "wrong", 5)
		if err != nil {
			t.Fatal(err)
		}
		if i == 1 && (!res.Halted || len(res.Remedial) != 3 || res.Remedial[0] != "calc.partial" || res.Remedial[1] != "py.frame") {
			t.Fatalf("halt should queue the two prerequisites then the topic: %+v", res)
		}
	}
	var served []string
	for i := 0; i < 3; i++ {
		q, err := e.NextQuestion(sess.ID, st.ID)
		if err != nil || q == nil {
			t.Fatalf("remedial question %d: %v %v", i, q, err)
		}
		served = append(served, q.ConceptID)
	}
	if served[0] != "calc.partial" || served[1] != "py.frame" || served[2] != "hoy.partial" {
		t.Fatalf("served %v, want the corpus concept, the pandas topic, then the topic", served)
	}
}

// Credit crosses modules: a textbook topic unlocks once its corpus and
// pandas prerequisites are mastered, wherever they were practised.
func TestRemedial_CreditAndEligibilityCrossModules(t *testing.T) {
	e := remedialEngine(t)
	st, _ := e.CreateStudent("credit")
	if el, _ := e.EligibilityFor(st.ID, "hoy.partial"); el.Eligible {
		t.Fatal("hoy.partial waits on calc.partial and py.frame")
	}
	if err := e.repo.UpsertProgressBatch([]*storage.ConceptProgress{
		{StudentID: st.ID, ConceptID: "calc.partial", Status: string(mastery.StatusMastered), SM2EFactor: 2.5},
		{StudentID: st.ID, ConceptID: "py.frame", Status: string(mastery.StatusMastered), SM2EFactor: 2.5},
	}); err != nil {
		t.Fatal(err)
	}
	if el, _ := e.EligibilityFor(st.ID, "hoy.partial"); !el.Eligible {
		t.Fatalf("mastery in the corpus and in the pandas module should unlock it: %+v", el)
	}
	// And mastering the textbook topic credits both prerequisites' modules.
	e.creditPrerequisites(st.ID, "hoy.partial", true)
	for _, id := range []string{"alg.func"} {
		p, err := e.repo.GetProgress(st.ID, id)
		if err != nil || p == nil || p.ImplicitCredit != creditPrereqWeight*creditPrereqWeight {
			t.Fatalf("%s should get a quarter credit from hoy.partial: %+v %v", id, p, err)
		}
	}
}
