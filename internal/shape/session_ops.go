package shape

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// SettleProposals applies the person's decision on product-context notes. Accepted notes
// are written to the context file, skipped ones are dropped and can be restored while the
// session is open. A note already written cannot be skipped: the file is only appended to.
// Every id is checked before anything changes.
func (e *Engine) SettleProposals(workspace, id string, accept, skip, restore []string) (*Session, error) {
	s, err := loadOpen(workspace, id)
	if err != nil {
		return nil, err
	}
	for _, pid := range append(append(append([]string{}, accept...), skip...), restore...) {
		if findProposal(s, pid) == nil {
			return nil, fmt.Errorf("proposal %q does not exist", pid)
		}
	}
	for _, pid := range skip {
		if findProposal(s, pid).Status == ProposalAccepted {
			return nil, fail("shape_context_written", "proposal %s is already in the product context", pid)
		}
	}
	for _, pid := range skip {
		findProposal(s, pid).Status = ProposalSkipped
	}
	for _, pid := range restore {
		if p := findProposal(s, pid); p.Status == ProposalSkipped {
			p.Status = ProposalProposed
		}
	}
	if err := Save(s); err != nil {
		return nil, err
	}
	if len(accept) == 0 {
		return s, nil
	}
	return e.AcceptProposals(workspace, id, accept)
}

// Diff line kinds of a context preview.
const (
	// DiffContext is a line the file already has.
	DiffContext = "ctx"
	// DiffAdd is a line accepting would add.
	DiffAdd = "add"
)

// DiffLine is one line of a preview, numbered as in the file after the change.
type DiffLine struct {
	N    int    `json:"n"`
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// ContextDiff is what accepting some proposals would change in the product context file.
type ContextDiff struct {
	Path  string     `json:"path"`
	Lines []DiffLine `json:"lines"`
}

// ContextPreview shows, without writing, how the product context file changes when the
// proposals ids are accepted: each added line with one line of context around it.
func ContextPreview(workspace, id string, ids []string) (ContextDiff, error) {
	s, err := Load(workspace, id)
	if err != nil {
		return ContextDiff{}, err
	}
	raw, err := os.ReadFile(ContextPath(workspace))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return ContextDiff{}, fmt.Errorf("read context: %w", err)
	}
	merged := string(raw)
	for _, pid := range ids {
		p := findProposal(s, pid)
		if p == nil {
			return ContextDiff{}, fmt.Errorf("proposal %q does not exist", pid)
		}
		if p.Status != ProposalAccepted {
			merged = mergeContext(merged, p.Section, p.Text)
		}
	}
	rel, relErr := filepath.Rel(workspace, ContextPath(workspace))
	if relErr != nil {
		rel = ContextPath(workspace)
	}
	return ContextDiff{Path: filepath.ToSlash(rel), Lines: diffLines(string(raw), merged)}, nil
}

// diffLines lists the lines of next that are not in prev, with a line of context each side.
// The change only ever inserts lines, so one forward pass over both is enough.
func diffLines(prev, next string) []DiffLine {
	if prev == next {
		return []DiffLine{}
	}
	old := splitLines(prev)
	var all []DiffLine
	i := 0
	for j, text := range splitLines(next) {
		kind := DiffAdd
		if i < len(old) && old[i] == text {
			kind = DiffContext
			i++
		}
		all = append(all, DiffLine{N: j + 1, Kind: kind, Text: text})
	}
	keep := make([]bool, len(all))
	for k, l := range all {
		if l.Kind != DiffAdd {
			continue
		}
		for near := max(k-1, 0); near <= min(k+1, len(all)-1); near++ {
			keep[near] = true
		}
	}
	out := []DiffLine{}
	for k, l := range all {
		if keep[k] {
			out = append(out, l)
		}
	}
	return out
}

