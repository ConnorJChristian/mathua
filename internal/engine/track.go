package engine

import (
	"log"
	"strings"
	"sync"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/modules"
	"github.com/chuma-beep/mathua/internal/scheduler"
)

// Tracks split the one concept graph into the sides of the site the header
// switch shows: "math" (the base install) and whatever tracks the enabled
// modules declare (internal/modules). They share storage, mastery, XP, spaced
// review and prerequisite credit; a track decides which concepts the
// scheduler may offer.
//
// Prerequisites are not sandboxed by track. A concept may need a concept in
// any domain (an econometrics topic after Mathua's coefficient of
// determination, a code task after a pandas topic), and the graph is
// sequenced as one: eligibility reads the learner's whole record, and credit
// for mastering a concept flows to everything below it, in every track
// (creditPrerequisites walks the full graph).
//
// Each track keeps two views of the graph:
//
//   - members: the track's own concepts. Reviews, the quiz and the curriculum
//     page use it, so a Math concept never appears in the Econometrics review
//     queue or domain list.
//   - offer: the members plus "bridges", the outside concepts a member
//     directly needs. The recommender uses it, so when an econometrics topic
//     waits on a statistics concept that is ready to learn, the Econometrics
//     shelf offers that concept rather than leaving the learner stuck.
type trackViews struct {
	once    sync.Once
	mods    *modules.Set
	dags    map[string]*concepts.DAG
	sched   map[string]*scheduler.Scheduler
	offer   map[string]*scheduler.Scheduler
	bridges map[string]map[string]bool
	allOnce sync.Once
	all     *scheduler.Scheduler // every track at once (AllTracks)
}

// SetModules sets the module set the engine's tracks come from. Without it
// the engine uses modules.Default(). Call before serving.
func (e *Engine) SetModules(m *modules.Set) { e.tracks.mods = m }

func (e *Engine) views() *trackViews {
	e.tracks.once.Do(func() {
		if e.tracks.mods == nil {
			m, err := modules.Default()
			if err != nil {
				log.Printf("warning: modules: %v (math track only)", err)
				m, _ = modules.Load(e.tracksFallbackRoot())
			}
			e.tracks.mods = m
		}
		e.tracks.dags = map[string]*concepts.DAG{}
		e.tracks.sched = map[string]*scheduler.Scheduler{}
		e.tracks.offer = map[string]*scheduler.Scheduler{}
		e.tracks.bridges = map[string]map[string]bool{}
		split := map[string][]concepts.Concept{}
		for _, c := range e.dag.Order() {
			t := e.tracks.mods.TrackOf(c.Domain)
			split[t] = append(split[t], *c)
		}
		for _, t := range e.tracks.mods.Tracks() {
			members := split[t.ID]
			in := make(map[string]bool, len(members))
			for _, c := range members {
				in[c.ID] = true
			}
			offer := append([]concepts.Concept(nil), members...)
			bridges := map[string]bool{}
			for _, c := range members {
				for _, pid := range c.Prerequisites {
					if in[pid] || bridges[pid] {
						continue
					}
					if p := e.dag.Concept(pid); p != nil {
						bridges[pid] = true
						offer = append(offer, *p)
					}
				}
			}
			dag := concepts.BuildView(members)
			e.tracks.dags[t.ID] = dag
			e.tracks.sched[t.ID] = scheduler.New(dag)
			e.tracks.offer[t.ID] = scheduler.New(concepts.BuildView(offer))
			e.tracks.bridges[t.ID] = bridges
		}
	})
	return &e.tracks
}

// tracksFallbackRoot is an empty root, so a broken modules folder degrades
// to the math track rather than to no engine.
func (e *Engine) tracksFallbackRoot() string { return "/nonexistent-mathua-root" }

// AllTracks is every track at once (fork change). The fork has no
// Math | Python split: one queue, one review session and one topic list
// span the learner's courses. A request that names no track gets this
// view; a request that names a module's track still gets that track (the
// per-track views stay for anything that asks for one by id).
const AllTracks = "all"

// NormalizeTrack maps a request parameter onto a known track; an empty or
// unknown one is AllTracks.
func (e *Engine) NormalizeTrack(track string) string {
	track = strings.ToLower(strings.TrimSpace(track))
	if track == "" || track == AllTracks {
		return AllTracks
	}
	for _, t := range e.views().mods.Tracks() {
		if t.ID == track {
			return t.ID
		}
	}
	return AllTracks
}

// Tracks lists the tracks this install offers, math first.
func (e *Engine) Tracks() []modules.Track { return e.views().mods.Tracks() }

// Modules is the enabled module set.
func (e *Engine) Modules() *modules.Set { return e.views().mods }

// TrackDAG is the concept graph of one track (an unknown track is math).
func (e *Engine) TrackDAG(track string) *concepts.DAG {
	track = e.NormalizeTrack(track)
	if track == AllTracks {
		return e.dag
	}
	return e.views().dags[track]
}

func (e *Engine) trackScheduler(track string) *scheduler.Scheduler {
	track = e.NormalizeTrack(track)
	if track == AllTracks {
		return e.allScheduler()
	}
	return e.views().sched[track]
}

// allScheduler is the whole graph's scheduler (AllTracks).
func (e *Engine) allScheduler() *scheduler.Scheduler {
	v := e.views()
	v.allOnce.Do(func() { v.all = scheduler.New(e.dag) })
	return v.all
}

// offerScheduler is the recommender's view of a track: its concepts plus the
// outside concepts they directly need (see trackViews).
func (e *Engine) offerScheduler(track string) *scheduler.Scheduler {
	track = e.NormalizeTrack(track)
	if track == AllTracks {
		return e.allScheduler()
	}
	return e.views().offer[track]
}

// IsBridge reports whether a concept is offered in a track it does not belong
// to, because one of the track's concepts needs it.
func (e *Engine) IsBridge(conceptID, track string) bool {
	track = e.NormalizeTrack(track)
	if track == AllTracks {
		return false
	}
	return e.views().bridges[track][conceptID]
}

// InTrack reports whether a concept belongs to a track.
func (e *Engine) InTrack(conceptID, track string) bool {
	track = e.NormalizeTrack(track)
	if track == AllTracks {
		return true
	}
	v := e.views()
	c := e.dag.Concept(conceptID)
	if c == nil {
		// A row for a concept the graph no longer has keeps the behaviour it
		// had before tracks existed: it belongs to the math side.
		return v.mods.Normalize(track) == modules.MathTrack
	}
	return v.mods.TrackOf(c.Domain) == v.mods.Normalize(track)
}
