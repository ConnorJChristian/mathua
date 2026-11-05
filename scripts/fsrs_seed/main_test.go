package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/storage"
)

// FSRS Phase 2 review, M1: -apply refuses while the database is open (its
// -wal / -shm files exist), names the fix, and -force overrides. A dry run
// is never refused. Once the app closes the file, -apply goes ahead (here
// it stops at the missing MATHUA_ROOT, past the check).
func TestApplyRefusesAnOpenDatabase(t *testing.T) {
	t.Setenv("MATHUA_ROOT", "")
	path := filepath.Join(t.TempDir(), "mathua.db")
	store, err := storage.NewSQLiteStore(path) // the running app
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateStudent("live"); err != nil {
		t.Fatal(err)
	}
	if liveWriter(path) == "" {
		t.Fatal("an open store left no -wal / -shm file; the check cannot see it")
	}
	err = run(path, "", true, false, os.Stdout)
	if err == nil || !strings.Contains(err.Error(), "Stop the app") || !strings.Contains(err.Error(), "-force") {
		t.Errorf("-apply on an open database: %v; want a refusal naming the app and -force", err)
	}
	for name, args := range map[string][2]bool{"-apply -force": {true, true}, "dry run": {false, false}} {
		if err := run(path, "", args[0], args[1], os.Stdout); err == nil || !strings.Contains(err.Error(), "MATHUA_ROOT") {
			t.Errorf("%s: %v; want it past the check (stopping at MATHUA_ROOT)", name, err)
		}
	}
	store.Close()
	if f := liveWriter(path); f != "" {
		t.Fatalf("%s left after a clean close", f)
	}
	if err := run(path, "", true, false, os.Stdout); err == nil || !strings.Contains(err.Error(), "MATHUA_ROOT") {
		t.Errorf("-apply on a closed database: %v; want it past the check", err)
	}
}
