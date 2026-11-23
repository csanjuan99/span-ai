package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"span/internal/runlock"
)

// Rewind restarts a run at a stage. A person decided the work so far is not what the
// handoff asked for, so the run walks again from that stage: a new iteration keeps the
// earlier attempts on disk as history, and the repair budget starts fresh because
// starting over deliberately is not the automatic loop the cap exists to stop.
func Rewind(root, id, stage string) error {
	if !validID(id) {
		return os.ErrNotExist
	}
	if stage == intake || !known(stage) {
		return fmt.Errorf("stage %q cannot be restarted; use one of %v", stage, order[1:])
	}
	dir := filepath.Join(root, id)
	if _, err := os.Stat(dir); err != nil {
		return err
	}
	// A run another process is driving would overwrite this rewrite on its next save.
	lock, err := runlock.Acquire(runlock.Run, runLockPath(root, id), "rewind "+id)
	if err != nil {
		return err
	}
	defer lock.Release()
	cur, err := readCursorFile(dir)
	if err != nil {
		return err
	}
	cur.Stage = stage
	cur.Iteration++
	cur.Hold = false
	cur.Stop = false
	cur.Done = false
	cur.Questions = nil
	cur.Repairs = nil
	cur.WideSHA = ""
	cur.AgentBlock = nil
	cur.InspectBlocks = 0
	cur.Reason = "restarted at " + stage + " by a person"
	return writeCursorFile(dir, cur)
}

func known(stage string) bool {
	for _, name := range order {
		if name == stage {
			return true
		}
	}
	return false
}

func readCursorFile(dir string) (Cursor, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "cursor.json"))
	if err != nil {
		return Cursor{}, err
	}
	var cur Cursor
	if err := json.Unmarshal(raw, &cur); err != nil {
		return Cursor{}, err
	}
	return cur, nil
}

func writeCursorFile(dir string, cur Cursor) error {
	raw, err := json.MarshalIndent(cur, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(dir, "cursor.json"), append(raw, '\n'))
}
