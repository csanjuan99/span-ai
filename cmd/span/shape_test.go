package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"span/internal/jira"
	"span/internal/shape"
)

type fakeIssues struct{ story jira.Story }

func (f fakeIssues) Issue(context.Context, string) (jira.Story, error) { return f.story, nil }

const openTurn = `ok
{"map":{"rules":[{"id":"R1","text":"r","source":{"kind":"prompt","ref":"idea"}}],"examples":[],"questions":[{"id":"Q1","text":"who?","assume":"all","blocks":"R1"}]},"split":[],"proposals":[],"story":""}`

// shapeRun runs one verb with the interview agent replaced by reply.
func shapeRun(t *testing.T, reply func() (string, error), args ...string) (int, string, string) {
	t.Helper()
	before := shapeAgents
	shapeAgents = func(string, string, string) shape.Runner {
		return func(context.Context, string, string) (string, error) { return reply() }
	}
	t.Cleanup(func() { shapeAgents = before })
	var out, errw bytes.Buffer
	code := shapeCmd(context.Background(), args, &out, &errw)
	return code, out.String(), errw.String()
}

func shapeWorkspace(t *testing.T) string {
	t.Helper()
	ws := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = ws
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	return ws
}

func okTurn() (string, error) { return openTurn, nil }

func TestShapeStartListShowFilterArchiveDelete(t *testing.T) {
	ws := shapeWorkspace(t)
	code, out, _ := shapeRun(t, okTurn, "start", "--workspace", ws, "--idea", "Export invoices as CSV", "--agent", "codex", "--json")
	var started shape.View
	if code != 0 || json.Unmarshal([]byte(out), &started) != nil || started.Agent != "codex" || started.OpenQuestions != 1 || !strings.HasPrefix(started.Ref, "SES-") {
		t.Fatalf("start %d %s", code, out)
	}
	code, out, _ = shapeRun(t, okTurn, "list", "--workspace", ws, "--query", "csv", "--status", "interviewing", "--json")
	var rows []shape.View
	if code != 0 || json.Unmarshal([]byte(out), &rows) != nil || len(rows) != 1 || rows[0].ID != started.ID {
		t.Fatalf("list %d %s", code, out)
	}
	if _, out, _ = shapeRun(t, okTurn, "list", "--workspace", ws, "--query", "nope"); strings.TrimSpace(out) != "" {
		t.Fatalf("the filter kept %q", out)
	}
	if code, _, _ = shapeRun(t, okTurn, "archive", "--workspace", ws, started.ID); code != 0 {
		t.Fatalf("archive %d", code)
	}
	if _, out, _ = shapeRun(t, okTurn, "list", "--workspace", ws); strings.TrimSpace(out) != "" {
		t.Fatalf("archived session listed: %q", out)
	}
	if code, _, _ = shapeRun(t, okTurn, "archive", "--workspace", ws, "--undo", started.ID); code != 0 {
		t.Fatalf("unarchive %d", code)
	}
	if code, _, _ = shapeRun(t, okTurn, "delete", "--workspace", ws, started.ID); code != shapeExitUsage {
		t.Fatalf("delete without --yes must be a usage error, got %d", code)
	}
	if code, _, _ = shapeRun(t, okTurn, "delete", "--workspace", ws, "--yes", started.ID); code != 0 {
		t.Fatalf("delete %d", code)
	}
	if code, _, _ = shapeRun(t, okTurn, "show", "--workspace", ws, started.ID); code != shapeExitNotFound {
		t.Fatalf("show after delete: %d", code)
	}
}

func TestShapeResumeAfterAFailureExitsFiveThenClears(t *testing.T) {
	ws := shapeWorkspace(t)
	_, out, _ := shapeRun(t, okTurn, "start", "--workspace", ws, "--idea", "x", "--json")
	var s shape.View
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		t.Fatal(err)
	}
	limit := func() (string, error) { return "", errors.New("usage limit reached") }
	if code, _, _ := shapeRun(t, limit, "resume", "--workspace", ws, s.ID); code != shapeExitAgent {
		t.Fatalf("failed resume exits %d", code)
	}
	code, out, _ := shapeRun(t, okTurn, "show", "--workspace", ws, "--json", s.ID)
	var shown shape.View
	if code != 0 || json.Unmarshal([]byte(out), &shown) != nil || shown.Failure == nil || shown.Failure.Kind != shape.FailureUsageLimit {
		t.Fatalf("failure not shown: %d %s", code, out)
	}
	code, out, _ = shapeRun(t, okTurn, "resume", "--workspace", ws, "--agent", "codex", "--json", s.ID)
	var resumed shape.View
	if code != 0 || json.Unmarshal([]byte(out), &resumed) != nil || resumed.Failure != nil || resumed.Agent != "codex" {
		t.Fatalf("resume %d %s", code, out)
	}
}

func TestShapeStartFromAnIssueKeyAndUsageErrors(t *testing.T) {
	ws := shapeWorkspace(t)
	before := shapeIssues
	shapeIssues = func(string) (shape.IssueReader, error) {
		return fakeIssues{jira.Story{Key: "BILL-2031", Brief: "Show tax per line"}}, nil
	}
	t.Cleanup(func() { shapeIssues = before })
	code, out, _ := shapeRun(t, okTurn, "start", "--workspace", ws, "--issue", "bill-2031", "--json")
	var s shape.View
	if code != 0 || json.Unmarshal([]byte(out), &s) != nil || s.SourceIssue != "BILL-2031" || !strings.Contains(s.Idea, "Show tax per line") {
		t.Fatalf("start from issue %d %s", code, out)
	}
	for _, args := range [][]string{{}, {"unknown"}, {"start", "--workspace", ws}, {"show", "--workspace", ws}, {"agent", "--workspace", ws, s.ID}} {
		if code, _, _ := shapeRun(t, okTurn, args...); code != shapeExitUsage {
			t.Errorf("%v exits %d, want usage", args, code)
		}
	}
	if code, _, _ := shapeRun(t, okTurn, "list", "--workspace", ws, "--status", "done"); code != shapeExitFailed {
		t.Errorf("an unknown status exits %d", code)
	}
}

func TestShapeFlagsAfterTheIDAndUsageNamesTheVerbs(t *testing.T) {
	ws := shapeWorkspace(t)
	code, out, _ := shapeRun(t, okTurn, "start", "--workspace", ws, "--idea", "Flags after the id", "--json")
	var started shape.View
	if code != 0 || json.Unmarshal([]byte(out), &started) != nil {
		t.Fatalf("start %d %s", code, out)
	}
	code, out, errw := shapeRun(t, okTurn, "show", started.ID, "--json", "--workspace", ws)
	var shown shape.View
	if code != 0 || json.Unmarshal([]byte(out), &shown) != nil || shown.ID != started.ID {
		t.Fatalf("flag after the id: %d %s %s", code, out, errw)
	}
	_, _, errw = shapeRun(t, okTurn)
	for _, verb := range []string{"log", "diff", "restore", "compare", "merge", "board", "activity", "bulk", "history", "undo", "import", "export", "share"} {
		if !strings.Contains(errw, verb) {
			t.Errorf("shape usage missing %s", verb)
		}
	}
	for _, verb := range []string{"./span implementation ", "log|diff|restore", "[--json] AET-1234"} {
		if !strings.Contains(usageText, verb) {
			t.Errorf("top usage missing %q", verb)
		}
	}
}
