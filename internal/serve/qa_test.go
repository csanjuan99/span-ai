package serve

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"span/internal/statement"
)

const failingJSON = `{"Action":"output","Package":"acme/api/internal/webhook","Test":"TestRetryBackoff","Output":"    retry_test.go:58: delay 2s, want 4s\n"}
{"Action":"output","Package":"acme/api/internal/webhook","Test":"TestRetryBackoff","Output":"--- FAIL: TestRetryBackoff (0.02s)\n"}
{"Action":"fail","Package":"acme/api/internal/webhook","Test":"TestRetryBackoff","Elapsed":0.02}
{"Action":"pass","Package":"acme/api/internal/webhook","Test":"TestSendOK","Elapsed":0.01}
{"Action":"fail","Package":"acme/api/internal/webhook","Elapsed":0.4}
`

// qaRun makes a run whose prove statement holds the given commands, with a log per command.
func qaRun(t *testing.T, commands []statement.Command, logs map[string]string, build bool) (root, id string) {
	t.Helper()
	root, id = featureRun(t, "pay", "")
	dir := filepath.Join(root, id)
	cursor := map[string]any{}
	raw, err := os.ReadFile(filepath.Join(dir, "cursor.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &cursor); err != nil {
		t.Fatal(err)
	}
	workspace, _ := cursor["workspace"].(string)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module acme/api\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for rel, text := range logs {
		writeLog(t, root, id, rel, text)
	}
	write := func(stage, verdict string, cmds []statement.Command) {
		st := statement.Statement{SpanID: id, Stage: stage, Iteration: 1, Verdict: verdict, Commands: cmds,
			Time: time.Date(2026, 11, 25, 14, 36, 9, 0, time.UTC)}
		if err := statement.Write(filepath.Join(dir, "iter", "1"), &st); err != nil {
			t.Fatal(err)
		}
	}
	if build {
		write("build", statement.Advance, nil)
	}
	if commands != nil {
		write("prove", "iterate", commands)
	}
	return root, id
}

func getJSON(t *testing.T, root, target string, into any) int {
	t.Helper()
	rec := httptest.NewRecorder()
	Handler(root, "").ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), into); err != nil {
			t.Fatal(err)
		}
	}
	return rec.Code
}

func TestAFailingGateReadsAsFailWithItsCountsAndCommand(t *testing.T) {
	cmds := []statement.Command{
		{Command: "go build ./...", Band: "fast", DurationMS: 4100},
		{Command: "go test -json ./internal/webhook/...", Band: "fast", ExitCode: 1, DurationMS: 6400, Lines: 5,
			LogPath: "iter/1/logs/prove-2.log", Tests: &statement.TestCounts{Passed: 41, Failed: 2}},
	}
	root, id := qaRun(t, cmds, map[string]string{"iter/1/logs/prove-2.log": failingJSON}, true)
	var view QAView
	if code := getJSON(t, root, "/spans/"+id+"/qa", &view); code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if view.Verdict != "fail" || view.Exit != QAExitFailed || view.Command != "go test -json ./internal/webhook/..." {
		t.Fatalf("view %+v", view)
	}
	if view.Counts == nil || view.Counts.Failed != 2 || view.Headline != "Fast-2 failed: 2 of 43 cases" {
		t.Fatalf("counts %+v headline %q", view.Counts, view.Headline)
	}
	if len(view.Gates) != 2 || view.Gates[0].Name != "fast-1" || view.Gates[1].Verdict != "fail" || view.Cases != 2 {
		t.Fatalf("gates %+v cases %d", view.Gates, view.Cases)
	}
}

