package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/engine"
	"span/internal/jira"
)

// The Jira login lives in the project, not in the run's worktree. A login that cannot be
// parsed shows which folder was read, without calling Jira.
func TestReadingTheStoryAgainUsesTheProjectLogin(t *testing.T) {
	project := t.TempDir()
	path, err := jira.Path(project)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	spans := opsRun(t, engine.Cursor{Issue: "AET-1", Project: project, Workspace: t.TempDir()})
	_, err = readStory(context.Background(), spans, "run-1")
	if err == nil || !strings.Contains(err.Error(), "cannot be read") {
		t.Fatalf("the project's login was not read: %v", err)
	}
}
