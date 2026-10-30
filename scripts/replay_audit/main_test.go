package main

import (
	"os"
	"path/filepath"
	"testing"
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
