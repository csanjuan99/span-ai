// Package implementation reads what a run changed and why: its files, commits, per-file
// diffs, the reasoning in its feature document and its audit chain. The web page and the
// span CLI both call it, so neither can know something the other cannot.
package implementation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"span/internal/adapt"
	"span/internal/branch"
	"span/internal/engine"
	"span/internal/fsutil"
	"span/internal/journal"
)

// RequestError is a failure of the request itself: the run, file, commit or decision asked
// for is not there, or the answer is not allowed. Code is the machine word the page and the
// CLI branch on ("not_found" or "invalid").
type RequestError struct{ Code, Msg string }

// Error is the message the person reads.
func (e *RequestError) Error() string { return e.Msg }

// SpanCode is the machine word the page and the CLI branch on.
func (e *RequestError) SpanCode() string { return e.Code }

func notFound(format string, args ...any) error {
	return &RequestError{Code: "not_found", Msg: fmt.Sprintf(format, args...)}
}

// Change is where the work of a run is read from: the repository and the two ends of its
// diff. A run that has no commit yet has none of them.
type Change struct {
	Git  adapt.Git
	Base string
	Head string
}

// Empty is true when there is nothing to compare yet.
func (c Change) Empty() bool { return c.Git.Dir == "" || c.Base == "" || c.Head == "" }

// RunChange finds where a run's work lives. A merged branch is already in its base, so its
// plain diff would be empty; the work is read from where the merge took it in. Failing to
// find that is not an error: the plain diff is still right for a run not merged that way.
func RunChange(ctx context.Context, root, id string) (Change, error) {
	snap, err := engine.Load(root, id)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Change{}, notFound("run %s does not exist", id)
		}
		return Change{}, err
	}
	cur := snap.Cursor
	if cur.Workspace == "" || cur.Base == "" || cur.GitSHA == "" {
		return Change{}, nil
	}
	git := adapt.Git{Dir: cur.Workspace}
	base := cur.Base
	if cur.Done {
		if before, ok, err := git.Landed(ctx, base, cur.GitSHA); err == nil && ok {
			base = before
		}
	}
	return Change{Git: git, Base: base, Head: cur.GitSHA}, nil
}

// FileView is a changed file with whether the person has marked it viewed on this run.
type FileView struct {
	adapt.FileInfo
	Viewed bool `json:"viewed"`
}

// ChangedFiles lists what the run changed with each file's status, counts, flags and viewed
// mark.
func ChangedFiles(ctx context.Context, root, id string) ([]FileView, error) {
	change, err := RunChange(ctx, root, id)
	if err != nil {
		return nil, err
	}
	if change.Empty() {
		return []FileView{}, nil
	}
	files, err := change.Git.Files(ctx, change.Base, change.Head)
	if err != nil {
		return nil, err
	}
	viewed, err := viewedSet(root, id)
	if err != nil {
		return nil, err
	}
	out := make([]FileView, 0, len(files))
	for _, file := range files {
		out = append(out, FileView{FileInfo: file, Viewed: viewed[file.Path]})
	}
	return out, nil
}

// RunCommits lists the commits of the run's branch, newest first, with their sizes.
func RunCommits(ctx context.Context, root, id string) ([]adapt.CommitStat, error) {
	change, err := RunChange(ctx, root, id)
	if err != nil {
		return nil, err
	}
	if change.Empty() {
		return []adapt.CommitStat{}, nil
	}
	return change.Git.Commits(ctx, change.Base, change.Head)
}

// RunCommitDiff is the diff of one commit of the run's branch. sha may be a unique prefix.
// A commit that is not on the branch is not found: this route reads the run's work only.
func RunCommitDiff(ctx context.Context, root, id, sha string) (adapt.CommitDiffResult, error) {
	commits, err := RunCommits(ctx, root, id)
	if err != nil {
		return adapt.CommitDiffResult{}, err
	}
	full, err := matchCommit(commits, sha)
	if err != nil {
		return adapt.CommitDiffResult{}, err
	}
	change, err := RunChange(ctx, root, id)
	if err != nil {
		return adapt.CommitDiffResult{}, err
	}
	return change.Git.CommitDiff(ctx, full)
}

