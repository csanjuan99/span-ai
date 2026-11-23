// Package launch turns a folder and an optional Jira key into the repos Span will run.
package launch

import (
	"context"
	"strings"

	"span/internal/jira"
	"span/internal/scope"
)

// Request is the caller's choice of folder, repos, and story.
type Request struct {
	Path  string
	Names []string
	Issue string
	Story string
}

// Target is one checkout to configure or run.
type Target struct {
	Name  string
	Path  string
	Issue string
	Story string
}

// Fetcher reads one Jira issue. Nil uses the saved account.
type Fetcher func(context.Context, string) (jira.Story, error)

// Resolve finds the selected repos and, when the story text is empty, loads it from Jira.
func Resolve(ctx context.Context, req Request, fetch Fetcher) ([]Target, error) {
	set, err := scope.Discover(req.Path)
	if err != nil {
		return nil, err
	}
	chosen, err := scope.Select(set, req.Names)
	if err != nil {
		return nil, err
	}
	story := strings.TrimSpace(req.Story)
	issue := strings.ToUpper(strings.TrimSpace(req.Issue))
	if issue != "" && story == "" {
		if fetch == nil {
			fetch = savedIssue(req.Path)
		}
		loaded, err := fetch(ctx, issue)
		if err != nil {
			return nil, err
		}
		story = strings.TrimSpace(loaded.Brief)
		issue = loaded.Key
	}
	repos := make([]scope.Repo, len(chosen))
	copy(repos, chosen)
	out := make([]Target, 0, len(chosen))
	for _, repo := range chosen {
		out = append(out, Target{
			Name:  repo.Name,
			Path:  repo.Path,
			Issue: issue,
			Story: scope.Focus(story, repo, repos),
		})
	}
	return out, nil
}

func savedIssue(project string) Fetcher {
	return func(ctx context.Context, key string) (jira.Story, error) {
		account, err := jira.Load(project)
		if err != nil {
			return jira.Story{}, err
		}
		return (jira.Client{Account: account}).Issue(ctx, key)
	}
}
