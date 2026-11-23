package adapt

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	branchpkg "span/internal/branch"
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
	label, ref, note := g.baseRef(ctx, base)
	if ref == "" {
		return engine.Preflight{Note: note}, nil
	}
	sha, err := g.hout(ctx, "rev-parse", ref+"^{commit}")
	if err != nil {
		return engine.Preflight{Note: err.Error()}, nil
	}
	out := engine.Preflight{Base: label, BaseSHA: sha}
	if _, err := g.hout(ctx, "merge-base", "--is-ancestor", sha, "HEAD"); err == nil {
		out.Checked = true
		return out, nil
	}
	cmd := g.command(ctx, "merge-tree", "--write-tree", "--name-only", "HEAD", sha)
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
	label, ref, note := g.baseRef(ctx, base)
	if ref == "" {
		return engine.Integration{}, errors.New(note)
	}
	sha, err := g.hout(ctx, "rev-parse", ref+"^{commit}")
	if err != nil {
		return engine.Integration{}, err
	}
	out := engine.Integration{Base: label, BaseSHA: sha}
	if _, err := g.hout(ctx, "merge-base", "--is-ancestor", sha, "HEAD"); err == nil {
		out.Clean = true
		return out, nil
	}
	mergeBase, err := g.hout(ctx, "merge-base", "HEAD", sha)
	if err != nil {
		return engine.Integration{}, err
	}
	_, mergeErr := g.hout(ctx, "merge", "--no-commit", "--no-ff", "--no-edit", sha)
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
	return g.hout(ctx, "status", "--porcelain", "--untracked-files=no", "--ignore-submodules=all", "--", ".", ":(exclude).span")
}

// Merging reports whether a merge is in progress in the checkout.
func (g Git) Merging(ctx context.Context) (bool, error) {
	path, err := g.hout(ctx, "rev-parse", "--git-path", "MERGE_HEAD")
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
	if g.Pin != nil {
		// A tampered repository gets no git merge --abort: span would run git on whatever the
		// agent pointed .git at.
		if err := g.Verify(*g.Pin); err != nil {
			return err
		}
	}
	merging, err := g.Merging(ctx)
	if err != nil || !merging {
		return err
	}
	if _, err := g.hout(ctx, "merge", "--abort"); err != nil {
		return err
	}
	left, err := g.trackedChanges(ctx)
	if err != nil || left == "" {
		return err
	}
	if _, err := g.hout(ctx, "reset", "-q"); err != nil {
		return err
	}
	if _, err := g.hout(ctx, "checkout", "--", "."); err != nil {
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
	// Children before their parents: a parent replaced by a symlink first would turn a later
	// delete of its child into a delete outside the checkout.
	sort.Slice(paths, func(i, j int) bool {
		di, dj := strings.Count(paths[i], "/"), strings.Count(paths[j], "/")
		if di != dj {
			return di > dj
		}
		return paths[i] < paths[j]
	})
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
		if err := g.safePath(path); err != nil {
			return err
		}
		var info fs.FileInfo
		var raw []byte
		err := g.inTree(func(root *os.Root) error {
			var err error
			if info, err = root.Lstat(filepath.FromSlash(path)); err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return errors.New("the regenerated path is not a regular file")
			}
			raw, err = root.ReadFile(filepath.FromSlash(path))
			return err
		})
		if errors.Is(err, fs.ErrNotExist) {
			return g.drop(ctx, path)
		}
		if err != nil {
			return err
		}
		if engine.HasConflictMarkers(raw) {
			return errors.New("conflict markers are still in the file")
		}
		sha, err := g.hout(ctx, "hash-object", "-w", "--", path)
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
	// Checked again here, right before the tree is touched: an earlier step may have changed
	// what lies on the way to this path.
	if err := g.safePath(path); err != nil {
		return err
	}
	rel := filepath.FromSlash(path)
	if mode != gitlink {
		if err := g.inTree(func(root *os.Root) error {
			if info, err := root.Lstat(rel); err == nil && info.IsDir() {
				g.removeEmptyDirs(root, path)
			}
			return nil
		}); err != nil {
			return err
		}
	}
	if _, err := g.hout(ctx, "update-index", "--add", "--cacheinfo", mode+","+sha+","+path); err != nil {
		return err
	}
	if mode == gitlink {
		// A submodule's commit is recorded; its checkout is the submodule's business.
		return g.inTree(func(root *os.Root) error { return root.MkdirAll(rel, 0o755) })
	}
	_, err = g.hout(ctx, "checkout-index", "-f", "--", path)
	return err
}

// drop removes path from the index and from the tree. A symlink is removed as a link; a
// directory is left alone. The tree is only touched when the path is, at this moment, inside
// the checkout: behind a symlinked parent there is nothing of ours to remove.
func (g Git) drop(ctx context.Context, path string) error {
	if _, err := g.hout(ctx, "update-index", "--force-remove", "--", path); err != nil {
		return err
	}
	if err := g.safePath(path); err != nil {
		return nil
	}
	return g.inTree(func(root *os.Root) error { return g.removeFile(root, path) })
}

