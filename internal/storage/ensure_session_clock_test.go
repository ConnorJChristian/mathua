package storage

import (
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/clock"
)

// FSRS Phase 2 review, L4: EnsureSession stamps started_at from the clock on
// both stores, in RFC3339 (Postgres used now()::text, which neither followed
// a simulated clock nor parsed back).
func TestEnsureSession_StartedAtIsTheClock(t *testing.T) {
	simNow := time.Date(2026, 11, 26, 17, 0, 7, 0, time.UTC)
	defer clock.NewSim(simNow).Install()()
	for name, repo := range fsrsStores(t) {
		st, err := repo.CreateStudent("ensure-" + name)
		if err != nil {
			t.Fatal(err)
		}
		id := "diag-ensure-" + name + "-" + st.ID
		if err := repo.EnsureSession(id, st.ID); err != nil {
			t.Fatal(err)
		}
		ses, err := repo.GetSession(id)
		if err != nil || ses == nil || !ses.StartedAt.Equal(simNow) {
			t.Errorf("[%s] started_at %+v (%v), want %v", name, ses, err, simNow)
		}
	}
}
