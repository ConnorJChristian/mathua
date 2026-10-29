package server

import (
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/engine"
	"github.com/chuma-beep/mathua/internal/mastery"
	"github.com/chuma-beep/mathua/internal/modules"
	"github.com/chuma-beep/mathua/internal/scheduler"
	"github.com/chuma-beep/mathua/internal/storage"
)

// Fork change: the course view. One request gives a course page everything it
// shows: the course's modules with each topic's state, reading and videos, its
// texts, what to do next in it, and how many of its topics are due a review.
// It is a view over the same engine state the recommender and the lesson pages
// use, so the course page and the Study queue never disagree.

type courseTopic struct {
	ID       string          `json:"id"`
	Label    string          `json:"label"`
	Status   string          `json:"status"` // unseen, learning, practicing, mastered
	Locked   bool            `json:"locked"`
	Due      bool            `json:"due"`
	Attempts int             `json:"attempts"`
	Reading  *readingOption  `json:"reading,omitempty"`
	Texts    []readingOption `json:"texts,omitempty"`
	Videos   []courseVideo   `json:"videos,omitempty"`
	Notes    []courseNote    `json:"notes,omitempty"`  // lecture notes (fork change)
	Cards    int             `json:"cards,omitempty"`  // flashcard decks that cover it
	Pinned   bool            `json:"pinned,omitempty"` // pinned for this week (fork change)
	// Placed: the placement test put the learner at or above this topic
	// (reachable, not yet studied). Shown as "placed", not "unseen" (Q33).
	Placed bool `json:"placed,omitempty"`
}

type courseModule struct {
	Title    string        `json:"title"`
	Topics   []courseTopic `json:"topics"`
	Mastered int           `json:"mastered"`
	// AfterDue: taught after the course's due date (planning.Module.AfterDue).
	AfterDue bool `json:"after_due,omitempty"`
}

type courseVideo struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	Kind     string `json:"kind"` // youtube, file or link
	Start    int    `json:"start,omitempty"`
	Duration int    `json:"duration,omitempty"`
	Note     string `json:"note,omitempty"`
}

// courseNote is a page of lecture notes (fork change).
type courseNote struct {
	Title   string `json:"title"`
	Unit    string `json:"unit,omitempty"`
	UnitSeq int    `json:"unit_seq,omitempty"`
	Seq     int    `json:"seq,omitempty"`
	URL     string `json:"url"`
	Topic   string `json:"topic,omitempty"` // the topic it is attached to, in the course-level list
}

type courseText struct {
	Key       string `json:"key"`
	Title     string `json:"title"`
	Kind      string `json:"kind"`
	URL       string `json:"url,omitempty"`
	LibraryID string `json:"library_id,omitempty"`
}

// resolveCourseText turns one of a course's authority-text keys into what the
// course page lists. "notes:<module>" (a module's lecture notes, which a
// course ranks among its texts so they open in the lesson) used to fall
// through to the file case and was listed as "notes:ma-program (online)".
func resolveCourseText(tx string, titles, libIDs map[string]string) courseText {
	ct := courseText{Key: tx}
	switch {
	case strings.HasPrefix(tx, "web:"):
		ct.Kind, ct.Title = "web", strings.TrimPrefix(tx, "web:")
	case strings.HasPrefix(tx, "notes:"):
		ct.Kind, ct.Title = "notes", "Lecture notes"
	default:
		ct.Kind = "pdf"
		ct.Title = titles[tx]
		if ct.Title == "" {
			ct.Title = tx
		}
		if i := strings.IndexByte(tx, '/'); i > 0 {
			ct.URL = "/api/modules/" + url.PathEscape(tx[:i]) + "/files/" + escapePath(tx[i+1:])
		}
		ct.LibraryID = libIDs[tx]
	}
	return ct
}

