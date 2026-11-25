package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"span/internal/policy"
)

func TestThePolicyAndTheEngineAgreeOnWhichStagesCallAnAgent(t *testing.T) {
	if !reflect.DeepEqual(policy.AgentStageNames, AgentStages) {
		t.Fatalf("policy %v engine %v", policy.AgentStageNames, AgentStages)
	}
}

func TestAStageEntryInThePolicyBeatsTheProjectAgentAndLosesToTheRun(t *testing.T) {
	r := &runner{opt: Options{Policy: policy.File{Agent: "claude", Agents: map[string]string{"build": "codex"}}}}
	if got := r.wantedFor("build", ""); got != "codex" {
		t.Fatalf("policy stage entry: %q", got)
	}
	if got := r.wantedFor("build", "cursor"); got != "cursor" {
		t.Fatalf("run choice: %q", got)
	}
	if got := r.wantedFor("frame", ""); got != "claude" {
		t.Fatalf("a stage without an entry follows the project agent: %q", got)
	}
}

func TestAPolicyWithoutAgentsAsksForTheSameCLIAsBefore(t *testing.T) {
	r := &runner{opt: Options{Policy: policy.File{Agent: "claude"}}}
	for _, stage := range AgentStages {
		if got, before := r.wantedFor(stage, ""), r.agentWanted(""); got != before {
			t.Fatalf("%s: %q, before %q", stage, got, before)
		}
	}
}

func TestEachStageRunsOnTheCLIItsEntryNames(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	opt.Policy.Agent = "claude"
	opt.Policy.Agents = map[string]string{"build": "codex"}
	if _, err := Start(context.Background(), opt); err != nil {
		t.Fatal(err)
	}
	if len(agents.wanted) < 2 {
		t.Fatalf("%v", agents.wanted)
	}
	sawCodex, sawClaude := false, false
	for _, want := range agents.wanted {
		sawCodex = sawCodex || want == "codex"
		sawClaude = sawClaude || want == "claude"
	}
	if !sawCodex || !sawClaude {
		t.Fatalf("build should ask for codex and the other stages for claude: %v", agents.wanted)
	}
}

// finishable gives a run on a Jira story the one criterion and the test that declares it, so it
// can reach the end instead of holding at frame or QA.
func finishable(t *testing.T, opt *Options) {
	t.Helper()
	opt.Brief = "AET-12 Charge\n\nAcceptance criteria:\nCharges the card.\n"
	body := "package pay\n\nfunc TestAC1_Charges(t *testing.T) {}\n"
	if err := os.WriteFile(filepath.Join(opt.Workspace, "pay_test.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAFinishedRunMovesItsStoryOnlyWhenTheDoneStatusIsSet(t *testing.T) {
	opt, _, _, _, _, _ := baseOpt(t)
	opt.Issue = "AET-12"
	finishable(t, &opt)
	board := &fakeBoard{}
	opt.Issues = board
	got, err := Start(context.Background(), opt)
	if err != nil || got.Stage != note {
		t.Fatalf("%+v %v", got, err)
	}
	if len(board.moves) != 0 {
		t.Fatalf("a policy without jira.doneStatus moved the story: %q", board.moves)
	}

	opt, _, _, _, _, _ = baseOpt(t)
	opt.Issue = "AET-12"
	finishable(t, &opt)
	opt.Policy.Jira.DoneStatus = "Done"
	board = &fakeBoard{}
	opt.Issues = board
	got, err = Start(context.Background(), opt)
	if err != nil || got.Stage != note {
		t.Fatalf("%+v %v", got, err)
	}
	if len(board.moves) != 1 || board.moves[0] != "AET-12 Done" || !strings.Contains(got.Reason, "moved AET-12 to Done") {
		t.Fatalf("moves %q reason %q", board.moves, got.Reason)
	}
}

func TestAStoryThatCannotBeMovedDoesNotFailTheRun(t *testing.T) {
	opt, _, _, _, _, _ := baseOpt(t)
	opt.Issue = "AET-12"
	finishable(t, &opt)
	opt.Policy.Jira.DoneStatus = "Done"
	board := &fakeBoard{moveErr: errors.New("no such transition")}
	opt.Issues = board
	got, err := Start(context.Background(), opt)
	if err != nil || got.Stage != note || got.Verdict != "advance" {
		t.Fatalf("%+v %v", got, err)
	}
	if !strings.Contains(got.Reason, "was not moved to Done") {
		t.Fatalf("%q", got.Reason)
	}
}
