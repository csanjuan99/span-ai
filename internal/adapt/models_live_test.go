package adapt

import "testing"

// The desk changes a model while a run is going. The next stage must use it.
func TestTheModelIsReadAgainEachTimeItIsAsked(t *testing.T) {
	current := map[string]string{"claude": "sonnet"}
	agents := CLIAgents{Models: map[string]string{"claude": "stale"}, Fresh: func() map[string]string { return current }}
	if got := agents.Model("claude"); got != "sonnet" {
		t.Fatalf("%q", got)
	}
	current = map[string]string{"claude": "opus"}
	if got := agents.Model("claude"); got != "opus" {
		t.Fatalf("a model chosen after the run began is ignored: %q", got)
	}
}

func TestWithoutAFreshSourceTheModelsGivenAtStartStand(t *testing.T) {
	agents := CLIAgents{Models: map[string]string{"claude": "sonnet"}}
	if got := agents.Model("claude"); got != "sonnet" {
		t.Fatalf("%q", got)
	}
	if got := agents.Model("not-a-cli"); got != "" {
		t.Fatalf("%q", got)
	}
}
