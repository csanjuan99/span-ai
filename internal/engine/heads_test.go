package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"span/internal/decision"
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

func TestTheListSaysWhatARunWaitsFor(t *testing.T) {
	root := t.TempDir()
	writeRun(t, root, "a-conflict", Cursor{Stage: "merge", Hold: true, Conflict: &Conflict{Groups: []ConflictGroup{
		{Files: []string{"a.go", "b.go"}}, {Files: []string{"c.go"}},
	}}}, "")
	writeRun(t, root, "b-questions", Cursor{Stage: "shape", Hold: true, Questions: []string{"one", "two", "three"}}, "")
	writeRun(t, root, "c-decision", Cursor{Stage: "plan", Hold: true, Decision: &decision.Request{ID: "d1"}}, "")
	writeRun(t, root, "d-plain", Cursor{Stage: "verify", Stop: true, Reason: "check failed"}, "")
	got := map[string]Head{}
	for _, head := range Heads(root) {
		got[head.ID] = head
	}
	cases := map[string]struct {
		need  string
		count int
	}{
		"a-conflict":  {"conflict", 3},
		"b-questions": {"questions", 3},
		"c-decision":  {"decision", 1},
		"d-plain":     {"", 0},
	}
	for id, want := range cases {
		if got[id].Need != want.need || got[id].Count != want.count {
			t.Errorf("%s: need=%q count=%d, want %q %d", id, got[id].Need, got[id].Count, want.need, want.count)
		}
	}
}

func TestTheListCarriesWhenARunStarted(t *testing.T) {
	root := t.TempDir()
	started := time.Date(2026, 11, 24, 18, 53, 0, 0, time.UTC)
	writeRun(t, root, "a", Cursor{Stage: "build", StartedAt: started}, "")
	writeRun(t, root, "b", Cursor{Stage: "build"}, "")
	heads := Heads(root)
	if heads[0].StartedAt != "2026-11-24T18:53:00Z" {
		t.Fatalf("startedAt = %q", heads[0].StartedAt)
	}
	if heads[1].StartedAt != "" {
		t.Fatalf("a run without a start time has none, got %q", heads[1].StartedAt)
	}
}
