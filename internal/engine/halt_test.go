//go:build !windows

package engine

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"span/internal/journal"
	"span/internal/proc"
)

func TestRewindStopsARunInFlightBeforeRewritingIt(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	waiting := &waitingBuild{fakeAgents: agents, started: make(chan struct{})}
	opt.Agents = waiting
	id, err := Open(opt)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		// The stopped flight ends with the cancel as its error; that is what a stop looks like.
		if _, err := Resume(context.Background(), opt, id); err != nil {
			t.Logf("the stopped flight ended: %v", err)
		}
	}()
	<-waiting.started

	if err := Rewind(opt.Root, id, frame); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the flight survived the rewind")
	}
	snap, err := Load(opt.Root, id)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Cursor.Stage != frame || snap.Cursor.Hold {
		t.Fatalf("%+v", snap.Cursor)
	}
}

func TestReapRunsEndsTheAgentOfARunNobodyDrivesAndJournalsIt(t *testing.T) {
	old := proc.Grace
	proc.Grace = 500 * time.Millisecond
	t.Cleanup(func() { proc.Grace = old })
	opt, _, _, _, _, _ := baseOpt(t)
	id, err := Open(opt)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(opt.Root, id)
	ended := startOrphan(t, dir)

	ReapRuns(opt.Root)

	select {
	case <-ended:
	case <-time.After(5 * time.Second):
		t.Fatal("the orphan survived the reap")
	}
	events, err := journal.Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.Type == "agent.terminated" && event.Detail["pid"] != "" && event.Detail["signal"] != "" && event.Detail["exit"] != "" {
			return
		}
	}
	t.Fatalf("no termination was journaled: %+v", events)
}

func TestARefusedRewindDoesNotStopTheRunInFlight(t *testing.T) {
	opt, _, _, _, _, _ := baseOpt(t)
	id, err := Open(opt)
	if err != nil {
		t.Fatal(err)
	}
	if err := Rewind(opt.Root, id, intake); err == nil {
		t.Fatal("intake cannot be restarted")
	}
	if _, err := os.Stat(filepath.Join(opt.Root, id, stopFile)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a refused rewind asked the run to stop: %v", err)
	}
}

// startOrphan starts a process that ignores SIGTERM and records it in dir as the agent of a
// driver that died: the func that clears the record never ran. The channel closes when the
// process ends.
func startOrphan(t *testing.T, dir string) <-chan struct{} {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "sh", "-c", "trap '' TERM; while :; do sleep 1; done")
	proc.Tame(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	ended := make(chan struct{})
	go func() {
		// The reap kills it, so the wait reports that signal.
		if err := cmd.Wait(); err != nil {
			t.Logf("the orphan ended: %v", err)
		}
		close(ended)
	}()
	t.Cleanup(func() {
		if err := cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			t.Logf("kill the orphan: %v", err)
		}
	})
	proc.Leave(dir, cmd.Process.Pid, "sh")
	return ended
}
