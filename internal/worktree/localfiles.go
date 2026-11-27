package worktree

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// CopyLocalFiles gives a new tree the .env files git ignores: the checkout has them and a
// tree made from a commit does not, so the project would not start there. Only files the
// tree ignores too are copied, so none can be committed. A file the tree already has is
// left alone, and each copy is the tree's own: nothing is ever written back.
func CopyLocalFiles(ctx context.Context, run Runner, project, tree string) error {
	out, err := run.Git(ctx, project, "ls-files", "-z", "--others", "--ignored", "--exclude-standard", "--",
		":(glob)**/.env", ":(glob)**/.env.*", ":(exclude).span",
		":(exclude,glob)**/node_modules/**", ":(exclude,glob)**/.venv/**", ":(exclude,glob)**/vendor/**")
	if err != nil {
		return err
	}
	var failed []error
	for _, name := range strings.Split(out, "\x00") {
		if name == "" {
			continue
		}
		// The tree's branch may not ignore what the checkout does. check-ignore fails for a
		// path that is not ignored, which includes one it cannot answer for.
		if _, err := run.Git(ctx, tree, "check-ignore", "-q", "--", name); err != nil && !behindLink(tree, name) {
			log.Printf("%s is not copied to %s: the tree does not ignore it", name, tree)
			continue
		}
		if err := copyMissing(project, tree, name); err != nil {
			failed = append(failed, fmt.Errorf("copy %s: %w", name, err))
		}
	}
	return errors.Join(failed...)
}

// behindLink is true when a folder on the way to name inside root is a symlink, so a write
// there would land somewhere else.
func behindLink(root, name string) bool {
	path := root
	parts := strings.Split(filepath.ToSlash(name), "/")
	for _, part := range parts[:len(parts)-1] {
		path = filepath.Join(path, part)
		info, err := os.Lstat(path)
		if err != nil {
			return false
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return true
		}
	}
	return false
}

// copyMissing copies one file of the project to the tree where there is nothing yet. A
// symlink in the checkout is copied as the file it points at.
func copyMissing(project, tree, name string) error {
	if behindLink(tree, name) {
		return errors.New("a folder on the way is a symlink")
	}
	to := filepath.Join(tree, name)
	if _, err := os.Lstat(to); !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	from := filepath.Join(project, name)
	info, err := os.Stat(from)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	body, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}
	return os.WriteFile(to, body, info.Mode().Perm())
}
