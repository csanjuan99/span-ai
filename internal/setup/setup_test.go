package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/policy"
)

func TestConfigureDetectsGoAndNamesTheBranch(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Configure(Request{
		Workspace: dir,
		Story:     "AET-12 Charge the card",
		Kind:      "fix",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Wrote || got.Branch != "fix/AET-12-charge-the-card" || got.Base != "dev" {
		t.Fatalf("%+v", got)
	}
	file, err := policy.Load(got.PolicyPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Prove.Fast) != 1 || file.Prove.Fast[0] != "go test ./..." {
		t.Fatalf("checks %+v", file.Prove.Fast)
	}
	if file.MergeMode() != "hold" || file.OpenMode() != "auto" {
		t.Fatalf("modes open %s merge %s", file.OpenMode(), file.MergeMode())
	}
	raw, err := os.ReadFile(got.PolicyPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "clean-code") {
		t.Fatalf("skills missing\n%s", raw)
	}
	brief, err := os.ReadFile(got.BriefPath)
	if err != nil || !strings.Contains(string(brief), "AET-12") {
		t.Fatalf("brief %q %v", brief, err)
	}
}

func TestConfigureDoesNotOverwriteAnExistingPolicy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "span.yaml")
	body := "feature: payments\nagent: any\nstages:\n  open: hold\n  merge: hold\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Configure(Request{Workspace: dir, Story: "Add saved cards"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Wrote {
		t.Fatal("rewrote policy")
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != body {
		t.Fatalf("policy changed %q", raw)
	}
}

func TestConfigureNoLongerWritesAFeatureLine(t *testing.T) {
	dir := t.TempDir()
	got, err := Configure(Request{Workspace: dir, Story: "SAL-7 Pay with the wallet"})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(got.PolicyPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "feature:") {
		t.Fatalf("span.yaml still names a feature:\n%s", raw)
	}
	if got.Feature != "sal-7-pay-with-the-wallet" {
		t.Fatalf("result feature %q", got.Feature)
	}
}

func TestLoadOrDefaultUsesTheQuickstartDefaultsWhenTheFileIsMissing(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "span.yaml")
	got, err := LoadOrDefault(path, dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Agent != "any" || got.Rules != "harness" || got.OpenMode() != "auto" || got.MergeMode() != "hold" {
		t.Fatalf("%+v", got)
	}
	if len(got.Prove.Fast) != 1 || got.Prove.Fast[0] != "go test ./..." {
		t.Fatalf("checks %+v", got.Prove.Fast)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("reading the defaults wrote span.yaml: %v", err)
	}
}

func TestLoadOrDefaultPrefersTheFileThatExists(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "span.yaml")
	if err := os.WriteFile(path, []byte("feature: hermes\nagent: pi\nprove:\n  fast:\n    - make check\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadOrDefault(path, dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Agent != "pi" || len(got.Prove.Fast) != 1 || got.Prove.Fast[0] != "make check" {
		t.Fatalf("%+v", got)
	}
	if err := os.WriteFile(path, []byte("stages:\n  merge: whenever\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOrDefault(path, dir); err == nil {
		t.Fatal("an invalid span.yaml was replaced by the defaults")
	}
}
