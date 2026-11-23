package engine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/agent"
	"span/internal/policy"
	"span/internal/statement"
)

// emptyHome is a home directory with no harness markers, so Gentle AI is missing.
func emptyHome(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// piHome is a home directory where only Pi has Gentle AI.
func piHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	marker := filepath.Join(dir, ".pi", "agent", "npm", "node_modules", "gentle-pi")
	if err := os.MkdirAll(marker, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func spanDirFor(t *testing.T, opt Options, id string) string {
	t.Helper()
	dir := filepath.Join(opt.Root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestTheChoiceAppliesToTheStageItNames(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	dir := spanDirFor(t, opt, "run-1")
	if err := SetStageAgent(dir, build, "pi", piHome(t)); err != nil {
		t.Fatal(err)
	}
	r := &runner{opt: opt, cur: Cursor{ID: "run-1"}}

	if _, err := r.pick(build); err != nil {
		t.Fatal(err)
	}
	if got := agents.wanted[len(agents.wanted)-1]; got != "pi" {
		t.Fatalf("build asked for %q, want pi", got)
	}

	if _, err := r.pick(frame); err != nil {
		t.Fatal(err)
	}
	if got := agents.wanted[len(agents.wanted)-1]; got != "" {
		t.Fatalf("frame asked for %q, want the policy default", got)
	}
}

func TestInspectKeepsADifferentAgentWhenNothingWasChosen(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	spanDirFor(t, opt, "run-1")
	r := &runner{opt: opt, cur: Cursor{ID: "run-1"}}

	got, err := r.pick(inspect)
	if err != nil {
		t.Fatal(err)
	}
	if got != "claude" {
		t.Fatalf("inspect runs %q, want the second ready CLI", got)
	}
	if agents.wanted[len(agents.wanted)-1] != "" {
		t.Fatalf("inspect asked for %q", agents.wanted[len(agents.wanted)-1])
	}
}

func TestChoosingAnAgentForInspectUsesThatAgent(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	dir := spanDirFor(t, opt, "run-1")
	if err := SetStageAgent(dir, inspect, "pi", piHome(t)); err != nil {
		t.Fatal(err)
	}
	r := &runner{opt: opt, cur: Cursor{ID: "run-1"}}
	if _, err := r.pick(inspect); err != nil {
		t.Fatal(err)
	}
	if got := agents.wanted[len(agents.wanted)-1]; got != "pi" {
		t.Fatalf("inspect asked for %q, want pi", got)
	}
}

func TestAnEmptyChoiceClearsTheException(t *testing.T) {
	dir := t.TempDir()
	if err := SetStageAgent(dir, build, "pi", piHome(t)); err != nil {
		t.Fatal(err)
	}
	if ReadStageAgents(dir)[build] != "pi" {
		t.Fatalf("choices = %v", ReadStageAgents(dir))
	}
	if err := SetStageAgent(dir, build, "any", piHome(t)); err != nil {
		t.Fatal(err)
	}
	if _, kept := ReadStageAgents(dir)[build]; kept {
		t.Fatalf("the exception survived: %v", ReadStageAgents(dir))
	}
}

func TestAStageWithoutAModelCannotCarryAChoice(t *testing.T) {
	err := SetStageAgent(t.TempDir(), "prove", "pi", piHome(t))
	if err == nil {
		t.Fatal("prove accepted an agent")
	}
	for _, want := range []string{"frame", "build", "inspect"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("the message does not name %s: %v", want, err)
		}
	}
}

func TestAnUnknownAgentIsRefused(t *testing.T) {
	err := SetStageAgent(t.TempDir(), build, "gpt", piHome(t))
	if err == nil {
		t.Fatal("an unknown agent was written")
	}
	if !strings.Contains(err.Error(), "cursor") {
		t.Fatalf("the message does not list the registry: %v", err)
	}
}

func TestAnAgentThatCannotRunIsRefused(t *testing.T) {
	err := SetStageAgent(t.TempDir(), build, "claude", emptyHome(t))
	if err == nil {
		t.Fatal("an agent without Gentle AI was written")
	}
	if !strings.Contains(err.Error(), "Gentle AI") {
		t.Fatalf("the message does not say what is missing: %v", err)
	}
}

func TestABrokenChoicesFileFallsBackToThePolicy(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, agentsFile), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := ReadStageAgents(dir); len(got) != 0 {
		t.Fatalf("a broken file produced %v", got)
	}
}

// pauseOn writes a pause request when the run first saves its cursor, which is before
// the first stage boundary the runner will check.
type pauseOn struct {
	root  string
	wrote bool
}

func (p *pauseOn) Publish(id string, event Event) {
	if p.wrote || event.Cursor == nil || event.Cursor.Stage != frame {
		return
	}
	p.wrote = true
	if err := RequestPause(p.root, id); err != nil {
		panic(err)
	}
}

func TestAPauseHoldsBeforeTheNextStageAndIsConsumed(t *testing.T) {
	opt, _, _, _, _, _ := baseOpt(t)
	watcher := &pauseOn{root: opt.Root}
	opt.Watch = watcher

	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(got.Reason, "paused before frame") {
		t.Fatalf("reason = %s", got.Reason)
	}
	if _, err := os.Stat(filepath.Join(opt.Root, got.ID, pauseFile)); !os.IsNotExist(err) {
		t.Fatal("the pause request was not consumed")
	}

	opt.Watch = nil
	again, err := Resume(context.Background(), opt, got.ID)
	if err != nil {
		t.Fatal(err)
	}
	if again.Verdict != statement.Advance {
		t.Fatalf("the resume did not run: %+v", again)
	}
}

func TestAPauseWrittenWhileStoppedIsDiscardedOnResume(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	agents.questions = []string{"Which permission atom?"}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.Reason, "open questions") {
		t.Fatalf("reason = %s", got.Reason)
	}
	if err := RequestPause(opt.Root, got.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := Resume(context.Background(), opt, got.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(opt.Root, got.ID, pauseFile)); !os.IsNotExist(err) {
		t.Fatal("a pause written while stopped survived the resume")
	}
}

func TestAFailedRunIsLeftHeldInsteadOfLookingIdle(t *testing.T) {
	opt, _, _, _, _, _ := baseOpt(t)
	id := "2026-01-01-0000-dead"
	dir := filepath.Join(opt.Root, id)
	if err := os.MkdirAll(filepath.Join(dir, "iter", "1"), 0o755); err != nil {
		t.Fatal(err)
	}
	cur := Cursor{ID: id, Stage: frame, Iteration: 1, Hold: true, Workspace: opt.Workspace}
	raw, err := json.Marshal(cur)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cursor.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	// A statement that cannot be read is enough to fail the loop.
	if err := os.WriteFile(filepath.Join(dir, "iter", "1", "frame.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Resume(context.Background(), opt, id); err == nil {
		t.Fatal("a broken statement did not fail the run")
	}
	got := readCursor(t, opt, id)
	if !got.Hold {
		t.Fatalf("the run is not held after a failure: %+v", got)
	}
	if !strings.Contains(got.Reason, "the run stopped") {
		t.Fatalf("reason = %q", got.Reason)
	}
}

func TestASameCommandOnADifferentCommitIsNotInconclusive(t *testing.T) {
	opt, _, _, _, _, check := baseOpt(t)
	opt.Policy.QA = policy.QA{Command: "go test ./..."}
	opt.Policy.Repair = policy.Repair{MaxAttempts: 3}
	opt.Brief = "Charge the card.\n\nAcceptance criteria:\nCharges the card.\n"
	body := "package pay\n\nfunc TestAC1_Charges(t *testing.T) {}\n"
	if err := os.WriteFile(filepath.Join(opt.Workspace, "pay_test.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	check.failAfter = map[string]int{"go test ./...": 1}
	check.texts = map[string]string{"go test ./...": "FAIL src/cart.test.ts > totals expected 2 to be 3"}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	// The second build commits again, so QA runs on a new commit and the failure is
	// attributable: it must repair, not hold as inconclusive.
	if got.Verdict == statement.Hold && strings.Contains(got.Reason, "inconclusive") {
		t.Fatalf("a different commit was read as inconclusive: %s", got.Reason)
	}
}

func TestTheFocusedBandUsesTheDeclaringFiles(t *testing.T) {
	opt, _, _, _, _, _ := baseOpt(t)
	opt.Brief = "Charge the card.\n\nAcceptance criteria:\nCharges the card.\n"
	opt.Policy.Prove.Focused = []string{"npm test -- {files}"}
	body := "it('AC-1 charges the card', () => {})\n"
	if err := os.MkdirAll(filepath.Join(opt.Workspace, "src", "cartera"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(opt.Workspace, "src", "cartera", "statement.test.tsx"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &runner{opt: opt, cur: Cursor{ID: "run-1"}}
	plan := r.provePlan()
	if len(plan.Focused) != 1 {
		t.Fatalf("focused = %v", plan.Focused)
	}
	if plan.Focused[0] != "npm test -- src/cartera/statement.test.tsx" {
		t.Fatalf("focused = %q", plan.Focused[0])
	}
}

func TestWithoutDeclarationsTheWideBandStillDecides(t *testing.T) {
	opt, _, _, _, _, _ := baseOpt(t)
	opt.Brief = "Charge the card.\n\nAcceptance criteria:\nCharges the card.\n"
	opt.Policy.Prove.Focused = []string{"npm test -- {files}"}
	r := &runner{opt: opt, cur: Cursor{ID: "run-1"}}
	plan := r.provePlan()
	if len(plan.Focused) != 0 {
		t.Fatalf("a scope was invented: %v", plan.Focused)
	}
	if len(plan.Fast) == 0 {
		t.Fatal("the wide band was dropped")
	}
}

func TestTheWideSuiteRunsOncePerCommit(t *testing.T) {
	opt, _, _, _, _, _ := baseOpt(t)
	opt.Policy.Run = policy.Run{Weight: policy.Standard}
	opt.Brief = "Charge the card.\n\nAcceptance criteria:\nCharges the card.\n"
	opt.Policy.Prove.Focused = []string{"npm test -- {files}"}
	body := "it('AC-1 charges the card', () => {})\n"
	if err := os.WriteFile(filepath.Join(opt.Workspace, "pay.test.ts"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &runner{opt: opt, cur: Cursor{ID: "run-1", GitSHA: "abc123", WideSHA: "abc123"}}
	plan := r.provePlan()
	if len(plan.Fast) != 0 {
		t.Fatalf("the wide suite was queued again for the same commit: %v", plan.Fast)
	}
	if len(plan.Focused) != 1 {
		t.Fatalf("focused = %v", plan.Focused)
	}

	r.cur.WideSHA = "old999"
	if len(r.provePlan().Fast) == 0 {
		t.Fatal("the wide suite was skipped on a new commit")
	}
}

func TestRewindRestartsAtAStageWithAFreshStreak(t *testing.T) {
	root := t.TempDir()
	id := "2026-01-01-0000-rew1"
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cur := Cursor{
		ID: id, Stage: proveSt, Iteration: 3, Hold: true, GitSHA: "abc123", WideSHA: "abc123",
		Repairs:    []Repair{{Iteration: 3, Stage: proveSt, Fingerprint: "aa"}},
		AgentBlock: &AgentBlock{Kind: agent.Quota, Agent: "claude"},
		Questions:  []string{"which one"},
	}
	raw, err := json.Marshal(cur)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cursor.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Rewind(root, id, build); err != nil {
		t.Fatal(err)
	}
	got, err := readCursorFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Stage != build || got.Iteration != 4 {
		t.Fatalf("stage=%s iteration=%d", got.Stage, got.Iteration)
	}
	if got.Hold || got.Stop || got.Done {
		t.Fatalf("the run is still paused: %+v", got)
	}
	if len(got.Repairs) != 0 || got.WideSHA != "" || got.AgentBlock != nil || len(got.Questions) != 0 {
		t.Fatalf("the restarted run kept old state: %+v", got)
	}
	if got.GitSHA != "abc123" {
		t.Fatalf("the commit was lost: %q", got.GitSHA)
	}
}

func TestRewindRefusesAStageItCannotStartFrom(t *testing.T) {
	root := t.TempDir()
	id := "2026-01-01-0000-rew2"
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(Cursor{ID: id, Stage: proveSt, Iteration: 1})
	if err := os.WriteFile(filepath.Join(dir, "cursor.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, stage := range []string{intake, "nope", ""} {
		if err := Rewind(root, id, stage); err == nil {
			t.Fatalf("stage %q was accepted", stage)
		}
	}
}

func TestARestartedRunGetsItsOwnRepairBudget(t *testing.T) {
	opt, _, _, _, _, check := baseOpt(t)
	opt.Policy.Repair = policy.Repair{MaxAttempts: 1}
	check.failAfter = map[string]int{"go test ./...": 0}
	check.texts = map[string]string{"go test ./...": "FAIL case"}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.Reason, "iteration cap 1") {
		t.Fatalf("reason = %s", got.Reason)
	}
	// A person restarts it from build: the cap must not fire on the first new failure.
	if err := Rewind(opt.Root, got.ID, build); err != nil {
		t.Fatal(err)
	}
	before := len(check.calls)
	if _, err := Resume(context.Background(), opt, got.ID); err != nil {
		t.Fatal(err)
	}
	// A fresh streak means the restarted run repairs at least once more: more than one
	// prove pass after the restart.
	if after := len(check.calls) - before; after < 2 {
		t.Fatalf("the restarted run inherited the spent budget: %d prove passes", after)
	}
}

func TestALightStoryRunsOnlyItsOwnTests(t *testing.T) {
	opt, _, _, _, _, _ := baseOpt(t)
	opt.Brief = "Show the balance.\n\nAcceptance criteria:\nShows the balance.\n"
	opt.Policy.Prove = policy.Bands{Focused: []string{"npm test -- {files}"}, Fast: []string{"npm test"}}
	body := "it('AC-1 shows the balance', () => {})\n"
	if err := os.WriteFile(filepath.Join(opt.Workspace, "cart.test.ts"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &runner{opt: opt, cur: Cursor{ID: "run-1"}}
	plan := r.provePlan()
	if len(plan.Focused) != 1 {
		t.Fatalf("focused = %v", plan.Focused)
	}
	if len(plan.Fast) != 0 {
		t.Fatalf("a fix paid for the whole suite: %v", plan.Fast)
	}
}

func TestAProvePassIsNeverLeftWithNothingToRun(t *testing.T) {
	opt, _, _, _, _, _ := baseOpt(t)
	opt.Brief = "Show the balance.\n\nAcceptance criteria:\nShows the balance.\n"
	r := &runner{opt: opt, cur: Cursor{ID: "run-1"}}
	plan := r.provePlan()
	if len(plan.Focused) != 0 {
		t.Fatalf("focused was invented: %v", plan.Focused)
	}
	if len(plan.Fast) == 0 {
		t.Fatal("prove was left with no command at all")
	}
}

func TestADeepStoryEarnsMoreRepairs(t *testing.T) {
	opt, _, _, _, _, check := baseOpt(t)
	opt.Brief = "Charge.\n\nAcceptance criteria:\n" + strings.Repeat("A criterion.\n", 20)
	body := "package pay\n\nfunc TestAC1_Charges(t *testing.T) {}\n"
	if err := os.WriteFile(filepath.Join(opt.Workspace, "pay_test.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	check.failAfter = map[string]int{"go test ./...": 0}
	check.texts = map[string]string{"go test ./...": "FAIL case"}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.Reason, "iteration cap 6") {
		t.Fatalf("a deep story did not earn six repairs: %s", got.Reason)
	}
}
