package engine

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/prove"
	"span/internal/statement"
)

const legacyFailure = "--- FAIL: TestLegacy (0.01s)\n    legacy_test.go:9: expected 1, got 2\nFAIL\tspan/legacy\t0.20s\n"

// baseCheck answers each command through respond and records whether a build had
// already run, so a test can tell a baseline call from a prove call.
type baseCheck struct {
	agents  *fakeAgents
	calls   []string
	before  int // calls made before the first build
	respond func(command string, builds int) (prove.Output, error)
}

func (b *baseCheck) Run(_ context.Context, command, _ string) (prove.Output, error) {
	b.calls = append(b.calls, command)
	if b.agents.builds == 0 {
		b.before++
	}
	return b.respond(command, b.agents.builds)
}

func baselineOpt(t *testing.T) (Options, *fakeAgents, *baseCheck) {
	t.Helper()
	opt, agents, source, _, _, _ := baseOpt(t)
	opt.Policy.Prove.Baseline = nil
	// The baseline is recorded only on a clean tree.
	source.dirty = false
	check := &baseCheck{agents: agents, respond: func(string, int) (prove.Output, error) {
		return prove.Output{ExitCode: 1, Text: legacyFailure}, nil
	}}
	opt.Check = check
	return opt, agents, check
}

func readBaseline(t *testing.T, opt Options, id string) (prove.Baseline, bool) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(opt.Root, id, "baseline.json"))
	if os.IsNotExist(err) {
		return prove.Baseline{}, false
	}
	if err != nil {
		t.Fatal(err)
	}
	var b prove.Baseline
	if err := json.Unmarshal(raw, &b); err != nil {
		t.Fatal(err)
	}
	return b, true
}

func TestAFailureTheBaseAlreadyHadAdvancesWithoutARepair(t *testing.T) {
	opt, agents, check := baselineOpt(t)
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Advance || got.Stage != note {
		t.Fatalf("%+v", got)
	}
	if check.before != 1 || len(check.calls) != 2 || agents.builds != 1 {
		t.Fatalf("baseline calls %d, all calls %v, builds %d", check.before, check.calls, agents.builds)
	}
	if cur := readCursor(t, opt, got.ID); len(cur.Repairs) != 0 {
		t.Fatalf("a pre-existing failure spent a repair: %+v", cur.Repairs)
	}
	base, ok := readBaseline(t, opt, got.ID)
	if !ok || base.SHA != "base" || base.Commands["go test ./..."].ExitCode != 1 || len(base.Commands["go test ./..."].Lines) != 3 {
		t.Fatalf("baseline ok=%v %+v", ok, base)
	}
	st, _, err := statement.Read(filepath.Join(opt.Root, got.ID, "iter", "1"), proveSt)
	if err != nil || st.Verdict != statement.Advance || len(st.Commands) != 1 || !st.Commands[0].Preexisting {
		t.Fatalf("prove err=%v %+v", err, st)
	}
	if !strings.Contains(st.Reason, "already present on the base commit") {
		t.Fatalf("reason = %s", st.Reason)
	}
	raw, err := os.ReadFile(filepath.Join(opt.Root, got.ID, "transcript.log"))
	if err != nil || !strings.Contains(string(raw), "baseline: go test ./... exited 1 (3 failing lines)") {
		t.Fatalf("transcript %s err %v", raw, err)
	}
}

