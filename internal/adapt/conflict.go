package adapt

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
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

const gitlink = "160000"

// IsConflictMessage reports whether a forge's refusal, as its stderr says it, is a merge
// conflict. A refusal from branch policy, a required check or a missing review is a plain
// failure, and so is a bare "not mergeable": it does not say why.
func IsConflictMessage(stderr string) bool {
	text := strings.ToLower(stderr)
	for _, policy := range []string{"base branch policy", "required status", "review required", "approving review"} {
		if strings.Contains(text, policy) {
			return false
		}
	}
	for _, phrase := range []string{"cannot be cleanly created", "merge conflict", "has conflicts"} {
		if strings.Contains(text, phrase) {
			return true
		}
	}
	return false
}

// Preflight computes the merge of base into HEAD with git merge-tree, which writes objects
// but never touches the index or the tree. A base that cannot be fetched or found, or a git
// without merge-tree --write-tree (older than 2.38), is not checked: Note says why.
func (g Git) Preflight(ctx context.Context, base string) (engine.Preflight, error) {
	ref, note := g.baseRef(ctx, base)
	if ref == "" {
		return engine.Preflight{Note: note}, nil
	}
	sha, err := g.output(ctx, "rev-parse", ref+"^{commit}")
	if err != nil {
		return engine.Preflight{Note: err.Error()}, nil
	}
	out := engine.Preflight{Base: ref, BaseSHA: sha}
	if _, err := g.output(ctx, "merge-base", "--is-ancestor", sha, "HEAD"); err == nil {
		out.Checked = true
		return out, nil
	}
	cmd := exec.CommandContext(ctx, "git", "merge-tree", "--write-tree", "--name-only", "HEAD", sha)
	cmd.Dir = g.Dir
	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr
	err = cmd.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		out.Checked = true
	case errors.As(err, &exit) && exit.ExitCode() == 1:
		out.Checked, out.Conflicts = true, true
	default:
		out.Note = "git merge-tree --write-tree is unavailable: " + strings.TrimSpace(stderr.String())
	}
	return out, nil
}

