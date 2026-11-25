package engine

import (
	"os"
	"path/filepath"
	"testing"
)

// writeBareCursor makes a run folder with a cursor in the given state.
func writeBareCursor(t *testing.T, root, id, body string) {
	t.Helper()
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cursor.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotSaysAPauseIsPendingUntilTheRunHolds(t *testing.T) {
	root := t.TempDir()
	writeBareCursor(t, root, "r1", `{"id":"r1","stage":"implement"}`)

	snap, err := Load(root, "r1")
	if err != nil {
		t.Fatal(err)
	}
	if snap.PauseRequested {
		t.Fatal("a run nobody asked to pause reports a pending pause")
	}

	if err := RequestPause(root, "r1"); err != nil {
		t.Fatal(err)
	}
	snap, err = Load(root, "r1")
	if err != nil {
		t.Fatal(err)
	}
	if !snap.PauseRequested {
		t.Fatal("a requested pause is not reported")
	}

	// The same request on a run that already holds is not pending: nothing is left to wait for.
	writeBareCursor(t, root, "r1", `{"id":"r1","stage":"implement","hold":true}`)
	snap, err = Load(root, "r1")
	if err != nil {
		t.Fatal(err)
	}
	if snap.PauseRequested {
		t.Fatal("a held run reports a pending pause")
	}
}
