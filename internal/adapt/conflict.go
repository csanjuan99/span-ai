package adapt

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"span/internal/engine"
)

// ErrConflict is a change the host refused to merge because it conflicts with its base.
// It is the engine's sentinel, so errors.Is works on either side of the port.
var ErrConflict = engine.ErrConflict

// SideCap is how much of one side of a conflicted file is read for the agent. A file past
// it is marked truncated; resolving it still writes the whole blob, never the cut text.
const SideCap = 256 * 1024

// IsConflictMessage reports whether a forge's refusal says the change cannot be merged
// because of conflicts, as opposed to a refusal whose outcome is unknown.
func IsConflictMessage(text string) bool {
	text = strings.ToLower(text)
	for _, phrase := range []string{"not mergeable", "merge conflict", "conflicts", "cannot be cleanly"} {
		if strings.Contains(text, phrase) {
			return true
		}
	}
	return false
}

// Integrate merges base into the checked-out branch without committing: origin/<base>
// after a fetch when the checkout has an origin remote, the local base otherwise. A clean
// merge is left staged for FinishMerge; a conflicted one is left in progress and every
// path git could not merge is returned with its kind and its three sides. A branch that
// already contains base merges nothing.
func (g Git) Integrate(ctx context.Context, base string) (engine.Integration, error) {
	base = strings.TrimSpace(base)
	if base == "" {
		return engine.Integration{}, errors.New("no base branch")
	}
	merging, err := g.Merging(ctx)
	if err != nil {
		return engine.Integration{}, err
	}
	if merging {
		return engine.Integration{}, errors.New("a merge is already in progress in this checkout")
	}
	// Untracked files (span.yaml, which span never commits) do not block a merge, and git
	// refuses one that would overwrite them. Tracked edits would be lost by an abort.
	changes, err := g.output(ctx, "status", "--porcelain", "--untracked-files=no", "--", ".", ":(exclude).span")
	if err != nil {
		return engine.Integration{}, err
	}
	if changes != "" {
		return engine.Integration{}, errors.New("the checkout has uncommitted changes; span merges the base only into a clean tree")
	}
	ref, err := g.baseRef(ctx, base)
	if err != nil {
		return engine.Integration{}, err
	}
	sha, err := g.output(ctx, "rev-parse", ref+"^{commit}")
	if err != nil {
		return engine.Integration{}, err
	}
	out := engine.Integration{Base: ref, BaseSHA: sha}
	if _, err := g.output(ctx, "merge-base", "--is-ancestor", sha, "HEAD"); err == nil {
		out.Clean = true
		return out, nil
	}
	mergeBase, err := g.output(ctx, "merge-base", "HEAD", sha)
	if err != nil {
		return engine.Integration{}, err
	}
	_, mergeErr := g.output(ctx, "merge", "--no-commit", "--no-ff", "--no-edit", sha)
	if mergeErr == nil {
		out.Clean = true
		out.Merged = true
		return out, nil
	}
	files, err := g.conflicts(ctx, mergeBase)
	if err != nil {
		return engine.Integration{}, errors.Join(err, g.Abort(ctx))
	}
	if len(files) == 0 {
		// The merge failed for a reason that is not a conflict. Nothing it touched may stay.
		return engine.Integration{}, errors.Join(mergeErr, g.Abort(ctx))
	}
	out.Files = files
	return out, nil
}

