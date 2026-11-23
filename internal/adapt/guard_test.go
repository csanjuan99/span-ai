package adapt

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func tracked(t *testing.T, dir string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "ls-files").CombinedOutput()
	if err != nil {
		t.Fatalf("%v %s", err, out)
	}
	return string(out)
}

func staged(t *testing.T, dir string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "diff", "--cached", "--name-only").CombinedOutput()
	if err != nil {
		t.Fatalf("%v %s", err, out)
	}
	return strings.TrimSpace(string(out))
}

func guarded(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	return repo(t)
}

func TestCommitLeavesBuildOutputAndCachesUncommitted(t *testing.T) {
	dir := guarded(t)
	writeFiles(t, dir, map[string]string{
		"main.go":                  "package main\n",
		"tests/__pycache__/t.pyc":  "x",
		"web/node_modules/a/i.js":  "x",
		".DS_Store":                "x",
		"tests/test_greet.cpython": "x",
	})
	if _, err := (Git{Dir: dir}).Commit(context.Background(), "feat: x"); err != nil {
		t.Fatal(err)
	}
	got := tracked(t, dir)
	for _, junk := range []string{"__pycache__", "node_modules", ".DS_Store"} {
		if strings.Contains(got, junk) {
			t.Fatalf("%s was committed:\n%s", junk, got)
		}
	}
	if !strings.Contains(got, "main.go") || !strings.Contains(got, "tests/test_greet.cpython") {
		t.Fatalf("real work was dropped:\n%s", got)
	}
}

func TestCommitBlocksAnEnvFileAndStagesNothing(t *testing.T) {
	dir := guarded(t)
	writeFiles(t, dir, map[string]string{"main.go": "package main\n", ".env": "TOKEN=abc\n"})
	_, err := (Git{Dir: dir}).Commit(context.Background(), "feat: x")
	if err == nil || !strings.Contains(err.Error(), ".env") || !strings.Contains(err.Error(), "commit.allow") {
		t.Fatalf("the .env was not blocked with a way out: %v", err)
	}
	if staged(t, dir) != "" {
		t.Fatalf("a blocked commit left files staged: %q", staged(t, dir))
	}
}

func TestCommitAllowsAnExampleEnvFile(t *testing.T) {
	dir := guarded(t)
	writeFiles(t, dir, map[string]string{".env.example": "TOKEN=\n"})
	if _, err := (Git{Dir: dir}).Commit(context.Background(), "feat: x"); err != nil {
		t.Fatalf(".env.example is the documented way to share a template: %v", err)
	}
}

func TestCommitBlocksAPrivateKeyByContentWithoutPrintingIt(t *testing.T) {
	dir := guarded(t)
	body := "-----BEGIN RSA PRIVATE KEY-----\nSECRETBODY123\n-----END RSA PRIVATE KEY-----\n"
	writeFiles(t, dir, map[string]string{"deploy/notes.txt": body})
	_, err := (Git{Dir: dir}).Commit(context.Background(), "feat: x")
	if err == nil || !strings.Contains(err.Error(), "deploy/notes.txt") || !strings.Contains(err.Error(), "private key") {
		t.Fatalf("the key was not named: %v", err)
	}
	if strings.Contains(err.Error(), "SECRETBODY123") {
		t.Fatalf("the secret reached the error text: %v", err)
	}
	if staged(t, dir) != "" {
		t.Fatalf("left staged: %q", staged(t, dir))
	}
}

func TestCommitAllowListLetsAReviewedPathThrough(t *testing.T) {
	dir := guarded(t)
	writeFiles(t, dir, map[string]string{".env": "TOKEN=abc\n", "fixtures/key.pem": "-----BEGIN PRIVATE KEY-----\nx\n"})
	if _, err := (Git{Dir: dir}).Commit(context.Background(), "feat: x"); err == nil {
		t.Fatal("expected a block before the allow list")
	}
	allowed := Git{Dir: dir, Allow: []string{".env", "fixtures/"}}
	if _, err := allowed.Commit(context.Background(), "feat: x"); err != nil {
		t.Fatalf("the allow list did not let reviewed paths through: %v", err)
	}
	got := tracked(t, dir)
	if !strings.Contains(got, ".env") || !strings.Contains(got, "fixtures/key.pem") {
		t.Fatalf("tracked:\n%s", got)
	}
}
