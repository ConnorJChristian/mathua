package pytrack

import (
	"os"
	"path/filepath"
	"testing"
)

// Each item's book: an explicit source_book, a book_ref's file, the
// concept's book for an authored bank, the module's sources.json for an
// imported deck (whose concept's book is the deck's course, not a book).
func TestSourceOf(t *testing.T) {
	root := t.TempDir()
	write := func(p, s string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(root, "ma", "bank", "01.json"), `{"chapter":"1","title":"t","concepts":[
	 {"id":"ma.hoy.5_4","book":{"name":"Hoy et al., Mathematics for Economics (3e)","section":"5.4","title":"Rules","url":""},"intro":"","items":[
	  {"id":"ex","kind":"recall","level":2,"prompt":"p","back":"b","explain":"","book_ref":{"file":"hoy-3e.pdf","module":"ma","page":3,"label":"Hoy 5.4 #1"}},
	  {"id":"own","kind":"recall","level":2,"prompt":"p","back":"b","explain":""},
	  {"id":"schaum","kind":"recall","level":2,"prompt":"p","back":"b","explain":"","source_book":"Schaums_Calc_Econ"}]},
	 {"id":"ma.notes","book":{"name":"","section":"","title":"Notes","url":""},"intro":"","items":[
	  {"id":"n","kind":"recall","level":2,"prompt":"p","back":"b","explain":""}]}]}`)
	write(filepath.Join(root, "anki", "bank", "anki_calc.json"), `{"chapter":"c","title":"t","concepts":[
	 {"id":"anki.deriv","book":{"name":"Mathematical Economics I (Hoy et al., Schaum's, Axler)","section":"","title":"Derivatives","url":""},"intro":"","items":[
	  {"id":"c1","kind":"recall","level":2,"prompt":"p","back":"b","explain":""},
	  {"id":"c2","kind":"recall","level":2,"prompt":"p","back":"b","explain":""},
	  {"id":"c3","kind":"recall","level":2,"prompt":"p","back":"b","explain":""}]}]}`)
	write(filepath.Join(root, "anki", "sources.json"), `{"c1":"schaums-calc-econ","c2":"hoy"}`)
	b, err := LoadBank(filepath.Join(root, "ma", "bank"), filepath.Join(root, "anki", "bank"))
	if err != nil {
		t.Fatal(err)
	}
	src := func(c, i string) string { return b.SourceOf(c, b.items[itemKey(c, i)]) }
	for _, tc := range []struct{ c, i, want string }{
		{"ma.hoy.5_4", "ex", "hoy"},
		{"ma.hoy.5_4", "own", "hoy"},
		{"ma.hoy.5_4", "schaum", "schaums-calc-econ"},
		{"ma.notes", "n", ""},
		{"anki.deriv", "c1", "schaums-calc-econ"},
		{"anki.deriv", "c2", "hoy"},
		{"anki.deriv", "c3", ""},
	} {
		if got := src(tc.c, tc.i); got != tc.want {
			t.Errorf("SourceOf(%s#%s) = %q, want %q", tc.c, tc.i, got, tc.want)
		}
	}
}
