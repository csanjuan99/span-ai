package start

import (
	"errors"
	"os"
	"path/filepath"

	"span/internal/agent"
	"span/internal/policy"
)

// AgentChoice is the agent section of the desk: every CLI Span knows, which one the
// project asked for, and which one would actually run today.
type AgentChoice struct {
	Agents   []agent.State `json:"agents"`
	Selected string        `json:"selected"`
	Chosen   string        `json:"chosen"`
	Exists   bool          `json:"exists"`
	Reason   string        `json:"reason,omitempty"`
	// Models is the model chosen for each CLI id. A CLI without an entry keeps the model
	// Gentle AI configured for it.
	Models map[string]string `json:"models"`
}

// ResolveAgent reads the project policy and resolves what would run. Reading the
// answer is not the same as the request: a project can ask for a CLI that is not
// ready, and the page has to be able to say so.
func ResolveAgent(project string) AgentChoice {
	home := agent.Home()
	view := AgentChoice{Agents: agent.Report(home), Selected: "any", Chosen: firstReady(home), Models: map[string]string{}}
	file, err := policy.Load(filepath.Join(project, policy.Name))
	if errors.Is(err, os.ErrNotExist) {
		view.Reason = "There is no span.yaml, so runs use the defaults. Choosing an agent creates one."
		return view
	}
	if err != nil {
		view.Reason = "span.yaml does not load: " + err.Error()
		return view
	}
	view.Exists = true
	for id, model := range file.Models {
		view.Models[id] = model
	}
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
