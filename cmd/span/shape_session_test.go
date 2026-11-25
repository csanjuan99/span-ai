package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/jira"
	"span/internal/shape"
)

type cliBacklog struct{ created []jira.Draft }

func (b *cliBacklog) Preview(context.Context, string, int) (jira.Preview, error) {
	return jira.Preview{Board: jira.Board{ID: 7, Name: "Billing"}, BacklogStatuses: []string{"Backlog"}}, nil
}

func (b *cliBacklog) CreateInBacklog(_ context.Context, d jira.Draft) (jira.Created, error) {
	b.created = append(b.created, d)
	return jira.Created{Key: "BILL-1", URL: "https://x/browse/BILL-1", Status: "Backlog"}, nil
}

const storyTurn = `ok
{"map":{"rules":[{"id":"R1","text":"r","source":{"kind":"prompt","ref":"idea"}}],"examples":[],"questions":[]},"split":[],"proposals":[{"id":"P1","section":"Access","text":"- Any user"}],"story":"# Log in\n\nAs a user."}`

func jsonOut(t *testing.T, out string, into any) {
	t.Helper()
	if err := json.Unmarshal([]byte(out), into); err != nil {
		t.Fatalf("not JSON %q: %v", out, err)
	}
}

func TestShapeSessionVerbsWalkASessionToTheEnd(t *testing.T) {
	ws := shapeWorkspace(t)
	reply := okTurn
	code, out, errText := shapeRun(t, func() (string, error) { return reply() }, "start", "--workspace", ws, "--idea", "log in", "--json")
	var v shape.View
	if code != 0 {
		t.Fatalf("start %d %s %s", code, out, errText)
	}
	jsonOut(t, out, &v)
	if v.OpenQuestions != 1 {
		t.Fatalf("start %d [%s] [%s]", code, out, errText)
	}
	reply = func() (string, error) { return storyTurn, nil }
	code, out, _ = shapeRun(t, func() (string, error) { return reply() }, "answer", "--workspace", ws, "--answer", "Q1=admins", "--json", v.ID)
	jsonOut(t, out, &v)
	if code != 0 || v.Status != shape.StatusReady || v.Map.Questions[0].AnswerID != "A1" {
		t.Fatalf("answer %d %s", code, out)
	}
	if code, _, errText := shapeRun(t, okTurn, "answer", "--workspace", ws, "--answer", "oops", v.ID); code != shapeExitUsage || !strings.Contains(errText, "Q1=text") {
		t.Fatalf("a malformed answer is a usage error: %d %s", code, errText)
	}
	code, out, _ = shapeRun(t, okTurn, "context", "--workspace", ws, "--accept", "P1", "--preview", "--json", v.ID)
	var diff shape.ContextDiff
	jsonOut(t, out, &diff)
	if code != 0 || len(diff.Lines) == 0 {
		t.Fatalf("preview %d %s", code, out)
	}
	if _, err := os.Stat(shape.ContextPath(ws)); err == nil {
		t.Fatal("--preview writes nothing")
	}
	if code, _, _ = shapeRun(t, okTurn, "context", "--workspace", ws, "--accept", "P1", v.ID); code != 0 {
		t.Fatalf("accept %d", code)
	}
	if code, _, _ = shapeRun(t, okTurn, "context", "--workspace", ws, "--skip", "P1", v.ID); code != shapeExitRefused {
		t.Fatalf("a written note cannot be skipped: exit %d", code)
	}
	if code, _, _ = shapeRun(t, okTurn, "story", "--workspace", ws, v.ID); code != shapeExitRefused {
		t.Fatalf("story of an unsealed session: exit %d", code)
	}
	code, out, _ = shapeRun(t, okTurn, "seal", "--workspace", ws, "--json", v.ID)
	jsonOut(t, out, &v)
	if code != 0 || v.Status != shape.StatusSealed || v.Digest == "" {
		t.Fatalf("seal %d %s", code, out)
	}
	code, out, _ = shapeRun(t, okTurn, "story", "--workspace", ws, v.ID)
	if code != 0 || !strings.HasPrefix(out, "# Log in") {
		t.Fatalf("story %d %q", code, out)
	}
	backlog := &cliBacklog{}
	before := shapeBacklog
	shapeBacklog = func(string) (shape.Backlog, error) { return backlog, nil }
	t.Cleanup(func() { shapeBacklog = before })
	code, out, _ = shapeRun(t, okTurn, "preview", "--workspace", ws, "--project", "bill", "--json", v.ID)
	var view shape.IssuePreview
	jsonOut(t, out, &view)
	if code != 0 || view.Project != "BILL" || view.Summary != "Log in" || len(backlog.created) != 0 {
		t.Fatalf("preview %d %s", code, out)
	}
	if code, _, _ = shapeRun(t, okTurn, "jira", "--workspace", ws, "--project", "BILL", v.ID); code != shapeExitRefused || len(backlog.created) != 0 {
		t.Fatalf("without --confirm nothing is sent: exit %d", code)
	}
	code, out, _ = shapeRun(t, okTurn, "jira", "--workspace", ws, "--project", "BILL", "--confirm", "--json", v.ID)
	jsonOut(t, out, &v)
	if code != 0 || v.JiraKey != "BILL-1" || len(backlog.created) != 1 {
		t.Fatalf("jira %d %s", code, out)
	}
	code, out, _ = shapeRun(t, okTurn, "fork", "--workspace", ws, "--json", v.ID)
	var forked shape.View
	jsonOut(t, out, &forked)
	if code != 0 || forked.ID == v.ID || forked.Parent != v.ID {
		t.Fatalf("fork %d %s", code, out)
	}
}

func TestShapeSplitAndSourceVerbs(t *testing.T) {
	ws := shapeWorkspace(t)
	splitTurn := func() (string, error) {
		return `ok
{"map":{"rules":[],"examples":[],"questions":[]},"split":[{"story":"A","leftOut":"B"},{"story":"B","leftOut":"A"}],"proposals":[],"story":""}`, nil
	}
	code, out, _ := shapeRun(t, splitTurn, "start", "--workspace", ws, "--idea", "two things", "--json")
	var v shape.View
	jsonOut(t, out, &v)
	if code != 0 || v.Status != shape.StatusSplit {
		t.Fatalf("start %d [%s] [%s]", code, out, out)
	}
	if code, _, _ = shapeRun(t, splitTurn, "split", "--workspace", ws, v.ID); code != shapeExitUsage {
		t.Fatalf("needs --split or --keep: %d", code)
	}
	code, out, _ = shapeRun(t, splitTurn, "split", "--workspace", ws, "--split", "--json", v.ID)
	jsonOut(t, out, &v)
	if code != 0 || len(v.Children) != 2 {
		t.Fatalf("split %d %s", code, out)
	}
	if err := os.WriteFile(filepath.Join(ws, "a.go"), []byte("x\ny\nz\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ = shapeRun(t, splitTurn, "source", "--workspace", ws, "--ref", "a.go:2", v.ID)
	if code != 0 || !strings.Contains(out, ">    2  y") {
		t.Fatalf("source %d %q", code, out)
	}
	if code, _, _ = shapeRun(t, splitTurn, "source", "--workspace", ws, "--ref", "nope.go:1", v.ID); code != shapeExitNotFound {
		t.Fatalf("missing file: exit %d", code)
	}
}
