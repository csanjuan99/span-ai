package adapt

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// originAndClone is a bare origin with dev and feat/SAL-3, and a clone that has only dev
// locally, so feat/SAL-3 lives on origin alone.
func originAndClone(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	git := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	seed := t.TempDir()
	git(seed, "init", "-q", "-b", "dev")
	git(seed, "config", "user.name", "Span Tests")
	git(seed, "config", "user.email", "tests@example.test")
	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("span\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(seed, "add", "README.md")
	git(seed, "commit", "-q", "-m", "init")
	git(seed, "branch", "feat/SAL-3")
	origin := filepath.Join(t.TempDir(), "origin.git")
	git(seed, "clone", "-q", "--bare", seed, origin)
	clone := filepath.Join(t.TempDir(), "clone")
	git(t.TempDir(), "clone", "-q", "-b", "dev", origin, clone)
	git(clone, "config", "user.name", "Span Tests")
	git(clone, "config", "user.email", "tests@example.test")
	return clone
}

// The open stage asks: can the forge target this base? Only a branch origin has can be.
func TestBaseExistsMeansOnOrigin(t *testing.T) {
	clone := originAndClone(t)
	git := Git{Dir: clone}
	if _, err := git.Git(context.Background(), clone, "branch", "feat/local-only"); err != nil {
		t.Fatal(err)
	}
	if _, err := git.Git(context.Background(), clone, "tag", "v1.0"); err != nil {
		t.Fatal(err)
	}
	for base, want := range map[string]bool{"dev": true, "feat/SAL-3": true, "feat/local-only": false, "v1.0": false, "feat/nowhere": false} {
		got, err := git.BaseExists(context.Background(), base)
		if err != nil || got != want {
			t.Errorf("BaseExists(%q) = %v, %v; want %v", base, got, err, want)
		}
	}
}

func TestEnsureBranchCutsAStackedBranchFromOrigin(t *testing.T) {
	clone := originAndClone(t)
	git := Git{Dir: clone}
	if err := git.EnsureBranch(context.Background(), "feat/SAL-3", "feat/SAL-4"); err != nil {
		t.Fatal(err)
	}
	head, err := git.Head(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want, err := git.Git(context.Background(), clone, "rev-parse", "origin/feat/SAL-3")
	if err != nil || head != want {
		t.Fatalf("HEAD %s, want origin/feat/SAL-3 %s (%v)", head, want, err)
	}
	if _, err := git.Git(context.Background(), clone, "config", "--get", "branch.feat/SAL-4.merge"); err == nil {
		t.Fatal("the new branch tracks its base")
	}
	err = git.EnsureBranch(context.Background(), "feat/nowhere", "feat/SAL-5")
	if err == nil || err.Error() != "base branch feat/nowhere does not exist on origin; push it or choose another base" {
		t.Fatalf("err = %v", err)
	}
}

func TestAHostileBaseOrBranchNeverReachesGit(t *testing.T) {
	clone := originAndClone(t)
	git := Git{Dir: clone}
	marker := filepath.Join(t.TempDir(), "PWNED1")
	hostile := "--upload-pack=touch " + marker
	for _, base := range []string{hostile, "-x", "HEAD", "--repo=evil/x"} {
		if _, err := git.BaseExists(context.Background(), base); err == nil || !strings.Contains(err.Error(), "invalid base branch") {
			t.Errorf("BaseExists(%q) err = %v", base, err)
		}
		if err := git.EnsureBranch(context.Background(), base, "feat/SAL-6"); err == nil || !strings.Contains(err.Error(), "invalid base branch") {
			t.Errorf("EnsureBranch(%q) err = %v", base, err)
		}
		if _, err := git.Changed(context.Background(), base); err == nil {
			t.Errorf("Changed(%q) was accepted", base)
		}
		if _, err := git.Diff(context.Background(), base, "HEAD"); err == nil && strings.HasPrefix(base, "-") {
			t.Errorf("Diff(%q) was accepted", base)
		}
	}
	if err := git.EnsureBranch(context.Background(), "dev", hostile); err == nil {
		t.Error("EnsureBranch accepted a hostile branch")
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("git ran the injected command")
	}
}
