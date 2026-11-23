package engine

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"span/internal/prove"
	"span/internal/statement"
)

// changeSource is a fake source that also answers what the branch changed. builds is the
// agent's build count at each call, so a test can tell a baseline-time call from a prove one.
type changeSource struct {
	*fakeSource
	agents  *fakeAgents
	changed []string
	// sequence, when set, answers each call in turn before changed does.
	sequence [][]string
	err      error
	builds   []int
}

func (s *changeSource) Changed(_ context.Context, base string) ([]string, error) {
	if base != "dev" {
		return nil, errors.New("unexpected base " + base)
	}
	s.builds = append(s.builds, s.agents.builds)
	if len(s.sequence) > 0 {
		next := s.sequence[0]
		s.sequence = s.sequence[1:]
		return append([]string{}, next...), s.err
	}
	return append([]string{}, s.changed...), s.err
}

// moduleListing is go list -json output for m/a, m/b importing m/a, and m/c, rooted at dir.
func moduleListing(t *testing.T, dir string) string {
	t.Helper()
	var b strings.Builder
	type listed struct {
		ImportPath string
		Dir        string
		Imports    []string
	}
	for _, pkg := range []listed{
		{ImportPath: "m/a", Dir: filepath.Join(dir, "a")},
		{ImportPath: "m/b", Dir: filepath.Join(dir, "b"), Imports: []string{"m/a"}},
		{ImportPath: "m/c", Dir: filepath.Join(dir, "c")},
	} {
		raw, err := json.Marshal(pkg)
		if err != nil {
			t.Fatal(err)
		}
		b.Write(raw)
		b.WriteString("\n")
	}
	return b.String()
}

// listCheck answers go list with a listing and every other command through the script.
type listCheck struct {
	*scriptCheck
	listing string
}

func (c *listCheck) Run(ctx context.Context, command, dir string) (prove.Output, error) {
	if strings.HasPrefix(command, "go list ") {
		c.calls = append(c.calls, "go list")
		return prove.Output{Text: c.listing}, nil
	}
	return c.scriptCheck.Run(ctx, command, dir)
}

// diffOpt is a run on base dev whose source answers Changed. The baseline is on and the tree
// clean, so a baseline is recorded: a diff focus needs one.
func diffOpt(t *testing.T) (Options, *fakeAgents, *changeSource, *listCheck) {
	t.Helper()
	opt, agents, source, _, _, check := baseOpt(t)
	opt.Base = "dev"
	opt.Policy.Prove.Baseline = nil
	source.dirty = false
	changer := &changeSource{fakeSource: source, agents: agents, changed: []string{"a/a.go"}}
	opt.Source = changer
	lc := &listCheck{scriptCheck: check, listing: moduleListing(t, opt.Workspace)}
	opt.Check = lc
	return opt, agents, changer, lc
}

func proveStatement(t *testing.T, opt Options, id string, iter int) statement.Statement {
	t.Helper()
	st, ok, err := statement.Read(filepath.Join(opt.Root, id, "iter", string(rune('0'+iter))), proveSt)
	if err != nil || !ok {
		t.Fatalf("prove statement %d ok=%v err=%v", iter, ok, err)
	}
	return st
}

func TestAFocusedBandIsDerivedFromTheDiff(t *testing.T) {
	opt, _, _, check := diffOpt(t)
	check.byCommand = map[string]int{"go test m/a m/b": 1}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Advance || got.Stage != note {
		t.Fatalf("%+v", got)
	}
	first := proveStatement(t, opt, got.ID, 1)
	if !first.Attributable || len(first.Commands) != 1 || first.Commands[0].Command != "go test m/a m/b" || first.Commands[0].Band != "focused" {
		t.Fatalf("a focused failure ran more than the focus: %+v", first.Commands)
	}
	second := proveStatement(t, opt, got.ID, 2)
	if len(second.Commands) != 2 || second.Commands[0].Command != "go test m/a m/b" || second.Commands[1].Command != "go test ./..." || second.Commands[1].Band != "wide" {
		t.Fatalf("the focus did not run before the wide band: %+v", second.Commands)
	}
	raw, err := os.ReadFile(filepath.Join(opt.Root, got.ID, "transcript.log"))
	if err != nil || !strings.Contains(string(raw), "prove: focused on the diff: go test m/a m/b") {
		t.Fatalf("transcript %s err %v", raw, err)
	}
	if cur := readCursor(t, opt, got.ID); cur.WideSHA == "" {
		t.Fatal("a wide band that passed was not recorded")
	}
}

// diffRun starts a run and returns every check call and the first prove statement's commands.
func diffRun(t *testing.T, mutate func(*Options, *changeSource)) ([]string, []statement.Command) {
	t.Helper()
	opt, _, changer, check := diffOpt(t)
	mutate(&opt, changer)
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	return check.calls, proveStatement(t, opt, got.ID, 1).Commands
}

