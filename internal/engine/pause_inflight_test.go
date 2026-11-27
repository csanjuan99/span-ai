package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"span/internal/statement"
)

// waitingBuild is a build agent that runs until its context ends, the first time only.
type waitingBuild struct {
	*fakeAgents
	started chan struct{}
	waited  bool
}

func (w *waitingBuild) Build(ctx context.Context, name, workspace, text string) error {
	if w.waited {
		return w.fakeAgents.Build(ctx, name, workspace, text)
	}
	w.waited = true
	close(w.started)
	<-ctx.Done()
	return ctx.Err()
}

func TestAPauseStopsTheRunningAgentAndTheStageRunsAgainOnResume(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	waiting := &waitingBuild{fakeAgents: agents, started: make(chan struct{})}
	opt.Agents = waiting
	id, err := Open(opt)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan Result, 1)
	go func() {
		got, err := Resume(context.Background(), opt, id)
		if err != nil {
			t.Error(err)
		}
		done <- got
	}()
	<-waiting.started
	if err := RequestPause(opt.Root, id); err != nil {
		t.Fatal(err)
	}
	var got Result
	select {
	case got = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the pause did not stop the agent")
	}
	if got.Verdict != statement.Hold || got.Stage != build || !strings.Contains(got.Reason, "paused during build") {
		t.Fatalf("%+v", got)
	}
	if _, err := os.Stat(filepath.Join(opt.Root, id, pauseFile)); !os.IsNotExist(err) {
		t.Fatal("the pause request was not consumed")
	}
	again, err := Resume(context.Background(), opt, id)
	if err != nil {
		t.Fatal(err)
	}
	if again.Verdict != statement.Advance {
		t.Fatalf("the resume did not run: %+v", again)
	}
}
