package pytrack

import (
	"strings"
	"testing"
)

// levelBank: a book exercise at level 2 and a practice version unlocking
// after it.
const levelBank = `{"chapter": "t", "title": "T", "concepts": [{"id": "t.l", "book": {"title": "L"}, "intro": "", "items": [
 {"id": "ex", "kind": "math", "level": 2, "prompt": "Enter the value.", "parts": [{"answer": "1", "grading": "numeric", "tex": "1"}],
  "book_ref": {"file": "b.pdf", "module": "mod", "page": 3, "label": "Book Exercise 1, #1"}, "explain": "."},
 {"id": "pr", "kind": "math", "level": 2, "prompt": "What is 1?", "parts": [{"answer": "1", "grading": "numeric", "tex": "1"}], "unlocks_after": "ex", "explain": "."}]}]}`

// An item that unlocks after another is never an easier step than it.
func TestUnlocksAfterNeverBelowItsItem(t *testing.T) {
	bad := strings.Replace(levelBank, `"id": "pr", "kind": "math", "level": 2`, `"id": "pr", "kind": "math", "level": 1`, 1)
	if bad == levelBank {
		t.Fatal("fixture did not change")
	}
	if _, err := LoadBank(writeModule(t, bad, false)); err == nil || !strings.Contains(err.Error(), "below the level") {
		t.Fatalf("want a level error, got %v", err)
	}
	if _, err := LoadBank(writeModule(t, levelBank, false)); err != nil {
		t.Fatal(err)
	}
}
