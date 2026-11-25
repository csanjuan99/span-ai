package serve

import (
	"context"
	"net/url"
	"strings"

	"span/internal/policy"
)

// ConfigOf reads the settings of a project as GET /config answers them.
func ConfigOf(project string) any { return configView(project) }

// EditConfig previews (save false) or writes (save true) form fields into the span.yaml of a
// project as POST /config/preview and POST /config do, with the same validation and the same
// effects. The keys are the form fields of the settings page: base, open, merge, promote, plan,
// conflict, focused, fast, acceptance, qa, repair, doneStatus and agents.<stage>.
func EditConfig(project string, form url.Values, save bool) (any, error) {
	edit, err := editConfig(project, form, save)
	return edit, err
}

// AgentOf reads the agent registry and the project's choice as GET /agent answers it.
func AgentOf(project string) any { return agentView(project) }

// ChooseAgent saves the project's agent as POST /agent does: a CLI name, "auto" or "any".
func ChooseAgent(project, agent string) (any, error) {
	path, err := policyFile(project)
	if err != nil {
		return nil, err
	}
	if err := policy.SetAgent(path, agent); err != nil {
		return nil, err
	}
	return agentView(project), nil
}

// ChooseModel saves the model of one CLI as POST /agent/model does. An empty model clears it.
func ChooseModel(project, cli, model string) (any, error) {
	path, err := policyFile(project)
	if err != nil {
		return nil, err
	}
	model = strings.TrimSpace(model)
	// Clearing a model is always allowed: a CLI that stopped being ready must not keep a
	// saved model the person can no longer remove.
	if model != "" {
		if err := canChooseModel(cli); err != nil {
			return nil, err
		}
	}
	if err := policy.SetModel(path, cli, model); err != nil {
		return nil, err
	}
	return agentView(project), nil
}

// ModelsOf lists the models a CLI offers, as GET /agent/models answers it.
func ModelsOf(ctx context.Context, cli string) (any, error) { return modelsOf(ctx, cli) }

// ProbeFolderOf says what Span finds in a folder, as GET /onboarding/folder answers it.
func ProbeFolderOf(ctx context.Context, raw string) any { return probeFolder(ctx, raw) }
