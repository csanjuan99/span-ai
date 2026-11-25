package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"span/internal/engine"
	"span/internal/statement"
)

const qaFailingJSON = `{"Action":"output","Package":"acme/api/internal/webhook","Test":"TestRetryBackoff","Output":"    retry_test.go:58: delay 2s, want 4s\n"}
{"Action":"fail","Package":"acme/api/internal/webhook","Test":"TestRetryBackoff","Elapsed":0.02}
{"Action":"pass","Package":"acme/api/internal/webhook","Test":"TestSendOK","Elapsed":0.01}
`

// qaFixture makes a run whose prove statement failed on a go test command with a kept log.
func qaFixture(t *testing.T) (root, id string) {
	t.Helper()
	root, id = t.TempDir(), "span-1"
	dir := filepath.Join(root, id)
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module acme/api\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "iter", "1", "logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(engine.Cursor{ID: id, Workspace: workspace})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cursor.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "iter", "1", "logs", "prove-1.log"), []byte(qaFailingJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	build := statement.Statement{SpanID: id, Stage: "build", Iteration: 1, Verdict: statement.Advance, Time: time.Now()}
	prove := statement.Statement{SpanID: id, Stage: "prove", Iteration: 1, Verdict: "iterate", Time: time.Now(), Commands: []statement.Command{{
		Command: "go test -json ./internal/webhook/...", Band: "fast", ExitCode: 1, DurationMS: 6400,
		LogPath: "iter/1/logs/prove-1.log", Tests: &statement.TestCounts{Passed: 1, Failed: 1},
	}}}
	for _, st := range []*statement.Statement{&build, &prove} {
		if err := statement.Write(filepath.Join(dir, "iter", "1"), st); err != nil {
			t.Fatal(err)
		}
	}
	return root, id
}

func runQA(root string, args ...string) (int, string, string) {
	var out, errw bytes.Buffer
	code := qaMain(append(args, "--spans", root), &out, &errw)
	return code, out.String(), errw.String()
}

func TestQAShowExitsTenWhenAGateFailedAndPrintsTheTable(t *testing.T) {
	root, id := qaFixture(t)
	code, out, _ := runQA(root, "show", id)
	if code != 10 || !strings.Contains(out, "GATE") || !strings.Contains(out, "fast") || !strings.Contains(out, "exit 10: a gate failed") {
		t.Fatalf("code %d out:\n%s", code, out)
	}
	code, out, _ = runQA(root, "show", id, "--json")
	var view struct {
		Verdict string `json:"verdict"`
		Exit    int    `json:"exit"`
	}
	if err := json.Unmarshal([]byte(out), &view); err != nil || code != 10 || view.Verdict != "fail" || view.Exit != 10 {
		t.Fatalf("code %d view %+v err %v", code, view, err)
	}
}

func TestQACasesAndCaseReadWhatTheOutputNames(t *testing.T) {
	root, id := qaFixture(t)
	code, out, _ := runQA(root, "cases", id, "--failed")
	if code != 0 || !strings.Contains(out, "TestRetryBackoff") || strings.Contains(out, "TestSendOK") {
		t.Fatalf("code %d out:\n%s", code, out)
	}
	code, out, _ = runQA(root, "case", id, "TestRetryBackoff")
	if code != 0 || !strings.Contains(out, "assertion: delay 2s, want 4s") || !strings.Contains(out, "internal/webhook/retry_test.go:58") {
		t.Fatalf("code %d out:\n%s", code, out)
	}
	if code, _, _ := runQA(root, "case", id, "TestNope"); code != exitNotFound {
		t.Fatalf("an unknown case exits 4, got %d", code)
	}
	if code, _, _ := runQA(root, "show", "no-such-run"); code != exitNotFound {
		t.Fatalf("an unknown run exits 4, got %d", code)
	}
}

func TestQALogPrintsWholeLogLineNumbersAndSearches(t *testing.T) {
	root, id := qaFixture(t)
	code, out, _ := runQA(root, "log", id, "fast", "--grep", "delay")
	if code != 0 || !strings.Contains(out, "1: {") || !strings.Contains(out, "1 matches") {
		t.Fatalf("code %d out:\n%s", code, out)
	}
	if code, _, _ := runQA(root, "log", id, "nope"); code != exitNotFound {
		t.Fatalf("an unknown gate exits 4, got %d", code)
	}
	if code, _, _ := runQA(root, "log", id, "fast", "--lines", "9:2"); code != 2 {
		t.Fatalf("a backwards range is a usage error, got %d", code)
	}
}

func TestQAAcceptNeedsAReason(t *testing.T) {
	root, id := qaFixture(t)
	if code, _, _ := runQA(root, "accept", id); code != 1 {
		t.Fatalf("accepting with no reason fails, got %d", code)
	}
	if code, _, errw := runQA(root, "rerun", id); code != 2 || errw == "" {
		t.Fatalf("a rerun with no case is a usage error, got %d", code)
	}
}

func TestTranscriptPrintsTheStageTurns(t *testing.T) {
	root, id := qaFixture(t)
	text := "first block\n\nsecond block\n"
	if err := os.WriteFile(filepath.Join(root, id, "transcript.log"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errw bytes.Buffer
	if code := transcriptMain([]string{id, "--spans", root, "--limit", "1"}, &out, &errw); code != 0 {
		t.Fatalf("code %d %s", code, errw.String())
	}
	if !strings.Contains(out.String(), "first block") || !strings.Contains(out.String(), "--from 1 reads on") {
		t.Fatalf("out:\n%s", out.String())
	}
}
