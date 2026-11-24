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

const hookOutput = "git commit -m feat(payments): span x: exit status 1: \n\x1b[0;36mSTATUS: FAILED\x1b[0m\nsrc/Modal.tsx:10 - submit enabled while invalid - keep the button disabled"

func TestARejectedCommitIsAClassifiedBlockerThatStopsSpendingAgentRuns(t *testing.T) {
	opt, agents, source, _, _, _ := baseOpt(t)
	source.commitErr = errors.New(hookOutput)

	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != "hold" || !strings.Contains(got.Reason, "commit rejected") {
		t.Fatalf("first rejection = %s %q", got.Verdict, got.Reason)
	}
	cur := readCursor(t, opt, got.ID)
	if cur.Blocker == nil || cur.Blocker.Class != "commit-rejected" || cur.Blocker.Count != 1 {
		t.Fatalf("blocker = %+v", cur.Blocker)
	}
	if strings.Contains(cur.Blocker.Evidence, "\x1b") || !strings.Contains(cur.Blocker.Evidence, "keep the button disabled") {
		t.Fatalf("evidence = %q", cur.Blocker.Evidence)
	}

	// The first resume still gives the agent a chance, and tells it what the hook said.
	if _, err := Resume(context.Background(), opt, got.ID); err != nil {
		t.Fatal(err)
	}
	if agents.builds != 2 {
		t.Fatalf("builds after first resume = %d", agents.builds)
	}
	if last := agents.prompts[len(agents.prompts)-1]; !strings.Contains(last, "keep the button disabled") {
		t.Fatalf("the build prompt did not carry the hook output:\n%s", last)
	}
	cur = readCursor(t, opt, got.ID)
	if cur.Blocker == nil || cur.Blocker.Count != 2 || cur.Blocker.Retryable() {
		t.Fatalf("blocker after repeat = %+v", cur.Blocker)
	}
	if !strings.Contains(cur.Reason, "cannot pass") {
		t.Fatalf("reason = %q", cur.Reason)
	}

	// The decision is open now: resuming acts on nothing, and the person's answer retries
	// the commit only, without another agent run.
	commits := source.commits
	if _, err := Resume(context.Background(), opt, got.ID); err != nil {
		t.Fatal(err)
	}
	if agents.builds != 2 || source.commits != commits {
		t.Fatalf("an open decision ran something: builds %d commits %d->%d", agents.builds, commits, source.commits)
	}
	if _, err := Decide(context.Background(), opt, got.ID, decisionAnswer("rule-changed", "")); err != nil {
		t.Fatal(err)
	}
	if source.commits != commits+1 || agents.builds != 2 {
		t.Fatalf("the commit was not retried alone: builds %d commits %d->%d", agents.builds, commits, source.commits)
	}
	// Still rejected, the same output comes back and asks again.
	if cur := readCursor(t, opt, got.ID); cur.Decision == nil || cur.Blocker.Count != 3 {
		t.Fatalf("a rejection that did not change was not asked again: %+v", cur.Blocker)
	}

	// Once what the hook rejects is fixed, the person's answer lets the commit through.
	source.commitErr = nil
	if _, err := Decide(context.Background(), opt, got.ID, decisionAnswer("rule-changed", "")); err != nil {
		t.Fatal(err)
	}
	if agents.builds != 2 {
		t.Fatalf("builds = %d", agents.builds)
	}
	cur = readCursor(t, opt, got.ID)
	if cur.Blocker != nil {
		t.Fatalf("a passed commit left the blocker: %+v", cur.Blocker)
	}
}

func TestADifferentRejectionStartsTheCountOver(t *testing.T) {
	opt, _, source, _, _, _ := baseOpt(t)
	source.commitErr = errors.New(hookOutput)
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	source.commitErr = errors.New("STATUS: FAILED\nsrc/Other.tsx:3 - another rule - something else")
	if _, err := Resume(context.Background(), opt, got.ID); err != nil {
		t.Fatal(err)
	}
	cur := readCursor(t, opt, got.ID)
	if cur.Blocker == nil || cur.Blocker.Count != 1 || !cur.Blocker.Retryable() {
		t.Fatalf("blocker = %+v", cur.Blocker)
	}
}

