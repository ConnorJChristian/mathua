package storage

// Phase 1 independent reviewer. review_log.at is TEXT in RFC3339Nano, and
// ReviewLogFor orders by it as a string. RFC3339Nano drops trailing zeros,
// so a whole-second instant ("...:00Z") and a later instant in the same
// second ("...:00.5Z") compare '.' < 'Z' and read back in the wrong order.
// Replay (and fsrs_doctor's replay check) applies events in this order.

import (
	"fmt"
	"testing"
	"time"
)

func TestReviewer_ReviewLogOrderWithinOneSecond(t *testing.T) {
	for tag, repo := range fsrsStores(t) {
		st, err := repo.CreateStudent(fmt.Sprintf("fsrs-order-%s-%d", tag, time.Now().UnixNano()))
		if err != nil {
			t.Fatal(err)
		}
		whole := time.Date(2026, 11, 5, 17, 0, 0, 0, time.UTC)
		later := whole.Add(500 * time.Millisecond)
		// Inserted in time order, so id order agrees with time order too.
		for i, at := range []time.Time{whole, later} {
			if _, err := repo.InsertReviewLog(ReviewLogEntry{StudentID: st.ID, ConceptID: "a", At: at,
				Source: "review", Rating: 3 - 2*i, IdempotencyKey: fmt.Sprintf("order:%d:%s", i, st.ID), EngineMode: "shadow"}); err != nil {
				t.Fatal(err)
			}
		}
		log, err := repo.ReviewLogFor(st.ID, "a")
		if err != nil || len(log) != 2 {
			t.Fatalf("[%s] %d rows, %v", tag, len(log), err)
		}
		if !log[0].At.Equal(whole) || !log[1].At.Equal(later) {
			t.Errorf("[%s] ReviewLogFor out of time order: %v then %v", tag, log[0].At, log[1].At)
		}
		_ = repo.DeleteAccount(st.ID)
	}
}
