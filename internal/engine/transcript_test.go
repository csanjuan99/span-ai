package engine

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func nextTranscript(t *testing.T, events <-chan Event) Event {
	t.Helper()
	select {
	case event := <-events:
		return event
	case <-time.After(3 * time.Second):
		t.Fatal("no transcript frame was published")
		return Event{}
	}
}

func fileSize(t *testing.T, path string) int64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Size()
}

func TestTranscriptFramesCarryTheFileOffsetAfterTheirLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "transcript.log")
	if err := os.WriteFile(path, []byte("earlier run\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	events := make(chan Event, 8)
	log, err := OpenTranscript(path, nil, func(event Event) { events <- event })
	if err != nil {
		t.Fatal(err)
	}
	log.SetStage("build")
	if _, err := log.Write([]byte("compiling pkg A\n")); err != nil {
		t.Fatal(err)
	}
	first := nextTranscript(t, events)
	if first.Type != "transcript" || first.Stage != "build" || len(first.Lines) != 1 || first.Lines[0] != "compiling pkg A" {
		t.Fatalf("first %+v", first)
	}
	if first.Start == nil || *first.Start != int64(len("earlier run\n")) {
		t.Fatalf("first start %v", first.Start)
	}
	if first.Offset != fileSize(t, path) {
		t.Fatalf("offset %d, file size %d", first.Offset, fileSize(t, path))
	}
	if _, err := log.Write([]byte("b\nc\n")); err != nil {
		t.Fatal(err)
	}
	second := nextTranscript(t, events)
	if second.Offset <= first.Offset || second.Offset != fileSize(t, path) {
		t.Fatalf("second offset %d after %d, file size %d", second.Offset, first.Offset, fileSize(t, path))
	}
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
}