type courseView struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Term string `json:"term,omitempty"`
	// Due and DaysLeft: the course's date (fork change), when it has one.
	Due         string         `json:"due,omitempty"`
	DaysLeft    *int           `json:"days_left,omitempty"`
	Description string         `json:"description"`
	Followed    bool           `json:"followed"`
	Modules     []courseModule `json:"modules"`
	// Prerequisites are the topics the plan needs that are not on the plan
	// itself (the path's fill-ins), unmastered ones only, in path order.
	// Progress counts the plan; these are what stands between you and it.
	Prerequisites []courseTopic `json:"prerequisites"`
	Texts         []courseText  `json:"texts"`
	Videos        []courseVideo `json:"videos"`
	// Notes: every page of lecture notes in the course, in lecture order
	// (the course's own, then each topic's), fork change.
	Notes    []courseNote `json:"notes"`
	Total    int          `json:"total"`
	Mastered int          `json:"mastered"`
	// DueTotal and DueMastered count only the topics due by the course's
	// date: modules marked after_due are left out. Pacing ("N topics a week
	// to finish") is computed from these.
	DueTotal    int          `json:"due_total"`
	DueMastered int          `json:"due_mastered"`
	Started     int          `json:"started"`
	ReviewsDue  int          `json:"reviews_due"`
	NextTopic   *courseTopic `json:"next_topic,omitempty"`
	// NeedsPlacement: no plan topic is open yet and the plan rests on
	// prerequisites the learner has not shown; the placement test is the way
	// in (a module with no placement never sets it).
	NeedsPlacement bool `json:"needs_placement"`
	// Tests: which course tests this course offers and how long they are
	// (fork change; engine.CourseTests has the viability rule).
	Tests *engine.CourseTestInfo `json:"tests,omitempty"`
}

// GET /api/courses/view?id=plan.x — the course page's data.
func (s *Server) handleCourseView(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, 405)
		return
	}
	id := r.URL.Query().Get("id")
	pl := s.eng.GetPlanner()
	if pl == nil || id == "" || pl.Course(id) == nil {
		writeError(w, "course not found", 404)
		return
	}
	view, err := s.courseView(s.lessonStudentID(r), id)
	if err != nil {
		writeError(w, "failed to load course", 500)
		return
	}
	writeJSON(w, view)
}

