package proc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
)

func itoa(n int) string { return strconv.Itoa(n) }

// pidFile is where a running agent is written down, inside the state folder of the tree it
// edits. A hard kill (SIGKILL, a crash, a power cut) gives Span no chance to stop its agent,
// and that agent keeps writing into the tree. The next run of the tree finds the note and
// ends it first.
const pidFile = "agent.pid"

type note struct {
	PID  int    `json:"pid"`
	Name string `json:"name"`
}

// Leave records that name runs as pid and returns the func that clears the note, for the
// normal end of the command. The folder must already exist: Leave never creates the state
// folder of a tree, since that folder is what keeps it out of git. A failure is not an
// error for the run, the note is only a safety net.
func Leave(dir string, pid int, name string) func() {
	path := filepath.Join(dir, pidFile)
	if pid <= 0 {
		return func() {}
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return func() {}
	}
	raw, err := json.Marshal(note{PID: pid, Name: name})
	if err != nil || os.WriteFile(path, raw, 0o644) != nil {
		return func() {}
	}
	return func() { _ = os.Remove(path) }
}

// Reap ends an agent a dead Span left running in this tree and reports whether it did. The
// caller holds the tree's lock, so no live run of this tree owns that agent. A process is
// only killed when its command line still names the recorded agent: a pid the system has
// since given to something else is left alone. The note is cleared either way.
func Reap(dir string) bool {
	path := filepath.Join(dir, pidFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	_ = os.Remove(path)
	var left note
	if json.Unmarshal(raw, &left) != nil || left.PID <= 0 || left.PID == os.Getpid() || left.Name == "" {
		return false
	}
	if !Alive(left.PID) || !runs(left.PID, left.Name) {
		return false
	}
	kill(left.PID)
	return true
}