// Integrate merges base into the checked-out branch without committing: origin/<base>
// after a fetch when the checkout has an origin remote, the local base otherwise. A clean
// merge is left staged for FinishMerge; a conflicted one is left in progress and every
// path git could not merge is returned with its kind and its three sides. A branch that
// already contains base merges nothing.
func (g Git) Integrate(ctx context.Context, base string) (engine.Integration, error) {
	merging, err := g.Merging(ctx)
	if err != nil {
		return engine.Integration{}, err
	}
	if merging {
		return engine.Integration{}, errors.New("a merge is already in progress in this checkout")
	}
	// Untracked files (span.yaml, which span never commits) do not block a merge, and git
	// refuses one that would overwrite them. Tracked edits would be lost by an abort.
	changes, err := g.trackedChanges(ctx)
	if err != nil {
		return engine.Integration{}, err
	}
	if changes != "" {
		return engine.Integration{}, errors.New("the checkout has uncommitted changes; span merges the base only into a clean tree")
	}
	ref, note := g.baseRef(ctx, base)
	if ref == "" {
		return engine.Integration{}, errors.New(note)
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
	files, err := g.conflicts(ctx, mergeBase, sha)
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

func (g Git) trackedChanges(ctx context.Context) (string, error) {
	return g.output(ctx, "status", "--porcelain", "--untracked-files=no", "--", ".", ":(exclude).span")
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

// Abort runs git merge --abort and then makes sure the tracked tree is the branch again.
// Integrate only starts on a clean tracked tree, so anything still changed afterwards (a
// file a command touched outside the merge) is put back. With no merge in progress there
// is nothing to undo.
func (g Git) Abort(ctx context.Context) error {
	merging, err := g.Merging(ctx)
	if err != nil || !merging {
		return err
	}
	if _, err := g.output(ctx, "merge", "--abort"); err != nil {
		return err
	}
	left, err := g.trackedChanges(ctx)
	if err != nil || left == "" {
		return err
	}
	if _, err := g.output(ctx, "reset", "-q"); err != nil {
		return err
	}
	if _, err := g.output(ctx, "checkout", "--", "."); err != nil {
		return err
	}
	if left, err = g.trackedChanges(ctx); err != nil {
		return err
	}
	if left != "" {
		return fmt.Errorf("the merge was aborted but the tree still differs from the branch: %s", left)
	}
	return nil
}

// Resolve stages one resolution per conflicted path through git itself (update-index and
// checkout-index), never by writing the file directly, so a symlink in the tree cannot send
// the bytes outside the checkout. A path that is absolute, leaves the checkout, or passes
// through a symlinked directory is refused before anything changes, and a text that still
// carries conflict markers is never staged.
func (g Git) Resolve(ctx context.Context, files map[string]engine.FileResolution) error {
	paths := make([]string, 0, len(files))
	for path, res := range files {
		for _, name := range append([]string{path}, res.Remove...) {
			if err := g.safePath(name); err != nil {
				return err
			}
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
	for _, temp := range res.Remove {
		if err := g.drop(ctx, temp); err != nil {
			return err
		}
	}
	mode := res.Mode
	if mode == "" {
		mode = "100644"
	}
	switch {
	case res.Delete:
		return g.drop(ctx, path)
	case res.Stage:
		full := filepath.Join(g.Dir, filepath.FromSlash(path))
		info, err := os.Lstat(full)
		if errors.Is(err, os.ErrNotExist) {
			return g.drop(ctx, path)
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return errors.New("the regenerated path is not a regular file")
		}
		raw, err := os.ReadFile(full)
		if err != nil {
			return err
		}
		if engine.HasConflictMarkers(raw) {
			return errors.New("conflict markers are still in the file")
		}
		sha, err := g.output(ctx, "hash-object", "-w", "--", path)
		if err != nil {
			return err
		}
		if res.Mode == "" && info.Mode()&0o111 != 0 {
			mode = "100755"
		}
		return g.place(ctx, path, mode, sha)
	case res.Write:
		if engine.HasConflictMarkers(res.Content) {
			return errors.New("conflict markers are still in the resolved text")
		}
		out, err := g.raw(ctx, res.Content, "hash-object", "-w", "--stdin")
		if err != nil {
			return err
		}
		return g.place(ctx, path, mode, strings.TrimSpace(string(out)))
	case res.Blob != "":
		return g.place(ctx, path, mode, res.Blob)
	default:
		return errors.New("the resolution writes nothing")
	}
}

// place records path as one stage-0 entry and writes it to the tree from the index. Tracked
// paths under path/ are dropped first, because a file cannot sit where a directory is.
func (g Git) place(ctx context.Context, path, mode, sha string) error {
	under, err := g.raw(ctx, nil, "ls-files", "-z", "--", ":(literal)"+path+"/")
	if err != nil {
		return err
	}
	for _, child := range splitZ(under) {
		if err := g.drop(ctx, child); err != nil {
			return err
		}
	}
	full := filepath.Join(g.Dir, filepath.FromSlash(path))
	if info, err := os.Lstat(full); err == nil && info.IsDir() && mode != gitlink {
		removeEmptyDirs(full)
	}
	if _, err := g.output(ctx, "update-index", "--add", "--cacheinfo", mode+","+sha+","+path); err != nil {
		return err
	}
	if mode == gitlink {
		// A submodule's commit is recorded; its checkout is the submodule's business.
		return os.MkdirAll(full, 0o755)
	}
	_, err = g.output(ctx, "checkout-index", "-f", "--", path)
	return err
}

// drop removes path from the index and from the tree. A symlink is removed as a link; a
// directory is left alone.
func (g Git) drop(ctx context.Context, path string) error {
	if _, err := g.output(ctx, "update-index", "--force-remove", "--", path); err != nil {
		return err
	}
	full := filepath.Join(g.Dir, filepath.FromSlash(path))
	info, err := os.Lstat(full)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}
	if err := os.Remove(full); err != nil {
		return err
	}
	removeEmptyDirs(filepath.Dir(full))
	return nil
}

// removeEmptyDirs removes dir and its parents while they are empty, stopping at the first
// that is not. A failure only means the directory stays.
func removeEmptyDirs(dir string) {
	for {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 || os.Remove(dir) != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
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

// Preview is the unified diff from before (one side, possibly absent) to what res would
// write. A regenerate resolution writes what its command produces, which is not known yet,
// and a submodule shows the two commits.
func (g Git) Preview(ctx context.Context, path string, before engine.FileSide, res engine.FileResolution) (string, error) {
	if err := checkPath(path); err != nil {
		return "", err
	}
	if res.Stage {
		return "", nil
	}
	if before.Mode == gitlink || res.Mode == gitlink {
		from, to := "none", "none"
		if before.Present {
			from = before.SHA
		}
		if !res.Delete {
			to = res.Blob
		}
		if from == to {
			return "", nil
		}
		return fmt.Sprintf("Submodule %s: commit %s becomes %s\n", path, from, to), nil
	}
	var old, next []byte
	hadOld, hasNext := before.Present, !res.Delete
	if hadOld {
		raw, err := g.blob(ctx, before.SHA, -1)
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

// Snapshot reads the index, the tracked paths whose tree differs from it, and the untracked
// files git does not ignore.
func (g Git) Snapshot(ctx context.Context) (engine.TreeState, error) {
	state := engine.TreeState{Index: map[string]string{}, Dirty: map[string]bool{}, Untracked: map[string]bool{}}
	index, err := g.raw(ctx, nil, "ls-files", "-s", "-z")
	if err != nil {
		return engine.TreeState{}, err
	}
	for _, entry := range splitZ(index) {
		meta, path, ok := strings.Cut(entry, "\t")
		if !ok {
			return engine.TreeState{}, fmt.Errorf("git ls-files printed %q", entry)
		}
		if state.Index[path] != "" {
			state.Index[path] += "\n"
		}
		state.Index[path] += meta
	}
	dirty, err := g.raw(ctx, nil, "diff-files", "--name-only", "-z")
	if err != nil {
		return engine.TreeState{}, err
	}
	for _, path := range splitZ(dirty) {
		state.Dirty[path] = true
	}
	untracked, err := g.raw(ctx, nil, "ls-files", "-o", "-z", "--exclude-standard")
	if err != nil {
		return engine.TreeState{}, err
	}
	for _, path := range splitZ(untracked) {
		state.Untracked[path] = true
	}
	return state, nil
}

// Restore puts back what changed since before, except the paths in keep: index entries are
// rewritten from the snapshot, tracked files that differ are checked out from it, and new
// untracked files are removed. It returns the paths it restored, sorted.
func (g Git) Restore(ctx context.Context, before engine.TreeState, keep []string) ([]string, error) {
	kept := map[string]bool{}
	for _, path := range keep {
		kept[path] = true
	}
	now, err := g.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	changed := map[string]bool{}
	var info bytes.Buffer
	paths := map[string]bool{}
	for path := range before.Index {
		paths[path] = true
	}
	for path := range now.Index {
		paths[path] = true
	}
	for path := range paths {
		if kept[path] || before.Index[path] == now.Index[path] {
			continue
		}
		changed[path] = true
		// A mode 0 entry removes every stage of the path; the snapshot's entries follow.
		fmt.Fprintf(&info, "0 0000000000000000000000000000000000000000 0\t%s\x00", path)
		if before.Index[path] != "" {
			for _, meta := range strings.Split(before.Index[path], "\n") {
				fmt.Fprintf(&info, "%s\t%s\x00", meta, path)
			}
		}
	}
	if info.Len() > 0 {
		if _, err := g.raw(ctx, info.Bytes(), "update-index", "-z", "--index-info"); err != nil {
			return nil, err
		}
	}
	after, err := g.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	for path := range after.Dirty {
		if kept[path] || before.Dirty[path] || !strings.HasSuffix(strings.SplitN(before.Index[path], "\n", 2)[0], " 0") {
			continue
		}
		if err := g.safePath(path); err != nil {
			return nil, err
		}
		if _, err := g.output(ctx, "checkout-index", "-f", "--", path); err != nil {
			return nil, err
		}
		changed[path] = true
	}
	for path := range after.Untracked {
		if kept[path] || before.Untracked[path] {
			continue
		}
		if err := g.safePath(path); err != nil {
			return nil, err
		}
		full := filepath.Join(g.Dir, filepath.FromSlash(path))
		if info, err := os.Lstat(full); err == nil && !info.IsDir() {
			if err := os.Remove(full); err != nil {
				return nil, err
			}
			removeEmptyDirs(filepath.Dir(full))
		}
		changed[path] = true
	}
	out := make([]string, 0, len(changed))
	for path := range changed {
		out = append(out, path)
	}
	sort.Strings(out)
	return out, nil
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

// safePath is checkPath plus the tree itself: no directory on the way to the path may be a
// symlink, so nothing written there can land outside the checkout.
func (g Git) safePath(path string) error {
	if err := checkPath(path); err != nil {
		return err
	}
	parts := strings.Split(filepath.ToSlash(filepath.Clean(filepath.FromSlash(path))), "/")
	dir := g.Dir
	for _, part := range parts[:len(parts)-1] {
		dir = filepath.Join(dir, part)
		info, err := os.Lstat(dir)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return fmt.Errorf("path %q passes through the symlink %s", path, part)
		}
	}
	return nil
}

// baseRef fetches base when the checkout has an origin, into refs/remotes/origin/<base> so a
// single-branch clone gets it too, and falls back to the local branch without one. An empty
// ref means the base cannot be found; the note says why.
func (g Git) baseRef(ctx context.Context, base string) (string, string) {
	base = strings.TrimSpace(base)
	if base == "" || strings.HasPrefix(base, "-") || strings.ContainsAny(base, ": \t\n") {
		return "", fmt.Sprintf("base branch %q is not a branch name", base)
	}
	if _, err := g.output(ctx, "check-ref-format", "--branch", base); err != nil {
		return "", fmt.Sprintf("base branch %q is not a branch name", base)
	}
	if _, err := g.output(ctx, "remote", "get-url", "origin"); err == nil {
		spec := "+refs/heads/" + base + ":refs/remotes/origin/" + base
		if _, err := g.output(ctx, "fetch", "--quiet", "--no-tags", "origin", spec); err != nil {
			return "", "the base " + base + " could not be fetched from origin: " + err.Error()
		}
		return "origin/" + base, ""
	}
	if _, err := g.output(ctx, "rev-parse", "--verify", "--quiet", base+"^{commit}"); err != nil {
		return "", "base branch " + base + " is missing"
	}
	return base, ""
}

type stagedEntry struct {
	state string
	sides [3]engine.FileSide
}

// conflicts lists the unmerged paths from git status --porcelain=v2, which names each
// path's state (UU, AA, AU, UA, DD, UD, DU) and its three stage blobs. A side git parked at
// a temporary name (path~HEAD, path~<sha>) because the types differed is folded back into
// the real path.
func (g Git) conflicts(ctx context.Context, mergeBase, theirs string) ([]engine.ConflictFile, error) {
	out, err := g.raw(ctx, nil, "status", "--porcelain=v2", "-z", "--untracked-files=no")
	if err != nil {
		return nil, err
	}
	entries := map[string]stagedEntry{}
	var order []string
	for _, entry := range splitZ(out) {
		if !strings.HasPrefix(entry, "u ") {
			continue
		}
		// u <XY> <sub> <m1> <m2> <m3> <mW> <h1> <h2> <h3> <path>
		fields := strings.SplitN(entry, " ", 11)
		if len(fields) != 11 {
			return nil, fmt.Errorf("git status printed %q", entry)
		}
		staged := stagedEntry{state: fields[1]}
		for i := range staged.sides {
			mode, sha := fields[3+i], fields[7+i]
			if strings.Trim(sha, "0") == "" {
				continue
			}
			side := engine.FileSide{Present: true, Mode: mode, SHA: sha}
			if err := g.readSide(ctx, &side); err != nil {
				return nil, err
			}
			staged.sides[i] = side
		}
		entries[fields[10]] = staged
		order = append(order, fields[10])
	}
	temps := map[string][]string{}
	for _, path := range order {
		real, ok := tempOf(path, theirs)
		if !ok {
			continue
		}
		temps[real] = append(temps[real], path)
	}
	var files []engine.ConflictFile
	seen := map[string]bool{}
	for _, path := range order {
		if _, isTemp := tempOf(path, theirs); isTemp {
			continue
		}
		seen[path] = true
		files = append(files, g.fileOf(ctx, path, entries[path], mergeBase, entries, temps[path]))
	}
	for _, path := range order {
		real, isTemp := tempOf(path, theirs)
		if !isTemp || seen[real] {
			continue
		}
		// Only the temporary name is unmerged: the other side is a directory at the real path.
		seen[real] = true
		files = append(files, g.fileOf(ctx, real, stagedEntry{}, mergeBase, entries, temps[real]))
	}
	return files, nil
}

func (g Git) fileOf(ctx context.Context, path string, staged stagedEntry, mergeBase string, entries map[string]stagedEntry, temps []string) engine.ConflictFile {
	file := engine.ConflictFile{Path: path, Base: staged.sides[0], Ours: staged.sides[1], Theirs: staged.sides[2]}
	for _, temp := range temps {
		other := entries[temp].sides
		targets := []*engine.FileSide{&file.Base, &file.Ours, &file.Theirs}
		for i, target := range targets {
			if !target.Present && other[i].Present {
				*target = other[i]
			}
		}
		file.Temps = append(file.Temps, temp)
	}
	renamed := false
	switch staged.state {
	case "AU", "UA", "DD":
		renamed = true
	case "UD", "DU":
		// A path the merge base never had was renamed on one side and deleted on the other.
		_, err := g.output(ctx, "cat-file", "-e", mergeBase+":"+path)
		renamed = err != nil
	}
	file.Kind = Kind(file, staged.state, renamed)
	return file
}

// tempOf maps git's temporary name for one side of a type conflict back to the real path.
func tempOf(path, theirs string) (string, bool) {
	i := strings.LastIndex(path, "~")
	if i <= 0 {
		return "", false
	}
	label := path[i+1:]
	if label == "HEAD" || (len(label) >= 7 && strings.HasPrefix(theirs, label)) {
		return path[:i], true
	}
	return "", false
}

func (g Git) readSide(ctx context.Context, side *engine.FileSide) error {
	if side.Mode == gitlink {
		// A submodule records a commit, not a blob: there is nothing to read.
		return nil
	}
	raw, err := g.blob(ctx, side.SHA, SideCap+1)
	if err != nil {
		return err
	}
	side.Binary = bytes.IndexByte(raw[:min(len(raw), 8000)], 0) >= 0
	if len(raw) > SideCap {
		raw, side.Truncated = raw[:SideCap], true
	}
	if !side.Binary {
		side.Text = string(raw)
	}
	return nil
}

// Kind names what kind of conflict one path has. A type conflict (a file against a symlink,
// a directory or a submodule) is other: only one side or the other can win.
func Kind(file engine.ConflictFile, state string, renamed bool) string {
	switch {
	case len(file.Temps) > 0 || file.Ours.Mode == gitlink || file.Theirs.Mode == gitlink || file.Base.Mode == gitlink:
		return "other"
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

// raw runs one git command with optional stdin and returns stdout untrimmed, for -z output.
func (g Git) raw(ctx context.Context, stdin []byte, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = g.Dir
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.Bytes(), nil
}

func splitZ(out []byte) []string {
	var items []string
	for _, item := range strings.Split(string(out), "\x00") {
		if item != "" {
			items = append(items, item)
		}
	}
	return items
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
