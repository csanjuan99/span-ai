package adapt

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"span/internal/engine"
)

// fakeGH puts a gh on PATH that records every argument on its own line and prints what the
// test says. It is how the tests see exactly what Host hands to the CLI.
func fakeGH(t *testing.T, stdout string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell script gh")
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "args.log")
	script := "#!/bin/sh\nfor a in \"$@\"; do printf '%s\\n' \"[$a]\" >> '" + log + "'; done\nprintf '%s' '" + stdout + "'\n"
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

func TestOpenReturnsTheChangeURLWithoutTheNewlineGhPrints(t *testing.T) {
	fakeGH(t, "https://github.com/o/r/pull/11\n")
	url, err := (Host{Workspace: t.TempDir()}).Open(context.Background(), "s", "feat/x", "dev", "t", "b")
	if err != nil {
		t.Fatal(err)
	}
	if url != "https://github.com/o/r/pull/11" {
		t.Fatalf("stored url %q: gh reads a newline in its argument as a branch name", url)
	}
}

func TestMergedAndMergeSendACleanURLEvenWhenTheCursorHoldsOneWithANewline(t *testing.T) {
	log := fakeGH(t, "MERGED\n")
	host := Host{Workspace: t.TempDir()}
	stored := "https://github.com/o/r/pull/11\n"
	merged, err := host.Merged(context.Background(), stored)
	if err != nil || !merged {
		t.Fatalf("merged %v err %v", merged, err)
	}
	if err := host.Merge(context.Background(), stored); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "pull/11\n]") || !strings.Contains(string(raw), "[https://github.com/o/r/pull/11]") {
		t.Fatalf("gh was handed a url with whitespace in it:\n%s", raw)
	}
}

func ghArgs(t *testing.T, log string) []string {
	t.Helper()
	raw, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(raw)), "\n")
}

func TestOpenTargetsTheRunsBaseNotTheDefaultBranch(t *testing.T) {
	log := fakeGH(t, "https://github.com/o/r/pull/12\n")
	if _, err := (Host{Workspace: t.TempDir()}).Open(context.Background(), "s", "feat/SAL-4", "feat/SAL-3", "-t", "- b"); err != nil {
		t.Fatal(err)
	}
	args := ghArgs(t, log)
	// Every value is one argv element with its flag, so a base, a title, or a brief that
	// starts with "-" can never be read as a flag of its own.
	want := []string{"[pr]", "[create]", "[--base=feat/SAL-3]", "[--head=feat/SAL-4]", "[--title=-t]", "[--body=- b]"}
	if strings.Join(args, " ") != strings.Join(want, " ") {
		t.Fatalf("gh %v, want %v", args, want)
	}
}

func TestOpenRefusesABaseGhWouldReadAsAFlag(t *testing.T) {
	log := fakeGH(t, "https://github.com/o/r/pull/12\n")
	for _, base := range []string{"--repo=evil/x", "-x", "HEAD", "", "feat/SAL-4"} {
		_, err := (Host{Workspace: t.TempDir()}).Open(context.Background(), "s", "feat/SAL-4", base, "t", "b")
		if err == nil {
			t.Fatalf("base %q was opened", base)
		}
	}
	if _, err := os.Stat(log); err == nil {
		t.Fatalf("gh was called: %v", ghArgs(t, log))
	}
}

func TestFindReportsEveryOpenChangeAndItsBase(t *testing.T) {
	log := fakeGH(t, `[{"url":"https://github.com/o/r/pull/9","baseRefName":"main"},{"url":"https://github.com/o/r/pull/10","baseRefName":"dev"}]`)
	changes, err := (Host{Workspace: t.TempDir()}).Find(context.Background(), "s", "feat/SAL-4")
	if err != nil {
		t.Fatal(err)
	}
	want := []engine.Change{{URL: "https://github.com/o/r/pull/9", Base: "main"}, {URL: "https://github.com/o/r/pull/10", Base: "dev"}}
	if !reflect.DeepEqual(changes, want) {
		t.Fatalf("changes %+v", changes)
	}
	if got := strings.Join(ghArgs(t, log), " "); got != "[pr] [list] [--head=feat/SAL-4] [--json] [url,baseRefName]" {
		t.Fatalf("gh %s", got)
	}
}

func TestFindWithNoChangeIsNotFound(t *testing.T) {
	fakeGH(t, "[]\n")
	changes, err := (Host{Workspace: t.TempDir()}).Find(context.Background(), "s", "feat/SAL-4")
	if err != nil || len(changes) != 0 {
		t.Fatalf("changes %+v err %v", changes, err)
	}
}

func TestMergedAndMergeRefuseAURLGhWouldReadAsAFlag(t *testing.T) {
	log := fakeGH(t, "MERGED\n")
	host := Host{Workspace: t.TempDir()}
	for _, url := range []string{"-R=evil/x", "--repo=evil/x", "feat/x"} {
		if _, err := host.Merged(context.Background(), url); err == nil {
			t.Errorf("Merged(%q) was accepted", url)
		}
		if err := host.Merge(context.Background(), url); err == nil {
			t.Errorf("Merge(%q) was accepted", url)
		}
	}
	if _, err := os.Stat(log); err == nil {
		t.Fatalf("gh was called: %v", ghArgs(t, log))
	}
}

func TestMergeEndsTheFlagsBeforeTheURL(t *testing.T) {
	log := fakeGH(t, "")
	if err := (Host{Workspace: t.TempDir()}).Merge(context.Background(), "https://github.com/o/r/pull/11"); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(ghArgs(t, log), " "); got != "[pr] [merge] [--merge] [--] [https://github.com/o/r/pull/11]" {
		t.Fatalf("gh %s", got)
	}
}
