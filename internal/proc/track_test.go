//go:build !windows

package proc

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

func TestKillAllEndsEveryTrackedGroupAtOnce(t *testing.T) {
	cmd := exec.CommandContext(context.Background(), "sh", "-c", "sleep 60 & wait")
	Tame(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	untrack := Track(cmd.Process.Pid)
	defer untrack()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	KillAll()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("a tracked command survived KillAll")
	}
	if Alive(cmd.Process.Pid) {
		t.Fatal("the process is still alive")
	}
}

func TestRunForgetsTheCommandWhenItEnds(t *testing.T) {
	cmd := exec.CommandContext(context.Background(), "sh", "-c", "exit 0")
	Tame(cmd)
	if err := Run(cmd); err != nil {
		t.Fatal(err)
	}
	if n := tracked(); n != 0 {
		t.Fatalf("%d commands still tracked after the run ended", n)
	}
}

func TestRunTracksTheCommandWhileItRuns(t *testing.T) {
	cmd := exec.CommandContext(context.Background(), "sh", "-c", "sleep 60")
	Tame(cmd)
	done := make(chan error, 1)
	go func() { done <- Run(cmd) }()
	deadline := time.Now().Add(2 * time.Second)
	for tracked() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("a running command was never tracked")
		}
		time.Sleep(10 * time.Millisecond)
	}
	KillAll()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return after KillAll")
	}
}

func TestAliveSeesARunningProcessAndNotAnEndedOne(t *testing.T) {
	cmd := exec.CommandContext(context.Background(), "sh", "-c", "sleep 60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if !Alive(cmd.Process.Pid) {
		t.Fatal("a running process reported dead")
	}
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
	if Alive(cmd.Process.Pid) {
		t.Fatal("an ended process reported alive")
	}
	if Alive(0) || Alive(-1) {
		t.Fatal("a non-pid reported alive")
	}
}