func TestAPassThatSkippedCasesIsHeldNotPassed(t *testing.T) {
	cmds := []statement.Command{{Command: "go test ./...", Band: "fast", Tests: &statement.TestCounts{Passed: 41, Skipped: 6}}}
	root, id := qaRun(t, cmds, nil, true)
	view, err := ReadQA(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if view.Verdict != "held" || view.Exit != QAExitNoProof || !strings.Contains(view.Headline, "6 cases were skipped") {
		t.Fatalf("view %+v", view)
	}
}

func TestACommandThatFoundNoTestIsNoCasesAndNotAPass(t *testing.T) {
	cmds := []statement.Command{{Command: "go test ./internal/webhook/...", Band: "fast", Output: "?   \tacme/api/internal/webhook\t[no test files]\n"}}
	root, id := qaRun(t, cmds, nil, true)
	view, err := ReadQA(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if view.State != "no_cases" || view.Verdict != "held" || view.Next == "" || view.Kept == "" {
		t.Fatalf("view %+v", view)
	}
}

func TestAPassWithoutASealedBuildIsNotShownAsAPass(t *testing.T) {
	cmds := []statement.Command{{Command: "go test ./...", Band: "fast", Tests: &statement.TestCounts{Passed: 43}}}
	root, id := qaRun(t, cmds, nil, false)
	view, err := ReadQA(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if view.State != "unsealed" || view.Verdict != "fail" || view.Exit != QAExitFailed {
		t.Fatalf("view %+v", view)
	}
}

func TestARunThatNeverReachedProveHasNoVerdict(t *testing.T) {
	root, id := qaRun(t, nil, nil, true)
	view, err := ReadQA(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if view.State != "never_ran" || view.Verdict != "notrun" || view.Exit != QAExitNoProof || len(view.Gates) != 0 {
		t.Fatalf("view %+v", view)
	}
}

func TestCasesListFailuresFirstAndPlaceTheDeclarations(t *testing.T) {
	cmds := []statement.Command{{Command: "go test -json ./internal/webhook/...", Band: "fast", ExitCode: 1,
		LogPath: "iter/1/logs/prove-1.log", Tests: &statement.TestCounts{Passed: 1, Failed: 1}}}
	root, id := qaRun(t, cmds, map[string]string{"iter/1/logs/prove-1.log": failingJSON}, true)
	var all QACases
	if code := getJSON(t, root, "/spans/"+id+"/qa/cases?group=file", &all); code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if !all.Known || all.Total != 2 || all.Failed != 1 || all.Cases[0].Name != "TestRetryBackoff" || all.Cases[0].Line != 58 {
		t.Fatalf("cases %+v", all)
	}
	if !strings.HasPrefix(all.Rerun, "go test ./internal/webhook/... -run ^(TestRetryBackoff)$ -count=1 -json") {
		t.Fatalf("rerun command %q", all.Rerun)
	}
	var failed QACases
	getJSON(t, root, "/spans/"+id+"/qa/cases?status=failed", &failed)
	if len(failed.Cases) != 1 || failed.Cases[0].Result != "fail" {
		t.Fatalf("failed filter %+v", failed.Cases)
	}
	var one QACase
	if code := getJSON(t, root, "/spans/"+id+"/qa/cases/TestRetryBackoff", &one); code != http.StatusOK || one.Message != "delay 2s, want 4s" || len(one.History) != 1 {
		t.Fatalf("case %d %+v", code, one)
	}
	if code := getJSON(t, root, "/spans/"+id+"/qa/cases/TestNope", &one); code != http.StatusNotFound {
		t.Fatalf("an unknown case answers 404, got %d", code)
	}
}

func TestOutputWithoutCasesSaysHowToGetThem(t *testing.T) {
	cmds := []statement.Command{{Command: "go test ./...", Band: "fast", Output: "ok  \tacme/api\t0.4s\n"}}
	root, id := qaRun(t, cmds, nil, true)
	var body QACases
	getJSON(t, root, "/spans/"+id+"/qa/cases", &body)
	if body.Known || !strings.Contains(body.Note, "go test -json") {
		t.Fatalf("body %+v", body)
	}
}

func TestRerunArgsKeepPackagesAndRefuseWhatIsNotGoTest(t *testing.T) {
	args, err := rerunArgs("go test -tags integration ./a/... ./b -run Old -count=3 -v", []string{"TestA", "TestB"})
	if err != nil {
		t.Fatal(err)
	}
	want := "test -tags integration ./a/... ./b -run ^(TestA|TestB)$ -count=1 -json"
	if strings.Join(args, " ") != want {
		t.Fatalf("args %q", strings.Join(args, " "))
	}
	for _, bad := range []struct {
		command string
		names   []string
	}{
		{"npm test", []string{"TestA"}},
		{"go test ./...", []string{"Test A; rm -rf /"}},
		{"go test ./...", nil},
	} {
		if _, err := rerunArgs(bad.command, bad.names); err == nil {
			t.Fatalf("%q %v must be refused", bad.command, bad.names)
		}
	}
}

func TestARerunRunsOnlyTheNamedCasesAndKeepsTheFirstResult(t *testing.T) {
	cmds := []statement.Command{{Command: "go test -json ./internal/webhook/...", Band: "fast", ExitCode: 1,
		LogPath: "iter/1/logs/prove-1.log", Tests: &statement.TestCounts{Passed: 1, Failed: 1}}}
	root, id := qaRun(t, cmds, map[string]string{"iter/1/logs/prove-1.log": failingJSON}, true)
	var got []string
	old := goTest
	defer func() { goTest = old }()
	goTest = func(_ context.Context, _ string, args []string) (string, int, error) {
		got = args
		return `{"Action":"pass","Package":"acme/api/internal/webhook","Test":"TestRetryBackoff","Elapsed":0.01}` + "\n", 0, nil
	}
	body, err := RerunQA(context.Background(), root, id, []string{"TestRetryBackoff"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, " ") != "test ./internal/webhook/... -run ^(TestRetryBackoff)$ -count=1 -json" {
		t.Fatalf("args %v", got)
	}
	if len(body.Cases) != 1 || body.Cases[0].Result != "pass" {
		t.Fatalf("rerun %+v", body)
	}
	one, err := ReadQACase(root, id, "TestRetryBackoff")
	if err != nil || len(one.History) != 2 || one.History[0].Result != "fail" || one.History[1].Kind != "rerun" || one.History[1].Result != "pass" {
		t.Fatalf("history %+v err %v", one.History, err)
	}
	if _, err := RerunQA(context.Background(), root, id, []string{"TestNotInThisRun"}); err == nil {
		t.Fatal("a case that is not in the run must be refused")
	}
}

func TestAcceptingNeedsAReasonAndIsRecordedWithIt(t *testing.T) {
	cmds := []statement.Command{{Command: "go test ./...", Band: "fast", Tests: &statement.TestCounts{Passed: 41, Skipped: 6}}}
	root, id := qaRun(t, cmds, nil, true)
	if _, err := AcceptQA(root, id, "  "); err == nil {
		t.Fatal("an empty reason must be refused")
	}
	accepted, err := AcceptQA(root, id, "docs only")
	if err != nil || accepted.Reason != "docs only" || accepted.Gate != "fast" {
		t.Fatalf("accepted %+v err %v", accepted, err)
	}
	view, err := ReadQA(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if view.Accepted == nil || view.Accepted.Reason != "docs only" {
		t.Fatalf("view %+v", view.Accepted)
	}
	pass := []statement.Command{{Command: "go test ./...", Band: "fast", Tests: &statement.TestCounts{Passed: 3}}}
	root2, id2 := qaRun(t, pass, nil, true)
	if _, err := AcceptQA(root2, id2, "why"); err == nil {
		t.Fatal("a gate that passed has nothing to accept")
	}
}

func TestAGateLogHasWholeLogLineNumbersFirstErrorAndMatches(t *testing.T) {
	log := strings.Repeat("=== RUN A\n--- PASS: A (0.00s)\n", 5) + "=== RUN B\n    b_test.go:9: boom\n--- FAIL: B (0.01s)\nFAIL\n"
	cmds := []statement.Command{{Command: "go test -v ./...", Band: "fast", ExitCode: 1, LogPath: "iter/1/logs/prove-1.log", DurationMS: 6400}}
	root, id := qaRun(t, cmds, map[string]string{"iter/1/logs/prove-1.log": log}, true)
	var all GateLog
	if code := getJSON(t, root, "/spans/"+id+"/logs/fast?q=BOOM", &all); code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if all.TotalLines != 14 || all.FirstError != 12 || len(all.Matches) != 1 || all.Matches[0] != 12 || len(all.Lines) != 14 {
		t.Fatalf("log %+v", all)
	}
	var near GateLog
	getJSON(t, root, "/spans/"+id+"/logs/fast?first_error=1", &near)
	if !near.Windowed || near.Lines[0].N != 9 || near.Lines[len(near.Lines)-1].N != 14 {
		t.Fatalf("a window around the first error: %+v", near.Lines)
	}
	var some GateLog
	getJSON(t, root, "/spans/"+id+"/logs/fast?from=13&to=14", &some)
	if len(some.Lines) != 2 || some.Lines[0].N != 13 || !some.Windowed {
		t.Fatalf("range %+v", some.Lines)
	}
	if code := getJSON(t, root, "/spans/"+id+"/logs/nope", &some); code != http.StatusNotFound {
		t.Fatalf("an unknown gate answers 404, got %d", code)
	}
}
