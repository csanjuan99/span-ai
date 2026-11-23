// Package proc makes a started command safe to cancel. A cancelled context must end
// the whole tree the command started, not only the process Span launched: agent CLIs
// and test runners spawn children of their own, and a child left running keeps
// editing the working tree after the run says it stopped.
package proc

import (
	"os/exec"
	"sync"
	"time"
)

// Grace is how long a cancelled command gets to exit after the polite signal before
// everything it started is killed.
var Grace = 5 * time.Second

// waitDelay bounds Wait after a cancel. Without it, Wait reads the output pipes until
// every process that inherited them closes them, and an orphan can hold them open.
const waitDelay = 5 * time.Second

// The registry is for the second Ctrl-C. The first one cancels contexts and gives every
// command its Grace to end cleanly; the second must not leave agents behind that keep
// editing the tree after Span is gone. Commands that were started through Run, or tracked
// by hand, are what it can reach.
var (
	trackMu sync.Mutex
	live    = map[int]struct{}{}
)

// Track remembers a started command's pid until the returned func is called.
func Track(pid int) func() {
	if pid <= 0 {
		return func() {}
	}
	trackMu.Lock()
	live[pid] = struct{}{}
	trackMu.Unlock()
	return func() {
		trackMu.Lock()
		delete(live, pid)
		trackMu.Unlock()
	}
}

func tracked() int {
	trackMu.Lock()
	defer trackMu.Unlock()
	return len(live)
}

// KillAll ends every tracked command's whole tree now, without a grace period.
func KillAll() {
	trackMu.Lock()
	pids := make([]int, 0, len(live))
	for pid := range live {
		pids = append(pids, pid)
	}
	trackMu.Unlock()
	for _, pid := range pids {
		kill(pid)
	}
}

// Run is Start and Wait with the command tracked in between, so KillAll can reach it.
func Run(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	untrack := Track(cmd.Process.Pid)
	defer untrack()
	return cmd.Wait()
}
