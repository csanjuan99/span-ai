package proc

import (
	"os"
	"path/filepath"
	"testing"
)

// A record rewritten in place can be cut in half by a kill, and a half record is dropped by
// Reap, which leaves the agent running. The previous file must stay whole until replaced.
func TestTheAgentRecordIsReplacedNotRewrittenInPlace(t *testing.T) {
	dir := t.TempDir()
	writeNote(dir, note{PID: 101, Name: "first"})
	before, err := os.ReadFile(filepath.Join(dir, pidFile))
	if err != nil {
		t.Fatal(err)
	}
	held := filepath.Join(dir, "held")
	if err := os.Link(filepath.Join(dir, pidFile), held); err != nil {
		t.Skipf("hard links are not available: %v", err)
	}

	writeNote(dir, note{PID: 202, Name: "second"})

	old, err := os.ReadFile(held)
	if err != nil {
		t.Fatal(err)
	}
	if string(old) != string(before) {
		t.Fatalf("the previous record was overwritten: %s", old)
	}
	now, err := os.ReadFile(filepath.Join(dir, pidFile))
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"pid":202,"name":"second"}`; string(now) != want {
		t.Fatalf("record = %s, want %s", now, want)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("a temporary file was left behind: %v", entries)
	}
}
