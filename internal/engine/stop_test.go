package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"span/internal/statement"
)

func TestRequestStopWritesAMarkerAndRefusesAnUnknownRun(t *testing.T) {
	root := t.TempDir()
	id := "2026-01-01-0000-stop"
	if err := os.MkdirAll(filepath.Join(root, id), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := RequestStop(root, id); err == nil {
		t.Fatal("a run with no cursor was asked to stop")
	}
	if err := os.WriteFile(filepath.Join(root, id, "cursor.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RequestStop(root, id); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, id, stopFile)); err != nil {
		t.Fatalf("no stop marker: %v", err)
	}
	if err := RequestStop(root, "../escape"); !os.IsNotExist(err) {
		t.Fatalf("an invalid id must be not-found, got %v", err)
	}
}

func TestAStopMarkerCancelsTheFlightAndIsConsumed(t *testing.T) {
	root := t.TempDir()
	id := "2026-01-01-0000-stop"
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cursor.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, release := StopOnRequest(context.Background(), root, id)
	defer release()
	if err := RequestStop(root, id); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ctx.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("the marker did not cancel the flight")
	}
	if _, err := os.Stat(filepath.Join(dir, stopFile)); !os.IsNotExist(err) {
		t.Fatal("the marker was not consumed")
	}
}

func TestAStaleStopMarkerDoesNotCancelANewFlight(t *testing.T) {
	root := t.TempDir()
	id := "2026-01-01-0000-stop"
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cursor.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RequestStop(root, id); err != nil {
		t.Fatal(err)
	}
	ctx, release := StopOnRequest(context.Background(), root, id)
	defer release()
	select {
	case <-ctx.Done():
		t.Fatal("a marker left from before cancelled the new flight")
	case <-time.After(1200 * time.Millisecond):
	}
	if _, err := os.Stat(filepath.Join(dir, stopFile)); !os.IsNotExist(err) {
		t.Fatal("the stale marker survived the start of the flight")
	}
}

func TestAStopWrittenWhileStoppedIsDiscardedOnResume(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	agents.questions = []string{"Which permission atom?"}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if err := RequestStop(opt.Root, got.ID); err != nil {
		t.Fatal(err)
	}
	agents.questions = nil
	again, err := Resume(context.Background(), opt, got.ID)
	if err != nil {
		t.Fatal(err)
	}
	if again.Verdict != statement.Advance {
		t.Fatalf("a stale stop held the resume: %+v", again)
	}
	if _, err := os.Stat(filepath.Join(opt.Root, got.ID, stopFile)); !os.IsNotExist(err) {
		t.Fatal("a stop written while stopped survived the resume")
	}
}
