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
	"sort"
	"strings"

	"span/internal/agent"
	"span/internal/engine"
	"span/internal/proc"
	"span/internal/prove"
)

// Exec runs a declared command as argv, without a shell.
type Exec struct{}

// OutputCap is how much of a command's output is kept. A failing suite prints the
// failure at the end, so the cap keeps the tail and drops the start. The captured
// text goes to the run log; only a short tail travels on the statement.
const OutputCap = 1 << 20

func (Exec) Run(ctx context.Context, command, dir string) (prove.Output, error) {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return prove.Output{}, errors.New("empty command")
	}
	cmd := exec.CommandContext(ctx, fields[0], fields[1:]...)
	proc.Tame(cmd)
	cmd.Dir = dir
	kept := &tailBuffer{max: OutputCap}
	cmd.Stdout = kept
	cmd.Stderr = kept
	err := proc.Run(cmd)
	out := prove.Output{Text: kept.String(), Truncated: kept.Truncated()}
	if err == nil {
		return out, nil
	}
	// A killed check did not fail, it did not finish. Reporting exit -1 would send the run
	// back to build and spend a repair on a Stop or Ctrl-C.
	if ctx.Err() != nil {
		return out, ctx.Err()
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		out.ExitCode = exit.ExitCode()
		return out, nil
	}
	return out, err
}

// tailBuffer keeps the last max bytes written to it. One goroutine writes at a
// time because the same pointer is handed to both stdout and stderr.
type tailBuffer struct {
	max     int
	data    []byte
	written int
}

// Truncated reports whether the start of the output was dropped.
func (b *tailBuffer) Truncated() bool { return b.written > b.max }

func (b *tailBuffer) Write(p []byte) (int, error) {
	b.written += len(p)
	b.data = append(b.data, p...)
	if len(b.data) > 2*b.max {
		b.data = append(b.data[:0], b.data[len(b.data)-b.max:]...)
	}
	return len(p), nil
}

func (b *tailBuffer) String() string {
	if len(b.data) <= b.max {
		return string(b.data)
	}
	return string(b.data[len(b.data)-b.max:])
}

// Git is the workspace source.
type Git struct{ Dir string }

// IsWorktree reports whether dir is a linked worktree. A linked worktree has a .git
// file that points at the shared repository, so the check is exact and needs no git call.
func IsWorktree(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil && !info.IsDir()
}

