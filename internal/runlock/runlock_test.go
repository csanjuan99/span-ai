package runlock

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestASecondHolderIsRefusedAndNamesTheFirst(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.lock")
	first, err := Acquire(Run, path, "run span-1")
	if err != nil {
		t.Fatal(err)
	}
	defer first.Release()
	_, err = Acquire(Run, path, "run span-2")
	var busy *BusyError
	if !errors.As(err, &busy) {
		t.Fatalf("err = %v, want a BusyError", err)
	}
	if busy.Scope != Run {
		t.Fatalf("scope %q", busy.Scope)
	}
	if !strings.Contains(busy.Holder, "span-1") || !strings.Contains(busy.Holder, "pid") {
		t.Fatalf("holder %q does not name who has the lock", busy.Holder)
	}
}

func TestReleaseLetsTheNextHolderIn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.lock")
	first, err := Acquire(Workspace, path, "run a")
	if err != nil {
		t.Fatal(err)
	}
	first.Release()
	second, err := Acquire(Workspace, path, "run b")
	if err != nil {
		t.Fatalf("lock was not freed: %v", err)
	}
	second.Release()
}

func TestReleaseTwiceIsHarmless(t *testing.T) {
	lock, err := Acquire(Run, filepath.Join(t.TempDir(), "l"), "x")
	if err != nil {
		t.Fatal(err)
	}
	lock.Release()
	lock.Release()
}

// A filesystem without flock (some network mounts) must not make runs undrivable.
func TestAFilesystemWithoutFlockDegradesToNoLock(t *testing.T) {
	old := lockFn
	lockFn = func(*os.File) error { return syscall.ENOLCK }
	t.Cleanup(func() { lockFn = old })
	lock, err := Acquire(Run, filepath.Join(t.TempDir(), "l"), "run x")
	if err != nil {
		t.Fatalf("an unsupported flock refused the run: %v", err)
	}
	lock.Release()
}

func TestDifferentFilesDoNotBlockEachOther(t *testing.T) {
	dir := t.TempDir()
	a, err := Acquire(Workspace, filepath.Join(dir, "a"), "a")
	if err != nil {
		t.Fatal(err)
	}
	defer a.Release()
	b, err := Acquire(Workspace, filepath.Join(dir, "b"), "b")
	if err != nil {
		t.Fatalf("independent workspaces blocked each other: %v", err)
	}
	b.Release()
}

func TestProbeReportsBusyWithoutKeepingTheLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "l")
	if _, busy := Probe(path); busy {
		t.Fatal("a missing lock file reported busy")
	}
	held, err := Acquire(Workspace, path, "run held")
	if err != nil {
		t.Fatal(err)
	}
	holder, busy := Probe(path)
	if !busy || !strings.Contains(holder, "run held") {
		t.Fatalf("probe = %q %v", holder, busy)
	}
	held.Release()
	if _, busy := Probe(path); busy {
		t.Fatal("probe kept the lock or the release did not free it")
	}
}

// The reason to use flock: a process that dies without releasing leaves no stale lock.
func TestALeftoverLockFileIsNotALock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "l")
	if err := os.WriteFile(path, []byte("pid 1 run dead\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lock, err := Acquire(Run, path, "run live")
	if err != nil {
		t.Fatalf("a file left by a dead process blocked a new holder: %v", err)
	}
	lock.Release()
}
