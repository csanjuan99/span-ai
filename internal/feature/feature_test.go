package feature

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidAcceptsODDDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "feature.md")
	body := "# Payments\n\n## Specs\n\nS1. Charge the card.\n\n## Tasks\n\n- T1. S1.\n\n## Log\n\nL1. Charge the card.\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Valid(path); err != nil {
		t.Fatal(err)
	}
}

func TestValidRejectsMissingLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "feature.md")
	body := "# Payments\n\n## Specs\n\nS1.\n\n## Tasks\n\n- T1.\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Valid(path); err == nil {
		t.Fatal("expected missing log")
	}
}
