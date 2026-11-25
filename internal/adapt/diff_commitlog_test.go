package adapt

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// commitMessage commits one file with the message the test names.
func commitMessage(t *testing.T, dir, name, msg string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "-A"}, {"commit", "-q", "-m", msg}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
}

func TestDiffListsTheCommitsOldestFirstWithTheirSubjects(t *testing.T) {
	dir := repo(t)
	commitAll(t, dir, "README.md", "base\n")
	base := headOf(t, dir)
	commitMessage(t, dir, "a.go", "feat: add the retry loop")
	commitMessage(t, dir, "b.go", "test: cover the retry loop")

	got, err := (Git{Dir: dir}).Diff(context.Background(), base, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.CommitLog) != 2 {
		t.Fatalf("log = %+v", got.CommitLog)
	}
	first, second := got.CommitLog[0], got.CommitLog[1]
	if first.Subject != "feat: add the retry loop" || second.Subject != "test: cover the retry loop" {
		t.Fatalf("subjects = %q, %q", first.Subject, second.Subject)
	}
	if len(first.SHA) != 40 || first.Time == "" {
		t.Fatalf("commit = %+v", first)
	}
	if got.Commits != len(got.CommitLog) {
		t.Fatalf("count %d disagrees with the list %d", got.Commits, len(got.CommitLog))
	}
}

func TestDiffListsNoCommitsWhenHeadIsTheBase(t *testing.T) {
	dir := repo(t)
	commitAll(t, dir, "README.md", "base\n")
	got, err := (Git{Dir: dir}).Diff(context.Background(), "HEAD", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if got.CommitLog == nil || len(got.CommitLog) != 0 {
		t.Fatalf("log = %#v, want an empty list so the page can tell none from unknown", got.CommitLog)
	}
}
