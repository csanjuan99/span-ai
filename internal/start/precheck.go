package start

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"

	"span/internal/adapt"
	"span/internal/agent"
	"span/internal/branch"
	"span/internal/engine"
	"span/internal/jira"
	"span/internal/launch"
	"span/internal/scope"
	"span/internal/setup"
	"span/internal/worktree"
)

// CheckInput is everything the desk sends before Start.
type CheckInput struct {
	Workspace string
	Story     string
	Kind      string
	Issue     string
	Base      string
	Repos     []string
	SpansRoot string
	// InPlace says the run would edit the checkout itself.
	InPlace bool
}

// Precheck returns the first refusal that would block a start, or nil when nothing blocks yet.
// An error means the check itself could not run, which is not the same as a refusal.
func Precheck(ctx context.Context, in CheckInput) (*Refusal, error) {
	req := setup.Request{
		Workspace: strings.TrimSpace(in.Workspace),
		Story:     strings.TrimSpace(in.Story),
		Kind:      strings.TrimSpace(in.Kind),
		Issue:     strings.TrimSpace(in.Issue),
		Base:      strings.TrimSpace(in.Base),
		Repos:     in.Repos,
	}
	if req.Workspace == "" {
		req.Workspace = "."
	}
	if _, err := scope.Discover(req.Workspace); err != nil {
		return refusalOf(err)
	}
	targets, err := launch.Resolve(ctx, launch.Request{
		Path: req.Workspace, Names: req.Repos, Issue: req.Issue, Story: req.Story,
	}, nil)
	if err != nil {
		return refusalOf(err)
	}
	for _, target := range targets {
		ref, err := checkTarget(ctx, req, target, in)
		if ref != nil || err != nil {
			return ref, err
		}
	}
	return checkAgents(req.Workspace), nil
}

// refusalOf turns a cause the desk can explain into a refusal. Any other error is returned as is.
func refusalOf(err error) (*Refusal, error) {
	var coded *scope.Error
	if errors.As(err, &coded) && coded.SpanCode() == "no_repo" {
		r := refuse(NumberNotGit, "not_git", "%s", coded.Error())
		return &r, nil
	}
	var je *jira.Error
	if errors.As(err, &je) {
		switch je.SpanCode() {
		case "not_configured":
			r := refuse(NumberJiraSignedOut, "jira_signed_out", "%s", je.Error())
			return &r, nil
		case "unauthorized":
			r := refuse(NumberTokenRefused, "token_refused", "%s", je.Error())
			return &r, nil
		}
	}
	return nil, err
}

func checkTarget(ctx context.Context, req setup.Request, target launch.Target, in CheckInput) (*Refusal, error) {
	var plan branch.Plan
	if strings.TrimSpace(target.Story) != "" {
		// Derive only names the branch; unlike setup.Configure it writes nothing to the project.
		derived, err := branch.Derive(target.Story, target.Issue, req.Kind, req.Base)
		if err != nil {
			return nil, err
		}
		plan = derived
	}
	// A busy checkout only blocks a run that would edit it; a run with its own worktree
	// starts beside whatever holds it.
	separate := !in.InPlace && worktree.Separate(ctx, adapt.Git{Dir: target.Path}, target.Path, plan.Name)
	if holder, busy := engine.WorkspaceBusy(target.Path); busy && !separate {
		r := refuse(NumberCheckoutBusy, "checkout_busy", "%s is busy with %s", filepath.Base(target.Path), holder)
		r.RunID = holder
		return &r, nil
	}
	if base := strings.TrimSpace(req.Base); base != "" {
		list, err := ListBranches(ctx, target.Path)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(list.Branches, base) {
			r := refuse(NumberBaseMissing, "base_missing", "branch %q is not in %s", base, target.Name)
			r.Base = base
			return &r, nil
		}
	}
	if plan.Name == "" {
		return nil, nil
	}
	if id := findRun(in.SpansRoot, target.Path, plan.Name); id != "" {
		r := refuse(NumberRunExists, "run_exists", "a run for %s already exists", plan.Name)
		r.RunID = id
		r.Branch = plan.Name
		return &r, nil
	}
	return nil, nil
}

func checkAgents(project string) *Refusal {
	pick := ResolveAgent(project)
	if pick.Chosen != "" {
		return nil
	}
	if len(agent.Available(agent.Home())) > 0 {
		return nil
	}
	reason := pick.Reason
	if reason == "" {
		reason = "no agent CLI is ready"
	}
	r := refuse(NumberNoAgent, "no_agent", "%s", reason)
	return &r
}

func findRun(spansRoot, workspace, branchName string) string {
	if strings.TrimSpace(spansRoot) == "" || branchName == "" {
		return ""
	}
	for _, head := range engine.Heads(spansRoot) {
		if head.Branch != branchName {
			continue
		}
		if workspace == "" || head.Repo == filepath.Base(workspace) {
			return head.ID
		}
	}
	return ""
}