func TestWithoutADiffFocusProveRunsExactlyAsBefore(t *testing.T) {
	wantCalls, wantCommands := diffRun(t, func(opt *Options, changer *changeSource) {
		opt.Source = changer.fakeSource
	})
	// The baseline run, then the prove run.
	if !reflect.DeepEqual(wantCalls, []string{"go test ./...", "go test ./..."}) {
		t.Fatalf("calls without a changer = %q", wantCalls)
	}
	off := false
	for name, mutate := range map[string]func(*Options, *changeSource){
		"broad change": func(_ *Options, c *changeSource) {
			c.err = errors.New("the change deletes, renames, copies or retypes a path")
		},
		"diff failed":  func(_ *Options, c *changeSource) { c.err = errors.New("base branch dev is missing") },
		"diff off":     func(opt *Options, _ *changeSource) { opt.Policy.Prove.Diff = &off },
		"no base":      func(opt *Options, _ *changeSource) { opt.Base = "" },
		"only docs":    func(_ *Options, c *changeSource) { c.changed = []string{"README.md"} },
		"all packages": func(_ *Options, c *changeSource) { c.changed = []string{"a/a.go", "c/c.go"} },
	} {
		t.Run(name, func(t *testing.T) {
			calls, commands := diffRun(t, mutate)
			// go list may run before giving up; the checks that decide must not change.
			var checks []string
			for _, call := range calls {
				if call != "go list" {
					checks = append(checks, call)
				}
			}
			if !reflect.DeepEqual(checks, wantCalls) || !reflect.DeepEqual(commands, wantCommands) {
				t.Fatalf("calls %q commands %+v, want %q %+v", calls, commands, wantCalls, wantCommands)
			}
		})
	}
}

