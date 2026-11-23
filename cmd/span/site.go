package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"span/internal/adapt"
	"span/internal/branch"
	"span/internal/engine"
	"span/internal/policy"
	"span/internal/runlock"
	"span/internal/setup"
	"span/internal/worktree"
)

// site is where one run will work and what protects that place.
type site struct {
	// Result carries the branch and base the run uses.
	Result setup.Result
	// Code is the tree the run edits and the project that keeps its state and policy.
	Code prepared
	// Claim is held on the checkout the run edits in place. It is nil when the run has its
	// own worktree: the engine locks that tree itself, so the checkout stays free for
	// the next run.
	Claim *runlock.Lock
}

// claimSite decides where a run works. A free checkout is edited in place, as before: one
// story at a time keeps the person's own checkout as the place to look at the result. A
// checkout another run is editing is never touched, because a second branch checked out
// there would swap the first run's branch and commit its files; the run gets its own
// git worktree instead, so parallel stories run side by side.
func claimSite(req setup.Request, spans, label string) (site, error) {
	res, claim, err := claimAndConfigure(req, label)
	if err == nil {
		code, err := prepare(req.Workspace, spans, "", res.Branch, res.Base, false)
		if err != nil {
			claim.Release()
			return site{}, err
		}
		code.Folder = req.Folder
		if res.Branch != "" {
			claim.Relabel(label + " " + res.Branch)
		}
		return site{Result: res, Code: code, Claim: claim}, nil
	}
	var busy *runlock.BusyError
	if !errors.As(err, &busy) {
		return site{}, err
	}
	return isolatedSite(req, spans, err)
}

// isolateWhenBusy keeps span run from queueing behind another run of the same checkout. A
// busy checkout is never switched from under its run, so a run with a branch gets that
// branch's own worktree, as a second launch from the page does, and the note says where.
// Without a branch there is no name for a tree, so it refuses and says how to go on. A free
// checkout, or an explicit --worktree, is left exactly as asked.
func isolateWhenBusy(f runFlags) (runFlags, string, error) {
	if f.isolate {
		return f, "", nil
	}
	project, err := filepath.Abs(f.workspace)
	if err != nil {
		return f, "", err
	}
	holder, busy := engine.WorkspaceBusy(project)
	if !busy {
		return f, "", nil
	}
	if strings.TrimSpace(f.branch) == "" {
		return f, "", fmt.Errorf("this checkout is busy (%s); pass --branch <name> to run in its own worktree", holder)
	}
	f.isolate = true
	return f, fmt.Sprintf("this checkout is busy (%s): running %s in its own worktree at %s", holder, f.branch, worktree.Path(project, f.branch)), nil
}

// isolatedSite gives a run its own worktree because the checkout is busy. Nothing is
// written into the busy checkout: no policy, no brief. It needs a story to derive a branch
// from; without one it returns the busy error it was given. A missing span.yaml is fine.
func isolatedSite(req setup.Request, spans string, busy error) (site, error) {
	if strings.TrimSpace(req.Story) == "" {
		return site{}, busy
	}
	project, err := filepath.Abs(strings.TrimSpace(req.Workspace))
	if err != nil {
		return site{}, err
	}
	// span.yaml is optional: without one the run uses the quickstart defaults, so a busy
	// checkout that was never configured still gets a worktree, and nothing is written here.
	policyPath := filepath.Join(project, "span.yaml")
	plan, err := branch.Derive(req.Story, req.Issue, req.Kind, req.Base)
	if err != nil {
		return site{}, err
	}
	if taken, _ := branchTaken(project, spans, plan.Name, busy); taken {
		return site{}, fmt.Errorf("%w; branch %s already belongs to another run", busy, plan.Name)
	}
	code, err := prepare(project, spans, "", plan.Name, plan.Base, true)
	if err != nil {
		return site{}, fmt.Errorf("%w; its own worktree could not be made: %v", busy, err)
	}
	code.Folder = req.Folder
	res := setup.Result{PolicyPath: policyPath, Branch: plan.Name, Base: plan.Base, Feature: plan.Feature}
	return site{Result: res, Code: code}, nil
}

