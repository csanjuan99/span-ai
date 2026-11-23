package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/engine"
	"span/internal/runlock"
	"span/internal/setup"
)

// gitRepo makes a repository with one commit on the branch "dev".
func gitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	steps := [][]string{
		{"init", "-b", "dev"},
		{"config", "user.email", "t@example.com"},
		{"config", "user.name", "T"},
		{"config", "commit.gpgsign", "false"},
	}
	for _, args := range steps {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Skipf("git unavailable: %v %s", err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "."}, {"commit", "-m", "init"}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	return dir
}

func story(text string) setup.Request {
	return setup.Request{Story: text, Kind: "fix", Base: "dev"}
}

func TestAFreeCheckoutIsEditedInPlaceAndStaysHeld(t *testing.T) {
	dir := gitRepo(t)
	req := story("AET-1 Charge once")
	req.Workspace = dir
	st, err := claimSite(req, "", "starting a run")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Claim.Release()
	if st.Code.Workspace != st.Code.Project {
		t.Fatalf("a free checkout must be edited in place, got %s", st.Code.Workspace)
	}
	if st.Claim == nil {
		t.Fatal("the run must hold the checkout it edits")
	}
	if _, busy := engine.WorkspaceBusy(dir); !busy {
		t.Fatal("the checkout is not held while the run edits it")
	}
}

