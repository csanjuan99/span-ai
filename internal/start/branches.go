package start

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"span/internal/adapt"
	"span/internal/branch"
	"span/internal/policy"
)

// maxBranches caps the list the desk offers.
const maxBranches = 200

// recentBranches is how many of the most recently committed branches lead the list.
const recentBranches = 50

// BranchList is what GET /branches and precheck use.
type BranchList struct {
	Branches []string `json:"branches"`
	Default  string   `json:"default"`
}

// ListBranches reads local and origin branch names for one repository.
func ListBranches(ctx context.Context, path string) (BranchList, error) {
	if strings.TrimSpace(path) == "" {
		return BranchList{}, fmt.Errorf("invalid: path is required")
	}
	dir, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil {
		return BranchList{}, err
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return BranchList{}, fmt.Errorf("invalid: %s is not a folder", dir)
	}
	base, err := policy.ProjectBase(dir)
	if err != nil {
		return BranchList{}, err
	}
	if base == "" {
		base = branch.Dev
	}
	out := BranchList{Branches: []string{}, Default: base}
	if !isRepo(dir) {
		return out, nil
	}
	refs, err := adapt.Git{Dir: dir}.Git(ctx, dir, "for-each-ref", "--sort=-committerdate", "--format=%(refname)", "refs/heads", "refs/remotes/origin")
	if err != nil {
		return BranchList{}, err
	}
	out.Branches = branchNames(refs, base)
	return out, nil
}

func isRepo(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

func branchNames(refs, fallback string) []string {
	seen := map[string]bool{}
	byDate := []string{}
	for _, ref := range strings.Split(refs, "\n") {
		ref = strings.TrimSpace(ref)
		name, ok := strings.CutPrefix(ref, "refs/heads/")
		if !ok {
			name, ok = strings.CutPrefix(ref, "refs/remotes/origin/")
		}
		if !ok || name == "" || name == "HEAD" || seen[name] {
			continue
		}
		seen[name] = true
		byDate = append(byDate, name)
	}
	names, pinned := []string{}, 0
	if seen[fallback] {
		names, pinned = append(names, fallback), 1
	}
	rest := []string{}
	for _, name := range byDate {
		switch {
		case name == fallback:
		case len(names) < pinned+recentBranches:
			names = append(names, name)
		default:
			rest = append(rest, name)
		}
	}
	sort.Strings(rest)
	names = append(names, rest...)
	if len(names) > maxBranches {
		names = names[:maxBranches]
	}
	return names
}