func splitLines(text string) []string {
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

// AcceptSplit replaces a split story with one linked session per proposed story. Each child
// starts from a copy of the example map, so what was answered is not asked again, and
// drafts its own story on its first turn. Accepting twice returns the same children.
func (e *Engine) AcceptSplit(workspace, id string) (*Session, error) {
	s, err := loadOpen(workspace, id)
	if err != nil {
		return nil, err
	}
	if len(s.Children) > 0 {
		return s, nil
	}
	if s.Status != StatusSplit || len(s.Split) == 0 {
		return nil, fail("shape_no_split", "session %s has no split to accept", id)
	}
	for _, item := range s.Split {
		child, err := newSession(StartInput{Workspace: s.Workspace, Repo: s.Repo, Idea: item.Story, Agent: s.Agent, Model: s.Model, SourceIssue: s.SourceIssue}, e.now())
		if err != nil {
			return nil, err
		}
		child.Map = copyMap(s.Map)
		child.Parent = s.ID
		if err := Save(child); err != nil {
			return nil, err
		}
		s.Children = append(s.Children, child.ID)
		// Each child is listed as soon as it exists, so a stop halfway leaves none orphaned;
		// only the last save makes the revision, so the history shows one accepted split.
		if err := persist(s); err != nil {
			return nil, err
		}
	}
	if err := Save(s); err != nil {
		return nil, err
	}
	return s, nil
}

// KeepWhole records that the person wants one story, then runs the turn again so the agent
// writes it. The split is not proposed again; a failed turn keeps the choice and the map.
func (e *Engine) KeepWhole(ctx context.Context, workspace, id string) (*Session, error) {
	s, err := loadOpen(workspace, id)
	if err != nil {
		return nil, err
	}
	if s.Status != StatusSplit || len(s.Children) > 0 {
		return nil, fail("shape_no_split", "session %s has no split to decline", id)
	}
	s.SplitKept, s.Split = true, []Split{}
	s.Status = computeStatus(s)
	if err := Save(s); err != nil {
		return nil, err
	}
	if err := e.turn(ctx, s); err != nil {
		return nil, recordFailure(s, e.now(), err)
	}
	if err := Save(s); err != nil {
		return nil, err
	}
	return s, nil
}

// Fork starts a new session from a sealed one, which stays as it is. The story and the
// answers carry over; assumptions reopen so the person can answer them this time.
func (e *Engine) Fork(workspace, id string) (*Session, error) {
	parent, err := Load(workspace, id)
	if err != nil {
		return nil, err
	}
	if parent.Status != StatusSealed {
		return nil, fail("shape_not_sealed", "session %s is not sealed", id)
	}
	return forkSession(parent, parent.Idea, parent.Map, parent.Story, e.now())
}

// forkSession starts a child of parent from the given idea, map and story. Every question
// the agent only assumed is reopened, so the person answers it for the new story.
func forkSession(parent *Session, idea string, from Map, story string, now time.Time) (*Session, error) {
	child, err := newSession(StartInput{Workspace: parent.Workspace, Repo: parent.Repo, Idea: idea, Agent: parent.Agent, Model: parent.Model, SourceIssue: parent.SourceIssue}, now)
	if err != nil {
		return nil, err
	}
	child.Map = copyMap(from)
	for i := range child.Map.Questions {
		if q := &child.Map.Questions[i]; q.Status == QuestionAssumed {
			q.Status, q.Answer = QuestionOpen, ""
		}
	}
	child.Story, child.Parent = story, parent.ID
	child.Status = computeStatus(child)
	if err := Save(child); err != nil {
		return nil, err
	}
	return child, nil
}

func copyMap(m Map) Map {
	return Map{
		Rules:     append([]Rule{}, m.Rules...),
		Examples:  append([]Example{}, m.Examples...),
		Questions: append([]Question{}, m.Questions...),
	}
}

// snippetRadius is how many lines a source snippet shows above and below the cited line.
const snippetRadius = 3

// maxSnippetFile is the largest file a snippet is cut from.
const maxSnippetFile = 2 << 20

// SnippetLine is one numbered line of a source snippet; Hit marks the cited line.
type SnippetLine struct {
	N    int    `json:"n"`
	Text string `json:"text"`
	Hit  bool   `json:"hit"`
}

// Snippet is the code around a "path:line" reference, read from the session's repository as
// it is now.
type Snippet struct {
	Path  string        `json:"path"`
	Line  int           `json:"line"`
	Lines []SnippetLine `json:"lines"`
}

// SourceSnippet reads the lines around ref ("path", "path:12" or "path:12-20") from the
// session's repository. The path must stay inside the repository, symlinks included.
func SourceSnippet(workspace, id, ref string) (Snippet, error) {
	s, err := Load(workspace, id)
	if err != nil {
		return Snippet{}, err
	}
	rel, line, err := parseRef(ref)
	if err != nil {
		return Snippet{}, err
	}
	root, err := filepath.EvalSymlinks(repoDir(s))
	if err != nil {
		return Snippet{}, fail("shape_source_not_found", "the repository cannot be read: %v", err)
	}
	full, err := filepath.EvalSymlinks(filepath.Join(root, rel))
	if err != nil || !within(root, full) {
		return Snippet{}, fail("shape_source_not_found", "%s is not a file of the repository", rel)
	}
	info, err := os.Stat(full)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxSnippetFile {
		return Snippet{}, fail("shape_source_not_found", "%s is not a readable file", rel)
	}
	raw, err := os.ReadFile(full)
	if err != nil {
		return Snippet{}, fmt.Errorf("read source: %w", err)
	}
	out := Snippet{Path: filepath.ToSlash(rel), Line: line, Lines: []SnippetLine{}}
	for n, text := range strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n") {
		n++
		if n >= line-snippetRadius && n <= line+snippetRadius {
			out.Lines = append(out.Lines, SnippetLine{N: n, Text: text, Hit: n == line})
		}
	}
	return out, nil
}

// parseRef splits "path:line" into a clean relative path and a line (1 when absent).
func parseRef(ref string) (string, int, error) {
	ref = strings.TrimSpace(ref)
	path, line := ref, 1
	if at := strings.LastIndex(ref, ":"); at > 0 {
		first, _, _ := strings.Cut(ref[at+1:], "-")
		if n, err := strconv.Atoi(first); err == nil && n > 0 {
			path, line = ref[:at], n
		}
	}
	path = filepath.Clean(filepath.FromSlash(path))
	if ref == "" || path == "." || filepath.IsAbs(path) || path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
		return "", 0, fail("invalid", "ref %q is not a path inside the repository", ref)
	}
	return path, line, nil
}

func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