func TestANewFailureLineStillSpendsARepair(t *testing.T) {
	opt, _, check := baselineOpt(t)
	check.respond = func(_ string, builds int) (prove.Output, error) {
		if builds == 0 {
			return prove.Output{ExitCode: 1, Text: legacyFailure}, nil
		}
		return prove.Output{ExitCode: 1, Text: legacyFailure + "--- FAIL: TestCharge (0.02s)\n"}, nil
	}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold || !strings.Contains(got.Reason, "no progress") {
		t.Fatalf("%+v", got)
	}
	cur := readCursor(t, opt, got.ID)
	if len(cur.Repairs) != 2 || cur.Repairs[0].Stage != proveSt {
		t.Fatalf("repairs %+v", cur.Repairs)
	}
	if check.before != 1 {
		t.Fatalf("the baseline ran %d times", check.before)
	}
	// One baseline plus one prove per iteration; the repair did not record another baseline.
	if len(check.calls) != 3 {
		t.Fatalf("calls %v", check.calls)
	}
	st, _, _ := statement.Read(filepath.Join(opt.Root, got.ID, "iter", "1"), proveSt)
	if st.Commands[0].Preexisting {
		t.Fatal("a new failure was marked preexisting")
	}
}

func TestARunPastBuildWithoutABaselineJudgesAsBefore(t *testing.T) {
	opt, _, check := baselineOpt(t)
	id := writeCursor(t, opt, Cursor{ID: "span-1", Iteration: 1, Stage: proveSt, GitSHA: "abc123"})
	got, err := Resume(context.Background(), opt, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold || !strings.Contains(got.Reason, "no progress") {
		t.Fatalf("%+v", got)
	}
	if _, ok := readBaseline(t, opt, id); ok {
		t.Fatal("a baseline was recorded after build")
	}
	if len(check.calls) != 2 {
		t.Fatalf("calls %v", check.calls)
	}
}

func TestABaselineThatCannotRunNeverHoldsTheRun(t *testing.T) {
	opt, _, check := baselineOpt(t)
	check.respond = func(_ string, builds int) (prove.Output, error) {
		if builds == 0 {
			return prove.Output{}, errors.New("executable not found")
		}
		return prove.Output{}, nil
	}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Advance {
		t.Fatalf("%+v", got)
	}
	base, ok := readBaseline(t, opt, got.ID)
	if !ok || len(base.Commands) != 0 {
		t.Fatalf("ok=%v %+v", ok, base)
	}
}

func TestABaselineThatCannotRunExcusesNothing(t *testing.T) {
	opt, _, check := baselineOpt(t)
	check.respond = func(_ string, builds int) (prove.Output, error) {
		if builds == 0 {
			return prove.Output{}, errors.New("executable not found")
		}
		return prove.Output{ExitCode: 1, Text: legacyFailure}, nil
	}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold || !strings.Contains(got.Reason, "no progress") {
		t.Fatalf("%+v", got)
	}
}

func TestACancelledBaselineStopsBeforeTheBuild(t *testing.T) {
	opt, agents, check := baselineOpt(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	check.respond = func(_ string, builds int) (prove.Output, error) {
		// Like the real runner: a killed command reports the context, not an exit code.
		cancel()
		return prove.Output{}, ctx.Err()
	}
	got, err := Start(ctx, opt)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, result %+v", err, got)
	}
	if agents.builds != 0 {
		t.Fatalf("the build agent ran %d times on a cancelled run", agents.builds)
	}
	entries, _ := os.ReadDir(opt.Root)
	for _, entry := range entries {
		if _, ok := readBaseline(t, opt, entry.Name()); ok {
			t.Fatal("a cancelled baseline was written")
		}
	}
}

func TestADirtyTreeGetsNoBaseline(t *testing.T) {
	opt, _, check := baselineOpt(t)
	opt.Source.(*fakeSource).dirty = true
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if check.before != 0 {
		t.Fatalf("baseline calls %d on a dirty tree", check.before)
	}
	if _, ok := readBaseline(t, opt, got.ID); ok {
		t.Fatal("a dirty tree was recorded as the base")
	}
	if got.Verdict != statement.Hold || !strings.Contains(got.Reason, "no progress") {
		t.Fatalf("%+v", got)
	}
	raw, _ := os.ReadFile(filepath.Join(opt.Root, got.ID, "transcript.log"))
	if !strings.Contains(string(raw), "baseline: not recorded: the checkout has uncommitted changes") {
		t.Fatalf("transcript %s", raw)
	}
}

func TestABaselineWhoseHeadMovedIsDiscarded(t *testing.T) {
	opt, _, check := baselineOpt(t)
	source := opt.Source.(*fakeSource)
	check.respond = func(_ string, builds int) (prove.Output, error) {
		if builds == 0 {
			source.sha = "moved"
		}
		return prove.Output{ExitCode: 1, Text: legacyFailure}, nil
	}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := readBaseline(t, opt, got.ID); ok {
		t.Fatal("a baseline taken while HEAD moved was kept")
	}
	if got.Verdict != statement.Hold || !strings.Contains(got.Reason, "no progress") {
		t.Fatalf("%+v", got)
	}
}

func TestATruncatedFailureIsNeverExcused(t *testing.T) {
	opt, _, check := baselineOpt(t)
	check.respond = func(_ string, builds int) (prove.Output, error) {
		return prove.Output{ExitCode: 1, Text: legacyFailure, Truncated: builds > 0}, nil
	}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold || !strings.Contains(got.Reason, "no progress") {
		t.Fatalf("%+v", got)
	}
}

func TestTheFramePromptForbidsEditing(t *testing.T) {
	opt, agents, _ := baselineOpt(t)
	if _, err := Start(context.Background(), opt); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(agents.prompts[0], "Do not edit files.") {
		t.Fatalf("frame prompt:\n%s", agents.prompts[0])
	}
}

func TestBaselineOffRunsNothingExtra(t *testing.T) {
	opt, _, check := baselineOpt(t)
	off := false
	opt.Policy.Prove.Baseline = &off
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if check.before != 0 {
		t.Fatalf("baseline calls %d", check.before)
	}
	if _, ok := readBaseline(t, opt, got.ID); ok {
		t.Fatal("a disabled baseline was written")
	}
	if got.Verdict != statement.Hold || !strings.Contains(got.Reason, "no progress") {
		t.Fatalf("%+v", got)
	}
}

func TestRewindKeepsTheBaselineAndDoesNotRecordAgain(t *testing.T) {
	opt, _, check := baselineOpt(t)
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(filepath.Join(opt.Root, got.ID, "baseline.json"))
	if err := Rewind(opt.Root, got.ID, build); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(opt.Root, got.ID, "baseline.json"))
	if err != nil || string(after) != string(before) {
		t.Fatalf("baseline changed by rewind: %v", err)
	}
	if _, err := Resume(context.Background(), opt, got.ID); err != nil {
		t.Fatal(err)
	}
	if check.before != 1 {
		t.Fatalf("the baseline ran %d times", check.before)
	}
}