// Merging reports whether a merge is in progress in the checkout.
func (g Git) Merging(ctx context.Context) (bool, error) {
	path, err := g.output(ctx, "rev-parse", "--git-path", "MERGE_HEAD")
	if err != nil {
		return false, err
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(g.Dir, path)
	}
	_, err = os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// Abort runs git merge --abort, which puts HEAD, the index and the tree back where they
// were before Integrate. With no merge in progress there is nothing to undo.
func (g Git) Abort(ctx context.Context) error {
	merging, err := g.Merging(ctx)
	if err != nil || !merging {
		return err
	}
	_, err = g.output(ctx, "merge", "--abort")
	return err
}

// Resolve writes one resolution per conflicted path and stages it. A path that is
// absolute or leaves the checkout is refused before anything is written, and a text file
// that still carries conflict markers is never staged.
func (g Git) Resolve(ctx context.Context, files map[string]engine.FileResolution) error {
	paths := make([]string, 0, len(files))
	for path := range files {
		if err := checkPath(path); err != nil {
			return err
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if err := g.resolveOne(ctx, path, files[path]); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	}
	return nil
}

func (g Git) resolveOne(ctx context.Context, path string, res engine.FileResolution) error {
	full := filepath.Join(g.Dir, filepath.FromSlash(path))
	switch {
	case res.Delete:
		if err := os.Remove(full); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		_, err := g.output(ctx, "rm", "--cached", "--quiet", "--ignore-unmatch", "--", path)
		return err
	case res.Stage:
		raw, err := os.ReadFile(full)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if HasMarkers(raw) {
			return errors.New("conflict markers are still in the file")
		}
		_, err = g.output(ctx, "add", "-A", "--", path)
		return err
	case res.Write:
		if HasMarkers(res.Content) {
			return errors.New("conflict markers are still in the resolved text")
		}
		return g.place(ctx, path, full, res.Content, res.Mode)
	case res.Blob != "":
		raw, err := g.blob(ctx, res.Blob, -1)
		if err != nil {
			return err
		}
		return g.place(ctx, path, full, raw, res.Mode)
	default:
		return errors.New("the resolution writes nothing")
	}
}

func (g Git) place(ctx context.Context, path, full string, data []byte, mode string) error {
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	perm := os.FileMode(0o644)
	if mode == "100755" {
		perm = 0o755
	}
	if err := os.WriteFile(full, data, perm); err != nil {
		return err
	}
	// WriteFile keeps the mode of a file that already existed.
	if err := os.Chmod(full, perm); err != nil {
		return err
	}
	if _, err := g.output(ctx, "add", "--", path); err != nil {
		return err
	}
	if mode == "100755" || mode == "100644" {
		// Windows has no executable bit to read back, so the index is told directly.
		flag := "--chmod=-x"
		if mode == "100755" {
			flag = "--chmod=+x"
		}
		if _, err := g.output(ctx, "update-index", flag, "--", path); err != nil {
			return err
		}
	}
	return nil
}

// FinishMerge commits the merge in progress and returns the new head. It refuses while any
// path is still unmerged.
func (g Git) FinishMerge(ctx context.Context, message string) (string, error) {
	left, err := g.output(ctx, "ls-files", "-u")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(left) != "" {
		return "", errors.New("some paths are still unmerged")
	}
	if _, err := g.output(ctx, "commit", "-m", message); err != nil {
		return "", err
	}
	return g.Head(ctx)
}

// Preview is the unified diff from before (a blob, or nothing) to what res would write.
// A regenerate resolution writes what its command produces, which is not known yet.
func (g Git) Preview(ctx context.Context, path, before string, res engine.FileResolution) (string, error) {
	if err := checkPath(path); err != nil {
		return "", err
	}
	if res.Stage {
		return "", nil
	}
	var old, next []byte
	hadOld, hasNext := before != "", !res.Delete
	if hadOld {
		raw, err := g.blob(ctx, before, -1)
		if err != nil {
			return "", err
		}
		old = raw
	}
	switch {
	case res.Write:
		next = res.Content
	case res.Blob != "":
		raw, err := g.blob(ctx, res.Blob, -1)
		if err != nil {
			return "", err
		}
		next = raw
	}
	return g.diffBytes(ctx, path, old, next, hadOld, hasNext)
}

func (g Git) diffBytes(ctx context.Context, path string, old, next []byte, hadOld, hasNext bool) (string, error) {
	if hadOld == hasNext && bytes.Equal(old, next) {
		return "", nil
	}
	dir, err := os.MkdirTemp("", "span-preview-*")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	if err := os.WriteFile(a, old, 0o600); err != nil {
		return "", err
	}
	if err := os.WriteFile(b, next, 0o600); err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, "git", "diff", "--no-index", "--no-color", "--no-ext-diff", "--", a, b)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// --no-index exits 1 when the files differ, which is the expected case.
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 1 {
			return "", fmt.Errorf("git diff --no-index: %w: %s", err, stderr.String())
		}
	}
	from, to := "a/"+path, "b/"+path
	if !hadOld {
		from = "/dev/null"
	}
	if !hasNext {
		to = "/dev/null"
	}
	var out strings.Builder
	fmt.Fprintf(&out, "--- %s\n+++ %s\n", from, to)
	body := false
	for _, line := range strings.SplitAfter(stdout.String(), "\n") {
		if strings.HasPrefix(line, "@@") {
			body = true
		}
		if strings.HasPrefix(line, "Binary files") {
			out.WriteString("Binary files differ\n")
			continue
		}
		if body {
			out.WriteString(line)
		}
	}
	return out.String(), nil
}

