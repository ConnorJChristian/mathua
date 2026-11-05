package fsrs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// referenceVectors is the schema written by tools/fsrs-reference/generate_vectors.py.
// It is generated from the installed fsrs 6.3.2 package, never from this package.
type referenceVectors struct {
	SchemaVersion int `json:"schema_version"`
	Reference     struct {
		Distribution string `json:"distribution"`
		Version      string `json:"version"`
		Python       string `json:"python"`
		Scheduler    struct {
			Parameters             []float64 `json:"parameters"`
			DesiredRetention       float64   `json:"desired_retention"`
			LearningStepsSeconds   []float64 `json:"learning_steps_seconds"`
			RelearningStepsSeconds []float64 `json:"relearning_steps_seconds"`
			MaximumInterval        int       `json:"maximum_interval"`
			EnableFuzzing          bool      `json:"enable_fuzzing"`
		} `json:"scheduler"`
	} `json:"reference"`
	Cases struct {
		Constants struct {
			Decay  float64 `json:"decay"`
			Factor float64 `json:"factor"`
		} `json:"constants"`
		InitialStability  []referenceValue `json:"initial_stability"`
		InitialDifficulty struct {
			Clamped   []referenceValue `json:"clamped"`
			Unclamped []referenceValue `json:"unclamped"`
		} `json:"initial_difficulty"`
		NextDifficulty []struct {
			referenceValue
			Difficulty float64 `json:"difficulty"`
		} `json:"next_difficulty"`
		NextStability []struct {
			referenceValue
			Stability      float64 `json:"stability"`
			Difficulty     float64 `json:"difficulty"`
			Retrievability float64 `json:"retrievability"`
		} `json:"next_stability"`
		Retrievability []struct {
			referenceValue
			Stability   float64 `json:"stability"`
			ElapsedDays int     `json:"elapsed_days"`
		} `json:"retrievability"`
		Interval []struct {
			Stability float64 `json:"stability"`
			Value     int     `json:"value"`
		} `json:"interval"`
		IntervalForRetention []struct {
			Stability float64 `json:"stability"`
			Retention float64 `json:"retention"`
			Value     int     `json:"value"`
		} `json:"interval_for_retention"`
	} `json:"cases"`
}

type referenceValue struct {
	Rating int     `json:"rating"`
	Value  float64 `json:"value"`
}

func loadReferenceVectors(t *testing.T) referenceVectors {
	t.Helper()
	path := filepath.Join("..", "..", "tools", "fsrs-reference", "vectors.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read reference vectors: %v", err)
	}
	var v referenceVectors
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("decode reference vectors: %v", err)
	}
	return v
}

