package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"span/internal/fsutil"
	"span/internal/prove"
	"span/internal/qa"
	"span/internal/statement"
)

// baselineFile records what the base commit already failed. It describes the base, not an
// iteration, so it lives next to the cursor and survives rewind and resume.
const baselineFile = "baseline.json"

func (r *runner) baselinePath() string { return filepath.Join(r.spanDir(), baselineFile) }

// recordBaseline runs the wide commands once on the base commit, right before the first
// build. A run that is already past build keeps no baseline: the tree no longer is the
// base, and recording it now would excuse this story's own failures. Nothing here holds
// or fails the run; a command that cannot run is simply not recorded, so nothing it
// prints later is excused. The one error it returns is a cancelled context: the run is
// stopping, and launching the build agent on it would be wasted work.
func (r *runner) recordBaseline(ctx context.Context) error {
	if r.cur.Iteration > 1 || r.cur.GitSHA != "" {
		return nil
	}
	_, built, err := statement.Read(r.iterDir(), build)
	if err != nil {
		r.logLine("baseline: not recorded: the earlier build result cannot be read: " + err.Error())
		return nil
	}
	if built {
		// A build already started on this tree once; it may have left changes behind.
		return nil
	}
	if _, err := os.Stat(r.baselinePath()); !errors.Is(err, os.ErrNotExist) {
		// Already recorded, so the tree was prepared before it.
		if err != nil {
			r.logLine("baseline: not recorded: " + err.Error())
		}
		return nil
	}
	ready, err := r.prepareTree(ctx)
	if err != nil {
		return err
	}
	if !ready || !r.opt.Policy.BaselineOn() {
		return nil
	}
	sha := ""
	if r.opt.Source != nil {
		// Uncommitted changes are not the base commit: their failures would be excused.
		dirty, err := r.opt.Source.Dirty(ctx)
		if err != nil {
			r.logLine("baseline: not recorded: cannot tell whether the checkout is clean: " + err.Error())
			return ctx.Err()
		}
		if dirty {
			r.logLine("baseline: not recorded: the checkout has uncommitted changes")
			return ctx.Err()
		}
		if sha, err = r.opt.Source.Head(ctx); err != nil {
			r.logLine("baseline: not recorded: cannot read HEAD: " + err.Error())
			return ctx.Err()
		}
	}
	base, notes, err := prove.Record(ctx, r.opt.Check, r.opt.Workspace, sha, r.baselineCommands())
	for _, line := range notes {
		r.logLine("baseline: " + line)
	}
	if err != nil {
		r.logLine("baseline: not recorded: " + err.Error())
		return ctx.Err()
	}
	if r.opt.Source != nil {
		now, err := r.opt.Source.Head(ctx)
		if err != nil {
			r.logLine("baseline: not recorded: cannot read HEAD again: " + err.Error())
			return ctx.Err()
		}
		if now != sha {
			r.logLine("baseline: not recorded: HEAD moved from " + sha + " to " + now + " while it ran")
			return nil
		}
	}
	if err := writeJSON(r.baselinePath(), base); err != nil {
		r.logLine("baseline: not recorded: " + err.Error())
	}
	return nil
}

// baselineCommands are the commands prove and qa would run for this story, minus the
// focused band: it is built from the diff, which the base commit does not have yet.
func (r *runner) baselineCommands() []string {
	plan := r.provePlan()
	commands := append(append([]string{}, plan.Fast...), plan.Acceptance...)
	if len(qa.Lines(r.opt.Brief)) > 0 {
		commands = append(commands, r.opt.Policy.QACommand(r.browserQA()))
	}
	return commands
}

// excuse is the prove hook for this run. nil when the baseline is off or was never
// recorded, which keeps every failure blocking exactly as before.
func (r *runner) excuse() prove.Excuse {
	if !r.opt.Policy.BaselineOn() {
		return nil
	}
	raw, err := os.ReadFile(r.baselinePath())
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			r.logLine("baseline: ignored, it cannot be read: " + err.Error())
		}
		return nil
	}
	var base prove.Baseline
	if err := json.Unmarshal(raw, &base); err != nil {
		r.logLine("baseline: ignored, it is not readable JSON: " + err.Error())
		return nil
	}
	return func(command string, out prove.Output) bool {
		ok, why := base.Excuses(command, out)
		if ok {
			r.logLine(fmt.Sprintf("baseline: %s exited %d; %s, not counted", command, out.ExitCode, why))
		}
		return ok
	}
}

