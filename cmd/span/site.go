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

// isolatedSite gives a run its own worktree because the checkout is busy. Nothing is
// written into the busy checkout: no policy, no brief. That is why it needs a policy that
// is already there and a story to derive a branch from; without either it returns the
// busy error it was given.
func isolatedSite(req setup.Request, spans string, busy error) (site, error) {
	if strings.TrimSpace(req.Story) == "" {
		return site{}, busy
	}
	project, err := filepath.Abs(strings.TrimSpace(req.Workspace))
	if err != nil {
		return site{}, err
	}
	policyPath := filepath.Join(project, "span.yaml")
	if _, err := os.Stat(policyPath); err != nil {
		return site{}, busy
	}
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

// projectPolicy is the policy of the project a run belongs to. span.yaml is not tracked, so
// a run's own worktree does not have it; "" means the default next to the tree.
func projectPolicy(project string) string {
	if project == "" {
		return ""
	}
	return filepath.Join(project, "span.yaml")
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
