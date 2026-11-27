package agent

import (
	"context"
	"sync"
	"time"

	"span/internal/proc"
)

type spanKey struct{}

type supervisorKey struct{}

// Supervisor is the run an agent process answers to.
type Supervisor struct {
	// RunDir is where the process tree is recorded while it runs.
	RunDir string
	// Stage is the stage that asked for the agent.
	Stage string
	// Ended is told about every process of the tree that had to be signalled.
	Ended func(proc.Termination)
}

// WithSupervisor marks ctx so the agent it runs is recorded and audited for that run.
func WithSupervisor(ctx context.Context, by Supervisor) context.Context {
	return context.WithValue(ctx, supervisorKey{}, by)
}

func supervisor(ctx context.Context) Supervisor {
	by, _ := ctx.Value(supervisorKey{}).(Supervisor)
	return by
}

// Proc is one CLI agent process tied to a span.
type Proc struct {
	Name  string
	PID   int
	Since time.Time
}

var (
	watchMu sync.Mutex
	watch   = map[string]Proc{}
)

// WithSpan marks ctx so the agent process can be reviewed for that span.
func WithSpan(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, spanKey{}, id)
}

// Enter records the process that is running id.
func Enter(id string, proc Proc) {
	if id == "" || proc.PID <= 0 {
		return
	}
	if proc.Since.IsZero() {
		proc.Since = time.Now()
	}
	watchMu.Lock()
	watch[id] = proc
	watchMu.Unlock()
}

// Leave forgets the process for id.
func Leave(id string) {
	watchMu.Lock()
	delete(watch, id)
	watchMu.Unlock()
}

// Current returns the process still recorded for id.
func Current(id string) (Proc, bool) {
	watchMu.Lock()
	proc, ok := watch[id]
	watchMu.Unlock()
	return proc, ok
}

// Alive reports whether pid is still a process. A permission error still counts.
func Alive(pid int) bool {
	return proc.Alive(pid)
}

func observe(ctx context.Context, name string, pid int) func() {
	id, _ := ctx.Value(spanKey{}).(string)
	if id == "" || pid <= 0 {
		return func() {}
	}
	Enter(id, Proc{Name: name, PID: pid, Since: time.Now()})
	return func() { Leave(id) }
}
