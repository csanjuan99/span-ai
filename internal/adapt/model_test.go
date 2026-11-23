package adapt

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeCLI puts a script named binary on PATH that records its arguments, one per line.
func fakeCLI(t *testing.T, binary string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell script CLI")
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "args")
	script := "#!/bin/sh\nfor a in \"$@\"; do printf '%s\\n' \"$a\" >> '" + log + "'; done\ncat >/dev/null\n"
	if err := os.WriteFile(filepath.Join(dir, binary), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

func TestTheChosenModelReachesTheCLIThatRuns(t *testing.T) {
	log := fakeCLI(t, "claude")
	agents := CLIAgents{Models: map[string]string{"claude": "opus", "cursor": "auto"}}
	if err := agents.Build(context.Background(), "claude", t.TempDir(), "do it"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Fields(string(raw))
	want := []string{"-p", "--dangerously-skip-permissions", "--model", "opus"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("args %q, want %q", got, want)
	}
}

func TestACLIWithoutAChosenModelKeepsItsOwn(t *testing.T) {
	log := fakeCLI(t, "claude")
	agents := CLIAgents{Models: map[string]string{"cursor": "auto"}}
	if err := agents.Build(context.Background(), "claude", t.TempDir(), "do it"); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(log)
	if strings.Contains(string(raw), "--model") {
		t.Fatalf("a model was passed to a CLI that has none chosen: %q", raw)
	}
}

func TestTheModelIsLookedUpByCLIIDEvenWhenTheBinaryNameIsGiven(t *testing.T) {
	agents := CLIAgents{Models: map[string]string{"cursor": "gpt-5.3-codex"}}
	if got := agents.model("cursor-agent"); got != "gpt-5.3-codex" {
		t.Fatalf("model %q", got)
	}
	if got := agents.model("unknown"); got != "" {
		t.Fatalf("model %q", got)
	}
}