// removeFile removes one file or link of the tree through the opened checkout, which refuses
// to leave it, and then the directories it emptied. A directory is left alone.
func (g Git) removeFile(root *os.Root, path string) error {
	rel := filepath.FromSlash(path)
	info, err := root.Lstat(rel)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}
	if err := root.Remove(rel); err != nil {
		return err
	}
	if parent := filepath.ToSlash(filepath.Dir(rel)); parent != "." {
		g.removeEmptyDirs(root, parent)
	}
	return nil
}

// inTree runs fn on the checkout opened as an os.Root, so no operation through it can leave
// the checkout, even through a symlink that appeared after a check.
func (g Git) inTree(fn func(root *os.Root) error) error {
	root, err := os.OpenRoot(g.Dir)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	return fn(root)
}

// removeEmptyDirs removes dir (slash-separated, relative to the checkout) and its parents
// while they are real, empty directories with no symlink on the way, stopping at the first
// that is not. A failure only means the directory stays.
func (g Git) removeEmptyDirs(root *os.Root, dir string) {
	for dir != "." && dir != "" {
		// The directory itself and every parent must be real directories, checked now.
		if g.safePath(dir+"/x") != nil {
			return
		}
		rel := filepath.FromSlash(dir)
		info, err := root.Lstat(rel)
		if err != nil || !info.IsDir() {
			return
		}
		names, err := listDir(root, rel)
		if err != nil || len(names) > 0 || root.Remove(rel) != nil {
			return
		}
		dir = filepath.ToSlash(filepath.Dir(rel))
	}
}

// FinishMerge commits the merge in progress and returns the new head. It refuses while any
// path is still unmerged.
func (g Git) FinishMerge(ctx context.Context, message string) (string, error) {
	left, err := g.hout(ctx, "ls-files", "-u")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(left) != "" {
		return "", errors.New("some paths are still unmerged")
	}
	// No hook runs on span's merge commit: an agent that wrote one cannot add a file to it.
	if _, err := g.hout(ctx, "commit", "--no-verify", "-m", message); err != nil {
		return "", err
	}
	return g.hout(ctx, "rev-parse", "HEAD")
}

// Parents lists HEAD's parents, first parent first.
func (g Git) Parents(ctx context.Context) ([]string, error) {
	out, err := g.hout(ctx, "rev-list", "--parents", "-n", "1", "HEAD")
	if err != nil {
		return nil, err
	}
	fields := strings.Fields(out)
	if len(fields) == 0 {
		return nil, errors.New("HEAD has no commit")
	}
	return fields[1:], nil
}

