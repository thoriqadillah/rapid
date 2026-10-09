package downloader

import (
	"os/exec"
	"sync"
	"time"
)

// daemon is a spawned aria2c process. One Wait goroutine owns the process and
// closes exited when it ends; alive() and stop() read that channel instead of
// probing the process (the old isAlive looked up zombies and could never report
// a crashed daemon as dead, so ensureDaemon never respawned).
type daemon struct {
	cmd    *exec.Cmd
	exited chan struct{}

	mu      sync.Mutex
	waitErr error
}

// startDaemon launches aria2c detached from our process group (so a terminal
// hangup does not take the daemon down with the app) and returns a handle whose
// Wait goroutine tracks the process lifetime.
func startDaemon(program string, args []string) (*daemon, error) {
	cmd := exec.Command(program, args...)
	cmd.SysProcAttr = sysProcAttr()

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	d := &daemon{cmd: cmd, exited: make(chan struct{})}
	go func() {
		err := cmd.Wait()
		d.mu.Lock()
		d.waitErr = err
		d.mu.Unlock()
		close(d.exited)
	}()
	return d, nil
}

// alive reports whether the process is still running. A nil handle is dead.
func (d *daemon) alive() bool {
	if d == nil {
		return false
	}
	select {
	case <-d.exited:
		return false
	default:
		return true
	}
}

// stop asks the daemon to exit, waits up to grace for it to save its session
// and finish, then kills it. A nil or already-dead handle is a no-op.
func (d *daemon) stop(grace time.Duration) {
	if d == nil || !d.alive() {
		return
	}

	terminate(d.cmd)
	select {
	case <-d.exited:
		return
	case <-time.After(grace):
	}

	d.cmd.Process.Kill()
	<-d.exited
}
