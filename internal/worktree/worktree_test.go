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

// remoteSetup is a bare origin with dev and feat/SAL-3, the seed repository that pushes to
// it, and a clone that only has dev locally: feat/SAL-3 exists on origin alone, as a
// teammate's pushed branch does.
type remoteSetup struct {
	clone, seed, origin string
	s                   shell
}

func cloneWithRemoteOnly(t *testing.T) remoteSetup {
	t.Helper()
	seed, s := repoWithDev(t)
	s.must(seed, "checkout", "-q", "-b", "feat/SAL-3")
	commitFile(t, s, seed, "wallet.txt")
	origin := filepath.Join(t.TempDir(), "origin.git")
	s.must(seed, "clone", "-q", "--bare", seed, origin)
	clone := filepath.Join(t.TempDir(), "clone")
	s.must(t.TempDir(), "clone", "-q", "-b", "dev", origin, clone)
	s.must(clone, "config", "user.name", "Span Tests")
	s.must(clone, "config", "user.email", "tests@example.test")
	return remoteSetup{clone: clone, seed: seed, origin: origin, s: s}
}

func commitFile(t *testing.T, s shell, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(name+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s.must(dir, "add", name)
	s.must(dir, "commit", "-q", "-m", name)
}

func TestStartPointPrefersTheLocalBranchWithoutARemote(t *testing.T) {
	repo, s := repoWithDev(t)
	got, err := StartPoint(context.Background(), s, repo, "dev")
	if err != nil || got != "refs/heads/dev" {
		t.Fatalf("start = %q, %v", got, err)
	}
}

func TestStartPointUsesOriginForABranchOnlyOnOrigin(t *testing.T) {
	r := cloneWithRemoteOnly(t)
	// Forget the remote-tracking ref so only a fetch can find it.
	r.s.must(r.clone, "update-ref", "-d", "refs/remotes/origin/feat/SAL-3")
	got, err := StartPoint(context.Background(), r.s, r.clone, "feat/SAL-3")
	if err != nil || got != "refs/remotes/origin/feat/SAL-3" {
		t.Fatalf("start = %q, %v", got, err)
	}
}

// A local base that is behind origin must not cut the branch from its old commit, and the
// person's own branch is left where it was.
func TestStartPointPrefersOriginOverAStaleLocalBase(t *testing.T) {
	r := cloneWithRemoteOnly(t)
	r.s.must(r.clone, "branch", "--no-track", "feat/SAL-3", "origin/feat/SAL-3")
	old := r.s.must(r.clone, "rev-parse", "refs/heads/feat/SAL-3")
	r.s.must(r.seed, "checkout", "-q", "feat/SAL-3")
	commitFile(t, r.s, r.seed, "more.txt")
	r.s.must(r.seed, "push", "-q", r.origin, "feat/SAL-3")
	want := r.s.must(r.seed, "rev-parse", "HEAD")
	got, err := StartPoint(context.Background(), r.s, r.clone, "feat/SAL-3")
	if err != nil || got != "refs/remotes/origin/feat/SAL-3" {
		t.Fatalf("start = %q, %v", got, err)
	}
	if sha := r.s.must(r.clone, "rev-parse", got); sha != want {
		t.Fatalf("origin ref %s, want the pushed %s", sha, want)
	}
	if local := r.s.must(r.clone, "rev-parse", "refs/heads/feat/SAL-3"); local != old {
		t.Fatalf("the local base moved from %s to %s", old, local)
	}
}

func TestStartPointPrefersOriginWhenLocalDiverged(t *testing.T) {
	r := cloneWithRemoteOnly(t)
	r.s.must(r.clone, "checkout", "-q", "--no-track", "-b", "feat/SAL-3", "origin/feat/SAL-3")
	commitFile(t, r.s, r.clone, "local.txt")
	r.s.must(r.clone, "checkout", "-q", "dev")
	got, err := StartPoint(context.Background(), r.s, r.clone, "feat/SAL-3")
	if err != nil || got != "refs/remotes/origin/feat/SAL-3" {
		t.Fatalf("start = %q, %v", got, err)
	}
}

func TestStartPointUsesALocalOnlyBranchForCutting(t *testing.T) {
	r := cloneWithRemoteOnly(t)
	r.s.must(r.clone, "branch", "feat/local-only")
	got, err := StartPoint(context.Background(), r.s, r.clone, "feat/local-only")
	if err != nil || got != "refs/heads/feat/local-only" {
		t.Fatalf("start = %q, %v", got, err)
	}
}

func TestStartPointOfflineFallsBackToTheLocalBranch(t *testing.T) {
	r := cloneWithRemoteOnly(t)
	r.s.must(r.clone, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "gone.git"))
	got, err := StartPoint(context.Background(), r.s, r.clone, "dev")
	if err != nil || got != "refs/heads/dev" {
		t.Fatalf("start = %q, %v", got, err)
	}
}

