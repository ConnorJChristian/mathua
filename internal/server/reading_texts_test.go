package server

import "testing"

func TestTextMatches(t *testing.T) {
	web := readingOption{Key: "web:An Introduction to Statistical Learning with Python", Kind: "web", Book: "An Introduction to Statistical Learning with Python"}
	pdf := readingOption{Key: "ma-program/hoy-3e.pdf", Kind: "pdf", Book: "Mathematics for Economics"}
	if !textMatches("web:statistical learning", web) {
		t.Error("a web key should match a book whose name contains it, ignoring case")
	}
	if textMatches("web:Wooldridge", web) {
		t.Error("a web key must not match another book")
	}
	if !textMatches("ma-program/hoy-3e.pdf", pdf) || textMatches("ma-program/other.pdf", pdf) {
		t.Error("a file key matches its own file only")
	}
	if textMatches("web:Mathematics", pdf) {
		t.Error("a web key never matches a file")
	}
}

// A bank cites its book in its own style, and the module's books.json names
// the same book differently, so the lesson's text picker showed it twice: the
// PDF, and a second button that opened nothing and only repeated the section.
func TestBankSourceDoesNotRepeatAModuleBook(t *testing.T) {
	cases := []struct {
		module, bank, url string
		kept              bool
	}{
		{"Mathematics for Economics (3rd ed.)", "Hoy et al., Mathematics for Economics (3e)", "", false},
		{"Academic Writing for Graduate Students: Essential Tasks and Skills (3rd ed.)", "Swales & Feak, Academic Writing for Graduate Students (3rd ed.)", "", false},
		// An online copy is a second way in, not a repeat.
		{"Los conceptos elementales del materialismo histórico", "Harnecker, Los conceptos elementales del materialismo histórico (1985)", "https://example.org/harnecker.pdf", true},
		// Another book is never dropped, linked or not.
		{"Mathematics for Economics (3rd ed.)", "Wooldridge, Introductory Econometrics 7e", "", true},
		{"Schaum's Outline of Mathematical Methods for Business, Economics and Finance (2nd ed.)", "Hoy et al., Mathematics for Economics (3e)", "", true},
	}
	for _, c := range cases {
		out := []readingOption{{Key: "m/book.pdf", Kind: "pdf", Book: c.module, URL: "/api/modules/m/files/book.pdf"}}
		got := withBankOption(out, readingOption{Key: "web:" + c.bank, Kind: "web", Book: c.bank, URL: c.url})
		if kept := len(got) == 2; kept != c.kept {
			t.Errorf("module %q, bank %q (url %q): kept=%v, want %v", c.module, c.bank, c.url, kept, c.kept)
		}
	}
	// With no module reading, the bank's citation is the only text and stays.
	if got := withBankOption(nil, readingOption{Kind: "web", Book: "Hoy et al., Mathematics for Economics (3e)"}); len(got) != 1 {
		t.Error("a bank source with nothing else to read must stay")
	}
}
