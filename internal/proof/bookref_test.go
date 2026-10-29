package proof

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
)

// A proof problem may point at a book exercise and unlock after another
// problem of its topic; both fields survive a load, and a missing PDF still
// names the exercise and its pages.
func TestBookRefProofs(t *testing.T) {
	dir := t.TempDir()
	book := testProblem()
	book.ID = "t.book"
	book.Statement = "Prove the claim in the exercise shown above."
	book.BookRef = &generator.BookRef{File: "hoy.pdf", Module: "mp", Page: 60, PrintedPage: 46, Label: "Hoy Exercise 2.4, #7"}
	practice := testProblem()
	practice.ID = "t.practice"
	practice.UnlocksAfter = "t.book"
	raw, _ := json.Marshal(File{Chapter: "c", Title: "t", Problems: []Problem{book, practice}})
	os.WriteFile(filepath.Join(dir, "a.json"), raw, 0o644)
	b, err := LoadBank(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := b.Problem("t.book").BookRef; !reflect.DeepEqual(got, book.BookRef) {
		t.Fatalf("book_ref %+v", got)
	}
	if b.Problem("t.practice").UnlocksAfter != "t.book" {
		t.Fatal("unlocks_after lost")
	}
	if !strings.Contains(string(raw), `"book_ref":{"file":"hoy.pdf"`) || !strings.Contains(string(raw), `"unlocks_after":"t.book"`) {
		t.Fatalf("field names: %s", raw)
	}

	sp, _ := b.ProblemFor("t.book")
	if !strings.HasPrefix(sp.Question, generator.BookMarker) || !strings.Contains(sp.Question, `"available":false`) ||
		!strings.Contains(sp.Question, "Hoy Exercise 2.4, #7") || !strings.Contains(sp.Question, "book page 46, PDF page 60") ||
		!strings.HasSuffix(sp.Question, Marker) {
		t.Fatalf("served: %q", sp.Question)
	}
	b.FileExists = func(module, file string) bool { return module == "mp" && file == "hoy.pdf" }
	sp, _ = b.ProblemFor("t.book")
	if !strings.Contains(sp.Question, `"url":"/api/modules/mp/files/hoy.pdf"`) {
		t.Fatalf("served with the PDF: %q", sp.Question)
	}

	practice.UnlocksAfter = "t.nowhere"
	raw, _ = json.Marshal(File{Chapter: "c", Title: "t", Problems: []Problem{book, practice}})
	os.WriteFile(filepath.Join(dir, "a.json"), raw, 0o644)
	if _, err := LoadBank(dir); err == nil {
		t.Fatal("unlocks_after naming no problem accepted")
	}
	book.BookRef.Label = ""
	if _, err := NewBank(book); err == nil {
		t.Fatal("book_ref without a label accepted")
	}
}
