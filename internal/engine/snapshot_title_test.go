package engine

import "testing"

func TestASnapshotNamesWhatTheRunIsAbout(t *testing.T) {
	root := t.TempDir()
	writeRun(t, root, "a", Cursor{Issue: "SAL-9", Stage: "prove", Workspace: "/work/billing-api"},
		"SAL-9 Register a payment\nType: Story\n\nBody")
	snap, err := Load(root, "a")
	if err != nil {
		t.Fatal(err)
	}
	if snap.Title != "Register a payment" {
		t.Fatalf("title = %q", snap.Title)
	}
	if snap.Repo != "billing-api" {
		t.Fatalf("repo = %q", snap.Repo)
	}
}

func TestASnapshotWithoutABriefHasNoTitle(t *testing.T) {
	root := t.TempDir()
	writeRun(t, root, "a", Cursor{Stage: "build"}, "")
	snap, err := Load(root, "a")
	if err != nil {
		t.Fatal(err)
	}
	if snap.Title != "" || snap.Repo != "" {
		t.Fatalf("snapshot = %+v", snap)
	}
}
