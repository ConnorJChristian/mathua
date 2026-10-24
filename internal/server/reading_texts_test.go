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
