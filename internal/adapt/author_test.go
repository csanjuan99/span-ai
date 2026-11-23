package adapt

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// A commit made by Span belongs to the person the checkout is configured with.
func TestACommitKeepsTheConfiguredAuthor(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	git(t, dir, "config", "user.name", "Carlos Sanjuan")
	git(t, dir, "config", "user.email", "carlos@example.test")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("span\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "README.md")
	git(t, dir, "commit", "-q", "-m", "init")

	if err := os.WriteFile(filepath.Join(dir, "cart.ts"), []byte("export const x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := (Git{Dir: dir}).Commit(context.Background(), "feat(cart): span 2026-11-19-0253-87f0"); err != nil {
		t.Fatal(err)
	}

	author := git(t, dir, "log", "-1", "--format=%an <%ae>")
	if author != "Carlos Sanjuan <carlos@example.test>" {
		t.Fatalf("author = %q, want the configured user", author)
	}
	committer := git(t, dir, "log", "-1", "--format=%cn <%ce>")
	if committer != "Carlos Sanjuan <carlos@example.test>" {
		t.Fatalf("committer = %q, want the configured user", committer)
	}
	if strings.Contains(author+committer, "Span <span@example.test>") {
		t.Fatal("Span wrote itself as the author")
	}
}

func TestTheCommitMessageCarriesNoCoAuthor(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	git(t, dir, "config", "user.name", "Carlos Sanjuan")
	git(t, dir, "config", "user.email", "carlos@example.test")
	if err := os.WriteFile(filepath.Join(dir, "cart.ts"), []byte("export const x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := (Git{Dir: dir}).Commit(context.Background(), "feat(cart): span 2026-11-19-0253-87f0"); err != nil {
		t.Fatal(err)
	}
	body := git(t, dir, "log", "-1", "--format=%B")
	if strings.Contains(strings.ToLower(body), "co-authored-by") {
		t.Fatalf("the message carries a co-author: %q", body)
	}
	if !strings.Contains(body, "2026-11-19-0253-87f0") {
		t.Fatalf("the message lost the session it belongs to: %q", body)
	}
}
