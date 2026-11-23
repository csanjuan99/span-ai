package engine

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"span/internal/runlock"
)

// Delete removes one run's record: its journal, cursor, brief and evidence under root.
// Nothing outside that folder is touched: the branch, the pull request and any worktree the
// run made stay where they are, because they may hold work the person still wants.
//
// A run that another process is driving is refused with a BusyError, so a stage in flight is
// never left writing into a folder that no longer exists. The run lock is held while the
// record is emptied, which keeps a resume from starting halfway through.
func Delete(root, id string) error {
	if !validID(id) {
		return fmt.Errorf("invalid span id %q", id)
	}
	dir := filepath.Join(root, id)
	if _, err := os.Stat(filepath.Join(dir, "cursor.json")); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("span %s does not exist", id)
		}
		return err
	}
	held, err := runlock.Acquire(runlock.Run, runLockPath(root, id), "delete "+id)
	if err != nil {
		var busy *runlock.BusyError
		if errors.As(err, &busy) {
			return fmt.Errorf("%w; stop it before deleting it", busy)
		}
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		held.Release()
		return err
	}
	// The lock file stays until the lock is released: some platforms cannot remove an open file.
	for _, entry := range entries {
		if entry.Name() == "lock" {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
			held.Release()
			return err
		}
	}
	held.Release()
	return os.RemoveAll(dir)
}
