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
)

func TestTheImplementationListsTheCommitsOfTheBranch(t *testing.T) {
	project := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", project, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	write := func(name string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(project, name), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("init", "-q", "-b", "main")
	write("README.md")
	run("add", "-A")
	run("commit", "-q", "-m", "base")
	base := strings.TrimSpace(gitOut(t, project, "rev-parse", "HEAD"))
	write("a.go")
	run("add", "-A")
	run("commit", "-q", "-m", "feat: add the retry loop")
	head := strings.TrimSpace(gitOut(t, project, "rev-parse", "HEAD"))

	root := t.TempDir()
	id := "span-1"
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(engine.Cursor{ID: id, Workspace: project, Base: base, GitSHA: head})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cursor.json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	Handler(root, "").ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/spans/"+id+"/implementation", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var body struct {
		Commits   int `json:"commits"`
		CommitLog []struct {
			SHA     string `json:"sha"`
			Subject string `json:"subject"`
		} `json:"commitLog"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Commits != 1 || len(body.CommitLog) != 1 || body.CommitLog[0].Subject != "feat: add the retry loop" || body.CommitLog[0].SHA != head {
		t.Fatalf("body = %+v", body)
	}
}