func TestARewindClearsTheBlockerSoTheAgentRunsAgain(t *testing.T) {
	opt, agents, source, _, _, _ := baseOpt(t)
	source.commitErr = errors.New(hookOutput)
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Resume(context.Background(), opt, got.ID); err != nil {
		t.Fatal(err)
	}
	if err := Rewind(opt.Root, got.ID, build); err != nil {
		t.Fatal(err)
	}
	if cur := readCursor(t, opt, got.ID); cur.Blocker != nil {
		t.Fatalf("rewind kept the blocker: %+v", cur.Blocker)
	}
	before := agents.builds
	if _, err := Resume(context.Background(), opt, got.ID); err != nil {
		t.Fatal(err)
	}
	if agents.builds != before+1 {
		t.Fatalf("the rewound run did not build again: %d -> %d", before, agents.builds)
	}
}

// parked returns a run held on a commit the hook keeps rejecting, with its decision open.
func parked(t *testing.T) (Options, *fakeAgents, *fakeSource, Result) {
	t.Helper()
	opt, agents, source, _, _, _ := baseOpt(t)
	source.commitErr = errors.New(hookOutput)
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Resume(context.Background(), opt, got.ID); err != nil {
		t.Fatal(err)
	}
	return opt, agents, source, got
}

func TestARepeatedRejectionRaisesAStructuredDecision(t *testing.T) {
	opt, agents, source, got := parked(t)
	cur := readCursor(t, opt, got.ID)
	d := cur.Decision
	if d == nil || d.Class != "commit-rejected" || d.Validate() != nil {
		t.Fatalf("decision = %+v", d)
	}
	if d.Recommendation != "fix-code" || len(d.Evidence) == 0 || !strings.Contains(d.Evidence[0].Excerpt, "keep the button disabled") {
		t.Fatalf("decision lacks its evidence or recommendation: %+v", d)
	}
	for _, option := range d.Options {
		if strings.Contains(strings.ToLower(option.Label+option.Consequence), "no-verify") {
			t.Fatalf("an option offers to skip the hook: %+v", option)
		}
	}
	// While it is open, resuming does nothing: neither the agent nor the commit runs.
	builds, commits := agents.builds, source.commits
	if _, err := Resume(context.Background(), opt, got.ID); err != nil {
		t.Fatal(err)
	}
	if agents.builds != builds || source.commits != commits {
		t.Fatalf("resume acted on an open decision: builds %d->%d commits %d->%d", builds, agents.builds, commits, source.commits)
	}
}

func TestAnsweringRetryOnlyRetriesTheCommit(t *testing.T) {
	opt, agents, source, got := parked(t)
	source.commitErr = nil
	if _, err := Decide(context.Background(), opt, got.ID, decisionAnswer("rule-changed", "")); err != nil {
		t.Fatal(err)
	}
	if agents.builds != 2 {
		t.Fatalf("builds = %d", agents.builds)
	}
	cur := readCursor(t, opt, got.ID)
	if cur.Decision != nil || cur.Blocker != nil || cur.Stage == "build" {
		t.Fatalf("cursor after retry = stage %s decision %v blocker %v", cur.Stage, cur.Decision, cur.Blocker)
	}
}

func TestAnsweringFixCodeRunsTheAgentAgainWithTheWordsGiven(t *testing.T) {
	opt, agents, source, got := parked(t)
	source.commitErr = nil
	if _, err := Decide(context.Background(), opt, got.ID, decisionAnswer("fix-code", "keep the button disabled")); err != nil {
		t.Fatal(err)
	}
	if agents.builds != 3 {
		t.Fatalf("builds = %d", agents.builds)
	}
	last := agents.prompts[len(agents.prompts)-1]
	if !strings.Contains(last, "keep the button disabled") || !strings.Contains(last, "Change the work to satisfy the hook") {
		t.Fatalf("the answer did not reach the build:\n%s", last)
	}
	if cur := readCursor(t, opt, got.ID); cur.Decision != nil || cur.Blocker != nil {
		t.Fatalf("cursor kept %v %v", cur.Decision, cur.Blocker)
	}
}

