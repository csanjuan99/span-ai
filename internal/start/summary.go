package start

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"span/internal/agent"
	"span/internal/branch"
	"span/internal/policy"
)

// Summary is what the desk shows under “What will happen”.
type Summary struct {
	Folder      string            `json:"folder"`
	Base        string            `json:"base"`
	Agent       string            `json:"agent"`
	Agents      map[string]string `json:"agents"`
	Stages      map[string]string `json:"stages"`
	Checks      []string          `json:"checks"`
	Repair      int               `json:"repair"`
	DoneStatus  string            `json:"doneStatus"`
	ChosenAgent string            `json:"chosenAgent"`
	AgentReason string            `json:"agentReason,omitempty"`
}

// WhatWillHappen reads effective saved settings for one folder.
func WhatWillHappen(project, kind, base string) Summary {
	project = strings.TrimSpace(project)
	if project == "" {
		project = "."
	}
	out := Summary{
		Folder: project,
		Agent:  "auto",
		Agents: emptyAgentStages(),
		Stages: map[string]string{
			"open": "auto", "merge": "auto", "promote": "auto",
			"plan": "auto", "conflict": "hold",
		},
		Checks:      []string{},
		Repair:      policy.DefaultRepairAttempts,
		ChosenAgent: firstReady(agent.Home()),
	}
	file, err := policy.Load(filepath.Join(project, policy.Name))
	if errors.Is(err, os.ErrNotExist) {
		out.AgentReason = "There is no span.yaml, so runs use the defaults."
	} else if err != nil {
		out.AgentReason = "span.yaml does not load: " + err.Error()
	} else {
		out.Base = file.Base
		out.Agent = orEmpty(file.Agent, "auto")
		out.Repair = file.Repair.MaxAttempts
		if out.Repair == 0 {
			out.Repair = policy.DefaultRepairAttempts
		}
		out.DoneStatus = file.DoneStatus()
		out.Checks = append([]string{}, file.Prove.Fast...)
		out.Stages = map[string]string{
			"open":     orEmpty(file.Stages.Open, "auto"),
			"merge":    orEmpty(file.Stages.Merge, "auto"),
			"promote":  orEmpty(file.Stages.Promote, "auto"),
			"plan":     orEmpty(file.Stages.Plan, "auto"),
			"conflict": orEmpty(file.Stages.Conflict, "hold"),
		}
		for _, stage := range policy.AgentStageNames {
			out.Agents[stage] = strings.TrimSpace(file.Agents[stage])
		}
		out.ChosenAgent, out.AgentReason = chosenAgent(file.Agent, agent.Home())
	}
	chosenBase := strings.TrimSpace(base)
	if chosenBase != "" {
		out.Base = chosenBase
	} else if out.Base == "" {
		if strings.TrimSpace(kind) == "hotfix" {
			out.Base = branch.Master
		} else {
			out.Base = branch.Dev
		}
	}
	return out
}

func emptyAgentStages() map[string]string {
	out := make(map[string]string, len(policy.AgentStageNames))
	for _, stage := range policy.AgentStageNames {
		out[stage] = ""
	}
	return out
}

func orEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func chosenAgent(selected, home string) (string, string) {
	if agent.Auto(selected) {
		return firstReady(home), ""
	}
	if spec, ok := agent.Find(selected); ok {
		return spec.ID, agent.Explain(selected, home)
	}
	return "", agent.Explain(selected, home)
}
