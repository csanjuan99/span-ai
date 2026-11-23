// Package setup writes a project's span.yaml from what the repository already is,
// and names the branch for a story.
package setup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"span/internal/branch"
	"span/internal/handoff"
	"span/internal/harness"
	"span/internal/policy"
)

// Request is one quickstart. An empty story only writes configuration.
type Request struct {
	Workspace string
	Story     string
	Kind      string
	Issue     string
	Base      string
	Repos     []string
	Files     []handoff.File
	Force     bool
	// Folder is the folder the launch started from when Workspace is a repository inside it.
	// It is not read here; the run carries it so the folder's agent and models apply.
	Folder string
}

// Result is what quickstart wrote and the branch the run should use.
type Result struct {
	PolicyPath string
	BriefPath  string
	Branch     string
	Base       string
	Feature    string
	Checks     []string
	Harness    string
	Wrote      bool
}

// Configure detects checks, writes span.yaml when missing, and plans the branch.
func Configure(req Request) (Result, error) {
	workspace, err := filepath.Abs(strings.TrimSpace(req.Workspace))
	if err != nil {
		return Result{}, err
	}
	info, err := os.Stat(workspace)
	if err != nil || !info.IsDir() {
		return Result{}, fmt.Errorf("workspace %s is not a directory", workspace)
	}
	root := harness.Root()
	checks := Detect(workspace)
	out := Result{PolicyPath: filepath.Join(workspace, "span.yaml"), Checks: checks, Harness: root}
	story := strings.TrimSpace(req.Story)
	var plan branch.Plan
	if story != "" {
		plan, err = branch.Derive(story, req.Issue, req.Kind, req.Base)
		if err != nil {
			return Result{}, err
		}
		out.Branch = plan.Name
		out.Base = plan.Base
		out.Feature = plan.Feature
		if err := os.MkdirAll(filepath.Join(workspace, ".span"), 0o755); err != nil {
			return Result{}, err
		}
		out.BriefPath = filepath.Join(workspace, ".span", "brief.md")
		if err := os.WriteFile(out.BriefPath, []byte(story+"\n"), 0o644); err != nil {
			return Result{}, err
		}
	}
	if _, err := os.Stat(out.PolicyPath); err == nil && !req.Force {
		return out, nil
	}
	body := Defaults(root, checks)
	if err := os.WriteFile(out.PolicyPath, []byte(body), 0o644); err != nil {
		return Result{}, err
	}
	if _, err := policy.Load(out.PolicyPath); err != nil {
		return Result{}, err
	}
	out.Wrote = true
	return out, nil
}

// LoadOrDefault reads the policy at path. A missing file is not an error: span.yaml is
// optional, and the run gets the same policy quickstart would write for workspace. Nothing is
// written. A file that exists but does not load is still an error.
func LoadOrDefault(path, workspace string) (policy.File, error) {
	file, err := policy.Load(path)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		return file, err
	}
	return policy.Parse([]byte(Defaults(harness.Root(), Detect(workspace))))
}

// Defaults is the span.yaml quickstart writes: any agent, harness rules, open auto, merge
// hold, the detected checks, and every harness skill for build and inspect. It names no
// feature: a run's identity comes from its branch.
func Defaults(root string, checks []string) string {
	skills, err := harness.Skills(root)
	if err != nil {
		// Without a readable catalog the policy simply names no skills, as before.
		skills = nil
	}
	return render(root, checks, skills)
}

// Detect returns one argv check from the files at the workspace root.
func Detect(workspace string) []string {
	switch {
	case exists(filepath.Join(workspace, "go.mod")):
		return []string{"go test ./..."}
	case exists(filepath.Join(workspace, "package.json")):
		return []string{"npm test"}
	case exists(filepath.Join(workspace, "Cargo.toml")):
		return []string{"cargo test"}
	case exists(filepath.Join(workspace, "pyproject.toml")), exists(filepath.Join(workspace, "requirements.txt")):
		return []string{"python -m pytest"}
	default:
		return nil
	}
}

func exists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func render(root string, checks []string, skills []harness.Skill) string {
	var b strings.Builder
	b.WriteString("agent: any\nrules: harness\n")
	if root != "" {
		fmt.Fprintf(&b, "harness: %q\n", root)
	}
	b.WriteString("stages:\n  open: auto\n  merge: hold\n")
	if len(checks) > 0 {
		b.WriteString("prove:\n  fast:\n")
		for _, check := range checks {
			fmt.Fprintf(&b, "    - %s\n", check)
		}
	} else {
		b.WriteString("# No test command detected. Add prove.fast before trusting this gate.\n")
	}
	if len(skills) > 0 {
		b.WriteString("manifest:\n  build:\n    skills:\n")
		for _, skill := range skills {
			fmt.Fprintf(&b, "      - %s\n", skill.Name)
		}
		b.WriteString("  inspect:\n    skills:\n")
		for _, skill := range skills {
			fmt.Fprintf(&b, "      - %s\n", skill.Name)
		}
	}
	return b.String()
}