func (s *Server) courseView(studentID, courseID string) (*courseView, error) {
	pl := s.eng.GetPlanner()
	c := pl.Course(courseID)
	path, err := pl.PathForCourse(courseID)
	if err != nil {
		return nil, err
	}
	var progress map[string]*storage.ConceptProgress
	followed := false
	if studentID != "" {
		if p, err := s.repo.GetAllProgress(studentID); err == nil {
			progress = p
		}
		if st, err := s.repo.GetStudent(studentID); err == nil && st != nil {
			for _, a := range engine.ActiveCourses(st.CourseID) {
				followed = followed || a == courseID
			}
		}
	}
	snaps := s.eng.ConceptSnapshots(studentID, progress)
	now := time.Now()
	mods := s.eng.Modules()
	decks := s.deckCounts()

	out := &courseView{ID: c.ID, Name: c.Name, Term: c.Term, Due: c.Due, Description: c.Description, Followed: followed}
	if due, ok := c.DueDate(); ok {
		d := int(due.Sub(now).Hours() / 24)
		out.DaysLeft = &d
	}
	pinned := map[string]bool{}
	for _, p := range s.eng.Pins(studentID) {
		pinned[p.Concept] = true
	}
	topic := func(cid string) *courseTopic {
		cc := s.eng.GetDAG().Concept(cid)
		if cc == nil {
			return nil
		}
		t := &courseTopic{ID: cid, Label: cc.Label, Status: "unseen", Cards: decks[cid], Pinned: pinned[cid]}
		if p := progress[cid]; p != nil && p.Attempts > 0 {
			t.Status = strings.ToLower(p.Status)
			t.Attempts = p.Attempts
			// Due means what the review session serves: a mastered topic
			// whose review date has come (Q2). A learning topic's
			// next_review_due is a practice date, not a review.
			t.Due = p.Status == string(mastery.StatusMastered) && p.NextReviewDue != nil && !p.NextReviewDue.After(now)
		} else if p != nil && p.PlacementSeeded {
			// Placed by the placement test: reachable, not studied (Q33).
			t.Placed = true
		}
		el := scheduler.EligibilityOf(snaps, now, cc)
		t.Locked = !el.Eligible
		opts := s.readingOptions(cid)
		if len(opts) > 0 {
			pick := 0
			for _, tx := range c.TextsFor(cid) {
				found := false
				for i, o := range opts {
					if textMatches(tx, o) {
						pick, found = i, true
						break
					}
				}
				if found {
					break
				}
			}
			opts[pick].Selected = true
			t.Reading = &opts[pick]
			t.Texts = opts
		}
		t.Videos = courseVideos(mods, cid)
		t.Notes = courseNotes(mods, cid)
		return t
	}

	// Modules: the course's own, or the plan as one unit. Plan topics not in
	// any module go in a last unit; the path's prerequisite fill-ins are
	// listed apart, unmastered ones only.
	onPlan := map[string]bool{}
	for _, cid := range c.Plan {
		onPlan[cid] = true
	}
	for _, cid := range c.Targets {
		onPlan[cid] = true
	}
	placed := map[string]bool{}
	for _, m := range c.Modules {
		cm := courseModule{Title: m.Title, AfterDue: m.AfterDue}
		for _, cid := range m.Topics {
			if t := topic(cid); t != nil && !placed[cid] {
				placed[cid] = true
				cm.Topics = append(cm.Topics, *t)
			}
		}
		out.Modules = append(out.Modules, cm)
	}
	var rest []courseTopic
	out.Prerequisites = []courseTopic{}
	for _, cc := range path.Concepts {
		if placed[cc.ID] {
			continue
		}
		t := topic(cc.ID)
		if t == nil {
			continue
		}
		placed[cc.ID] = true
		if onPlan[cc.ID] {
			rest = append(rest, *t)
		} else if t.Status != "mastered" && !t.Placed {
			// A prerequisite the placement test already put the learner
			// above is not work to list (Q27).
			out.Prerequisites = append(out.Prerequisites, *t)
		}
	}
	if len(rest) > 0 {
		title := "Topics"
		if len(c.Modules) > 0 {
			title = "Also on the plan"
		}
		out.Modules = append(out.Modules, courseModule{Title: title, Topics: rest})
	}
	for i := range out.Modules {
		for _, t := range out.Modules[i].Topics {
			out.Total++
			if !out.Modules[i].AfterDue {
				out.DueTotal++
			}
			if t.Status == "mastered" {
				out.Modules[i].Mastered++
				out.Mastered++
				if !out.Modules[i].AfterDue {
					out.DueMastered++
				}
			}
			if t.Attempts > 0 {
				out.Started++
			}
			if t.Due {
				out.ReviewsDue++
			}
			if out.NextTopic == nil && t.Status != "mastered" && !t.Locked {
				nt := t
				out.NextTopic = &nt
			}
		}
	}
	// None of the plan is open and nothing has been started: place first.
	if out.NextTopic == nil && out.Started == 0 && len(out.Prerequisites) > 0 {
		for _, m := range out.Modules {
			for _, t := range m.Topics {
				if cc := s.eng.GetDAG().Concept(t.ID); cc != nil && mods.HasPlacement(mods.TrackOf(cc.Domain)) {
					out.NeedsPlacement = true
				}
			}
		}
	}
	// Every open plan topic done, or none open yet: the next thing is the
	// first open prerequisite.
	//
	// "First" is nearest to the plan, not first in topological order: the
	// order starts at the roots, so a course whose first plan topic was locked
	// offered Single-digit addition as its next step (Q27). Walk back from the
	// first unfinished plan topic and take the closest open prerequisite.
	if out.NextTopic == nil && len(out.Prerequisites) > 0 {
		open := map[string]courseTopic{}
		for _, t := range out.Prerequisites {
			if !t.Locked {
				open[t.ID] = t
			}
		}
		for _, cid := range append(append([]string{}, c.Plan...), c.Targets...) {
			if p := progress[cid]; p != nil && p.Status == string(mastery.StatusMastered) {
				continue
			}
			if t, ok := nearestOpenPrereq(s.eng.GetDAG(), cid, open); ok {
				nt := t
				out.NextTopic = &nt
				break
			}
		}
		if out.NextTopic == nil {
			for _, t := range out.Prerequisites {
				if !t.Locked {
					nt := t
					out.NextTopic = &nt
					break
				}
			}
		}
	}

	// Texts, resolved to titles and library ids.
	titles := map[string]string{}
	libIDs := map[string]string{}
	for _, b := range mods.Books() {
		key := b.Module + "/" + b.File
		titles[key] = b.Title
		libIDs[key] = slug(key)
	}
	for _, tx := range c.Texts {
		out.Texts = append(out.Texts, resolveCourseText(tx, titles, libIDs))
	}
	out.Videos = courseVideos(mods, c.ID)
	// Lecture notes: the course's own pages, then every topic's, each once,
	// in unit order and sequence.
	out.Notes = courseNotes(mods, c.ID)
	seenNote := map[string]bool{}
	for _, n := range out.Notes {
		seenNote[n.URL] = true
	}
	for _, m := range out.Modules {
		for _, t := range m.Topics {
			for _, n := range t.Notes {
				if !seenNote[n.URL] {
					seenNote[n.URL] = true
					n.Topic = t.ID
					out.Notes = append(out.Notes, n)
				}
			}
		}
	}
	// Units in course order (unit_seq), unnumbered ones in first-seen order
	// after them; pages by seq inside a unit.
	unitOrder := map[string]int{}
	for _, n := range out.Notes {
		if n.UnitSeq > 0 {
			unitOrder[n.Unit] = n.UnitSeq
		}
	}
	next := 1 << 20
	for _, n := range out.Notes {
		if _, ok := unitOrder[n.Unit]; !ok {
			unitOrder[n.Unit] = next
			next++
		}
	}
	sort.SliceStable(out.Notes, func(a, b int) bool {
		if ua, ub := unitOrder[out.Notes[a].Unit], unitOrder[out.Notes[b].Unit]; ua != ub {
			return ua < ub
		}
		return out.Notes[a].Seq < out.Notes[b].Seq
	})
	if info, err := s.eng.CourseTests(studentID, courseID); err == nil {
		out.Tests = info
	}
	if out.Texts == nil {
		out.Texts = []courseText{}
	}
	if out.Videos == nil {
		out.Videos = []courseVideo{}
	}
	if out.Notes == nil {
		out.Notes = []courseNote{}
	}
	return out, nil
}