// Git runs one git command in dir for the packages that own a command surface.
func (g Git) Git(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (g Git) Head(ctx context.Context) (string, error) {
	return g.output(ctx, "rev-parse", "HEAD")
}

func (g Git) Dirty(ctx context.Context) (bool, error) {
	// Span's own state lives at .span inside the workspace. Counting it here would
	// make every run permanently dirty, which turns off evidence reuse and leaves a
	// check that already passed looking like a fresh change.
	out, err := g.output(ctx, "status", "--porcelain", "--", ".", ":(exclude).span")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

func (g Git) Commit(ctx context.Context, message string) (string, error) {
	// span.yaml is the person's policy, not the story's work: committing it puts it on one
	// branch only, and the next checkout of another branch loses it.
	if _, err := g.output(ctx, "add", "-A", "--", ".", ":(exclude).span", ":(exclude)span.yaml"); err != nil {
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
	// The author is whoever this checkout is configured with. Span does not invent an
	// identity and does not overwrite the user's: the commit is theirs.
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = g.Dir
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

// ErrBroadChange means the branch deleted, renamed, copied or retyped a path. What used to
// import or load that path is not in a list of changed names, so the caller proves the
// whole suite instead of a narrowed one.
var ErrBroadChange = errors.New("the change deletes, renames, copies or retypes a path")

// Changed lists the files this branch added or modified since it left base, as git diff
// base...HEAD prints them, sorted. base is tried as given, then as origin/<base>; neither
// resolving is an error. Any other kind of change returns ErrBroadChange.
func (g Git) Changed(ctx context.Context, base string) ([]string, error) {
	base = strings.TrimSpace(base)
	if base == "" {
		return nil, errors.New("no base branch")
	}
	ref := ""
	for _, candidate := range []string{base, "origin/" + base} {
		if _, err := g.output(ctx, "rev-parse", "--verify", "--quiet", candidate+"^{commit}"); err == nil {
			ref = candidate
			break
		}
	}
	if ref == "" {
		return nil, fmt.Errorf("base branch %s is missing", base)
	}
	// -z keeps every path byte for byte: no quoting of unusual names to undo.
	out, err := g.output(ctx, "diff", "--name-status", "-z", "-M", ref+"...HEAD")
	if err != nil {
		return nil, err
	}
	files := []string{}
	fields := strings.Split(strings.Trim(out, "\x00"), "\x00")
	for i := 0; i < len(fields); i++ {
		status := fields[i]
		if status == "" {
			continue
		}
		switch status[0] {
		case 'A', 'M':
			if i+1 >= len(fields) {
				return nil, fmt.Errorf("git diff printed %q without a path", status)
			}
			i++
			files = append(files, fields[i])
		default:
			return nil, ErrBroadChange
		}
	}
	sort.Strings(files)
	return files, nil
}

func (g Git) bounded(ctx context.Context, limit int, args ...string) (string, bool, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = g.Dir
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
	out, err := h.gh(ctx, "pr", "create", "--head", branch, "--title", title, "--body", body)
	if err != nil {
		return "", err
	}
	// gh prints the URL and a newline. Kept, the newline makes a later `gh pr view <url>`
	// read the argument as a branch name.
	return strings.TrimSpace(out), nil
}

func (h Host) Merged(ctx context.Context, url string) (bool, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return LocalReview{Dir: h.State}.Merged(ctx, url)
	}
	out, err := h.gh(ctx, "pr", "view", strings.TrimSpace(url), "--json", "state", "--jq", ".state")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "MERGED", nil
}

func (h Host) Merge(ctx context.Context, url string) error {
	if _, err := exec.LookPath("gh"); err != nil {
		return LocalReview{Dir: h.State}.Merge(ctx, url)
	}
	_, err := h.gh(ctx, "pr", "merge", strings.TrimSpace(url), "--merge")
	var refused *ghError
	if errors.As(err, &refused) && IsConflictMessage(refused.stderr) {
		// A conflict is a known outcome: nothing was merged, so the run can resolve it
		// instead of holding on an unknown one. Only gh's own words are read: the command
		// line carries the URL, and a repository can be named after conflicts.
		return fmt.Errorf("%w: %s", ErrConflict, err.Error())
	}
	if errors.As(err, &refused) && IsBehindMessage(refused.stderr) {
		return fmt.Errorf("%w: %s", engine.ErrBehind, err.Error())
	}
	return err
}

// ghError is a gh command that failed, with its stderr kept apart from the command line.
type ghError struct {
	args   []string
	err    error
	stderr string
}

func (e *ghError) Error() string {
	return fmt.Sprintf("gh %s: %v: %s", strings.Join(e.args, " "), e.err, e.stderr)
}

func (e *ghError) Unwrap() error { return e.err }

func (h Host) gh(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Dir = h.Workspace
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", &ghError{args: args, err: err, stderr: stderr.String()}
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
	proc.Tame(cmd)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "SPAN_SHA="+sha)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := proc.Run(cmd); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
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
		home = agent.Home()
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

// Propose asks the agent for conflict options. Its stdout is not copied to the run
// transcript: the reply carries file contents, which belong in the plan, not the log.
func (c CLIAgents) Propose(ctx context.Context, name, workspace, text string) (engine.ConflictReply, error) {
	out, err := agent.RunWriter(ctx, name, workspace, text, nil)
	if err != nil {
		return engine.ConflictReply{}, err
	}
	var reply engine.ConflictReply
	if err := agent.ParseJSON(out, &reply); err != nil {
		return engine.ConflictReply{}, err
	}
	return reply, nil
}
