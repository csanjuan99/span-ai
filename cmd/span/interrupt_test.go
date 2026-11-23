//go:build !windows

package main

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func TestTheSecondInterruptForcesTheExitAfterTheFirstCancels(t *testing.T) {
	forced := make(chan struct{}, 1)
	old := forceExit
	forceExit = func() { forced <- struct{}{} }
	defer func() { forceExit = old }()

	ctx, stop := interruptContext()
	defer stop()

	if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("the first interrupt did not cancel the context")
	}
	select {
	case <-forced:
		t.Fatal("one interrupt must be a graceful stop, not a forced exit")
	case <-time.After(150 * time.Millisecond):
	}
	if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
		t.Fatal(err)
	}
	select {
	case <-forced:
	case <-time.After(2 * time.Second):
		t.Fatal("the second interrupt did not force the exit")
	}
}

func TestStoppingTheContextWithoutASignalNeverForcesAnExit(t *testing.T) {
	forced := make(chan struct{}, 1)
	old := forceExit
	forceExit = func() { forced <- struct{}{} }
	defer func() { forceExit = old }()

	_, stop := interruptContext()
	stop()
	select {
	case <-forced:
		t.Fatal("finishing normally forced an exit")
	case <-time.After(150 * time.Millisecond):
	}
}
