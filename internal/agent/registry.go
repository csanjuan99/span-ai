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
}

// Specs is the registry and the order agent: any follows.
var Specs = []Spec{
	{
		ID: "cursor", Label: "Cursor", Binary: "cursor-agent",
		Args:   []string{"-p", "--trust", "--force"},
		Marker: ".cursor/rules/gentle-ai.mdc",
	},
	{
		ID: "claude", Label: "Claude", Binary: "claude",
		Args:   []string{"-p", "--dangerously-skip-permissions"},
		Marker: ".claude/CLAUDE.md", Needle: "Organic Driven Development",
	},
	{
		ID: "codex", Label: "Codex", Binary: "codex",
		Args:   []string{"exec", "--approve-for-me"},
		Marker: ".codex/AGENTS.md", Needle: "Organic Driven Development",
	},
	{
		// Pi runs once with --print and trusts project-local files with --approve.
		ID: "pi", Label: "Pi", Binary: "pi",
		Args:   []string{"--print", "--approve"},
		Marker: ".pi/agent/npm/node_modules/gentle-pi",
	},
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
		out = append(out, State{
			ID: spec.ID, Label: spec.Label, Binary: spec.Binary,
			Installed: installed, Gentle: gentle, Ready: ready, Reason: reason,
		})
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
