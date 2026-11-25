package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/shape"
)

const readyTurn = `ok
{"story":"# Log in\n\nAs a user I log in."}`

func startOpen(t *testing.T, ws string) shape.View {
	t.Helper()
	code, out, _ := shapeRun(t, okTurn, "start", "--workspace", ws, "--idea", "Export invoices", "--json")
	var v shape.View
	if code != 0 || json.Unmarshal([]byte(out), &v) != nil {
		t.Fatalf("start %d %s", code, out)
	}
	return v
}

func TestShapeBoardActivityAndBulk(t *testing.T) {
	ws := shapeWorkspace(t)
	a := startOpen(t, ws)
	code, out, _ := shapeRun(t, okTurn, "board", "--workspace", ws, "--json")
	var view shape.BoardView
	if code != 0 || json.Unmarshal([]byte(out), &view) != nil || view.NeedYou != 1 || view.Columns[0].Count != 1 {
		t.Fatalf("board %d %s", code, out)
	}
	if _, out, _ = shapeRun(t, okTurn, "board", "--workspace", ws); !strings.Contains(out, "INTERVIEWING (1)") || !strings.Contains(out, a.Ref) {
		t.Fatalf("board text %q", out)
	}
	if code, out, _ = shapeRun(t, okTurn, "activity", "--workspace", ws, "--json"); code != 0 || !strings.Contains(out, a.Ref) {
		t.Fatalf("activity %d %s", code, out)
	}
	if code, _, _ = shapeRun(t, okTurn, "activity", "--workspace", ws, "--since", "yesterday"); code != shapeExitUsage {
		t.Fatalf("bad since %d", code)
	}
	if code, out, _ = shapeRun(t, okTurn, "bulk", "--workspace", ws, "assume", a.Ref); code != 0 || !strings.Contains(out, "nothing changed") {
		t.Fatalf("without --yes bulk only says what would change: %d %s", code, out)
	}
	if got := loaded(t, ws, a.ID); got.Map.Questions[0].Status != shape.QuestionOpen {
		t.Fatal("nothing changed")
	}
	if code, out, _ = shapeRun(t, okTurn, "bulk", "--workspace", ws, "--yes", "--json", "assume", a.Ref, "missing"); code != shapeExitFailed {
		t.Fatalf("a failing item exits 1: %d %s", code, out)
	}
	if got := loaded(t, ws, a.ID); got.Map.Questions[0].Status != shape.QuestionAssumed {
		t.Fatal("assumed")
	}
}

func TestShapeHistoryAndUndo(t *testing.T) {
	ws := shapeWorkspace(t)
	a := startOpen(t, ws)
	if code, _, _ := shapeRun(t, okTurn, "answer", "--workspace", ws, "--assume", "Q1", a.ID); code != 0 {
		t.Fatalf("assume %d", code)
	}
	code, out, _ := shapeRun(t, okTurn, "history", "--workspace", ws, "--actor", "you", "--kind", "assumptions", "--json", a.ID)
	var view shape.HistoryView
	if code != 0 || json.Unmarshal([]byte(out), &view) != nil || len(view.Events) != 1 {
		t.Fatalf("history %d %s", code, out)
	}
	step := view.Events[0].ID
	if code, _, _ = shapeRun(t, okTurn, "undo", "--workspace", ws, a.ID, step); code != shapeExitUsage {
		t.Fatalf("undo without --yes says what it would do and exits 2: %d", code)
	}
	if got := loaded(t, ws, a.ID); got.Map.Questions[0].Status != shape.QuestionAssumed {
		t.Fatal("still assumed")
	}
	if code, out, _ = shapeRun(t, okTurn, "undo", "--workspace", ws, "--yes", a.ID, step); code != 0 {
		t.Fatalf("undo %d %s", code, out)
	}
	if got := loaded(t, ws, a.ID); got.Map.Questions[0].Status != shape.QuestionOpen {
		t.Fatal("open again")
	}
	if code, _, _ = shapeRun(t, okTurn, "undo", "--workspace", ws, "--yes", a.ID, step); code != shapeExitRefused {
		t.Fatalf("a second undo is refused: %d", code)
	}
	if code, _, _ = shapeRun(t, okTurn, "undo", "--workspace", ws, "--yes", a.ID, "h99"); code != shapeExitNotFound {
		t.Fatalf("unknown step: %d", code)
	}
}

