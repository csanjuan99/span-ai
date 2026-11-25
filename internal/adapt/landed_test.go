package adapt

import (
	"context"
	"strings"
	"testing"
)

// merged builds main with one commit, a branch with two, and the branch merged back with a
// merge commit. It returns the repo, the branch head and main before the merge.
func merged(t *testing.T) (dir, head, before string) {
	t.Helper()
	dir = repo(t)
	git(t, dir, "checkout", "-q", "-b", "main")
	commitAll(t, dir, "README.md", "base\n")
	before = git(t, dir, "rev-parse", "HEAD")
	git(t, dir, "checkout", "-q", "-b", "feat/x")
	commitAll(t, dir, "a.go", "package a\n")
	commitAll(t, dir, "b.go", "package b\n")
	head = git(t, dir, "rev-parse", "HEAD")
	git(t, dir, "checkout", "-q", "main")
	git(t, dir, "merge", "-q", "--no-ff", "-m", "Merge pull request #1", "feat/x")
	return dir, head, before
}

func TestLandedFindsMainBeforeTheMergeThatTookTheBranch(t *testing.T) {
	dir, head, before := merged(t)
	got, ok, err := (Git{Dir: dir}).Landed(context.Background(), "main", head)
	if err != nil || !ok || got != before {
		t.Fatalf("Landed = %q, %v, %v; want %q", got, ok, err, before)
	}
}

func TestADiffFromWhereItLandedStillShowsTheWork(t *testing.T) {
	dir, head, before := merged(t)
	if plain, _ := (Git{Dir: dir}).Diff(context.Background(), "main", head); plain.Stat != "" {
		t.Fatalf("a merged branch diffed against its base should be empty, got %q", plain.Stat)
	}
	got, err := (Git{Dir: dir}).Diff(context.Background(), before, head)
	if err != nil {
		t.Fatal(err)
	}
	if got.Commits != 2 || !strings.Contains(got.Stat, "a.go") || !strings.Contains(got.Stat, "b.go") {
		t.Fatalf("diff = %+v", got)
	}
}

func TestLandedSaysNothingForABranchThatWasNotMerged(t *testing.T) {
	dir := repo(t)
	git(t, dir, "checkout", "-q", "-b", "main")
	commitAll(t, dir, "README.md", "base\n")
	git(t, dir, "checkout", "-q", "-b", "feat/y")
	commitAll(t, dir, "c.go", "package c\n")
	head := git(t, dir, "rev-parse", "HEAD")
	got, ok, err := (Git{Dir: dir}).Landed(context.Background(), "main", head)
	if err != nil || ok || got != "" {
		t.Fatalf("Landed = %q, %v, %v; want nothing", got, ok, err)
	}
}

func TestLandedSaysNothingForAFastForward(t *testing.T) {
	dir := repo(t)
	git(t, dir, "checkout", "-q", "-b", "main")
	commitAll(t, dir, "README.md", "base\n")
	git(t, dir, "checkout", "-q", "-b", "feat/z")
	commitAll(t, dir, "d.go", "package d\n")
	head := git(t, dir, "rev-parse", "HEAD")
	git(t, dir, "checkout", "-q", "main")
	git(t, dir, "merge", "-q", "--ff-only", "feat/z")
	if _, ok, err := (Git{Dir: dir}).Landed(context.Background(), "main", head); err != nil || ok {
		t.Fatalf("a fast-forward leaves no merge commit: ok=%v err=%v", ok, err)
	}
}

func TestLandedRefusesAnOptionInPlaceOfACommit(t *testing.T) {
	if _, _, err := (Git{Dir: repo(t)}).Landed(context.Background(), "-x", "HEAD"); err == nil {
		t.Fatal("want an error for a base that git could read as an option")
	}
}

func TestLandedReportsAnUnknownCommitInsteadOfSayingNothingLanded(t *testing.T) {
	dir := repo(t)
	git(t, dir, "checkout", "-q", "-b", "main")
	commitAll(t, dir, "README.md", "base\n")
	if _, ok, err := (Git{Dir: dir}).Landed(context.Background(), "main", strings.Repeat("0", 40)); err == nil || ok {
		t.Fatalf("a commit git does not know must be an error, got ok=%v err=%v", ok, err)
	}
}
