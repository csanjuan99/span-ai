package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"span/internal/adapt"
	"span/internal/branch"
	"span/internal/engine"
	"span/internal/live"
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
	// Claim is held on the project checkout. A run that edits it in place keeps it until it
	// stops; a run with its own worktree only needs it until its record exists, so the
	// checkout is free for the next launch. It is nil when the checkout was already busy.
	Claim *runlock.Lock
	// newTree and newBranch say what this launch created, so discard undoes only that.
	newTree, newBranch bool
}

// inPlace is true when the run edits the project checkout itself.
func (s site) inPlace() bool { return s.Code.Workspace == s.Code.Project }

// open records the run of a launch in its tree. A run that edits the checkout keeps the
// claim on it; the other launches release theirs once the run record exists.
func (s site) open(req setup.Request, watch *live.Hub) (engine.Options, string, error) {
	opt, err := options(s.Code.Workspace, s.Code.Spans, s.Code.Policy, s.Code.Project, s.Code.Folder, s.Code.Explicit)
	if err != nil {
		return engine.Options{}, "", err
	}
	fromRequest(&opt, req, s.Result)
	if watch != nil {
		opt.Watch = watch
	}
	if s.inPlace() {
		opt.HeldWorkspace = s.Claim
	}
	id, err := openOwned(opt)
	return opt, id, err
}

// openOwned records a run. The branch is checked again and the run written while no other
// launch of the project can do the same, so one branch never gets two runs.
func openOwned(opt engine.Options) (string, error) {
	unlock, err := engine.LockLaunches(opt.Project)
	if err != nil {
		return "", err
	}
	defer unlock()
	if _, err := os.Stat(opt.Workspace); err != nil {
		// A launch that failed removed the tree it had made while this one was reusing it.
		return "", fmt.Errorf("the tree of this run is gone, try again: %w", err)
	}
	if opt.Branch != "" {
		if err := branchFree(opt.Project, opt.Root, opt.Branch); err != nil {
			return "", err
		}
	}
	return engine.Open(opt)
}

// claimSite decides where a run works, for every launch that configures a project. A run
// with a branch gets its own git worktree, so the person's checkout is never switched and
// stories run side by side, whether or not the others are working at this moment: a run
// that is waiting for an answer still owns its changes. It edits the checkout only when
// asked to (InPlace), when the checkout is already on its branch, or when there is no tree
// to make (see worktree.Separate).
func claimSite(req setup.Request, spans, label string) (site, error) {
	res, claim, err := claimAndConfigure(req, label)
	if err != nil {
		var busy *runlock.BusyError
		if !errors.As(err, &busy) || req.InPlace {
			return site{}, err
		}
		return isolatedSite(req, spans, err)
	}
	st, err := placeSite(req, spans, label, res, claim)
	if err != nil {
		claim.Release()
		return site{}, err
	}
	return st, nil
}

// placeSite picks the tree of a launch that holds the project checkout.
func placeSite(req setup.Request, spans, label string, res setup.Result, claim *runlock.Lock) (site, error) {
	project, err := filepath.Abs(req.Workspace)
	if err != nil {
		return site{}, err
	}
	if res.Branch != "" {
		if err := branchFree(project, spans, res.Branch); err != nil {
			return site{}, err
		}
		claim.Relabel(label + " " + res.Branch)
	}
	separate := !req.InPlace && worktree.Separate(context.Background(), adapt.Git{Dir: project}, project, res.Branch)
	st := site{Result: res, Claim: claim}
	st.Code, st.newTree, st.newBranch, err = prepareOwn(req.Workspace, spans, res.Branch, res.Base, separate)
	if err != nil {
		return site{}, err
	}
	st.Code.Folder = req.Folder
	return st, nil
}

// prepareOwn is prepare, and says what it created. Looking and creating happen under the
// launch lock, so only one of two launches of a branch can believe the tree is its own.
func prepareOwn(workspace, spans, name, base string, separate bool) (code prepared, newTree, newBranch bool, err error) {
	if !separate {
		code, err = prepare(workspace, spans, "", name, base, false)
		return code, false, false, err
	}
	project, err := filepath.Abs(workspace)
	if err != nil {
		return prepared{}, false, false, err
	}
	newTree, newBranch, err = createLocked(project, name, func() error {
		code, err = prepare(workspace, spans, "", name, base, true)
		return err
	})
	return code, newTree, newBranch, err
}

