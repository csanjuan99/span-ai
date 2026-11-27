package engine

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"span/internal/qa"
	"span/internal/runlock"
	"span/internal/statement"
)

// Load reads the cursor, statements, acceptance lines, and gate for one run.
func Load(root, id string) (Snapshot, error) {
	if !validID(id) {
		return Snapshot{}, os.ErrNotExist
	}
	dir := filepath.Join(root, id)
	raw, err := os.ReadFile(filepath.Join(dir, "cursor.json"))
	if err != nil {
		return Snapshot{}, err
	}
	snap := Snapshot{Statements: []statement.Statement{}, Acceptance: []qa.Line{}}
	if err := json.Unmarshal(raw, &snap.Cursor); err != nil {
		return Snapshot{}, err
	}
	if interrupted(dir, snap.Cursor, filepath.Join(dir, "cursor.json")) {
		snap.Cursor.Hold = true
		snap.Cursor.Reason = interruptedReason
	}
	if !snap.Cursor.Hold && !snap.Cursor.Stop && !snap.Cursor.Done {
		_, err := os.Stat(filepath.Join(dir, pauseFile))
		logUnread(pauseFile, err)
		snap.PauseRequested = err == nil
	}
	snap.Label = Label(snap.Cursor)
	snap.Title = titleOf(dir, snap.Cursor.Issue)
	snap.Repo = RepoName(snap.Cursor.Home())
	snap.Statements = readStatements(dir)
	if choices := ReadStageAgents(dir); len(choices) > 0 {
		snap.StageAgents = choices
	}
	var lines []qa.Line
	if readOptionalJSON(filepath.Join(dir, "acceptance.json"), &lines) && lines != nil {
		snap.Acceptance = lines
	}
	var gate qa.Gate
	if readOptionalJSON(filepath.Join(dir, "qa-gate.json"), &gate) {
		snap.Gate = &gate
	}
	return snap, nil
}

// readOptionalJSON decodes a file a run may not have. A file that cannot be read or
// decoded leaves the run visible without it, and says so.
func readOptionalJSON(path string, into any) bool {
	body, err := os.ReadFile(path)
	if err != nil {
		logUnread(path, err)
		return false
	}
	if err := json.Unmarshal(body, into); err != nil {
		logUnread(path, err)
		return false
	}
	return true
}

// logUnread reports a read that failed for any reason other than the file not being there.
func logUnread(what string, err error) {
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("%s is left out of the run: %v", what, err)
	}
}

// interruptedReason is what a person reads when the process that drove a run ended without
// the run finishing or pausing.
const interruptedReason = "the run was interrupted: the process driving it ended. Resume to continue from this stage."

// interruptedAfter is how old the cursor must be before a run with no driver counts as
// interrupted. A run that was just opened has no lock yet for a moment.
const interruptedAfter = 10 * time.Second

// interrupted reports a run the cursor calls running while nobody holds its lock: a
// SIGKILL, a crash or a power cut stopped its process before it could pause. On disk such a
// run looks identical to a live one, so the page offered nothing to do with it.
func interrupted(dir string, cur Cursor, cursorFile string) bool {
	if cur.Hold || cur.Stop || cur.Done {
		return false
	}
	info, err := os.Stat(cursorFile)
	logUnread(cursorFile, err)
	if err != nil || time.Since(info.ModTime()) < interruptedAfter {
		return false
	}
	_, held := runlock.Probe(filepath.Join(dir, "lock"))
	return !held
}