// IsBehindMessage reports a forge refusal because the change is behind its base: nothing was
// merged, and the change has to be updated first.
func IsBehindMessage(stderr string) bool {
	text := strings.ToLower(stderr)
	return strings.Contains(text, "not up to date with the base branch") || strings.Contains(text, "head branch is not up to date")
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
	cmd := g.command(ctx, "diff", "--no-index", "--no-color", "--no-ext-diff", "--no-textconv", "--", a, b)
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

// Snapshot reads the repository around the tree (HEAD, the branch it names, every local
// branch, the merge state files, hooks and config) and the tree itself: the index, the
// tracked paths whose tree differs from it, the untracked files, and the ignored ones.
func (g Git) Snapshot(ctx context.Context) (engine.TreeState, error) {
	state := engine.TreeState{
		Index: map[string]string{}, Dirty: map[string]bool{}, Untracked: map[string]bool{}, Ignored: map[string]bool{},
		Refs: map[string]string{}, Merge: map[string][]byte{}, Git: map[string]engine.GitFile{},
	}
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
	for set, args := range map[*map[string]bool][]string{
		&state.Dirty:     {"diff-files", "--ignore-submodules=all", "--name-only", "-z"},
		&state.Untracked: {"ls-files", "-o", "-z", "--exclude-standard"},
		&state.Ignored:   {"ls-files", "-o", "-i", "-z", "--exclude-standard", "--directory"},
	} {
		out, err := g.raw(ctx, nil, args...)
		if err != nil {
			return engine.TreeState{}, err
		}
		for _, path := range splitZ(out) {
			(*set)[path] = true
		}
	}
	if state.Head, err = g.hout(ctx, "rev-parse", "HEAD"); err != nil {
		return engine.TreeState{}, err
	}
	if symref, err := g.hout(ctx, "symbolic-ref", "-q", "HEAD"); err == nil {
		state.Symref = symref
	}
	refs, err := g.hout(ctx, "for-each-ref", "--format=%(refname) %(objectname)")
	if err != nil {
		return engine.TreeState{}, err
	}
	for _, line := range strings.Split(refs, "\n") {
		if name, sha, ok := strings.Cut(line, " "); ok {
			state.Refs[name] = sha
		}
	}
	if state.GitDir, err = g.hout(ctx, "rev-parse", "--absolute-git-dir"); err != nil {
		return engine.TreeState{}, err
	}
	common, err := g.hout(ctx, "rev-parse", "--git-common-dir")
	if err != nil {
		return engine.TreeState{}, err
	}
	if !filepath.IsAbs(common) {
		common = filepath.Join(g.Dir, common)
	}
	state.CommonDir = filepath.Clean(common)
	if g.Pin != nil {
		state.Pin = *g.Pin
	} else if state.Pin, err = g.PinRepo(ctx); err != nil {
		return engine.TreeState{}, err
	}
	if state.Git, err = readManaged(state.GitDir, state.CommonDir); err != nil {
		return engine.TreeState{}, err
	}
	for _, name := range mergeFiles {
		if file, ok := state.Git["git:"+name]; ok {
			state.Merge[name] = file.Data
		}
	}
	return state, nil
}

var mergeFiles = []string{"MERGE_HEAD", "MERGE_MSG", "MERGE_MODE"}

// The entries under the git directories span puts back after an agent ran: what would act on
// a later git command (config, hooks, info/attributes and info/exclude) and the merge state.
// "common" is the shared git directory, "git" this worktree's own.
var (
	managedFiles = [][2]string{
		{"common", "config"}, {"git", "config.worktree"}, {"git", "HEAD"}, {"git", "commondir"}, {"git", "gitdir"},
		{"common", "objects/info/alternates"}, {"git", "MERGE_HEAD"}, {"git", "MERGE_MSG"}, {"git", "MERGE_MODE"},
	}
	managedDirs = [][2]string{{"common", "hooks"}, {"common", "info"}}
)

func gitRoots(gitDir, commonDir string) (map[string]*os.Root, func(), error) {
	common, err := os.OpenRoot(commonDir)
	if err != nil {
		return nil, nil, err
	}
	own, err := os.OpenRoot(gitDir)
	if err != nil {
		_ = common.Close()
		return nil, nil, err
	}
	return map[string]*os.Root{"common": common, "git": own}, func() { _ = common.Close(); _ = own.Close() }, nil
}

// readManaged reads the managed entries without following any symlink out of the git
// directories: a link is recorded as a link.
func readManaged(gitDir, commonDir string) (map[string]engine.GitFile, error) {
	roots, done, err := gitRoots(gitDir, commonDir)
	if err != nil {
		return nil, err
	}
	defer done()
	files := map[string]engine.GitFile{}
	read := func(root *os.Root, key, rel string) (engine.GitFile, bool, error) {
		file, ok, err := readEntry(root, rel)
		if ok {
			files[key] = file
		}
		return file, ok, err
	}
	for _, item := range managedFiles {
		if _, _, err := read(roots[item[0]], item[0]+":"+item[1], item[1]); err != nil {
			return nil, err
		}
	}
	for _, item := range managedDirs {
		root := roots[item[0]]
		dir, ok, err := read(root, item[0]+":"+item[1], item[1])
		if err != nil {
			return nil, err
		}
		if !ok || !dir.Mode.IsDir() {
			continue
		}
		names, err := listDir(root, item[1])
		if err != nil {
			return nil, err
		}
		for _, name := range names {
			if _, _, err := read(root, item[0]+":"+item[1]+"/"+name, item[1]+"/"+name); err != nil {
				return nil, err
			}
		}
	}
	return files, nil
}

func readEntry(root *os.Root, rel string) (engine.GitFile, bool, error) {
	info, err := root.Lstat(rel)
	if errors.Is(err, fs.ErrNotExist) {
		return engine.GitFile{}, false, nil
	}
	if err != nil {
		return engine.GitFile{}, false, err
	}
	file := engine.GitFile{Mode: info.Mode()}
	switch {
	case info.Mode()&fs.ModeSymlink != 0:
		target, err := root.Readlink(rel)
		if err != nil {
			return engine.GitFile{}, false, err
		}
		file.Data = []byte(target)
	case info.Mode().IsRegular():
		if file.Data, err = root.ReadFile(rel); err != nil {
			return engine.GitFile{}, false, err
		}
	}
	return file, true, nil
}

// listDir names the entries of a directory already proven to be a real one.
func listDir(root *os.Root, rel string) ([]string, error) {
	dir, err := root.Open(rel)
	if err != nil {
		return nil, err
	}
	defer func() { _ = dir.Close() }()
	entries, err := dir.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}

// restoreManaged puts the managed entries back with plain file operations, before any git
// command runs: until then the agent's config, attributes and hooks are live. Every entry is
// Lstat'ed first; a symlink is removed as a link, a directory only after Lstat proved it real,
// and a file is written to a temporary name beside it and renamed over, so nothing is ever
// opened through a link.
func restoreManaged(before engine.TreeState) ([]string, error) {
	roots, done, err := gitRoots(before.GitDir, before.CommonDir)
	if err != nil {
		return nil, err
	}
	defer done()
	var changed []string
	put := func(root *os.Root, key, rel string) error {
		want, had := before.Git[key]
		moved, err := restoreEntry(root, rel, want, had)
		if moved {
			changed = append(changed, ".git/"+rel)
		}
		return err
	}
	for _, item := range managedFiles {
		if err := put(roots[item[0]], item[0]+":"+item[1], item[1]); err != nil {
			return nil, err
		}
	}
	for _, item := range managedDirs {
		root, key := roots[item[0]], item[0]+":"+item[1]
		if err := put(root, key, item[1]); err != nil {
			return nil, err
		}
		if dir, ok := before.Git[key]; !ok || !dir.Mode.IsDir() {
			continue
		}
		names, err := listDir(root, item[1])
		if err != nil {
			return nil, err
		}
		for _, name := range names {
			if _, ok := before.Git[key+"/"+name]; !ok {
				if err := put(root, key+"/"+name, item[1]+"/"+name); err != nil {
					return nil, err
				}
			}
		}
		for name := range before.Git {
			if child, ok := strings.CutPrefix(name, key+"/"); ok {
				if err := put(root, name, item[1]+"/"+child); err != nil {
					return nil, err
				}
			}
		}
	}
	sort.Strings(changed)
	return changed, nil
}

// restoreEntry makes rel what want says, or absent when had is false. It reports whether it
// changed anything.
func restoreEntry(root *os.Root, rel string, want engine.GitFile, had bool) (bool, error) {
	info, err := root.Lstat(rel)
	exists := err == nil
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	if exists && had && sameEntry(root, rel, info, want) {
		return false, nil
	}
	if !exists && !had {
		return false, nil
	}
	if exists {
		if info.IsDir() {
			// Lstat proved a real directory: RemoveAll cannot leave it through a link.
			err = root.RemoveAll(rel)
		} else {
			// A file or a link: the link itself is removed, never what it points at.
			err = root.Remove(rel)
		}
		if err != nil {
			return false, err
		}
	}
	if !had {
		return true, nil
	}
	switch {
	case want.Mode.IsDir():
		return true, root.Mkdir(rel, want.Mode.Perm())
	case want.Mode&fs.ModeSymlink != 0:
		return true, root.Symlink(string(want.Data), rel)
	default:
		return true, writeBeside(root, rel, want.Data, want.Mode.Perm())
	}
}

func sameEntry(root *os.Root, rel string, info fs.FileInfo, want engine.GitFile) bool {
	switch {
	case want.Mode.IsDir():
		return info.IsDir()
	case want.Mode&fs.ModeSymlink != 0:
		target, err := root.Readlink(rel)
		return info.Mode()&fs.ModeSymlink != 0 && err == nil && target == string(want.Data)
	default:
		if !info.Mode().IsRegular() || info.Mode().Perm() != want.Mode.Perm() {
			return false
		}
		data, err := root.ReadFile(rel)
		return err == nil && bytes.Equal(data, want.Data)
	}
}

// writeBeside writes data to a new temporary file next to rel and renames it over rel.
func writeBeside(root *os.Root, rel string, data []byte, perm fs.FileMode) error {
	var suffix [8]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return err
	}
	tmp := rel + ".span-" + hex.EncodeToString(suffix[:])
	f, err := root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	_, writeErr := f.Write(data)
	if err := errors.Join(writeErr, f.Close()); err != nil {
		_ = root.Remove(tmp)
		return err
	}
	if err := root.Chmod(tmp, perm); err != nil {
		_ = root.Remove(tmp)
		return err
	}
	return root.Rename(tmp, rel)
}

