package adapt

import (
	"context"
	"fmt"
	"path"
	"strconv"
	"strings"
)

// largeLines is the number of changed lines past which a file is not shown as a diff by
// default: nobody reads that many lines in a browser, and rendering them would freeze it.
const largeLines = 5000

// FileInfo is one file of a change as a reader needs it before opening its diff: what
// happened to it, how much, and whether it is the kind of file a person reads line by line.
type FileInfo struct {
	Path string `json:"path"`
	// From is the old path of a renamed file, empty otherwise.
	From string `json:"from,omitempty"`
	// Status is A added, M modified, D deleted or R renamed.
	Status  string `json:"status"`
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
	// Binary is true when a line diff does not apply.
	Binary bool `json:"binary,omitempty"`
	// Generated is true for a file a tool wrote, judged by its name.
	Generated bool `json:"generated,omitempty"`
	// Large is true when the file changed more than largeLines lines.
	Large bool `json:"large,omitempty"`
}

// Files lists the files head changed since base with their status and counts, in path order
// as git prints them. It reads git diff --numstat and --name-status together so a rename is
// one entry that names where it came from.
func (g Git) Files(ctx context.Context, base, head string) ([]FileInfo, error) {
	if strings.TrimSpace(base) == "" || strings.TrimSpace(head) == "" {
		return []FileInfo{}, nil
	}
	if strings.HasPrefix(base, "-") || strings.HasPrefix(head, "-") {
		return nil, fmt.Errorf("files %q...%q: not a commit", base, head)
	}
	return g.filesOf(ctx, base+"..."+head)
}

// CommitFiles lists the files one commit changed against its first parent. A root commit
// is read against the empty tree.
func (g Git) CommitFiles(ctx context.Context, sha string) ([]FileInfo, error) {
	if err := checkRev(sha); err != nil {
		return nil, err
	}
	parent, err := g.parentOf(ctx, sha)
	if err != nil {
		return nil, err
	}
	return g.filesOf(ctx, parent+".."+sha)
}

// filesOf reads the two views of one range and joins them by path.
func (g Git) filesOf(ctx context.Context, spec string) ([]FileInfo, error) {
	numstat, err := g.output(ctx, "diff", "--numstat", "-z", "-M", spec, "--")
	if err != nil {
		return nil, err
	}
	names, err := g.output(ctx, "diff", "--name-status", "-z", "-M", spec, "--")
	if err != nil {
		return nil, err
	}
	statuses := parseNameStatus(names)
	files := []FileInfo{}
	for _, entry := range parseNumstat(numstat) {
		info := FileInfo{Path: entry.path, From: entry.from, Added: entry.added, Removed: entry.removed, Binary: entry.binary}
		info.Status = statuses[entry.path]
		if info.Status == "" {
			info.Status = "M"
		}
		info.Generated = IsGenerated(info.Path)
		info.Large = info.Added+info.Removed > largeLines
		files = append(files, info)
	}
	return files, nil
}

type numstatEntry struct {
	path, from     string
	added, removed int
	binary         bool
}

// parseNumstat reads `git diff --numstat -z`: "<added>\t<removed>\t<path>\0" and, for a
// rename, "<added>\t<removed>\t\0<old>\0<new>\0". A binary file counts "-" for both.
func parseNumstat(raw string) []numstatEntry {
	var entries []numstatEntry
	parts := strings.Split(raw, "\x00")
	for i := 0; i < len(parts); i++ {
		head := parts[i]
		if head == "" {
			continue
		}
		fields := strings.SplitN(head, "\t", 3)
		if len(fields) != 3 {
			continue
		}
		entry := numstatEntry{}
		if fields[0] == "-" || fields[1] == "-" {
			entry.binary = true
		} else {
			added, addErr := strconv.Atoi(fields[0])
			removed, removeErr := strconv.Atoi(fields[1])
			if addErr != nil || removeErr != nil {
				// Git prints a number or "-"; anything else is not a numstat line to trust.
				continue
			}
			entry.added, entry.removed = added, removed
		}
		if fields[2] == "" && i+2 < len(parts) {
			entry.from, entry.path = parts[i+1], parts[i+2]
			i += 2
		} else {
			entry.path = fields[2]
		}
		entries = append(entries, entry)
	}
	return entries
}

// parseNameStatus reads `git diff --name-status -z` into the letter of each path, keyed by
// the new path. A rename or copy is "R100\0old\0new\0" and counts as R.
func parseNameStatus(raw string) map[string]string {
	out := map[string]string{}
	parts := strings.Split(raw, "\x00")
	for i := 0; i < len(parts); i++ {
		code := parts[i]
		if code == "" {
			continue
		}
		letter := code[:1]
		switch letter {
		case "R", "C":
			if i+2 < len(parts) {
				out[parts[i+2]] = "R"
				i += 2
			}
		default:
			if i+1 < len(parts) {
				out[parts[i+1]] = letter
				i++
			}
		}
	}
	return out
}

// IsGenerated says whether a path names a file a tool wrote: protobuf and similar stubs,
// lockfiles, minified bundles and anything marked .gen. or .generated.
func IsGenerated(p string) bool {
	base := path.Base(p)
	for _, suffix := range []string{".pb.go", ".pb.cc", ".pb.h", "_pb2.py", "_pb2_grpc.py", ".min.js", ".min.css", ".map", "_generated.go", "_gen.go", ".gen.go"} {
		if strings.HasSuffix(base, suffix) {
			return true
		}
	}
	switch base {
	case "package-lock.json", "yarn.lock", "pnpm-lock.yaml", "go.sum", "Cargo.lock", "poetry.lock", "Gemfile.lock", "composer.lock":
		return true
	}
	return strings.Contains(base, ".generated.") || strings.Contains(base, ".gen.")
}

