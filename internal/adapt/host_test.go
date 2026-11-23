package adapt

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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
	url, err := (Host{Workspace: t.TempDir()}).Open(context.Background(), "s", "feat/x", "t", "b")
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