// matchCommit resolves a full or abbreviated sha to one commit of the list.
func matchCommit(commits []adapt.CommitStat, sha string) (string, error) {
	sha = strings.ToLower(strings.TrimSpace(sha))
	if len(sha) < 4 {
		return "", &RequestError{Code: "invalid", Msg: "a commit needs at least 4 characters of its sha"}
	}
	found := ""
	for _, commit := range commits {
		if strings.HasPrefix(commit.SHA, sha) {
			if found != "" {
				return "", &RequestError{Code: "invalid", Msg: fmt.Sprintf("%s matches more than one commit", sha)}
			}
			found = commit.SHA
		}
	}
	if found == "" {
		return "", notFound("commit %s is not on this run's branch", sha)
	}
	return found, nil
}

// FileDiffView is the diff of one file of the run, either on the whole branch or in one
// commit, with the facts a reader needs to decide how to show it.
type FileDiffView struct {
	FileView
	// Commit is the commit the diff is limited to, empty for the whole branch.
	Commit    string `json:"commit,omitempty"`
	Patch     string `json:"patch"`
	Truncated bool   `json:"truncated"`
	// Hunks is the number of @@ hunks in the patch.
	Hunks int `json:"hunks"`
}

// RunFileDiff reads the diff of one changed file. commit may be empty (the whole branch
// against its base) or the sha of a commit on the branch. A path the change does not touch
// is not found. Binary, generated and very large files come back with their flags and no
// patch unless force is set, so a remote reader is not sent megabytes nobody reads.
func RunFileDiff(ctx context.Context, root, id, file, commit string, force bool) (FileDiffView, error) {
	change, err := RunChange(ctx, root, id)
	if err != nil {
		return FileDiffView{}, err
	}
	if change.Empty() {
		return FileDiffView{}, notFound("run %s has no change yet", id)
	}
	var spec string
	var files []adapt.FileInfo
	view := FileDiffView{}
	if commit == "" {
		if spec, err = adapt.Range(change.Base, change.Head); err != nil {
			return FileDiffView{}, err
		}
		files, err = change.Git.Files(ctx, change.Base, change.Head)
	} else {
		commits, listErr := RunCommits(ctx, root, id)
		if listErr != nil {
			return FileDiffView{}, listErr
		}
		if view.Commit, err = matchCommit(commits, commit); err != nil {
			return FileDiffView{}, err
		}
		if spec, err = change.Git.CommitRange(ctx, view.Commit); err != nil {
			return FileDiffView{}, err
		}
		files, err = change.Git.CommitFiles(ctx, view.Commit)
	}
	if err != nil {
		return FileDiffView{}, err
	}
	var info *adapt.FileInfo
	for i := range files {
		if files[i].Path == file {
			info = &files[i]
		}
	}
	if info == nil {
		return FileDiffView{}, notFound("%s is not in this change", file)
	}
	viewed, err := viewedSet(root, id)
	if err != nil {
		return FileDiffView{}, err
	}
	view.FileView = FileView{FileInfo: *info, Viewed: viewed[file]}
	if !force && (info.Binary || info.Generated || info.Large) {
		return view, nil
	}
	view.Patch, view.Truncated, err = change.Git.FileDiff(ctx, spec, file)
	if err != nil {
		return FileDiffView{}, err
	}
	view.Hunks = strings.Count(view.Patch, "\n@@ ")
	if strings.HasPrefix(view.Patch, "@@ ") {
		view.Hunks++
	}
	return view, nil
}

const viewedFile = "viewed.json"

type viewedStore struct {
	Paths []string `json:"paths"`
}

func viewedSet(root, id string) (map[string]bool, error) {
	if !validID(id) {
		return nil, notFound("run %s does not exist", id)
	}
	raw, err := os.ReadFile(filepath.Join(root, id, viewedFile))
	if errors.Is(err, os.ErrNotExist) {
		return map[string]bool{}, nil
	}
	if err != nil {
		return nil, err
	}
	var store viewedStore
	if err := json.Unmarshal(raw, &store); err != nil {
		return nil, fmt.Errorf("%s of run %s is damaged: %w", viewedFile, id, err)
	}
	set := make(map[string]bool, len(store.Paths))
	for _, path := range store.Paths {
		set[path] = true
	}
	return set, nil
}

