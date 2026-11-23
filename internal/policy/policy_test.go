package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDefaultsExternalStagesToAuto(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "span.yaml")
	body := []byte("feature: payments\nprove:\n  fast:\n    - go test ./...\n")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.OpenMode() != "auto" || got.MergeMode() != "auto" {
		t.Fatalf("modes = %s %s", got.OpenMode(), got.MergeMode())
	}
	if len(got.Prove.Fast) != 1 {
		t.Fatalf("fast = %#v", got.Prove.Fast)
	}
}

func TestLoadRejectsUnknownMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "span.yaml")
	body := []byte("feature: payments\nstages:\n  merge: whenever\n")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected invalid mode")
	}
}

// feature: used to name every run of the project. Runs take their identity from their
// branch now, so the line is ignored: an old file keeps loading, and a file without it is fine.
func TestLoadIgnoresTheFeatureLine(t *testing.T) {
	dir := t.TempDir()
	for _, body := range []string{"feature: hermes\n", "feature: ../payments\n", "agent: any\n", ""} {
		path := filepath.Join(dir, "span.yaml")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err != nil {
			t.Fatalf("%q: %v", body, err)
		}
	}
}

func TestLoadStillRejectsAFileThatIsNotYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "span.yaml")
	if err := os.WriteFile(path, []byte("agent: [any\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected a parse error")
	}
	if _, err := Parse([]byte("stages: [")); err == nil {
		t.Fatal("Parse accepted broken YAML")
	}
}

func TestBaselineIsOnUnlessTurnedOff(t *testing.T) {
	dir := t.TempDir()
	for body, want := range map[string]bool{
		"feature: payments\n":                            true,
		"feature: payments\nprove:\n  baseline: true\n":  true,
		"feature: payments\nprove:\n  baseline: false\n": false,
	} {
		path := filepath.Join(dir, "span.yaml")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if got.BaselineOn() != want {
			t.Fatalf("%q: baseline on = %v, want %v", body, got.BaselineOn(), want)
		}
	}
}

func TestDiffScopeIsOnUnlessTurnedOff(t *testing.T) {
	dir := t.TempDir()
	for body, want := range map[string]bool{
		"feature: payments\n":                        true,
		"feature: payments\nprove:\n  diff: true\n":  true,
		"feature: payments\nprove:\n  diff: false\n": false,
	} {
		path := filepath.Join(dir, "span.yaml")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if got.DiffScopeOn() != want {
			t.Fatalf("%q: diff scope on = %v, want %v", body, got.DiffScopeOn(), want)
		}
	}
}

func TestBaseIsReadAndValidated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "span.yaml")
	if err := os.WriteFile(path, []byte("feature: payments\nbase: feat/SAL-3-wallet\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Base != "feat/SAL-3-wallet" {
		t.Fatalf("base = %q", got.Base)
	}
	base, err := ProjectBase(dir)
	if err != nil || base != "feat/SAL-3-wallet" {
		t.Fatalf("project base = %q, %v", base, err)
	}
	if err := os.WriteFile(path, []byte("feature: payments\nbase: \"a..b\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("an invalid base was accepted")
	}
}

func TestNoBaseKeepsTheDefaults(t *testing.T) {
	dir := t.TempDir()
	base, err := ProjectBase(dir)
	if err != nil || base != "" {
		t.Fatalf("no span.yaml: base = %q, %v", base, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "span.yaml"), []byte("feature: payments\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	base, err = ProjectBase(dir)
	if err != nil || base != "" {
		t.Fatalf("no base: line: base = %q, %v", base, err)
	}
}

func TestABaseThatGitWouldReadAsAnOptionOrACommitIsRefused(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "span.yaml")
	for _, bad := range []string{"\"--upload-pack=touch PWNED1\"", "\"-x\"", "HEAD", "deadbeef", "\" dev\""} {
		if err := os.WriteFile(path, []byte("feature: payments\nbase: "+bad+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := Load(path)
		if err == nil || !strings.Contains(err.Error(), "invalid base branch") {
			t.Errorf("base %s: err = %v", bad, err)
		}
	}
}

func TestSetAgentCreatesAMissingFileWithOnlyThatLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "span.yaml")
	if err := SetAgent(path, "claude"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != "agent: claude\n" {
		t.Fatalf("span.yaml %q %v", raw, err)
	}
	if _, err := Load(path); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(t.TempDir(), "span.yaml")
	if err := SetAgent(missing, "gpt"); err == nil {
		t.Fatal("an unknown agent was written")
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatalf("a refused agent created the file: %v", err)
	}
}
