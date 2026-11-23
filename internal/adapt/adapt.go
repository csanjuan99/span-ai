// Package adapt binds the engine to git, a command checker, and a change host.
package adapt

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"span/internal/agent"
	"span/internal/engine"
)

// Exec runs a declared command as argv, without a shell.
type Exec struct{}

func (Exec) Run(ctx context.Context, command, dir string) (int, error) {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return 0, errors.New("empty command")
	}
	cmd := exec.CommandContext(ctx, fields[0], fields[1:]...)
	cmd.Dir = dir
	err := cmd.Run()
	if err == nil {
		return 0, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), nil
	}
	return 0, err
}

// Git is the workspace source.
type Git struct{ Dir string }

func (g Git) Head(ctx context.Context) (string, error) {
	return g.output(ctx, "rev-parse", "HEAD")
}

func (g Git) Dirty(ctx context.Context) (bool, error) {
	out, err := g.output(ctx, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

func (g Git) Commit(ctx context.Context, message string) (string, error) {
	if _, err := g.output(ctx, "add", "-A", "--", ".", ":(exclude).span"); err != nil {
		return "", err
	}
	if _, err := g.output(ctx, "commit", "-m", message); err != nil {
		return "", err
	}
	return g.Head(ctx)
}

func (g Git) Push(ctx context.Context) error {
	_, err := g.output(ctx, "push", "origin", "HEAD")
	return err
}

func (g Git) Branch(ctx context.Context) (string, error) {
	return g.output(ctx, "rev-parse", "--abbrev-ref", "HEAD")
}

// EnsureBranch checks out branch. An existing branch is not reset to base.
func (g Git) EnsureBranch(ctx context.Context, base, branch string) error {
	current, err := g.Branch(ctx)
	if err != nil {
		return err
	}
	if current == branch {
		return nil
	}
	if _, err := g.output(ctx, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch); err == nil {
		_, err = g.output(ctx, "checkout", branch)
		return err
	}
	if _, err := g.output(ctx, "rev-parse", "--verify", "--quiet", base); err != nil {
		return fmt.Errorf("base branch %s is missing", base)
	}
	_, err = g.output(ctx, "checkout", "-b", branch, base)
	return err
}

func (g Git) output(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = g.Dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Span",
		"GIT_AUTHOR_EMAIL=span@example.test",
		"GIT_COMMITTER_NAME=Span",
		"GIT_COMMITTER_EMAIL=span@example.test",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

const patchLimit = 256 * 1024

// Diff is the change from base to head. Patch is cut at 256 KiB.
type Diff struct {
	Stat      string `json:"stat"`
	Patch     string `json:"patch"`
	Truncated bool   `json:"truncated"`
}

// Diff reads git diff base...head without a shell.
func (g Git) Diff(ctx context.Context, base, head string) (Diff, error) {
	if strings.TrimSpace(base) == "" || strings.TrimSpace(head) == "" {
		return Diff{}, nil
	}
	spec := base + "..." + head
	stat, err := g.output(ctx, "diff", "--stat", spec)
	if err != nil {
		return Diff{}, err
	}
	patch, truncated, err := g.bounded(ctx, patchLimit, "diff", spec)
	if err != nil {
		return Diff{}, err
	}
	return Diff{Stat: stat, Patch: patch, Truncated: truncated}, nil
}

func (g Git) bounded(ctx context.Context, limit int, args ...string) (string, bool, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = g.Dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Span",
		"GIT_AUTHOR_EMAIL=span@example.test",
		"GIT_COMMITTER_NAME=Span",
		"GIT_COMMITTER_EMAIL=span@example.test",
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", false, err
	}
	if err := cmd.Start(); err != nil {
		return "", false, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}
	buf := make([]byte, limit+1)
	n, readErr := io.ReadFull(stdout, buf)
	if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
		_ = cmd.Wait()
		return "", false, readErr
	}
	truncated := n > limit
	if truncated {
		n = limit
	}
	_, _ = io.Copy(io.Discard, stdout)
	if err := cmd.Wait(); err != nil {
		return "", false, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}
	return strings.TrimSpace(string(buf[:n])), truncated, nil
}

// Host opens a GitHub pull request when gh is installed, and records a local
// change URL otherwise.
type Host struct {
	Workspace string
	State     string
}

func (h Host) Find(ctx context.Context, spanID, branch string) (string, bool, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return LocalReview{Dir: h.State}.Find(ctx, spanID, branch)
	}
	out, err := h.gh(ctx, "pr", "list", "--head", branch, "--json", "url", "--jq", ".[0].url")
	if err != nil {
		return "", false, err
	}
	out = strings.TrimSpace(out)
	if out == "" || out == "null" {
		return "", false, nil
	}
	return out, true, nil
}

