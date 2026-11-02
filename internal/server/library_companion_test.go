package server

import (
	"testing"

	"github.com/chuma-beep/mathua/internal/modules"
)

// A companion book (an online supplement) is listed inside its main book:
// each of its chapters after the main book's chapter it supplements, with
// its sections, and every row opens the companion's own PDF.
func TestInterleaveCompanions(t *testing.T) {
	main := libraryBook{ID: slug("m/main.pdf"), Kind: "pdf", Title: "Main", URL: "/api/modules/m/files/main.pdf", Available: true,
		Sections: []librarySection{
			{Title: "1.1 One", Page: 10, EndPage: 19, Topics: []libraryTopic{{"a", "A"}}},
			{Title: "1.2 One more", Level: 1, Page: 15, EndPage: 16},
			{Title: "2.1 Two", Page: 20, EndPage: 29},
			{Title: "3.1 Three", Page: 30, EndPage: 39},
		}}
	supp := libraryBook{ID: slug("m/supp.pdf"), Kind: "pdf", Title: "Supplement", URL: "/api/modules/m/files/supp.pdf", Available: true,
		Sections: []librarySection{
			{Title: "Chapter S1", Page: 1, EndPage: 4},
			{Title: "S1.1 Extra", Level: 1, Page: 1, EndPage: 2, Topics: []libraryTopic{{"s11", "S1.1 Extra"}}},
			{Title: "S1.2 More", Level: 1, Page: 3, EndPage: 4, Topics: []libraryTopic{{"s12", "S1.2 More"}}},
			{Title: "Chapter S3", Page: 5, EndPage: 8},
			{Title: "S3.1 Late", Level: 1, Page: 6, EndPage: 8},
		}}
	other := libraryBook{ID: slug("m/other.pdf"), Kind: "pdf", Title: "Other", Sections: []librarySection{{Title: "x", Page: 1}}}
	books := []modules.BookMeta{
		{Module: "m", File: "main.pdf", Title: "Main"},
		{Module: "m", File: "supp.pdf", Title: "Supplement", CompanionOf: "main.pdf", TOC: []modules.TOCEntry{
			{Title: "Chapter S1", Page: 1, EndPage: 4, AfterPage: 19},
			{Title: "Chapter S3", Page: 5, EndPage: 8, AfterPage: 39},
		}},
	}
	out := interleaveCompanions([]libraryBook{main, supp, other}, books)
	var got []string
	for _, s := range out[0].Sections {
		got = append(got, s.Title)
	}
	want := []string{"1.1 One", "1.2 One more", "Chapter S1", "S1.1 Extra", "S1.2 More", "2.1 Two", "3.1 Three", "Chapter S3", "S3.1 Late"}
	if len(got) != len(want) {
		t.Fatalf("main book rows = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("main book rows = %v, want %v", got, want)
		}
	}
	for _, s := range out[0].Sections {
		isSupp := s.Title[0] == 'C' || s.Title[0] == 'S'
		if isSupp && (s.PDF != supp.URL || s.Book != "Supplement" || s.PDFMissing) {
			t.Errorf("companion row %q opens %q (book %q, missing %v), want the supplement's PDF", s.Title, s.PDF, s.Book, s.PDFMissing)
		}
		if !isSupp && s.PDF != "" {
			t.Errorf("main book row %q has pdf %q, want none (it opens the book's own)", s.Title, s.PDF)
		}
	}
	if n := len(out[0].Sections[3].Topics); n != 1 || out[0].Sections[3].Topics[0].ID != "s11" {
		t.Errorf("companion section keeps its topics: got %+v", out[0].Sections[3].Topics)
	}
	// The companion keeps its own entry, unchanged; other books are untouched.
	if len(out[1].Sections) != 5 || out[1].Sections[1].PDF != "" {
		t.Errorf("the companion's own page changed: %+v", out[1].Sections)
	}
	if len(out[2].Sections) != 1 {
		t.Errorf("an unrelated book changed: %+v", out[2].Sections)
	}
}

// A companion whose PDF is not on this computer is still listed, marked so
// the page does not open the main book's PDF at the companion's page; one
// whose main book is not shelved, or with no after_page, changes nothing.
func TestInterleaveCompanionsEdgeCases(t *testing.T) {
	main := libraryBook{ID: slug("m/main.pdf"), Kind: "pdf", Title: "Main", Sections: []librarySection{{Title: "1 One", Page: 10}}}
	supp := libraryBook{ID: slug("m/supp.pdf"), Kind: "pdf", Title: "Supp", URL: "/s.pdf", Available: false,
		Sections: []librarySection{{Title: "Chapter S1", Page: 1, EndPage: 3}}}
	meta := func(after int) []modules.BookMeta {
		return []modules.BookMeta{{Module: "m", File: "supp.pdf", CompanionOf: "main.pdf",
			TOC: []modules.TOCEntry{{Title: "Chapter S1", Page: 1, EndPage: 3, AfterPage: after}}}}
	}
	out := interleaveCompanions([]libraryBook{main, supp}, meta(5))
	if len(out[0].Sections) != 2 || out[0].Sections[0].Title != "Chapter S1" || !out[0].Sections[0].PDFMissing {
		t.Errorf("after_page before the first row puts the companion first, marked missing: %+v", out[0].Sections)
	}
	main.Sections = []librarySection{{Title: "1 One", Page: 10}}
	out = interleaveCompanions([]libraryBook{main, supp}, meta(0))
	if len(out[0].Sections) != 1 {
		t.Errorf("rows with no after_page are not interleaved: %+v", out[0].Sections)
	}
	out = interleaveCompanions([]libraryBook{supp}, meta(5))
	if len(out) != 1 || len(out[0].Sections) != 1 {
		t.Errorf("no main book on the shelf: nothing changes, got %+v", out)
	}
}