// createLocked says what a launch of the branch is about to create and runs create, both
// while no other launch of the project can do the same.
func createLocked(project, name string, create func() error) (newTree, newBranch bool, err error) {
	unlock, err := engine.LockLaunches(project)
	if err != nil {
		return false, false, err
	}
	defer unlock()
	newTree, newBranch = missing(project, name)
	return newTree, newBranch, create()
}

// missing says whether the worktree and the branch of a launch do not exist yet.
func missing(project, name string) (tree, branchMissing bool) {
	_, err := os.Stat(worktree.Path(project, name))
	return errors.Is(err, os.ErrNotExist), !hasLocalBranch(project, name)
}

func hasLocalBranch(project, name string) bool {
	return worktree.HasBranch(context.Background(), adapt.Git{Dir: project}, project, name)
}

// branchFree refuses a branch that an unfinished run of this project works on: two runs on
// one branch would share one tree and commit each other's files.
func branchFree(project, spans, name string) error {
	for _, cur := range engine.RunsOnBranch(project, stateRoot(project, spans), name) {
		if !cur.Done {
			return fmt.Errorf("branch %s already belongs to another run (%s); resume or delete it", name, cur.ID)
		}
	}
	return nil
}

// stateRoot is where the runs of a project are kept: spans, or the project's default.
func stateRoot(project, spans string) string {
	if spans != "" {
		return spans
	}
	return defaultSpans(project)
}

// placeRunFlags decides where span run works, with the same rule as claimSite: a branch
// gets its own worktree and the note says where. Without a branch there is no name for a
// tree, so the run stays on the checked-out branch, and a busy checkout refuses and says
// how to go on. --in-place and --worktree are left exactly as asked.
func placeRunFlags(f runFlags) (runFlags, string, error) {
	if f.isolate && f.inPlace {
		return f, "", errUsage
	}
	project, err := filepath.Abs(f.workspace)
	if err != nil {
		return f, "", err
	}
	name := strings.TrimSpace(f.branch)
	if name != "" {
		if err := branchFree(project, f.spans, name); err != nil {
			return f, "", err
		}
	}
	if f.inPlace || f.isolate {
		return f, "", nil
	}
	holder, busy := engine.WorkspaceBusy(project)
	if busy && name == "" {
		return f, "", fmt.Errorf("this checkout is busy (%s); pass --branch <name> to run in its own worktree", holder)
	}
	if !busy && !wantsOwnTree(project, f.base, name) {
		return f, "", nil
	}
	f.isolate = true
	note := fmt.Sprintf("running %s in its own worktree at %s", name, worktree.Path(project, name))
	if busy {
		note = fmt.Sprintf("this checkout is busy (%s): %s", holder, note)
	}
	return f, note, nil
}

// wantsOwnTree says whether span run isolates by itself. A name the run would refuse gets
// no tree: the run holds on it and nothing is left behind.
func wantsOwnTree(project, explicitBase, name string) bool {
	base, err := runBase(explicitBase, project, name)
	if err != nil || !branch.Valid(name, base) {
		return false
	}
	return worktree.Separate(context.Background(), adapt.Git{Dir: project}, project, name)
}

// errUsage is a flag combination that cannot be honored.
var errUsage = errors.New("--worktree and --in-place cannot be used together")

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
	policyPath := policyIn(project)
	plan, err := branch.Derive(req.Story, req.Issue, req.Kind, req.Base)
	if err != nil {
		return site{}, err
	}
	if branchTaken(project, spans, plan.Name, busy) {
		return site{}, fmt.Errorf("%w; branch %s already belongs to another run", busy, plan.Name)
	}
	code, newTree, newBranch, err := prepareOwn(project, spans, plan.Name, plan.Base, true)
	if err != nil {
		return site{}, fmt.Errorf("%w; its own worktree could not be made: %v", busy, err)
	}
	code.Folder = req.Folder
	res := setup.Result{PolicyPath: policyPath, Branch: plan.Name, Base: plan.Base, Feature: plan.Feature}
	return site{Result: res, Code: code, newTree: newTree, newBranch: newBranch}, nil
}

