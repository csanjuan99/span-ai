package prove

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"span/internal/statement"
)

func TestFailureLinesStripsOnlyColorsAndTimingTokens(t *testing.T) {
	text := strings.Join([]string{
		"\x1b[31m--- FAIL: TestCharge (0.12s)\x1b[0m",
		"    cart_test.go:12: got 1, want 2",
		"FAIL",
		"FAIL\tspan/pay\t1.234s",
		"ok  \tspan/other\t0.50s",
		"  × charges (7 ms)",
		"Error: timed out after 500ms",
		"",
	}, "\n")
	got := FailureLines(text)
	want := []string{
		"Error: timed out after 500ms",
		"FAIL span/pay",
		"[span/pay] --- FAIL: TestCharge",
		"[span/pay] FAIL",
		"[span/pay] TestCharge: cart_test.go:12: got 1, want 2",
		"× charges",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("lines =\n%q\nwant\n%q", got, want)
	}
	if again := FailureLines(text); !reflect.DeepEqual(again, got) {
		t.Fatalf("not deterministic: %q vs %q", again, got)
	}
}

func TestFailureLinesKeepsRepeats(t *testing.T) {
	got := FailureLines("FAIL x\nFAIL x\n")
	if !reflect.DeepEqual(got, []string{"FAIL x", "FAIL x"}) {
		t.Fatalf("lines %q", got)
	}
}

func TestFailureLinesKeepsDigitsThatAreNotTimes(t *testing.T) {
	a := FailureLines("expected 200 to be 402")
	b := FailureLines("expected 200 to be 403")
	if reflect.DeepEqual(a, b) {
		t.Fatalf("different numbers collapsed: %q", a)
	}
}

func TestFailureLinesIsCapped(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&b, "FAIL case %d\n", i)
	}
	if got := FailureLines(b.String()); len(got) != maxFailureLines {
		t.Fatalf("kept %d lines, want %d", len(got), maxFailureLines)
	}
}

func TestFingerprintIsUnchanged(t *testing.T) {
	// Fingerprints are stored in cursors; this pins the existing algorithm.
	got := Fingerprint("go test ./...", "FAIL src/cart.test.ts > totals\nexpected 2 to be 3\nok other\n")
	want := statement.Signature("go test ./...", "FAIL src/cart.test.ts > totals", "expected 2 to be 3")
	if got != want {
		t.Fatalf("fingerprint %s, want %s", got, want)
	}
}

func baseOf(command string, exit int, text string) Baseline {
	return Baseline{Commands: map[string]BaselineCommand{command: {ExitCode: exit, Lines: FailureLines(text), BuildFailed: buildFailed(text)}}}
}

func TestExcusesTruthTable(t *testing.T) {
	old := "--- FAIL: TestLegacy (0.01s)\n    legacy_test.go:9: got 1, want 2\nFAIL\nFAIL\tspan/legacy\t0.20s\n"
	twoPkgs := "--- FAIL: TestA (0.00s)\nFAIL\nFAIL\tspan/a\t0.1s\n--- FAIL: TestX (0.00s)\nFAIL\nFAIL\tspan/b\t0.1s\n"
	cmd := "go test ./..."
	cases := []struct {
		name string
		base Baseline
		cmd  string
		out  Output
		want bool
	}{
		{"same failure", baseOf(cmd, 1, old), cmd, Output{ExitCode: 1, Text: old}, true},
		{"same failure, other timings and color", baseOf(cmd, 1, old), cmd, Output{ExitCode: 2,
			Text: "\x1b[31m--- FAIL: TestLegacy (0.55s)\x1b[0m\n    legacy_test.go:9: got 1, want 2\nFAIL\nFAIL\tspan/legacy\t0.90s\n"}, true},
		{"a subset of the old failure", baseOf(cmd, 1, old), cmd, Output{ExitCode: 1, Text: "--- FAIL: TestLegacy (0.01s)\nFAIL\tspan/legacy\t0.20s\n"}, true},
		{"one new test", baseOf(cmd, 1, old), cmd, Output{ExitCode: 1, Text: "--- FAIL: TestCharge (0.01s)\n" + old}, false},
		{"a changed unmarked assertion", baseOf(cmd, 1, old), cmd, Output{ExitCode: 1, Text: strings.Replace(old, "got 1", "got 3", 1)}, false},
		{"same test name in another package", baseOf(cmd, 1, twoPkgs), cmd, Output{ExitCode: 1,
			Text: twoPkgs + "--- FAIL: TestA (0.00s)\nFAIL\nFAIL\tspan/b\t0.1s\n"}, false},
		{"one more copy of an old line", baseOf("npm test", 1, "× totals\n"), "npm test", Output{ExitCode: 1, Text: "× totals\n× totals\n"}, false},
		{"another timeout value", baseOf(cmd, 1, "Error: timed out after 30000ms\n"), cmd, Output{ExitCode: 1, Text: "Error: timed out after 500ms\n"}, false},
		{"another received date", baseOf(cmd, 1, "received 2024-01-02T03:04:05Z\n"), cmd, Output{ExitCode: 1, Text: "received 2031-01-02T03:04:05Z\n"}, false},
		{"another clock time", baseOf(cmd, 1, "assert: got 18:00:00\n"), cmd, Output{ExitCode: 1, Text: "assert: got 12:30:01\n"}, false},
		{"another latency", baseOf(cmd, 1, "fail: latency 9.5s\n"), cmd, Output{ExitCode: 1, Text: "fail: latency 1.5s\n"}, false},
		{"no marker lines", baseOf(cmd, 1, old), cmd, Output{ExitCode: 1, Text: "exit status 1\n"}, false},
		{"empty output", baseOf(cmd, 1, old), cmd, Output{ExitCode: 1}, false},
		{"truncated output", baseOf(cmd, 1, old), cmd, Output{ExitCode: 1, Text: old, Truncated: true}, false},
		{"truncated baseline", Baseline{Commands: map[string]BaselineCommand{cmd: {ExitCode: 1, Lines: FailureLines(old), Truncated: true}}},
			cmd, Output{ExitCode: 1, Text: old}, false},
		{"different command", baseOf(cmd, 1, old), "go test ./pay", Output{ExitCode: 1, Text: old}, false},
		{"base passed", baseOf(cmd, 0, old), cmd, Output{ExitCode: 1, Text: old}, false},
		{"current passed", baseOf(cmd, 1, old), cmd, Output{Text: old}, false},
		{"no baseline", Baseline{}, cmd, Output{ExitCode: 1, Text: old}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, why := tc.base.Excuses(tc.cmd, tc.out)
			if got != tc.want {
				t.Fatalf("excused = %v (%s), want %v", got, why, tc.want)
			}
			if why == "" {
				t.Fatal("no reason given")
			}
		})
	}
}

