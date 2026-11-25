package journal

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestAppendNumbersEventsAndChainsThem(t *testing.T) {
	dir := t.TempDir()
	for _, typ := range []string{"stop", "rewind", "approve"} {
		if err := Append(dir, Person, typ, map[string]string{"stage": "build"}); err != nil {
			t.Fatal(err)
		}
	}
	events, err := Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("%d events", len(events))
	}
	for index, event := range events {
		if event.Seq != index+1 || event.Actor != Person || event.Time.IsZero() {
			t.Fatalf("%d: %+v", index, event)
		}
	}
	if events[0].Prev != "" || events[1].Prev == "" || events[1].Prev == events[2].Prev {
		t.Fatalf("the chain is not linked: %q %q %q", events[0].Prev, events[1].Prev, events[2].Prev)
	}
	if bad, err := Verify(dir); err != nil || bad != 0 {
		t.Fatalf("a good journal failed to verify: bad=%d err=%v", bad, err)
	}
}

func TestVerifyFindsAnEditedEvent(t *testing.T) {
	dir := t.TempDir()
	for _, typ := range []string{"stop", "rewind", "approve"} {
		if err := Append(dir, Person, typ, nil); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(dir, File)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(raw), `"type":"rewind"`, `"type":"resume"`, 1)
	if edited == string(raw) {
		t.Fatal("the test did not edit anything")
	}
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	bad, err := Verify(dir)
	if err != nil || bad != 3 {
		t.Fatalf("the edit was not found at the event that follows it: bad=%d err=%v", bad, err)
	}
}

func TestAppendFromManyGoroutinesKeepsOneSequence(t *testing.T) {
	dir := t.TempDir()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := Append(dir, Span, "stage", nil); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	events, err := Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 20 {
		t.Fatalf("%d events", len(events))
	}
	if bad, err := Verify(dir); err != nil || bad != 0 {
		t.Fatalf("concurrent appends broke the chain: bad=%d err=%v", bad, err)
	}
}

func TestReadOfAMissingJournalIsEmpty(t *testing.T) {
	events, err := Read(t.TempDir())
	if err != nil || len(events) != 0 {
		t.Fatalf("%v %v", events, err)
	}
}

func TestChainNamesTheFirstBrokenLink(t *testing.T) {
	dir := t.TempDir()
	for _, typ := range []string{"started", "sealed", "held"} {
		if err := Append(dir, Span, typ, nil); err != nil {
			t.Fatal(err)
		}
	}
	entries, broken, err := Chain(dir)
	if err != nil || broken != 0 || len(entries) != 3 {
		t.Fatalf("intact chain = %d entries, broken %d, err %v", len(entries), broken, err)
	}
	if entries[1].Prev != entries[0].Hash || entries[1].Want != entries[0].Hash || entries[0].Want != "" {
		t.Fatalf("links = %+v", entries)
	}
	raw, err := os.ReadFile(filepath.Join(dir, File))
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(raw), `"type":"sealed"`, `"type":"sealed!"`, 1)
	if err := os.WriteFile(filepath.Join(dir, File), []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, broken, err = Chain(dir)
	if err != nil || broken != 3 || len(entries) != 3 {
		t.Fatalf("edited chain broke at %d (entries %d, err %v), want 3", broken, len(entries), err)
	}
	if entries[2].Prev == entries[2].Want {
		t.Fatalf("the entry after the edit should not match what is stored: %+v", entries[2])
	}
}