// Restore puts the repository back where before found it, except the paths in keep: hooks
// and config, local branches, HEAD, the merge state, the index, tracked files that differ,
// and new untracked files, which are removed. Ignored files are never deleted; the new ones
// are reported. It checks the result and fails when it cannot prove the repository is back.
func (g Git) Restore(ctx context.Context, before engine.TreeState, keep []string) (engine.Restored, error) {
	kept := map[string]bool{}
	for _, path := range keep {
		kept[path] = true
	}
	changed := map[string]bool{}
	// Nothing runs git until the repository is proven to be where it was: a .git that became
	// a gitfile or a symlink, or git directories that moved, fail closed.
	if err := verifyCore(before.Pin); err != nil {
		return engine.Restored{}, err
	}
	// Then, with no git command: the agent's config, attributes, hooks, HEAD, commondir and
	// alternates go away.
	gitChanged, err := restoreManaged(before)
	if err != nil {
		return engine.Restored{}, err
	}
	for _, name := range gitChanged {
		changed[name] = true
	}
	if err := g.Verify(before.Pin); err != nil {
		return engine.Restored{}, err
	}
	pin := before.Pin
	g.Pin = &pin
	now, err := g.Snapshot(ctx)
	if err != nil {
		return engine.Restored{}, err
	}
	if err := g.restoreRefs(ctx, before, now, changed); err != nil {
		return engine.Restored{}, err
	}
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
			return engine.Restored{}, err
		}
	}
	after, err := g.Snapshot(ctx)
	if err != nil {
		return engine.Restored{}, err
	}
	for path := range after.Dirty {
		if kept[path] || before.Dirty[path] || !strings.HasSuffix(strings.SplitN(before.Index[path], "\n", 2)[0], " 0") {
			continue
		}
		if err := g.safePath(path); err != nil {
			return engine.Restored{}, err
		}
		if _, err := g.hout(ctx, "checkout-index", "-f", "--", path); err != nil {
			return engine.Restored{}, err
		}
		changed[path] = true
	}
	for path := range after.Untracked {
		if kept[path] || before.Untracked[path] {
			continue
		}
		if err := g.safePath(path); err != nil {
			return engine.Restored{}, err
		}
		if err := g.inTree(func(root *os.Root) error { return g.removeFile(root, path) }); err != nil {
			return engine.Restored{}, err
		}
		changed[path] = true
	}
	var out engine.Restored
	for path := range after.Ignored {
		if !kept[path] && !before.Ignored[path] {
			out.Ignored = append(out.Ignored, path)
		}
	}
	sort.Strings(out.Ignored)
	for path := range changed {
		out.Paths = append(out.Paths, path)
	}
	sort.Strings(out.Paths)
	if err := g.provenBack(ctx, before, kept); err != nil {
		return out, err
	}
	return out, nil
}