// goV is one package of go test -v output: TestA fails with the given line, TestB passes
// with its own log line, and the timings are whatever the run printed.
func goV(pkg, failLine, passLog, timing string) string {
	return "=== RUN   TestA\n    a_test.go:5: " + failLine + "\n--- FAIL: TestA (" + timing + "s)\n" +
		"=== RUN   TestB\n    b_test.go:3: " + passLog + "\n--- PASS: TestB (" + timing + "s)\n" +
		"FAIL\nFAIL\t" + pkg + "\t0.0" + timing[len(timing)-1:] + "1s\n"
}

func goParallel(failLine, timing string) string {
	return "=== RUN   TestA\n=== PAUSE TestA\n=== RUN   TestB\n=== PAUSE TestB\n=== CONT  TestA\n=== CONT  TestB\n" +
		"=== NAME  TestB\n    b_test.go:3: noise " + timing + "\n=== NAME  TestA\n    a_test.go:5: " + failLine + "\n" +
		"--- PASS: TestB (" + timing + "s)\n--- FAIL: TestA (" + timing + "s)\nFAIL\nFAIL\tdemo/a\t0.02s\n"
}

func goSub(value, timing string) string {
	return "=== RUN   TestA\n=== RUN   TestA/sub\n    a_test.go:9: got " + value + "\n--- FAIL: TestA (" + timing + "s)\n" +
		"    --- FAIL: TestA/sub (" + timing + "s)\nFAIL\nFAIL\tdemo/a\t0.01s\n"
}