func validID(id string) bool {
	return id != "" && id != "." && id != ".." && !strings.ContainsAny(id, `/\`)
}

func readStatements(dir string) []statement.Statement {
	entries, err := os.ReadDir(filepath.Join(dir, "iter"))
	if err != nil {
		logUnread(filepath.Join(dir, "iter"), err)
		return []statement.Statement{}
	}
	var out []statement.Statement
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		iter := filepath.Join(dir, "iter", entry.Name())
		files, err := os.ReadDir(iter)
		if err != nil {
			logUnread(iter, err)
			continue
		}
		for _, file := range files {
			name := file.Name()
			if file.IsDir() || !strings.HasSuffix(name, ".json") {
				continue
			}
			item, ok, err := statement.Read(iter, strings.TrimSuffix(name, ".json"))
			logUnread(filepath.Join(iter, name), err)
			if err != nil || !ok {
				continue
			}
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Iteration == out[j].Iteration {
			return out[i].Time.Before(out[j].Time)
		}
		return out[i].Iteration < out[j].Iteration
	})
	return out
}

// Label is how a person names a run without reading its id: story key, session id, and
// repository. A part the run does not have is skipped, so the label never has a gap.
func Label(cur Cursor) string {
	parts := make([]string, 0, 3)
	if key := strings.TrimSpace(cur.Issue); key != "" {
		parts = append(parts, key)
	}
	if cur.ID != "" {
		parts = append(parts, cur.ID)
	}
	if repo := RepoName(cur.Home()); repo != "" {
		parts = append(parts, repo)
	}
	return strings.Join(parts, "-")
}

// Home is the checkout a run belongs to, where its login and name come from. The code of
// an isolated run lives in a worktree of it; older runs only recorded the workspace.
func (c Cursor) Home() string {
	if c.Project != "" {
		return c.Project
	}
	return c.Workspace
}

// RepoName is the name of a run's repository folder, which is what tells two runs of the
// same story apart.
func RepoName(workspace string) string {
	clean := strings.TrimSpace(workspace)
	if clean == "" {
		return ""
	}
	if abs, err := filepath.Abs(clean); err == nil {
		clean = abs
	}
	base := filepath.Base(filepath.Clean(clean))
	if base == "." || base == string(filepath.Separator) {
		return ""
	}
	return base
}

// Head is what a list needs from a run: the label and where it stands. Reading the
// statements of every run to draw a list is not worth it.
type Head struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Issue   string `json:"issue,omitempty"`
	Repo    string `json:"repo,omitempty"`
	Stage   string `json:"stage"`
	Verdict string `json:"verdict"`
	// Title is the first line of the brief without the story key: what the run is about.
	Title string `json:"title,omitempty"`
	// Branch is the branch the run works on; Reason is why it is held, stopped or failing.
	Branch string `json:"branch,omitempty"`
	Reason string `json:"reason,omitempty"`
	// UpdatedAt is when the cursor last moved, in RFC 3339 and UTC.
	UpdatedAt string `json:"updatedAt,omitempty"`
	// StartedAt is when the run was opened, in RFC 3339 and UTC. A run that never recorded it has none.
	StartedAt string `json:"startedAt,omitempty"`
	// Need is what a held run waits for: conflict, questions or decision. Count says how many
	// files, questions or decisions. A run that waits for something else has neither.
	Need  string `json:"need,omitempty"`
	Count int    `json:"count,omitempty"`
}

// Heads reads the cursor of every run under root, newest id last.
func Heads(root string) []Head {
	entries, err := os.ReadDir(root)
	if err != nil {
		logUnread(root, err)
		return []Head{}
	}
	heads := make([]Head, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || !validID(entry.Name()) {
			continue
		}
		cur, err := readCursorFile(filepath.Join(root, entry.Name()))
		if err != nil {
			logUnread(filepath.Join(root, entry.Name()), err)
			continue
		}
		dir := filepath.Join(root, entry.Name())
		heads = append(heads, Head{
			ID: cur.ID, Label: Label(cur), Issue: cur.Issue,
			Repo: RepoName(cur.Home()), Stage: cur.Stage, Verdict: verdictOf(cur),
			Title: titleOf(dir, cur.Issue), Branch: cur.Branch, Reason: cur.Reason,
			UpdatedAt: movedAt(dir), StartedAt: startedAt(cur),
		}.withNeed(cur))
	}
	sort.Slice(heads, func(i, j int) bool { return heads[i].ID < heads[j].ID })
	return heads
}

func verdictOf(cur Cursor) string {
	switch {
	case cur.Done:
		return "done"
	case cur.Stop:
		return "stop"
	case cur.Hold:
		return "hold"
	default:
		return "running"
	}
}

// withNeed names what a held run waits for, so the desk can say "3 questions" and offer Answer
// instead of a generic Review. A run that is not held, or waits for something else, has none.
func (h Head) withNeed(cur Cursor) Head {
	if !cur.Hold || cur.Done || cur.Stop {
		return h
	}
	switch {
	case cur.Decision != nil:
		h.Need, h.Count = "decision", 1
	case cur.Conflict != nil:
		h.Need, h.Count = "conflict", conflictFiles(cur.Conflict)
	case len(cur.Questions) > 0:
		h.Need, h.Count = "questions", len(cur.Questions)
	}
	return h
}

// conflictFiles counts the files that conflict across every group.
func conflictFiles(c *Conflict) int {
	total := 0
	for _, group := range c.Groups {
		total += len(group.Files)
	}
	return total
}

// startedAt is when the run was opened. A cursor without it has no time.
func startedAt(cur Cursor) string {
	if cur.StartedAt.IsZero() {
		return ""
	}
	return cur.StartedAt.UTC().Format(time.RFC3339)
}

// titleOf is what the run is about: the first line of its brief, without the story key a
// Jira brief starts with. A run without a readable brief has no title.
func titleOf(dir, issue string) string {
	raw, err := os.ReadFile(filepath.Join(dir, "brief.md"))
	if err != nil {
		logUnread(filepath.Join(dir, "brief.md"), err)
		return ""
	}
	line, _, _ := strings.Cut(string(raw), "\n")
	line = strings.TrimSpace(line)
	if key := strings.TrimSpace(issue); key != "" {
		line = strings.TrimSpace(strings.TrimPrefix(line, key))
	}
	return line
}

// movedAt is when the run's cursor last changed. An unreadable cursor has no time.
func movedAt(dir string) string {
	info, err := os.Stat(filepath.Join(dir, "cursor.json"))
	if err != nil {
		logUnread(filepath.Join(dir, "cursor.json"), err)
		return ""
	}
	return info.ModTime().UTC().Format(time.RFC3339)
}
