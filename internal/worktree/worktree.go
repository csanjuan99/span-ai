// Package worktree keeps parallel runs apart. Every run that walks a branch does it in
// its own git worktree, so two stories in one repository never share a working tree and
// the checkout the person is using is never switched.
//
// The commands here are the whole surface Span uses. Nothing is inferred:
//
//	git worktree add <path> <branch>          the branch already exists
//	git worktree add -b <branch> <path> <base>  the branch is new
//	git worktree list --porcelain             what is checked out where
//	git worktree remove --force <path>        drop an isolated tree
//	git worktree prune                        forget trees that are gone
package worktree

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// Runner runs one git command in a repository. adapt.Git implements it.
type Runner interface {
	Git(ctx context.Context, dir string, args ...string) (string, error)
}

// Entry is one worktree git knows about.
type Entry struct {
	Path   string
	Branch string
	Bare   bool
	Head   string
}

// Add checks out branch at path. A branch that does not exist yet is created from base;
// one that does is reused, so resuming a run lands on the same tree instead of a second
// copy of the same branch.
func Add(ctx context.Context, run Runner, repo, path, branch, base string) error {
	if strings.TrimSpace(branch) == "" {
		return fmt.Errorf("worktree needs a branch")
	}
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("worktree needs a path")
	}
	entries, err := List(ctx, run, repo)
	if err != nil {
		return err
	}
	if entry, ok := At(entries, path); ok {
		if entry.Branch == branch {
			// Already checked out for this run: resume lands on the same tree.
			return nil
		}
		return fmt.Errorf("%s already holds branch %s, not %s", path, entry.Branch, branch)
	}
	exists, err := hasBranch(ctx, run, repo, branch)
	if err != nil {
		return err
	}
	args := []string{"worktree", "add"}
	// The last argument is what gets checked out: the branch when it already exists, the
	// base commit when the branch is being created.
	start := branch
	if !exists {
		if strings.TrimSpace(base) == "" {
			return fmt.Errorf("branch %s does not exist and no base was given", branch)
		}
		args = append(args, "-b", branch)
		start = base
	}
	args = append(args, path, start)
	_, err = run.Git(ctx, repo, args...)
	return err
}

func hasBranch(ctx context.Context, run Runner, repo, branch string) (bool, error) {
	_, err := run.Git(ctx, repo, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// List reads what git has checked out for this repository.
func List(ctx context.Context, run Runner, repo string) ([]Entry, error) {
	out, err := run.Git(ctx, repo, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	var entries []Entry
	var current Entry
	flush := func() {
		if current.Path != "" {
			current.Branch = strings.TrimPrefix(current.Branch, "refs/heads/")
			entries = append(entries, current)
		}
		current = Entry{}
	}
	for _, line := range strings.Split(out, "\n") {
		key, value, _ := strings.Cut(line, " ")
		switch key {
		case "worktree":
			flush()
			current.Path = value
		case "branch":
			current.Branch = value
		case "HEAD":
			current.Head = value
		case "bare":
			current.Bare = true
		}
	}
	flush()
	return entries, nil
}

// At reports whether a path is already a worktree of this repository.
func At(entries []Entry, path string) (Entry, bool) {
	clean := filepath.Clean(path)
	for _, entry := range entries {
		if filepath.Clean(entry.Path) == clean {
			return entry, true
		}
	}
	return Entry{}, false
}

// Remove drops one worktree. The branch stays, so the work is not lost.
func Remove(ctx context.Context, run Runner, repo, path string) error {
	_, err := run.Git(ctx, repo, "worktree", "remove", "--force", path)
	return err
}

// Prune forgets worktrees whose directory is gone.
func Prune(ctx context.Context, run Runner, repo string) error {
	_, err := run.Git(ctx, repo, "worktree", "prune")
	return err
}

// Path is where a run of this branch is checked out. The branch name becomes a directory
// name, so feat/sal-3 lives in .span/worktrees/feat-sal-3.
func Path(root, branch string) string {
	name := strings.ReplaceAll(strings.TrimSpace(branch), "/", "-")
	name = strings.Trim(name, "-")
	if name == "" {
		name = "run"
	}
	return filepath.Join(root, ".span", "worktrees", name)
}
