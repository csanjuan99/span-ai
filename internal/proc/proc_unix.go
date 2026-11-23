//go:build !windows

package proc

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// Tame puts the command in its own process group and makes cancel end the group:
// SIGTERM first, SIGKILL after Grace. Call it before Start.
func Tame(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	cmd.Cancel = func() error { return terminate(cmd.Process.Pid) }
	cmd.WaitDelay = waitDelay
}

// terminate signals the group led by pid and returns once the group is gone or has
// been killed. Wait does not return until this does, so it must not outlive Grace.
func terminate(pid int) error {
	if err := syscall.Kill(-pid, syscall.SIGTERM); err != nil {
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	// A process stopped by SIGTTIN/SIGSTOP (a tool that opens /dev/tty from a background
	// group) only acts on SIGTERM once it is continued.
	_ = syscall.Kill(-pid, syscall.SIGCONT)
	deadline := time.Now().Add(Grace)
	for time.Now().Before(deadline) {
		time.Sleep(25 * time.Millisecond)
		if errors.Is(syscall.Kill(-pid, 0), syscall.ESRCH) {
			return nil
		}
	}
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	return nil
}
