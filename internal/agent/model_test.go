package agent

import (
	"strings"
	"testing"
)

func TestAModelIsPassedAsItsOwnArgumentAfterTheFixedFlags(t *testing.T) {
	tests := []struct{ name, model, want string }{
		{"cursor-agent", "gpt-5.3-codex", "-p --trust --force --model gpt-5.3-codex"},
		{"claude", "opus", "-p --dangerously-skip-permissions --model opus"},
		{"codex", "gpt-5.3-codex", "exec --approve-for-me --model gpt-5.3-codex"},
		{"pi", "xai/grok-4.7", "--print --approve --model xai/grok-4.7"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := commandArgsModel(tt.name, tt.model)
			if strings.Join(got, " ") != tt.want {
				t.Fatalf("args %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNoModelLeavesTheFixedFlagsAlone(t *testing.T) {
	if got := strings.Join(commandArgsModel("claude", ""), " "); got != "-p --dangerously-skip-permissions" {
		t.Fatalf("an empty model changed the command: %q", got)
	}
}

func TestAModelThatCouldBeAFlagIsNeverPassed(t *testing.T) {
	for _, bad := range []string{"--approve", "-p", "a b", "x\ny", "a;b", "$(id)", strings.Repeat("a", 101), ""} {
		if ValidModel(bad) {
			t.Fatalf("%q was accepted as a model", bad)
		}
		if got := strings.Join(commandArgsModel("claude", bad), " "); strings.Contains(got, "--model") {
			t.Fatalf("%q reached the command: %q", bad, got)
		}
	}
	for _, good := range []string{"opus", "gpt-5.3-codex-high", "xai/grok-4.7", "deepseek-v4.1-flash:cloud", "claude-sonnet-5-5"} {
		if !ValidModel(good) {
			t.Fatalf("%q was refused", good)
		}
	}
}

func TestEveryAgentNamesItsModelFlag(t *testing.T) {
	for _, spec := range Specs {
		if spec.ModelFlag != "--model" {
			t.Fatalf("%s has model flag %q", spec.ID, spec.ModelFlag)
		}
	}
}
