package serve

import (
	"path/filepath"

	"span/internal/agent"
	"span/internal/policy"
)

// agentPick is the agent section of the page: every CLI Span knows, which one the
// project asked for, and which one would actually run today.
type agentPick struct {
	Agents   []agent.State `json:"agents"`
	Selected string        `json:"selected"`
	Chosen   string        `json:"chosen"`
	Exists   bool          `json:"exists"`
	Reason   string        `json:"reason,omitempty"`
}

// agentView reads the project policy and resolves what would run. Reading the
// answer is not the same as the request: a project can ask for a CLI that is not
// ready, and the page has to be able to say so.
func agentView(project string) agentPick {
	home := agent.Home()
	view := agentPick{Agents: agent.Report(home), Selected: "any", Chosen: firstReady(home)}
	file, err := policy.Load(filepath.Join(project, policy.Name))
	if err != nil {
		view.Reason = "span.yaml is not there yet. Span writes it on the first run."
		return view
	}
	view.Exists = true
	view.Selected = file.Agent
	if agent.Auto(file.Agent) {
		return view
	}
	if spec, ok := agent.Find(file.Agent); ok {
		view.Chosen = spec.ID
	}
	view.Reason = agent.Explain(file.Agent, home)
	return view
}

func firstReady(home string) string {
	if ready := agent.Available(home); len(ready) > 0 {
		return ready[0].ID
	}
	return ""
}
