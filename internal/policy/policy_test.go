package policy

import (
	"os"
	"path/filepath"
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

func TestLoadRejectsPathFeature(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "span.yaml")
	if err := os.WriteFile(path, []byte("feature: ../payments\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected invalid feature")
	}
}
