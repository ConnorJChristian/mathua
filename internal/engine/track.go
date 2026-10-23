package engine

import (
	"log"
	"sync"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/modules"
	"github.com/chuma-beep/mathua/internal/scheduler"
)

// Tracks split the one concept graph into the sides of the site the header
// switch shows: "math" (the base install) and whatever tracks the enabled
// modules declare (internal/modules). They share storage, mastery, XP and
// spaced review; what a track changes is which concepts the scheduler may
// offer.
//
// The split is a separate DAG per track rather than a filter threaded through
// the scheduler. The scheduler walks dag.Order() in several places, and a view
// that simply does not contain the other track's concepts cannot leak one
// through a path someone forgot to filter. It needs tracks that do not depend
// on each other; a cross-track prerequisite makes that track's view fall back
// to the whole graph, with a warning.
type trackViews struct {
	once  sync.Once
	mods  *modules.Set
	dags  map[string]*concepts.DAG
	sched map[string]*scheduler.Scheduler
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
		split := map[string][]concepts.Concept{}
		for _, c := range e.dag.Order() {
			t := e.tracks.mods.TrackOf(c.Domain)
			split[t] = append(split[t], *c)
		}
		for _, t := range e.tracks.mods.Tracks() {
			dag := concepts.Empty()
			if len(split[t.ID]) > 0 {
				d, err := concepts.Build(split[t.ID])
				if err != nil {
					log.Printf("warning: %s track view: %v (serving the whole graph)", t.ID, err)
					dag = e.dag
				} else {
					dag = d
				}
			}
			e.tracks.dags[t.ID] = dag
			e.tracks.sched[t.ID] = scheduler.New(dag)
		}
	})
	return &e.tracks
}

// tracksFallbackRoot is an empty root, so a broken modules folder degrades
// to the math track rather than to no engine.
func (e *Engine) tracksFallbackRoot() string { return "/nonexistent-mathua-root" }

// NormalizeTrack maps a request parameter onto a known track.
func (e *Engine) NormalizeTrack(track string) string { return e.views().mods.Normalize(track) }

// Tracks lists the tracks this install offers, math first.
func (e *Engine) Tracks() []modules.Track { return e.views().mods.Tracks() }

// Modules is the enabled module set.
func (e *Engine) Modules() *modules.Set { return e.views().mods }

// TrackDAG is the concept graph of one track (an unknown track is math).
func (e *Engine) TrackDAG(track string) *concepts.DAG {
	v := e.views()
	return v.dags[v.mods.Normalize(track)]
}

func (e *Engine) trackScheduler(track string) *scheduler.Scheduler {
	v := e.views()
	return v.sched[v.mods.Normalize(track)]
}

// InTrack reports whether a concept belongs to a track.
func (e *Engine) InTrack(conceptID, track string) bool {
	v := e.views()
	c := e.dag.Concept(conceptID)
	if c == nil {
		// A row for a concept the graph no longer has keeps the behaviour it
		// had before tracks existed: it belongs to the math side.
		return v.mods.Normalize(track) == modules.MathTrack
	}
	return v.mods.TrackOf(c.Domain) == v.mods.Normalize(track)
}
