package review

import (
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/fsrs"
)

var t0 = time.Date(2026, 11, 2, 17, 0, 0, 0, time.UTC)

func TestApply_FirstEventIsTheReferenceFirstReview(t *testing.T) {
	a, err := DefaultParams().Apply(Memory{}, time.Time{}, Step{Rating: fsrs.Good, At: t0, Cap: true, LapseBelowR: 0.95})
	if err != nil || a.Held != "" || !a.Raised {
		t.Fatalf("%+v %v", a, err)
	}
	if want := Initial(fsrs.Good); a.Next != want || a.Next.Stability != fsrs.DefaultParameters()[2] {
		t.Errorf("first Good: %+v, want %+v (S = w[2])", a.Next, want)
	}
}

func TestApply_PracticeLapseThreshold(t *testing.T) {
	p := DefaultParams()
	m := Initial(fsrs.Good) // S 2.31: R(1 day) ~ 0.95, R(5 days) well below
	high, err := p.Apply(m, t0, Step{Rating: fsrs.Again, At: t0.Add(2 * time.Hour), LapseBelowR: 0.95})
	if err != nil || high.Held != HeldLapseRHigh || high.Next != m || high.R != 1 {
		t.Fatalf("a same-day miss at R 1 must be held: %+v %v", high, err)
	}
	low, err := p.Apply(m, t0, Step{Rating: fsrs.Again, At: t0.Add(5 * 24 * time.Hour), LapseBelowR: 0.95})
	if err != nil || low.Held != "" || low.R >= 0.95 || low.Next.Lapses != 1 || low.Next.Stability >= m.Stability {
		t.Fatalf("a miss at low R is a lapse: %+v %v", low, err)
	}
	// No threshold: a review miss always applies.
	rev, _ := p.Apply(m, t0, Step{Rating: fsrs.Again, At: t0.Add(2 * time.Hour)})
	if rev.Held != "" || rev.Next.Lapses != 1 {
		t.Errorf("a review miss applies whatever R: %+v", rev)
	}
}

func TestApply_DailyCap(t *testing.T) {
	p := DefaultParams()
	// After a first Again (S = w[0]) a same-day Good raises S; after a
	// first Good the default weights' short-term factor floors to x1.
	m := Initial(fsrs.Again)
	held, _ := p.Apply(m, t0, Step{Rating: fsrs.Good, At: t0.Add(3 * time.Hour), Cap: true, RaisedToday: true})
	if held.Held != HeldDailyCap || held.Next != m {
		t.Fatalf("a second raise the same day must be held: %+v", held)
	}
	ok, _ := p.Apply(m, t0, Step{Rating: fsrs.Good, At: t0.Add(3 * time.Hour), Cap: false, RaisedToday: true})
	if ok.Held != "" || ok.Next.Stability <= m.Stability {
		t.Fatalf("without the cap the same-day Good raises S: %+v", ok)
	}
	// Again lowers S, so the cap never holds it.
	miss, _ := p.Apply(m, t0, Step{Rating: fsrs.Again, At: t0.Add(3 * time.Hour), Cap: true, RaisedToday: true})
	if miss.Held != "" || miss.Raised {
		t.Fatalf("a lapse is never held by the cap: %+v", miss)
	}
}

func TestApply_InvalidRatingFailsClosed(t *testing.T) {
	m := Initial(fsrs.Good)
	a, err := DefaultParams().Apply(m, t0, Step{Rating: 0, At: t0.Add(48 * time.Hour)})
	if !errors.Is(err, ErrInvalidRating) || a.Next != m {
		t.Fatalf("%+v %v", a, err)
	}
}

// Event sourcing with held events: whatever Apply committed, Replay of the
// log (held rows included) gives the same memory and the same last review.
func TestApply_ReplayReproducesWithHeldEvents(t *testing.T) {
	p := DefaultParams()
	rng := rand.New(rand.NewSource(20261105))
	for trial := 0; trial < 300; trial++ {
		var m Memory
		var last time.Time
		var log []Event
		at := t0
		raisedDay := ""
		for i := 0; i < 2+rng.Intn(12); i++ {
			at = at.Add(time.Duration(rng.Intn(60*60*24*9)) * time.Second)
			day := at.Format("2006-01-02")
			rating := fsrs.Good
			if rng.Intn(3) == 0 {
				rating = fsrs.Again
			}
			st := Step{Rating: rating, At: at, Cap: true, RaisedToday: raisedDay == day}
			if rating == fsrs.Again && rng.Intn(2) == 0 {
				st.LapseBelowR = 0.95
			}
			a, err := p.Apply(m, last, st)
			if err != nil {
				t.Fatal(err)
			}
			log = append(log, Event{At: at, Rating: rating, ElapsedDays: a.ElapsedDays, Source: "review", SAfter: a.Next.Stability, DAfter: a.Next.Difficulty, Held: a.Held})
			if a.Held == "" {
				m, last = a.Next, at
				if a.Raised {
					raisedDay = day
				}
			}
		}
		got, gotLast, err := ReplayLast(p, log)
		if err != nil {
			t.Fatal(err)
		}
		if d := Diff(m, got); d != "" || !gotLast.Equal(last) {
			t.Fatalf("trial %d: %s; last %v vs %v", trial, d, gotLast, last)
		}
	}
}
