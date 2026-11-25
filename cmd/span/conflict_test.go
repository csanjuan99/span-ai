package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/decision"
	"span/internal/engine"
)

func conflictRun(t *testing.T, conflict *engine.Conflict) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "run-1")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(engine.Cursor{ID: "run-1", Stage: "merge", Hold: true, Conflict: conflict})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cursor.json"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func twoOptions() *engine.Conflict {
	return &engine.Conflict{Base: "main", State: "choosing", Groups: []engine.ConflictGroup{{
		ID: "g1", Kind: "content", Files: []string{"a.go"}, Summary: "both changed a", Chosen: "ours",
		Options: []engine.ConflictOption{
			{ID: "ours", Title: "Keep branch", Risk: "low", Action: "ours", Diff: "@@ -1 +1 @@\n-x\n+y\n"},
			{ID: "theirs", Title: "Take main", Risk: "high", Action: "theirs", Diff: "@@ -1 +1 @@\n-x\n+z\n"},
		},
	}}}
}

func TestConflictShowListsEveryOptionWithItsDiff(t *testing.T) {
	root := conflictRun(t, twoOptions())
	var out, errw bytes.Buffer
	if code := conflictMain([]string{"show", "run-1", "--options", "--spans", root}, &out, &errw); code != 0 {
		t.Fatalf("exit %d: %s", code, errw.String())
	}
	for _, want := range []string{"state choosing", "A  ours  Keep branch  (low risk, ours)  [chosen]", "B  theirs  Take main  (high risk, theirs)", "| +y", "| +z"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output lacks %q:\n%s", want, out.String())
		}
	}
}

func TestConflictShowWithoutOptionsLeavesTheDiffsOut(t *testing.T) {
	root := conflictRun(t, twoOptions())
	var out, errw bytes.Buffer
	if code := conflictMain([]string{"show", "run-1", "--json", "--spans", root}, &out, &errw); code != 0 {
		t.Fatalf("exit %d: %s", code, errw.String())
	}
	var got engine.Conflict
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Groups[0].Options[0].Diff != "" || got.Groups[0].Options[0].ID != "ours" {
		t.Errorf("a plain show must carry no diff: %+v", got.Groups[0].Options[0])
	}
	out.Reset()
	if code := conflictMain([]string{"show", "--options", "--json", "run-1", "--spans", root}, &out, &errw); code != 0 {
		t.Errorf("flags may come before the run id, got exit %d: %s", code, errw.String())
	}
	if code := conflictMain([]string{"show"}, &out, &errw); code != 2 {
		t.Errorf("show without a run is a usage error, got exit %d", code)
	}
}

func TestConflictShowJSONWithOptionsCarriesDiffs(t *testing.T) {
	root := conflictRun(t, twoOptions())
	var out, errw bytes.Buffer
	if code := conflictMain([]string{"show", "run-1", "--options", "--json", "--spans", root}, &out, &errw); code != 0 {
		t.Fatalf("exit %d: %s", code, errw.String())
	}
	var got engine.Conflict
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.Groups[0].Options[1].Diff, "+z") {
		t.Errorf("diff missing: %+v", got.Groups[0].Options[1])
	}
}

func TestConflictShowRefusesARunWithoutAConflict(t *testing.T) {
	root := conflictRun(t, nil)
	var out, errw bytes.Buffer
	if code := conflictMain([]string{"show", "run-1", "--spans", root}, &out, &errw); code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	if !strings.Contains(errw.String(), "no conflict") {
		t.Errorf("stderr = %q", errw.String())
	}
	if code := conflictMain([]string{"show", "nope", "--spans", root}, &out, &errw); code != 1 {
		t.Errorf("unknown run: exit %d, want 1", code)
	}
}

// fakeContinue replaces the step that drives the engine and records what it was asked.
func fakeContinue(t *testing.T) *[]string {
	t.Helper()
	var calls []string
	before := continueRun
	continueRun = func(_ context.Context, _, id string, answer *decision.Answer) (engine.Result, error) {
		entry := id
		if answer != nil {
			entry += " decision:" + answer.Option + ":" + answer.Text
		}
		calls = append(calls, entry)
		return engine.Result{ID: id, Stage: "prove", Verdict: "advance", Reason: "merged"}, nil
	}
	t.Cleanup(func() { continueRun = before })
	return &calls
}

func twoGroups() *engine.Conflict {
	c := twoOptions()
	c.Groups = append(c.Groups, engine.ConflictGroup{ID: "g2", Kind: "lockfile", Files: []string{"go.sum"}, Options: []engine.ConflictOption{
		{ID: "ours", Title: "Keep", Risk: "low", Action: "ours"}, {ID: "theirs", Title: "Take", Risk: "high", Action: "theirs"},
	}})
	return c
}

