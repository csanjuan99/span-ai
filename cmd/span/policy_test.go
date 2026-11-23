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

func TestNoPolicyAnywhereLeavesTheDefaultSoTheErrorNamesIt(t *testing.T) {
	root := t.TempDir()
	if got := resolvePolicy(filepath.Join(root, "proj", ".span", "runs"), filepath.Join(root, "tree"), ""); got != "" {
		t.Fatalf("policy %q, want empty", got)
	}
}
