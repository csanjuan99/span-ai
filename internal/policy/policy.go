// Package policy reads the project file that declares checks, environments,
// and which stages may proceed without a person.
package policy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"span/internal/agent"
)

// Name is the file Span reads at the workspace root.
const Name = "span.yaml"

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
	Repair   Repair              `yaml:"repair"`
	Run      Run                 `yaml:"run"`
	Manifest map[string]Manifest `yaml:"manifest"`
}

// Jira names the status used when a story cannot be implemented yet.
type Jira struct {
	BlockStatus string `yaml:"blockStatus"`
}

// QA is the acceptance command. Browser asks for Playwright when no command is set.
// Report is the per-test report the gate reads; empty looks in the default places.
type QA struct {
	Command string `yaml:"command"`
	Browser bool   `yaml:"browser"`
	Report  string `yaml:"report"`
}

// Run is how heavy this story is. An empty weight is derived from the story itself.
type Run struct {
	Weight string `yaml:"weight"`
}

// Repair is how many times a failing check may send the span back to build.
// It is a ceiling, not a target: the same failure twice stops first.
type Repair struct {
	MaxAttempts int `yaml:"maxAttempts"`
}

// The repair ceiling. One is the honest fix, the second is a correction, and a
// third that still fails usually means the failure was not understood.
const (
	DefaultRepairAttempts = 3
	MaxRepairAttempts     = 8
)

// Modes selects auto or hold for the external stages.
type Modes struct {
	Open    string `yaml:"open"`
	Merge   string `yaml:"merge"`
	Promote string `yaml:"promote"`
	// Conflict decides what happens when the base cannot be merged cleanly into the branch.
	// hold (the default) waits for a person to choose; auto applies the first option only
	// when every group of the conflict is low risk.
	Conflict string `yaml:"conflict"`
}

// Bands are the declared checks. They are argv words, not a shell line.
// A focused band is run against the files that declare this story's criteria; {files}
// is replaced with them. When it is set, it is the verdict and Fast becomes a report.
type Bands struct {
	Focused    []string `yaml:"focused"`
	Fast       []string `yaml:"fast"`
	Acceptance []string `yaml:"acceptance"`
	// Prepare runs once in a run's own git worktree before its baseline and build, to put
	// the dependencies a fresh tree lacks (for example pnpm install --frozen-lockfile). It
	// never runs in the person's own checkout.
	Prepare []string `yaml:"prepare"`
	// Baseline runs the wide commands once on the base commit before the first build, so a
	// failure that was already there reports instead of spending a repair. Absent means on.
	Baseline *bool `yaml:"baseline"`
	// Diff derives a focused band from the branch's diff when neither the criteria nor
	// Focused name one, so a repair runs only the tests the change can affect. Absent means on.
	Diff *bool `yaml:"diff"`
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
	if !agent.Known(f.Agent) {
		return fmt.Errorf("agent %q is not one of auto, any, or %s", f.Agent, agent.IDs())
	}
	for _, mode := range []string{f.OpenMode(), f.MergeMode()} {
		if mode != "auto" && mode != "hold" {
			return fmt.Errorf("stage mode %q is not auto or hold", mode)
		}
	}
	if mode := f.ConflictMode(); mode != "auto" && mode != "hold" {
		return fmt.Errorf("stages.conflict %q is not auto or hold", mode)
	}
	if !ValidWeight(f.Run.Weight) {
		return WeightError(f.Run.Weight)
	}
	if attempts := f.Repair.MaxAttempts; attempts < 0 || attempts > MaxRepairAttempts {
		return fmt.Errorf("repair.maxAttempts is %d, allowed 1 to %d", attempts, MaxRepairAttempts)
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

// ConflictMode returns how a merge conflict is resolved. Unlike the other stages it
// defaults to hold: nothing is written to the branch until a person chooses.
func (f File) ConflictMode() string {
	if mode := strings.TrimSpace(f.Stages.Conflict); mode != "" {
		return mode
	}
	return "hold"
}

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
	if len(f.Prove.Focused) > 0 {
		return strings.TrimSpace(f.Prove.Focused[0])
	}
	return ""
}

// SetAgent rewrites the one agent: line. Every other line, comment, and blank line
// stays where it was, and the result is validated before it reaches the disk.
func SetAgent(path, value string) error {
	if !agent.Known(value) {
		return fmt.Errorf("agent %q is not one of auto, any, or %s", value, agent.IDs())
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	next := replaceAgent(string(raw), value)
	var check File
	if err := yaml.Unmarshal([]byte(next), &check); err != nil {
		return err
	}
	if err := check.Validate(); err != nil {
		return err
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".span-yaml-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer func() { _ = os.Remove(name) }()
	if _, err := tmp.WriteString(next); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

func replaceAgent(body, value string) string {
	lines := strings.Split(body, "\n")
	line := "agent: " + value
	for i, item := range lines {
		if strings.HasPrefix(item, "agent:") {
			lines[i] = line
			return strings.Join(lines, "\n")
		}
	}
	out := make([]string, 0, len(lines)+1)
	inserted := false
	for _, item := range lines {
		out = append(out, item)
		if !inserted && strings.HasPrefix(item, "feature:") {
			out = append(out, line)
			inserted = true
		}
	}
	if !inserted {
		out = append([]string{line}, out...)
	}
	return strings.Join(out, "\n")
}

// BaselineOn reports whether failures already present on the base commit are recorded
// and excused. It is on unless prove.baseline is false.
func (f File) BaselineOn() bool { return f.Prove.Baseline == nil || *f.Prove.Baseline }

// DiffScopeOn reports whether prove may narrow its focused band to what the branch's diff
// can affect. On unless the policy says diff: false.
func (f File) DiffScopeOn() bool { return f.Prove.Diff == nil || *f.Prove.Diff }

// RepairCap is how many repairs a failing check may start.
func (f File) RepairCap() int {
	if f.Repair.MaxAttempts > 0 {
		return f.Repair.MaxAttempts
	}
	return DefaultRepairAttempts
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
