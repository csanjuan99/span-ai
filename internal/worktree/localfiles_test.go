package worktree

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// A tree gets the env files git ignores, which a checkout has and a new tree does not.
func TestLocalEnvFilesFollowTheRunIntoItsTree(t *testing.T) {
	project := t.TempDir()
	run := shell{t}
	run.must(project, "init", "-q", "-b", "dev")
	writeLocal(t, filepath.Join(project, ".gitignore"), ".env*\nnode_modules/\n.span/\n")
	for _, name := range []string{".env", "web/.env.local", "node_modules/p/.env", ".span/x/.env", "notes.txt"} {
		writeLocal(t, filepath.Join(project, name), name)
	}
	tree := t.TempDir()
	run.must(tree, "init", "-q", "-b", "dev")
	writeLocal(t, filepath.Join(tree, ".gitignore"), ".env*\nnode_modules/\n.span/\n")
	writeLocal(t, filepath.Join(tree, ".env"), "kept")
	if err := CopyLocalFiles(context.Background(), run, project, tree); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{".env": "kept", "web/.env.local": "web/.env.local", "node_modules/p/.env": "", ".span/x/.env": "", "notes.txt": ""} {
		raw, err := os.ReadFile(filepath.Join(tree, name))
		if want == "" {
			if !os.IsNotExist(err) {
				t.Fatalf("%s was copied", name)
			}
			continue
		}
		if err != nil || string(raw) != want {
			t.Fatalf("%s: %q %v", name, raw, err)
		}
	}
	info, err := os.Stat(filepath.Join(tree, "web", ".env.local"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("the permissions changed: %v", info.Mode())
	}
}

// A copy must never become something to commit, leave the tree, or lead back to the
// checkout.
func TestCopiedEnvFilesStayLocalToTheTree(t *testing.T) {
	project, tree, outside := t.TempDir(), t.TempDir(), t.TempDir()
	run := shell{t}
	for _, dir := range []string{project, tree} {
		run.must(dir, "init", "-q", "-b", "dev")
	}
	// The checkout ignores every env file; the tree's branch only ignores some of them.
	writeLocal(t, filepath.Join(project, ".gitignore"), ".env*\n")
	writeLocal(t, filepath.Join(tree, ".gitignore"), ".env\ncfg\n")
	writeLocal(t, filepath.Join(project, ".env.unignored"), "secret")
	writeLocal(t, filepath.Join(project, "cfg", ".env"), "secret")
	writeLocal(t, filepath.Join(project, "shared"), "original")
	if err := os.Symlink(filepath.Join(project, "shared"), filepath.Join(project, ".env")); err != nil {
		t.Skipf("no symlinks here: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(tree, "cfg")); err != nil {
		t.Fatal(err)
	}
	// The refused path is reported; the other files still follow.
	if err := CopyLocalFiles(context.Background(), run, project, tree); err == nil {
		t.Fatal("a destination behind a symlink must be reported")
	}
	if _, err := os.Stat(filepath.Join(tree, ".env.unignored")); !os.IsNotExist(err) {
		t.Fatal("a file the tree would commit was copied")
	}
	if _, err := os.Stat(filepath.Join(outside, ".env")); !os.IsNotExist(err) {
		t.Fatal("a file was written outside the tree")
	}
	info, err := os.Lstat(filepath.Join(tree, ".env"))
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("the tree's env file must be its own copy: %v", err)
	}
}

// writeLocal writes a private file, making its folders.
func writeLocal(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
