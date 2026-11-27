package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/setup"
	"span/internal/worktree"
)

func placed(t *testing.T, dir, text string, inPlace bool) (site, error) {
	t.Helper()
	req := story(text)
	req.Workspace = dir
	req.InPlace = inPlace
	st, err := claimSite(req, "", "starting a run")
	if err == nil {
		t.Cleanup(st.Claim.Release)
	}
	return st, err
}

// ownRun records an unfinished run of branch in the project's state directory.
func ownRun(t *testing.T, dir, id, name string, done bool) {
	t.Helper()
	runDir := filepath.Join(dir, ".span", "runs", id)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cursor := `{"id":"` + id + `","branch":"` + name + `"`
	if done {
		cursor += `,"done":true`
	}
	if err := os.WriteFile(filepath.Join(runDir, "cursor.json"), []byte(cursor+"}"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAStoryOnAFreeCheckoutGetsItsOwnWorktree(t *testing.T) {
	dir := gitRepo(t)
	st, err := placed(t, dir, "AET-1 Charge once", false)
	if err != nil {
		t.Fatal(err)
	}
	if st.Code.Workspace != worktree.Path(st.Code.Project, st.Result.Branch) {
		t.Fatalf("workspace %s", st.Code.Workspace)
	}
	if got := gitOut(t, dir, "rev-parse", "--abbrev-ref", "HEAD"); got != "dev" {
		t.Fatalf("the checkout was switched to %q", got)
	}
}

func TestInPlaceKeepsTheRunInTheCheckout(t *testing.T) {
	dir := gitRepo(t)
	st, err := placed(t, dir, "AET-1 Charge once", true)
	if err != nil {
		t.Fatal(err)
	}
	if st.Code.Workspace != st.Code.Project || st.Claim == nil {
		t.Fatalf("%+v", st.Code)
	}
}

func TestACheckoutAlreadyOnTheBranchIsUsedInPlace(t *testing.T) {
	dir := gitRepo(t)
	first, err := placed(t, dir, "AET-1 Charge once", false)
	if err != nil {
		t.Fatal(err)
	}
	name := first.Result.Branch
	first.discard()
	first.Claim.Release()
	gitOut(t, dir, "checkout", "-q", "-b", name)
	st, err := placed(t, dir, "AET-1 Charge once", false)
	if err != nil {
		t.Fatal(err)
	}
	if st.Code.Workspace != st.Code.Project {
		t.Fatalf("workspace %s", st.Code.Workspace)
	}
}

func TestABranchOfAnUnfinishedRunIsNotShared(t *testing.T) {
	dir := gitRepo(t)
	first, err := placed(t, dir, "AET-1 Charge once", false)
	if err != nil {
		t.Fatal(err)
	}
	ownRun(t, dir, "r1", first.Result.Branch, false)
	first.Claim.Release()
	if _, err := placed(t, dir, "AET-1 Charge once", false); err == nil || !strings.Contains(err.Error(), "already belongs") {
		t.Fatalf("%v", err)
	}
	ownRun(t, dir, "r1", first.Result.Branch, true)
	if _, err := placed(t, dir, "AET-1 Charge once", false); err != nil {
		t.Fatalf("a finished run must not block its branch: %v", err)
	}
}

func TestDiscardingKeepsABranchThatAlreadyExisted(t *testing.T) {
	dir := gitRepo(t)
	first, err := placed(t, dir, "AET-1 Charge once", false)
	if err != nil {
		t.Fatal(err)
	}
	name := first.Result.Branch
	first.discard()
	first.Claim.Release()
	gitOut(t, dir, "branch", name)
	st, err := placed(t, dir, "AET-1 Charge once", false)
	if err != nil {
		t.Fatal(err)
	}
	st.discard()
	if got := gitOut(t, dir, "branch", "--list", name); got == "" {
		t.Fatal("a branch this launch did not create was deleted")
	}
}

func TestRunWithABranchGetsItsOwnWorktree(t *testing.T) {
	dir, brief := runRepo(t)
	flags, note, err := placeRunFlags(runFlags{workspace: dir, brief: brief, branch: "feat/x"})
	if err != nil || !flags.isolate || !strings.Contains(note, worktree.Path(dir, "feat/x")) {
		t.Fatalf("isolate %v note %q err %v", flags.isolate, note, err)
	}
}

func TestRunInPlaceOptOut(t *testing.T) {
	dir, brief := runRepo(t)
	flags, note, err := placeRunFlags(runFlags{workspace: dir, brief: brief, branch: "feat/x", inPlace: true})
	if err != nil || flags.isolate || note != "" {
		t.Fatalf("isolate %v note %q err %v", flags.isolate, note, err)
	}
	if code := run([]string{"--brief", brief, "--workspace", dir, "--branch", "feat/x", "--in-place", "--worktree"}); code != 2 {
		t.Fatalf("--worktree with --in-place exited %d, want 2", code)
	}
}

func TestQuickstartConfiguresWithoutAStoryInPlace(t *testing.T) {
	dir := gitRepo(t)
	out, code := runOne(context.Background(), setup.Request{Workspace: dir})
	if code != 0 || !strings.Contains(out, "configuration ready") {
		t.Fatalf("%d %s", code, out)
	}
}

func TestAnExplicitWorktreeDoesNotShareTheBranchOfAnUnfinishedRun(t *testing.T) {
	dir, brief := runRepo(t)
	ownRun(t, dir, "r1", "feat/x", false)
	_, _, err := placeRunFlags(runFlags{workspace: dir, brief: brief, branch: "feat/x", isolate: true})
	if err == nil || !strings.Contains(err.Error(), "already belongs") {
		t.Fatalf("%v", err)
	}
}
