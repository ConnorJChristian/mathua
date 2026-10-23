//go:build linux || darwin || freebsd

package pytrack

import (
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

// sandboxUID is the account learner code runs as when the server is root,
// which it is in the Docker image. 65534 is "nobody" on Debian.
func sandboxUID() (uint32, bool) {
	if os.Geteuid() != 0 {
		return 0, false
	}
	uid := uint32(65534)
	if v, err := strconv.Atoi(os.Getenv("PYRUNNER_UID")); err == nil && v > 0 {
		uid = uint32(v)
	}
	return uid, true
}

// sandbox puts the runner in its own process group, so a timeout kills
// anything it started, and drops to the unprivileged account when root.
func sandbox(cmd *exec.Cmd) {
	attr := &syscall.SysProcAttr{Setpgid: true}
	if uid, ok := sandboxUID(); ok {
		attr.Credential = &syscall.Credential{Uid: uid, Gid: uid, Groups: []uint32{}}
		cmd.Env = append(cmd.Env, "PYRUNNER_SANDBOX_USER=1")
	}
	cmd.SysProcAttr = attr
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		// Negative pid: the whole process group.
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}

// prepareWorkdir hands the temporary folder to the sandbox account so the
// learner's code can write scratch files there and nowhere else.
func prepareWorkdir(dir string) error {
	if uid, ok := sandboxUID(); ok {
		return os.Chown(dir, int(uid), int(uid))
	}
	return nil
}

// killedByLimit reports a process stopped by the CPU rlimit or by our kill.
func killedByLimit(ps *os.ProcessState) bool {
	if ps == nil {
		return false
	}
	ws, ok := ps.Sys().(syscall.WaitStatus)
	if !ok || !ws.Signaled() {
		return false
	}
	sig := ws.Signal()
	return sig == syscall.SIGXCPU || sig == syscall.SIGKILL
}

func killedByMemory(ps *os.ProcessState) bool {
	if ps == nil {
		return false
	}
	ws, ok := ps.Sys().(syscall.WaitStatus)
	return ok && ws.Signaled() && ws.Signal() == syscall.SIGSEGV
}
