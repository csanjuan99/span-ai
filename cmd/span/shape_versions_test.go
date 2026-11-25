package main

import (
	"strings"
	"testing"

	"span/internal/shape"
)

// startTwo starts two sessions whose first turn says the same rule.
func startTwo(t *testing.T, ws string) (a, b shape.View) {
	t.Helper()
	for _, into := range []*shape.View{&a, &b} {
		code, out, errText := shapeRun(t, okTurn, "start", "--workspace", ws, "--idea", "log in", "--json")
		if code != 0 {
			t.Fatalf("start %d %s %s", code, out, errText)
		}
		jsonOut(t, out, into)
	}
	return a, b
}

func TestShapeVersionsDiffAndRestoreFromTheCommandLine(t *testing.T) {
	ws := shapeWorkspace(t)
	a, _ := startTwo(t, ws)
	if code, _, _ := shapeRun(t, func() (string, error) { return storyTurn, nil }, "answer", "--workspace", ws, "--answer", "Q1=admins", a.ID); code != 0 {
		t.Fatalf("answer %d", code)
	}
	code, out, _ := shapeRun(t, okTurn, "versions", "--workspace", ws, "--json", a.ID)
	var list struct {
		Revisions []shape.RevisionInfo `json:"revisions"`
	}
	jsonOut(t, out, &list)
	if code != 0 || len(list.Revisions) < 3 || !list.Revisions[len(list.Revisions)-1].Current {
		t.Fatalf("versions %d %s", code, out)
	}
	code, out, _ = shapeRun(t, okTurn, "versions", "--workspace", ws, a.ID)
	if code != 0 || !strings.Contains(out, "Idea saved") || !strings.Contains(out, "* r") {
		t.Fatalf("table %d %q", code, out)
	}
	code, out, _ = shapeRun(t, okTurn, "diff", "--workspace", ws, "--json", a.ID, "r1", "r3")
	var d shape.RevisionDiff
	jsonOut(t, out, &d)
	if code != 0 || d.Questions.Changed+d.Questions.Added == 0 {
		t.Fatalf("diff %d %s", code, out)
	}
	if code, out, _ = shapeRun(t, okTurn, "diff", "--workspace", ws, a.ID, "r1", "r3"); code != 0 || !strings.Contains(out, "question Q1") {
		t.Fatalf("diff text %d %q", code, out)
	}
	if code, _, _ = shapeRun(t, okTurn, "diff", "--workspace", ws, a.ID, "r1", "r99"); code != shapeExitNotFound {
		t.Fatalf("unknown revision exit %d", code)
	}
	if code, _, _ = shapeRun(t, okTurn, "diff", "--workspace", ws, a.ID, "r1"); code != shapeExitUsage {
		t.Fatalf("missing argument exit %d", code)
	}
	if code, _, _ = shapeRun(t, okTurn, "restore", "--workspace", ws, a.ID, "r2"); code != 0 {
		t.Fatalf("restore exit %d", code)
	}
	_, out, _ = shapeRun(t, okTurn, "versions", "--workspace", ws, a.ID)
	if !strings.Contains(out, "Restored r2") {
		t.Fatalf("restore not in history: %q", out)
	}
}

func TestShapeCompareAndMergeFromTheCommandLine(t *testing.T) {
	ws := shapeWorkspace(t)
	a, b := startTwo(t, ws)
	code, out, _ := shapeRun(t, okTurn, "compare", "--workspace", ws, "--json", a.ID, b.Ref)
	var c shape.Comparison
	jsonOut(t, out, &c)
	if code != 0 || len(c.Same) != 1 || !c.Estimate {
		t.Fatalf("compare %d %s", code, out)
	}
	if code, out, _ = shapeRun(t, okTurn, "compare", "--workspace", ws, a.ID, b.ID); code != 0 || !strings.Contains(out, "same") || !strings.Contains(out, "estimates") {
		t.Fatalf("text %d %q", code, out)
	}
	if code, _, _ = shapeRun(t, okTurn, "compare", "--workspace", ws, a.ID); code != shapeExitFailed {
		t.Fatalf("one session exit %d", code)
	}
	if code, out, _ = shapeRun(t, okTurn, "compare", "--workspace", ws, "--dismiss", "--json", a.ID, b.ID); code != 0 {
		t.Fatalf("dismiss %d %s", code, out)
	}
	if code, _, _ = shapeRun(t, okTurn, "merge", "--workspace", ws, a.ID, b.ID); code != shapeExitUsage {
		t.Fatalf("merge without --yes exit %d", code)
	}
	if code, _, errText := shapeRun(t, okTurn, "merge", "--workspace", ws, "--yes", "--resolve", "C1", a.ID, b.ID); code != shapeExitUsage || !strings.Contains(errText, "C1=session") {
		t.Fatalf("malformed resolve exit %d %s", code, errText)
	}
	code, out, _ = shapeRun(t, okTurn, "merge", "--workspace", ws, "--yes", "--json", a.ID, b.ID)
	var merged struct {
		Merged []string `json:"merged"`
	}
	jsonOut(t, out, &merged)
	if code != 0 || len(merged.Merged) != 1 {
		t.Fatalf("merge %d %s", code, out)
	}
	if code, out, _ = shapeRun(t, okTurn, "merge", "--workspace", ws, "--yes", "--json", b.ID, a.ID); code != shapeExitFailed || !strings.Contains(out, "already merged") {
		t.Fatalf("a merged session is not merged again: %d %s", code, out)
	}
}

func TestShapeLogAliasDiffFormatAndRestoreAsNew(t *testing.T) {
	ws := shapeWorkspace(t)
	a, _ := startTwo(t, ws)
	if code, _, _ := shapeRun(t, func() (string, error) { return storyTurn, nil }, "answer", "--workspace", ws, "--answer", "Q1=admins", a.ID); code != 0 {
		t.Fatalf("answer %d", code)
	}
	if code, out, _ := shapeRun(t, okTurn, "log", "--workspace", ws, a.ID); code != 0 || !strings.Contains(out, "Idea saved") {
		t.Fatalf("log %d %q", code, out)
	}
	if code, out, _ := shapeRun(t, okTurn, "diff", "--workspace", ws, "--format", "side", a.ID, "r1", "r3"); code != 0 || !strings.Contains(out, "question Q1") {
		t.Fatalf("side %d %q", code, out)
	}
	if code, _, _ := shapeRun(t, okTurn, "diff", "--workspace", ws, "--format", "wide", a.ID, "r1", "r3"); code != shapeExitUsage {
		t.Fatalf("bad format exit %d", code)
	}
	code, out, _ := shapeRun(t, okTurn, "restore", "--workspace", ws, "--new", "--json", a.ID, "r2")
	var child shape.View
	jsonOut(t, out, &child)
	if code != 0 || child.Parent != a.ID || child.ID == a.ID {
		t.Fatalf("restore --new %d %s", code, out)
	}
}