// branchTaken says whether a branch already belongs to something that must not be shared
// while another launch holds the checkout: a launch that is about to create it, a run
// that has it, or a branch git already has.
func branchTaken(project, spans, name string, busy error) bool {
	var held *runlock.BusyError
	if errors.As(busy, &held) && strings.Contains(held.Holder, name) {
		// Launching now: its run has no id yet, so only the label it holds names its branch.
		return true
	}
	return len(engine.RunsOnBranch(project, stateRoot(project, spans), name)) > 0 || hasLocalBranch(project, name)
}

// inheritFolder fills the agent and models a repository's policy leaves unset from the
// span.yaml of the folder the launch started from. A folder without a readable span.yaml
// changes nothing: the repository's own policy stands.
func inheritFolder(file policy.File, folder string) policy.File {
	merged, err := inheritFolderChecked(file, folder)
	if err != nil {
		log.Printf("folder policy %s ignored: %v", policyIn(folder), err)
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
	parent, err := policy.Load(policyIn(folder))
	if errors.Is(err, os.ErrNotExist) {
		return file, nil
	}
	if err != nil {
		return file, err
	}
	return file.Inherit(parent), nil
}

// projectPolicy is the policy of the project a run belongs to. span.yaml is not tracked, so
// a run's own worktree does not have it; "" means the default next to the tree.
func projectPolicy(project string) string {
	if project == "" {
		return ""
	}
	return policyIn(project)
}

// policyIn is where the span.yaml of a folder is.
func policyIn(folder string) string {
	return filepath.Join(folder, policy.Name)
}

// defaultSpans is the state directory of a project that names none.
func defaultSpans(project string) string {
	return engine.DefaultSpans(project)
}

// resolvePolicy picks the policy file for a run that is woken again. The project's file
// wins; a run recorded before projects were stored falls back to the checkout, then to the
// project that owns the state directory (<project>/.span/runs). "" keeps the default next to
// the checkout; span.yaml is optional, so a missing one gives the run the quickstart defaults.
func resolvePolicy(spans, workspace, project string) string {
	if project != "" {
		return policyIn(project)
	}
	candidates := []string{policyIn(workspace)}
	if spans != "" {
		candidates = append(candidates, policyIn(filepath.Dir(filepath.Dir(spans))))
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// discard undoes an isolated site whose launch failed before a run existed, so the next
// attempt does not find a tree and a branch nobody owns. Only what this launch created is
// removed. The branch is deleted with -d: it was cut at the base commit, so it carries no
// work, and git refuses if it somehow does. A site edited in place has nothing to undo.
func (s site) discard() {
	if s.inPlace() || s.Code.Workspace == "" {
		return
	}
	unlock, err := engine.LockLaunches(s.Code.Project)
	if err != nil {
		log.Printf("discard %s: %v", s.Code.Workspace, err)
		return
	}
	defer unlock()
	if branchFree(s.Code.Project, s.Code.Spans, s.Result.Branch) != nil {
		// Another launch recorded a run on this branch meanwhile: the tree is its own.
		return
	}
	ctx := context.Background()
	git := adapt.Git{Dir: s.Code.Project}
	if s.newTree {
		if err := worktree.Remove(ctx, git, s.Code.Project, s.Code.Workspace); err != nil {
			log.Printf("discard worktree %s: %v", s.Code.Workspace, err)
		}
		if err := worktree.Prune(ctx, git, s.Code.Project); err != nil {
			log.Printf("prune worktrees of %s: %v", s.Code.Project, err)
		}
	}
	if s.newBranch && s.Result.Branch != "" {
		if _, err := git.Git(ctx, s.Code.Project, "branch", "-d", s.Result.Branch); err != nil {
			log.Printf("delete branch %s: %v", s.Result.Branch, err)
		}
	}
}
