package adapt

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGitCommitExcludesSpanState(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := t.TempDir()
	git := Git{Dir: dir}
	for _, args := range [][]string{
		{"init"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".span"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".span", "secret"), []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	sha, err := git.Commit(context.Background(), "feat: start")
	if err != nil {
		t.Fatal(err)
	}
	if sha == "" {
		t.Fatal("empty sha")
	}
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "main.go\n" {
		t.Fatalf("tracked %q", out)
	}
}

func TestLocalReviewOpenThenFind(t *testing.T) {
	dir := t.TempDir()
	review := LocalReview{Dir: dir}
	url, err := review.Open(context.Background(), "span-1", "feat/payments", "payments", "brief")
	if err != nil {
		t.Fatal(err)
	}
	foundURL, found, err := review.Find(context.Background(), "span-1", "feat/payments")
	if err != nil || !found || foundURL != url {
		t.Fatalf("found %v %s %v", found, foundURL, err)
	}
	merged, err := review.Merged(context.Background(), url)
	if err != nil || merged {
		t.Fatalf("merged %v %v", merged, err)
	}
	if err := review.Merge(context.Background(), url); err != nil {
		t.Fatal(err)
	}
	merged, err = review.Merged(context.Background(), url)
	if err != nil || !merged {
		t.Fatalf("after merge %v %v", merged, err)
	}
}

func TestEnsureBranchCreatesFeatFromDevAndLeavesItAlone(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	run("init", "-b", "master")
	run("config", "user.name", "Span Tests")
	run("config", "user.email", "tests@example.test")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("span\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "README.md")
	run("commit", "-m", "init")
	run("branch", "dev")
	git := Git{Dir: dir}
	if err := git.EnsureBranch(context.Background(), "dev", "feat/add-saved-cards"); err != nil {
		t.Fatal(err)
	}
	name, err := git.Branch(context.Background())
	if err != nil || name != "feat/add-saved-cards" {
		t.Fatalf("branch %s %v", name, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := git.Commit(context.Background(), "feat: keep"); err != nil {
		t.Fatal(err)
	}
	run("checkout", "dev")
	if err := git.EnsureBranch(context.Background(), "dev", "feat/add-saved-cards"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "keep.txt")); err != nil {
		t.Fatal("existing branch was reset")
	}
	if err := git.EnsureBranch(context.Background(), "nope", "chore/nope"); err == nil {
		t.Fatal("created a branch outside feat, fix, and hotfix")
	}
}

func TestGitCommitLeavesThePolicyFileOutOfTheBranch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	for name, body := range map[string]string{"main.go": "package main\n", "span.yaml": "feature: x\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := (Git{Dir: dir}).Commit(context.Background(), "feat: start"); err != nil {
		t.Fatal(err)
	}
	list := exec.Command("git", "ls-files")
	list.Dir = dir
	out, err := list.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "main.go\n" {
		t.Fatalf("span.yaml is the person's policy, not the story's work; tracked %q", out)
	}
}

func TestGitCommitWorksWhenTheRepositoryIgnoresSpanState(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := repo(t)
	if err := os.MkdirAll(filepath.Join(dir, ".span"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		".gitignore": ".span/\nspan.yaml\n", "main.go": "package main\n", "span.yaml": "agent: any\n", ".span/state": "x\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := (Git{Dir: dir}).Commit(context.Background(), "feat: start"); err != nil {
		t.Fatalf("a repository that ignores .span and span.yaml stopped the commit: %v", err)
	}
	out, err := exec.Command("git", "-C", dir, "ls-files").CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != ".gitignore\nmain.go\n" {
		t.Fatalf("tracked %q", out)
	}
}