func (g Git) restoreRefs(ctx context.Context, before, now engine.TreeState, changed map[string]bool) error {
	for name, sha := range before.Refs {
		if now.Refs[name] == sha {
			continue
		}
		if _, err := g.hout(ctx, "update-ref", "--no-deref", name, sha); err != nil {
			return err
		}
		changed[name] = true
	}
	for name := range now.Refs {
		if _, ok := before.Refs[name]; ok {
			continue
		}
		if _, err := g.hout(ctx, "update-ref", "--no-deref", "-d", name); err != nil {
			return err
		}
		changed[name] = true
	}
	switch {
	case before.Symref != "" && now.Symref != before.Symref:
		if _, err := g.hout(ctx, "symbolic-ref", "HEAD", before.Symref); err != nil {
			return err
		}
		changed["HEAD"] = true
	case before.Symref == "" && (now.Symref != "" || now.Head != before.Head):
		if _, err := g.hout(ctx, "update-ref", "--no-deref", "HEAD", before.Head); err != nil {
			return err
		}
		changed["HEAD"] = true
	}
	return nil
}

// provenBack compares the repository with before and says what still differs.
func (g Git) provenBack(ctx context.Context, before engine.TreeState, kept map[string]bool) error {
	now, err := g.Snapshot(ctx)
	if err != nil {
		return err
	}
	var differ []string
	if now.Head != before.Head || now.Symref != before.Symref {
		differ = append(differ, "HEAD")
	}
	for _, name := range mergeFiles {
		if !bytes.Equal(now.Merge[name], before.Merge[name]) {
			differ = append(differ, name)
		}
	}
	for name, sha := range before.Refs {
		if now.Refs[name] != sha {
			differ = append(differ, name)
		}
	}
	for name := range now.Refs {
		if _, ok := before.Refs[name]; !ok {
			differ = append(differ, name)
		}
	}
	for key, want := range before.Git {
		if got, ok := now.Git[key]; !ok || got.Mode != want.Mode || !bytes.Equal(got.Data, want.Data) {
			differ = append(differ, key)
		}
	}
	for key := range now.Git {
		if _, ok := before.Git[key]; !ok {
			differ = append(differ, key)
		}
	}
	for path, meta := range before.Index {
		if !kept[path] && now.Index[path] != meta {
			differ = append(differ, path)
		}
	}
	for path := range now.Index {
		if _, ok := before.Index[path]; !ok && !kept[path] {
			differ = append(differ, path)
		}
	}
	if len(differ) > 0 {
		sort.Strings(differ)
		return fmt.Errorf("the repository still differs from before the agent ran: %s", strings.Join(differ, ", "))
	}
	return nil
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
func (g Git) baseRef(ctx context.Context, base string) (string, string, string) {
	// The same check every base passes before it reaches git: never an option, HEAD or a
	// commit.
	if err := branchpkg.CheckBase(base); err != nil {
		return "", "", err.Error()
	}
	if _, err := g.hout(ctx, "remote", "get-url", "origin"); err == nil {
		// The same refresh worktree.StartPoint does: origin's copy only, never refs/heads.
		remote := "refs/remotes/origin/" + base
		if _, err := g.fetch(ctx, "--quiet", "--no-tags", "origin", "--", "+refs/heads/"+base+":"+remote); err != nil {
			return "", "", "the base " + base + " could not be fetched from origin: " + err.Error()
		}
		return "origin/" + base, remote, ""
	}
	local := "refs/heads/" + base
	if _, err := g.hout(ctx, "rev-parse", "--verify", "--quiet", local+"^{commit}"); err != nil {
		return "", "", "base branch " + base + " is missing"
	}
	return base, local, ""
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
		_, err := g.hout(ctx, "cat-file", "-e", mergeBase+":"+path)
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

// hardened are the options every git command of the conflict phase runs with. An agent ran
// in this checkout: a hook, an fsmonitor, a pager or an ext:: transport it configured must not
// run when span reads or writes the repository. The repository's own config and attributes
// are put back from the snapshot with plain file operations before any git command.
// A nested repository staged as a gitlink never starts git inside it.
var hardened = []string{
	"--no-optional-locks",
	"-c", "core.fsmonitor=false",
	"-c", "core.pager=cat",
	"-c", "protocol.ext.allow=never",
	"-c", "diff.ignoreSubmodules=all",
	"-c", "submodule.recurse=false",
	"-c", "status.submoduleSummary=false",
}

// command is one hardened git command of the conflict phase in the checkout. System config
// and attributes are off too, and git never prompts.
func (g Git) command(ctx context.Context, args ...string) *exec.Cmd {
	cmd := g.gitCmd(ctx, g.Dir, args...)
	// Repository hooks never run in the conflict phase: the agent may have written them. The
	// ordinary build, commit and push keep the project's own hooks.
	cmd.Args = append([]string{cmd.Args[0], "-c", "core.hooksPath=" + os.DevNull}, cmd.Args[1:]...)
	cmd.Env = append(cmd.Env, "GIT_CONFIG_NOSYSTEM=1", "GIT_ATTR_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0")
	return cmd
}

// gitCmd is every git command span runs: hardened options, and with a pin, the repository
// given explicitly so a .git the agent redirected is never followed. The person's global
// config (credential helpers, ssh) still applies, so a push authenticates as it always did.
func (g Git) gitCmd(ctx context.Context, dir string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", append(append([]string{}, hardened...), args...)...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	if g.Pin != nil {
		cmd.Env = append(cmd.Env, "GIT_DIR="+g.Pin.GitDir, "GIT_COMMON_DIR="+g.Pin.CommonDir, "GIT_WORK_TREE="+g.Pin.WorkTree)
	}
	return cmd
}

// WithPin is the same checkout with every git command pointed at the pinned repository.
func (g Git) WithPin(pin engine.RepoPin) engine.Integrator {
	g.Pin = &pin
	return g
}

// PinRepo records where the repository is and the shape of its git directory. It runs git,
// so it is taken before an agent runs, never after.
func (g Git) PinRepo(ctx context.Context) (engine.RepoPin, error) {
	plain := Git{Dir: g.Dir}
	work, err := filepath.Abs(g.Dir)
	if err != nil {
		return engine.RepoPin{}, err
	}
	pin := engine.RepoPin{WorkTree: work}
	if pin.GitDir, err = plain.hout(ctx, "rev-parse", "--absolute-git-dir"); err != nil {
		return engine.RepoPin{}, err
	}
	common, err := plain.hout(ctx, "rev-parse", "--git-common-dir")
	if err != nil {
		return engine.RepoPin{}, err
	}
	if !filepath.IsAbs(common) {
		common = filepath.Join(g.Dir, common)
	}
	pin.CommonDir = filepath.Clean(common)
	if pin.RealGitDir, err = filepath.EvalSymlinks(pin.GitDir); err != nil {
		return engine.RepoPin{}, err
	}
	if pin.RealCommonDir, err = filepath.EvalSymlinks(pin.CommonDir); err != nil {
		return engine.RepoPin{}, err
	}
	shape, err := readShape(pin)
	if err != nil {
		return engine.RepoPin{}, err
	}
	// git repacks refs on its own, and other runs of this repository add and move branches
	// and worktrees, so the bytes of packed-refs and the list of worktrees are not what this
	// run depends on. The values of the branches nobody has checked out are.
	shape.PackedRefs = ""
	if shape.Heads, err = quietHeads(shape); err != nil {
		return engine.RepoPin{}, err
	}
	return shape, nil
}

// headValues is the value of every local branch: loose files first, then the packed ones
// for the names a loose file did not give. A loose file wins over the packed line, as in
// git. The loose files are read first on purpose: git pack-refs copies a ref into the packed
// file and only then deletes the loose one, so a ref is always in at least one of the two
// when the second is read. The other order could read a stale packed file, then find the
// loose file already pruned, and report a ref that never moved as gone.
func headValues(commonDir string) (map[string]string, error) {
	values := map[string]string{}
	root := filepath.Join(commonDir, "refs", "heads")
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if errors.Is(walkErr, fs.ErrNotExist) {
				return nil
			}
			return walkErr
		}
		if entry.IsDir() || !entry.Type().IsRegular() {
			return nil
		}
		raw, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(commonDir, path)
		if err != nil {
			return err
		}
		values[filepath.ToSlash(rel)] = strings.TrimSpace(string(raw))
		return nil
	})
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(filepath.Join(commonDir, "packed-refs"))
	if errors.Is(err, fs.ErrNotExist) {
		return values, nil
	}
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		sum, name, ok := strings.Cut(strings.TrimSpace(line), " ")
		if !ok || strings.HasPrefix(sum, "#") || strings.HasPrefix(sum, "^") || !strings.HasPrefix(name, "refs/heads/") {
			continue
		}
		if _, loose := values[name]; !loose {
			values[name] = sum
		}
	}
	return values, nil
}

