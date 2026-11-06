//go:build linux

package main

import (
	"errors"
	"os"
	"syscall"
)

// fOFDSetLK is Linux's F_OFD_SETLK: an open-file-description lock, which
// conflicts with SQLite's POSIX locks even inside one process.
const fOFDSetLK = 37

// shmDMSByte is the -shm byte every open SQLite connection holds a shared
// lock on (unix VFS: UNIX_SHM_BASE 120 + SQLITE_SHM_NLOCK 8).
const shmDMSByte = 128

// liveWriter reports why the database looks open by another connection,
// "" when it does not. A WAL database's connection holds a shared lock on
// the -shm file's DMS byte for as long as it is open; if a write lock on
// that byte can be taken, nothing has the file open. A -wal or -shm file
// alone means nothing: a read-only reader (the dry run, fsrs_doctor, the
// shadow report) leaves both behind, and so does a crash.
func liveWriter(dbPath string) string {
	f, err := os.OpenFile(dbPath+"-shm", os.O_RDWR, 0)
	if errors.Is(err, os.ErrNotExist) {
		return ""
	}
	if err != nil {
		return dbPath + "-shm cannot be opened to check for a live connection (" + err.Error() + ")"
	}
	defer f.Close()
	lk := syscall.Flock_t{Type: syscall.F_WRLCK, Whence: 0, Start: shmDMSByte, Len: 1}
	if err := syscall.FcntlFlock(f.Fd(), fOFDSetLK, &lk); err != nil {
		if errors.Is(err, syscall.EAGAIN) || errors.Is(err, syscall.EACCES) {
			return "another connection holds " + dbPath + "-shm"
		}
		return "checking " + dbPath + "-shm for a live connection: " + err.Error()
	}
	lk.Type = syscall.F_UNLCK
	syscall.FcntlFlock(f.Fd(), fOFDSetLK, &lk)
	return ""
}
