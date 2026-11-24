package engine

import (
	"context"
	"errors"
	"strings"
	"testing"
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

	// A third resume with nothing changed retries the commit only.
	commits := source.commits
	if _, err := Resume(context.Background(), opt, got.ID); err != nil {
		t.Fatal(err)
	}
	if agents.builds != 2 {
		t.Fatalf("an unchanged blocker ran the agent again: %d builds", agents.builds)
	}
	if source.commits != commits+1 {
		t.Fatalf("the commit was not retried: %d -> %d", commits, source.commits)
	}

	// Once what the hook rejects is fixed, the same resume goes through without the agent.
	source.commitErr = nil
	if _, err := Resume(context.Background(), opt, got.ID); err != nil {
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
