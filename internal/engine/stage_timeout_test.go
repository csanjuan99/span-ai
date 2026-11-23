package engine

import (
	"context"
	"strings"
	"testing"
	"time"

	"span/internal/statement"
)

func TestAnAgentThatRunsPastTheStageTimeoutIsStoppedAndTheStageHolds(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	agents.buildHang = true
	opt.Policy.Limits.StageTimeout = "1s"
	started := time.Now()
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(started) > 10*time.Second {
		t.Fatal("the hung agent was not stopped")
	}
	if got.Verdict != statement.Hold || !strings.Contains(got.Reason, "limits.stageTimeout") || !strings.Contains(got.Reason, "build") {
		t.Fatalf("%+v", got)
	}
	if agents.builds != 1 {
		t.Fatalf("a timeout spent a repair: %d builds", agents.builds)
	}
}

func TestWithoutAStageTimeoutNothingIsCut(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict == statement.Hold && strings.Contains(got.Reason, "limits.stageTimeout") {
		t.Fatalf("a limit appeared from nowhere: %+v", got)
	}
	if agents.builds != 1 {
		t.Fatalf("builds %d", agents.builds)
	}
}
