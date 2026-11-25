package serve

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"span/internal/jira"
)

// folderView is what the welcome page needs to say about a folder: whether it is there,
// whether it is a git repository, where it stands, and why when it is not usable.
type folderView struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	Git    bool   `json:"git"`
	Branch string `json:"branch,omitempty"`
	Remote string `json:"remote,omitempty"`
	// Dirty counts the files with uncommitted changes, untracked ones included.
	Dirty  int    `json:"dirty"`
	Reason string `json:"reason,omitempty"`
}

// folderEntry is one directory in the folder picker.
type folderEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Git  bool   `json:"git"`
}

// folderList is the directories under a path, for the Browse button.
type folderList struct {
	Path   string        `json:"path"`
	Parent string        `json:"parent"`
	Dirs   []folderEntry `json:"dirs"`
}

// maxFolders caps one listing so a folder with thousands of entries stays usable.
const maxFolders = 300

// jiraCheck is the answer to testing a Jira account: ok, unauthorized, unreachable or invalid.
type jiraCheck struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// expandHome turns a leading ~ into the user's home folder and makes the path absolute.
func expandHome(path string) string {
	clean := strings.TrimSpace(path)
	if clean == "~" || strings.HasPrefix(clean, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			clean = filepath.Join(home, strings.TrimPrefix(clean, "~"))
		}
	}
	if abs, err := filepath.Abs(clean); err == nil {
		return abs
	}
	return clean
}

// folderGit runs one git command in dir and returns its trimmed output.
func folderGit(ctx context.Context, dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// probeFolder reads a folder without changing it. A path that starts with a dash is refused
// so it can never be taken for an option.
func probeFolder(ctx context.Context, raw string) folderView {
	path := strings.TrimSpace(raw)
	view := folderView{Path: path}
	if path == "" || strings.HasPrefix(path, "-") {
		view.Reason = "Enter the folder with your code."
		return view
	}
	dir := expandHome(path)
	view.Path = dir
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		view.Reason = "The folder does not exist or is not a directory."
		return view
	}
	view.Exists = true
	if _, err := folderGit(ctx, dir, "rev-parse", "--git-dir"); err != nil {
		view.Reason = "This folder is not a git repository. Run git init, or pick the folder that holds your code."
		return view
	}
	view.Git = true
	if branch, err := folderGit(ctx, dir, "symbolic-ref", "--short", "-q", "HEAD"); err == nil {
		view.Branch = branch
	}
	if remote, err := folderGit(ctx, dir, "remote"); err == nil {
		view.Remote, _, _ = strings.Cut(remote, "\n")
	}
	if status, err := folderGit(ctx, dir, "status", "--porcelain"); err == nil && status != "" {
		view.Dirty = len(strings.Split(status, "\n"))
	}
	return view
}

// listFolders lists the directories under a path, hidden ones left out, repositories marked.
func listFolders(raw string) (folderList, error) {
	path := strings.TrimSpace(raw)
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return folderList{}, err
		}
		path = home
	}
	if strings.HasPrefix(path, "-") {
		return folderList{}, errors.New("that path is not a folder")
	}
	dir := expandHome(path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return folderList{}, err
	}
	list := folderList{Path: dir, Parent: filepath.Dir(dir), Dirs: []folderEntry{}}
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		child := filepath.Join(dir, entry.Name())
		_, gitErr := os.Stat(filepath.Join(child, ".git"))
		list.Dirs = append(list.Dirs, folderEntry{Name: entry.Name(), Path: child, Git: gitErr == nil})
	}
	sort.Slice(list.Dirs, func(i, j int) bool { return list.Dirs[i].Name < list.Dirs[j].Name })
	if len(list.Dirs) > maxFolders {
		list.Dirs = list.Dirs[:maxFolders]
	}
	return list, nil
}

// handleFolder answers GET /onboarding/folder?path=.
func handleFolder(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, probeFolder(r.Context(), r.URL.Query().Get("path")))
}

// handleFolders answers GET /onboarding/folders?path=.
func handleFolders(w http.ResponseWriter, r *http.Request) {
	list, err := listFolders(r.URL.Query().Get("path"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// checkJira asks the site whether it accepts the account. A blank token falls back to the one
// already saved for the project. Nothing is written.
func checkJira(ctx context.Context, project string, form url.Values) jiraCheck {
	account := jira.Account{
		BaseURL: strings.TrimRight(strings.TrimSpace(form.Get("url")), "/"),
		Email:   strings.TrimSpace(form.Get("email")),
		Token:   strings.TrimSpace(form.Get("token")),
	}
	if account.Token == "" {
		if saved, err := jira.Load(project); err == nil {
			account.Token = saved.Token
		}
	}
	if !strings.Contains(account.BaseURL, "://") || account.Token == "" {
		return jiraCheck{Status: "invalid", Message: "Enter the Jira site and an API token."}
	}
	_, err := (jira.Client{Account: account}).Search(ctx, "created >= -1d ORDER BY created DESC")
	var failure *jira.Error
	switch {
	case err == nil:
		return jiraCheck{Status: "ok"}
	case errors.As(err, &failure) && failure.Code == "unauthorized":
		return jiraCheck{Status: "unauthorized", Message: "Jira did not accept this token. It is wrong or was revoked."}
	case errors.As(err, &failure) && failure.Code == "unreachable":
		return jiraCheck{Status: "unreachable", Message: "The site did not answer."}
	default:
		return jiraCheck{Status: "invalid", Message: err.Error()}
	}
}

// handleJiraCheck answers POST /jira/check.
func handleJiraCheck(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, checkJira(r.Context(), r.PostForm.Get("project"), r.PostForm))
}
