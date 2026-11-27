package engine

import (
	"context"
	"path/filepath"
	"testing"

	"span/internal/agent"
	"span/internal/journal"
)

func TestARunAgentIsWhatEveryAgentRunReports(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	id, err := Open(opt)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(opt.Root, id)
	if err := SetRunAgent(dir, "pi", piHome(t)); err != nil {
		t.Fatal(err)
	}
	opt.Agents = echoAgents{agents}
	if _, err := Resume(context.Background(), opt, id); err != nil {
		t.Fatal(err)
	}
	events, err := journal.Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, event := range events {
		if event.Type == "agent.run" {
			seen[event.Detail["stage"]] = event.Detail["agent"]
		}
	}
	for _, stage := range AgentStages {
		if seen[stage] != "pi" {
			t.Fatalf("agent.run = %v", seen)
		}
	}
}

// echoAgents selects exactly the CLI asked for.
type echoAgents struct{ *fakeAgents }

func (e echoAgents) Select(want string) (agent.Selection, error) {
	return agent.Selection{Build: want, Inspect: want}, nil
}
