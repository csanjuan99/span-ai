package serve

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"span/internal/adapt"
	"span/internal/branch"
	"span/internal/policy"
)

// maxBranches caps the list the desk offers; a repository with thousands of branches would
// otherwise send them all to a datalist nobody scrolls.
const maxBranches = 200

// branchList is what GET /branches returns: the branches a run may start from, and the base
// it uses when the form leaves the field empty.
type branchList struct {
	Branches []string `json:"branches"`
	Default  string   `json:"default"`
}

func handleBranches(w http.ResponseWriter, r *http.Request) {
	list, err := listBranches(r.Context(), r.URL.Query().Get("path"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// listBranches reads the local and origin branches of the repository at path. A folder that
// is not a repository (a folder of repositories, say) has none to offer, which is not an
// error: the field still takes a typed name.
func listBranches(ctx context.Context, path string) (branchList, error) {
	if strings.TrimSpace(path) == "" {
		return branchList{}, errors.New("invalid: path is required")
	}
	dir, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil {
		return branchList{}, err
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return branchList{}, fmt.Errorf("invalid: %s is not a folder", dir)
	}
	base, err := policy.ProjectBase(dir)
	if err != nil {
		return branchList{}, err
	}
	if base == "" {
		base = branch.Dev
	}
	out := branchList{Branches: []string{}, Default: base}
	if !isRepo(dir) {
		return out, nil
	}
	refs, err := adapt.Git{Dir: dir}.Git(ctx, dir, "for-each-ref", "--sort=-committerdate", "--format=%(refname)", "refs/heads", "refs/remotes/origin")
	if err != nil {
		return branchList{}, err
	}
	out.Branches = branchNames(refs, base)
	return out, nil
}

func isRepo(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

// recentBranches is how many of the most recently committed branches lead the list.
const recentBranches = 50

// branchNames turns full ref names, newest commit first, into branch names: origin/ is
// dropped so a branch known locally and on origin is listed once, and origin's HEAD is not
// a branch. The default base leads when it exists, then the 50 most recently committed
// branches, then the rest alphabetically, 200 in all.
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
