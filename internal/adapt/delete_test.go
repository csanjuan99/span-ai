package adapt

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"span/internal/worktree"
)

func TestDeletingARunRemovesItsWorktreeOnlyWhenNothingWouldBeLost(t *testing.T) {
	for name, tc := range map[string]struct {
		file string
		kept bool
	}{
		"clean":                       {},
		"with a file git would hide":  {file: "work.txt", kept: true},
		"with an ignored file":        {file: ".env", kept: true},
		"with only Span state":        {file: ".span/workspace.lock"},
		"with installed dependencies": {file: "node_modules/p/index.js"},
		"with the checkout's env":     {file: ".env.copy"},
	} {
		t.Run(name, func(t *testing.T) {
			project := t.TempDir()
			git := func(args ...string) string {
				t.Helper()
				return gitAt(t, project, args...)
			}
			git("init", "-q", "-b", "dev")
			git("config", "status.showUntrackedFiles", "no")
			if err := os.WriteFile(filepath.Join(project, ".gitignore"), []byte(".env*\nnode_modules/\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			git("add", ".gitignore")
			git("commit", "-q", "-m", "init")
			// The tree writes "x"; an env file identical to the checkout's loses nothing.
			if err := os.WriteFile(filepath.Join(project, ".env.copy"), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
			tree := worktree.Path(project, "feat/x")
			git("worktree", "add", "-q", "-b", "feat/x", "--", tree)
			if tc.file != "" {
				path := filepath.Join(tree, tc.file)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			deleteRecordedRun(t, project, tree)
			_, err := os.Stat(tree)
			if kept := err == nil; kept != tc.kept {
				t.Fatalf("tree kept %v, want %v", kept, tc.kept)
			}
			if git("branch", "--list", "feat/x") == "" {
				t.Fatal("the branch was deleted")
			}
		})
	}
}

// Commits made off the run's branch live only in the tree.
func TestDeletingARunKeepsATreeWhoseHeadLeftTheBranch(t *testing.T) {
	project := t.TempDir()
	gitAt(t, project, "init", "-q", "-b", "dev")
	gitAt(t, project, "commit", "-q", "--allow-empty", "-m", "init")
	tree := worktree.Path(project, "feat/x")
	gitAt(t, project, "worktree", "add", "-q", "-b", "feat/x", "--", tree)
	gitAt(t, tree, "checkout", "-q", "--detach")
	gitAt(t, tree, "commit", "-q", "--allow-empty", "-m", "only here")
	deleteRecordedRun(t, project, tree)
	if _, err := os.Stat(tree); err != nil {
		t.Fatalf("the tree with an unreferenced commit was removed: %v", err)
	}
}

// gitAt runs git in dir with a test identity.
func gitAt(t *testing.T, dir string, args ...string) string {
	t.Helper()
	identity := []string{"-C", dir, "-c", "user.email=t@example.com", "-c", "user.name=T", "-c", "commit.gpgsign=false"}
	out, err := exec.Command("git", append(identity, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
	return string(out)
}

// deleteRecordedRun records a run of feat/x that works in tree, and deletes it.
func deleteRecordedRun(t *testing.T, project, tree string) {
	t.Helper()
	root := filepath.Join(project, ".span", "runs")
	if err := os.MkdirAll(filepath.Join(root, "span-1"), 0o755); err != nil {
		t.Fatal(err)
	}
	cursor := `{"id":"span-1","branch":"feat/x","project":"` + project + `","workspace":"` + tree + `"}`
	if err := os.WriteFile(filepath.Join(root, "span-1", "cursor.json"), []byte(cursor), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := DeleteRun(context.Background(), root, "span-1"); err != nil {
		t.Fatal(err)
	}
}
