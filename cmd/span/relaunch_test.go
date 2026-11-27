package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/agent"
	"span/internal/decision"
	"span/internal/engine"
)

func TestRelaunchGivesEveryAgentStageTheChosenCLIAndSaysSoForOneStage(t *testing.T) {
	home := t.TempDir()
	for _, spec := range agent.Specs {
		path := filepath.Join(home, filepath.FromSlash(spec.Marker))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("Organic Driven Development\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	origHome, origLook, before := agent.Home, agent.Look, continueRun
	agent.Home = func() string { return home }
	agent.Look = func(string) error { return nil }
	continued := 0
	continueRun = func(context.Context, string, string, *decision.Answer) (engine.Result, error) {
		continued++
		return engine.Result{Stage: "build", Verdict: "advance"}, nil
	}
	t.Cleanup(func() { agent.Home, agent.Look, continueRun = origHome, origLook, before })

	root := opsRun(t, engine.Cursor{Stage: "build", Hold: true, AgentBlock: &engine.AgentBlock{Kind: "quota", Agent: "cursor-agent"}})
	set := func(a []string, o, e *bytes.Buffer) int { return stageAgentMain(a, o, e) }
	code, out, errs := call(set, "run-1", "frame", "claude", "--spans", root)
	if code != 0 || !strings.Contains(out, "build, inspect keep the default") {
		t.Fatalf("exit %d: %s %s", code, out, errs)
	}
	relaunch := func(a []string, o, e *bytes.Buffer) int { return relaunchMain(a, o, e) }
	if code, _, errs := call(relaunch, "run-1", "--agent", "claude", "--spans", root); code != 0 || continued != 1 {
		t.Fatalf("exit %d, continued %d: %s", code, continued, errs)
	}
	got := engine.ReadStageAgents(filepath.Join(root, "run-1"))
	for _, stage := range engine.AgentStages {
		if got[stage] != "claude" {
			t.Fatalf("agents = %v", got)
		}
	}
}
