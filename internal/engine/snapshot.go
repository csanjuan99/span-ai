package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"span/internal/qa"
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
	snap.Statements = readStatements(dir)
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