func TestExcusesUnderVerboseGoTest(t *testing.T) {
	cmd := "go test -v ./..."
	two := func(a, b string) string { return goV("demo/a", a, "log", "0.00") + goV("demo/b", b, "log", "0.00") }
	cases := []struct {
		name       string
		base, now  string
		wantExcuse bool
	}{
		{"same output, other timings", goV("demo/a", "got 1, want 2", "log", "0.00"), goV("demo/a", "got 1, want 2", "log", "0.31"), true},
		{"another value", goV("demo/a", "got 1, want 2", "log", "0.00"), goV("demo/a", "got 5, want 2", "log", "0.00"), false},
		{"another clock time", goV("demo/a", "got 18:00:00", "log", "0.00"), goV("demo/a", "got 12:30:01", "log", "0.00"), false},
		{"another latency", goV("demo/a", "latency 9.5s", "log", "0.00"), goV("demo/a", "latency 1.5s", "log", "0.00"), false},
		{"a passing test logs something else", goV("demo/a", "got 1, want 2", "log one", "0.00"), goV("demo/a", "got 1, want 2", "log two", "0.00"), true},
		{"subtest, same", goSub("1", "0.00"), goSub("1", "0.40"), true},
		{"subtest, another value", goSub("1", "0.00"), goSub("2", "0.00"), false},
		{"one copy then two", goV("demo/a", "got 1", "log", "0.00"),
			strings.Replace(goV("demo/a", "got 1", "log", "0.00"), "    a_test.go:5: got 1\n", "    a_test.go:5: got 1\n    a_test.go:5: got 1\n", 1), false},
		{"parallel, same", goParallel("got 1, want 2", "0.00"), goParallel("got 1, want 2", "0.20"), true},
		{"parallel, another value", goParallel("got 1, want 2", "0.00"), goParallel("got 5, want 2", "0.00"), false},
		{"-p 4, same", two("got 1", "got 2"), two("got 1", "got 2"), true},
		{"-p 4, another value in the second package", two("got 1", "got 2"), two("got 1", "got 3"), false},
		{"-p 4, the same test now failing in another package", goV("demo/a", "got 1", "log", "0.00"), two("got 1", "got 1"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, why := baseOf(cmd, 1, tc.base).Excuses(cmd, Output{ExitCode: 1, Text: tc.now})
			if got != tc.wantExcuse {
				t.Fatalf("excused = %v (%s), want %v\nbase lines %q\nnow lines %q", got, why, tc.wantExcuse, FailureLines(tc.base), FailureLines(tc.now))
			}
		})
	}
}

func TestFailureLinesUnderVerboseKeepOnlyFailingTests(t *testing.T) {
	got := FailureLines(goV("demo/a", "got 1, want 2", "passing log", "0.00"))
	for _, line := range got {
		if strings.Contains(line, "passing log") || strings.Contains(line, "TestB") {
			t.Fatalf("a passing test reached the failure set: %q", got)
		}
	}
	want := "[demo/a] TestA: a_test.go:5: got 1, want 2"
	found := false
	for _, line := range got {
		found = found || line == want
	}
	if !found {
		t.Fatalf("assertion text missing: %q", got)
	}
}

func TestAPackageThatDoesNotBuildIsNeverExcused(t *testing.T) {
	cmd := "go test ./..."
	broken := func(name string) string {
		return "# demo/c\nc/c.go:3:9: undefined: " + name + "\nFAIL\tdemo/c [build failed]\nFAIL\n"
	}
	if ok, _ := baseOf(cmd, 1, broken("undefinedOne")).Excuses(cmd, Output{ExitCode: 1, Text: broken("undefinedTwo")}); ok {
		t.Fatal("another compile error was excused")
	}
	if ok, _ := baseOf(cmd, 1, broken("undefinedOne")).Excuses(cmd, Output{ExitCode: 1, Text: broken("undefinedOne")}); ok {
		t.Fatal("a package that does not build was excused")
	}
	setup := "FAIL\tdemo/d [setup failed]\n"
	if ok, _ := baseOf(cmd, 1, setup).Excuses(cmd, Output{ExitCode: 1, Text: setup}); ok {
		t.Fatal("a setup failure was excused")
	}
	// A baseline that did not build says nothing about the tests that run now.
	plain := "--- FAIL: TestA (0.00s)\nFAIL\nFAIL\tdemo/c\t0.01s\n"
	base := baseOf(cmd, 1, broken("undefinedOne")+plain)
	if ok, _ := base.Excuses(cmd, Output{ExitCode: 1, Text: plain}); ok {
		t.Fatal("a baseline with a build failure excused a test failure")
	}
}

func TestExcusesRefusesAnOutputPastTheCap(t *testing.T) {
	var b strings.Builder
	for i := 0; i < maxFailureLines; i++ {
		fmt.Fprintf(&b, "FAIL case %03d\n", i)
	}
	base := baseOf("go test ./...", 1, b.String())
	// The extra line sorts after every kept one; a capped comparison would miss it.
	current := b.String() + "FAIL zzz new\n"
	if ok, _ := base.Excuses("go test ./...", Output{ExitCode: 1, Text: current}); ok {
		t.Fatal("a truncated output was excused")
	}
}

