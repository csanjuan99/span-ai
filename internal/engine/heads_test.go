package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeRun(t *testing.T, root, id string, cur Cursor, brief string) {
	t.Helper()
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cur.ID = id
	raw, err := json.Marshal(cur)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cursor.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if brief != "" {
		if err := os.WriteFile(filepath.Join(dir, "brief.md"), []byte(brief), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTheListNamesWhatTheRunIsAbout(t *testing.T) {
	root := t.TempDir()
	writeRun(t, root, "a", Cursor{Issue: "SAL-9", Stage: "prove", Hold: true, Reason: "prove exited 1", Branch: "feat/SAL-9-payments"},
		"SAL-9 Register a payment\nType: Story\n\nBody")
	head := Heads(root)[0]
	if head.Title != "Register a payment" {
		t.Fatalf("title = %q", head.Title)
	}
	if head.Branch != "feat/SAL-9-payments" || head.Reason != "prove exited 1" {
		t.Fatalf("head = %+v", head)
	}
}

func TestTheListTitleNeedsNoIssueKey(t *testing.T) {
	root := t.TempDir()
	writeRun(t, root, "a", Cursor{Stage: "build"}, "Add a /healthz endpoint\n\nReturns the version.")
	if got := Heads(root)[0].Title; got != "Add a /healthz endpoint" {
		t.Fatalf("title = %q", got)
	}
}

func TestTheListSurvivesARunWithoutABrief(t *testing.T) {
	root := t.TempDir()
	writeRun(t, root, "a", Cursor{Stage: "build"}, "")
	if got := Heads(root)[0].Title; got != "" {
		t.Fatalf("title = %q", got)
	}
}

func TestTheListSaysWhenTheRunLastMoved(t *testing.T) {
	root := t.TempDir()
	writeRun(t, root, "a", Cursor{Stage: "build"}, "")
	when := time.Date(2026, 11, 24, 14, 0, 0, 0, time.UTC)
	if err := os.Chtimes(filepath.Join(root, "a", "cursor.json"), when, when); err != nil {
		t.Fatal(err)
	}
	if got := Heads(root)[0].UpdatedAt; got != "2026-11-24T14:00:00Z" {
		t.Fatalf("updatedAt = %q", got)
	}
}
