package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"span/internal/fsutil"
	"span/internal/runlock"
)

// Rewind restarts a run at a stage. A person decided the work so far is not what the
// handoff asked for, so the run walks again from that stage: a new iteration keeps the
// earlier attempts on disk as history, and the repair budget starts fresh because
// starting over deliberately is not the automatic loop the cap exists to stop.
func Rewind(root, id, stage string) error {
	// Refused requests must not stop anything, so they are judged before the halt.
	if err := CanRewind(root, id, stage); err != nil {
		return err
	}
	// A run being driven would overwrite this rewrite on its next save, and its agent would
	// keep writing, so it is stopped first.
	if err := Halt(root, id); err != nil {
		return err
	}
	dir := filepath.Join(root, id)
	lock, err := runlock.Acquire(runlock.Run, runLockPath(root, id), "rewind "+id)
	if err != nil {
		return err
	}
	defer lock.Release()
	// Read again: the stopped driver saved the cursor on its way out.
	cur, err := rewindable(dir)
	if err != nil {
		return err
	}
	applyRewind(&cur, stage)
	return writeCursorFile(dir, cur)
}

// CanRewind reports why a run cannot be restarted at stage, without touching it.
func CanRewind(root, id, stage string) error {
	if !validID(id) {
		return os.ErrNotExist
	}
	if stage == intake || !known(stage) {
		return fmt.Errorf("stage %q cannot be restarted; use one of %v", stage, order[1:])
	}
	_, err := rewindable(filepath.Join(root, id))
	return err
}

func rewindable(dir string) (Cursor, error) {
	cur, err := readCursorFile(dir)
	if err != nil {
		return Cursor{}, err
	}
	if cur.Conflict != nil {
		// The checkout is in the middle of a merge; another stage would commit it half done.
		return Cursor{}, errors.New("this run is resolving a merge conflict; abort the conflict before restarting from a stage")
	}
	return cur, nil
}

func stageIndex(stage string) int {
	for index, name := range order {
		if name == stage {
			return index
		}
	}
	return -1
}

func known(stage string) bool { return stageIndex(stage) >= 0 }

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
	return fsutil.WriteAtomic(filepath.Join(dir, "cursor.json"), append(raw, '\n'))
}

// applyRewind points a cursor at stage for a fresh attempt: the holds, the questions, the
// repair ledger and the blocker belong to the attempt being left behind.
func applyRewind(cur *Cursor, stage string) {
	cur.Stage = stage
	cur.Iteration++
	cur.Hold = false
	cur.Stop = false
	cur.Done = false
	cur.Questions = nil
	cur.Repairs = nil
	cur.WideSHA = ""
	cur.AgentBlock = nil
	cur.Blocker = nil
	cur.Decision = nil
	cur.InspectBlocks = 0
	cur.InspectFeedback = ""
	cur.Rewound = true
	// The change belongs to the attempt that opened it. A restart before open will open the
	// change again, and open reuses a live one on the branch, so the old link only misleads.
	if stageIndex(stage) < stageIndex(open) {
		cur.ChangeURL = ""
		cur.Pending = nil
	}
	cur.Reason = "restarted at " + stage + " by a person"
}
