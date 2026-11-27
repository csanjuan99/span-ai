package engine

import (
	"path/filepath"
	"testing"
)

// A run in a worktree belongs to its repository, not to the folder named after the branch.
func TestARunInAWorktreeIsNamedAfterItsRepository(t *testing.T) {
	project := filepath.Join(t.TempDir(), "api")
	cur := Cursor{ID: "span-1", Project: project, Workspace: filepath.Join(project, ".span", "worktrees", "feat-x")}
	if got := Label(cur); got != "span-1-api" {
		t.Fatalf("label %q", got)
	}
	if got := RepoName(cur.Home()); got != "api" {
		t.Fatalf("repo %q", got)
	}
}
