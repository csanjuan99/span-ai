package adapt

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"span/internal/prove"
)

// branchRepo is a repository whose dev branch holds base.txt and keep.txt, checked out on
// feat/x one commit later with nothing changed yet.
func branchRepo(t *testing.T) string {
	t.Helper()
	dir := repo(t)
	write(t, dir, "keep.txt", "keep\n")
	commitAll(t, dir, "base.txt", "base\n")
	git(t, dir, "branch", "-M", "dev")
	git(t, dir, "checkout", "-q", "-b", "feat/x")
	return dir
}

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestChangedListsAddedAndModifiedFiles(t *testing.T) {
	dir := branchRepo(t)
	write(t, dir, "base.txt", "changed\n")
	write(t, dir, "pkg/new.go", "package pkg\n")
	commitAll(t, dir, "base.txt", "changed\n")
	got, err := Git{Dir: dir}.Changed(context.Background(), "dev")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"base.txt", "pkg/new.go"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("changed = %q, want %q", got, want)
	}
}

func TestChangedFallsBackToTheRemoteBase(t *testing.T) {
	dir := branchRepo(t)
	git(t, dir, "update-ref", "refs/remotes/origin/dev", git(t, dir, "rev-parse", "dev"))
	git(t, dir, "branch", "-D", "dev")
	commitAll(t, dir, "a.go", "package a\n")
	got, err := Git{Dir: dir}.Changed(context.Background(), "dev")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"a.go"}) {
		t.Fatalf("changed = %q", got)
	}
}

func TestChangedFailsWhenTheBaseIsMissing(t *testing.T) {
	dir := branchRepo(t)
	if _, err := (Git{Dir: dir}).Changed(context.Background(), "release"); err == nil {
		t.Fatal("a missing base gave a diff")
	}
	if _, err := (Git{Dir: dir}).Changed(context.Background(), ""); err == nil {
		t.Fatal("an empty base gave a diff")
	}
}

func TestChangedRefusesADeleteOrARename(t *testing.T) {
	for name, change := range map[string][]string{
		"delete": {"rm", "-q", "keep.txt"},
		"rename": {"mv", "keep.txt", "moved.txt"},
	} {
		t.Run(name, func(t *testing.T) {
			dir := branchRepo(t)
			git(t, dir, change...)
			git(t, dir, "commit", "-q", "-m", name)
			got, err := Git{Dir: dir}.Changed(context.Background(), "dev")
			if !errors.Is(err, ErrBroadChange) {
				t.Fatalf("changed = %q err = %v", got, err)
			}
		})
	}
}

func TestChangedIsEmptyWithoutChanges(t *testing.T) {
	dir := branchRepo(t)
	got, err := Git{Dir: dir}.Changed(context.Background(), "dev")
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("changed = %#v err = %v", got, err)
	}
}

// The derived go test command runs through the real checker and the real go list, so the
// command string has to survive Exec's whitespace split.
func TestADiffFocusFromARealModule(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go is not installed")
	}
	t.Setenv("GOWORK", "off")
	t.Setenv("GOFLAGS", "")
	dir := branchRepo(t)
	write(t, dir, "go.mod", "module example.test/m\n\ngo 1.21\n")
	write(t, dir, "a/a.go", "package a\n\nfunc One() int { return 1 }\n")
	write(t, dir, "b/b.go", "package b\n\nimport \"example.test/m/a\"\n\nfunc Two() int { return a.One() + 1 }\n")
	write(t, dir, "c/c.go", "package c\n")
	write(t, dir, "c/c_test.go", "package c\n\nimport (\n\t\"testing\"\n\n\t\"example.test/m/a\"\n)\n\nfunc TestC(t *testing.T) { _ = a.One() }\n")
	write(t, dir, "d/d.go", "package d\n")
	write(t, dir, "e/e.go", "package e\n")
	commitAll(t, dir, "README.md", "module\n")
	git(t, dir, "branch", "-f", "dev", "HEAD")
	write(t, dir, "a/a.go", "package a\n\nfunc One() int { return 2 - 1 }\n")
	commitAll(t, dir, "NOTES.md", "notes outside every package\n")
	changed, err := Git{Dir: dir}.Changed(context.Background(), "dev")
	if err != nil {
		t.Fatal(err)
	}
	got := prove.DiffFocus(context.Background(), Exec{}, dir, []string{"go test -count=1 ./..."}, changed)
	want := []string{"go test -count=1 example.test/m/a example.test/m/b example.test/m/c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("focus = %q, want %q (changed %q)", got, want, changed)
	}
	out, err := Exec{}.Run(context.Background(), got[0], dir)
	if err != nil || out.ExitCode != 0 {
		t.Fatalf("the derived command failed: %v %d %s", err, out.ExitCode, out.Text)
	}
}
