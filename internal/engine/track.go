package engine

import (
	"log"
	"sync"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/pytrack"
	"github.com/chuma-beep/mathua/internal/scheduler"
)

// Tracks split the one concept graph into the Math and Python sides of the
// site. They share storage, mastery, XP and spaced review; what a track
// changes is which concepts the scheduler may offer.
//
// The split is a separate DAG per track rather than a filter threaded through
// the scheduler. The scheduler walks dag.Order() in several places, and a view
// that simply does not contain the other track's concepts cannot leak one
// through a path someone forgot to filter. It works because the tracks never
// depend on each other: pytrack's tests reject a Python concept with a
// mathematics prerequisite, and no mathematics concept names a Python one.
type trackViews struct {
	once  sync.Once
	dags  map[string]*concepts.DAG
	sched map[string]*scheduler.Scheduler
}

func (e *Engine) views() *trackViews {
	e.tracks.once.Do(func() {
		e.tracks.dags = map[string]*concepts.DAG{}
		e.tracks.sched = map[string]*scheduler.Scheduler{}
		split := map[string][]concepts.Concept{}
		for _, c := range e.dag.Order() {
			t := pytrack.TrackOfDomain(c.Domain)
			split[t] = append(split[t], *c)
		}
		for _, t := range []string{pytrack.TrackMath, pytrack.TrackPython} {
			dag := e.dag
			if len(split[t]) > 0 {
				d, err := concepts.Build(split[t])
				if err != nil {
					// A cross-track prerequisite would land here. Serving the whole
					// graph is the safe degradation: nothing is hidden, the tracks
					// are just not separated.
					log.Printf("warning: %s track view: %v (serving the whole graph)", t, err)
				} else {
					dag = d
				}
			}
			e.tracks.dags[t] = dag
			e.tracks.sched[t] = scheduler.New(dag)
		}
	})
	return &e.tracks
}

// TrackDAG is the concept graph of one track ("math" or "python"; anything
// else is math).
func (e *Engine) TrackDAG(track string) *concepts.DAG {
	return e.views().dags[pytrack.NormalizeTrack(track)]
}

func (e *Engine) trackScheduler(track string) *scheduler.Scheduler {
	return e.views().sched[pytrack.NormalizeTrack(track)]
}

// InTrack reports whether a concept belongs to a track.
func (e *Engine) InTrack(conceptID, track string) bool {
	c := e.dag.Concept(conceptID)
	if c == nil {
		// A row for a concept the graph no longer has keeps the behaviour it
		// had before tracks existed: it belongs to the Math side.
		return pytrack.NormalizeTrack(track) == pytrack.TrackMath
	}
	return pytrack.TrackOfDomain(c.Domain) == pytrack.NormalizeTrack(track)
}