// HasMarkers reports a text that still carries git conflict markers. Binary content is
// never checked: its bytes are not lines.
func HasMarkers(data []byte) bool {
	if bytes.IndexByte(data, 0) >= 0 {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "<<<<<<< ") || strings.HasPrefix(line, ">>>>>>> ") || line == "<<<<<<<" || line == ">>>>>>>" {
			return true
		}
	}
	return false
}

// checkPath refuses a path that is absolute, empty, leaves the checkout, or points into
// git's own directory.
func checkPath(path string) error {
	clean := filepath.FromSlash(path)
	if path == "" || filepath.IsAbs(clean) || strings.HasPrefix(path, "/") || !filepath.IsLocal(clean) {
		return fmt.Errorf("path %q is outside the checkout", path)
	}
	first := strings.SplitN(filepath.ToSlash(filepath.Clean(clean)), "/", 2)[0]
	if strings.EqualFold(first, ".git") {
		return fmt.Errorf("path %q is inside .git", path)
	}
	return nil
}

func (g Git) baseRef(ctx context.Context, base string) (string, error) {
	if _, err := g.output(ctx, "remote", "get-url", "origin"); err == nil {
		if _, err := g.output(ctx, "fetch", "--quiet", "origin", base); err != nil {
			return "", err
		}
		remote := "origin/" + base
		if _, err := g.output(ctx, "rev-parse", "--verify", "--quiet", remote+"^{commit}"); err == nil {
			return remote, nil
		}
	}
	if _, err := g.output(ctx, "rev-parse", "--verify", "--quiet", base+"^{commit}"); err != nil {
		return "", fmt.Errorf("base branch %s is missing", base)
	}
	return base, nil
}

