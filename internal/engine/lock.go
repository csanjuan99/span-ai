package engine

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"span/internal/runlock"
	"span/internal/statement"
)

// Two things must not happen at once. Two processes must not drive the same run (serve
// plus `span resume`, or two serves), and two runs must not edit the same checkout
// (EnsureBranch checks out the run's branch in place and Commit stages everything, so
// they would swap branches under each other and commit each other's files).
//
// The run lock is taken before the cursor is read for writing and held until the entry
// point returns. The workspace lock is taken when work actually starts, so a run that only
// reports its state does not need the checkout.

// WorkspaceLockPath is the lock file that says one run is editing the checkout.
func WorkspaceLockPath(workspace string) string {
	return filepath.Join(workspace, ".span", "workspace.lock")
}

// DefaultSpans is the state directory of a project that names none.
func DefaultSpans(project string) string {
	return filepath.Join(project, ".span", "runs")
}

// WorkspaceBusy reports whether a run is editing the checkout now, and which one.
func WorkspaceBusy(workspace string) (string, bool) {
	if workspace == "" {
		return "", false
	}
	// Probing must not create the directory it asks about.
	if info, err := os.Stat(workspace); err != nil || !info.IsDir() {
		return "", false
	}
	return runlock.Probe(WorkspaceLockPath(workspace))
}

// RunBusy reports whether another process is driving the run now, and which.
func RunBusy(root, id string) (string, bool) {
	if !validID(id) {
		return "", false
	}
	if info, err := os.Stat(filepath.Join(root, id)); err != nil || !info.IsDir() {
		return "", false
	}
	return runlock.Probe(runLockPath(root, id))
}

func runLockPath(root, id string) string { return filepath.Join(root, id, "lock") }

// acquire loads a run and takes its lock. It loads twice: the first read finds the run,
// the second sees what the previous driver left once nobody else can write.
//
// The context it returns is cancelled when a stop marker appears for the run, so a stop
// asked from another process ends the stage in flight like the page's button does.
func acquire(ctx context.Context, opt Options, id string) (context.Context, *runner, func(), error) {
	first, err := load(opt, id)
	if err != nil {
		return ctx, nil, nil, err
	}
	lock, err := runlock.Acquire(runlock.Run, runLockPath(opt.Root, first.cur.ID), "run "+first.cur.ID)
	if err != nil {
		return ctx, nil, nil, err
	}
	r, err := load(opt, id)
	if err != nil {
		lock.Release()
		return ctx, nil, nil, err
	}
	flight, stop := StopOnRequest(ctx, opt.Root, first.cur.ID)
	return flight, r, func() {
		stop()
		lock.Release()
	}, nil
}

// ClaimWorkspace takes the checkout before a run exists, so the launcher can configure the
// project knowing no other run is editing it. Hand the claim to the run in
// Options.HeldWorkspace; Relabel it once the run has an id. A busy checkout is a
// BusyError; a checkout that cannot hold a lock returns a nil claim and no error.
func ClaimWorkspace(workspace, label string) (*runlock.Lock, error) {
	info, err := os.Stat(workspace)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a folder", workspace)
	}
	ignoreWorkspaceState(workspace)
	lock, err := runlock.Acquire(runlock.Workspace, WorkspaceLockPath(workspace), label)
	if err != nil {
		var busy *runlock.BusyError
		if errors.As(err, &busy) {
			return nil, err
		}
		// Span cannot write a lock file here (a read-only tree). lockWorkspace already runs
		// unlocked in that case, and the launcher must not be stricter than the run: a nil
		// claim is valid, and Release and Relabel on it do nothing.
		log.Printf("launch without a checkout lock in %s: %v", workspace, err)
		return nil, nil
	}
	return lock, nil
}

// ignoreWorkspaceState makes <workspace>/.span ignore itself, whatever the state root is.
// The lock lives there, and a project whose .gitignore does not name .span would show it
// as an untracked change. An existing file is left alone: a person may have edited it.
func ignoreWorkspaceState(workspace string) {
	dir := filepath.Join(workspace, ".span")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("create %s: %v", dir, err)
		return
	}
	writeIgnoreOnce(dir)
}

// writeIgnoreOnce writes dir/.gitignore with `*` when there is none. Every writer of that
// file goes through here, so none of them replaces one a person edited.
func writeIgnoreOnce(dir string) {
	file, err := os.OpenFile(filepath.Join(dir, ".gitignore"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if !errors.Is(err, os.ErrExist) {
			log.Printf("write the .gitignore of %s: %v", dir, err)
		}
		return
	}
	if _, err := file.WriteString("*\n"); err != nil {
		log.Printf("write %s: %v", file.Name(), err)
	}
	if err := file.Close(); err != nil {
		log.Printf("close %s: %v", file.Name(), err)
	}
}

// clearBusyHold lifts a hold that only meant "the checkout was taken". Any other reason
// stays until a verdict replaces it, because it still describes why the run stopped.
func clearBusyHold(cur *Cursor) {
	cur.Hold = false
	if runlock.IsWorkspaceBusy(cur.Reason) {
		cur.Reason = ""
	}
}

// lockWorkspace takes the checkout. A busy checkout is returned as a BusyError; any
// other failure to lock (a read-only tree) is not a reason to refuse to run.
func (r *runner) lockWorkspace() (func(), *runlock.BusyError) {
	if held := r.opt.HeldWorkspace; held != nil {
		return held.Release, nil
	}
	workspace := r.cur.Workspace
	if workspace == "" {
		workspace = r.opt.Workspace
	}
	if workspace == "" {
		return func() {}, nil
	}
	// A folder that was deleted is not ours to recreate just to hold a lock.
	if info, err := os.Stat(workspace); err != nil || !info.IsDir() {
		return func() {}, nil
	}
	ignoreWorkspaceState(workspace)
	lock, err := runlock.Acquire(runlock.Workspace, WorkspaceLockPath(workspace), "run "+r.cur.ID)
	if err != nil {
		var busy *runlock.BusyError
		if errors.As(err, &busy) {
			return nil, busy
		}
		log.Printf("run %s without a checkout lock: %v", r.cur.ID, err)
		return func() {}, nil
	}
	return lock.Release, nil
}

// holdBusy parks the run until the checkout is free. It is a hold, not an error: the
// person resumes it from the page once the other run is done.
func (r *runner) holdBusy(busy *runlock.BusyError) Result {
	r.cur.Hold = true
	r.cur.Reason = busy.Error()
	if err := r.save(); err != nil {
		log.Printf("save %s hold: %v", r.cur.ID, err)
	}
	return r.finish(statement.Hold)
}