// checkedOut is every branch some worktree of the repository has checked out: the main
// one and each linked one.
func checkedOut(pin engine.RepoPin) (map[string]bool, error) {
	heads := []string{filepath.Join(pin.CommonDir, "HEAD"), filepath.Join(pin.GitDir, "HEAD")}
	linked, err := os.ReadDir(filepath.Join(pin.CommonDir, "worktrees"))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	for _, entry := range linked {
		heads = append(heads, filepath.Join(pin.CommonDir, "worktrees", entry.Name(), "HEAD"))
	}
	out := map[string]bool{}
	for _, path := range heads {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if name, ok := strings.CutPrefix(strings.TrimSpace(string(raw)), "ref: "); ok {
			out[name] = true
		}
	}
	return out, nil
}

// quietHeads is the value of every local branch no worktree has checked out.
func quietHeads(pin engine.RepoPin) (map[string]string, error) {
	values, err := headValues(pin.CommonDir)
	if err != nil {
		return nil, err
	}
	busy, err := checkedOut(pin)
	if err != nil {
		return nil, err
	}
	for name := range busy {
		delete(values, name)
	}
	if values == nil {
		values = map[string]string{}
	}
	return values, nil
}

// readShape fills the parts of a pin that are read with plain file operations.
func readShape(pin engine.RepoPin) (engine.RepoPin, error) {
	info, err := os.Lstat(filepath.Join(pin.WorkTree, ".git"))
	if err != nil {
		return engine.RepoPin{}, err
	}
	pin.DotGit = uint32(info.Mode().Type())
	pin.GitFile = ""
	if info.Mode().IsRegular() {
		raw, err := os.ReadFile(filepath.Join(pin.WorkTree, ".git"))
		if err != nil {
			return engine.RepoPin{}, err
		}
		pin.GitFile = string(raw)
	}
	optional := func(path string) (*string, error) {
		raw, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		text := string(raw)
		return &text, nil
	}
	if pin.Commondir, err = optional(filepath.Join(pin.GitDir, "commondir")); err != nil {
		return engine.RepoPin{}, err
	}
	if pin.Gitdir, err = optional(filepath.Join(pin.GitDir, "gitdir")); err != nil {
		return engine.RepoPin{}, err
	}
	if pin.Alternates, err = optional(filepath.Join(pin.CommonDir, "objects", "info", "alternates")); err != nil {
		return engine.RepoPin{}, err
	}
	pin.PackedRefs = ""
	if raw, err := os.ReadFile(filepath.Join(pin.CommonDir, "packed-refs")); err == nil {
		sum := sha256.Sum256(raw)
		pin.PackedRefs = hex.EncodeToString(sum[:])
	} else if !errors.Is(err, fs.ErrNotExist) {
		return engine.RepoPin{}, err
	}
	names := func(dir string) ([]string, error) {
		entries, err := os.ReadDir(dir)
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		var out []string
		for _, entry := range entries {
			out = append(out, entry.Name())
		}
		return out, err
	}
	if pin.Modules, err = names(filepath.Join(pin.CommonDir, "modules")); err != nil {
		return engine.RepoPin{}, err
	}
	if pin.Worktrees, err = names(filepath.Join(pin.CommonDir, "worktrees")); err != nil {
		return engine.RepoPin{}, err
	}
	return pin, nil
}