// SetViewed marks a changed file viewed or not viewed on the run. The mark is the
// person's place in the list; it changes no code and the run does not wait for it. A path
// the change does not touch is not found.
func SetViewed(ctx context.Context, root, id, file string, viewed bool) error {
	files, err := ChangedFiles(ctx, root, id)
	if err != nil {
		return err
	}
	known := false
	for _, f := range files {
		known = known || f.Path == file
	}
	if !known {
		return notFound("%s is not in this change", file)
	}
	set, err := viewedSet(root, id)
	if err != nil {
		return err
	}
	if viewed {
		set[file] = true
	} else {
		delete(set, file)
	}
	paths := make([]string, 0, len(set))
	for path := range set {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	raw, err := json.Marshal(viewedStore{Paths: paths})
	if err != nil {
		return err
	}
	return fsutil.WriteAtomic(filepath.Join(root, id, viewedFile), raw)
}

// FeatureLimit bounds the feature document the report reads.
const FeatureLimit = 64 * 1024

// FeatureDocument reads odd/tasks/<feature>.md from the run's workspace. A run with no
// document yet, or a feature name that is not a plain file name, is an empty document and
// not an error. path is the repository-relative location, empty when none applies.
func FeatureDocument(root, id string) (path, markdown string, truncated bool, err error) {
	snap, err := engine.Load(root, id)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", "", false, notFound("run %s does not exist", id)
		}
		return "", "", false, err
	}
	feature := snap.Cursor.Feature
	if snap.Cursor.Workspace == "" || !branch.ValidFeature(feature) {
		return "", "", false, nil
	}
	rel := filepath.Join("odd", "tasks", feature+".md")
	file, err := os.Open(filepath.Join(snap.Cursor.Workspace, rel))
	if err != nil {
		return filepath.ToSlash(rel), "", false, nil
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, FeatureLimit+1))
	if err != nil {
		return "", "", false, err
	}
	if len(raw) > FeatureLimit {
		return filepath.ToSlash(rel), string(raw[:FeatureLimit]), true, nil
	}
	return filepath.ToSlash(rel), string(raw), false, nil
}

// Spec is one requirement of the story, quoted as written.
type Spec struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// Task is one line of the plan and the commit that holds it, when the line names one.
type Task struct {
	Text   string `json:"text"`
	Done   bool   `json:"done"`
	Commit string `json:"commit,omitempty"`
}

// DecisionView is a choice the agent says it made, with why, where it came from and
// whether a person has checked it.
type DecisionView struct {
	ID   string `json:"id"`
	What string `json:"what"`
	Why  string `json:"why"`
	// Source names the requirement the choice follows ("S2"). Empty means the story did not
	// say, so the choice is the agent's assumption.
	Source  string   `json:"source,omitempty"`
	Files   []string `json:"files,omitempty"`
	Assumed bool     `json:"assumed"`
	Checked bool     `json:"checked"`
	// CheckedAt is when a person checked it, RFC 3339.
	CheckedAt string `json:"checkedAt,omitempty"`
}

// ReasoningView is what a run's feature document says about why the change is as it is.
type ReasoningView struct {
	Title     string         `json:"title"`
	Path      string         `json:"path,omitempty"`
	Specs     []Spec         `json:"specs"`
	Tasks     []Task         `json:"tasks"`
	Decisions []DecisionView `json:"decisions"`
	// Unchecked is how many assumed decisions no person has checked yet.
	Unchecked int `json:"unchecked"`
	// Recorded is true when the document has a Log section, so an empty decision list means
	// the agent recorded none and not that nothing was read.
	Recorded bool `json:"recorded"`
}