// branchTaken says whether a branch already belongs to something that must not be shared:
// a launch that is about to create it, a run that owns it, or a branch git already has.
// The first run in a checkout creates its branch later, at checkout, so git alone cannot
// see it; two runs of one story would then fight over that name.
func branchTaken(project, spans, name string, busy error) (bool, error) {
	var held *runlock.BusyError
	if errors.As(busy, &held) && strings.Contains(held.Holder, name) {
		// Launching now: its run has no id yet, so only the label it holds names its branch.
		return true, nil
	}
	root := spans
	if root == "" {
		root = filepath.Join(project, ".span", "runs")
	}
	entries, _ := os.ReadDir(root)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		snap, err := engine.Load(root, entry.Name())
		if err == nil && snap.Cursor.Branch == name {
			return true, nil
		}
	}
	_, err := adapt.Git{Dir: project}.Git(context.Background(), project, "rev-parse", "--verify", "--quiet", "refs/heads/"+name)
	return err == nil, nil
}

// inheritFolder fills the agent and models a repository's policy leaves unset from the
// span.yaml of the folder the launch started from. A folder without a readable span.yaml
// changes nothing: the repository's own policy stands.
func inheritFolder(file policy.File, folder string) policy.File {
	merged, err := inheritFolderChecked(file, folder)
	if err != nil {
		return file
	}
	return merged
}

// inheritFolderChecked is inheritFolder that says when the folder's span.yaml exists but
// cannot be read, so a caller rereading it mid-run can keep what it had instead of letting a
// half-written file change the agent. A missing file is no error: there is nothing to inherit.
func inheritFolderChecked(file policy.File, folder string) (policy.File, error) {
	if folder == "" {
		return file, nil
	}
	parent, err := policy.Load(filepath.Join(folder, policy.Name))
	if errors.Is(err, os.ErrNotExist) {
		return file, nil
	}
	if err != nil {
		return file, err
	}
	return file.Inherit(parent), nil
}

// launchFolder is the folder to remember for a run in target: the folder the launch started
// from when target is a repository inside it, and "" when the run starts in the repository
// itself (a single repository, or the repository chosen directly).
func launchFolder(folder, target string) string {
	from, err := filepath.Abs(folder)
	if err != nil {
		return ""
	}
	to, err := filepath.Abs(target)
	if err != nil || filepath.Clean(from) == filepath.Clean(to) {
		return ""
	}
	return from
}

// projectPolicy is the policy of the project a run belongs to. span.yaml is not tracked, so
// a run's own worktree does not have it; "" means the default next to the tree.
func projectPolicy(project string) string {
	if project == "" {
		return ""
	}
	return filepath.Join(project, "span.yaml")
}

// resolvePolicy picks the policy file for a run that is woken again. The project's file
// wins; a run recorded before projects were stored falls back to the checkout, then to the
// project that owns the state directory (<project>/.span/runs). "" keeps the default next to
// the checkout; span.yaml is optional, so a missing one gives the run the quickstart defaults.
func resolvePolicy(spans, workspace, project string) string {
	if project != "" {
		return filepath.Join(project, "span.yaml")
	}
	candidates := []string{filepath.Join(workspace, "span.yaml")}
	if spans != "" {
		candidates = append(candidates, filepath.Join(filepath.Dir(filepath.Dir(spans)), "span.yaml"))
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// discard undoes an isolated site whose launch failed before a run existed, so the next
// attempt does not find a tree and a branch nobody owns. The branch is deleted with -d:
// it was created for this launch at the base commit, so it carries no work, and git refuses
// if it somehow does. A site edited in place has nothing to undo.
func (s site) discard() {
	if s.Code.Workspace == s.Code.Project || s.Code.Workspace == "" {
		return
	}
	ctx := context.Background()
	git := adapt.Git{Dir: s.Code.Project}
	_ = worktree.Remove(ctx, git, s.Code.Project, s.Code.Workspace)
	_ = worktree.Prune(ctx, git, s.Code.Project)
	if s.Result.Branch != "" {
		_, _ = git.Git(ctx, s.Code.Project, "branch", "-d", s.Result.Branch)
	}
}
