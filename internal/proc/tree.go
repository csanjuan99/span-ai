package proc

import (
	"log"
	"os"
	"os/exec"
	"sort"
	"sync"
	"time"
)

// Termination is one process of an agent's tree that Span had to signal.
type Termination struct {
	Stage string
	Agent string
	PID   int
	// Signal is the strongest signal the process was sent.
	Signal string
	// Exit says how it ended: the exit status for the process Span started, else "exited",
	// or "still running" when not even SIGKILL ended it.
	Exit string
}

const stillRunning = "still running"

// entry is one row of the system's process table.
type entry struct {
	pid, ppid, pgid int
	started         string
	command         string
}

// members picks out of rows the processes of a tree: the ones already known, everything in
// the leader's group, and every descendant of those, including a child that moved to a
// group of its own. Span itself is never a member.
func members(rows []entry, group int, known map[int]string) map[int]entry {
	found := map[int]entry{}
	groups := map[int]bool{}
	if group > 1 {
		groups[group] = true
	}
	for grew := true; grew; {
		grew = false
		for _, row := range rows {
			if _, ok := found[row.pid]; ok || row.pid <= 1 || row.pid == os.Getpid() {
				continue
			}
			started, ok := known[row.pid]
			_, child := found[row.ppid]
			if !(ok && (started == "" || started == row.started)) && !groups[row.pgid] && !child {
				continue
			}
			found[row.pid] = row
			if row.pgid == row.pid {
				groups[row.pgid] = true
			}
			grew = true
		}
	}
	return found
}

// finder follows one tree over time: a process seen once stays a member even after its
// parent is gone.
type finder struct {
	mu    sync.Mutex
	group int
	known map[int]string
}

// live returns the members running now.
func (f *finder) live() (map[int]entry, error) {
	rows, err := table()
	if err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	found := members(rows, f.group, f.known)
	for pid, row := range found {
		f.known[pid] = row.started
	}
	return found, nil
}

// listLive is how endTree looks at the tree; a test makes it fail.
var listLive = (*finder).live

// endTree sends SIGTERM to every member, waits Grace, sends SIGKILL to what is left and
// returns once nothing runs or SIGKILL had its chance.
func endTree(f *finder) []Termination {
	sent := map[int]string{}
	alive := map[int]entry{}
	blind := false
	round := func(name string) bool {
		found, err := listLive(f)
		if err != nil {
			// Without a process table the group is all that can be reached: it still gets
			// the grace period, the SIGKILL and a record.
			if !blind {
				log.Printf("list the processes of group %d: %v", f.group, err)
				blind = true
			}
			if !groupAlive(f.group) {
				alive = map[int]entry{}
				return false
			}
			alive = map[int]entry{f.group: {pid: f.group, pgid: f.group}}
			if sent[f.group] != name {
				sent[f.group] = name
				signalGroup(f.group, name)
			}
			return true
		}
		alive = found
		for pid, row := range found {
			if sent[pid] == name {
				continue
			}
			sent[pid] = name
			if f.group > 1 && row.pgid == f.group {
				signalGroup(f.group, name)
			}
			signalPID(pid, name)
		}
		return len(found) > 0
	}
	wait := func(name string, limit time.Duration) {
		deadline := time.Now().Add(limit)
		for round(name) && time.Now().Before(deadline) {
			time.Sleep(50 * time.Millisecond)
		}
	}
	wait("SIGTERM", Grace)
	if len(alive) > 0 {
		wait("SIGKILL", killWait)
	}
	ends := make([]Termination, 0, len(sent))
	for pid, name := range sent {
		exit := "exited"
		if _, ok := alive[pid]; ok {
			exit = stillRunning
		}
		ends = append(ends, Termination{PID: pid, Signal: name, Exit: exit})
	}
	sort.Slice(ends, func(i, j int) bool { return ends[i].PID < ends[j].PID })
	return ends
}

// killWait is how long a process gets to disappear after SIGKILL.
const killWait = 3 * time.Second

// notePoll is how often a running tree is looked at, so a child that later loses its
// parent is still known.
const notePoll = 500 * time.Millisecond

func anyAlive(ends []Termination) bool {
	for _, end := range ends {
		if end.Exit == stillRunning {
			return true
		}
	}
	return false
}

// Tree is an agent command and every process it starts.
type Tree struct {
	cmd          *exec.Cmd
	name, stage  string
	dirs         []string
	find         finder
	quit, polled chan struct{}
	// unlisted is only touched by record, which never runs twice at once.
	unlisted bool

	mu   sync.Mutex
	ends []Termination
}

// Supervise prepares cmd like Tame and makes its cancel end the whole tree. Call it before
// Start, then Started after it and Close after Wait. The agent is recorded in every dir
// while it runs, so Reap can end it if Span dies.
func Supervise(cmd *exec.Cmd, name, stage string, dirs ...string) *Tree {
	Tame(cmd)
	t := &Tree{cmd: cmd, name: name, stage: stage, dirs: dirs, find: finder{known: map[int]string{}}}
	if canList {
		cmd.Cancel = func() error {
			if !t.end() {
				return os.ErrProcessDone
			}
			return nil
		}
	}
	return t
}

// Started records the running command and begins following its tree.
func (t *Tree) Started() {
	pid := t.cmd.Process.Pid
	t.find.mu.Lock()
	t.find.group, t.find.known[pid] = pid, ""
	t.find.mu.Unlock()
	t.quit, t.polled = make(chan struct{}), make(chan struct{})
	if !canList {
		// Nothing can be followed here: the leader alone is recorded, once.
		for _, dir := range t.dirs {
			writeNote(dir, note{PID: pid, Name: t.name, Stage: t.stage})
		}
		close(t.polled)
		return
	}
	t.record()
	go func() {
		defer close(t.polled)
		tick := time.NewTicker(notePoll)
		defer tick.Stop()
		for {
			select {
			case <-t.quit:
				return
			case <-tick.C:
				t.record()
			}
		}
	}()
}

func (t *Tree) record() {
	if _, err := listLive(&t.find); err != nil {
		// The last record stays: it is still the best that is known. A listing that keeps
		// failing is reported once, not twice a second for the whole call.
		if !t.unlisted {
			t.unlisted = true
			log.Printf("record the tree of agent %d: %v", t.cmd.Process.Pid, err)
		}
		return
	}
	left := note{PID: t.cmd.Process.Pid, Name: t.name, Stage: t.stage}
	t.find.mu.Lock()
	for pid, started := range t.find.known {
		left.Members = append(left.Members, member{PID: pid, Started: started})
	}
	t.find.mu.Unlock()
	sort.Slice(left.Members, func(i, j int) bool { return left.Members[i].PID < left.Members[j].PID })
	for _, dir := range t.dirs {
		writeNote(dir, left)
	}
}

// end ends whatever of the tree still runs and reports whether there was anything.
func (t *Tree) end() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	ends := endTree(&t.find)
	t.ends = append(t.ends, ends...)
	return len(ends) > 0
}

// Close ends what the command left behind, even after it finished on its own, clears the
// records and returns every process that had to be signalled. Call it after Wait.
func (t *Tree) Close() []Termination {
	close(t.quit)
	<-t.polled
	if canList {
		t.end()
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	for i := range t.ends {
		t.ends[i].Agent, t.ends[i].Stage = t.name, t.stage
		if t.ends[i].PID == t.cmd.Process.Pid && t.cmd.ProcessState != nil {
			t.ends[i].Exit = t.cmd.ProcessState.String()
		}
	}
	if !anyAlive(t.ends) {
		for _, dir := range t.dirs {
			removeNote(dir)
		}
	}
	return t.ends
}
