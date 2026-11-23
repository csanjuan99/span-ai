package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	if _, built, err := statement.Read(r.iterDir(), build); err != nil || built {
		// A build already started on this tree once; it may have left changes behind.
		return nil
	}
	if _, err := os.Stat(r.baselinePath()); !os.IsNotExist(err) {
		// Already recorded, so the tree was prepared before it.
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
		if dirty, err := r.opt.Source.Dirty(ctx); err != nil || dirty {
			r.logLine("baseline: not recorded: the checkout has uncommitted changes")
			return ctx.Err()
		}
		sha, _ = r.opt.Source.Head(ctx)
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
		if now, _ := r.opt.Source.Head(ctx); now != sha {
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
		return nil
	}
	var base prove.Baseline
	if err := json.Unmarshal(raw, &base); err != nil {
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
	_, _ = r.log.Write([]byte(strings.TrimRight(line, "\n") + "\n"))
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
	commands := r.opt.Policy.Prove.Prepare
	for _, command := range commands {
		out, err := r.opt.Check.Run(ctx, command, r.opt.Workspace)
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		if err != nil {
			r.logLine(fmt.Sprintf("baseline: not recorded: prepare %q could not run: %v", command, err))
			return false, nil
		}
		if out.ExitCode != 0 {
			r.logLine(fmt.Sprintf("baseline: not recorded: prepare %q exited %d", command, out.ExitCode))
			return false, nil
		}
		r.logLine("prepare: " + command + " ok")
	}
	if len(commands) == 0 && missingDependencies(r.opt.Workspace) {
		r.logLine("baseline: not recorded: this worktree has a package.json but no node_modules; set prove.prepare in span.yaml to install them")
		return false, nil
	}
	return true, nil
}

// missingDependencies is the one environment gap Span can see without running anything: a
// JavaScript project whose packages were never installed in this tree.
func missingDependencies(workspace string) bool {
	if _, err := os.Stat(filepath.Join(workspace, "package.json")); err != nil {
		return false
	}
	_, err := os.Stat(filepath.Join(workspace, "node_modules"))
	return os.IsNotExist(err)
}

// isolated is true when the run edits a tree that is not the project it belongs to: its own
// worktree. A person's checkout, even one that is itself a linked worktree, is not.
func (r *runner) isolated() bool {
	return r.opt.Project != "" && filepath.Clean(r.opt.Project) != filepath.Clean(r.opt.Workspace)
}
