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

// opsRun writes a run with the given cursor and returns the state directory.
func opsRun(t *testing.T, cur engine.Cursor) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "run-1")
	if err := os.MkdirAll(filepath.Join(dir, "iter", "1", "logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	cur.ID = "run-1"
	body, err := json.Marshal(cur)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cursor.json"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// call runs one verb and returns its exit code and what it printed.
func call(verb func([]string, *bytes.Buffer, *bytes.Buffer) int, args ...string) (int, string, string) {
	var out, errw bytes.Buffer
	code := verb(args, &out, &errw)
	return code, out.String(), errw.String()
}

func TestRunsListAndShowReadWhatTheRoutesRead(t *testing.T) {
	root := opsRun(t, engine.Cursor{Stage: "prove", Hold: true})
	code, out, errs := call(func(a []string, o, e *bytes.Buffer) int { return runsMain(a, o, e) }, "list", "--json", "--spans", root)
	if code != 0 {
		t.Fatalf("list exit %d: %s", code, errs)
	}
	var list struct{ Spans []engine.Head }
	if err := json.Unmarshal([]byte(out), &list); err != nil || len(list.Spans) != 1 || list.Spans[0].ID != "run-1" {
		t.Fatalf("list = %s (%v)", out, err)
	}
	code, out, _ = call(func(a []string, o, e *bytes.Buffer) int { return runsMain(a, o, e) }, "show", "run-1", "--spans", root)
	if code != 0 || !strings.Contains(out, "stage prove") {
		t.Errorf("show exit %d: %s", code, out)
	}
	if code, _, _ := call(func(a []string, o, e *bytes.Buffer) int { return runsMain(a, o, e) }, "show", "nope", "--spans", root); code != 1 {
		t.Errorf("a missing run exits 1, got %d", code)
	}
	if code, _, _ := call(func(a []string, o, e *bytes.Buffer) int { return runsMain(a, o, e) }, "show", "--spans", root); code != 2 {
		t.Errorf("show without a run is a usage error, got %d", code)
	}
}

func TestRunsReadersNeedNoWorkspace(t *testing.T) {
	root := opsRun(t, engine.Cursor{Stage: "prove"})
	for _, verb := range []string{"implementation", "feature", "acceptance", "events"} {
		code, out, errs := call(func(a []string, o, e *bytes.Buffer) int { return runsMain(a, o, e) }, verb, "run-1", "--json", "--spans", root)
		if code != 0 || !json.Valid([]byte(out)) {
			t.Errorf("%s exit %d: %s %s", verb, code, out, errs)
		}
	}
}

func TestLogPrintsTheCommandLogAndKeepsTheRoutesLimits(t *testing.T) {
	root := opsRun(t, engine.Cursor{Stage: "prove"})
	file := filepath.Join(root, "run-1", "iter", "1", "logs", "prove-1.log")
	if err := os.WriteFile(file, []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "run-1", "cursor.json.bak"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	logv := func(a []string, o, e *bytes.Buffer) int { return logMain(a, o, e) }
	code, out, _ := call(logv, "run-1", "iter/1/logs/prove-1.log", "--spans", root)
	if code != 0 || out != "one\ntwo\n" {
		t.Errorf("exit %d, text %q", code, out)
	}
	_, out, _ = call(logv, "run-1", "iter/1/logs/prove-1.log", "--json", "--spans", root)
	var view struct {
		Lines     int  `json:"lines"`
		Truncated bool `json:"truncated"`
	}
	if err := json.Unmarshal([]byte(out), &view); err != nil || view.Lines != 2 || view.Truncated {
		t.Errorf("json = %s (%v)", out, err)
	}
	for _, bad := range []string{"../run-1/cursor.json", "iter/../cursor.json.bak", "cursor.json.bak", "iter/1/logs/missing.log", "/etc/passwd"} {
		if code, _, _ := call(logv, "run-1", bad, "--spans", root); code != 1 {
			t.Errorf("path %q must be refused with exit 1, got %d", bad, code)
		}
	}
	if code, _, _ := call(logv, "run-1", "--spans", root); code != 2 {
		t.Errorf("a log without a path is a usage error, got %d", code)
	}
}

func TestPauseRequestsAHoldAndRefusesAFinishedRun(t *testing.T) {
	root := opsRun(t, engine.Cursor{Stage: "build"})
	pause := func(a []string, o, e *bytes.Buffer) int { return pauseMain(a, o, e) }
	code, out, errs := call(pause, "run-1", "--json", "--spans", root)
	if code != 0 || !strings.Contains(out, `"done": true`) {
		t.Fatalf("exit %d: %s %s", code, out, errs)
	}
	if _, err := os.Stat(filepath.Join(root, "run-1", "pause")); err != nil {
		entries, readErr := os.ReadDir(filepath.Join(root, "run-1"))
		if readErr != nil {
			t.Fatal(readErr)
		}
		t.Errorf("no pause request was written (%v): %v", err, entries)
	}
	done := opsRun(t, engine.Cursor{Stage: "merge", Done: true})
	if code, _, _ := call(pause, "run-1", "--spans", done); code != 1 {
		t.Errorf("a finished run cannot be paused, got exit %d", code)
	}
}

func TestNoteSavesTheInstructionForAStage(t *testing.T) {
	root := opsRun(t, engine.Cursor{Stage: "build", Hold: true})
	note := func(a []string, o, e *bytes.Buffer) int { return noteMain(a, o, e) }
	if code, _, errs := call(note, "run-1", "prove", "--text", "run the slow suite too", "--spans", root); code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	raw, err := os.ReadFile(filepath.Join(root, "run-1", "instructions", "prove.md"))
	if err != nil || string(raw) != "run the slow suite too" {
		t.Errorf("note = %q (%v)", raw, err)
	}
	if code, _, _ := call(note, "run-1", "prove", "--spans", root); code != 2 {
		t.Errorf("a note needs --text or --file, got %d", code)
	}
	if code, _, _ := call(note, "run-1", "../prove", "--text", "x", "--spans", root); code != 1 {
		t.Errorf("a stage with a path is refused, got %d", code)
	}
}

func TestRewindRestartsAtAStageAndContinuesUnlessAsked(t *testing.T) {
	var continued int
	previous := continueRun
	continueRun = func(_ context.Context, _, _ string, answer *decision.Answer) (engine.Result, error) {
		continued++
		return engine.Result{Stage: "build", Verdict: "pass"}, nil
	}
	t.Cleanup(func() { continueRun = previous })
	rewind := func(a []string, o, e *bytes.Buffer) int { return rewindMain(a, o, e) }

	root := opsRun(t, engine.Cursor{Stage: "prove", Hold: true})
	if code, out, errs := call(rewind, "run-1", "build", "--note", "mind the cache", "--no-resume", "--spans", root); code != 0 {
		t.Fatalf("exit %d: %s %s", code, out, errs)
	}
	if continued != 0 {
		t.Error("--no-resume must not continue the run")
	}
	snap, err := engine.Load(root, "run-1")
	if err != nil || snap.Cursor.Stage != "build" {
		t.Errorf("stage = %q (%v), want build", snap.Cursor.Stage, err)
	}
	if raw, readErr := os.ReadFile(filepath.Join(root, "run-1", "instructions", "build.md")); readErr != nil || string(raw) != "mind the cache" {
		t.Errorf("the note was not kept: %q", raw)
	}
	if code, _, _ := call(rewind, "run-1", "build", "--spans", root); code != 0 || continued != 1 {
		t.Errorf("without --no-resume the run goes on once, exit %d, continued %d", code, continued)
	}
	if code, _, _ := call(rewind, "run-1", "intake", "--no-resume", "--spans", root); code != 1 {
		t.Errorf("intake cannot be restarted, got %d", code)
	}
	merging := opsRun(t, engine.Cursor{Stage: "merge", Hold: true, Conflict: twoOptions()})
	if code, _, _ := call(rewind, "run-1", "build", "--no-resume", "--spans", merging); code != 1 {
		t.Errorf("a run in a merge conflict refuses a restart, got %d", code)
	}
}

func TestStageAgentClearsAndRefusesWhatTheRouteRefuses(t *testing.T) {
	root := opsRun(t, engine.Cursor{Stage: "build"})
	set := func(a []string, o, e *bytes.Buffer) int { return stageAgentMain(a, o, e) }
	if code, _, errs := call(set, "run-1", "build", "any", "--spans", root); code != 0 {
		t.Fatalf("clearing exits 0, got %d: %s", code, errs)
	}
	if code, _, _ := call(set, "run-1", "build", "no-such-agent", "--spans", root); code != 1 {
		t.Errorf("an unknown agent is refused, got %d", code)
	}
	if code, _, _ := call(set, "run-1", "merge", "any", "--spans", root); code != 1 {
		t.Errorf("a stage with no agent is refused, got %d", code)
	}
}

func TestApproveRefusesAStageThatWaitsForNothing(t *testing.T) {
	root := opsRun(t, engine.Cursor{Stage: "build", Hold: true})
	approve := func(a []string, o, e *bytes.Buffer) int { return approveMain(a, o, e) }
	if code, _, _ := call(approve, "run-1", "build", "--no-resume", "--spans", root); code != 1 {
		t.Errorf("a stage with no approval to give exits 1, got %d", code)
	}
	if code, _, _ := call(approve, "run-1", "--spans", root); code != 2 {
		t.Errorf("approve needs a stage, got %d", code)
	}
}

func TestAnswerRecordsAllTheAnswersAndContinuesOnce(t *testing.T) {
	var got []engine.Answer
	previous := continueAnswers
	continueAnswers = func(_ context.Context, _, _ string, answers []engine.Answer) (engine.Result, error) {
		got = answers
		return engine.Result{Stage: "build", Verdict: "pass", Reason: "ok"}, nil
	}
	t.Cleanup(func() { continueAnswers = previous })
	answer := func(a []string, o, e *bytes.Buffer) int { return answerMain(a, o, e) }

	waiting := opsRun(t, engine.Cursor{Stage: "frame", Hold: true, Questions: []string{"Which db?", "Keep the old route?"}})
	code, out, errs := call(answer, "run-1", "--answer", "Which db?=postgres", "--answer", "Keep the old route?=no, drop it", "--json", "--spans", waiting)
	if code != 0 {
		t.Fatalf("exit %d: %s %s", code, out, errs)
	}
	if len(got) != 2 || got[1].Question != "Keep the old route?" || got[1].Body != "no, drop it" {
		t.Errorf("answers = %+v", got)
	}
	if code, _, _ := call(answer, "run-1", "--answer", "Which db?=", "--spans", waiting); code != 2 {
		t.Errorf("an empty answer is refused as a usage error, got %d", code)
	}
	if code, _, _ := call(answer, "run-1", "--spans", waiting); code != 2 {
		t.Errorf("no answers is a usage error, got %d", code)
	}
	idle := opsRun(t, engine.Cursor{Stage: "build"})
	if code, _, _ := call(answer, "run-1", "--answer", "x=y", "--spans", idle); code != 1 {
		t.Errorf("a run that is not waiting refuses, got %d", code)
	}
}

func TestDeleteNeedsConfirmation(t *testing.T) {
	root := opsRun(t, engine.Cursor{Stage: "build"})
	del := func(a []string, o, e *bytes.Buffer) int { return deleteMain(a, o, e) }
	if code, _, _ := call(del, "run-1", "--spans", root); code != 2 {
		t.Errorf("delete without --yes is a usage error, got %d", code)
	}
	if _, err := os.Stat(filepath.Join(root, "run-1")); err != nil {
		t.Fatal("the run must survive an unconfirmed delete")
	}
	if code, _, errs := call(del, "run-1", "--yes", "--spans", root); code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	if _, err := os.Stat(filepath.Join(root, "run-1", "cursor.json")); err == nil {
		t.Error("the run is still there")
	}
}
