package setup

import (
	"context"
	"fmt"
	"path/filepath"

	"span/internal/branch"
	"span/internal/launch"
	"span/internal/policy"
)

// Targets resolves the repositories a launch request runs in.
func Targets(ctx context.Context, req Request) ([]launch.Target, error) {
	return launch.Resolve(ctx, launch.Request{
		Path: req.Workspace, Names: req.Repos, Issue: req.Issue, Story: req.Story,
	}, nil)
}

// ForTarget is the request for one repository of a launch: its path, the folder the launch
// started from, its own story and issue, and its base.
func ForTarget(req Request, target launch.Target) (Request, error) {
	one := req
	one.Workspace = target.Path
	one.Folder = LaunchFolder(req.Workspace, target.Path)
	one.Story = target.Story
	one.Issue = target.Issue
	return WithBase(one)
}

// LaunchFolder is the folder to remember for a run in target: the folder the launch started
// from when target is a repository inside it, and "" when the run starts in the repository
// itself (a single repository, or the repository chosen directly).
func LaunchFolder(folder, target string) string {
	from, err := filepath.Abs(folder)
	if err != nil {
		return ""
	}
	to, err := filepath.Abs(target)
	if err != nil || filepath.Clean(from) == filepath.Clean(to) {
		return ""
	}
	return from
}

// WithBase settles a launch's kind and base before its branch is derived (branch.Choose):
// the base the run names wins; otherwise span.yaml's base applies to feat and fix only,
// and the kind is fixed so that base cannot make the story a hotfix. Still empty, the
// kind's default applies when the branch is derived.
func WithBase(req Request) (Request, error) {
	chosen, err := ProjectBase(req.Workspace)
	if err != nil {
		return req, err
	}
	kind, base, err := branch.Choose(req.Story, req.Kind, req.Base, chosen)
	if err != nil {
		return req, err
	}
	req.Kind, req.Base = kind, base
	return req, nil
}

// ProjectBase is the base: line of span.yaml in project, or "" when there is none.
func ProjectBase(project string) (string, error) {
	if project == "" {
		return "", nil
	}
	return policy.ProjectBase(project)
}

// Summary is what a launch prints for a repository it configured.
func Summary(repo string, res Result) string {
	return fmt.Sprintf("repo %s\npolicy %s\nbranch %s\nbase %s\n", repo, res.PolicyPath, res.Branch, res.Base)
}
