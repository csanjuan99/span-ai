package engine

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"span/internal/fsutil"
	"span/internal/runlock"
)

// Runs of one project can be kept under several state roots (--spans, a server started
// elsewhere), so the project itself records where each run with a branch lives.

func ownersDir(project string) string { return filepath.Join(project, ".span", "owners") }

const launchWait = 2 * time.Minute

// LockLaunches makes the caller the only one launching or removing runs of the project
// until it calls the returned function. A project that cannot hold a lock is not locked,
// as with the checkout lock.
func LockLaunches(project string) (func(), error) {
	info, err := os.Stat(project)
	if err != nil || !info.IsDir() {
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Printf("launch without a lock in %s: %v", project, err)
		}
		return func() {}, nil
	}
	ignoreWorkspaceState(project)
	path := filepath.Join(project, ".span", "launch.lock")
	lock, err := runlock.AcquireWithin(runlock.Workspace, path, "launch", launchWait, 20*time.Millisecond)
	if err != nil {
		var busy *runlock.BusyError
		if errors.As(err, &busy) {
			return nil, errors.New("another launch of this project did not finish; try again")
		}
		log.Printf("launch without a lock in %s: %v", project, err)
		return func() {}, nil
	}
	return lock.Release, nil
}

// recordOwner notes in the project that the run under root works on a branch of it.
func recordOwner(project, root, id string) {
	abs, err := filepath.Abs(root)
	if err == nil {
		ignoreWorkspaceState(project)
		err = os.MkdirAll(ownersDir(project), 0o755)
	}
	if err == nil {
		err = os.WriteFile(filepath.Join(ownersDir(project), id), []byte(abs+"\n"), 0o644)
	}
	if err != nil {
		log.Printf("record the branch of run %s in %s: %v", id, project, err)
	}
}

// RunsOnBranch lists the runs of the project that work on the branch: those under root and
// those the project recorded under any other state root.
func RunsOnBranch(project, root, name string) []Cursor {
	dirs := map[string]string{}
	add := func(root, id string) {
		if validID(id) {
			dirs[id] = filepath.Join(root, id)
		}
	}
	for _, entry := range listDir(root) {
		add(root, entry.Name())
	}
	for _, entry := range listDir(ownersDir(project)) {
		raw, err := os.ReadFile(filepath.Join(ownersDir(project), entry.Name()))
		if err != nil {
			log.Printf("read the record of run %s: %v", entry.Name(), err)
			continue
		}
		add(strings.TrimSpace(string(raw)), entry.Name())
	}
	var out []Cursor
	for _, dir := range dirs {
		cur, err := readCursorFile(dir)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				log.Printf("read run %s: %v", dir, err)
			}
			continue
		}
		if cur.Branch == name && (cur.Project == "" || fsutil.SamePath(cur.Project, project)) {
			out = append(out, cur)
		}
	}
	return out
}

// listDir is the entries of dir; a missing one is empty.
func listDir(dir string) []os.DirEntry {
	entries, err := os.ReadDir(dir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("list %s: %v", dir, err)
	}
	return entries
}
