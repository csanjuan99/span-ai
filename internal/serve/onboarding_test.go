package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func getJSONBody(t *testing.T, target string, into any) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	Handler(t.TempDir(), "").ServeHTTP(rec, req)
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), into); err != nil {
			t.Fatalf("%s: %v (%s)", target, err, rec.Body.String())
		}
	}
	return rec.Code
}

func gitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "first"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	return dir
}

func TestAFolderThatDoesNotExistSaysSo(t *testing.T) {
	var view folderView
	getJSONBody(t, "/onboarding/folder?path="+url.QueryEscape(filepath.Join(t.TempDir(), "nope")), &view)
	if view.Exists || view.Reason == "" {
		t.Fatalf("view = %+v", view)
	}
}

func TestAFolderThatIsNotARepositorySaysSo(t *testing.T) {
	var view folderView
	getJSONBody(t, "/onboarding/folder?path="+url.QueryEscape(t.TempDir()), &view)
	if !view.Exists || view.Git || view.Reason == "" {
		t.Fatalf("view = %+v", view)
	}
}

func TestARepositoryReportsBranchAndUncommittedFiles(t *testing.T) {
	dir := gitRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	var view folderView
	getJSONBody(t, "/onboarding/folder?path="+url.QueryEscape(dir), &view)
	if !view.Git || view.Branch != "main" || view.Dirty != 2 || view.Remote != "" {
		t.Fatalf("view = %+v", view)
	}
}

func TestAPathThatLooksLikeAFlagIsRefused(t *testing.T) {
	var view folderView
	getJSONBody(t, "/onboarding/folder?path="+url.QueryEscape("--upload-pack=x"), &view)
	if view.Exists || view.Git {
		t.Fatalf("view = %+v", view)
	}
}

func TestTheFolderListShowsDirectoriesAndMarksRepositories(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "api", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".hidden"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	var list folderList
	getJSONBody(t, "/onboarding/folders?path="+url.QueryEscape(root), &list)
	if len(list.Dirs) != 2 || list.Dirs[0].Name != "api" || !list.Dirs[0].Git || list.Dirs[1].Name != "docs" || list.Dirs[1].Git {
		t.Fatalf("list = %+v", list)
	}
	if list.Parent != filepath.Dir(root) || list.Path != root {
		t.Fatalf("list = %+v", list)
	}
}

func jiraStub(t *testing.T, status int) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"issues":[]}`))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func postJiraCheck(t *testing.T, form url.Values) jiraCheck {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/jira/check", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	Handler(t.TempDir(), "").ServeHTTP(rec, req)
	var body jiraCheck
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("%v (%s)", err, rec.Body.String())
	}
	return body
}

func TestJiraCheckSaysTheAccountWorks(t *testing.T) {
	got := postJiraCheck(t, url.Values{"project": {t.TempDir()}, "url": {jiraStub(t, http.StatusOK)}, "email": {"a@b.c"}, "token": {"t"}})
	if got.Status != "ok" {
		t.Fatalf("got %+v", got)
	}
}

func TestJiraCheckSaysTheTokenWasRejected(t *testing.T) {
	got := postJiraCheck(t, url.Values{"project": {t.TempDir()}, "url": {jiraStub(t, http.StatusUnauthorized)}, "email": {"a@b.c"}, "token": {"bad"}})
	if got.Status != "unauthorized" || got.Message == "" {
		t.Fatalf("got %+v", got)
	}
}

func TestJiraCheckSaysTheSiteDoesNotAnswer(t *testing.T) {
	got := postJiraCheck(t, url.Values{"project": {t.TempDir()}, "url": {"http://127.0.0.1:1"}, "email": {"a@b.c"}, "token": {"t"}})
	if got.Status != "unreachable" {
		t.Fatalf("got %+v", got)
	}
}

func TestJiraCheckNeedsASiteAndAToken(t *testing.T) {
	got := postJiraCheck(t, url.Values{"project": {t.TempDir()}, "url": {""}, "token": {""}})
	if got.Status != "invalid" {
		t.Fatalf("got %+v", got)
	}
}