var (
	specID     = regexp.MustCompile(`^((?:S|AC)\d+)\b[\s:.\-–—]*`)
	commitText = regexp.MustCompile(`(?i)\b([0-9a-f]{7,40})\b\s*$`)
	sourceTail = regexp.MustCompile(`\[source:\s*([^;\]]*?)\s*(?:;\s*files:\s*([^\]]*?))?\s*\]\s*$`)
	bulletMark = regexp.MustCompile(`^\s*(?:[-*]\s+|\d+[.)]\s+)?`)
	becauseCut = regexp.MustCompile(`(?i)\s(?:because|porque)\s`)
)

// ParseReasoning reads the feature document the build stage leaves behind. A shape it does
// not know gives empty lists and never an error.
func ParseReasoning(markdown string) ReasoningView {
	out := ReasoningView{Specs: []Spec{}, Tasks: []Task{}, Decisions: []DecisionView{}}
	section := ""
	for _, raw := range strings.Split(markdown, "\n") {
		if heading := strings.TrimSpace(raw); strings.HasPrefix(heading, "## ") {
			section = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(heading, "## ")))
			if section == "log" {
				out.Recorded = true
			}
			continue
		}
		if title, ok := strings.CutPrefix(strings.TrimSpace(raw), "# "); ok && out.Title == "" {
			out.Title = stripMarks(title)
			continue
		}
		line := strings.TrimSpace(bulletMark.ReplaceAllString(raw, ""))
		if line == "" {
			continue
		}
		plain := stripMarks(line)
		if kind, body, ok := strings.Cut(plain, ":"); ok && strings.EqualFold(strings.TrimSpace(kind), "decision") {
			out.Decisions = append(out.Decisions, parseDecision(len(out.Decisions)+1, strings.TrimSpace(body)))
			continue
		}
		switch section {
		case "specs":
			spec := Spec{ID: fmt.Sprintf("S%d", len(out.Specs)+1), Text: plain}
			if m := specID.FindStringSubmatch(plain); m != nil {
				spec.ID, spec.Text = m[1], strings.TrimSpace(plain[len(m[0]):])
			}
			out.Specs = append(out.Specs, spec)
		case "tasks":
			out.Tasks = append(out.Tasks, parseTask(line))
		}
	}
	return out
}

func stripMarks(text string) string {
	return strings.TrimSpace(strings.NewReplacer("**", "", "__", "").Replace(text))
}

func parseTask(line string) Task {
	task := Task{Text: stripMarks(line)}
	if len(line) >= 3 && line[0] == '[' && line[2] == ']' {
		task.Done = line[1] != ' '
		task.Text = stripMarks(strings.TrimSpace(line[3:]))
	}
	if m := commitText.FindStringSubmatch(task.Text); m != nil {
		task.Commit = strings.ToLower(m[1])
		task.Text = strings.TrimSpace(strings.TrimRight(strings.TrimSpace(task.Text[:len(task.Text)-len(m[0])]), "·|-–—, "))
	}
	return task
}

func parseDecision(n int, body string) DecisionView {
	d := DecisionView{ID: fmt.Sprintf("D%d", n)}
	if m := sourceTail.FindStringSubmatch(body); m != nil {
		d.Source = strings.TrimSpace(m[1])
		for _, file := range strings.Split(m[2], ",") {
			if file = strings.TrimSpace(file); file != "" {
				d.Files = append(d.Files, file)
			}
		}
		body = strings.TrimSpace(body[:len(body)-len(m[0])])
	}
	d.What = body
	if at := becauseCut.FindStringIndex(body); at != nil {
		d.What = strings.TrimSpace(body[:at[0]])
		d.Why = strings.TrimSpace(body[at[1]:])
	}
	d.Assumed = d.Source == ""
	return d
}

const decisionsFile = "decisions.json"

type decisionStore struct {
	Checked map[string]string `json:"checked"`
}

func readDecisionStore(root, id string) (decisionStore, error) {
	store := decisionStore{Checked: map[string]string{}}
	raw, err := os.ReadFile(filepath.Join(root, id, decisionsFile))
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return store, err
	}
	if err := json.Unmarshal(raw, &store); err != nil {
		return store, fmt.Errorf("%s of run %s is damaged: %w", decisionsFile, id, err)
	}
	if store.Checked == nil {
		store.Checked = map[string]string{}
	}
	return store, nil
}

