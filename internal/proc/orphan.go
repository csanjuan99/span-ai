package proc

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// pidFile is where a running agent is written down: inside the state folder of the tree it
// edits and inside the run that started it. A hard kill (SIGKILL, a crash, a power cut)
// gives Span no chance to stop its agent, and that agent keeps writing into the tree. The
// next run of the tree, or the next server, finds the note and ends it first.
const pidFile = "agent.pid"

type note struct {
	PID   int    `json:"pid"`
	Name  string `json:"name"`
	Stage string `json:"stage,omitempty"`
	// Members are every process of the agent's tree seen so far, the leader included. A
	// child can outlive the leader, so the leader's pid alone does not find it.
	Members []member `json:"members,omitempty"`
}

// member names one process for good: a pid the system reuses has another start time.
type member struct {
	PID     int    `json:"pid"`
	Started string `json:"started"`
}

// writeNote records the agent in dir. It never creates the folder, since the state folder
// of a tree is what keeps the note out of git. A failure is not an error for the run, the
// note is only a safety net.
func writeNote(dir string, left note) {
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return
	}
	raw, err := json.Marshal(left)
	if err != nil {
		log.Printf("encode the record of agent %d: %v", left.PID, err)
		return
	}
	if err := os.WriteFile(filepath.Join(dir, pidFile), raw, 0o644); err != nil {
		log.Printf("record agent %d in %s: %v", left.PID, dir, err)
	}
}

func removeNote(dir string) {
	if err := os.Remove(filepath.Join(dir, pidFile)); err != nil && !os.IsNotExist(err) {
		log.Printf("remove the agent record in %s: %v", dir, err)
	}
}

// Leave records that name runs as pid and returns the func that clears the note, for the
// normal end of the command.
func Leave(dir string, pid int, name string) func() {
	if pid <= 0 {
		return func() {}
	}
	left := note{PID: pid, Name: name}
	rows, err := table()
	if err != nil {
		log.Printf("list the processes of agent %d: %v", pid, err)
	} else {
		for _, row := range members(rows, pid, map[int]string{pid: ""}) {
			left.Members = append(left.Members, member{PID: row.pid, Started: row.started})
		}
	}
	writeNote(dir, left)
	return func() { removeNote(dir) }
}

// Reap ends what a dead Span left running of the agent recorded in dir and reports every
// process it signalled. The caller holds the lock of what dir belongs to, so no live run
// owns that agent. A process is only signalled when it still is the one recorded: a pid
// the system has since given to something else is left alone. The note is cleared once
// nothing of the agent is left. Windows cannot identify the processes and only clears it.
func Reap(dir string) []Termination {
	raw, err := os.ReadFile(filepath.Join(dir, pidFile))
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("read the agent record in %s: %v", dir, err)
		}
		return nil
	}
	var left note
	if err := json.Unmarshal(raw, &left); err != nil || left.PID <= 0 || left.PID == os.Getpid() || left.Name == "" {
		log.Printf("drop the unusable agent record in %s: %v", dir, err)
		removeNote(dir)
		return nil
	}
	rows, err := table()
	if err != nil {
		// The record is the only trace of the orphan, so a failed look keeps it for the
		// next one. Where processes can never be listed it would never be usable.
		log.Printf("list the processes to reap %s: %v", dir, err)
		if !canList {
			removeNote(dir)
		}
		return nil
	}
	known := map[int]string{}
	for _, m := range left.Members {
		known[m.PID] = m.Started
	}
	group := 0
	for _, row := range rows {
		started, recorded := known[row.pid]
		if row.pid == left.PID {
			if recorded && started != row.started {
				delete(known, row.pid)
				continue
			}
			// The recorded leader is alive. Unless it still runs the agent, the pid was
			// never the agent's and nothing of the note can be trusted.
			if !strings.Contains(row.command, left.Name) {
				removeNote(dir)
				return nil
			}
			known[row.pid], group = row.started, left.PID
			continue
		}
		if recorded && started == row.started && row.pgid == left.PID {
			group = left.PID
		}
	}
	ends := endTree(&finder{group: group, known: known})
	for i := range ends {
		ends[i].Agent, ends[i].Stage = filepath.Base(left.Name), left.Stage
	}
	if !anyAlive(ends) {
		removeNote(dir)
	}
	return ends
}
