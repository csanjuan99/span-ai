package main

import (
	"context"
	"os"
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

// waitStarted returns the id of the next run that starts flying.
func waitStarted(t *testing.T, started chan string) string {
	t.Helper()
	select {
	case id := <-started:
		return id
	case <-time.After(3 * time.Second):
		t.Fatal("the run never started flying")
		return ""
	}
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
	if got := waitStarted(t, started); got != id {
		t.Fatalf("flew run %s, launched %s", got, id)
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
		waitStarted(t, started)
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

// The first repository already started when the second one's configuration cannot be read, so
// the launch reports both instead of answering with an error that drops the running one.
func TestARepositoryWithABrokenPolicyDoesNotHideTheRunsThatStarted(t *testing.T) {
	folder := t.TempDir()
	seed := gitRepo(t)
	for _, name := range []string{"api", "web"} {
		gitOut(t, folder, "clone", "-q", seed, name)
	}
	if err := os.WriteFile(filepath.Join(folder, "web", "span.yaml"), []byte("agent: [\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	started := stubRun(t)
	hub := live.New()
	t.Cleanup(func() {
		hub.StopAll()
		if !hub.Drain(3 * time.Second) {
			t.Error("the run did not stop")
		}
	})
	req := story("AET-1 Charge once")
	req.Workspace, req.Repos = folder, []string{"api", "web"}

	out, err := beginRuns(context.Background(), req, "", hub)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Runs) != 1 || out.Runs[0].Repo != "api" || out.ID != out.Runs[0].ID {
		t.Fatalf("runs %+v id %q", out.Runs, out.ID)
	}
	if !out.Partial || len(out.Failed) != 1 || out.Failed[0].Repo != "web" || out.Failed[0].Reason == "" {
		t.Fatalf("partial=%v failed %+v", out.Partial, out.Failed)
	}
	if !strings.Contains(out.Text, "repo web did not start") {
		t.Fatalf("text %q", out.Text)
	}
	waitStarted(t, started)
}
