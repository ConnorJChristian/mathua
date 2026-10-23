//go:build !(linux || darwin || freebsd)

package pytrack

import (
	"os"
	"os/exec"
)

// On platforms without process groups or setuid (Windows outside Docker), the
// runner still has the wall-clock deadline and pyrunner.py's own audit hook,
// but no account drop. The supported way to run this track is the Docker image.
func sandbox(cmd *exec.Cmd) {}

func prepareWorkdir(dir string) error { return nil }

func killedByLimit(ps *os.ProcessState) bool { return false }

func killedByMemory(ps *os.ProcessState) bool { return false }
