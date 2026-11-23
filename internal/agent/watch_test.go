package agent

import (
	"context"
	"os"
	"os/exec"
	"testing"
)

func TestObserveTracksTheSpanProcess(t *testing.T) {
	ctx := WithSpan(context.Background(), "span-1")
	stop := observe(ctx, "claude", os.Getpid())
	proc, ok := Current("span-1")
	if !ok || proc.Name != "claude" || proc.PID != os.Getpid() || !Alive(proc.PID) {
		t.Fatalf("%+v %v", proc, ok)
	}
	stop()
	if _, ok := Current("span-1"); ok {
		t.Fatal("process still recorded")
	}
}

func TestAliveRejectsAnExitedProcess(t *testing.T) {
	cmd := exec.Command("true")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	if Alive(pid) {
		t.Fatal("exited process still counts as alive")
	}
	if !Alive(os.Getpid()) {
		t.Fatal("this process counts as dead")
	}
}
