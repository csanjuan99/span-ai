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

func TestConflictModeDefaultsToHold(t *testing.T) {
	for body, want := range map[string]string{
		"feature: payments\n":                                 "hold",
		"feature: payments\nstages:\n  conflict: auto\n":      "auto",
		"feature: payments\nstages:\n  conflict: hold\n":      "hold",
		"feature: payments\nstages:\n  merge: hold\n":         "hold",
		"feature: payments\nstages:\n  conflict: \" auto\"\n": "auto",
	} {
		path := filepath.Join(t.TempDir(), "span.yaml")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatalf("%q: %v", body, err)
		}
		if got.ConflictMode() != want {
			t.Fatalf("%q: conflict mode %q, want %q", body, got.ConflictMode(), want)
		}
		if got.MergeMode() == "" {
			t.Fatalf("%q: merge mode empty", body)
		}
	}
}

func TestConflictModeRejectsUnknownValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "span.yaml")
	if err := os.WriteFile(path, []byte("feature: payments\nstages:\n  conflict: sometimes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "stages.conflict") {
		t.Fatalf("err = %v", err)
	}
}
