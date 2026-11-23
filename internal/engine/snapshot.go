package engine

import (
	"encoding/json"
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
	snap.Label = Label(snap.Cursor)
	snap.Statements = readStatements(dir)
	if choices := ReadStageAgents(dir); len(choices) > 0 {
		snap.StageAgents = choices
	}
	if body, err := os.ReadFile(filepath.Join(dir, "acceptance.json")); err == nil {
		_ = json.Unmarshal(body, &snap.Acceptance)
	}
	if snap.Acceptance == nil {
		snap.Acceptance = []qa.Line{}
	}
	if body, err := os.ReadFile(filepath.Join(dir, "qa-gate.json")); err == nil {
		var gate qa.Gate
		if json.Unmarshal(body, &gate) == nil {
			snap.Gate = &gate
		}
	}
	return snap, nil
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
			continue
		}
		for _, file := range files {
			name := file.Name()
			if file.IsDir() || !strings.HasSuffix(name, ".json") {
				continue
			}
			item, ok, err := statement.Read(iter, strings.TrimSuffix(name, ".json"))
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
	if repo := RepoName(cur.Workspace); repo != "" {
		parts = append(parts, repo)
	}
	return strings.Join(parts, "-")
}

// RepoName is the folder a run writes in, which is what tells two runs of the same story
// apart.
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
}

// Heads reads the cursor of every run under root, newest id last.
func Heads(root string) []Head {
	entries, err := os.ReadDir(root)
	if err != nil {
		return []Head{}
	}
	heads := make([]Head, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || !validID(entry.Name()) {
			continue
		}
		cur, err := readCursorFile(filepath.Join(root, entry.Name()))
		if err != nil {
			continue
		}
		heads = append(heads, Head{
			ID: cur.ID, Label: Label(cur), Issue: cur.Issue,
			Repo: RepoName(cur.Workspace), Stage: cur.Stage, Verdict: verdictOf(cur),
		})
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
