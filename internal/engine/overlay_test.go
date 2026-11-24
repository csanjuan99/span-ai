package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"span/internal/overlay"
)

// waitingOnFrame returns a run held on a question frame asked, ready to be reloaded.
func waitingOnFrame(t *testing.T) (Options, *fakeAgents, Result) {
	t.Helper()
	opt, agents, _, _, _, _ := baseOpt(t)
	agents.questions = []string{"Which currency?"}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if !Waiting(readCursor(t, opt, got.ID)) {
		t.Fatal("the run is not waiting")
	}
	return opt, agents, got
}

func takeDecision(t *testing.T, opt Options, id string) {
	t.Helper()
	_, err := overlay.Append(filepath.Join(opt.Root, id), overlay.Entry{
		Kind: overlay.Decision, Stage: "build", Title: "AC 400", Option: "follow-rule",
		Text: "Follow AGENTS.md: keep the button disabled", Dedupe: "ac400", Time: time.Unix(60, 0),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestADecisionIsInThePromptOfEveryStage(t *testing.T) {
	opt, _, got := waitingOnFrame(t)
	takeDecision(t, opt, got.ID)
	r := &runner{opt: opt, cur: readCursor(t, opt, got.ID)}
	for _, stage := range []string{frame, build, "inspect"} {
		in := r.promptInput(stage, "")
		if !strings.Contains(in.Decisions, "keep the button disabled") || !strings.Contains(in.Decisions, "[D1]") {
			t.Fatalf("%s did not get the decision: %q", stage, in.Decisions)
		}
	}
}

func TestAReloadedStoryRaisesAQuestionAboutADecisionTakenOnTheOldOne(t *testing.T) {
	opt, agents, got := waitingOnFrame(t)
	takeDecision(t, opt, got.ID)
	prompts := len(agents.prompts)

	if _, err := Reload(context.Background(), opt, got.ID, "Charge the card in USD."); err != nil {
		t.Fatal(err)
	}
	cur := readCursor(t, opt, got.ID)
	d := cur.Decision
	if d == nil || d.Class != "overlay-stale" || d.Validate() != nil || !strings.Contains(d.Evidence[0].Excerpt, "keep the button disabled") {
		t.Fatalf("decision = %+v", d)
	}
	if len(agents.prompts) != prompts {
		t.Fatal("the agent ran before the person said whether the decision still holds")
	}

	// Keeping it anchors it to the new story and the run goes on with it in the prompt. A
	// wake never carries the story: the engine reads brief.md, which Reload rewrote.
	opt.Brief = ""
	if _, err := Decide(context.Background(), opt, got.ID, decisionAnswer("keep", "")); err != nil {
		t.Fatal(err)
	}
	last := agents.prompts[len(agents.prompts)-1]
	if !strings.Contains(last, "keep the button disabled") || strings.Contains(last, "earlier version") {
		t.Fatalf("the kept decision is not in the prompt as current:\n%s", last)
	}
	if !strings.Contains(last, "Charge the card in USD.") {
		t.Fatalf("the reloaded story is not in the prompt:\n%s", last)
	}
}

func TestDroppingAStaleDecisionTakesItOutOfThePromptButNotOutOfTheLog(t *testing.T) {
	opt, agents, got := waitingOnFrame(t)
	takeDecision(t, opt, got.ID)
	if _, err := Reload(context.Background(), opt, got.ID, "Charge the card in USD."); err != nil {
		t.Fatal(err)
	}
	if _, err := Decide(context.Background(), opt, got.ID, decisionAnswer("drop", "")); err != nil {
		t.Fatal(err)
	}
	if last := agents.prompts[len(agents.prompts)-1]; strings.Contains(last, "keep the button disabled") {
		t.Fatalf("a dropped decision still reaches the agent:\n%s", last)
	}
	raw, err := os.ReadFile(filepath.Join(opt.Root, got.ID, overlay.File))
	if err != nil || !strings.Contains(string(raw), "keep the button disabled") {
		t.Fatalf("the log lost the decision: %s (%v)", raw, err)
	}
}

func TestReloadingTheSameStoryDoesNotQuestionDecisions(t *testing.T) {
	opt, _, got := waitingOnFrame(t)
	takeDecision(t, opt, got.ID)
	if _, err := Reload(context.Background(), opt, got.ID, opt.Brief); err != nil {
		t.Fatal(err)
	}
	if cur := readCursor(t, opt, got.ID); cur.Decision != nil || cur.BriefEpoch != 0 {
		t.Fatalf("an unchanged story raised %+v at epoch %d", cur.Decision, cur.BriefEpoch)
	}
}

func TestAnswersThatGuideTheBuildAreLoggedAndTheOthersAreOnlyHistory(t *testing.T) {
	opt, _, source, got := parked(t)
	source.commitErr = nil
	if _, err := Decide(context.Background(), opt, got.ID, decisionAnswer("fix-code", "keep the button disabled")); err != nil {
		t.Fatal(err)
	}
	entries, err := overlay.Read(filepath.Join(opt.Root, got.ID))
	if err != nil {
		t.Fatal(err)
	}
	active := overlay.Active(entries)
	if len(active) != 1 || !strings.Contains(active[0].Text, "keep the button disabled") || active[0].Option != "fix-code" {
		t.Fatalf("active = %+v", active)
	}
}