func TestACriteriaFocusIsNotReplacedByTheDiff(t *testing.T) {
	opt, _, changer, _ := diffOpt(t)
	opt.Brief = "Charge the card.\n\nAcceptance criteria:\nCharges the card.\n"
	opt.Policy.Prove.Fast = []string{"npm test"}
	if err := os.MkdirAll(filepath.Join(opt.Workspace, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(opt.Workspace, "src", "card.test.ts"), []byte("it('AC-1 charges the card', () => {})\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &runner{opt: opt, cur: Cursor{ID: "run-1", Base: "dev"}}
	plan := r.focusOnDiff(context.Background(), withExcuse(r.provePlan()))
	if !reflect.DeepEqual(plan.Focused, []string{"npm test -- src/card.test.ts"}) || len(changer.builds) != 0 {
		t.Fatalf("focused %q, Changed called %d times", plan.Focused, len(changer.builds))
	}
	opt.Policy.Prove.Focused = []string{"npm test -- {files}"}
	opt.Brief = "Charge the card."
	r = &runner{opt: opt, cur: Cursor{ID: "run-1", Base: "dev"}}
	if plan := r.focusOnDiff(context.Background(), withExcuse(r.provePlan())); len(plan.Focused) != 0 || len(changer.builds) != 0 {
		t.Fatalf("a configured focused band was replaced: %q", plan.Focused)
	}
}

// withExcuse stands for a recorded baseline that excuses nothing.
func withExcuse(plan prove.Plan) prove.Plan {
	plan.Excuse = func(string, prove.Output) bool { return false }
	return plan
}

func TestADiffFocusNeedsABaseline(t *testing.T) {
	opt, _, changer, _ := diffOpt(t)
	r := &runner{opt: opt, cur: Cursor{ID: "run-1", Base: "dev"}}
	before := r.provePlan()
	if plan := r.focusOnDiff(context.Background(), before); !reflect.DeepEqual(plan.Focused, before.Focused) || !reflect.DeepEqual(plan.Fast, before.Fast) || plan.WideBlocks || plan.FocusLabel != "" || len(changer.builds) != 0 {
		t.Fatalf("narrowed without a baseline: %+v", plan)
	}
	plan := r.focusOnDiff(context.Background(), withExcuse(r.provePlan()))
	if !reflect.DeepEqual(plan.Focused, []string{"go test m/a m/b"}) || !reflect.DeepEqual(plan.Fast, []string{"go test ./..."}) || !plan.WideBlocks || plan.FocusLabel != "go test ./..." {
		t.Fatalf("%+v", plan)
	}
	r.cur.GitSHA, r.cur.WideSHA = "abc123", "abc123"
	if plan := r.focusOnDiff(context.Background(), withExcuse(r.provePlan())); len(plan.Fast) != 0 || len(plan.Focused) != 1 {
		t.Fatalf("the wide band reran on a commit it already passed: %+v", plan)
	}
}

func TestWithoutABaselineTheWideBandDecidesAsBefore(t *testing.T) {
	off := false
	for name, mutate := range map[string]func(*Options, *changeSource){
		"baseline off": func(opt *Options, _ *changeSource) { opt.Policy.Prove.Baseline = &off },
		"dirty tree":   func(_ *Options, c *changeSource) { c.dirty = true },
	} {
		t.Run(name, func(t *testing.T) {
			opt, _, changer, check := diffOpt(t)
			mutate(&opt, changer)
			check.byCommand = map[string]int{"go test ./...": 1}
			got, err := Start(context.Background(), opt)
			if err != nil {
				t.Fatal(err)
			}
			first := proveStatement(t, opt, got.ID, 1)
			if !first.Attributable || len(first.Commands) != 1 || first.Commands[0].Command != "go test ./..." || first.Commands[0].Band != "fast" {
				t.Fatalf("the wide failure did not block: %+v", first)
			}
			if len(changer.builds) != 0 {
				t.Fatalf("the diff was read without a baseline: %v", changer.builds)
			}
		})
	}
}

// focusRun is a run with a recorded baseline that passed, whose prove answers come from respond.
func focusRun(t *testing.T, changer func(*changeSource), respond func(command string) (prove.Output, error)) (Options, Result) {
	t.Helper()
	opt, agents, check := baselineOpt(t)
	opt.Base = "dev"
	source := &changeSource{fakeSource: opt.Source.(*fakeSource), agents: agents, changed: []string{"a/a.go"}}
	changer(source)
	opt.Source = source
	listing := moduleListing(t, opt.Workspace)
	check.respond = func(command string, builds int) (prove.Output, error) {
		switch {
		case strings.HasPrefix(command, "go list "):
			return prove.Output{Text: listing}, nil
		case builds == 0:
			return prove.Output{}, nil
		}
		return respond(command)
	}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	return opt, got
}

func TestAWideCommandThatCannotStartHoldsWithADiffFocus(t *testing.T) {
	opt, got := focusRun(t, func(*changeSource) {}, func(command string) (prove.Output, error) {
		if command == "go test ./..." {
			return prove.Output{}, errors.New("executable not found")
		}
		return prove.Output{}, nil
	})
	if got.Verdict != statement.Hold || got.Stage != proveSt || !strings.Contains(got.Reason, "did not run") {
		t.Fatalf("%+v", got)
	}
	if cur := readCursor(t, opt, got.ID); cur.WideSHA != "" {
		t.Fatalf("a wide band that never ran was recorded as passed: %q", cur.WideSHA)
	}
}

func TestTheSameFailureUnderADifferentNarrowingIsNoProgress(t *testing.T) {
	text := "--- FAIL: TestB (0.01s)\nFAIL\tm/b\t0.1s\n"
	opt, got := focusRun(t, func(c *changeSource) {
		c.sequence = [][]string{{"a/a.go"}, {"b/b.go"}}
		c.changed = []string{"b/b.go"}
	}, func(command string) (prove.Output, error) {
		if command == "go test m/a m/b" || command == "go test m/b" {
			return prove.Output{ExitCode: 1, Text: text}, nil
		}
		return prove.Output{}, nil
	})
	if got.Verdict != statement.Hold || !strings.Contains(got.Reason, "no progress") {
		t.Fatalf("%+v", got)
	}
	// The ledger keeps the attempt that held; the hold comes on the second, not a third.
	if cur := readCursor(t, opt, got.ID); len(cur.Repairs) != 2 || cur.Repairs[1].Iteration != 2 || cur.Repairs[0].Fingerprint != cur.Repairs[1].Fingerprint {
		t.Fatalf("the same failure was not recognised: %+v", cur.Repairs)
	}
	second := proveStatement(t, opt, got.ID, 2)
	if second.Commands[0].Command != "go test m/b" {
		t.Fatalf("the statement lost the real command: %+v", second.Commands)
	}
}

func TestANewWideFailureOutsideTheDiffBlocksWithABaseline(t *testing.T) {
	opt, agents, check := baselineOpt(t)
	opt.Base = "dev"
	changer := &changeSource{fakeSource: opt.Source.(*fakeSource), agents: agents, changed: []string{"a/a.go"}}
	opt.Source = changer
	listing := moduleListing(t, opt.Workspace)
	check.respond = func(command string, builds int) (prove.Output, error) {
		switch {
		case strings.HasPrefix(command, "go list "):
			return prove.Output{Text: listing}, nil
		case command == "go test ./..." && builds > 0:
			return prove.Output{ExitCode: 1, Text: "--- FAIL: TestC (0.01s)\nFAIL\tm/c\t0.1s\n"}, nil
		}
		return prove.Output{}, nil
	}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold || !strings.Contains(got.Reason, "no progress") {
		t.Fatalf("%+v", got)
	}
	cur := readCursor(t, opt, got.ID)
	if len(cur.Repairs) == 0 || cur.Repairs[0].Stage != proveSt {
		t.Fatalf("a new wide failure did not spend a repair: %+v", cur.Repairs)
	}
	first := proveStatement(t, opt, got.ID, 1)
	if !first.Attributable || first.Fingerprint != first.Commands[1].Fingerprint || first.Commands[1].Band != "wide" {
		t.Fatalf("%+v", first)
	}
	// The baseline ran only the wide band, before any build, and never asked for the diff.
	if check.before != 1 || check.calls[0] != "go test ./..." {
		t.Fatalf("baseline calls %d: %q", check.before, check.calls)
	}
	for _, builds := range changer.builds {
		if builds == 0 {
			t.Fatalf("the diff was read before the first build: %v", changer.builds)
		}
	}
}