func TestABusyCheckoutSendsTheNextRunToItsOwnWorktree(t *testing.T) {
	dir := gitRepo(t)
	first := story("AET-1 Charge once")
	first.Workspace = dir
	one, err := claimSite(first, "", "starting a run")
	if err != nil {
		t.Fatal(err)
	}
	defer one.Claim.Release()
	configured, err := os.ReadFile(filepath.Join(dir, "span.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	brief, err := os.ReadFile(filepath.Join(dir, ".span", "brief.md"))
	if err != nil {
		t.Fatal(err)
	}

	second := story("AET-2 Refund twice")
	second.Workspace = dir
	two, err := claimSite(second, "", "starting a run")
	if err != nil {
		t.Fatalf("a busy checkout must not stop a second run: %v", err)
	}
	if two.Code.Workspace == two.Code.Project {
		t.Fatal("the second run shares the checkout the first one is editing")
	}
	if !strings.HasPrefix(two.Code.Workspace, filepath.Join(dir, ".span", "worktrees")) {
		t.Fatalf("worktree is outside .span/worktrees: %s", two.Code.Workspace)
	}
	if _, err := os.Stat(filepath.Join(two.Code.Workspace, "README.md")); err != nil {
		t.Fatalf("the worktree has no code in it: %v", err)
	}
	if two.Claim != nil {
		t.Fatal("an isolated run must not hold the checkout another run is editing")
	}
	if two.Result.Branch == "" || two.Result.Branch == one.Result.Branch {
		t.Fatalf("branches must differ: %q %q", one.Result.Branch, two.Result.Branch)
	}
	if two.Code.Spans != filepath.Join(dir, ".span", "runs") {
		t.Fatalf("state must stay in the project so the desk lists every run: %s", two.Code.Spans)
	}
	after, _ := os.ReadFile(filepath.Join(dir, "span.yaml"))
	if string(after) != string(configured) {
		t.Fatal("the busy checkout's span.yaml was rewritten")
	}
	afterBrief, _ := os.ReadFile(filepath.Join(dir, ".span", "brief.md"))
	if string(afterBrief) != string(brief) {
		t.Fatal("the busy checkout's brief was overwritten under the running story")
	}
	if _, busy := engine.WorkspaceBusy(two.Code.Workspace); busy {
		t.Fatal("the worktree is not held until its own run takes it")
	}
}

func TestParallelRunsEachGetADistinctWorktree(t *testing.T) {
	dir := gitRepo(t)
	first := story("AET-1 Charge once")
	first.Workspace = dir
	one, err := claimSite(first, "", "starting a run")
	if err != nil {
		t.Fatal(err)
	}
	defer one.Claim.Release()
	seen := map[string]bool{}
	for _, text := range []string{"AET-2 Refund twice", "AET-3 Audit log"} {
		req := story(text)
		req.Workspace = dir
		st, err := claimSite(req, "", "starting a run")
		if err != nil {
			t.Fatal(err)
		}
		if seen[st.Code.Workspace] {
			t.Fatalf("two runs share %s", st.Code.Workspace)
		}
		seen[st.Code.Workspace] = true
	}
}

// span.yaml is optional, so a busy checkout that was never configured still sends the next
// story to its own worktree, and writes nothing into the checkout another run holds.
func TestABusyCheckoutThatWasNeverConfiguredGetsItsOwnWorktree(t *testing.T) {
	dir := gitRepo(t)
	held, err := engine.ClaimWorkspace(dir, "another run")
	if err != nil {
		t.Fatal(err)
	}
	defer held.Release()
	req := story("AET-1 Charge once")
	req.Workspace = dir
	st, err := claimSite(req, "", "starting a run")
	if err != nil {
		t.Fatalf("a missing span.yaml stopped an isolated run: %v", err)
	}
	defer st.discard()
	if st.Code.Workspace == st.Code.Project || st.Claim != nil {
		t.Fatalf("the run was not isolated: %+v", st.Code)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "span.yaml")); statErr == nil {
		t.Fatal("configuration was written into a checkout another run holds")
	}
}

func TestConfiguringWithoutAStoryNeverIsolates(t *testing.T) {
	dir := gitRepo(t)
	held, err := engine.ClaimWorkspace(dir, "another run")
	if err != nil {
		t.Fatal(err)
	}
	defer held.Release()
	_, err = claimSite(setup.Request{Workspace: dir}, "", "configuring")
	var busy *runlock.BusyError
	if !errors.As(err, &busy) {
		t.Fatalf("configuration has no run to isolate; want busy, got %v", err)
	}
}

func TestTheSameStoryTwiceNeverSharesABranch(t *testing.T) {
	dir := gitRepo(t)
	first := story("AET-1 Charge once")
	first.Workspace = dir
	one, err := claimSite(first, "", "starting a run")
	if err != nil {
		t.Fatal(err)
	}
	defer one.Claim.Release()
	again := story("AET-1 Charge once")
	again.Workspace = dir
	_, err = claimSite(again, "", "starting a run")
	if err == nil {
		t.Fatal("the same story was isolated onto the branch the first run is about to create; the first run would then fail at checkout")
	}
	if !strings.Contains(err.Error(), "branch") {
		t.Fatalf("the refusal must say which branch is taken: %v", err)
	}
}

func TestABranchThatAlreadyExistsIsNotIsolatedOnto(t *testing.T) {
	dir := gitRepo(t)
	first := story("AET-1 Charge once")
	first.Workspace = dir
	one, err := claimSite(first, "", "starting a run")
	if err != nil {
		t.Fatal(err)
	}
	one.Claim.Relabel("run abc")
	defer one.Claim.Release()
	second := story("AET-2 Refund twice")
	second.Workspace = dir
	probe, err := claimSite(second, "", "starting a run")
	if err != nil {
		t.Fatal(err)
	}
	// Launching the same story again now finds the branch of the first isolated run.
	_, err = claimSite(second, "", "starting a run")
	if err == nil {
		t.Fatalf("a third launch reused the tree of %s and would run a duplicate on one branch", probe.Code.Workspace)
	}
}

func TestAnIdleRunOfTheSameBranchBlocksIsolation(t *testing.T) {
	dir := gitRepo(t)
	first := story("AET-1 Charge once")
	first.Workspace = dir
	one, err := claimSite(first, "", "starting a run")
	if err != nil {
		t.Fatal(err)
	}
	defer one.Claim.Release()
	runDir := filepath.Join(dir, ".span", "runs", "r1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cursor := `{"id":"r1","branch":"` + one.Result.Branch + `"}`
	if err := os.WriteFile(filepath.Join(runDir, "cursor.json"), []byte(cursor), 0o644); err != nil {
		t.Fatal(err)
	}
	one.Claim.Relabel("run r1")
	same := story("AET-1 Charge once")
	same.Workspace = dir
	if _, err := claimSite(same, "", "starting a run"); err == nil {
		t.Fatal("a run already owns this branch")
	}
}

func TestDiscardingAnIsolatedSiteLeavesNoWorktreeAndNoBranch(t *testing.T) {
	dir := gitRepo(t)
	first := story("AET-1 Charge once")
	first.Workspace = dir
	one, err := claimSite(first, "", "starting a run")
	if err != nil {
		t.Fatal(err)
	}
	defer one.Claim.Release()
	second := story("AET-2 Refund twice")
	second.Workspace = dir
	two, err := claimSite(second, "", "starting a run")
	if err != nil {
		t.Fatal(err)
	}
	two.discard()
	if _, err := os.Stat(two.Code.Workspace); !os.IsNotExist(err) {
		t.Fatalf("the worktree of a launch that never started a run is still there: %v", err)
	}
	out, _ := exec.Command("git", "-C", dir, "branch", "--list", two.Result.Branch).Output()
	if strings.TrimSpace(string(out)) != "" {
		t.Fatalf("the branch was left behind: %s", out)
	}
	// The same story can now be launched again.
	if _, err := claimSite(second, "", "starting a run"); err != nil {
		t.Fatalf("relaunch after a discarded launch: %v", err)
	}
}

func TestDiscardingAnInPlaceSiteTouchesNothing(t *testing.T) {
	dir := gitRepo(t)
	req := story("AET-1 Charge once")
	req.Workspace = dir
	st, err := claimSite(req, "", "starting a run")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Claim.Release()
	st.discard()
	if _, err := os.Stat(filepath.Join(dir, "README.md")); err != nil {
		t.Fatalf("the person's checkout was touched: %v", err)
	}
}