func TestConflictChooseRecordsEveryGroupThenResumes(t *testing.T) {
	root := conflictRun(t, twoGroups())
	calls := fakeContinue(t)
	var out, errw bytes.Buffer
	if code := conflictMain([]string{"choose", "run-1", "g1=theirs", "g2=ours", "--json", "--spans", root}, &out, &errw); code != 0 {
		t.Fatalf("exit %d: %s", code, errw.String())
	}
	raw, err := os.ReadFile(filepath.Join(root, "run-1", "conflict-choice.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"g1":"theirs"`) || !strings.Contains(string(raw), `"g2":"ours"`) {
		t.Errorf("choice file = %s", raw)
	}
	if len(*calls) != 1 || (*calls)[0] != "run-1" {
		t.Errorf("the run must be continued once, got %v", *calls)
	}
	var got outcome
	if err := json.Unmarshal(out.Bytes(), &got); err != nil || !got.Resumed || got.Verdict != "advance" {
		t.Errorf("outcome = %+v (%v)", got, err)
	}
}

func TestConflictChooseRefusesWhatTheEngineRefuses(t *testing.T) {
	root := conflictRun(t, twoGroups())
	calls := fakeContinue(t)
	var out, errw bytes.Buffer
	if code := conflictMain([]string{"choose", "run-1", "g1=theirs", "--spans", root}, &out, &errw); code != 1 {
		t.Errorf("a missing group: exit %d, want 1", code)
	}
	if !strings.Contains(errw.String(), "group g2 has no choice") {
		t.Errorf("stderr = %q", errw.String())
	}
	if code := conflictMain([]string{"choose", "run-1", "oops", "--spans", root}, &out, &errw); code != 2 {
		t.Errorf("a malformed pair: exit %d, want 2", code)
	}
	if len(*calls) != 0 {
		t.Errorf("a refused choice must not wake the run: %v", *calls)
	}
}

func TestConflictChooseTwoWordFormAndNoResume(t *testing.T) {
	root := conflictRun(t, twoOptions())
	calls := fakeContinue(t)
	var out, errw bytes.Buffer
	if code := conflictMain([]string{"choose", "run-1", "g1", "ours", "--no-resume", "--spans", root}, &out, &errw); code != 0 {
		t.Fatalf("exit %d: %s", code, errw.String())
	}
	if len(*calls) != 0 || !strings.Contains(out.String(), "./span resume run-1") {
		t.Errorf("--no-resume must only record: calls %v, out %q", *calls, out.String())
	}
}

func TestConflictHintAndAbort(t *testing.T) {
	root := conflictRun(t, twoOptions())
	calls := fakeContinue(t)
	var out, errw bytes.Buffer
	if code := conflictMain([]string{"hint", "run-1", "--spans", root}, &out, &errw); code != 2 {
		t.Errorf("a hint needs words: exit %d, want 2", code)
	}
	if code := conflictMain([]string{"hint", "run-1", "--text", "keep the new rounding", "--spans", root}, &out, &errw); code != 0 {
		t.Fatalf("hint exit %d: %s", code, errw.String())
	}
	raw, err := os.ReadFile(filepath.Join(root, "run-1", "conflict-hint"))
	if err != nil || !strings.Contains(string(raw), "keep the new rounding") {
		t.Errorf("hint file = %q (%v)", raw, err)
	}
	if code := conflictMain([]string{"abort", "run-1", "--spans", root}, &out, &errw); code != 0 {
		t.Fatalf("abort exit %d: %s", code, errw.String())
	}
	if len(*calls) != 2 {
		t.Errorf("each recorded decision continues the run: %v", *calls)
	}
	bare := conflictRun(t, nil)
	if code := conflictMain([]string{"abort", "run-1", "--spans", bare}, &out, &errw); code != 1 {
		t.Errorf("a run with no conflict: exit %d, want 1", code)
	}
}

func TestDecisionShowAndChoose(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "run-1")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	req := decision.Request{ID: "d1", Class: decision.Question, Stage: "frame", Title: "Where?", Happened: "h", Recommendation: "a", Options: []decision.Option{
		{ID: "a", Label: "Table", Consequence: "c", Reversible: true, Then: decision.Build, Gain: "Durable", Cost: "A migration"},
		{ID: "b", Label: "Memory", Consequence: "c", Then: decision.Stop},
	}}
	body, err := json.Marshal(engine.Cursor{ID: "run-1", Stage: "frame", Hold: true, Decision: &req})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cursor.json"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	calls := fakeContinue(t)
	var out, errw bytes.Buffer
	if code := decisionMain([]string{"show", "run-1", "--spans", root}, &out, &errw); code != 0 {
		t.Fatalf("show exit %d: %s", code, errw.String())
	}
	for _, want := range []string{"Where?", "A  a  Table  [recommended]", "gain: Durable | cost: A migration", "This cannot be undone from here."} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("show lacks %q:\n%s", want, out.String())
		}
	}
	if code := decisionMain([]string{"choose", "run-1", "zzz", "--spans", root}, &out, &errw); code != 1 {
		t.Errorf("an unknown option: exit %d, want 1", code)
	}
	if code := decisionMain([]string{"choose", "run-1", "b", "--text", "no table", "--spans", root}, &out, &errw); code != 0 {
		t.Fatalf("choose exit %d: %s", code, errw.String())
	}
	if len(*calls) != 1 || (*calls)[0] != "run-1 decision:b:no table" {
		t.Errorf("calls = %v", *calls)
	}
	if code := decisionMain([]string{"choose", "run-1", "b", "--decision", "old", "--spans", root}, &out, &errw); code != 1 {
		t.Errorf("an answer for another decision: exit %d, want 1", code)
	}
}
