package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"span/internal/proc"
	"span/internal/runlock"
	"span/internal/statement"
)

func TestAnotherProcessDrivingTheRunRefusesAResume(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	id, err := Open(opt)
	if err != nil {
		t.Fatal(err)
	}
	// `span resume` from the CLI while serve already drives the run.
	held, err := runlock.Acquire(runlock.Run, filepath.Join(opt.Root, id, "lock"), "run "+id)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Release()
	before, err := readCursorFile(filepath.Join(opt.Root, id))
	if err != nil {
		t.Fatal(err)
	}
	_, err = Resume(context.Background(), opt, id)
	var busy *runlock.BusyError
	if !errors.As(err, &busy) || busy.Scope != runlock.Run {
		t.Fatalf("err = %v, want a run BusyError", err)
	}
	after, err := readCursorFile(filepath.Join(opt.Root, id))
	if err != nil {
		t.Fatal(err)
	}
	if after.Hold != before.Hold || after.Stage != before.Stage || after.Reason != before.Reason {
		t.Fatalf("a refused resume rewrote the cursor of a run someone else drives: %+v -> %+v", before, after)
	}
	if len(agents.prompts) != 0 {
		t.Fatal("the refused resume still ran a stage")
	}
}

func TestASecondRunInTheSameCheckoutHoldsInsteadOfEditingIt(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	id, err := Open(opt)
	if err != nil {
		t.Fatal(err)
	}
	// Another run owns the checkout.
	owner, err := runlock.Acquire(runlock.Workspace, WorkspaceLockPath(opt.Workspace), "run other")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Resume(context.Background(), opt, id)
	if err != nil {
		t.Fatalf("a busy checkout is a hold, not an error: %v", err)
	}
	if got.Verdict != statement.Hold || !strings.Contains(got.Reason, "another run is active") ||
		!strings.Contains(got.Reason, "run other") {
		t.Fatalf("%+v", got)
	}
	if len(agents.prompts) != 0 {
		t.Fatal("the run edited a checkout another run owns")
	}
	cur, err := readCursorFile(filepath.Join(opt.Root, id))
	if err != nil {
		t.Fatal(err)
	}
	if !cur.Hold || cur.Stage != intake {
		t.Fatalf("cursor %+v", cur)
	}

	// Once the owner is done, the same run goes ahead.
	owner.Release()
	got, err = Resume(context.Background(), opt, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Advance {
		t.Fatalf("%+v", got)
	}
}

func TestTheLocksAreFreeWhenTheRunReturns(t *testing.T) {
	opt, _, _, _, _, _ := baseOpt(t)
	if _, err := Start(context.Background(), opt); err != nil {
		t.Fatal(err)
	}
	if holder, busy := WorkspaceBusy(opt.Workspace); busy {
		t.Fatalf("workspace still locked by %q after the run ended", holder)
	}
	ids, err := filepath.Glob(filepath.Join(opt.Root, "*", "lock"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range ids {
		if holder, busy := runlock.Probe(path); busy {
			t.Fatalf("%s still locked by %q", path, holder)
		}
	}
}

func TestARunningRunCannotBeRewound(t *testing.T) {
	shortHalt(t)
	opt, _, _, _, _, _ := baseOpt(t)
	id, err := Open(opt)
	if err != nil {
		t.Fatal(err)
	}
	held, err := runlock.Acquire(runlock.Run, filepath.Join(opt.Root, id, "lock"), "run "+id)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Release()
	var busy *runlock.BusyError
	if err := Rewind(opt.Root, id, build); !errors.As(err, &busy) {
		t.Fatalf("err = %v, want a BusyError", err)
	}
}

func TestWorkspaceBusyNamesTheRunThatHoldsIt(t *testing.T) {
	ws := t.TempDir()
	if _, busy := WorkspaceBusy(ws); busy {
		t.Fatal("a fresh checkout reported busy")
	}
	held, err := runlock.Acquire(runlock.Workspace, WorkspaceLockPath(ws), "run span-7")
	if err != nil {
		t.Fatal(err)
	}
	defer held.Release()
	holder, busy := WorkspaceBusy(ws)
	if !busy || !strings.Contains(holder, "span-7") {
		t.Fatalf("%q %v", holder, busy)
	}
}

func TestRunBusyNamesTheProcessDrivingTheRun(t *testing.T) {
	opt, _, _, _, _, _ := baseOpt(t)
	id, err := Open(opt)
	if err != nil {
		t.Fatal(err)
	}
	if _, busy := RunBusy(opt.Root, id); busy {
		t.Fatal("an idle run reported busy")
	}
	held, err := runlock.Acquire(runlock.Run, runLockPath(opt.Root, id), "run "+id)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Release()
	holder, busy := RunBusy(opt.Root, id)
	if !busy || !strings.Contains(holder, id) {
		t.Fatalf("%q %v", holder, busy)
	}
	if _, busy := RunBusy(opt.Root, "no-such-run"); busy {
		t.Fatal("an unknown run reported busy")
	}
}

func TestClaimWorkspaceRefusesABusyCheckoutAndWritesNothing(t *testing.T) {
	ws := t.TempDir()
	first, err := ClaimWorkspace(ws, "starting a run")
	if err != nil {
		t.Fatal(err)
	}
	defer first.Release()
	_, err = ClaimWorkspace(ws, "starting another")
	var busy *runlock.BusyError
	if !errors.As(err, &busy) || busy.Scope != runlock.Workspace {
		t.Fatalf("err = %v, want a workspace BusyError", err)
	}
}

func TestClaimWorkspaceRefusesAFolderThatIsNotThere(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	if _, err := ClaimWorkspace(missing, "x"); err == nil {
		t.Fatal("a missing folder was claimed")
	}
	if _, err := os.Stat(missing); err == nil {
		t.Fatal("claiming created the folder it was asked about")
	}
}

func TestAHeldCheckoutIsUsedByTheRunInsteadOfBeingTakenAgain(t *testing.T) {
	opt, _, _, _, _, _ := baseOpt(t)
	held, err := ClaimWorkspace(opt.Workspace, "starting a run")
	if err != nil {
		t.Fatal(err)
	}
	opt.HeldWorkspace = held
	id, err := Open(opt)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Resume(context.Background(), opt, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Advance {
		t.Fatalf("the run refused its own claim: %+v", got)
	}
	if holder, busy := WorkspaceBusy(opt.Workspace); busy {
		t.Fatalf("the checkout is still held by %q after the run ended", holder)
	}
}

func TestTheWorkspaceStateIgnoresItselfWhateverTheStateRoot(t *testing.T) {
	opt, _, _, _, _, _ := baseOpt(t)
	// Root is outside the workspace, as with serve's default --spans.
	if _, err := Start(context.Background(), opt); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(opt.Workspace, ".span", ".gitignore"))
	if err != nil {
		t.Fatalf("the workspace .span has no .gitignore, so git status shows the lock as untracked: %v", err)
	}
	if strings.TrimSpace(string(raw)) != "*" {
		t.Fatalf(".gitignore = %q", raw)
	}
}

func TestAStaleBusyReasonIsClearedWhenTheRunIsResumed(t *testing.T) {
	cur := Cursor{Hold: true, Reason: "another run is active in this checkout (pid 1 run a)"}
	clearBusyHold(&cur)
	if cur.Hold || cur.Reason != "" {
		t.Fatalf("%+v", cur)
	}
	kept := Cursor{Hold: true, Reason: "go test ./... exited 1"}
	clearBusyHold(&kept)
	if kept.Reason != "go test ./... exited 1" {
		t.Fatalf("an unrelated reason was cleared: %+v", kept)
	}
}

func TestAPersonsGitignoreInTheWorkspaceStateIsNeverOverwritten(t *testing.T) {
	opt, _, _, _, _, _ := baseOpt(t)
	// The default layout: the state root is <workspace>/.span/runs.
	opt.Root = filepath.Join(opt.Workspace, ".span", "runs")
	if err := os.MkdirAll(filepath.Join(opt.Workspace, ".span"), 0o755); err != nil {
		t.Fatal(err)
	}
	mine := filepath.Join(opt.Workspace, ".span", ".gitignore")
	if err := os.WriteFile(mine, []byte("# mine\nkeep-me\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	claim, err := ClaimWorkspace(opt.Workspace, "starting a run")
	if err != nil {
		t.Fatal(err)
	}
	opt.HeldWorkspace = claim
	if _, err := Start(context.Background(), opt); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(mine)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "# mine\nkeep-me\n" {
		t.Fatalf("the person's .gitignore was overwritten: %q", raw)
	}
}

func TestResumingARunWhoseFolderIsGoneDoesNotRecreateIt(t *testing.T) {
	opt, _, _, _, _, _ := baseOpt(t)
	id, err := Open(opt)
	if err != nil {
		t.Fatal(err)
	}
	gone := filepath.Join(t.TempDir(), "deleted")
	opt.Workspace = gone
	_, r, release, err := acquire(context.Background(), opt, id)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	r.cur.Workspace = gone
	unlock, busy := r.lockWorkspace()
	if busy != nil {
		t.Fatal(busy)
	}
	unlock()
	if _, err := os.Stat(gone); err == nil {
		t.Fatal("locking recreated a folder that was deleted")
	}
}

// A tree Span cannot write a lock file into is not a reason to refuse the work: the run
// itself already carries on unlocked, and the launcher must agree with it.
func TestClaimingAReadOnlyCheckoutDoesNotBlockTheWork(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	ws := t.TempDir()
	if err := os.Chmod(ws, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(ws, 0o755); err != nil {
			t.Error(err)
		}
	})
	lock, err := ClaimWorkspace(ws, "starting a run")
	if err != nil {
		t.Fatalf("a read-only checkout refused the launch: %v", err)
	}
	lock.Release() // a nil claim is valid and releases nothing
	lock.Relabel("run x")
}

// shortHalt keeps a test whose lock holder never ends from waiting the real limit.
func shortHalt(t *testing.T) {
	t.Helper()
	wait, grace := HaltWait, proc.Grace
	HaltWait, proc.Grace = 300*time.Millisecond, 200*time.Millisecond
	t.Cleanup(func() { HaltWait, proc.Grace = wait, grace })
}
