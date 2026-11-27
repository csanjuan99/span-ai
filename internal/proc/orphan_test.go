//go:build !windows

package proc

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// start launches a long command the way an agent is launched and returns once it runs. The
// caller never waits for it except through the returned channel, like a dead owner.
func start(t *testing.T) (*exec.Cmd, chan error) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "sh", "-c", "sleep 60 & wait")
	Tame(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	t.Cleanup(func() { kill(cmd.Process.Pid) })
	return cmd, done
}

func TestReapEndsAnAgentItsOwnerLeftBehind(t *testing.T) {
	dir := t.TempDir()
	cmd, done := start(t)
	Leave(dir, cmd.Process.Pid, "sleep") // the owner died, so the returned func never ran

	if len(Reap(dir)) == 0 {
		t.Fatal("a live orphan was not reaped")
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("the orphan survived Reap")
	}
	if _, err := os.Stat(filepath.Join(dir, pidFile)); !os.IsNotExist(err) {
		t.Fatal("the pid file was left behind")
	}
}

func TestReapNeverKillsAProcessThatIsNotTheRecordedAgent(t *testing.T) {
	dir := t.TempDir()
	cmd, _ := start(t)
	// The pid was reused by something else: the recorded name is not in its command line.
	Leave(dir, cmd.Process.Pid, "claude")

	if len(Reap(dir)) != 0 {
		t.Fatal("reaped a process that is not the agent")
	}
	if !Alive(cmd.Process.Pid) {
		t.Fatal("an unrelated process was killed")
	}
}

func TestReapOfAFinishedAgentOnlyClearsTheFile(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("true")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	Leave(dir, cmd.Process.Pid, "true")
	if len(Reap(dir)) != 0 {
		t.Fatal("nothing was running")
	}
	if _, err := os.Stat(filepath.Join(dir, pidFile)); !os.IsNotExist(err) {
		t.Fatal("a stale pid file was kept")
	}
}

func TestAnAgentThatEndsNormallyLeavesNoPidFile(t *testing.T) {
	dir := t.TempDir()
	forget := Leave(dir, os.Getpid(), "x")
	forget()
	if _, err := os.Stat(filepath.Join(dir, pidFile)); !os.IsNotExist(err) {
		t.Fatal("the pid file outlived its agent")
	}
}

func TestReapWithoutAFileDoesNothing(t *testing.T) {
	if len(Reap(t.TempDir())) != 0 {
		t.Fatal("reaped from nothing")
	}
}
