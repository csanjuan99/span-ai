package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/setup"
)

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// repoWithFeature is gitRepo plus feat/SAL-3, one commit ahead of dev: the branch a stacked
// story starts from.
func repoWithFeature(t *testing.T) (string, string) {
	t.Helper()
	dir := gitRepo(t)
	gitOut(t, dir, "checkout", "-q", "-b", "feat/SAL-3")
	if err := os.WriteFile(filepath.Join(dir, "wallet.txt"), []byte("wallet\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitOut(t, dir, "add", "wallet.txt")
	gitOut(t, dir, "commit", "-q", "-m", "wallet")
	sha := gitOut(t, dir, "rev-parse", "HEAD")
	gitOut(t, dir, "checkout", "-q", "dev")
	return dir, sha
}

func TestPrepareStartsTheWorktreeFromTheChosenBase(t *testing.T) {
	dir, want := repoWithFeature(t)
	code, err := prepare(dir, "", "", "feat/SAL-4-cards", "feat/SAL-3", true)
	if err != nil {
		t.Fatal(err)
	}
	if got := gitOut(t, code.Workspace, "rev-parse", "HEAD"); got != want {
		t.Fatalf("worktree HEAD %s, want feat/SAL-3 %s", got, want)
	}
}

func TestPrepareWithoutABaseUsesThePolicyBase(t *testing.T) {
	dir, want := repoWithFeature(t)
	writeBody(t, dir, "feature: app\nbase: feat/SAL-3\n")
	code, err := prepare(dir, "", "", "feat/SAL-4-cards", "", true)
	if err != nil {
		t.Fatal(err)
	}
	if got := gitOut(t, code.Workspace, "rev-parse", "HEAD"); got != want {
		t.Fatalf("worktree HEAD %s, want feat/SAL-3 %s", got, want)
	}
}

func TestPrepareWithoutAnyBaseStillStartsFromDev(t *testing.T) {
	dir, _ := repoWithFeature(t)
	want := gitOut(t, dir, "rev-parse", "dev")
	code, err := prepare(dir, "", "", "feat/SAL-4-cards", "", true)
	if err != nil {
		t.Fatal(err)
	}
	if got := gitOut(t, code.Workspace, "rev-parse", "HEAD"); got != want {
		t.Fatalf("worktree HEAD %s, want dev %s", got, want)
	}
}

func TestPrepareRefusesABaseThatIsNowhere(t *testing.T) {
	dir, _ := repoWithFeature(t)
	_, err := prepare(dir, "", "", "feat/SAL-4-cards", "feat/nowhere", true)
	if err == nil || !strings.Contains(err.Error(), "base branch feat/nowhere does not exist on origin; push it or choose another base") {
		t.Fatalf("err = %v", err)
	}
}

func TestALaunchTakesThePolicyBaseForFeatAndFixOnly(t *testing.T) {
	dir, _ := repoWithFeature(t)
	writeBody(t, dir, "feature: app\nbase: feat/SAL-3\n")
	cases := []struct {
		name               string
		req                setup.Request
		wantKind, wantBase string
	}{
		{name: "feat story", req: setup.Request{Story: "Add cards"}, wantKind: "feat", wantBase: "feat/SAL-3"},
		{name: "explicit wins", req: setup.Request{Story: "Add cards", Base: "release/2.1"}, wantKind: "", wantBase: "release/2.1"},
		{name: "hotfix keeps master", req: setup.Request{Story: "refund", Kind: "hotfix"}, wantKind: "hotfix", wantBase: ""},
	}
	for _, tc := range cases {
		tc.req.Workspace = dir
		got, err := setup.WithBase(tc.req)
		if err != nil || got.Kind != tc.wantKind || got.Base != tc.wantBase {
			t.Errorf("%s: kind %q base %q, %v", tc.name, got.Kind, got.Base, err)
		}
	}
}

func TestAProjectMasterBaseNeverMakesAStoryAHotfix(t *testing.T) {
	dir := gitRepo(t)
	writeBody(t, dir, "feature: app\nbase: master\n")
	got, err := setup.WithBase(setup.Request{Workspace: dir, Story: "Add cards"})
	if err != nil || got.Kind != "feat" || got.Base != "master" {
		t.Fatalf("kind %q base %q, %v", got.Kind, got.Base, err)
	}
}

func TestRunBaseDefaultsByKind(t *testing.T) {
	dir := gitRepo(t)
	for name, want := range map[string]string{"": "dev", "feat/x": "dev", "hotfix/x": "master"} {
		got, err := runBase("", dir, name)
		if err != nil || got != want {
			t.Errorf("runBase(%q) = %q, %v; want %q", name, got, err, want)
		}
	}
	writeBody(t, dir, "feature: app\nbase: release/2.1\n")
	for name, want := range map[string]string{"": "release/2.1", "feat/x": "release/2.1", "hotfix/x": "master"} {
		got, err := runBase("", dir, name)
		if err != nil || got != want {
			t.Errorf("with base: runBase(%q) = %q, %v; want %q", name, got, err, want)
		}
	}
}

// hostileBases are what the verifier sent: each must stop before git or gh sees it.
func hostileBases(marker string) []string {
	return []string{"--upload-pack=touch " + marker, "--repo=evil/x", "-x", "HEAD", "0123456789abcdef0123456789abcdef01234567"}
}

func cloneWithOrigin(t *testing.T) string {
	t.Helper()
	seed := gitRepo(t)
	origin := filepath.Join(t.TempDir(), "origin.git")
	gitOut(t, seed, "clone", "-q", "--bare", seed, origin)
	clone := filepath.Join(t.TempDir(), "clone")
	gitOut(t, t.TempDir(), "clone", "-q", "-b", "dev", origin, clone)
	writeBody(t, clone, "feature: app\n")
	return clone
}

// span run --brief b.md --workspace clone --worktree --branch feat/x --base "--upload-pack=touch $S/PWNED1"
func TestRunNeverHandsAnInjectedBaseToGit(t *testing.T) {
	clone := cloneWithOrigin(t)
	marker := filepath.Join(t.TempDir(), "PWNED1")
	brief := filepath.Join(t.TempDir(), "b.md")
	if err := os.WriteFile(brief, []byte("Add saved cards\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, base := range hostileBases(marker) {
		if code := run([]string{"--brief", brief, "--workspace", clone, "--worktree", "--branch", "feat/x", "--base", base}); code != 1 {
			t.Errorf("--worktree --base %q: exit %d", base, code)
		}
		if code := run([]string{"--brief", brief, "--workspace", clone, "--base", base}); code != 1 {
			t.Errorf("--base %q: exit %d", base, code)
		}
		if _, err := prepare(clone, "", "", "feat/x", base, true); err == nil || !strings.Contains(err.Error(), "invalid base branch") {
			t.Errorf("prepare --base %q: err = %v", base, err)
		}
		if _, err := runBase(base, clone, ""); err == nil || !strings.Contains(err.Error(), "invalid base branch") {
			t.Errorf("runBase %q: err = %v", base, err)
		}
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("git ran the injected command")
	}
}

func TestQuickstartRefusesAnInjectedBase(t *testing.T) {
	clone := cloneWithOrigin(t)
	marker := filepath.Join(t.TempDir(), "PWNED1")
	for _, base := range hostileBases(marker) {
		out, code := runQuickstart(context.Background(), setup.Request{Workspace: clone, Story: "Add saved cards", Base: base})
		if code != 1 || !strings.Contains(out, "invalid base branch") {
			t.Errorf("--base %q: exit %d %s", base, code, out)
		}
		if _, err := beginRuns(context.Background(), setup.Request{Workspace: clone, Story: "Add saved cards", Base: base}, "", nil); err == nil || !strings.Contains(err.Error(), "invalid base branch") {
			t.Errorf("desk form base %q: err = %v", base, err)
		}
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("git ran the injected command")
	}
}

func runRepo(t *testing.T) (string, string) {
	t.Helper()
	dir := gitRepo(t)
	writeBody(t, dir, "feature: app\n")
	brief := filepath.Join(t.TempDir(), "b.md")
	if err := os.WriteFile(brief, []byte("Refund the double charge\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir, brief
}

// Without --branch the run works on the checked-out branch, so its prefix decides the
// default base: a hotfix opens into master, never into dev.
func TestRunTakesTheDefaultBaseFromTheCheckedOutBranch(t *testing.T) {
	dir, brief := runRepo(t)
	gitOut(t, dir, "checkout", "-q", "-b", "hotfix/h")
	opt, err := runOptions(runFlags{workspace: dir, brief: brief})
	if err != nil || opt.Base != "master" || opt.Branch != "" {
		t.Fatalf("base %q branch %q, %v", opt.Base, opt.Branch, err)
	}
	opt, err = runOptions(runFlags{workspace: dir, brief: brief, base: "release/2.1"})
	if err != nil || opt.Base != "release/2.1" {
		t.Fatalf("an explicit base must win: %q, %v", opt.Base, err)
	}
	gitOut(t, dir, "checkout", "-q", "dev")
	opt, err = runOptions(runFlags{workspace: dir, brief: brief})
	if err != nil || opt.Base != "dev" {
		t.Fatalf("on dev: base %q, %v", opt.Base, err)
	}
}

// --branch without --worktree is the branch the run works on: the engine checks it out
// from the base instead of the flag being dropped.
func TestRunBranchWithoutAWorktreeIsTheRunsBranch(t *testing.T) {
	dir, brief := runRepo(t)
	opt, err := runOptions(runFlags{workspace: dir, brief: brief, branch: "feat/x"})
	if err != nil || opt.Branch != "feat/x" || opt.Base != "dev" {
		t.Fatalf("branch %q base %q, %v", opt.Branch, opt.Base, err)
	}
	opt, err = runOptions(runFlags{workspace: dir, brief: brief, branch: "hotfix/h"})
	if err != nil || opt.Branch != "hotfix/h" || opt.Base != "master" {
		t.Fatalf("branch %q base %q, %v", opt.Branch, opt.Base, err)
	}
}