// RunReasoning reads the run's feature document and joins it with the decisions a person
// has checked.
func RunReasoning(root, id string) (ReasoningView, error) {
	path, markdown, _, err := FeatureDocument(root, id)
	if err != nil {
		return ReasoningView{}, err
	}
	view := ParseReasoning(markdown)
	view.Path = path
	store, err := readDecisionStore(root, id)
	if err != nil {
		return ReasoningView{}, err
	}
	for i := range view.Decisions {
		if at, ok := store.Checked[view.Decisions[i].ID]; ok {
			view.Decisions[i].Checked, view.Decisions[i].CheckedAt = true, at
		}
		if view.Decisions[i].Assumed && !view.Decisions[i].Checked {
			view.Unchecked++
		}
	}
	return view, nil
}

// CheckDecisions marks assumed decisions as checked by a person, saved on the run and
// recorded in its journal. It changes no code. A decision that does not exist, or that the
// story already settled, is refused so a mark always means "I looked at an assumption".
func CheckDecisions(root, id string, ids []string) error {
	if len(ids) == 0 {
		return &RequestError{Code: "invalid", Msg: "name at least one decision to check"}
	}
	view, err := RunReasoning(root, id)
	if err != nil {
		return err
	}
	byID := map[string]DecisionView{}
	for _, d := range view.Decisions {
		byID[d.ID] = d
	}
	store, err := readDecisionStore(root, id)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, name := range ids {
		d, ok := byID[name]
		if !ok {
			return notFound("decision %s is not in this run", name)
		}
		if !d.Assumed {
			return &RequestError{Code: "invalid", Msg: fmt.Sprintf("decision %s follows the story, so there is nothing to check", name)}
		}
		store.Checked[name] = now
	}
	raw, err := json.Marshal(store)
	if err != nil {
		return err
	}
	if err := fsutil.WriteAtomic(filepath.Join(root, id, decisionsFile), raw); err != nil {
		return err
	}
	return journal.Append(filepath.Join(root, id), journal.Person, "decisions.checked", map[string]string{"ids": strings.Join(ids, ",")})
}

// AuditEntry is one journal line with a plain sentence for what happened.
type AuditEntry struct {
	journal.Entry
	What string `json:"what"`
}

// AuditView is the run's journal and whether its hash chain holds.
type AuditView struct {
	Entries []AuditEntry `json:"entries"`
	// Intact is true when every entry links to the one before it.
	Intact bool `json:"intact"`
	// BrokenAt is the seq of the first entry that does not link, 0 when intact.
	BrokenAt int `json:"brokenAt"`
	// Head is the hash of the last entry as stored.
	Head string `json:"head,omitempty"`
}

// RunAudit reads the run's journal with each entry's hash and the verdict on its chain. An
// entry after a broken link is returned as stored and must not be trusted.
func RunAudit(root, id string) (AuditView, error) {
	if !validID(id) {
		return AuditView{}, notFound("run %s does not exist", id)
	}
	if _, err := engine.Load(root, id); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return AuditView{}, notFound("run %s does not exist", id)
		}
		return AuditView{}, err
	}
	entries, broken, err := journal.Chain(filepath.Join(root, id))
	if err != nil {
		return AuditView{}, err
	}
	view := AuditView{Entries: make([]AuditEntry, 0, len(entries)), Intact: broken == 0, BrokenAt: broken}
	for _, entry := range entries {
		view.Entries = append(view.Entries, AuditEntry{Entry: entry, What: describeEvent(entry.Event)})
	}
	if len(entries) > 0 {
		view.Head = entries[len(entries)-1].Hash
	}
	return view, nil
}

// describeEvent turns a journal event into the sentence the audit view shows.
func describeEvent(event journal.Event) string {
	what := strings.ReplaceAll(event.Type, ".", " ")
	if stage := event.Detail["stage"]; stage != "" {
		what += " " + stage
	}
	keys := make([]string, 0, len(event.Detail))
	for key := range event.Detail {
		if key != "stage" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		what += " · " + key + " " + event.Detail[key]
	}
	return what
}

func validID(id string) bool {
	return id != "" && id != "." && id != ".." && !strings.ContainsAny(id, `/\`)
}
