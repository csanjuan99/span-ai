package prove

import (
	"context"
	"errors"
	"strings"
	"testing"

	"span/internal/statement"
)

type script struct {
	calls  []string
	fail   map[string]int
	err    map[string]error
	output map[string]string
}

func (s *script) Run(_ context.Context, command, _ string) (Output, error) {
	s.calls = append(s.calls, command)
	if err := s.err[command]; err != nil {
		return Output{}, err
	}
	return Output{ExitCode: s.fail[command], Text: s.output[command]}, nil
}

func TestFastFailureSkipsAcceptance(t *testing.T) {
	s := &script{fail: map[string]int{"go test ./...": 1}}
	got, err := Bands(context.Background(), s, t.TempDir(), Plan{Fast: []string{"go test ./..."}, Acceptance: []string{"go test ./acceptance"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome != "iterate" {
		t.Fatalf("outcome %s", got.Outcome)
	}
	if len(s.calls) != 1 {
		t.Fatalf("calls %#v", s.calls)
	}
}

func TestProseHoldsWithoutRunning(t *testing.T) {
	s := &script{}
	got, err := Bands(context.Background(), s, t.TempDir(), Plan{Fast: []string{"go test ./... | wc"}, Acceptance: nil})
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome != statement.Hold || len(s.calls) != 0 {
		t.Fatalf("outcome %s calls %#v", got.Outcome, s.calls)
	}
}

func TestStartErrorHolds(t *testing.T) {
	s := &script{err: map[string]error{"missing": errors.New("executable not found")}}
	got, err := Bands(context.Background(), s, t.TempDir(), Plan{Fast: []string{"missing"}, Acceptance: nil})
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome != statement.Hold {
		t.Fatalf("outcome %s", got.Outcome)
	}
}

func TestBothBandsPass(t *testing.T) {
	s := &script{}
	got, err := Bands(context.Background(), s, t.TempDir(), Plan{Fast: []string{"go test ./..."}, Acceptance: []string{"go test ./acceptance"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome != statement.Advance || len(got.Commands) != 2 {
		t.Fatalf("%+v", got)
	}
}

func TestAFailingBandKeepsTheOutputAndOneFingerprint(t *testing.T) {
	text := "❯ src/cart.test.ts (3 tests | 1 failed)\n FAIL  src/cart.test.ts > totals > sums the lines\n"
	s := &script{
		fail:   map[string]int{"npm test": 1},
		output: map[string]string{"npm test": text},
	}
	got, err := Bands(context.Background(), s, t.TempDir(), Plan{Fast: []string{"npm test"}, Acceptance: nil})
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome != "iterate" {
		t.Fatalf("outcome = %s", got.Outcome)
	}
	item := got.Commands[0]
	if !strings.Contains(item.Output, "sums the lines") {
		t.Fatalf("the output did not travel with the statement: %q", item.Output)
	}
	if item.Fingerprint == "" || got.Fingerprint != item.Fingerprint {
		t.Fatalf("fingerprint %q report %q", item.Fingerprint, got.Fingerprint)
	}
}

func TestTheSameFailureKeepsOneFingerprint(t *testing.T) {
	text := "FAIL src/cart.test.ts > totals > sums the lines\nexpected 2 to be 3\n"
	if Fingerprint("npm test", text) != Fingerprint("npm test", text) {
		t.Fatal("the same failure produced two fingerprints")
	}
	other := "FAIL src/auth.test.ts > session > renews the token\nexpected 401 to be 200\n"
	if Fingerprint("npm test", text) == Fingerprint("npm test", other) {
		t.Fatal("a different failure produced the same fingerprint")
	}
}

func TestAFocusedFailureBlocksAndSkipsTheWideSuite(t *testing.T) {
	s := &script{fail: map[string]int{"npm test -- a.test.ts": 1}}
	got, err := Bands(context.Background(), s, t.TempDir(), Plan{
		Focused: []string{"npm test -- a.test.ts"},
		Fast:    []string{"npm test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome != "iterate" {
		t.Fatalf("outcome = %s", got.Outcome)
	}
	for _, call := range s.calls {
		if call == "npm test" {
			t.Fatal("the wide suite ran after a focused failure")
		}
	}
	if len(got.Commands) != 1 || got.Commands[0].Band != "focused" {
		t.Fatalf("commands = %+v", got.Commands)
	}
}

func TestAWideFailureOnlyReportsWhenAFocusedBandExists(t *testing.T) {
	s := &script{fail: map[string]int{"npm test": 1}}
	got, err := Bands(context.Background(), s, t.TempDir(), Plan{
		Focused: []string{"npm test -- a.test.ts"},
		Fast:    []string{"npm test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome != statement.Advance {
		t.Fatalf("a failure outside the story held the pass: %s", got.Outcome)
	}
	if !strings.Contains(got.Reason, "outside this story") {
		t.Fatalf("reason = %q", got.Reason)
	}
	if len(got.Commands) != 2 || got.Commands[1].Band != "wide" || got.Commands[1].ExitCode != 1 {
		t.Fatalf("commands = %+v", got.Commands)
	}
}

func TestAWideFailureBlocksWhenWideBlocksIsSet(t *testing.T) {
	text := "--- FAIL: TestCharge (0.01s)\nFAIL\tspan/pay\t0.1s\n"
	s := &script{fail: map[string]int{"go test ./...": 1}, output: map[string]string{"go test ./...": text}}
	got, err := Bands(context.Background(), s, t.TempDir(), Plan{
		Focused:    []string{"go test span/a"},
		Fast:       []string{"go test ./..."},
		WideBlocks: true,
		Excuse:     func(string, Output) bool { return false },
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome != "iterate" {
		t.Fatalf("outcome = %s reason %q", got.Outcome, got.Reason)
	}
	if want := Fingerprint("go test ./...", text); got.Fingerprint != want {
		t.Fatalf("fingerprint %q, want the wide one %q", got.Fingerprint, want)
	}
	if got.Reason != "go test ./... exited 1" {
		t.Fatalf("reason = %q", got.Reason)
	}
	if len(got.Commands) != 2 || got.Commands[0].Band != "focused" || got.Commands[1].Band != "wide" {
		t.Fatalf("commands = %+v", got.Commands)
	}
}

func TestAnExcusedWideFailureStillAdvancesWhenWideBlocksIsSet(t *testing.T) {
	s := &script{fail: map[string]int{"go test ./...": 1}}
	got, err := Bands(context.Background(), s, t.TempDir(), Plan{
		Focused:    []string{"go test span/a"},
		Fast:       []string{"go test ./..."},
		WideBlocks: true,
		Excuse:     func(string, Output) bool { return true },
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome != statement.Advance || got.Fingerprint != "" {
		t.Fatalf("%+v", got)
	}
	if !got.Commands[1].Preexisting {
		t.Fatalf("commands = %+v", got.Commands)
	}
}

func TestWideBlocksOffKeepsTheReportOnlyReason(t *testing.T) {
	s := &script{fail: map[string]int{"npm test": 1}}
	got, err := Bands(context.Background(), s, t.TempDir(), Plan{
		Focused: []string{"npm test -- a.test.ts"},
		Fast:    []string{"npm test"},
		Excuse:  func(string, Output) bool { return false },
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome != statement.Advance || got.Reason != "focused passed; npm test exited 1 outside this story" {
		t.Fatalf("outcome %s reason %q", got.Outcome, got.Reason)
	}
}

func TestTheWideSuiteStillDecidesWithoutAFocusedBand(t *testing.T) {
	s := &script{fail: map[string]int{"npm test": 1}}
	got, err := Bands(context.Background(), s, t.TempDir(), Plan{Fast: []string{"npm test"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome != "iterate" {
		t.Fatalf("outcome = %s", got.Outcome)
	}
	if got.Commands[0].Band != "fast" {
		t.Fatalf("band = %s", got.Commands[0].Band)
	}
}

func TestDeriveFocusKnowsTheCommonRunners(t *testing.T) {
	files := []string{"src/cartera/statement.test.tsx", "src/cartera/sidebar.test.tsx"}
	cases := map[string]string{
		"npm test":            "npm test -- src/cartera/statement.test.tsx src/cartera/sidebar.test.tsx",
		"npm run test:unit":   "npm run test:unit -- src/cartera/statement.test.tsx src/cartera/sidebar.test.tsx",
		"npx vitest run":      "npx vitest run src/cartera/statement.test.tsx src/cartera/sidebar.test.tsx",
		"pytest":              "pytest src/cartera/statement.test.tsx src/cartera/sidebar.test.tsx",
		"npx playwright test": "npx playwright test src/cartera/statement.test.tsx src/cartera/sidebar.test.tsx",
	}
	for command, want := range cases {
		got := DeriveFocus([]string{command}, files)
		if len(got) != 1 || got[0] != want {
			t.Fatalf("%q derived %v, want %q", command, got, want)
		}
	}
}

func TestDeriveFocusGivesUpHonestly(t *testing.T) {
	files := []string{"a.test.ts"}
	for _, command := range []string{"go test ./...", "make test", "cargo test"} {
		if got := DeriveFocus([]string{command}, files); got != nil {
			t.Fatalf("%q invented a scope: %v", command, got)
		}
	}
	if got := DeriveFocus([]string{"npm test"}, nil); got != nil {
		t.Fatalf("a scope appeared without files: %v", got)
	}
}

func TestAWideCommandThatCannotStartHoldsWhenWideBlocksIsSet(t *testing.T) {
	s := &script{err: map[string]error{"go test ./...": errors.New("executable not found")}}
	got, err := Bands(context.Background(), s, t.TempDir(), Plan{
		Focused:    []string{"go test m/a"},
		Fast:       []string{"go test ./..."},
		WideBlocks: true,
		Excuse:     func(string, Output) bool { return false },
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome != statement.Hold {
		t.Fatalf("outcome = %s reason %q", got.Outcome, got.Reason)
	}
}

func TestAFocusLabelKeepsTheFingerprintAcrossNarrowedCommands(t *testing.T) {
	text := "--- FAIL: TestCharge (0.01s)\nFAIL\tm/b\t0.1s\n"
	fingerprint := func(focused string) string {
		s := &script{fail: map[string]int{focused: 1}, output: map[string]string{focused: text}}
		got, err := Bands(context.Background(), s, t.TempDir(), Plan{
			Focused:    []string{focused},
			Fast:       []string{"go test ./..."},
			FocusLabel: "go test ./...",
		})
		if err != nil {
			t.Fatal(err)
		}
		if got.Outcome != "iterate" || got.Commands[0].Command != focused || got.Commands[0].Fingerprint != got.Fingerprint {
			t.Fatalf("%+v", got)
		}
		return got.Fingerprint
	}
	if fingerprint("go test m/a m/b") != fingerprint("go test m/b") {
		t.Fatal("the same failure got two fingerprints because the package list changed")
	}
	if fingerprint("go test m/b") != Fingerprint("go test ./...", text) {
		t.Fatal("the label was not used for the fingerprint")
	}
}