// TestReferenceVectors compares Mathua's pure FSRS-6 functions with values
// produced by the pinned fsrs 6.3.2 package under tools/fsrs-reference.
//
// This test is deliberately separate from TestValues: TestValues records the
// numbers literally so a formula change fails without needing Python, while
// this test proves those numbers still come from the reference rather than
// from Mathua's own output.
func TestReferenceVectors(t *testing.T) {
	v := loadReferenceVectors(t)

	t.Run("reference metadata", func(t *testing.T) {
		if v.SchemaVersion != 1 {
			t.Fatalf("schema_version = %d, want 1", v.SchemaVersion)
		}
		if v.Reference.Distribution != "fsrs" {
			t.Fatalf("distribution = %q, want fsrs", v.Reference.Distribution)
		}
		if v.Reference.Version != "6.3.2" {
			t.Fatalf("version = %q, want 6.3.2", v.Reference.Version)
		}
		if v.Reference.Scheduler.DesiredRetention != DefaultRetention {
			t.Fatalf("desired retention = %v, want %v", v.Reference.Scheduler.DesiredRetention, DefaultRetention)
		}
		if v.Reference.Scheduler.MaximumInterval != MaxIntervalDays {
			t.Fatalf("maximum interval = %d, want %d", v.Reference.Scheduler.MaximumInterval, MaxIntervalDays)
		}
		if v.Reference.Scheduler.EnableFuzzing {
			t.Fatal("fuzzing must be disabled for reproducible vectors")
		}
		if len(v.Reference.Scheduler.LearningStepsSeconds) != 0 || len(v.Reference.Scheduler.RelearningStepsSeconds) != 0 {
			t.Fatal("reference must have no learning or relearning steps")
		}
		if len(v.Reference.Scheduler.Parameters) != len(DefaultParameters()) {
			t.Fatalf("parameter count = %d, want %d", len(v.Reference.Scheduler.Parameters), len(DefaultParameters()))
		}
		for i, want := range DefaultParameters() {
			got := v.Reference.Scheduler.Parameters[i]
			if got != want {
				t.Fatalf("parameter[%d] = %v, want %v", i, got, want)
			}
		}
	})

	p := DefaultParameters()

	t.Run("constants", func(t *testing.T) {
		closeEnough(t, "decay", p.Decay(), v.Cases.Constants.Decay)
		closeEnough(t, "factor", p.Factor(), v.Cases.Constants.Factor)
	})

	t.Run("initial stability", func(t *testing.T) {
		for _, tc := range v.Cases.InitialStability {
			closeEnough(t, ratingName(Rating(tc.Rating)), p.InitialStability(Rating(tc.Rating)), tc.Value)
		}
	})

	t.Run("initial difficulty clamped", func(t *testing.T) {
		for _, tc := range v.Cases.InitialDifficulty.Clamped {
			closeEnough(t, ratingName(Rating(tc.Rating)), p.InitialDifficulty(Rating(tc.Rating)), tc.Value)
		}
	})

	t.Run("initial difficulty unclamped", func(t *testing.T) {
		for _, tc := range v.Cases.InitialDifficulty.Unclamped {
			closeEnough(t, ratingName(Rating(tc.Rating)), p.initialDifficultyUnclamped(Rating(tc.Rating)), tc.Value)
		}
	})

	t.Run("next difficulty", func(t *testing.T) {
		for _, tc := range v.Cases.NextDifficulty {
			label := ratingName(Rating(tc.Rating)) + " from " + floatName(tc.Difficulty)
			closeEnough(t, label, p.NextDifficulty(tc.Difficulty, Rating(tc.Rating)), tc.Value)
		}
	})

	t.Run("next stability", func(t *testing.T) {
		for _, tc := range v.Cases.NextStability {
			label := ratingName(Rating(tc.Rating)) + " s=" + floatName(tc.Stability) +
				" d=" + floatName(tc.Difficulty) + " r=" + floatName(tc.Retrievability)
			got := p.NextStability(tc.Stability, tc.Difficulty, tc.Retrievability, Rating(tc.Rating))
			closeEnough(t, label, got, tc.Value)
		}
	})

	t.Run("retrievability", func(t *testing.T) {
		for _, tc := range v.Cases.Retrievability {
			label := "s=" + floatName(tc.Stability) + " t=" + floatName(float64(tc.ElapsedDays))
			closeEnough(t, label, p.Retrievability(float64(tc.ElapsedDays), tc.Stability), tc.Value)
		}
	})

	t.Run("interval", func(t *testing.T) {
		for _, tc := range v.Cases.Interval {
			got := p.Interval(tc.Stability)
			if got != tc.Value {
				t.Errorf("interval for stability %s = %d, want %d", floatName(tc.Stability), got, tc.Value)
			}
		}
	})

	t.Run("interval for retention", func(t *testing.T) {
		for _, tc := range v.Cases.IntervalForRetention {
			got := p.IntervalForRetention(tc.Stability, tc.Retention)
			if got != tc.Value {
				t.Errorf("interval for stability %s at retention %s = %d, want %d",
					floatName(tc.Stability), floatName(tc.Retention), got, tc.Value)
			}
		}
	})
}
