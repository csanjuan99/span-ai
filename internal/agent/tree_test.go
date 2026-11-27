//go:build !windows

package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"span/internal/proc"
)

// fakeAgent writes an executable named like a real CLI that runs body under sh.
func fakeAgent(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cursor-agent")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func shortGrace(t *testing.T) {
	t.Helper()
	old := proc.Grace
	proc.Grace = 500 * time.Millisecond
	t.Cleanup(func() { proc.Grace = old })
}

// readPID waits for the fake agent to write the pid of its child.
func readPID(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if raw, err := os.ReadFile(path); err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil {
				t.Cleanup(func() {
					if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
						t.Logf("kill %d: %v", pid, err)
					}
				})
				return pid
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("the fake agent never wrote its child pid")
	return 0
}

func dead(pid int) bool {
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if errors.Is(syscall.Kill(pid, 0), syscall.ESRCH) {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

type recorded struct {
	ended []proc.Termination
}

func supervised(ctx context.Context, dir string, got *recorded) context.Context {
	return WithSupervisor(ctx, Supervisor{RunDir: dir, Stage: "build", Ended: func(end proc.Termination) {
		got.ended = append(got.ended, end)
	}})
}

func TestCancelEndsAnAgentTreeThatIgnoresTerm(t *testing.T) {
	cases := map[string]string{
		"child in the agent's group": `trap '' TERM
sh -c 'trap "" TERM; while :; do sleep 1; done' & echo $! > "$PWD/child"
while :; do sleep 1; done`,
		"child in a group of its own": `trap '' TERM
set -m
sh -c 'trap "" TERM; while :; do sleep 1; done' & echo $! > "$PWD/child"
set +m
while :; do sleep 1; done`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			shortGrace(t)
			workspace, runDir := t.TempDir(), t.TempDir()
			got := &recorded{}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			bin := fakeAgent(t, body)
			done := make(chan error, 1)
			go func() {
				_, err := RunModel(supervised(ctx, runDir, got), bin, "", workspace, "x", nil)
				done <- err
			}()
			child := readPID(t, filepath.Join(workspace, "child"))
			cancel()
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("err = %v", err)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("RunModel did not return")
			}
			// Everything is already gone when RunModel returns: the run lock is released after it.
			if err := syscall.Kill(child, 0); !errors.Is(err, syscall.ESRCH) {
				t.Fatalf("child %d outlived RunModel: %v", child, err)
			}
			seen := false
			for _, end := range got.ended {
				if end.PID == child {
					seen = end.Stage == "build" && end.Agent == "cursor-agent" && end.Signal == "SIGKILL" && end.Exit != ""
				}
			}
			if !seen || len(got.ended) < 2 {
				t.Fatalf("terminations = %+v", got.ended)
			}
			if _, err := os.Stat(filepath.Join(runDir, "agent.pid")); !os.IsNotExist(err) {
				t.Fatal("the run kept a record of a dead agent")
			}
		})
	}
}

func TestAnAgentThatEndsNormallyTakesItsLeftoverChildWithIt(t *testing.T) {
	shortGrace(t)
	workspace, runDir := t.TempDir(), t.TempDir()
	got := &recorded{}
	// The child keeps the output pipe open, as a forked worker server does.
	bin := fakeAgent(t, `sh -c 'trap "" TERM; while :; do sleep 1; done' & echo $! > "$PWD/child"
echo done`)
	out, err := RunModel(supervised(context.Background(), runDir, got), bin, "", workspace, "x", nil)
	if err != nil || !strings.Contains(out, "done") {
		t.Fatalf("out = %q, err = %v", out, err)
	}
	child := readPID(t, filepath.Join(workspace, "child"))
	if !dead(child) {
		t.Fatalf("child %d was left orphaned", child)
	}
	if len(got.ended) == 0 {
		t.Fatal("ending a leftover child recorded nothing")
	}
}
