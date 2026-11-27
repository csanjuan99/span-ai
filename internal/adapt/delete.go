package adapt

import (
	"bytes"
	"context"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"span/internal/engine"
	"span/internal/fsutil"
	"span/internal/worktree"
)

// DeleteRun deletes a run's record and then the worktree the run made, when nothing in it
// would be lost. The branch always stays. A tree with uncommitted or untracked files, a
// tree another run uses, and the person's own checkout are left where they are.
func DeleteRun(ctx context.Context, root, id string) error {
	snap, loadErr := engine.Load(root, id)
	if err := engine.Delete(root, id); err != nil {
		return err
	}
	if loadErr != nil {
		// The record could not be read, so no tree can be attributed to the run.
		log.Printf("delete %s: its worktree is kept, the record could not be read: %v", id, loadErr)
		return nil
	}
	removeTree(ctx, root, snap.Cursor)
	return nil
}

// removeTree drops the worktree of a deleted run when nothing in it would be lost. It
// keeps the tree on any doubt.
func removeTree(ctx context.Context, root string, cur engine.Cursor) {
	if cur.Project == "" || cur.Branch == "" || !fsutil.SamePath(cur.Workspace, worktree.Path(cur.Project, cur.Branch)) {
		return
	}
	unlock, err := engine.LockLaunches(cur.Project)
	if err != nil {
		log.Printf("delete %s: its worktree is kept: %v", cur.ID, err)
		return
	}
	defer unlock()
	for _, other := range engine.RunsOnBranch(cur.Project, root, cur.Branch) {
		if fsutil.SamePath(other.Workspace, cur.Workspace) {
			return
		}
	}
	// Commits made with HEAD elsewhere (detached, or a rebase in progress) and a merge in
	// progress live only in the tree.
	tree := Git{Dir: cur.Workspace}
	head, err := tree.Git(ctx, cur.Workspace, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil || head != cur.Branch {
		log.Printf("delete %s: its worktree is kept at %s, HEAD is not on %s (%v)", cur.ID, cur.Workspace, cur.Branch, err)
		return
	}
	if _, err := tree.Git(ctx, cur.Workspace, "rev-parse", "--quiet", "--verify", "MERGE_HEAD"); err == nil {
		log.Printf("delete %s: its worktree is kept at %s, a merge is in progress", cur.ID, cur.Workspace)
		return
	}
	// The flags are spelled out because a person's git settings can hide untracked files,
	// and git removes a tree with ignored files without asking.
	left, err := (Git{Dir: cur.Workspace}).Git(ctx, cur.Workspace, "status", "--porcelain", "-z", "--untracked-files=all", "--ignored=matching", "--ignore-submodules=none", "--", ".", ":(exclude).span")
	if err != nil {
		log.Printf("delete %s: its worktree is kept at %s: %v", cur.ID, cur.Workspace, err)
		return
	}
	for _, entry := range strings.Split(left, "\x00") {
		if entry != "" && !rebuildable(cur.Project, cur.Workspace, entry) {
			log.Printf("delete %s: its worktree is kept at %s, it holds %s", cur.ID, cur.Workspace, entry)
			return
		}
	}
	if _, err := (Git{Dir: cur.Project}).Git(ctx, cur.Project, "worktree", "remove", "--", cur.Workspace); err != nil {
		log.Printf("delete %s: its worktree is kept at %s: %v", cur.ID, cur.Workspace, err)
	}
}

// rebuildable is true for a status entry that loses nothing with the tree: an ignored
// folder of installed dependencies or tool caches, or an env file the checkout still has
// byte for byte.
func rebuildable(project, tree, entry string) bool {
	name, ignored := strings.CutPrefix(entry, "!! ")
	if !ignored {
		return false
	}
	folder := strings.HasSuffix(name, "/")
	base := path.Base(name)
	switch base {
	case "node_modules", ".venv", "__pycache__", ".pytest_cache":
		return folder
	case "vendor":
		_, err := os.Stat(filepath.Join(tree, path.Dir(strings.TrimSuffix(name, "/")), "composer.lock"))
		return folder && err == nil
	}
	if base != ".env" && !strings.HasPrefix(base, ".env.") {
		return false
	}
	mine, err := os.ReadFile(filepath.Join(tree, name))
	if err != nil {
		return false
	}
	theirs, err := os.ReadFile(filepath.Join(project, name))
	return err == nil && bytes.Equal(mine, theirs)
}
