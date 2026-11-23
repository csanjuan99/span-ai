//go:build windows

package proc

import "os/exec"

// Tame ends the command on cancel. Windows has no process groups to signal, so only the
// process Span started is killed; WaitDelay still keeps Wait from hanging on its pipes.
func Tame(cmd *exec.Cmd) {
	cmd.Cancel = func() error { return cmd.Process.Kill() }
	cmd.WaitDelay = waitDelay
}