func TestAPreexistingFailureLeavesTheBuildPromptAlone(t *testing.T) {
	opt, agents, check := baselineOpt(t)
	opt.Policy.Prove.Acceptance = []string{"go test ./acceptance"}
	check.respond = func(command string, builds int) (prove.Output, error) {
		if command == "go test ./acceptance" && builds == 1 {
			return prove.Output{ExitCode: 1, Text: "--- FAIL: TestCharge (0.02s)\n"}, nil
		}
		if command == "go test ./acceptance" {
			return prove.Output{}, nil
		}
		return prove.Output{ExitCode: 1, Text: legacyFailure}, nil
	}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Advance {
		t.Fatalf("%+v", got)
	}
	last := agents.prompts[len(agents.prompts)-1]
	for _, prompt := range agents.prompts {
		if strings.Contains(prompt, "The prove check failed") {
			last = prompt
		}
	}
	if !strings.Contains(last, "TestCharge") || strings.Contains(last, "TestLegacy") {
		t.Fatalf("the repair packet carried the wrong failure:\n%s", last)
	}
}

// qaReport writes a JUnit report next to the QA command, like a configured reporter.
func qaReport(t *testing.T, workspace string, failed bool) {
	t.Helper()
	body := `<testsuites><testsuite name="pay"><testcase classname="pay" name="TestAC1_Charges"></testcase></testsuite></testsuites>`
	if failed {
		body = `<testsuites><testsuite name="pay"><testcase classname="pay" name="TestAC1_Charges"><failure>boom</failure></testcase></testsuite></testsuites>`
	}
	if err := os.WriteFile(filepath.Join(workspace, "junit.xml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func qaOpt(t *testing.T, after string) (Options, *baseCheck) {
	t.Helper()
	opt, _, check := baselineOpt(t)
	opt.Policy.QA.Command = "qa-cmd"
	opt.Brief = "Charge the card.\n\nAcceptance criteria:\nCharges the card.\n"
	body := "package pay\n\nfunc TestAC1_Charges(t *testing.T) {}\n"
	if err := os.WriteFile(filepath.Join(opt.Workspace, "pay_test.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	check.respond = func(command string, builds int) (prove.Output, error) {
		if command != "qa-cmd" {
			return prove.Output{}, nil
		}
		qaReport(t, opt.Workspace, false)
		if builds == 0 {
			return prove.Output{ExitCode: 1, Text: legacyFailure}, nil
		}
		return prove.Output{ExitCode: 1, Text: after}, nil
	}
	return opt, check
}

func TestQAPassingWithAPreexistingFailureAdvances(t *testing.T) {
	opt, check := qaOpt(t, legacyFailure)
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Advance || got.Stage != note {
		t.Fatalf("%+v calls %v", got, check.calls)
	}
	st, _, err := statement.Read(filepath.Join(opt.Root, got.ID, "iter", "1"), qaSt)
	if err != nil || st.Verdict != statement.Advance || !st.Commands[0].Preexisting {
		t.Fatalf("qa err=%v %+v", err, st)
	}
	if !strings.Contains(st.Reason, "already present on the base commit") {
		t.Fatalf("reason = %s", st.Reason)
	}
	if cur := readCursor(t, opt, got.ID); len(cur.Repairs) != 0 {
		t.Fatalf("repairs %+v", cur.Repairs)
	}
}

func TestQAWithANewFailureLineIteratesAsBefore(t *testing.T) {
	opt, _ := qaOpt(t, legacyFailure+"--- FAIL: TestRefund (0.02s)\n")
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold {
		t.Fatalf("%+v", got)
	}
	st, _, _ := statement.Read(filepath.Join(opt.Root, got.ID, "iter", "1"), qaSt)
	if st.Verdict != statement.Hold || st.Commands[0].Preexisting {
		t.Fatalf("qa %+v", st)
	}
	if cur := readCursor(t, opt, got.ID); len(cur.Repairs) == 0 || cur.Repairs[0].Stage != qaSt {
		t.Fatalf("repairs %+v", cur.Repairs)
	}
}

func TestQAWithAFailingCriterionIsNeverExcused(t *testing.T) {
	opt, check := qaOpt(t, legacyFailure)
	inner := check.respond
	check.respond = func(command string, builds int) (prove.Output, error) {
		out, err := inner(command, builds)
		if command == "qa-cmd" && builds > 0 {
			qaReport(t, opt.Workspace, true)
		}
		return out, err
	}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	st, _, _ := statement.Read(filepath.Join(opt.Root, got.ID, "iter", "1"), qaSt)
	if got.Verdict != statement.Hold || st.Verdict != statement.Hold || st.Commands[0].Preexisting {
		t.Fatalf("%+v qa %+v", got, st)
	}
}

// qaPlain is a story whose QA command writes no per-test report.
func qaPlain(t *testing.T, respond func(command string, builds int) (prove.Output, error)) (Options, *baseCheck) {
	t.Helper()
	opt, _, check := baselineOpt(t)
	opt.Policy.QA.Command = "qa-cmd"
	opt.Brief = "Charge the card.\n\nAcceptance criteria:\nCharges the card.\n"
	body := "package pay\n\nfunc TestAC1_Charges(t *testing.T) {}\n"
	if err := os.WriteFile(filepath.Join(opt.Workspace, "pay_test.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	check.respond = respond
	return opt, check
}

func legacyQA(after prove.Output) func(string, int) (prove.Output, error) {
	return func(command string, builds int) (prove.Output, error) {
		if command != "qa-cmd" {
			return prove.Output{}, nil
		}
		if builds == 0 {
			return prove.Output{ExitCode: 1, Text: legacyFailure}, nil
		}
		return after, nil
	}
}

func TestQAWithoutAReportAdvancesOnAnExcusedExit(t *testing.T) {
	opt, _ := qaPlain(t, legacyQA(prove.Output{ExitCode: 1, Text: legacyFailure}))
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Advance || got.Stage != note {
		t.Fatalf("%+v", got)
	}
	st, _, _ := statement.Read(filepath.Join(opt.Root, got.ID, "iter", "1"), qaSt)
	if st.Verdict != statement.Advance || !st.Commands[0].Preexisting || st.Commands[0].ExitCode != 1 {
		t.Fatalf("qa %+v", st)
	}
	if !strings.Contains(st.Reason, "already present on the base commit") {
		t.Fatalf("reason = %s", st.Reason)
	}
	raw, _ := os.ReadFile(filepath.Join(opt.Root, got.ID, "qa-gate.json"))
	if !strings.Contains(string(raw), `"verdict": "pass"`) {
		t.Fatalf("gate %s", raw)
	}
	if cur := readCursor(t, opt, got.ID); len(cur.Repairs) != 0 {
		t.Fatalf("repairs %+v", cur.Repairs)
	}
}

func TestQAWithoutAReportAndANewLineIteratesAsBefore(t *testing.T) {
	opt, _ := qaPlain(t, legacyQA(prove.Output{ExitCode: 1, Text: legacyFailure + "--- FAIL: TestRefund (0.02s)\n"}))
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	st, _, _ := statement.Read(filepath.Join(opt.Root, got.ID, "iter", "1"), qaSt)
	if got.Verdict != statement.Hold || st.Verdict != statement.Hold || st.Commands[0].Preexisting {
		t.Fatalf("%+v qa %+v", got, st)
	}
	if !strings.Contains(st.Reason, "no per-test report") {
		t.Fatalf("reason = %s", st.Reason)
	}
}

func TestQAWithATruncatedOutputIteratesAsBefore(t *testing.T) {
	opt, _ := qaPlain(t, legacyQA(prove.Output{ExitCode: 1, Text: legacyFailure, Truncated: true}))
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	st, _, _ := statement.Read(filepath.Join(opt.Root, got.ID, "iter", "1"), qaSt)
	if got.Verdict != statement.Hold || st.Verdict != statement.Hold || st.Commands[0].Preexisting {
		t.Fatalf("%+v qa %+v", got, st)
	}
}

func TestQAInconclusiveStillWinsOverAnExcuse(t *testing.T) {
	var opt Options
	calls := 0
	opt, _ = qaPlain(t, func(command string, builds int) (prove.Output, error) {
		if builds == 0 {
			return prove.Output{ExitCode: 1, Text: legacyFailure}, nil
		}
		// After the build the tree stays dirty, so QA runs again instead of reusing prove.
		opt.Source.(*fakeSource).dirty = true
		calls++
		if calls == 1 {
			return prove.Output{}, nil // prove passes on this commit
		}
		return prove.Output{ExitCode: 1, Text: legacyFailure}, nil
	})
	opt.Policy.Run.Weight = "standard"
	opt.Policy.Prove.Fast = []string{"qa-cmd"}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold || !strings.Contains(got.Reason, "inconclusive") {
		t.Fatalf("%+v", got)
	}
}
