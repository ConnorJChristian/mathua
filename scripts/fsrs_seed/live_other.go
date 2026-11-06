//go:build !linux

package main

import "os"

// liveWriter, off Linux, reports a -wal or -shm file beside the database:
// SQLite keeps them while a connection is open. A read-only reader or a
// crash can leave them too; with the app stopped, -force.
func liveWriter(dbPath string) string {
	for _, suffix := range []string{"-wal", "-shm"} {
		if _, err := os.Stat(dbPath + suffix); err == nil {
			return dbPath + suffix + " exists"
		}
	}
	return ""
}
