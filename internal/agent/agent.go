// Package agent selects an installed CLI that already has Gentle AI.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrNone means no selected CLI is installed with Gentle AI.
var ErrNone = errors.New("no agent with Gentle AI")

// Names is the lookup order for agent: any.
var Names = []string{"cursor-agent", "claude", "codex"}

// Selection is the build CLI and, when another is enabled, a different inspect CLI.
type Selection struct {
	Build   string
	Inspect string
}

// Look resolves a binary name. Tests replace it.
var Look = func(name string) error {
	_, err := exec.LookPath(name)
	return err
}

// Select picks the requested agent, or the first enabled one.
func Select(want, home string) (Selection, error) {
	enabled := enabled(home)
	if len(enabled) == 0 {
		return Selection{}, ErrNone
	}
	if want == "" || want == "any" {
		sel := Selection{Build: enabled[0], Inspect: enabled[0]}
		if len(enabled) > 1 {
			sel.Inspect = enabled[1]
		}
		return sel, nil
	}
	for _, name := range enabled {
		if name == want {
			sel := Selection{Build: name, Inspect: name}
			for _, other := range enabled {
				if other != name {
					sel.Inspect = other
					break
				}
			}
			return sel, nil
		}
	}
	return Selection{}, fmt.Errorf("%w: %s", ErrNone, want)
}

func enabled(home string) []string {
	var out []string
	for _, name := range Names {
		if Look(name) != nil {
			continue
		}
		if !Gentle(name, home) {
			continue
		}
		out = append(out, name)
	}
	return out
}

// Gentle reports whether that harness has Gentle AI installed for this user.
func Gentle(name, home string) bool {
	switch name {
	case "cursor-agent":
		_, err := os.Stat(filepath.Join(home, ".cursor", "rules", "gentle-ai.mdc"))
		return err == nil
	case "claude":
		return contains(filepath.Join(home, ".claude", "CLAUDE.md"), "Organic Driven Development")
	case "codex":
		return contains(filepath.Join(home, ".codex", "AGENTS.md"), "Organic Driven Development")
	default:
		return false
	}
}

func contains(path, needle string) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(raw), needle)
}

// commandArgs starts a headless agent that can write. Nobody is present to approve a prompt.
func commandArgs(name string) []string {
	switch name {
	case "cursor-agent":
		return []string{"-p", "--trust", "--force"}
	case "claude":
		return []string{"-p", "--dangerously-skip-permissions"}
	case "codex":
		return []string{"exec", "--sandbox", "workspace-write", "--approve-for-me"}
	default:
		return []string{"-p"}
	}
}

// Run executes the CLI with the prompt on stdin and returns stdout.
func Run(ctx context.Context, name, workspace, prompt string) (string, error) {
	return RunWriter(ctx, name, workspace, prompt, nil)
}

// RunWriter does what Run does and copies stdout to sink while the process runs.
func RunWriter(ctx context.Context, name, workspace, prompt string, sink io.Writer) (string, error) {
	cmd := exec.CommandContext(ctx, name, commandArgs(name)...)
	cmd.Dir = workspace
	cmd.Stdin = strings.NewReader(prompt)
	var stdout, stderr strings.Builder
	var out io.Writer = &stdout
	if sink != nil {
		out = io.MultiWriter(&stdout, sink)
	}
	cmd.Stdout = out
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("%s: %w: %s", name, err, stderr.String())
	}
	pid := 0
	if cmd.Process != nil {
		pid = cmd.Process.Pid
	}
	stop := observe(ctx, name, pid)
	defer stop()
	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("%s: %w: %s", name, err, stderr.String())
	}
	return stdout.String(), nil
}

// ParseJSON extracts the first JSON object from an agent reply.
func ParseJSON(raw string, dest any) error {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end < start {
		return errors.New("agent reply has no JSON object")
	}
	if err := json.Unmarshal([]byte(raw[start:end+1]), dest); err != nil {
		return fmt.Errorf("agent JSON: %w", err)
	}
	return nil
}