// verifyCore checks, without git, that .git and the pinned git directories are still the
// real directories they were: nothing may be written there otherwise.
func verifyCore(pin engine.RepoPin) error {
	dotGit := filepath.Join(pin.WorkTree, ".git")
	info, err := os.Lstat(dotGit)
	if err != nil || uint32(info.Mode().Type()) != pin.DotGit {
		return &engine.TamperedError{Path: dotGit}
	}
	if info.Mode().IsRegular() {
		raw, err := os.ReadFile(dotGit)
		if err != nil || string(raw) != pin.GitFile {
			return &engine.TamperedError{Path: dotGit}
		}
	}
	for path, real := range map[string]string{pin.GitDir: pin.RealGitDir, pin.CommonDir: pin.RealCommonDir} {
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() {
			return &engine.TamperedError{Path: path}
		}
		if now, err := filepath.EvalSymlinks(path); err != nil || now != real {
			return &engine.TamperedError{Path: path}
		}
	}
	return nil
}

// Verify checks the whole pinned shape without git: .git and the git directories, HEAD and
// the index as regular files, commondir, gitdir, alternates, packed-refs, and the modules and
// worktrees the repository had.
func (g Git) Verify(pin engine.RepoPin) error {
	if err := verifyCore(pin); err != nil {
		return err
	}
	for _, name := range []string{"HEAD", "index"} {
		path := filepath.Join(pin.GitDir, name)
		info, err := os.Lstat(path)
		if errors.Is(err, fs.ErrNotExist) && name == "index" {
			continue
		}
		if err != nil || !info.Mode().IsRegular() {
			return &engine.TamperedError{Path: path}
		}
	}
	now, err := readShape(pin)
	if err != nil {
		return &engine.TamperedError{Path: pin.GitDir}
	}
	same := func(a, b *string) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }
	switch {
	case !same(now.Commondir, pin.Commondir):
		return &engine.TamperedError{Path: filepath.Join(pin.GitDir, "commondir")}
	case !same(now.Gitdir, pin.Gitdir):
		return &engine.TamperedError{Path: filepath.Join(pin.GitDir, "gitdir")}
	case !same(now.Alternates, pin.Alternates):
		return &engine.TamperedError{Path: filepath.Join(pin.CommonDir, "objects", "info", "alternates")}
	case strings.Join(now.Modules, "\n") != strings.Join(pin.Modules, "\n"):
		return &engine.TamperedError{Path: filepath.Join(pin.CommonDir, "modules")}
	}
	if pin.Heads == nil {
		// A pin from before Heads existed: the whole file and the whole list, as it was.
		if now.PackedRefs != pin.PackedRefs {
			return &engine.TamperedError{Path: filepath.Join(pin.CommonDir, "packed-refs")}
		}
		if strings.Join(now.Worktrees, "\n") != strings.Join(pin.Worktrees, "\n") {
			return &engine.TamperedError{Path: filepath.Join(pin.CommonDir, "worktrees")}
		}
		return nil
	}
	return verifyHeads(pin, now)
}

