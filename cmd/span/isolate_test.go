package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/engine"
	"span/internal/worktree"
)

func holdCheckout(t *testing.T, dir string) {
	t.Helper()
	claim, err := engine.ClaimWorkspace(dir, "run span-1")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(claim.Release)
}

// A checkout another run is editing is never switched from under it: span run with a
// branch gets its own worktree and starts there, the way a second launch from the page does.
func TestRunOnABusyCheckoutGetsItsOwnWorktree(t *testing.T) {
	dir, brief := runRepo(t)
	holdCheckout(t, dir)
	before, err := os.ReadFile(filepath.Join(dir, "span.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	flags, note, err := isolateWhenBusy(runFlags{workspace: dir, brief: brief, branch: "feat/second"})
	if err != nil {
		t.Fatal(err)
	}
	tree := worktree.Path(dir, "feat/second")
	if !flags.isolate || !strings.Contains(note, tree) || !strings.Contains(note, "busy") {
		t.Fatalf("isolate %v, note %q", flags.isolate, note)
	}
	opt, err := runOptions(flags)
	if err != nil {
		t.Fatal(err)
	}
	if opt.Workspace != tree || opt.Project != dir || opt.Branch != "feat/second" {
		t.Fatalf("workspace %q project %q branch %q", opt.Workspace, opt.Project, opt.Branch)
	}
	if got := gitOut(t, tree, "rev-parse", "--abbrev-ref", "HEAD"); strings.TrimSpace(got) != "feat/second" {
		t.Fatalf("the tree is on %q", got)
	}
	if got := gitOut(t, dir, "rev-parse", "--abbrev-ref", "HEAD"); strings.TrimSpace(got) != "dev" {
		t.Fatalf("the busy checkout was switched to %q", got)
	}
	after, err := os.ReadFile(filepath.Join(dir, "span.yaml"))
	if err != nil || string(after) != string(before) {
		t.Fatalf("span.yaml in the busy checkout changed: %q %v", after, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "brief.md")); err == nil {
		t.Fatal("a brief was written into the busy checkout")
	}
}

// A free checkout is still edited in place: nothing changes for the one-story case.
func TestRunOnAFreeCheckoutStaysInPlace(t *testing.T) {
	dir, brief := runRepo(t)
	flags, note, err := isolateWhenBusy(runFlags{workspace: dir, brief: brief, branch: "feat/x"})
	if err != nil || flags.isolate || note != "" {
		t.Fatalf("isolate %v note %q err %v", flags.isolate, note, err)
	}
	opt, err := runOptions(flags)
	if err != nil || opt.Workspace != dir {
		t.Fatalf("workspace %q, %v", opt.Workspace, err)
	}
}

// Without a branch there is no name for a tree, so a busy checkout still refuses, and the
// message says how to go on.
func TestRunOnABusyCheckoutWithoutABranchSaysWhatToDo(t *testing.T) {
	dir, brief := runRepo(t)
	holdCheckout(t, dir)
	_, _, err := isolateWhenBusy(runFlags{workspace: dir, brief: brief})
	if err == nil || !strings.Contains(err.Error(), "pass --branch <name> to run in its own worktree") {
		t.Fatalf("%v", err)
	}
	if !strings.Contains(err.Error(), "span-1") {
		t.Fatalf("the holder is not named: %v", err)
	}
}

// An explicit --worktree on a busy checkout keeps working as it did.
func TestRunWithWorktreeOnABusyCheckoutIsUnchanged(t *testing.T) {
	dir, brief := runRepo(t)
	holdCheckout(t, dir)
	flags, note, err := isolateWhenBusy(runFlags{workspace: dir, brief: brief, branch: "feat/w", isolate: true})
	if err != nil || !flags.isolate || note != "" {
		t.Fatalf("isolate %v note %q err %v", flags.isolate, note, err)
	}
}
