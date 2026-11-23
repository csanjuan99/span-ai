package worktree

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// shell is a Runner that runs git for real. The package is only useful if the commands
// it prints are the commands git accepts, so the tests use git, not a fake.
type shell struct{ t *testing.T }

func (s shell) Git(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out)), nil
}

func (s shell) must(dir string, args ...string) string {
	s.t.Helper()
	out, err := s.Git(context.Background(), dir, args...)
	if err != nil {
		s.t.Fatalf("git %v: %v", args, err)
	}
	return out
}

// repoWithDev makes a checkout with one commit on dev, which is what a run needs.
func repoWithDev(t *testing.T) (string, shell) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := t.TempDir()
	s := shell{t: t}
	s.must(dir, "init", "-q", "-b", "dev")
	s.must(dir, "config", "user.name", "Span Tests")
	s.must(dir, "config", "user.email", "tests@example.test")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s.must(dir, "add", "README.md")
	s.must(dir, "commit", "-q", "-m", "base")
	return dir, s
}

func TestAWorktreeIsABranchInItsOwnDirectory(t *testing.T) {
	repo, s := repoWithDev(t)
	path := filepath.Join(t.TempDir(), "sal-3")

	if err := Add(context.Background(), s, repo, path, "feat/sal-3", "dev"); err != nil {
		t.Fatal(err)
	}
	entries, err := List(context.Background(), s, repo)
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := At(entries, path)
	if !ok {
		t.Fatalf("the worktree is not listed: %+v", entries)
	}
	if entry.Branch != "feat/sal-3" {
		t.Fatalf("branch = %q", entry.Branch)
	}
	// The shared checkout is untouched: still on dev.
	if head := s.must(repo, "rev-parse", "--abbrev-ref", "HEAD"); head != "dev" {
		t.Fatalf("the shared checkout moved to %q", head)
	}
}

func TestTwoRunsGetTwoTrees(t *testing.T) {
	repo, s := repoWithDev(t)
	one := filepath.Join(t.TempDir(), "sal-3")
	two := filepath.Join(t.TempDir(), "sal-4")
	if err := Add(context.Background(), s, repo, one, "feat/sal-3", "dev"); err != nil {
		t.Fatal(err)
	}
	if err := Add(context.Background(), s, repo, two, "feat/sal-4", "dev"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(one, "only-here.ts"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(two, "only-here.ts")); !os.IsNotExist(err) {
		t.Fatal("a file from one run appeared in the other tree")
	}
	if _, err := os.Stat(filepath.Join(repo, "only-here.ts")); !os.IsNotExist(err) {
		t.Fatal("a file from a run appeared in the shared checkout")
	}
}

func TestReusingAWorktreeLandsOnTheSameBranch(t *testing.T) {
	repo, s := repoWithDev(t)
	path := filepath.Join(t.TempDir(), "sal-3")
	if err := Add(context.Background(), s, repo, path, "feat/sal-3", "dev"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "work.ts"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// The branch exists now, so a second call reuses it instead of failing.
	if err := Add(context.Background(), s, repo, path+"/", "feat/sal-3", "dev"); err != nil {
		t.Fatalf("reusing a worktree failed: %v", err)
	}
}

func TestRemovingAWorktreeKeepsTheBranch(t *testing.T) {
	repo, s := repoWithDev(t)
	path := filepath.Join(t.TempDir(), "sal-3")
	if err := Add(context.Background(), s, repo, path, "feat/sal-3", "dev"); err != nil {
		t.Fatal(err)
	}
	if err := Remove(context.Background(), s, repo, path); err != nil {
		t.Fatal(err)
	}
	entries, err := List(context.Background(), s, repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := At(entries, path); ok {
		t.Fatal("the worktree is still listed")
	}
	// The branch survives, so the work is not lost.
	s.must(repo, "rev-parse", "--verify", "refs/heads/feat/sal-3")
}
