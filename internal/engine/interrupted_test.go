package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"span/internal/runlock"
)

// interruptedRun writes a run that was mid-stage when its process ended: the cursor says it
// is running, and its age says nobody has touched it for a while.
func interruptedRun(t *testing.T, age time.Duration) (root, id string) {
	t.Helper()
	root, id = t.TempDir(), "2026-11-23-1200-abcd"
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(Cursor{ID: id, Stage: "build", Iteration: 1})
	path := filepath.Join(dir, "cursor.json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-age)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	return root, id
}

func TestARunWhoseProcessDiedIsShownAsPausedWithTheReason(t *testing.T) {
	root, id := interruptedRun(t, time.Minute)
	snap, err := Load(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if !snap.Cursor.Hold || !strings.Contains(snap.Cursor.Reason, "interrupted") {
		t.Fatalf("a dead run still looks like it is running: %+v", snap.Cursor)
	}
}

func TestARunSomeoneIsDrivingIsNotCalledInterrupted(t *testing.T) {
	root, id := interruptedRun(t, time.Minute)
	lock, err := runlock.Acquire(runlock.Run, runLockPath(root, id), "run "+id)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	snap, err := Load(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Cursor.Hold {
		t.Fatalf("a run with a live driver was called interrupted: %+v", snap.Cursor)
	}
}

func TestARunThatJustStartedIsNotCalledInterruptedBeforeItTakesItsLock(t *testing.T) {
	root, id := interruptedRun(t, 0)
	snap, err := Load(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Cursor.Hold {
		t.Fatalf("a run was called interrupted the moment it was created: %+v", snap.Cursor)
	}
}
