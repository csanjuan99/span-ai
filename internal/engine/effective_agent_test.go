package engine

import (
	"context"
	"path/filepath"
	"testing"

	"span/internal/journal"
	"span/internal/policy"
)

// modelAgents is a fake that also says which model each CLI would run.
type modelAgents struct{ *fakeAgents }

// Model is the model the fake says a CLI runs.
func (modelAgents) Model(binary string) string { return "model-of-" + binary }

// The desk shows what really ran, not what was configured, so every agent stage leaves the
// CLI and model it used in the run's journal.
func TestAnAgentStageRecordsTheCLIAndModelThatRan(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	opt.Agents = modelAgents{agents}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	events, err := journal.Read(filepath.Join(opt.Root, got.ID))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, event := range events {
		if event.Type == "agent.run" {
			seen[event.Detail["stage"]] = event.Detail["agent"] + "/" + event.Detail["model"]
		}
	}
	if seen["frame"] != "cursor-agent/model-of-cursor-agent" || seen["build"] != "cursor-agent/model-of-cursor-agent" {
		t.Fatalf("frame and build must record who ran and with which model: %v", seen)
	}
	if bad, err := journal.Verify(filepath.Join(opt.Root, got.ID)); err != nil || bad != 0 {
		t.Fatalf("the journal chain broke: bad=%d err=%v", bad, err)
	}
}

// A CLI that does not report a model leaves the field empty instead of inventing one.
func TestAnAgentThatReportsNoModelRecordsNone(t *testing.T) {
	opt, _, _, _, _, _ := baseOpt(t)
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	events, err := journal.Read(filepath.Join(opt.Root, got.ID))
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.Type == "agent.run" && event.Detail["model"] != "" {
			t.Fatalf("a model nobody reported was recorded: %+v", event)
		}
	}
}

// span.yaml is written by the desk while a run is going. A stage that starts after the
// change must ask for the new agent, not the one the process read when it began.
func TestTheAgentIsReadAgainFromThePolicyBeforeEachStage(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	opt.Policy.Agent = "claude"
	opt.Fresh = func() policy.File {
		file := opt.Policy
		file.Agent = "codex"
		return file
	}
	if _, err := Start(context.Background(), opt); err != nil {
		t.Fatal(err)
	}
	if len(agents.wanted) == 0 {
		t.Fatal("no agent was selected")
	}
	for _, want := range agents.wanted {
		if want != "codex" {
			t.Fatalf("asked for %q although span.yaml now says codex: %v", want, agents.wanted)
		}
	}
}