// verifyHeads is the part of Verify that tolerates what other runs and git itself do to a
// shared repository: this run's own linked worktree must still be there, and no branch that
// was quiet at the pin may have moved or gone. Packing refs, new branches and other runs'
// worktrees change none of that.
func verifyHeads(pin, now engine.RepoPin) error {
	if rel, err := filepath.Rel(filepath.Join(pin.CommonDir, "worktrees"), pin.GitDir); err == nil && !strings.HasPrefix(rel, "..") && !strings.Contains(rel, string(filepath.Separator)) {
		found := false
		for _, name := range now.Worktrees {
			found = found || name == rel
		}
		if !found {
			return &engine.TamperedError{Path: filepath.Join(pin.CommonDir, "worktrees")}
		}
	}
	values, err := headValues(pin.CommonDir)
	if err != nil {
		return &engine.TamperedError{Path: filepath.Join(pin.CommonDir, "packed-refs")}
	}
	for name, want := range pin.Heads {
		if values[name] != want {
			return &engine.TamperedError{Path: filepath.Join(pin.CommonDir, filepath.FromSlash(name))}
		}
	}
	return nil
}

// hout runs one hardened git command and returns its trimmed stdout.
func (g Git) hout(ctx context.Context, args ...string) (string, error) {
	out, err := g.raw(ctx, nil, args...)
	return strings.TrimSpace(string(out)), err
}

// fetch is the one command of the conflict phase that reaches the network. It keeps the
// hardened options but not GIT_CONFIG_NOSYSTEM: on macOS the keychain credential helper is
// set only in the system config, so without it a private origin asks for a password that
// git is forbidden to prompt for, and the run holds. The system config is the machine's,
// not the repository's, so an agent that ran in the checkout cannot have written to it.
func (g Git) fetch(ctx context.Context, args ...string) (string, error) {
	cmd := g.command(ctx, append([]string{"fetch"}, args...)...)
	kept := cmd.Env[:0:0]
	for _, entry := range cmd.Env {
		if entry != "GIT_CONFIG_NOSYSTEM=1" {
			kept = append(kept, entry)
		}
	}
	cmd.Env = kept
	out, err := g.runCommand(cmd, nil, append([]string{"fetch"}, args...))
	return strings.TrimSpace(string(out)), err
}

// raw runs one hardened git command with optional stdin and returns stdout untrimmed, for -z
// output.
func (g Git) raw(ctx context.Context, stdin []byte, args ...string) ([]byte, error) {
	return g.runCommand(g.command(ctx, args...), stdin, args)
}

// runCommand runs a prepared git command and returns its stdout, or an error that carries
// git's stderr.
func (g Git) runCommand(cmd *exec.Cmd, stdin []byte, args []string) ([]byte, error) {
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
	cmd := g.command(ctx, "cat-file", "blob", sha)
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