func (r *runner) logLine(line string) {
	if r.log == nil {
		return
	}
	if _, err := r.log.Write([]byte(strings.TrimRight(line, "\n") + "\n")); err != nil {
		log.Printf("run %s: transcript line lost: %v: %s", r.cur.ID, err, line)
	}
}

// excusedQA judges a failing QA run whose every failure line was already on the base
// commit. That exit code is not this story's, so the criteria are judged as if it were
// zero: without a per-test report they pass on the exit alone, and with one each
// criterion still decides. ok is false when nothing was excused or a criterion still
// fails; the caller then judges the run exactly as before. The caller never calls this
// when the same command passed on this commit, so an inconclusive result still wins.
func (r *runner) excusedQA(lines []qa.Line, covered map[string]bool, outcomes qa.Outcomes, reportPath string, item statement.Command, out prove.Output) (string, string, bool, error) {
	excuse := r.excuse()
	if excuse == nil || out.Truncated {
		return "", "", false, nil
	}
	gate := qa.Judge(lines, 0, covered, outcomes)
	if gate.Verdict != qa.Pass || !excuse(item.Command, out) {
		return "", "", false, nil
	}
	gate.ExitCode = out.ExitCode
	gate.Command = item.Command
	gate.Report = reportPath
	if err := r.writeGate(gate); err != nil {
		return "", "", false, err
	}
	item.Preexisting = true
	r.last = failure{}
	reason := fmt.Sprintf("acceptance gate passed; %s exited %d with failures already present on the base commit, not counted", item.Command, out.ExitCode)
	verdict, why, err := r.record(qaSt, statement.Advance, reason, func(s *statement.Statement) {
		s.Commands = r.stamp(qaSt, []statement.Command{item})
		s.GitSHA = r.cur.GitSHA
		s.Cap = r.opt.Policy.RepairCap()
	})
	return verdict, why, err == nil, err
}

// prepareTree readies a run's own worktree before its baseline. A fresh worktree has the
// code but not what is installed beside it, so a baseline taken there would record the
// environment as broken and then excuse every failure that follows. It reports whether
// the tree can be trusted for a baseline. The person's own checkout is never prepared:
// it already has its dependencies.
func (r *runner) prepareTree(ctx context.Context) (bool, error) {
	if !r.isolated() {
		return true, nil
	}
	var steps []prepareStep
	for _, command := range r.opt.Policy.Prove.Prepare {
		steps = append(steps, prepareStep{dir: r.opt.Workspace, command: command})
	}
	// The marker outlives a run that was stopped mid-install, whose half-filled folders
	// would otherwise pass for installed ones.
	marker := filepath.Join(r.opt.Workspace, ".span", "preparing")
	if len(steps) == 0 {
		steps = detectPrepare(r.opt.Workspace, present(marker))
	}
	if len(steps) == 0 {
		return true, nil
	}
	ignoreWorkspaceState(r.opt.Workspace)
	if err := os.WriteFile(marker, nil, 0o644); err != nil {
		r.logLine("prepare: an interrupted install will not be noticed: " + err.Error())
	}
	ready := true
	for _, step := range steps {
		out, err := r.opt.Check.Run(ctx, step.command, step.dir)
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		switch {
		case err != nil:
			r.logLine(fmt.Sprintf("baseline: not recorded: prepare %q could not run: %v; set prove.prepare in span.yaml if this project installs another way", step.command, err))
			ready = false
		case out.ExitCode != 0:
			r.logLine(fmt.Sprintf("baseline: not recorded: prepare %q exited %d; set prove.prepare in span.yaml if this project installs another way", step.command, out.ExitCode))
			ready = false
		default:
			r.logLine("prepare: " + step.command + " ok")
		}
	}
	if ready {
		if err := os.Remove(marker); err != nil && !errors.Is(err, os.ErrNotExist) {
			r.logLine("prepare: " + err.Error())
		}
	}
	return ready, nil
}

// isolated is true when the run edits a tree that is not the project it belongs to: its own
// worktree. A person's checkout, even one that is itself a linked worktree, is not.
func (r *runner) isolated() bool {
	return r.opt.Project != "" && !fsutil.SamePath(r.opt.Project, r.opt.Workspace)
}
