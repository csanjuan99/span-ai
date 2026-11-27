// Package runlock keeps two processes from driving the same run, or two runs from
// editing the same checkout, at once. The lock is an advisory flock on a file: the
// kernel drops it when the holder exits, so a crash leaves no stale lock to clean up.
// The text inside the file only says who holds it; the lock itself is the flock.
package runlock

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Scope says what a lock protects.
type Scope string

const (
	// Run protects one run's state and its flight: one driver at a time.
	Run Scope = "run"
	// Workspace protects one checkout: one run edits it at a time.
	Workspace Scope = "workspace"
	// Lessons protects the project's lesson book while one run reads it and writes it back.
	Lessons Scope = "lessons"
	// Journal protects one run's event journal while a line is appended to it.
	Journal Scope = "journal"
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

// Error says what is held and by whom.
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
			closeFile(file)
			return nil, &BusyError{Scope: scope, Path: path, Holder: holder}
		}
		// The filesystem cannot lock (some network mounts return ENOLCK or ENOTSUP). Refusing
		// to run would make every run undrivable there, so run unlocked and let the
		// in-process claim keep serve from driving a run twice.
		log.Printf("running without a lock at %s: %v", path, err)
		return &Lock{file: file}, nil
	}
	writeHolder(file, label)
	return &Lock{file: file}, nil
}

// AcquireWithin is Acquire that retries a busy lock every pause until wait has passed.
func AcquireWithin(scope Scope, path, label string, wait, pause time.Duration) (*Lock, error) {
	deadline := time.Now().Add(wait)
	for {
		lock, err := Acquire(scope, path, label)
		var busy *BusyError
		if err == nil || !errors.As(err, &busy) || time.Now().After(deadline) {
			return lock, err
		}
		time.Sleep(pause)
	}
}

// Relabel changes who the lock says holds it, for a claim taken before the run has an id.
func (l *Lock) Relabel(label string) {
	if l == nil || l.file == nil {
		return
	}
	writeHolder(l.file, label)
}

// Release frees the lock. Releasing twice does nothing.
func (l *Lock) Release() {
	if l == nil || l.file == nil {
		return
	}
	unlockFile(l.file)
	closeFile(l.file)
	l.file = nil
}

// writeHolder replaces the text that says who holds the lock. The text is only for the
// person who finds it busy, so a failure is logged and the lock stays held.
func writeHolder(file *os.File, label string) {
	clearHolder(file)
	if _, err := file.WriteAt([]byte(fmt.Sprintf("pid %d %s\n", os.Getpid(), label)), 0); err != nil {
		log.Printf("name the holder of %s: %v", file.Name(), err)
	}
}

func clearHolder(file *os.File) {
	if err := file.Truncate(0); err != nil {
		log.Printf("clear the holder of %s: %v", file.Name(), err)
	}
}

func closeFile(file *os.File) {
	if err := file.Close(); err != nil {
		log.Printf("close %s: %v", file.Name(), err)
	}
}

// Probe reports whether the lock at path is held now, and by whom. It never keeps it.
func Probe(path string) (string, bool) {
	lock, err := Acquire(Run, path, "probe")
	if err == nil {
		// A probe overwrote the holder text; clear it so a later Probe of a free lock
		// does not show a name.
		clearHolder(lock.file)
		lock.Release()
		return "", false
	}
	var busy *BusyError
	if asBusy(err, &busy) {
		return busy.Holder, true
	}
	log.Printf("the lock at %s could not be asked, taken as free: %v", path, err)
	return "", false
}

func readHolder(file *os.File) string {
	buf := make([]byte, 256)
	n, err := file.ReadAt(buf, 0)
	// A text shorter than the buffer ends in EOF; that is the normal read.
	if err != nil && !errors.Is(err, io.EOF) {
		log.Printf("read the holder of %s: %v", file.Name(), err)
	}
	return strings.TrimSpace(string(buf[:n]))
}

func asBusy(err error, target **BusyError) bool {
	busy, ok := err.(*BusyError)
	if ok {
		*target = busy
	}
	return ok
}
