package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/engine"
	"span/internal/journal"
)

// branchRun builds a repository with two commits on a branch and a run root pointing at it.
func branchRun(t *testing.T) (root, id string) {
	t.Helper()
	project := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", append([]string{"-C", project, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(project, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("init", "-q", "-b", "main")
	write("README.md", "x\n")
	run("add", "-A")
	run("commit", "-q", "-m", "base")
	base := strings.TrimSpace(gitOut(t, project, "rev-parse", "HEAD"))
	write("retry.go", "a\nb\n")
	run("add", "-A")
	run("commit", "-q", "-m", "Add retry")
	write("README.md", "x\ny\n")
	run("commit", "-q", "-am", "Document retry")
	head := strings.TrimSpace(gitOut(t, project, "rev-parse", "HEAD"))
	root, id = t.TempDir(), "span-1"
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(engine.Cursor{ID: id, Workspace: project, Base: base, GitSHA: head, Feature: "retry"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cursor.json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	doc := "# Retry\n\n## Specs\nS1 Retry failed sends.\n\n## Log\n- Decision: Retry on 429 only because the story names no other code.\n"
	if err := os.MkdirAll(filepath.Join(project, "odd", "tasks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "odd", "tasks", "retry.md"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, id
}

func call(t *testing.T, root, method, url, contentType, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, url, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	Handler(root, "").ServeHTTP(rec, req)
	var out map[string]any
	// Only a JSON answer is decoded: the router's own 404 is plain text.
	if strings.HasPrefix(rec.Body.String(), "{") {
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("%s %s answered %d with a body that is not JSON: %v", method, url, rec.Code, err)
		}
	}
	return rec, out
}

func TestTheFilesRouteListsStatusAndTheViewedMark(t *testing.T) {
	root, id := branchRun(t)
	rec, body := call(t, root, http.MethodGet, "/spans/"+id+"/files", "", "")
	files, _ := body["files"].([]any)
	if rec.Code != http.StatusOK || len(files) != 2 {
		t.Fatalf("status %d body %v", rec.Code, body)
	}
	if rec, _ := call(t, root, http.MethodPost, "/spans/"+id+"/files/retry.go/viewed", "", ""); rec.Code != http.StatusOK {
		t.Fatalf("mark viewed: %d %s", rec.Code, rec.Body)
	}
	_, body = call(t, root, http.MethodGet, "/spans/"+id+"/files", "", "")
	marked := 0
	for _, f := range body["files"].([]any) {
		if f.(map[string]any)["viewed"] == true {
			marked++
		}
	}
	if marked != 1 {
		t.Fatalf("one file should be viewed: %v", body)
	}
	call(t, root, http.MethodPost, "/spans/"+id+"/files/retry.go/viewed?viewed=false", "", "")
	_, body = call(t, root, http.MethodGet, "/spans/"+id+"/files", "", "")
	for _, f := range body["files"].([]any) {
		if f.(map[string]any)["viewed"] == true {
			t.Fatalf("the mark should be gone: %v", body)
		}
	}
	if rec, body := call(t, root, http.MethodPost, "/spans/"+id+"/files/nothing.go/viewed", "", ""); rec.Code != http.StatusNotFound || body["error"] != "not_found" {
		t.Fatalf("a file outside the change: %d %v", rec.Code, body)
	}
}

func TestTheFileDiffRouteReadsOneFile(t *testing.T) {
	root, id := branchRun(t)
	rec, body := call(t, root, http.MethodGet, "/spans/"+id+"/files/retry.go/diff", "", "")
	if rec.Code != http.StatusOK || !strings.Contains(body["patch"].(string), "+a") || body["status"] != "A" {
		t.Fatalf("diff: %d %v", rec.Code, body)
	}
	if rec, _ := call(t, root, http.MethodGet, "/spans/"+id+"/files/missing.go/diff", "", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("missing file: %d", rec.Code)
	}
	if rec, _ := call(t, root, http.MethodGet, "/spans/"+id+"/files/retry.go", "", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("a path without /diff is not a route: %d", rec.Code)
	}
}

func TestTheCommitRoutesListThenShowOne(t *testing.T) {
	root, id := branchRun(t)
	_, body := call(t, root, http.MethodGet, "/spans/"+id+"/commits", "", "")
	commits := body["commits"].([]any)
	if len(commits) != 2 || commits[0].(map[string]any)["subject"] != "Document retry" {
		t.Fatalf("commits = %v", body)
	}
	first := commits[1].(map[string]any)["sha"].(string)
	rec, diff := call(t, root, http.MethodGet, "/spans/"+id+"/commits/"+first[:8]+"/diff", "", "")
	if rec.Code != http.StatusOK || !strings.Contains(diff["patch"].(string), "retry.go") || strings.Contains(diff["patch"].(string), "README") {
		t.Fatalf("commit diff: %d %v", rec.Code, diff)
	}
	if rec, _ := call(t, root, http.MethodGet, "/spans/"+id+"/commits/abcdef12/diff", "", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("a commit off the branch: %d", rec.Code)
	}
}

func TestReasoningCheckAcceptsJSONAndFormBodies(t *testing.T) {
	root, id := branchRun(t)
	_, body := call(t, root, http.MethodGet, "/spans/"+id+"/reasoning", "", "")
	if body["unchecked"].(float64) != 1 {
		t.Fatalf("reasoning = %v", body)
	}
	if rec, _ := call(t, root, http.MethodPost, "/spans/"+id+"/decisions/check", "application/json", `{"ids":[]}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("no ids: %d", rec.Code)
	}
	rec, body := call(t, root, http.MethodPost, "/spans/"+id+"/decisions/check", "application/x-www-form-urlencoded", "ids=D1")
	if rec.Code != http.StatusOK || body["unchecked"].(float64) != 0 {
		t.Fatalf("check: %d %v", rec.Code, body)
	}
	if rec, _ := call(t, root, http.MethodPost, "/spans/"+id+"/decisions/check", "application/json", `{"ids":["D9"]}`); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown decision: %d", rec.Code)
	}
}

func TestTheAuditRoutesVerifyTheChain(t *testing.T) {
	root, id := branchRun(t)
	dir := filepath.Join(root, id)
	for _, typ := range []string{"started", "sealed"} {
		if err := journal.Append(dir, journal.Span, typ, nil); err != nil {
			t.Fatal(err)
		}
	}
	_, audit := call(t, root, http.MethodGet, "/spans/"+id+"/audit", "", "")
	if audit["intact"] != true || len(audit["entries"].([]any)) != 2 {
		t.Fatalf("audit = %v", audit)
	}
	_, verdict := call(t, root, http.MethodGet, "/spans/"+id+"/audit/verify", "", "")
	if verdict["intact"] != true || verdict["entries"].(float64) != 2 {
		t.Fatalf("verify = %v", verdict)
	}
	path := filepath.Join(dir, journal.File)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), `"started"`, `"started!"`, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	_, verdict = call(t, root, http.MethodGet, "/spans/"+id+"/audit/verify", "", "")
	if verdict["intact"] != false || verdict["brokenAt"].(float64) != 2 {
		t.Fatalf("a broken chain: %v", verdict)
	}
}
