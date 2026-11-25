package agent

import (
	"os"
	"path/filepath"
	"strings"
)

// Spec is everything Span needs to know about one CLI: how to find it, how to call
// it headless, and how to tell it already has Gentle AI.
type Spec struct {
	ID     string
	Label  string
	Binary string
	// Args make the CLI run once, write, and exit. Nobody is present to approve.
	Args []string
	// Marker is the path, under the home directory, that the harness installs.
	Marker string
	// Needle is a phrase that must appear in Marker. Empty means presence is enough.
	Needle string
	// ModelFlag selects the model for one run. It is passed with the model as a separate
	// argument, never joined, so a value cannot become a flag of its own.
	ModelFlag string
	// Install is the command a person runs to get the CLI. It is shown beside a CLI that
	// is not on PATH; Span never runs it.
	Install string
}

// Specs is the registry and the order agent: any follows.
var Specs = []Spec{
	{
		ID: "cursor", Label: "Cursor", Binary: "cursor-agent",
		Args: []string{"-p", "--trust", "--force"}, ModelFlag: "--model",
		Install: "curl https://cursor.com/install -fsS | bash",
		Marker:  ".cursor/rules/gentle-ai.mdc",
	},
	{
		ID: "claude", Label: "Claude", Binary: "claude",
		Args: []string{"-p", "--dangerously-skip-permissions"}, ModelFlag: "--model",
		Install: "npm install -g @anthropic-ai/claude-code",
		Marker:  ".claude/CLAUDE.md", Needle: "Organic Driven Development",
	},
	{
		ID: "codex", Label: "Codex", Binary: "codex",
		Args: []string{"exec", "--approve-for-me"}, ModelFlag: "--model",
		Install: "npm install -g @openai/codex",
		Marker:  ".codex/AGENTS.md", Needle: "Organic Driven Development",
	},
	{
		// Pi runs once with --print and trusts project-local files with --approve.
		ID: "pi", Label: "Pi", Binary: "pi",
		Args: []string{"--print", "--approve"}, ModelFlag: "--model",
		Install: "npm install -g @mariozechner/pi-coding-agent",
		Marker:  ".pi/agent/npm/node_modules/gentle-pi",
	},
}

// MaxModel is the longest model name Span accepts.
const MaxModel = 100

// ValidModel reports whether value is safe to hand to a CLI as a model: a name made of
// letters, digits and . _ : / @ + - that does not start with a dash (so it cannot be read
// as a flag), with no spaces, quotes or line breaks. An empty value is not a model.
func ValidModel(value string) bool {
	if value == "" || len(value) > MaxModel {
		return false
	}
	for i, r := range value {
		alnum := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
		if i == 0 && !alnum {
			return false
		}
		if !alnum && !strings.ContainsRune("._:/@+-", r) {
			return false
		}
	}
	return true
}

// Home is where the harnesses keep their install for this user. Tests replace it.
var Home = func() string {
	dir, _ := os.UserHomeDir()
	return dir
}

// State is one row the page can render without knowing the registry.
type State struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Binary    string `json:"binary"`
	Installed bool   `json:"installed"`
	Gentle    bool   `json:"gentle"`
	Ready     bool   `json:"ready"`
	Reason    string `json:"reason,omitempty"`
	// Install is the command that installs the CLI. It is only set while the CLI is missing.
	Install string `json:"install,omitempty"`
}

// IsID reports whether value is exactly the id of a CLI Span knows.
func IsID(value string) bool {
	for _, spec := range Specs {
		if value == spec.ID {
			return true
		}
	}
	return false
}

// Find resolves an id or a binary name. The binary name is accepted so an existing
// span.yaml keeps working.
func Find(value string) (Spec, bool) {
	clean := strings.ToLower(strings.TrimSpace(value))
	for _, spec := range Specs {
		if clean == spec.ID || clean == spec.Binary {
			return spec, true
		}
	}
	return Spec{}, false
}

// Known reports whether a policy value names an agent.
func Known(value string) bool {
	if Auto(value) {
		return true
	}
	_, ok := Find(value)
	return ok
}

// Auto reports whether the value lets Span choose.
func Auto(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "auto", "any":
		return true
	}
	return false
}

// Installed reports whether the binary is on PATH.
func (s Spec) Installed() bool { return Look(s.Binary) == nil }

// Gentle reports whether that harness already has Gentle AI for this user.
func (s Spec) Gentle(home string) bool {
	if s.Marker == "" {
		return false
	}
	path := filepath.Join(home, filepath.FromSlash(s.Marker))
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if s.Needle == "" {
		return true
	}
	if info.IsDir() {
		return true
	}
	return contains(path, s.Needle)
}

// Ready is what a person needs before choosing it: the CLI is installed and the
// harness has Gentle AI. The reason explains the missing half.
func (s Spec) Ready(home string) (bool, string) {
	if !s.Installed() {
		return false, s.Binary + " is not on PATH"
	}
	if !s.Gentle(home) {
		return false, s.Label + " does not have Gentle AI installed"
	}
	return true, ""
}

// Report describes every agent for the page, in the order agent: any follows.
func Report(home string) []State {
	out := make([]State, 0, len(Specs))
	for _, spec := range Specs {
		installed := spec.Installed()
		gentle := spec.Gentle(home)
		ready, reason := spec.Ready(home)
		state := State{
			ID: spec.ID, Label: spec.Label, Binary: spec.Binary,
			Installed: installed, Gentle: gentle, Ready: ready, Reason: reason,
		}
		if !installed {
			state.Install = spec.Install
		}
		out = append(out, state)
	}
	return out
}

// Available returns the agents a person can choose today, in preference order.
func Available(home string) []Spec {
	var out []Spec
	for _, spec := range Specs {
		if ok, _ := spec.Ready(home); ok {
			out = append(out, spec)
		}
	}
	return out
}

// Explain says why a requested agent cannot run, in the words the page shows.
func Explain(want, home string) string {
	spec, ok := Find(want)
	if !ok {
		return "agent " + want + " is not one of " + IDs()
	}
	if _, reason := spec.Ready(home); reason != "" {
		return reason
	}
	return ""
}

// IDs lists the configurable ids for a message.
func IDs() string {
	names := make([]string, 0, len(Specs))
	for _, spec := range Specs {
		names = append(names, spec.ID)
	}
	return strings.Join(names, ", ")
}
