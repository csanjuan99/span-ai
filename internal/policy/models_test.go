package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writePolicy(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), Name)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func readPolicy(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestSetModelCreatesAMissingFileWithOnlyThatEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), Name)
	if err := SetModel(path, "claude", "opus"); err != nil {
		t.Fatal(err)
	}
	if got := readPolicy(t, path); got != "models:\n  claude: opus\n" {
		t.Fatalf("%q", got)
	}
	f, err := Load(path)
	if err != nil || f.ModelFor("claude") != "opus" || f.ModelFor("cursor") != "" {
		t.Fatalf("%+v %v", f, err)
	}
}

func TestSetModelKeepsEveryOtherLineAndComment(t *testing.T) {
	path := writePolicy(t, "# policy\nagent: any\nmodels:\n  # per CLI\n  cursor: auto\nstages:\n  merge: hold\n")
	if err := SetModel(path, "claude", "sonnet"); err != nil {
		t.Fatal(err)
	}
	want := "# policy\nagent: any\nmodels:\n  # per CLI\n  cursor: auto\n  claude: sonnet\nstages:\n  merge: hold\n"
	if got := readPolicy(t, path); got != want {
		t.Fatalf("got %q", got)
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v", info.Mode().Perm())
	}
}

func TestSetModelReplacesTheEntryItAlreadyHas(t *testing.T) {
	path := writePolicy(t, "models:\n    cursor: auto\n    pi: xai/grok-4.7\nagent: any\n")
	if err := SetModel(path, "cursor", "gpt-5.3-codex"); err != nil {
		t.Fatal(err)
	}
	if got := readPolicy(t, path); got != "models:\n    cursor: gpt-5.3-codex\n    pi: xai/grok-4.7\nagent: any\n" {
		t.Fatalf("%q", got)
	}
}

func TestAnEmptyModelRemovesTheEntryAndTheEmptyBlock(t *testing.T) {
	path := writePolicy(t, "agent: any\nmodels:\n  cursor: auto\n  pi: xai/grok-4.7\nrules: r.md\n")
	if err := SetModel(path, "cursor", ""); err != nil {
		t.Fatal(err)
	}
	if got := readPolicy(t, path); got != "agent: any\nmodels:\n  pi: xai/grok-4.7\nrules: r.md\n" {
		t.Fatalf("%q", got)
	}
	if err := SetModel(path, "pi", ""); err != nil {
		t.Fatal(err)
	}
	if got := readPolicy(t, path); got != "agent: any\nrules: r.md\n" {
		t.Fatalf("the empty block stayed: %q", got)
	}
	if err := SetModel(path, "pi", ""); err != nil {
		t.Fatalf("removing what is not there is not an error: %v", err)
	}
}

func TestSetModelRefusesWhatCouldNotBeAModelAndLeavesTheFileAlone(t *testing.T) {
	const body = "agent: any\n"
	path := writePolicy(t, body)
	for _, bad := range []string{"--approve", "a b", "x\nstages: {merge: auto}", "a#b", "$(id)"} {
		if err := SetModel(path, "claude", bad); err == nil {
			t.Fatalf("%q was written", bad)
		}
	}
	if err := SetModel(path, "gemini", "x"); err == nil {
		t.Fatal("an unknown CLI was written")
	}
	if got := readPolicy(t, path); got != body {
		t.Fatalf("the file changed: %q", got)
	}
}

func TestSetModelRefusesAnInlineBlockInsteadOfGuessing(t *testing.T) {
	const body = "models: {cursor: auto}\n"
	path := writePolicy(t, body)
	if err := SetModel(path, "claude", "opus"); err == nil || !strings.Contains(err.Error(), "inline") {
		t.Fatalf("%v", err)
	}
	if got := readPolicy(t, path); got != body {
		t.Fatalf("%q", got)
	}
}

func TestAPolicyWithAModelItCannotTrustDoesNotLoad(t *testing.T) {
	for _, body := range []string{
		"models:\n  gemini: x\n",
		"models:\n  claude: --approve\n",
		"models:\n  claude: \"a b\"\n",
		"models:\n  claude: \"\"\n",
	} {
		if _, err := Parse([]byte(body)); err == nil {
			t.Fatalf("%q loaded", body)
		}
	}
	if _, err := Parse([]byte("models:\n  pi: xai/grok-4.7\n")); err != nil {
		t.Fatal(err)
	}
}
