package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/prove"
	"span/internal/statement"
)

// preparedOpt is a run in a linked worktree whose wide command fails the way the base
// does, with a prepare command configured.
func preparedOpt(t *testing.T) (Options, *fakeAgents, *baseCheck) {
	t.Helper()
	opt, agents, check := baselineOpt(t)
	opt.Worktree = true
	opt.Project = t.TempDir()
	opt.Policy.Prove.Prepare = []string{"pnpm install --frozen-lockfile"}
	check.respond = func(command string, _ int) (prove.Output, error) {
		if command == "pnpm install --frozen-lockfile" {
			return prove.Output{}, nil
		}
		return prove.Output{ExitCode: 1, Text: legacyFailure}, nil
	}
	return opt, agents, check
}

func TestAWorktreeRunPreparesItsTreeBeforeTheBaseline(t *testing.T) {
	opt, _, check := preparedOpt(t)
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Advance {
		t.Fatalf("%+v", got)
	}
	if len(check.calls) < 2 || check.calls[0] != "pnpm install --frozen-lockfile" || check.calls[1] != "go test ./..." {
		t.Fatalf("prepare must run first, then the baseline: %v", check.calls)
	}
	installs := 0
	for _, c := range check.calls {
		if strings.HasPrefix(c, "pnpm install") {
			installs++
		}
	}
	if installs != 1 {
		t.Fatalf("prepare ran %d times: %v", installs, check.calls)
	}
	if _, ok := readBaseline(t, opt, got.ID); !ok {
		t.Fatal("no baseline after a successful prepare")
	}
}

func TestAFailedPrepareRecordsNoBaseline(t *testing.T) {
	opt, agents, check := preparedOpt(t)
	check.respond = func(command string, _ int) (prove.Output, error) {
		if command == "pnpm install --frozen-lockfile" {
			return prove.Output{ExitCode: 1, Text: "ERR_PNPM_OUTDATED_LOCKFILE"}, nil
		}
		return prove.Output{ExitCode: 1, Text: legacyFailure}, nil
	}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := readBaseline(t, opt, got.ID); ok {
		t.Fatal("a baseline was recorded on a tree whose dependencies failed to install; every failure in it would be excused")
	}
	if agents.builds == 0 {
		t.Fatal("a failed prepare must not stop the run before the agent can fix the tree")
	}
	raw, _ := os.ReadFile(filepath.Join(opt.Root, got.ID, "transcript.log"))
	if !strings.Contains(string(raw), "prepare") {
		t.Fatalf("the transcript does not say why: %s", raw)
	}
}

func TestAPrepareThatCannotStartRecordsNoBaseline(t *testing.T) {
	opt, _, check := preparedOpt(t)
	check.respond = func(command string, _ int) (prove.Output, error) {
		if command == "pnpm install --frozen-lockfile" {
			return prove.Output{}, errors.New("pnpm: not found")
		}
		return prove.Output{ExitCode: 1, Text: legacyFailure}, nil
	}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := readBaseline(t, opt, got.ID); ok {
		t.Fatal("a baseline was recorded after prepare could not start")
	}
}

func TestACheckoutRunNeverRunsPrepare(t *testing.T) {
	opt, _, check := preparedOpt(t)
	opt.Project = opt.Workspace
	if _, err := Start(context.Background(), opt); err != nil {
		t.Fatal(err)
	}
	for _, c := range check.calls {
		if strings.HasPrefix(c, "pnpm install") {
			t.Fatalf("the person's own checkout already has its dependencies: %v", check.calls)
		}
	}
}

func TestAWorktreeWithoutInstalledDependenciesGetsNoBaseline(t *testing.T) {
	opt, _, check := baselineOpt(t)
	opt.Worktree = true
	opt.Project = t.TempDir()
	if err := os.WriteFile(filepath.Join(opt.Workspace, "package.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := readBaseline(t, opt, got.ID); ok {
		t.Fatal("a baseline was recorded with no node_modules; the base would look broken by the environment, not by its tests")
	}
	raw, _ := os.ReadFile(filepath.Join(opt.Root, got.ID, "transcript.log"))
	if !strings.Contains(string(raw), "prove.prepare") {
		t.Fatalf("the transcript must tell the person what to configure: %s", raw)
	}
	if len(check.calls) == 0 {
		t.Fatal("prove itself still has to run")
	}
}

func TestAWorktreeWithDependenciesAlreadyThereStillGetsABaseline(t *testing.T) {
	opt, _, _ := baselineOpt(t)
	opt.Worktree = true
	opt.Project = t.TempDir()
	if err := os.WriteFile(filepath.Join(opt.Workspace, "package.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(opt.Workspace, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := readBaseline(t, opt, got.ID); !ok {
		t.Fatal("no baseline although the dependencies are installed")
	}
}

func TestACancelledPrepareStopsBeforeTheBuild(t *testing.T) {
	opt, agents, check := preparedOpt(t)
	ctx, cancel := context.WithCancel(context.Background())
	check.respond = func(command string, _ int) (prove.Output, error) {
		cancel()
		return prove.Output{}, ctx.Err()
	}
	_, err := Start(ctx, opt)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want a cancelled run, got %v", err)
	}
	if agents.builds != 0 {
		t.Fatal("the build agent was launched on a cancelled context")
	}
}

func TestAPersonsOwnLinkedWorktreeIsNotPrepared(t *testing.T) {
	opt, _, check := preparedOpt(t)
	// The person's checkout happens to be a linked worktree, but it is the project itself.
	opt.Worktree = true
	opt.Project = opt.Workspace
	if _, err := Start(context.Background(), opt); err != nil {
		t.Fatal(err)
	}
	for _, c := range check.calls {
		if strings.HasPrefix(c, "pnpm install") {
			t.Fatalf("prepare ran in a tree that is the project itself: %v", check.calls)
		}
	}
}
