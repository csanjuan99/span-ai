// Package policy reads the project file that declares checks, environments,
// and which stages may proceed without a person.
package policy

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// File is span.yaml. Empty modes mean auto.
type File struct {
	Feature  string              `yaml:"feature"`
	Rules    string              `yaml:"rules"`
	Harness  string              `yaml:"harness"`
	Agent    string              `yaml:"agent"`
	Stages   Modes               `yaml:"stages"`
	Prove    Bands               `yaml:"prove"`
	Promote  Ladder              `yaml:"promote"`
	Jira     Jira                `yaml:"jira"`
	QA       QA                  `yaml:"qa"`
	Manifest map[string]Manifest `yaml:"manifest"`
}

// Jira names the status used when a story cannot be implemented yet.
type Jira struct {
	BlockStatus string `yaml:"blockStatus"`
}

// QA is the acceptance command. Browser asks for Playwright when no command is set.
type QA struct {
	Command string `yaml:"command"`
	Browser bool   `yaml:"browser"`
}

// Modes selects auto or hold for the external stages.
type Modes struct {
	Open    string `yaml:"open"`
	Merge   string `yaml:"merge"`
	Promote string `yaml:"promote"`
}

// Bands are the declared checks. They are argv words, not a shell line.
type Bands struct {
	Fast       []string `yaml:"fast"`
	Acceptance []string `yaml:"acceptance"`
}

// Ladder is the promotion order.
type Ladder struct {
	Environments []Environment `yaml:"environments"`
}

// Environment is one deploy target.
type Environment struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command"`
	Mode    string `yaml:"mode"`
}

// Manifest names the skills, rules, and MCP servers a stage may use.
// Bodies are not stored here.
type Manifest struct {
	Skills []string `yaml:"skills"`
	Rules  []string `yaml:"rules"`
	MCP    []string `yaml:"mcp"`
}

// Load reads and validates a policy file.
func Load(path string) (File, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return File{}, err
	}
	var f File
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return File{}, err
	}
	if err := f.Validate(); err != nil {
		return File{}, err
	}
	return f, nil
}

// Validate checks modes and the feature name.
func (f File) Validate() error {
	if !validFeature(f.Feature) {
		return fmt.Errorf("feature name must be letters, digits, dots, dashes, or underscores")
	}
	for _, mode := range []string{f.OpenMode(), f.MergeMode()} {
		if mode != "auto" && mode != "hold" {
			return fmt.Errorf("stage mode %q is not auto or hold", mode)
		}
	}
	for _, env := range f.Promote.Environments {
		mode := env.Mode
		if mode == "" {
			mode = "auto"
		}
		if mode != "auto" && mode != "hold" {
			return fmt.Errorf("environment %s mode %q is not auto or hold", env.Name, env.Mode)
		}
		if env.Name == "" || env.Command == "" {
			return fmt.Errorf("environment needs a name and a command")
		}
	}
	return nil
}

// OpenMode returns the open stage mode.
func (f File) OpenMode() string { return orAuto(f.Stages.Open) }

// MergeMode returns the merge stage mode.
func (f File) MergeMode() string { return orAuto(f.Stages.Merge) }

// BlockStatus is the Jira status for a story that is missing information.
func (f File) BlockStatus() string {
	if status := strings.TrimSpace(f.Jira.BlockStatus); status != "" {
		return status
	}
	return "Block"
}

// QACommand is the one acceptance command. An explicit command wins.
// Otherwise a browser repo uses Playwright, then the first acceptance command, then the first fast command.
func (f File) QACommand(browser bool) string {
	if command := strings.TrimSpace(f.QA.Command); command != "" {
		return command
	}
	if browser || f.QA.Browser {
		return "npx playwright test"
	}
	if len(f.Prove.Acceptance) > 0 {
		return strings.TrimSpace(f.Prove.Acceptance[0])
	}
	if len(f.Prove.Fast) > 0 {
		return strings.TrimSpace(f.Prove.Fast[0])
	}
	return ""
}

func orAuto(mode string) string {
	if mode == "" {
		return "auto"
	}
	return mode
}

func validFeature(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_' || r == '.':
		default:
			return false
		}
	}
	return true
}