func TestShapeImportExportAndShare(t *testing.T) {
	ws := shapeWorkspace(t)
	file := filepath.Join(t.TempDir(), "story.md")
	if err := os.WriteFile(file, []byte(importedStory), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, out, _ := shapeRun(t, okTurn, "import", "--workspace", ws, "--file", file, "--dry-run", "--json"); code != 0 || !strings.Contains(out, "Tax is shown") {
		t.Fatalf("dry run %d %s", code, out)
	}
	if list := listed(t, ws); len(list) != 0 {
		t.Fatal("a dry run starts nothing")
	}
	code, out, _ := shapeRun(t, okTurn, "import", "--workspace", ws, "--file", file, "--no-run", "--json")
	var imported shape.View
	if code != 0 || json.Unmarshal([]byte(out), &imported) != nil || imported.ID == "" {
		t.Fatalf("import %d %s", code, out)
	}
	if code, _, _ = shapeRun(t, okTurn, "import", "--workspace", ws); code != shapeExitUsage {
		t.Fatalf("neither source: %d", code)
	}

	sealedRun := func() (string, error) { return readyTurn, nil }
	_, sout, _ := shapeRun(t, sealedRun, "start", "--workspace", ws, "--idea", "Log in", "--json")
	var s shape.View
	if err := json.Unmarshal([]byte(sout), &s); err != nil {
		t.Fatal(err)
	}
	if code, _, _ = shapeRun(t, sealedRun, "export", "--workspace", ws, s.ID); code != shapeExitRefused {
		t.Fatalf("export before sealing: %d", code)
	}
	if code, _, _ = shapeRun(t, sealedRun, "seal", "--workspace", ws, s.ID); code != 0 {
		t.Fatalf("seal %d", code)
	}
	target := filepath.Join(t.TempDir(), "out.json")
	if code, _, _ = shapeRun(t, sealedRun, "export", "--workspace", ws, "--format", "json", "--digest", "--out", target, s.ID); code != 0 {
		t.Fatalf("export %d", code)
	}
	if raw, err := os.ReadFile(target); err != nil || !strings.Contains(string(raw), "Log in") {
		t.Fatalf("%s %v", raw, err)
	}
	code, out, _ = shapeRun(t, sealedRun, "share", "--workspace", ws, "--json", "--days", "2", s.ID)
	var made struct {
		Link shape.Share `json:"link"`
		Path string      `json:"path"`
	}
	if code != 0 || json.Unmarshal([]byte(out), &made) != nil || made.Link.Token == "" {
		t.Fatalf("share %d %s", code, out)
	}
	if code, out, _ = shapeRun(t, sealedRun, "share", "--workspace", ws, "--open", made.Link.Token, s.ID); code != 0 || !strings.Contains(out, "Log in") {
		t.Fatalf("open %d %s", code, out)
	}
	if code, _, _ = shapeRun(t, sealedRun, "share", "--workspace", ws, "--revoke", made.Link.Token, s.ID); code != 0 {
		t.Fatalf("revoke %d", code)
	}
	if code, _, _ = shapeRun(t, sealedRun, "share", "--workspace", ws, "--open", made.Link.Token, s.ID); code != shapeExitRefused {
		t.Fatalf("revoked link: %d", code)
	}
}

const importedStory = "# Tax per line\n\n## Tax is shown on each line\n- 100 with 20% shows 20\n"

var _ = context.Background

// loaded reads a session back, failing the test when it cannot.
func loaded(t *testing.T, ws, id string) *shape.Session {
	t.Helper()
	got, err := shape.Load(ws, id)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// listed reads the sessions of a workspace, failing the test when it cannot.
func listed(t *testing.T, ws string) []*shape.Session {
	t.Helper()
	list, err := shape.List(ws)
	if err != nil {
		t.Fatal(err)
	}
	return list
}
