package proof

import (
	"context"
	"sync"
)

// Store is the persistence the proof grader needs: the month's spend, so the
// budget holds across restarts, and finished grades, so a resubmitted
// identical proof costs nothing. The SQLite and Postgres stores implement it
// (internal/storage).
type Store interface {
	// ProofSpend is the dollars recorded for a month ("2026-10").
	ProofSpend(ctx context.Context, month string) (float64, error)
	// AddProofSpend adds one grading's cost and tokens to a month.
	AddProofSpend(ctx context.Context, month string, usd float64, inputTokens, outputTokens int) error
	// ProofGrade returns a cached grade (JSON) by key.
	ProofGrade(ctx context.Context, key string) (string, bool, error)
	// PutProofGrade stores a grade (JSON) under key.
	PutProofGrade(ctx context.Context, key, value string) error
}

// MemStore is an in-process Store for tests and tools. Its spend does not
// survive a restart, so the server uses the database store.
type MemStore struct {
	mu     sync.Mutex
	spend  map[string]float64
	grades map[string]string
}

// NewMemStore returns an empty MemStore.
func NewMemStore() *MemStore {
	return &MemStore{spend: map[string]float64{}, grades: map[string]string{}}
}

func (m *MemStore) ProofSpend(_ context.Context, month string) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.spend[month], nil
}

func (m *MemStore) AddProofSpend(_ context.Context, month string, usd float64, _, _ int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.spend[month] += usd
	return nil
}

func (m *MemStore) ProofGrade(_ context.Context, key string) (string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.grades[key]
	return v, ok, nil
}

func (m *MemStore) PutProofGrade(_ context.Context, key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.grades[key] = value
	return nil
}
