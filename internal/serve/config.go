package serve

import (
	"errors"
	"os"
	"path/filepath"

	"span/internal/policy"
)

// configPick is span.yaml as the settings page reads it: the values a person can see, with the
// defaults filled in when the file or a key is missing. Reading never writes anything.
type configPick struct {
	Folder       string       `json:"folder"`
	Exists       bool         `json:"exists"`
	Reason       string       `json:"reason,omitempty"`
	Base         string       `json:"base"`
	Stages       configStages `json:"stages"`
	Prove        configProve  `json:"prove"`
	QA           string       `json:"qa"`
	Repair       int          `json:"repair"`
	BlockStatus  string       `json:"blockStatus"`
	StageTimeout string       `json:"stageTimeout"`
}

type configStages struct {
	Open     string `json:"open"`
	Merge    string `json:"merge"`
	Promote  string `json:"promote"`
	Plan     string `json:"plan"`
	Conflict string `json:"conflict"`
}

type configProve struct {
	Focused    []string `json:"focused"`
	Fast       []string `json:"fast"`
	Acceptance []string `json:"acceptance"`
}

// orElse is the value, or the default when the value is empty.
func orElse(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func nonNil(items []string) []string {
	if items == nil {
		return []string{}
	}
	return items
}

// configView reads the span.yaml of a project. A missing or unreadable file is not an error:
// the page shows the defaults and says why.
func configView(project string) configPick {
	view := configPick{
		Folder: project,
		Stages: configStages{Open: "auto", Merge: "auto", Promote: "auto", Plan: "auto", Conflict: "hold"},
		Prove:  configProve{Focused: []string{}, Fast: []string{}, Acceptance: []string{}},
		Repair: policy.DefaultRepairAttempts,
	}
	file, err := policy.Load(filepath.Join(project, policy.Name))
	if errors.Is(err, os.ErrNotExist) {
		view.Reason = "There is no span.yaml, so runs use the defaults."
		return view
	}
	if err != nil {
		view.Reason = "span.yaml does not load: " + err.Error()
		return view
	}
	view.Exists = true
	view.Base = file.Base
	view.Stages = configStages{
		Open:     orElse(file.Stages.Open, "auto"),
		Merge:    orElse(file.Stages.Merge, "auto"),
		Promote:  orElse(file.Stages.Promote, "auto"),
		Plan:     orElse(file.Stages.Plan, "auto"),
		Conflict: orElse(file.Stages.Conflict, "hold"),
	}
	view.Prove = configProve{Focused: nonNil(file.Prove.Focused), Fast: nonNil(file.Prove.Fast), Acceptance: nonNil(file.Prove.Acceptance)}
	view.QA = file.QA.Command
	if file.Repair.MaxAttempts > 0 {
		view.Repair = file.Repair.MaxAttempts
	}
	view.BlockStatus = file.Jira.BlockStatus
	view.StageTimeout = file.Limits.StageTimeout
	return view
}
