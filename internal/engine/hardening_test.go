package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/decision"
)

func TestACommitFailureThatIsNotTheHookStaysAPlainHold(t *testing.T) {
	opt, _, source, _, _, _ := baseOpt(t)
	source.commitErr = errors.New("git add -A -- .: exit status 128: fatal: Unable to create index.lock")
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if got, err = Resume(context.Background(), opt, got.ID); err != nil {
			t.Fatal(err)
		}
	}
	cur := readCursor(t, opt, got.ID)
	if cur.Blocker != nil || cur.Decision != nil || !strings.Contains(cur.Reason, "index.lock") {
		t.Fatalf("blocker %+v decision %+v reason %q", cur.Blocker, cur.Decision, cur.Reason)
	}
}

func TestWordsTypedOnAnyOptionReachThePrompt(t *testing.T) {
	opt, agents, got := waitingOnFrame(t)
	takeDecision(t, opt, got.ID)
	if _, err := Reload(context.Background(), opt, got.ID, "Charge the card in USD."); err != nil {
		t.Fatal(err)
	}
	opt.Brief = ""
	if _, err := Decide(context.Background(), opt, got.ID, decision.Answer{Option: "keep", Text: "still true for USD"}); err != nil {
		t.Fatal(err)
	}
	if last := agents.prompts[len(agents.prompts)-1]; !strings.Contains(last, "still true for USD") {
		t.Fatalf("the person's words were filed as history:\n%s", last)
	}
}

const askRetry = `{"title":"Is the hook fixed?","happened":"The commit needs a repository change.",
"options":[{"id":"fixed","label":"I fixed the repository","consequence":"Span tries the commit again.","reversible":true,"then":"retry"},
{"id":"stop","label":"Stop","consequence":"The run ends.","reversible":false,"then":"stop"}]}`

func TestAnAgentQuestionAnsweredWithRetryOnlyTriesTheCommit(t *testing.T) {
	opt, agents, source, _, _, _ := baseOpt(t)
	asked := false
	agents.buildHook = func(text string) {
		if asked {
			return
		}
		asked = true
		if err := os.WriteFile(decisionPathFrom(text), []byte(askRetry), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if source.commits != 0 || agents.builds != 1 {
		t.Fatalf("commits %d builds %d", source.commits, agents.builds)
	}
	if _, err := Decide(context.Background(), opt, got.ID, decisionAnswer("fixed", "")); err != nil {
		t.Fatal(err)
	}
	if agents.builds != 1 || source.commits != 1 {
		t.Fatalf("the answer paid for another agent run: builds %d commits %d", agents.builds, source.commits)
	}
}

func TestADifferentRuleIsAskedEvenAfterAnotherWasAnswered(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	agents.conflicts = []decision.Clash{ac400}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	other := decision.Clash{Story: "show a toast", Rule: "no toasts", Ref: "AGENTS.md:30"}
	agents.conflicts = []decision.Clash{ac400, other}
	if _, err := Decide(context.Background(), opt, got.ID, decisionAnswer("follow-rule", "")); err != nil {
		t.Fatal(err)
	}
	d := readCursor(t, opt, got.ID).Decision
	if d == nil || d.Class != "spec-conflict" || agents.builds != 0 {
		t.Fatalf("the new rule was resolved silently: %+v builds %d", d, agents.builds)
	}
	var refs []string
	for _, e := range d.Evidence {
		refs = append(refs, e.Ref)
	}
	if joined := strings.Join(refs, "|"); !strings.Contains(joined, "AGENTS.md:30") || strings.Contains(joined, "AGENTS.md:12") {
		t.Fatalf("the question should be about the new rule only: %v", refs)
	}
}

func TestReloadingTheSameStoryAfterCriteriaWereInstalledAsksNothing(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	story := opt.Brief
	agents.questions = []string{"Which currency?"}
	agents.acceptance = []string{"Charges the card."}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(opt.Root, got.ID, "brief.md"))
	if err != nil || string(raw) == story {
		t.Fatalf("frame did not install criteria: %q (%v)", raw, err)
	}
	takeDecision(t, opt, got.ID)
	opt.Brief = ""
	if _, err := Reload(context.Background(), opt, got.ID, story); err != nil {
		t.Fatal(err)
	}
	if cur := readCursor(t, opt, got.ID); cur.Decision != nil || cur.BriefEpoch != 0 {
		t.Fatalf("the same story raised %+v at epoch %d", cur.Decision, cur.BriefEpoch)
	}
}

func TestAnApprovalLeftBehindDoesNotReleaseANewPlan(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	opt.Policy.Stages.Plan = "hold"
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	cur := readCursor(t, opt, got.ID)
	if err := os.WriteFile(filepath.Join(opt.Root, got.ID, "approve-frame"), []byte(cur.GitSHA+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Rewind(opt.Root, got.ID, frame); err != nil {
		t.Fatal(err)
	}
	if _, err := Resume(context.Background(), opt, got.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := Decide(context.Background(), opt, got.ID, decisionAnswer("revise", "split it")); err != nil {
		t.Fatal(err)
	}
	if agents.builds != 0 || readCursor(t, opt, got.ID).Decision == nil {
		t.Fatalf("a stale approval built a plan nobody approved: builds %d", agents.builds)
	}
}

func TestARequestFileFromARunThatDiedIsNotTakenByTheNextBuild(t *testing.T) {
	opt, agents, source, _, _, _ := baseOpt(t)
	agents.buildHook = func(text string) {
		if err := os.WriteFile(decisionPathFrom(text), []byte(conflictFile), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	agents.buildErr = errors.New("the agent crashed")
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	agents.buildHook, agents.buildErr = nil, nil
	if _, err := Resume(context.Background(), opt, got.ID); err != nil {
		t.Fatal(err)
	}
	if source.commits == 0 || readCursor(t, opt, got.ID).Decision != nil {
		t.Fatalf("a dead run's question parked this one: commits %d", source.commits)
	}
}

func TestAnOversizedRequestFileIsRefusedWithoutBecomingADecision(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	agents.buildHook = func(text string) {
		big := strings.Repeat("x", decision.MaxFile+10)
		if err := os.WriteFile(decisionPathFrom(text), []byte(big), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := Start(context.Background(), opt)
	if err == nil && got.Verdict == "advance" {
		t.Fatal("an oversized request was ignored")
	}
	if cur := readCursor(t, opt, got.ID); cur.Decision != nil {
		t.Fatalf("an oversized file became a decision: %+v", cur.Decision)
	}
}
