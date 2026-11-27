package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/worktree"
)

// Two launches of one branch must not both record a run, whatever state root each one uses
// and however each one spells the project folder.
func TestABranchIsOwnedOnceAcrossStateRootsAndSpellings(t *testing.T) {
	dir := gitRepo(t)
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(dir, link); err != nil {
		t.Skipf("no symlinks here: %v", err)
	}
	opt, err := runOptions(runFlags{workspace: dir, brief: briefFile(t), branch: "feat/x", base: "dev"})
	if err != nil {
		t.Fatal(err)
	}
	opt.Root = t.TempDir()
	if _, err := openOwned(opt); err != nil {
		t.Fatal(err)
	}
	again := opt
	again.Root = t.TempDir()
	again.Project, again.Workspace = link, link
	if _, err := openOwned(again); err == nil || !strings.Contains(err.Error(), "already belongs") {
		t.Fatalf("%v", err)
	}
}

func TestConcurrentLaunchesOfOneBranchRecordOneRun(t *testing.T) {
	dir := gitRepo(t)
	opt, err := runOptions(runFlags{workspace: dir, brief: briefFile(t), branch: "feat/x", base: "dev"})
	if err != nil {
		t.Fatal(err)
	}
	opened := countOpened(8, func() error {
		_, err := openOwned(opt)
		return err
	})
	if opened != 1 {
		t.Fatalf("%d runs were recorded on one branch", opened)
	}
}

func TestARunThatCannotStartLeavesNoBranchAndNoTree(t *testing.T) {
	dir, _ := runRepo(t)
	missingBrief := filepath.Join(t.TempDir(), "nope.md")
	if code := run([]string{"--brief", missingBrief, "--workspace", dir, "--branch", "feat/x"}); code != 1 {
		t.Fatalf("exit %d", code)
	}
	if got := gitOut(t, dir, "branch", "--list", "feat/x"); got != "" {
		t.Fatalf("a branch was left: %s", got)
	}
	if _, err := os.Stat(worktree.Path(dir, "feat/x")); !os.IsNotExist(err) {
		t.Fatalf("a tree was left: %v", err)
	}
}

func TestAnInvalidBranchNameMakesNoTree(t *testing.T) {
	dir, brief := runRepo(t)
	flags, note, err := placeRunFlags(runFlags{workspace: dir, brief: brief, branch: "wip/x"})
	if err != nil || flags.isolate || note != "" {
		t.Fatalf("isolate %v note %q err %v", flags.isolate, note, err)
	}
}

func briefFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "b.md")
	if err := os.WriteFile(path, []byte("Refund the double charge\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// The launch that loses a race for a branch must not remove the tree or the branch of the
// one that won it.
func TestALostLaunchKeepsTheTreeOfTheWinner(t *testing.T) {
	for range 5 {
		dir, brief := runRepo(t)
		flags := runFlags{workspace: dir, brief: brief, branch: "feat/x", base: "dev", isolate: true}
		opened := countOpened(6, func() error {
			_, _, err := openRun(flags)
			return err
		})
		if opened != 1 {
			t.Fatalf("%d runs opened", opened)
		}
		if _, err := os.Stat(worktree.Path(dir, "feat/x")); err != nil {
			t.Fatalf("the winner lost its tree: %v", err)
		}
		if gitOut(t, dir, "branch", "--list", "feat/x") == "" {
			t.Fatal("the winner lost its branch")
		}
	}
}

// countOpened runs launch n times at once and counts the launches that opened a run.
func countOpened(n int, launch func() error) int {
	results := make(chan error, n)
	for range n {
		go func() { results <- launch() }()
	}
	opened := 0
	for range n {
		if <-results == nil {
			opened++
		}
	}
	return opened
}
