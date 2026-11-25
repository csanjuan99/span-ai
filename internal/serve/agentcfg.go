package serve

import (
	"context"
	"fmt"

	"span/internal/agent"
	"span/internal/start"
)

// agentPick is the agent section of the page.
type agentPick = start.AgentChoice

// agentView is what GET /agent returns for the project.
func agentView(project string) agentPick { return start.ResolveAgent(project) }

// modelView is the answer to GET /agent/models: what the desk can offer for one CLI.
type modelView struct {
	agent.ModelList
	// Error says why the CLI could not list its models; the person can still type a name.
	Error string `json:"error,omitempty"`
}

// modelsOf lists the models of the CLI with this id.
func modelsOf(ctx context.Context, id string) (modelView, error) {
	if !agent.IsID(id) {
		return modelView{}, fmt.Errorf("%q is not one of %s", id, agent.IDs())
	}
	list, err := agent.Models(ctx, id)
	view := modelView{ModelList: list}
	if err != nil {
		view.Error = err.Error()
	}
	if view.Models == nil {
		view.Models = []agent.Model{}
	}
	return view, nil
}

// canChooseModel refuses a CLI that cannot run yet. Models are managed through Gentle AI, so
// a CLI without it is not configured, and a model chosen for it would be a setting nobody
// can use.
func canChooseModel(id string) error {
	spec, ok := agent.Find(id)
	if !ok {
		return fmt.Errorf("%q is not one of %s", id, agent.IDs())
	}
	if ready, reason := spec.Ready(agent.Home()); !ready {
		return fmt.Errorf("%s: %s. Gentle AI and its configuration are required: Span uses the models it sets up for each CLI", spec.Label, reason)
	}
	return nil
}