func TestAnsweringStopEndsTheRun(t *testing.T) {
	opt, _, _, got := parked(t)
	res, err := Decide(context.Background(), opt, got.ID, decisionAnswer("stop", ""))
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != "stop" {
		t.Fatalf("verdict = %s", res.Verdict)
	}
}

func TestAnAnswerThatIsNotAnOptionChangesNothing(t *testing.T) {
	opt, agents, _, got := parked(t)
	before := readCursor(t, opt, got.ID)
	if _, err := Decide(context.Background(), opt, got.ID, decisionAnswer("skip-the-hook", "")); err == nil {
		t.Fatal("an unknown option was accepted")
	}
	after := readCursor(t, opt, got.ID)
	if after.Decision == nil || after.Decision.ID != before.Decision.ID || agents.builds != 2 {
		t.Fatalf("state changed: %+v builds %d", after.Decision, agents.builds)
	}
	if _, err := Decide(context.Background(), opt, "no-such-run", decisionAnswer("stop", "")); err == nil {
		t.Fatal("a run without a decision was accepted")
	}
}

func decisionAnswer(option, text string) decision.Answer {
	return decision.Answer{Option: option, Text: text}
}

const conflictFile = `{"class":"spec-conflict","title":"AC 400 contradicts AGENTS.md","happened":"The brief wants the button enabled and AGENTS.md wants it disabled.",
"evidence":[{"ref":"brief:AC 400","excerpt":"muestra el rechazo al pulsar"},{"ref":"AGENTS.md:12","excerpt":"disable submit while invalid"}],
"options":[{"id":"follow-rule","label":"Follow AGENTS.md","consequence":"The button is disabled; AC 400 is not met as written.","reversible":true,"then":"build"},
{"id":"stop","label":"Stop","consequence":"The run ends.","reversible":false,"then":"stop"}],"recommendation":"follow-rule"}`

func TestABuildThatWritesADecisionFileParksTheRunInsteadOfCommitting(t *testing.T) {
	opt, agents, source, _, _, _ := baseOpt(t)
	agents.buildHook = func(text string) {
		path := decisionPathFrom(text)
		if err := os.WriteFile(path, []byte(conflictFile), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != "hold" || !strings.Contains(got.Reason, "decision needed") {
		t.Fatalf("result = %s %q", got.Verdict, got.Reason)
	}
	if source.commits != 0 {
		t.Fatalf("a run with an open question committed %d times", source.commits)
	}
	cur := readCursor(t, opt, got.ID)
	if cur.Decision == nil || cur.Decision.Class != "spec-conflict" || cur.Decision.Stage != "build" || len(cur.Decision.Evidence) != 2 {
		t.Fatalf("decision = %+v", cur.Decision)
	}
	if _, err := os.Stat(filepath.Join(opt.Root, got.ID, "decision.json")); err == nil {
		t.Fatal("the request file was left behind to be read twice")
	}
}

func TestAnInvalidDecisionFileHoldsWithWhatIsWrong(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	agents.buildHook = func(text string) {
		path := decisionPathFrom(text)
		bad := `{"title":"x","happened":"y","options":[{"id":"a","label":"A","consequence":"c","then":"skip-hook"},{"id":"b","label":"B","consequence":"c","then":"stop"}]}`
		if err := os.WriteFile(path, []byte(bad), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != "hold" || !strings.Contains(got.Reason, "does something Span does not offer") {
		t.Fatalf("result = %s %q", got.Verdict, got.Reason)
	}
	if cur := readCursor(t, opt, got.ID); cur.Decision != nil {
		t.Fatalf("an invalid request became a decision: %+v", cur.Decision)
	}
}

// decisionPathFrom reads the request file the build prompt names, the way an agent would.
func decisionPathFrom(prompt string) string {
	_, rest, _ := strings.Cut(prompt, "Write this JSON to ")
	path, _, _ := strings.Cut(rest, " and stop")
	return path
}
