package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/chuma-beep/mathua/internal/scheduler"
)

// dueReport is what each surface says is due for review at one moment
// (FSRS plan Phase 0: one due rule, read everywhere). Disagreements names
// every pair that differs.
type dueReport struct {
	Due       []string // engine.DueReviewIDs: the rule itself
	Count     int      // /api/reviews/due, the Study banner and the sidebar badge
	CourseDue []string // the course pages' due marks (the banner's fallback), on the courses' own topics
	UpNext    bool     // Up next shows a review card
	UpNextWhy string   // its reason, "review due N days ago (M due)"
	Review    []string // the review session, served to exhaustion
	Feed      []string // the mixed session's review serves
	FeedFirst string   // the mixed session's first question, "" when none

	Disagreements []string
}

func (d dueReport) String() string {
	return fmt.Sprintf("due=%d count=%d course=%d upnext=%v review=%d feed=%d", len(d.Due), d.Count, len(d.CourseDue), d.UpNext, len(d.Review), len(d.Feed))
}

var dueSuffix = regexp.MustCompile(`\((\d+) due\)$`)

// dueSurfaces reads what is due from every surface, without answering
// anything: the review session is served to exhaustion on a throwaway
// session (a review session serves a topic once, so the loop ends), and
// the mixed session is asked for as many questions as there are due
// topics, plus a few.
func (r *replay) dueSurfaces() dueReport {
	var rep dueReport
	progress, err := r.repo.GetAllProgress(r.sid)
	if err != nil {
		rep.Disagreements = append(rep.Disagreements, "progress: "+err.Error())
		return rep
	}
	now := time.Now()
	rep.Due = r.eng.DueReviewIDs(r.sid, "", progress, now)
	rep.Count, _ = r.eng.ReviewsDue(r.sid)
	dueSet := map[string]bool{}
	for _, id := range rep.Due {
		dueSet[id] = true
	}
	// The course page marks a followed course's own topics
	// (courseview.go); the Study banner falls back to their union.
	courseDue := r.eng.DueReviewSet(r.sid, progress, now)
	for cid := range courseDue {
		if r.own[cid] {
			rep.CourseDue = append(rep.CourseDue, cid)
		}
	}
	sort.Strings(rep.CourseDue)

	if rec, err := r.eng.RecommendNext(r.sid, nil); err == nil {
		for _, rc := range recsOf(rec) {
			if rc.Kind == scheduler.KindReview {
				rep.UpNext = true
				rep.UpNextWhy = rc.Why
			}
		}
	}

	if rs, err := r.repo.CreateSession(r.sid); err == nil {
		seen := map[string]bool{}
		for i := 0; i <= len(progress); i++ {
			q, err := r.eng.NextReviewQuestion(rs.ID, r.sid)
			if err != nil || q == nil {
				break
			}
			if seen[q.ConceptID] {
				rep.Disagreements = append(rep.Disagreements, "review session served "+q.ConceptID+" twice")
				break
			}
			seen[q.ConceptID] = true
			rep.Review = append(rep.Review, q.ConceptID)
			if !strings.HasPrefix(q.Reason, "review due") {
				rep.Disagreements = append(rep.Disagreements, fmt.Sprintf("review session reason for %s is %q", q.ConceptID, q.Reason))
			}
		}
		r.repo.DeleteActiveSession(rs.ID)
	}

	if fs, err := r.repo.CreateSession(r.sid); err == nil {
		for i := 0; i < len(rep.Due)+3; i++ {
			q, err := r.eng.NextQuestion(fs.ID, r.sid)
			if err != nil || q == nil {
				break
			}
			if i == 0 {
				rep.FeedFirst = q.ConceptID
			}
			if q.IsReview {
				rep.Feed = append(rep.Feed, q.ConceptID)
				if !strings.HasPrefix(q.Reason, "review due") {
					rep.Disagreements = append(rep.Disagreements, fmt.Sprintf("mixed session reason for %s is %q", q.ConceptID, q.Reason))
				}
			}
		}
		r.repo.DeleteActiveSession(fs.ID)
	}

	// Compare.
	if rep.Count != len(rep.Due) {
		rep.Disagreements = append(rep.Disagreements, fmt.Sprintf("count %d, due set %d", rep.Count, len(rep.Due)))
	}
	for _, cid := range rep.CourseDue {
		if !dueSet[cid] {
			rep.Disagreements = append(rep.Disagreements, "course page marks "+cid+" due, the rule does not")
		}
	}
	for _, cid := range rep.Due {
		if r.own[cid] && !courseDue[cid] {
			rep.Disagreements = append(rep.Disagreements, "course page does not mark "+cid+" due")
		}
	}
	if rep.UpNext != (len(rep.Due) > 0) {
		rep.Disagreements = append(rep.Disagreements, fmt.Sprintf("Up next review card %v with %d due", rep.UpNext, len(rep.Due)))
	}
	if rep.UpNext {
		n := 1
		if m := dueSuffix.FindStringSubmatch(rep.UpNextWhy); m != nil {
			n, _ = strconv.Atoi(m[1])
		}
		if n != len(rep.Due) || !strings.HasPrefix(rep.UpNextWhy, "review due") {
			rep.Disagreements = append(rep.Disagreements, fmt.Sprintf("Up next reason %q with %d due", rep.UpNextWhy, len(rep.Due)))
		}
	}
	served := append([]string(nil), rep.Review...)
	sort.Strings(served)
	want := append([]string(nil), rep.Due...)
	sort.Strings(want)
	if strings.Join(served, ",") != strings.Join(want, ",") {
		rep.Disagreements = append(rep.Disagreements, fmt.Sprintf("review session served %v, due set %v", served, want))
	}
	for _, cid := range rep.Feed {
		if !dueSet[cid] {
			rep.Disagreements = append(rep.Disagreements, "mixed session reviewed "+cid+", not due")
		}
	}
	if len(rep.Due) > 0 && rep.FeedFirst != "" && !dueSet[rep.FeedFirst] && len(r.own) > 0 {
		rep.Disagreements = append(rep.Disagreements, fmt.Sprintf("mixed session opened with %s while %d reviews are due", rep.FeedFirst, len(rep.Due)))
	}
	return rep
}
