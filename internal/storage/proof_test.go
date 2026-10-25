package storage

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/chuma-beep/mathua/internal/proof"
)

// Both dialects persist the proof grader's spend and grade cache.
var (
	_ proof.Store = (*SQLiteStore)(nil)
	_ proof.Store = (*PostgresStore)(nil)
)

func TestProofSpendAndGradeCache(t *testing.T) {
	ctx := context.Background()
	for name, repo := range aiStores(t) {
		st, ok := repo.(proof.Store)
		if !ok {
			t.Fatalf("%s: store does not implement proof.Store", name)
		}
		month := fmt.Sprintf("t%d", time.Now().UnixNano())
		if v, err := st.ProofSpend(ctx, month); err != nil || v != 0 {
			t.Fatalf("%s: empty month %v %v", name, v, err)
		}
		if err := st.AddProofSpend(ctx, month, 0.25, 1000, 200); err != nil {
			t.Fatal(err)
		}
		if err := st.AddProofSpend(ctx, month, 0.5, 1000, 200); err != nil {
			t.Fatal(err)
		}
		st.AddProofSpend(ctx, month+"-next", 9, 1, 1)
		if v, err := st.ProofSpend(ctx, month); err != nil || math.Abs(v-0.75) > 1e-9 {
			t.Fatalf("%s: spend %v %v, want 0.75 for the month alone", name, v, err)
		}
		if _, ok, err := st.ProofGrade(ctx, "k-"+month); err != nil || ok {
			t.Fatalf("%s: missing grade found", name)
		}
		st.PutProofGrade(ctx, "k-"+month, `{"score":3}`)
		st.PutProofGrade(ctx, "k-"+month, `{"score":5}`)
		if v, ok, err := st.ProofGrade(ctx, "k-"+month); err != nil || !ok || v != `{"score":5}` {
			t.Fatalf("%s: grade %q %v %v", name, v, ok, err)
		}
	}
}