// deckCounts is how many flashcard decks cover each generated topic
// (engine/cards.go): the course page can say "12 of your cards".
func (s *Server) deckCounts() map[string]int {
	out := map[string]int{}
	dag := s.eng.GetDAG()
	for _, c := range dag.Order() {
		if !strings.HasPrefix(c.ID, "anki.") {
			continue
		}
		for _, p := range c.Prerequisites {
			if !strings.HasPrefix(p, "anki.") {
				out[p]++
			}
		}
	}
	return out
}

func courseNotes(mods *modules.Set, id string) []courseNote {
	var out []courseNote
	for _, n := range mods.NotesFor(id) {
		cn := courseNote{Title: n.Title, Unit: n.Unit, UnitSeq: n.UnitSeq, Seq: n.Seq, URL: n.URL}
		if n.File != "" {
			cn.URL = "/api/modules/" + url.PathEscape(n.Module) + "/files/" + escapePath(n.File)
		}
		out = append(out, cn)
	}
	return out
}

func courseVideos(mods *modules.Set, id string) []courseVideo {
	var out []courseVideo
	for _, v := range mods.VideosFor(id) {
		cv := courseVideo{Title: v.Title, Start: v.Start, Duration: v.Duration, Note: v.Note}
		switch {
		case v.File != "":
			cv.Kind = "file"
			cv.URL = "/api/modules/" + url.PathEscape(v.Module) + "/files/" + escapePath(v.File)
		case strings.Contains(v.URL, "youtube.com/") || strings.Contains(v.URL, "youtu.be/"):
			cv.Kind, cv.URL = "youtube", v.URL
		default:
			cv.Kind, cv.URL = "link", v.URL
		}
		out = append(out, cv)
	}
	return out
}

// nearestOpenPrereq walks a topic's prerequisites breadth-first and returns
// the first one in open: the closest step toward the topic.
func nearestOpenPrereq(dag *concepts.DAG, target string, open map[string]courseTopic) (courseTopic, bool) {
	seen := map[string]bool{target: true}
	queue := []string{target}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		for _, p := range dag.PrereqsOf(id) {
			if seen[p.ID] {
				continue
			}
			seen[p.ID] = true
			if t, ok := open[p.ID]; ok {
				return t, true
			}
			queue = append(queue, p.ID)
		}
	}
	return courseTopic{}, false
}
