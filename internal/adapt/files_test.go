package adapt

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// repoWith builds a repository with a base commit and a branch of two commits, and returns
// the git helper for it.
func repoWith(t *testing.T) Git {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	write := func(name, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("init", "-q", "-b", "main")
	write("event.go", strings.Repeat("line\n", 20))
	write("a.go", "one\n")
	run("add", ".")
	run("commit", "-q", "-m", "base")
	run("checkout", "-q", "-b", "feat")
	run("mv", "event.go", "payload.go")
	run("commit", "-q", "-am", "Rename event.go to payload.go")
	write("retry.go", "a\nb\nc\n")
	write("a.go", "one\ntwo\n")
	write("api.pb.go", "x\n")
	run("add", ".")
	run("commit", "-q", "-m", "Add retry")
	return Git{Dir: dir}
}

func TestFilesNamesStatusRenameAndGenerated(t *testing.T) {
	g := repoWith(t)
	files, err := g.Files(context.Background(), "main", "feat")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]FileInfo{}
	for _, f := range files {
		got[f.Path] = f
	}
	if r := got["payload.go"]; r.Status != "R" || r.From != "event.go" || r.Added+r.Removed != 0 {
		t.Fatalf("rename = %+v", r)
	}
	if a := got["retry.go"]; a.Status != "A" || a.Added != 3 {
		t.Fatalf("added = %+v", a)
	}
	if m := got["a.go"]; m.Status != "M" || m.Added != 1 {
		t.Fatalf("modified = %+v", m)
	}
	if !got["api.pb.go"].Generated {
		t.Fatalf("api.pb.go should be generated: %+v", got["api.pb.go"])
	}
}

func TestCommitsAreNewestFirstWithSizes(t *testing.T) {
	g := repoWith(t)
	commits, err := g.Commits(context.Background(), "main", "feat")
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 2 || commits[0].Subject != "Add retry" || commits[1].Subject != "Rename event.go to payload.go" {
		t.Fatalf("commits = %+v", commits)
	}
	if !commits[1].RenameOnly || commits[1].Files != 1 {
		t.Fatalf("rename commit = %+v", commits[1])
	}
	if commits[0].Files != 3 || commits[0].Added != 5 || commits[0].Parent != commits[1].SHA {
		t.Fatalf("second commit = %+v", commits[0])
	}
}

func TestCommitDiffShowsOnlyThatCommit(t *testing.T) {
	g := repoWith(t)
	commits, err := g.Commits(context.Background(), "main", "feat")
	if err != nil {
		t.Fatal(err)
	}
	res, err := g.CommitDiff(context.Background(), commits[0].SHA)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Patch, "retry.go") || strings.Contains(res.Patch, "payload.go") {
		t.Fatalf("patch should hold only the second commit:\n%s", res.Patch)
	}
	if res.Parent != commits[1].SHA || len(res.Files) != 3 {
		t.Fatalf("result = %+v", res)
	}
	if _, err := g.CommitDiff(context.Background(), "--output=x"); err == nil {
		t.Fatal("an option must be refused as a commit")
	}
}

func TestFileDiffRefusesPathsOutsideTheChange(t *testing.T) {
	g := repoWith(t)
	spec, err := Range("main", "feat")
	if err != nil {
		t.Fatal(err)
	}
	patch, _, err := g.FileDiff(context.Background(), spec, "a.go")
	if err != nil || !strings.Contains(patch, "+two") {
		t.Fatalf("patch = %q err = %v", patch, err)
	}
	for _, bad := range []string{"../x", "/etc/passwd", "-p", "a\nb"} {
		if _, _, err := g.FileDiff(context.Background(), spec, bad); err == nil {
			t.Fatalf("%q should be refused", bad)
		}
	}
}

func TestParseNumstatReadsRenamesAndBinary(t *testing.T) {
	raw := "3\t1\ta.go\x00-\t-\timg.png\x000\t0\t\x00old.go\x00new.go\x00"
	got := parseNumstat(raw)
	if len(got) != 3 || got[0].added != 3 || !got[1].binary || got[2].from != "old.go" || got[2].path != "new.go" {
		t.Fatalf("parsed = %+v", got)
	}
}
