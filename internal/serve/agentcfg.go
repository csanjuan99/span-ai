package serve

import (
	"context"
	"errors"
	"fmt"
	"os"
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
	// Models is the model chosen for each CLI id. A CLI without an entry keeps the model
	// Gentle AI configured for it.
	Models map[string]string `json:"models"`
}

// agentView reads the project policy and resolves what would run. Reading the
// answer is not the same as the request: a project can ask for a CLI that is not
// ready, and the page has to be able to say so.
func agentView(project string) agentPick {
	home := agent.Home()
	view := agentPick{Agents: agent.Report(home), Selected: "any", Chosen: firstReady(home), Models: map[string]string{}}
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
