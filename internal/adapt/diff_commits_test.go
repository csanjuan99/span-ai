package adapt

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

func headOf(t *testing.T, dir string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func TestDiffCountsTheCommitsTheBranchAddedToTheBase(t *testing.T) {
	dir := repo(t)
	commitAll(t, dir, "README.md", "base\n")
	base := headOf(t, dir)
	commitAll(t, dir, "a.go", "package a\n")
	commitAll(t, dir, "b.go", "package b\n")
	commitAll(t, dir, "c.go", "package c\n")

	got, err := (Git{Dir: dir}).Diff(context.Background(), base, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if got.Commits != 3 {
		t.Fatalf("commits = %d, want 3", got.Commits)
	}
}

func TestDiffCountsNoCommitsWhenHeadIsTheBase(t *testing.T) {
	dir := repo(t)
	commitAll(t, dir, "README.md", "base\n")
	got, err := (Git{Dir: dir}).Diff(context.Background(), "HEAD", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if got.Commits != 0 {
		t.Fatalf("commits = %d, want 0", got.Commits)
	}
}
