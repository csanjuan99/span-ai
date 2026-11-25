package start

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
}

func TestPrecheckRefusesANonGitFolder(t *testing.T) {
	dir := t.TempDir()
	ref, err := Precheck(context.Background(), CheckInput{Workspace: dir, Story: "Add a button"})
	if err != nil {
		t.Fatal(err)
	}
	if ref == nil || ref.Number != NumberNotGit {
		t.Fatalf("ref = %+v", ref)
	}
}

func TestWhatWillHappenDefaultsWithoutSpanYaml(t *testing.T) {
	dir := t.TempDir()
	sum := WhatWillHappen(dir, "", "")
	if sum.Folder != dir || sum.Base != "dev" {
		t.Fatalf("summary = %+v", sum)
	}
}

func TestListSealedEmptyWorkspace(t *testing.T) {
	dir := t.TempDir()
	list, err := ListSealed(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("list = %+v", list)
	}
}

func TestPrecheckAllowsMinimalGitRepo(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-b", "dev")
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "go.mod")
	git(t, dir, "commit", "-m", "init")
	ref, err := Precheck(context.Background(), CheckInput{
		Workspace: dir, Story: "Add a health check",
	})
	if err != nil {
		t.Fatal(err)
	}
	if ref != nil && ref.Number == NumberNotGit {
		t.Fatalf("unexpected: %+v", ref)
	}
}

func TestFromErrorMapsBusyCheckout(t *testing.T) {
	ref := FromError(stringsErr("another run is busy in this checkout"))
	if ref == nil || ref.Number != NumberCheckoutBusy {
		t.Fatalf("ref = %+v", ref)
	}
}

type stringsErr string

func (e stringsErr) Error() string { return string(e) }

func TestPrecheckReportsAnErrorInsteadOfNothingBlocks(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-b", "dev")
	git(t, dir, "commit", "--allow-empty", "-m", "init")
	ref, err := Precheck(context.Background(), CheckInput{
		Workspace: dir, Story: "Add a health check", Kind: "not-a-kind",
	})
	if err == nil || ref != nil {
		t.Fatalf("ref = %+v err = %v, want an error and no refusal", ref, err)
	}
}

func TestPrecheckWritesNothingToTheProject(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-b", "dev")
	git(t, dir, "commit", "--allow-empty", "-m", "init")
	if _, err := Precheck(context.Background(), CheckInput{Workspace: dir, Story: "Add a health check"}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"span.yaml", filepath.Join(".span", "brief.md")} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("%s exists after a precheck: %v", name, err)
		}
	}
}
