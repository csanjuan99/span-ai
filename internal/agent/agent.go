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
	"strings"

	"span/internal/proc"
)

// ErrNone means no selected CLI is installed with Gentle AI.
var ErrNone = errors.New("no agent with Gentle AI")

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

// Select picks the requested agent, or the first ready one. Ready means the
// binary is on PATH and that harness already has Gentle AI.
func Select(want, home string) (Selection, error) {
	ready := Available(home)
	if len(ready) == 0 {
		return Selection{}, ErrNone
	}
	if Auto(want) {
		sel := Selection{Build: ready[0].Binary, Inspect: ready[0].Binary}
		if len(ready) > 1 {
			sel.Inspect = ready[1].Binary
		}
		return sel, nil
	}
	spec, ok := Find(want)
	if !ok {
		return Selection{}, fmt.Errorf("%w: %s is not one of %s", ErrNone, want, IDs())
	}
	if reason := Explain(want, home); reason != "" {
		return Selection{}, fmt.Errorf("%w: %s", ErrNone, reason)
	}
	sel := Selection{Build: spec.Binary, Inspect: spec.Binary}
	for _, other := range ready {
		if other.Binary != spec.Binary {
			sel.Inspect = other.Binary
			break
		}
	}
	return sel, nil
}

func contains(path, needle string) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(raw), needle)
}

// commandArgs starts a headless agent that can write. Nobody is present to approve
// a prompt, so the flags come from the registry and are never built here.
func commandArgs(name string) []string {
	if spec, ok := Find(name); ok {
		return spec.Args
	}
	return []string{"-p"}
}

// Run executes the CLI with the prompt on stdin and returns stdout.
func Run(ctx context.Context, name, workspace, prompt string) (string, error) {
	return RunWriter(ctx, name, workspace, prompt, nil)
}

// RunWriter does what Run does and copies stdout to sink while the process runs.
func RunWriter(ctx context.Context, name, workspace, prompt string, sink io.Writer) (string, error) {
	cmd := exec.CommandContext(ctx, name, commandArgs(name)...)
	proc.Tame(cmd)
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
	defer proc.Track(pid)()
	if err := cmd.Wait(); err != nil {
		// A cancelled run is not a blocked one: the output of a killed CLI can mention
		// quota or login by accident, and Classify must not turn a Stop into a hold.
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		combined := stdout.String() + "\n" + stderr.String()
		if kind, detail, found := Classify(combined); found {
			return "", &Blocked{Agent: name, Kind: kind, Detail: detail, Err: err}
		}
		return "", fmt.Errorf("%s: %w: %s", name, err, tailOutput(combined))
	}
	return stdout.String(), nil
}

// tailOutput keeps the last lines of what the CLI printed. A CLI can write its whole
// diagnosis to stdout, and an empty reason is worse than a short one.
func tailOutput(text string) string {
	var kept []string
	lines := strings.Split(strings.TrimSpace(text), "\n")
	for i := len(lines) - 1; i >= 0 && len(kept) < 6; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			kept = append([]string{line}, kept...)
		}
	}
	out := strings.Join(kept, " | ")
	if len(out) > 400 {
		out = out[len(out)-400:]
	}
	return out
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
