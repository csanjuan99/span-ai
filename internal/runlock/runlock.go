// Package runlock keeps two processes from driving the same run, or two runs from
// editing the same checkout, at once. The lock is an advisory flock on a file: the
// kernel drops it when the holder exits, so a crash leaves no stale lock to clean up.
// The text inside the file only says who holds it; the lock itself is the flock.
package runlock

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Scope says what a lock protects.
type Scope string

const (
	// Run protects one run's state and its flight: one driver at a time.
	Run Scope = "run"
	// Workspace protects one checkout: one run edits it at a time.
	Workspace Scope = "workspace"
)

// BusyError means another holder has the lock.
type BusyError struct {
	Scope  Scope
	Path   string
	Holder string
}

// workspaceBusy starts the message a run holds with when the checkout is taken. The page and
// Resume recognise it by this prefix, so it lives here with the text it describes.
const workspaceBusy = "another run is active in this checkout"

// IsWorkspaceBusy reports whether a hold reason is the busy-checkout message.
func IsWorkspaceBusy(reason string) bool {
	return strings.HasPrefix(reason, workspaceBusy)
}

func (e *BusyError) Error() string {
	who := e.Holder
	if who == "" {
		who = "another process"
	}
	switch e.Scope {
	case Workspace:
		return fmt.Sprintf("%s (%s)", workspaceBusy, who)
	default:
		return fmt.Sprintf("this run is already being driven (%s)", who)
	}
}

// lockFn is the platform lock; tests replace it to simulate a filesystem without flock.
var lockFn = lockFile

// Lock is a held lock.
type Lock struct {
	file *os.File
}

// Acquire takes the lock at path without waiting. label says who is asking and is
// written into the file for the next person who finds it busy.
func Acquire(scope Scope, path, label string) (*Lock, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := lockFn(file); err != nil {
		if isBusy(err) {
			holder := readHolder(file)
			_ = file.Close()
			return nil, &BusyError{Scope: scope, Path: path, Holder: holder}
		}
		// The filesystem cannot lock (some network mounts return ENOLCK or ENOTSUP). Refusing
		// to run would make every run undrivable there, so run unlocked and let the
		// in-process claim keep serve from driving a run twice.
		return &Lock{file: file}, nil
	}
	_ = file.Truncate(0)
	_, _ = file.WriteAt([]byte(fmt.Sprintf("pid %d %s\n", os.Getpid(), label)), 0)
	return &Lock{file: file}, nil
}

// Relabel changes who the lock says holds it, for a claim taken before the run has an id.
func (l *Lock) Relabel(label string) {
	if l == nil || l.file == nil {
		return
	}
	_ = l.file.Truncate(0)
	_, _ = l.file.WriteAt([]byte(fmt.Sprintf("pid %d %s\n", os.Getpid(), label)), 0)
}

// Release frees the lock. Releasing twice does nothing.
func (l *Lock) Release() {
	if l == nil || l.file == nil {
		return
	}
	unlockFile(l.file)
	_ = l.file.Close()
	l.file = nil
}

// Probe reports whether the lock at path is held now, and by whom. It never keeps it.
func Probe(path string) (string, bool) {
	lock, err := Acquire(Run, path, "probe")
	if err == nil {
		// A probe overwrote the holder text; clear it so a later Probe of a free lock
		// does not show a name.
		_ = lock.file.Truncate(0)
		lock.Release()
		return "", false
	}
	var busy *BusyError
	if asBusy(err, &busy) {
		return busy.Holder, true
	}
	return "", false
}

func readHolder(file *os.File) string {
	buf := make([]byte, 256)
	n, _ := file.ReadAt(buf, 0)
	return strings.TrimSpace(string(buf[:n]))
}

func asBusy(err error, target **BusyError) bool {
	busy, ok := err.(*BusyError)
	if ok {
		*target = busy
	}
	return ok
}
