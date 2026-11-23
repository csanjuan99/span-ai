//go:build !windows

package proc

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// grandchild starts `script` under sh with Tame applied, waits for the script to write
// the pid of its background child, cancels, and returns that pid and how long Wait took.
func grandchild(t *testing.T, script string) (int, time.Duration, error) {
	t.Helper()
	pidFile := filepath.Join(t.TempDir(), "pid")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", script)
	cmd.Env = append(os.Environ(), "PIDFILE="+pidFile)
	Tame(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	var pid int
	deadline := time.Now().Add(3 * time.Second)
	for pid == 0 {
		if raw, err := os.ReadFile(pidFile); err == nil {
			pid, _ = strconv.Atoi(strings.TrimSpace(string(raw)))
		}
		if pid == 0 && time.Now().After(deadline) {
			t.Fatal("the script never wrote the grandchild pid")
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
	started := time.Now()
	cancel()
	err := cmd.Wait()
	return pid, time.Since(started), err
}

func gone(pid int) bool {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if errors.Is(syscall.Kill(pid, 0), syscall.ESRCH) {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

func TestCancelKillsTheGrandchildToo(t *testing.T) {
	pid, _, err := grandchild(t, `sleep 60 & echo $! > "$PIDFILE"; wait`)
	if err == nil {
		t.Fatal("a cancelled command reported success")
	}
	if !gone(pid) {
		t.Fatalf("grandchild %d survived the cancel", pid)
	}
}

func TestCancelEscalatesWhenTheGroupIgnoresTerm(t *testing.T) {
	old := Grace
	Grace = 200 * time.Millisecond
	t.Cleanup(func() { Grace = old })
	pid, took, _ := grandchild(t, `trap '' TERM; sleep 60 & echo $! > "$PIDFILE"; wait`)
	if !gone(pid) {
		t.Fatalf("grandchild %d ignored TERM and was never killed", pid)
	}
	if took > 5*time.Second {
		t.Fatalf("cancel took %s", took)
	}
}

func TestCancelReturnsPromptlyWhenNothingIsOrphaned(t *testing.T) {
	_, took, _ := grandchild(t, `echo $$ > "$PIDFILE"; sleep 60`)
	if took > 2*time.Second {
		t.Fatalf("cancel took %s for a plain child", took)
	}
}

func TestCancelEndsAProcessThatIsStopped(t *testing.T) {
	old := Grace
	Grace = 3 * time.Second
	t.Cleanup(func() { Grace = old })
	// A command that opens /dev/tty from a background group gets SIGTTIN and stops. A
	// stopped process does not act on SIGTERM until it is continued.
	pid, took, _ := grandchild(t, `echo $$ > "$PIDFILE"; kill -STOP $$`)
	if !gone(pid) {
		t.Fatalf("stopped process %d survived", pid)
	}
	if took > 2*time.Second {
		t.Fatalf("cancel waited %s for Grace instead of continuing the stopped process", took)
	}
}