// CommitStat is one commit of a branch with the size of what it changed.
type CommitStat struct {
	Commit
	// Parent is the first parent, empty for a root commit.
	Parent string `json:"parent,omitempty"`
	Files  int    `json:"files"`
	Added  int    `json:"added"`
	// Removed is the number of removed lines.
	Removed int `json:"removed"`
	// RenameOnly is true when every file was renamed with no content change.
	RenameOnly bool `json:"renameOnly,omitempty"`
}

// Commits lists the commits head has that base does not, newest first, each with how many
// files and lines it changed.
func (g Git) Commits(ctx context.Context, base, head string) ([]CommitStat, error) {
	if strings.TrimSpace(base) == "" || strings.TrimSpace(head) == "" {
		return []CommitStat{}, nil
	}
	if strings.HasPrefix(base, "-") || strings.HasPrefix(head, "-") {
		return nil, fmt.Errorf("commits %q..%q: not a commit", base, head)
	}
	log, err := g.commitLog(ctx, base, head)
	if err != nil {
		return nil, err
	}
	stats := make([]CommitStat, 0, len(log))
	for i := len(log) - 1; i >= 0; i-- {
		stat, err := g.commitStat(ctx, log[i])
		if err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}
	return stats, nil
}

func (g Git) commitStat(ctx context.Context, commit Commit) (CommitStat, error) {
	stat := CommitStat{Commit: commit}
	parent, err := g.parentOf(ctx, commit.SHA)
	if err != nil {
		return stat, err
	}
	if parent != emptyTree {
		stat.Parent = parent
	}
	files, err := g.filesOf(ctx, parent+".."+commit.SHA)
	if err != nil {
		return stat, err
	}
	renamed := 0
	for _, file := range files {
		stat.Added += file.Added
		stat.Removed += file.Removed
		if file.Status == "R" && file.Added+file.Removed == 0 {
			renamed++
		}
	}
	stat.Files = len(files)
	stat.RenameOnly = len(files) > 0 && renamed == len(files)
	return stat, nil
}

// emptyTree is git's well-known empty tree, the parent a root commit is read against.
const emptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

// parentOf is the first parent of a commit, or the empty tree for a root commit.
func (g Git) parentOf(ctx context.Context, sha string) (string, error) {
	out, err := g.output(ctx, "rev-list", "--parents", "-n", "1", sha, "--")
	if err != nil {
		return "", fmt.Errorf("read commit %q: %w", sha, err)
	}
	fields := strings.Fields(out)
	if len(fields) < 2 {
		return emptyTree, nil
	}
	return fields[1], nil
}

// checkRev refuses text git could read as an option or that is not a plain revision name.
func checkRev(rev string) error {
	rev = strings.TrimSpace(rev)
	if rev == "" || strings.HasPrefix(rev, "-") || strings.ContainsAny(rev, " \t\n\\") {
		return fmt.Errorf("%q is not a commit", rev)
	}
	return nil
}

// CommitDiff is the diff one commit made against its first parent, bounded like the branch
// diff, so a reader sees what changed in it and nothing before it.
func (g Git) CommitDiff(ctx context.Context, sha string) (CommitDiffResult, error) {
	if err := checkRev(sha); err != nil {
		return CommitDiffResult{}, err
	}
	parent, err := g.parentOf(ctx, sha)
	if err != nil {
		return CommitDiffResult{}, err
	}
	spec := parent + ".." + sha
	stat, patch, truncated, err := g.statAndPatch(ctx, spec, "-M")
	if err != nil {
		return CommitDiffResult{}, err
	}
	files, err := g.filesOf(ctx, spec)
	if err != nil {
		return CommitDiffResult{}, err
	}
	res := CommitDiffResult{Diff: Diff{Stat: stat, Patch: patch, Truncated: truncated}, Files: files}
	if parent != emptyTree {
		res.Parent = parent
	}
	return res, nil
}

// CommitDiffResult is the diff of one commit with the files it touched.
type CommitDiffResult struct {
	Diff
	Parent string     `json:"parent,omitempty"`
	Files  []FileInfo `json:"files"`
}

// FileDiff is the diff of one path in a range, bounded. A path git does not know in that
// range gives an empty patch, and the caller decides if that is not found.
func (g Git) FileDiff(ctx context.Context, spec, file string) (string, bool, error) {
	if strings.ContainsAny(file, "\x00\n") || strings.HasPrefix(file, "-") || strings.HasPrefix(file, "/") || strings.Contains(file, "..") {
		return "", false, fmt.Errorf("%q is not a path in the change", file)
	}
	return g.bounded(ctx, patchLimit, "diff", "-M", spec, "--", file)
}

// Range is the spec for a branch diff, base...head, checked the way Diff checks it.
func Range(base, head string) (string, error) {
	if strings.TrimSpace(base) == "" || strings.TrimSpace(head) == "" {
		return "", fmt.Errorf("the run has no base or head to compare")
	}
	if err := checkRev(base); err != nil {
		return "", err
	}
	if err := checkRev(head); err != nil {
		return "", err
	}
	return base + "..." + head, nil
}

// CommitRange is the spec of what one commit changed: its first parent up to it.
func (g Git) CommitRange(ctx context.Context, sha string) (string, error) {
	if err := checkRev(sha); err != nil {
		return "", err
	}
	parent, err := g.parentOf(ctx, sha)
	if err != nil {
		return "", err
	}
	return parent + ".." + sha, nil
}
