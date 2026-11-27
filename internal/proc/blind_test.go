//go:build !windows

package proc

import (
	"context"
	"errors"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestATreeThatCannotBeListedIsStillKilledAndRecorded(t *testing.T) {
	grace, list := Grace, listLive
	Grace = 300 * time.Millisecond
	listLive = func(*finder) (map[int]entry, error) { return nil, errors.New("no ps") }
	t.Cleanup(func() { Grace, listLive = grace, list })
	cmd := exec.CommandContext(context.Background(), "sh", "-c", "trap '' TERM; while :; do sleep 1; done")
	Tame(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	t.Cleanup(func() { kill(cmd.Process.Pid) })
	time.Sleep(100 * time.Millisecond)

	ends := endTree(&finder{group: cmd.Process.Pid, known: map[int]string{}})

	if len(ends) != 1 || ends[0].Signal != "SIGKILL" || ends[0].Exit != "exited" {
		t.Fatalf("terminations = %+v", ends)
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("the process survived: %v", syscall.Kill(cmd.Process.Pid, 0))
	}
}

func TestAListingThatKeepsFailingIsLoggedOnce(t *testing.T) {
	list := listLive
	listLive = func(*finder) (map[int]entry, error) { return nil, errors.New("no ps") }
	t.Cleanup(func() { listLive = list })
	var logged strings.Builder
	log.SetOutput(&logged)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	tree := &Tree{cmd: &exec.Cmd{Process: &os.Process{Pid: 4242}}}

	tree.record()
	tree.record()

	if n := strings.Count(logged.String(), "record the tree"); n != 1 {
		t.Fatalf("logged %d times: %s", n, logged.String())
	}
}
