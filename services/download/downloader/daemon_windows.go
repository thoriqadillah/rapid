//go:build windows

package downloader

import (
	"os/exec"
	"syscall"
)

// sysProcAttr puts the daemon in a new process group so it is not killed with
// the app's console.
func sysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

// terminate kills the daemon: there is no portable graceful signal here.
func terminate(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
}
