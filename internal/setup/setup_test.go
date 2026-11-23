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