func TestStartPointNamesAMissingBase(t *testing.T) {
	r := cloneWithRemoteOnly(t)
	r.s.must(r.clone, "tag", "v1.0")
	for _, base := range []string{"feat/nowhere", "v1.0"} {
		_, err := StartPoint(context.Background(), r.s, r.clone, base)
		if err == nil || err.Error() != "base branch "+base+" does not exist on origin; push it or choose another base" {
			t.Fatalf("%s: err = %v", base, err)
		}
	}
}

func TestAHostileBaseNeverReachesGit(t *testing.T) {
	r := cloneWithRemoteOnly(t)
	marker := filepath.Join(t.TempDir(), "PWNED1")
	for _, base := range []string{"--upload-pack=touch " + marker, "-x", "HEAD", r.s.must(r.clone, "rev-parse", "HEAD")} {
		if _, err := StartPoint(context.Background(), r.s, r.clone, base); err == nil || !strings.Contains(err.Error(), "invalid base branch") {
			t.Fatalf("StartPoint(%q) err = %v", base, err)
		}
		if _, err := OnOrigin(context.Background(), r.s, r.clone, base); err == nil || !strings.Contains(err.Error(), "invalid base branch") {
			t.Fatalf("OnOrigin(%q) err = %v", base, err)
		}
		if err := Add(context.Background(), r.s, r.clone, filepath.Join(t.TempDir(), "w"), "feat/x", base); err == nil {
			t.Fatalf("Add accepted base %q", base)
		}
	}
	if err := Add(context.Background(), r.s, r.clone, filepath.Join(t.TempDir(), "w"), "--upload-pack=touch "+marker, "dev"); err == nil {
		t.Fatal("Add accepted a hostile branch")
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("git ran the injected command")
	}
}

// The open stage needs the base on origin: the forge cannot target a branch only this
// checkout has, nor a tag. Without a remote there is no forge, so local is enough.
func TestOnOriginForTheOpenStage(t *testing.T) {
	r := cloneWithRemoteOnly(t)
	r.s.must(r.clone, "branch", "feat/local-only")
	r.s.must(r.clone, "tag", "v1.0")
	for base, want := range map[string]bool{"dev": true, "feat/SAL-3": true, "feat/local-only": false, "v1.0": false, "feat/nowhere": false} {
		got, err := OnOrigin(context.Background(), r.s, r.clone, base)
		if err != nil || got != want {
			t.Errorf("OnOrigin(%q) = %v, %v; want %v", base, got, err, want)
		}
	}
	repo, s := repoWithDev(t)
	for base, want := range map[string]bool{"dev": true, "feat/nowhere": false} {
		got, err := OnOrigin(context.Background(), s, repo, base)
		if err != nil || got != want {
			t.Errorf("no remote: OnOrigin(%q) = %v, %v; want %v", base, got, err, want)
		}
	}
}

func TestANewBranchStartsFromTheOriginOnlyBase(t *testing.T) {
	r := cloneWithRemoteOnly(t)
	path := filepath.Join(t.TempDir(), "stacked")
	if err := Add(context.Background(), r.s, r.clone, path, "feat/SAL-4", "feat/SAL-3"); err != nil {
		t.Fatal(err)
	}
	want := r.s.must(r.clone, "rev-parse", "origin/feat/SAL-3")
	if got := r.s.must(path, "rev-parse", "HEAD"); got != want {
		t.Fatalf("HEAD = %s, want %s", got, want)
	}
	// Cut without tracking: the new branch pushes to its own name, not into its base.
	if _, err := r.s.Git(context.Background(), r.clone, "config", "--get", "branch.feat/SAL-4.merge"); err == nil {
		t.Fatal("the new branch tracks its base")
	}
}
