package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chuma-beep/mathua/internal/generator"
)

// The replay never writes its -db: it works on a fresh copy, and refuses a
// snapshot with a non-empty write-ahead log beside it, whose frames a plain
// copy would leave behind (a stale log replayed onto a fresh copy is how an
// early run corrupted its database).
func TestFreshCopy(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "snap.db")
	if err := os.WriteFile(src, []byte("sqlite bytes"), 0o444); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src+"-wal", nil, 0o444); err != nil {
		t.Fatal(err)
	}
	dst, err := freshCopy(src)
	if err != nil {
		t.Fatalf("an empty -wal is harmless: %v", err)
	}
	defer os.RemoveAll(filepath.Dir(dst))
	if b, _ := os.ReadFile(dst); string(b) != "sqlite bytes" || filepath.Dir(dst) == dir {
		t.Errorf("copy %s = %q", dst, b)
	}
	if err := os.WriteFile(src+"-wal", []byte("frames"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := freshCopy(src); err == nil {
		t.Error("a snapshot with a non-empty -wal must be refused")
	}
}

// The served-order check: main book, parallel books, practice set within a
// step; a new step starts over; a retry ranks nowhere.
func TestServedOrderError(t *testing.T) {
	p := func(step int, tier string) generator.Problem { return generator.Problem{Step: step, Tier: tier} }
	ok := []generator.Problem{p(1, "main"), p(1, "parallel"), p(1, "retry"), p(1, "parallel"), p(1, "practice"), p(2, "main"), p(2, "practice")}
	if msg := servedOrderError(ok); msg != "" {
		t.Errorf("in order, got %q", msg)
	}
	for _, bad := range [][]generator.Problem{
		{p(1, "parallel"), p(1, "main")},
		{p(1, "main"), p(1, "practice"), p(1, "parallel")},
	} {
		if servedOrderError(bad) == "" {
			t.Errorf("%v: out of order not caught", bad)
		}
	}
}
