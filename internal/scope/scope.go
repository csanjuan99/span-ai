// Package scope finds the git repositories Span may write to.
// A git checkout is that repository. A folder of checkouts is a choice.
package scope

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Repo is one git checkout the caller can select.
type Repo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Set is what the current folder contains.
// Mode "repo" means the folder is one checkout. Mode "folder" means the
// caller is looking at several checkouts and must choose.
type Set struct {
	Root  string `json:"root"`
	Mode  string `json:"mode"`
	Repos []Repo `json:"repos"`
}

// Error is a failure the caller can handle.
type Error struct {
	Code string
	Msg  string
}

func (e *Error) Error() string {
	if e.Msg == "" {
		return e.Code
	}
	return e.Code + ": " + e.Msg
}

func (e *Error) SpanCode() string { return e.Code }

// Discover reads path. A checkout with no child checkouts is that repo.
// Child checkouts are listed with the parent when the parent is also a checkout.
func Discover(path string) (Set, error) {
	abs, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil {
		return Set{}, err
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return Set{}, &Error{Code: "no_repo", Msg: "path is not a directory"}
	}
	var repos []Repo
	if isGit(abs) {
		repos = append(repos, Repo{Name: filepath.Base(abs), Path: abs})
	}
	children, err := childRepos(abs)
	if err != nil {
		return Set{}, err
	}
	repos = append(repos, children...)
	if len(repos) == 0 {
		return Set{Root: abs, Mode: "folder"}, &Error{Code: "no_repo", Msg: "no git repository here"}
	}
	mode := "folder"
	if len(repos) == 1 && repos[0].Path == abs {
		mode = "repo"
	}
	return Set{Root: abs, Mode: mode, Repos: repos}, nil
}

// Select returns the named repos. An empty selection keeps the only repo.
func Select(set Set, names []string) ([]Repo, error) {
	if len(set.Repos) == 0 {
		return nil, &Error{Code: "no_repo", Msg: "no git repository here"}
	}
	wanted := clean(names)
	if len(wanted) == 0 {
		if len(set.Repos) == 1 {
			return set.Repos, nil
		}
		return nil, &Error{Code: "selection_required", Msg: "choose one or more repos"}
	}
	byName := map[string]Repo{}
	for _, repo := range set.Repos {
		byName[repo.Name] = repo
	}
	var out []Repo
	seen := map[string]bool{}
	for _, name := range wanted {
		repo, ok := byName[name]
		if !ok {
			return nil, &Error{Code: "unknown_repo", Msg: name}
		}
		if seen[repo.Path] {
			continue
		}
		seen[repo.Path] = true
		out = append(out, repo)
	}
	return out, nil
}

func childRepos(root string) ([]Repo, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var repos []Repo
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() || strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
			continue
		}
		path := filepath.Join(root, name)
		if !isGit(path) {
			continue
		}
		repos = append(repos, Repo{Name: name, Path: path})
	}
	sort.Slice(repos, func(i, j int) bool { return repos[i].Name < repos[j].Name })
	return repos, nil
}

func isGit(path string) bool {
	info, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil && (info.IsDir() || info.Mode().IsRegular())
}

func clean(names []string) []string {
	var out []string
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" {
			out = append(out, name)
		}
	}
	return out
}

// Focus tells the agent which checkout this run may change.
func Focus(story string, repo Repo, all []Repo) string {
	story = strings.TrimSpace(story)
	if story == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString(story)
	fmt.Fprintf(&b, "\n\nWrite only in repo %s.\n", repo.Name)
	if len(all) < 2 {
		return b.String()
	}
	var others []string
	for _, item := range all {
		if item.Path != repo.Path {
			others = append(others, item.Name)
		}
	}
	fmt.Fprintf(&b, "Also selected: %s.\n", strings.Join(others, ", "))
	return b.String()
}
