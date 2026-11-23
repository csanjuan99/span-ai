package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"span/internal/engine"
	"span/internal/live"
	"span/internal/setup"
)

// stubRun replaces the engine's Resume so a launch can be tested without an agent CLI.
// Every run it is given blocks until its flight is stopped.
func stubRun(t *testing.T) chan string {
	t.Helper()
	started := make(chan string, 8)
	old := resumeRun
	resumeRun = func(ctx context.Context, _ engine.Options, id string) (engine.Result, error) {
		started <- id
		<-ctx.Done()
		return engine.Result{}, ctx.Err()
	}
	t.Cleanup(func() { resumeRun = old })
	return started
}

func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestALaunchHoldsTheCheckoutForAsLongAsItsRunFlies(t *testing.T) {
	dir := gitRepo(t)
	started := stubRun(t)
	hub := live.New()
	req := story("AET-1 Charge once")
	req.Workspace = dir
	id, stage, err := beginOne(req, "", hub)
	if err != nil {
		t.Fatal(err)
	}
	if id == "" || stage != "intake" {
		t.Fatalf("id=%q stage=%q", id, stage)
	}
	select {
	case got := <-started:
		if got != id {
			t.Fatalf("flew run %s, launched %s", got, id)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the run never started flying")
	}
	holder, busy := engine.WorkspaceBusy(dir)
	if !busy || !strings.Contains(holder, id) {
		t.Fatalf("checkout busy=%v holder=%q, want it held by run %s", busy, holder, id)
	}
	if !hub.Stop(id) {
		t.Fatal("a run must be stoppable from its first frame")
	}
	waitFor(t, "the checkout to be released", func() bool {
		_, busy := engine.WorkspaceBusy(dir)
		return !busy
	})
}

func TestASecondLaunchFliesInItsOwnWorktreeBesideTheFirst(t *testing.T) {
	dir := gitRepo(t)
	started := stubRun(t)
	hub := live.New()
	defer hub.StopAll()
	first := setup.Request{Workspace: dir, Story: "AET-1 Charge once", Kind: "fix", Base: "dev"}
	one, _, err := beginOne(first, "", hub)
	if err != nil {
		t.Fatal(err)
	}
	second := setup.Request{Workspace: dir, Story: "AET-2 Refund twice", Kind: "fix", Base: "dev"}
	two, _, err := beginOne(second, "", hub)
	if err != nil {
		t.Fatalf("a busy checkout must not stop a second story: %v", err)
	}
	for range 2 {
		select {
		case <-started:
		case <-time.After(3 * time.Second):
			t.Fatal("both runs must be flying at once")
		}
	}
	root := filepath.Join(dir, ".span", "runs")
	a, err := engine.Load(root, one)
	if err != nil {
		t.Fatal(err)
	}
	b, err := engine.Load(root, two)
	if err != nil {
		t.Fatal(err)
	}
	if a.Cursor.Workspace == b.Cursor.Workspace {
		t.Fatalf("two runs share %s", a.Cursor.Workspace)
	}
	if b.Cursor.Project != dir || !strings.HasPrefix(b.Cursor.Workspace, filepath.Join(dir, ".span", "worktrees")) {
		t.Fatalf("the second run is not isolated: project=%s workspace=%s", b.Cursor.Project, b.Cursor.Workspace)
	}
	// The launcher leaves the isolated tree to its run, which takes the lock inside the
	// engine; what it must not do is let go of the checkout the first run is editing.
	if holder, busy := engine.WorkspaceBusy(dir); !busy || !strings.Contains(holder, one) {
		t.Fatalf("the first run lost its checkout: busy=%v holder=%q", busy, holder)
	}
}