func (h Host) Open(ctx context.Context, spanID, branch, title, body string) (string, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return LocalReview{Dir: h.State}.Open(ctx, spanID, branch, title, body)
	}
	return h.gh(ctx, "pr", "create", "--head", branch, "--title", title, "--body", body)
}

func (h Host) Merged(ctx context.Context, url string) (bool, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return LocalReview{Dir: h.State}.Merged(ctx, url)
	}
	out, err := h.gh(ctx, "pr", "view", url, "--json", "state", "--jq", ".state")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "MERGED", nil
}

func (h Host) Merge(ctx context.Context, url string) error {
	if _, err := exec.LookPath("gh"); err != nil {
		return LocalReview{Dir: h.State}.Merge(ctx, url)
	}
	_, err := h.gh(ctx, "pr", "merge", url, "--merge")
	return err
}

func (h Host) gh(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Dir = h.Workspace
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gh %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String(), nil
}

// LocalReview records the change inside the span directory when no forge CLI exists.
type LocalReview struct{ Dir string }

func (r LocalReview) Find(_ context.Context, spanID, _ string) (string, bool, error) {
	raw, err := os.ReadFile(filepath.Join(r.Dir, spanID, "change.url"))
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return strings.TrimSpace(string(raw)), true, nil
}

func (r LocalReview) Open(_ context.Context, spanID, branch, _, _ string) (string, error) {
	url := "local:" + spanID + ":" + branch
	dir := filepath.Join(r.Dir, spanID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return url, os.WriteFile(filepath.Join(dir, "change.url"), []byte(url+"\n"), 0o644)
}

func (r LocalReview) Merged(_ context.Context, url string) (bool, error) {
	dir, ok := r.spanDir(url)
	if !ok {
		return false, nil
	}
	_, err := os.Stat(filepath.Join(dir, "merged"))
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func (r LocalReview) Merge(_ context.Context, url string) error {
	dir, ok := r.spanDir(url)
	if !ok {
		return fmt.Errorf("not a local change: %s", url)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "merged"), []byte("merged\n"), 0o644)
}

func (r LocalReview) spanDir(url string) (string, bool) {
	rest, ok := strings.CutPrefix(url, "local:")
	if !ok {
		return "", false
	}
	id, _, ok := strings.Cut(rest, ":")
	if !ok || id == "" {
		return "", false
	}
	return filepath.Join(r.Dir, id), true
}

// CommandRelease runs the environment command declared by the project.
type CommandRelease struct{}

func (CommandRelease) Run(ctx context.Context, command, dir, sha string) error {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return errors.New("empty release command")
	}
	cmd := exec.CommandContext(ctx, fields[0], fields[1:]...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "SPAN_SHA="+sha)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, stderr.String())
	}
	return nil
}

// CLIAgents speaks to the selected agent binary.
type CLIAgents struct {
	Home string
	sink io.Writer
}

// Bind copies later agent stdout to w.
func (c *CLIAgents) Bind(w io.Writer) { c.sink = w }

func (c CLIAgents) Select(want string) (agent.Selection, error) {
	home := c.Home
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return agent.Select(want, home)
}

func (c CLIAgents) Frame(ctx context.Context, name, workspace, text string) (engine.FrameDoc, error) {
	out, err := agent.RunWriter(ctx, name, workspace, text, c.sink)
	if err != nil {
		return engine.FrameDoc{}, err
	}
	var doc engine.FrameDoc
	if err := agent.ParseJSON(out, &doc); err != nil {
		return engine.FrameDoc{}, err
	}
	return doc, nil
}

func (c CLIAgents) Build(ctx context.Context, name, workspace, text string) error {
	_, err := agent.RunWriter(ctx, name, workspace, text, c.sink)
	return err
}

func (c CLIAgents) Inspect(ctx context.Context, name, workspace, text string) (engine.InspectDoc, error) {
	out, err := agent.RunWriter(ctx, name, workspace, text, c.sink)
	if err != nil {
		return engine.InspectDoc{}, err
	}
	var doc engine.InspectDoc
	if err := agent.ParseJSON(out, &doc); err != nil {
		return engine.InspectDoc{}, err
	}
	return doc, nil
}