func TestBandsWithANilExcuseIsUnchanged(t *testing.T) {
	text := "FAIL x\nexpected 1\n"
	s := &script{fail: map[string]int{"go test ./...": 1}, output: map[string]string{"go test ./...": text}}
	got, err := Bands(context.Background(), s, t.TempDir(), Plan{Fast: []string{"go test ./..."}, Acceptance: []string{"go test ./acceptance"}})
	if err != nil {
		t.Fatal(err)
	}
	want := Report{Outcome: "iterate", Reason: "go test ./... exited 1", Fingerprint: Fingerprint("go test ./...", text),
		Commands: []statement.Command{{Command: "go test ./...", ExitCode: 1, Band: "fast", Output: text, Fingerprint: Fingerprint("go test ./...", text)}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("report\n%+v\nwant\n%+v", got, want)
	}
}

func TestAnExcusedFailureContinuesTheBand(t *testing.T) {
	text := "FAIL legacy\n"
	s := &script{fail: map[string]int{"go test ./...": 1}, output: map[string]string{"go test ./...": text}}
	excuse := func(command string, _ Output) bool { return command == "go test ./..." }
	got, err := Bands(context.Background(), s, t.TempDir(), Plan{Fast: []string{"go test ./..."}, Acceptance: []string{"go test ./acceptance"}, Excuse: excuse})
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome != statement.Advance || got.Fingerprint != "" {
		t.Fatalf("%+v", got)
	}
	if len(s.calls) != 2 {
		t.Fatalf("calls %#v", s.calls)
	}
	if !got.Commands[0].Preexisting || got.Commands[1].Preexisting {
		t.Fatalf("commands %+v", got.Commands)
	}
	if !strings.Contains(got.Reason, "1 failure already present on the base commit was not counted") {
		t.Fatalf("reason = %s", got.Reason)
	}
}

func TestALaterUnexcusedFailureStillFails(t *testing.T) {
	newText := "FAIL new\n"
	s := &script{
		fail:   map[string]int{"go test ./...": 1, "go test ./acceptance": 1},
		output: map[string]string{"go test ./...": "FAIL legacy\n", "go test ./acceptance": newText},
	}
	excuse := func(command string, _ Output) bool { return command == "go test ./..." }
	got, err := Bands(context.Background(), s, t.TempDir(), Plan{Fast: []string{"go test ./..."}, Acceptance: []string{"go test ./acceptance"}, Excuse: excuse})
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome != "iterate" || got.Reason != "go test ./acceptance exited 1" {
		t.Fatalf("%+v", got)
	}
	if got.Fingerprint != Fingerprint("go test ./acceptance", newText) {
		t.Fatalf("fingerprint changed: %s", got.Fingerprint)
	}
}

func TestAnExcuseNeverCoversAStartError(t *testing.T) {
	s := &script{err: map[string]error{"missing": errors.New("executable not found")}}
	excuse := func(string, Output) bool { return true }
	got, err := Bands(context.Background(), s, t.TempDir(), Plan{Fast: []string{"missing"}, Excuse: excuse})
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome != statement.Hold {
		t.Fatalf("outcome %s", got.Outcome)
	}
}

type cancelling struct {
	cancel context.CancelFunc
	calls  int
}

func (c *cancelling) Run(context.Context, string, string) (Output, error) {
	c.calls++
	c.cancel()
	return Output{ExitCode: 1, Text: "FAIL killed"}, nil
}

func TestRecordKeepsWhatRanAndSkipsWhatCannot(t *testing.T) {
	s := &script{
		fail:   map[string]int{"go test ./...": 1},
		output: map[string]string{"go test ./...": "--- FAIL: TestLegacy (0.01s)\n"},
		err:    map[string]error{"missing": errors.New("executable not found")},
	}
	got, notes, err := Record(context.Background(), s, t.TempDir(), "base", []string{"go test ./...", "missing", "a | b", "go test ./...", "go vet ./..."})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s.calls, []string{"go test ./...", "missing", "go vet ./..."}) {
		t.Fatalf("calls %#v", s.calls)
	}
	want := Baseline{SHA: "base", Commands: map[string]BaselineCommand{
		"go test ./...": {ExitCode: 1, Lines: []string{"--- FAIL: TestLegacy"}},
		"go vet ./...":  {ExitCode: 0, Lines: []string{}},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("baseline %+v", got)
	}
	if len(notes) != 4 {
		t.Fatalf("notes %#v", notes)
	}
}

func TestRecordStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &cancelling{cancel: cancel}
	if _, _, err := Record(ctx, c, t.TempDir(), "", []string{"a", "b"}); err == nil {
		t.Fatal("a cancelled baseline was returned")
	}
	if c.calls != 1 {
		t.Fatalf("calls %d", c.calls)
	}
}

type truncating struct{}

func (truncating) Run(context.Context, string, string) (Output, error) {
	return Output{ExitCode: 1, Text: "FAIL tail\n", Truncated: true}, nil
}

func TestRecordMarksATruncatedOutput(t *testing.T) {
	got, _, err := Record(context.Background(), truncating{}, t.TempDir(), "", []string{"go test ./..."})
	if err != nil {
		t.Fatal(err)
	}
	entry := got.Commands["go test ./..."]
	if !entry.Truncated {
		t.Fatalf("entry %+v", entry)
	}
	if ok, _ := got.Excuses("go test ./...", Output{ExitCode: 1, Text: "FAIL tail\n"}); ok {
		t.Fatal("a truncated baseline excused a failure")
	}
}
