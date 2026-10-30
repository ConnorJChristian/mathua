package generator

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// A "(yes/no: <answer>)" prompt prints its own answer and asks a question
// that yes or no cannot answer ("What is uniform continuity? (yes/no:
// delta depends only on epsilon)" keyed yes). Q31 fixed one; Q56 found
// eight more. This scans every generator source so the pattern cannot come
// back in a package that has no sweep of its own.
func TestNoPromptStatesItsAnswerAfterYesNo(t *testing.T) {
	re := regexp.MustCompile(`\(yes/no: [^)]`)
	var hits []string
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			if re.MatchString(line) {
				hits = append(hits, path+":"+strconv.Itoa(i+1))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) > 0 {
		t.Fatalf("%d prompts state their answer after \"(yes/no:\": %s", len(hits), strings.Join(hits, ", "))
	}
}
