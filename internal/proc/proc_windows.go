//go:build windows

package proc

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"syscall"
)

// Tame ends the command on cancel. Windows has no process groups to signal, so only the
// process Span started is killed; WaitDelay still keeps Wait from hanging on its pipes.
func Tame(cmd *exec.Cmd) {
	cmd.Cancel = func() error { return cmd.Process.Kill() }
	cmd.WaitDelay = waitDelay
}

// kill ends the process; Windows has no group to signal.
func kill(pid int) {
	p, err := os.FindProcess(pid)
	if err != nil {
		log.Printf("find process %d: %v", pid, err)
		return
	}
	logSignal(pid, p.Kill())
}

// Alive reports whether pid is a running process: a handle that opens and whose exit code
// is still STILL_ACTIVE.
func Alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	const processQueryLimitedInformation = 0x1000
	const stillActive = 259
	handle, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return false
	}
	defer func() {
		if err := syscall.CloseHandle(handle); err != nil {
			log.Printf("close the handle of process %d: %v", pid, err)
		}
	}()
	var code uint32
	if err := syscall.GetExitCodeProcess(handle, &code); err != nil {
		return false
	}
	return code == stillActive
}

func processGone(err error) bool { return errors.Is(err, os.ErrProcessDone) }
