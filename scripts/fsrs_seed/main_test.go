package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/storage"

	_ "github.com/mattn/go-sqlite3"
)

// holdOpen runs this test binary as the app: a child process that opens
// the store at path, answers one query, prints "ready" and keeps the
// connection until its stdin closes. (The check must run against another
// process, as it does in use: closing any descriptor of a file drops this
// process's own POSIX locks on it, SQLite's included.)
func holdOpen(t *testing.T, path string) func() {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperHoldOpen$")
	cmd.Env = append(os.Environ(), "FSRS_SEED_HOLD="+path)
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil || !strings.HasPrefix(line, "ready") {
		t.Fatalf("holder: %q %v", line, err)
	}
	return func() { stdin.Close(); cmd.Wait() }
}

func TestHelperHoldOpen(t *testing.T) {
	path := os.Getenv("FSRS_SEED_HOLD")
	if path == "" {
		t.Skip("helper process only")
	}
	store, err := storage.NewSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateStudent("live"); err != nil {
		t.Fatal(err)
	}
	fmt.Println("ready")
	io.Copy(io.Discard, os.Stdin)
	store.Close()
}

// FSRS Phase 2 review, M1: -apply refuses while the app (another process)
// has the database open, names the fix, and -force overrides. A dry run is
// never refused. Once the app closes the file, -apply goes ahead (here it
// stops at the missing MATHUA_ROOT, past the check).
func TestApplyRefusesAnOpenDatabase(t *testing.T) {
	t.Setenv("MATHUA_ROOT", "")
	path := filepath.Join(t.TempDir(), "mathua.db")
	store, err := storage.NewSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	store.Close()
	stop := holdOpen(t, path) // the running app
	if liveWriter(path) == "" {
		stop()
		t.Fatal("the check does not see the running app")
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
	stop()
	if why := liveWriter(path); why != "" {
		t.Fatalf("after the app stopped: %s", why)
	}
	if err := run(path, "", true, false, os.Stdout); err == nil || !strings.Contains(err.Error(), "MATHUA_ROOT") {
		t.Errorf("-apply on a closed database: %v; want it past the check", err)
	}
}

// The documented sequence is a dry run, then -apply, with the app stopped.
// A read-only reader (the dry run itself, fsrs_doctor) leaves -wal and -shm
// behind; that must not make -apply refuse (Linux: the check is the live
// connection's lock, not the files).
func TestApplyAfterAReadOnlyReader(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the lock check is Linux's; elsewhere the files decide")
	}
	t.Setenv("MATHUA_ROOT", "")
	path := filepath.Join(t.TempDir(), "mathua.db")
	store, err := storage.NewSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	store.Close()
	ro, err := sql.Open("sqlite3", "file:"+path+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ro.Exec("VACUUM INTO ?", filepath.Join(t.TempDir(), "copy.db")); err != nil {
		t.Fatal(err)
	}
	ro.Close()
	if _, err := os.Stat(path + "-shm"); err != nil {
		t.Skip("this SQLite removed -shm after a read-only reader; nothing to check")
	}
	if why := liveWriter(path); why != "" {
		t.Errorf("after a read-only reader closed: %s; want no live connection", why)
	}
}
