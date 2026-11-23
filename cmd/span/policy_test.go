package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writePolicy(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "span.yaml")
	if err := os.WriteFile(path, []byte("feature: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAResumedRunReadsThePolicyOfItsProject(t *testing.T) {
	root := t.TempDir()
	want := writePolicy(t, filepath.Join(root, "proj"))
	got := resolvePolicy(filepath.Join(root, "proj", ".span", "runs"), filepath.Join(root, "tree"), filepath.Join(root, "proj"))
	if got != want {
		t.Fatalf("policy %q, want %q", got, want)
	}
}

func TestAnOldRunWithNoProjectFallsBackToTheProjectThatOwnsTheState(t *testing.T) {
	root := t.TempDir()
	tree := filepath.Join(root, "tree")
	if err := os.MkdirAll(tree, 0o755); err != nil {
		t.Fatal(err)
	}
	want := writePolicy(t, filepath.Join(root, "proj"))
	got := resolvePolicy(filepath.Join(root, "proj", ".span", "runs"), tree, "")
	if got != want {
		t.Fatalf("a checkout that switched branches no longer has span.yaml; policy %q, want %q", got, want)
	}
}

func TestACheckoutThatStillHasThePolicyKeepsWinningOverTheStateOwner(t *testing.T) {
	root := t.TempDir()
	tree := filepath.Join(root, "tree")
	want := writePolicy(t, tree)
	writePolicy(t, filepath.Join(root, "proj"))
	got := resolvePolicy(filepath.Join(root, "proj", ".span", "runs"), tree, "")
	if got != want {
		t.Fatalf("policy %q, want %q", got, want)
	}
}

func TestNoPolicyAnywhereLeavesTheDefaultPath(t *testing.T) {
	root := t.TempDir()
	if got := resolvePolicy(filepath.Join(root, "proj", ".span", "runs"), filepath.Join(root, "tree"), ""); got != "" {
		t.Fatalf("policy %q, want empty", got)
	}
}

func TestARunWithNoPolicyAnywhereUsesTheDefaults(t *testing.T) {
	root := t.TempDir()
	tree := filepath.Join(root, "tree")
	if err := os.MkdirAll(tree, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tree, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	spans := filepath.Join(root, "proj", ".span", "runs")
	opt, err := options(tree, spans, resolvePolicy(spans, tree, ""), "")
	if err != nil {
		t.Fatalf("a missing span.yaml stopped the run: %v", err)
	}
	if opt.Policy.MergeMode() != "hold" || len(opt.Policy.Prove.Fast) != 1 || opt.Policy.Prove.Fast[0] != "go test ./..." {
		t.Fatalf("policy %+v", opt.Policy)
	}
	if _, err := os.Stat(filepath.Join(tree, "span.yaml")); !os.IsNotExist(err) {
		t.Fatalf("a run wrote span.yaml: %v", err)
	}
}