// conflicts lists the unmerged paths from git status --porcelain=v2, which names each
// path's state (UU, AA, AU, UA, DD, UD, DU) and its three stage blobs.
func (g Git) conflicts(ctx context.Context, mergeBase string) ([]engine.ConflictFile, error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain=v2", "-z", "--untracked-files=no")
	cmd.Dir = g.Dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git status: %w: %s", err, stderr.String())
	}
	var files []engine.ConflictFile
	for _, entry := range strings.Split(stdout.String(), "\x00") {
		if !strings.HasPrefix(entry, "u ") {
			continue
		}
		// u <XY> <sub> <m1> <m2> <m3> <mW> <h1> <h2> <h3> <path>
		fields := strings.SplitN(entry, " ", 11)
		if len(fields) != 11 {
			return nil, fmt.Errorf("git status printed %q", entry)
		}
		file := engine.ConflictFile{Path: fields[10]}
		state := fields[1]
		sides := []*engine.FileSide{&file.Base, &file.Ours, &file.Theirs}
		for i, side := range sides {
			mode, sha := fields[3+i], fields[7+i]
			if strings.Trim(sha, "0") == "" {
				continue
			}
			side.Present, side.Mode, side.SHA = true, mode, sha
			raw, err := g.blob(ctx, sha, SideCap+1)
			if err != nil {
				return nil, err
			}
			side.Binary = bytes.IndexByte(raw[:min(len(raw), 8000)], 0) >= 0
			if len(raw) > SideCap {
				raw, side.Truncated = raw[:SideCap], true
			}
			if !side.Binary {
				side.Text = string(raw)
			}
		}
		renamed := false
		switch state {
		case "AU", "UA", "DD":
			renamed = true
		case "UD", "DU":
			// A path the merge base never had was renamed on one side and deleted on the other.
			_, err := g.output(ctx, "cat-file", "-e", mergeBase+":"+file.Path)
			renamed = err != nil
		}
		file.Kind = Kind(file, state, renamed)
		files = append(files, file)
	}
	return files, nil
}

// Kind names what kind of conflict one path has. Lockfiles and generated files come first
// because they are regenerated rather than merged by hand.
func Kind(file engine.ConflictFile, state string, renamed bool) string {
	switch {
	case Lockfile(file.Path):
		return "lockfile"
	case Generated(file.Path, file.Ours.Text, file.Theirs.Text):
		return "generated"
	case file.Ours.Binary || file.Theirs.Binary || file.Base.Binary:
		return "binary"
	case renamed:
		return "rename"
	case state == "UD" || state == "DU":
		return "modify-delete"
	case file.Ours.Present && file.Theirs.Present && file.Ours.SHA == file.Theirs.SHA && file.Ours.Mode != file.Theirs.Mode:
		return "mode"
	case state == "AA":
		return "add-add"
	case state == "UU":
		return "content"
	default:
		return "other"
	}
}

var lockfiles = map[string]bool{
	"package-lock.json": true, "yarn.lock": true, "pnpm-lock.yaml": true, "go.sum": true,
	"Cargo.lock": true, "poetry.lock": true, "Gemfile.lock": true, "composer.lock": true,
	"bun.lockb": true, "bun.lock": true, "npm-shrinkwrap.json": true, "uv.lock": true,
}

// Lockfile reports a dependency lockfile a package manager writes.
func Lockfile(path string) bool {
	return lockfiles[filepath.Base(filepath.FromSlash(path))]
}

// Generated reports a file a tool writes: by its path, or by a generated header in the
// first lines of either side.
func Generated(path string, sides ...string) bool {
	slash := "/" + filepath.ToSlash(path)
	for _, part := range []string{".generated.", "_generated.", "/generated/", "/dist/"} {
		if strings.Contains(slash, part) {
			return true
		}
	}
	if strings.HasSuffix(slash, ".pb.go") || strings.HasSuffix(slash, ".min.js") {
		return true
	}
	for _, text := range sides {
		lines := strings.SplitN(text, "\n", 6)
		if len(lines) > 5 {
			lines = lines[:5]
		}
		for _, line := range lines {
			if strings.Contains(line, "@generated") || strings.Contains(line, "DO NOT EDIT") {
				return true
			}
		}
	}
	return false
}

// blob reads one object. A negative limit reads all of it; otherwise at most limit bytes.
func (g Git) blob(ctx context.Context, sha string, limit int) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", "cat-file", "blob", sha)
	cmd.Dir = g.Dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	var reader io.Reader = stdout
	if limit >= 0 {
		reader = io.LimitReader(stdout, int64(limit))
	}
	raw, readErr := io.ReadAll(reader)
	_, _ = io.Copy(io.Discard, stdout)
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("git cat-file blob %s: %w: %s", sha, err, stderr.String())
	}
	if readErr != nil {
		return nil, readErr
	}
	return raw, nil
}
