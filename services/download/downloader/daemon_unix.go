//go:build !windows

package downloader

import (
	"os/exec"
	"syscall"
)

// sysProcAttr detaches the daemon into its own session/process group so it
// survives the app's terminal hangup and is not killed by our ctx.
func sysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}

// terminate sends SIGTERM so aria2 shuts down gracefully and saves its session
// (the old CommandContext sent SIGKILL on ctx cancel, losing the session).
func terminate(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Signal(syscall.SIGTERM)
}
