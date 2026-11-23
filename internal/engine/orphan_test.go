//go:build !windows

package engine

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"span/internal/proc"
)

// A Span that was killed with SIGKILL cannot stop its agent. The agent keeps running with
// no owner, so the next run of the tree has to end it before it starts one of its own.
func TestAResumedRunEndsAnAgentItsDeadPredecessorLeftRunning(t *testing.T) {
	opt, _, _, _, _, _ := baseOpt(t)
	orphan := exec.CommandContext(context.Background(), "sh", "-c", "sleep 60 & wait")
	proc.Tame(orphan)
	if err := orphan.Start(); err != nil {
		t.Fatal(err)
	}
	gone := make(chan struct{})
	go func() { _ = orphan.Wait(); close(gone) }()
	t.Cleanup(func() { _ = orphan.Process.Kill() })

	state := filepath.Join(opt.Workspace, ".span")
	if err := os.MkdirAll(state, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]any{"pid": orphan.Process.Pid, "name": "sleep"})
	if err := os.WriteFile(filepath.Join(state, "agent.pid"), raw, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Start(context.Background(), opt); err != nil {
		t.Fatal(err)
	}
	select {
	case <-gone:
	case <-time.After(5 * time.Second):
		t.Fatal("the orphan agent was still running after the run started")
	}
}
