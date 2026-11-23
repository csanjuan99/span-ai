package runlock

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRelabelChangesWhatTheNextPersonSees(t *testing.T) {
	path := filepath.Join(t.TempDir(), "l")
	lock, err := Acquire(Workspace, path, "starting a run")
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	lock.Relabel("run span-9")
	holder, busy := Probe(path)
	if !busy || !strings.Contains(holder, "run span-9") || strings.Contains(holder, "starting") {
		t.Fatalf("holder %q busy %v", holder, busy)
	}
}

func TestTheBusyCheckoutMessageIsRecognisedAsStale(t *testing.T) {
	msg := (&BusyError{Scope: Workspace, Holder: "pid 1 run a"}).Error()
	if !IsWorkspaceBusy(msg) {
		t.Fatalf("%q is not recognised", msg)
	}
	if IsWorkspaceBusy("go test ./... exited 1") || IsWorkspaceBusy("") {
		t.Fatal("an unrelated reason was taken for the busy message")
	}
}
