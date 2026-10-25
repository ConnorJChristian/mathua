package all

// Regression tests for the 2026-10-25 QA run (run 5) on the python-dsh bank
// (modules/python-dsh/src builds modules/python-dsh/bank).

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type run5BankItem struct {
	ID      string   `json:"id"`
	Kind    string   `json:"kind"`
	Prompt  string   `json:"prompt"`
	Correct string   `json:"correct"`
	Wrong   []string `json:"wrong"`
	Starter string   `json:"starter"`
}

func run5Bank(t *testing.T) map[string]map[string]run5BankItem {
	t.Helper()
	files, err := filepath.Glob("../../../modules/python-dsh/bank/*.json")
	if err != nil || len(files) == 0 {
		t.Skip("python-dsh bank not present")
	}
	out := map[string]map[string]run5BankItem{}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var doc struct {
			Concepts []struct {
				ID    string         `json:"id"`
				Items []run5BankItem `json:"items"`
			} `json:"concepts"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for _, c := range doc.Concepts {
			out[c.ID] = map[string]run5BankItem{}
			for _, it := range c.Items {
				out[c.ID][it.ID] = it
			}
		}
	}
	return out
}

// The eleven multiple-choice items the review found keyed by the longest
// option by more than 40% (fails on the old bank).
func TestRun5BankKeysAreNotTheLongestByFar(t *testing.T) {
	bank := run5Bank(t)
	for _, ref := range [][2]string{
		{"py.np.partition", "m1"}, {"py.np.partition", "m2"}, {"py.np.partition", "m4"},
		{"py.np.structured", "m2"}, {"py.np.structured", "m5"},
		{"py.mpl.annotation", "m5"}, {"py.mpl.annotation", "m6"},
		{"py.mpl.three_d", "m2"}, {"py.mpl.three_d", "m5"}, {"py.mpl.three_d", "m6"},
	} {
		it, ok := bank[ref[0]][ref[1]]
		if !ok {
			t.Fatalf("%s %s missing", ref[0], ref[1])
		}
		longest := 0
		for _, w := range it.Wrong {
			if n := len([]rune(w)); n > longest {
				longest = n
			}
		}
		if n := len([]rune(it.Correct)); float64(n) > 1.4*float64(longest) {
			t.Errorf("%s %s: key %d characters, longest distractor %d", ref[0], ref[1], n, longest)
		}
	}
}

// py.np.partition m5 offered `K` and `K + 1` for the partition position, and
// both work: kth=K already places the K + 1 smallest first (fails on the old
// bank). The rewritten item asks which columns hold the point and its K
// nearest neighbours, where only `idx[:, :K + 1]` is right.
func TestRun5PartitionHasOneRightOption(t *testing.T) {
	it := run5Bank(t)["py.np.partition"]["m5"]
	if it.Correct != "`idx[:, :K + 1]`" {
		t.Fatalf("key %q", it.Correct)
	}
	for _, w := range it.Wrong {
		if w == "`K`" || w == "`K + 1`" {
			t.Fatalf("ambiguous partition-position option %q", w)
		}
	}
}

// The 3D surface task's starter built a 2D axes (fails on the old bank).
func TestRun5ThreeDStarterIs3D(t *testing.T) {
	it := run5Bank(t)["py.mpl.three_d"]["c1"]
	if !strings.Contains(it.Starter, `projection="3d"`) {
		t.Fatalf("starter: %q", it.Starter)
	}
}
