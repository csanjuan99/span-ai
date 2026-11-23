// Package policy reads the project file that declares checks, environments,
// and which stages may proceed without a person.
package policy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"span/internal/agent"
	"span/internal/branch"
)

// Name is the file Span reads at the workspace root.
const Name = "span.yaml"

// File is span.yaml: project policy only. Empty modes mean auto. A run's identity (its
// feature document, pull request title, and commit scope) comes from its branch, not from
// this file.
type File struct {
	// Feature is the old project-wide feature name. It is still read, so an existing
	// span.yaml keeps loading, but nothing uses or validates it.
	Feature string `yaml:"feature"`
	Rules   string `yaml:"rules"`
	Harness string `yaml:"harness"`
	Agent   string `yaml:"agent"`
	// Models picks one model per CLI, by CLI id (cursor, claude, codex, pi). A CLI without an
	// entry runs on the model Gentle AI configured for it.
	Models map[string]string `yaml:"models"`
	// Base is the branch new runs start from and open their change into when the run does
	// not name one. Empty keeps the defaults: dev for feat and fix, master for hotfix.
	Base     string              `yaml:"base"`
	Stages   Modes               `yaml:"stages"`
	Prove    Bands               `yaml:"prove"`
	Promote  Ladder              `yaml:"promote"`
	Jira     Jira                `yaml:"jira"`
	QA       QA                  `yaml:"qa"`
	Repair   Repair              `yaml:"repair"`
	Run      Run                 `yaml:"run"`
	Manifest map[string]Manifest `yaml:"manifest"`
	Commit   Commit              `yaml:"commit"`
}

// Commit tunes what a stage's commit may take. Span never commits build output, and it
// stops on a file that looks like a secret; Allow lists the paths a person checked.
type Commit struct {
	// Allow is exact paths, or folders written with a trailing slash, relative to the
	// repository. A path that leaves the repository is refused when the policy loads.
	Allow []string `yaml:"allow"`
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
	return Parse(raw)
}

// Parse reads and validates the body of a policy file.
func Parse(raw []byte) (File, error) {
	var f File
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return File{}, err
	}
	if err := f.Validate(); err != nil {
		return File{}, err
	}
	return f, nil
}

// ProjectBase is the base: line of the span.yaml in dir. A missing file or line is "", which
// means the defaults apply.
func ProjectBase(dir string) (string, error) {
	path := filepath.Join(dir, Name)
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	f, err := Load(path)
	if err != nil {
		return "", err
	}
	return f.Base, nil
}

// Validate checks the agent, modes, base, weight, repairs, and environments. The feature
// line is not checked: it is ignored.
func (f File) Validate() error {
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
	for id, model := range f.Models {
		if !agent.IsID(id) {
			return fmt.Errorf("models: %q is not one of %s", id, agent.IDs())
		}
		if !agent.ValidModel(model) {
			return fmt.Errorf("models.%s: %q is not a model name", id, model)
		}
	}
	for _, entry := range f.Commit.Allow {
		clean := filepath.ToSlash(filepath.Clean(entry))
		if strings.TrimSpace(entry) == "" || filepath.IsAbs(entry) || strings.HasPrefix(entry, "/") || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
			return fmt.Errorf("commit.allow: %q is not a path inside the repository", entry)
		}
	}
	if f.Base != "" {
		if err := branch.CheckBase(f.Base); err != nil {
			return err
		}
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

// ModelFor is the model chosen for a CLI id, or "" when the CLI keeps its own.
func (f File) ModelFor(id string) string { return f.Models[id] }

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
// stays where it was, and the result is validated before it reaches the disk. A missing
// file is created holding only that line: span.yaml is optional, and choosing an agent is
// an explicit request to record one.
func SetAgent(path, value string) error {
	if !agent.Known(value) {
		return fmt.Errorf("agent %q is not one of auto, any, or %s", value, agent.IDs())
	}
	// A new file is 0644 like the one quickstart writes; a rewrite keeps the file's mode.
	mode := os.FileMode(0o644)
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		raw, err = nil, nil
	} else if err == nil {
		info, statErr := os.Stat(path)
		if statErr != nil {
			return statErr
		}
		mode = info.Mode().Perm()
	}
	if err != nil {
		return err
	}
	return writeChecked(path, replaceAgent(string(raw), value), mode)
}

// writeChecked validates the new body and replaces the file with it through a temp file, so
// a body that would not load never reaches the disk.
func writeChecked(path, next string, mode os.FileMode) error {
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
	if err := os.Chmod(name, mode); err != nil {
		return err
	}
	return os.Rename(name, path)
}

// SetModel records the model for one CLI under models:, or removes it when model is empty.
// Every other line stays where it was. A missing file is created holding only that entry.
func SetModel(path, id, model string) error {
	if !agent.IsID(id) {
		return fmt.Errorf("%q is not one of %s", id, agent.IDs())
	}
	if model != "" && !agent.ValidModel(model) {
		return fmt.Errorf("%q is not a model name", model)
	}
	mode := os.FileMode(0o644)
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		raw, err = nil, nil
	} else if err == nil {
		info, statErr := os.Stat(path)
		if statErr != nil {
			return statErr
		}
		mode = info.Mode().Perm()
	}
	if err != nil {
		return err
	}
	next, err := replaceModel(string(raw), id, model)
	if err != nil {
		return err
	}
	return writeChecked(path, next, mode)
}

// replaceModel edits the models: block of body. It refuses a models: written inline
// (models: {cursor: x}) rather than guess how to rewrite it.
func replaceModel(body, id, model string) (string, error) {
	lines := strings.Split(body, "\n")
	start := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "models:") {
			rest := strings.TrimSpace(strings.TrimPrefix(line, "models:"))
			if rest != "" && !strings.HasPrefix(rest, "#") {
				return "", fmt.Errorf("models: is written inline; rewrite it as a block to change it from here")
			}
			start = i
			break
		}
	}
	entry := "  " + id + ": " + model
	if start < 0 {
		if model == "" {
			return body, nil
		}
		out := strings.TrimRight(body, "\n")
		if out != "" {
			out += "\n"
		}
		return out + "models:\n" + entry + "\n", nil
	}
	end := start + 1
	last := start
	found := -1
	for ; end < len(lines); end++ {
		line := lines[end]
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && !strings.HasPrefix(trimmed, "#") {
			break
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		last = end
		if strings.HasPrefix(trimmed, id+":") {
			found = end
		}
	}
	switch {
	case found >= 0 && model == "":
		lines = append(lines[:found], lines[found+1:]...)
		last--
	case found >= 0:
		indent := lines[found][:len(lines[found])-len(strings.TrimLeft(lines[found], " \t"))]
		lines[found] = indent + id + ": " + model
	case model != "":
		lines = append(lines[:last+1], append([]string{entry}, lines[last+1:]...)...)
	}
	// An entry removed from a block that had only that entry leaves models: with no children
	// (last is still the header), and an empty header is dropped too.
	if model == "" && found >= 0 && last == start {
		lines = append(lines[:start], lines[start+1:]...)
	}
	return strings.Join(lines, "\n"), nil
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
