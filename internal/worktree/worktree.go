// Package worktree keeps parallel runs apart. Every run that walks a branch does it in
// its own git worktree, so two stories in one repository never share a working tree and
// the checkout the person is using is never switched.
//
// The commands here are the whole surface Span uses. Nothing is inferred:
//
//	git worktree add -- <path> <branch>                       the branch already exists
//	git worktree add --no-track -b <branch> -- <path> <base>  the branch is new
//	git fetch --quiet origin -- +refs/heads/<base>:refs/remotes/origin/<base>  best effort
//	git worktree list --porcelain             what is checked out where
//	git worktree remove --force <path>        drop an isolated tree
//	git worktree prune                        forget trees that are gone
package worktree

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	branchpkg "span/internal/branch"
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

// StartPoint is the full ref a new branch is cut from for base. It always tries a
// best-effort fetch of the base from origin first, then prefers refs/remotes/origin/<base>:
// a local base that is behind or has diverged must not decide what a new branch starts
// from. Without origin's copy it uses the local branch; when the fetch failed (offline) and
// both exist, the local branch wins silently. The person's local base branch is never
// moved. Tags and commits are not bases; anything else is a branch.MissingBase error, and a
// base git could read as an option never reaches git.
func StartPoint(ctx context.Context, run Runner, repo, base string) (string, error) {
	if err := branchpkg.CheckBase(base); err != nil {
		return "", err
	}
	local, remote := "refs/heads/"+base, "refs/remotes/origin/"+base
	fetchErr := fetchBase(ctx, run, repo, base)
	if fetchErr == nil && hasRef(ctx, run, repo, remote) {
		return remote, nil
	}
	if hasRef(ctx, run, repo, local) {
		return local, nil
	}
	if hasRef(ctx, run, repo, remote) {
		return remote, nil
	}
	return "", branchpkg.MissingBase{Name: base, Cause: fetchErr}
}

// OnOrigin reports whether base is a branch the forge can target: origin has it after a
// best-effort fetch. A repository without an origin remote has no forge, so its local
// branch is enough. An invalid base is an error, not a false.
func OnOrigin(ctx context.Context, run Runner, repo, base string) (bool, error) {
	if err := branchpkg.CheckBase(base); err != nil {
		return false, err
	}
	if !hasOrigin(ctx, run, repo) {
		return hasRef(ctx, run, repo, "refs/heads/"+base), nil
	}
	remote := "refs/remotes/origin/" + base
	// A failed fetch (offline) leaves the remote-tracking ref already known to decide.
	if err := fetchBase(ctx, run, repo, base); err != nil && !hasRef(ctx, run, repo, remote) {
		return false, nil
	}
	return hasRef(ctx, run, repo, remote), nil
}

// fetchBase refreshes origin's copy of base into its remote-tracking ref only. It never
// touches refs/heads, so the person's own branch stays where they left it. A missing origin
// remote is reported like any other failed fetch.
func fetchBase(ctx context.Context, run Runner, repo, base string) error {
	if !hasOrigin(ctx, run, repo) {
		return fmt.Errorf("no origin remote")
	}
	_, err := run.Git(ctx, repo, "fetch", "--quiet", "origin", "--", "+refs/heads/"+base+":refs/remotes/origin/"+base)
	return err
}

func hasOrigin(ctx context.Context, run Runner, repo string) bool {
	_, err := run.Git(ctx, repo, "config", "--get", "remote.origin.url")
	return err == nil
}

// hasRef checks one full ref name (refs/heads/... or refs/remotes/...) exactly; it never
// resolves arbitrary text the way rev-parse does.
func hasRef(ctx context.Context, run Runner, repo, ref string) bool {
	_, err := run.Git(ctx, repo, "show-ref", "--verify", "--quiet", "--", ref)
	return err == nil
}

// Add checks out branch at path. A branch that does not exist yet is created from base;
// one that does is reused, so resuming a run lands on the same tree instead of a second
// copy of the same branch.
func Add(ctx context.Context, run Runner, repo, path, branch, base string) error {
	if strings.TrimSpace(branch) == "" {
		return fmt.Errorf("worktree needs a branch")
	}
	if !branchpkg.ValidRef(branch) {
		return fmt.Errorf("worktree branch %q is not a valid branch name", branch)
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
	// The last argument is what gets checked out: the branch when it already exists, the
	// base ref when the branch is being created. "--" keeps both from being read as options.
	if exists {
		_, err = run.Git(ctx, repo, "worktree", "add", "--", path, branch)
		return err
	}
	if strings.TrimSpace(base) == "" {
		return fmt.Errorf("branch %s does not exist and no base was given", branch)
	}
	start, err := StartPoint(ctx, run, repo, base)
	if err != nil {
		return err
	}
	// --no-track: the new branch pushes to its own name and is not tied to its base.
	_, err = run.Git(ctx, repo, "worktree", "add", "--no-track", "-b", branch, "--", path, start)
	return err
}

func hasBranch(ctx context.Context, run Runner, repo, branch string) (bool, error) {
	return hasRef(ctx, run, repo, "refs/heads/"+branch), nil
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
