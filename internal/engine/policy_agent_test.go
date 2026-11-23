package engine

import (
	"context"
	"testing"

	"span/internal/policy"
)

// span.yaml names the CLI a run uses. The page writes that line when a person picks one, so
// the engine has to honour it, not only the per-stage choices.
func TestTheAgentNamedInThePolicyIsTheOneAsked(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	opt.Policy.Agent = "claude"
	if _, err := Start(context.Background(), opt); err != nil {
		t.Fatal(err)
	}
	if len(agents.wanted) == 0 {
		t.Fatal("no agent was selected")
	}
	for _, want := range agents.wanted {
		if want != "claude" {
			t.Fatalf("asked for %q, the policy says claude: %v", want, agents.wanted)
		}
	}
}

func TestAStageChoiceStillBeatsThePolicy(t *testing.T) {
	r := &runner{opt: Options{Policy: policy.File{Agent: "claude"}}, cur: Cursor{}}
	if got := r.agentWanted("codex"); got != "codex" {
		t.Fatalf("%q", got)
	}
	if got := r.agentWanted(""); got != "claude" {
		t.Fatalf("%q", got)
	}
}
