package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/engine"
	"span/internal/journal"
)

// implementationRun builds a repository with a two commit branch and a run root pointing at
// it, with a feature document holding one assumed and one settled decision.
func implementationRun(t *testing.T) (root, id string) {
	t.Helper()
	project := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", project, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	write := func(name, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(project, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(project, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git("init", "-q", "-b", "main")
	write("README.md", "x\n")
	git("add", "-A")
	git("commit", "-q", "-m", "base")
	base := git("rev-parse", "HEAD")
	write("retry.go", "a\nb\n")
	git("add", "-A")
	git("commit", "-q", "-m", "Add retry")
	head := git("rev-parse", "HEAD")
	write("odd/tasks/retry.md", "# Retry\n\n## Log\n- Decision: Retry on 429 only because the story names no code.\n- Decision: Five attempts because S3 says so. [source: S3]\n")
	root, id = t.TempDir(), "span-1"
	if err := os.MkdirAll(filepath.Join(root, id), 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(engine.Cursor{ID: id, Workspace: project, Base: base, GitSHA: head, Feature: "retry"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, id, "cursor.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return root, id
}

func runImplementation(root string, args ...string) (code int, out, errw string) {
	var o, e bytes.Buffer
	code = implementationMain(append(args, "--spans", root), &o, &e)
	return code, o.String(), e.String()
}

func TestImplementationFilesAndViewed(t *testing.T) {
	root, id := implementationRun(t)
	code, out, _ := runImplementation(root, id, "--files")
	if code != 0 || !strings.Contains(out, "retry.go") || !strings.Contains(out, "+2 -0") {
		t.Fatalf("files: %d %q", code, out)
	}
	if code, out, _ = runImplementation(root, id, "--viewed", "retry.go", "--json"); code != 0 || !strings.Contains(out, `"viewed": true`) {
		t.Fatalf("viewed: %d %q", code, out)
	}
	if _, out, _ = runImplementation(root, id, "--files"); !strings.Contains(out, "[viewed]") {
		t.Fatalf("the mark should show: %q", out)
	}
	if code, _, errw := runImplementation(root, id, "--viewed", "ghost.go"); code != exitNotFound || errw == "" {
		t.Fatalf("a file outside the change: %d %q", code, errw)
	}
}

func TestImplementationDiffAndCommits(t *testing.T) {
	root, id := implementationRun(t)
	code, out, _ := runImplementation(root, id, "--diff", "retry.go")
	if code != 0 || !strings.Contains(out, "+a") {
		t.Fatalf("diff: %d %q", code, out)
	}
	code, out, _ = runImplementation(root, id, "--commits", "--json")
	var body struct {
		Commits []struct{ SHA, Subject string } `json:"commits"`
	}
	if err := json.Unmarshal([]byte(out), &body); code != 0 || err != nil || len(body.Commits) != 1 || body.Commits[0].Subject != "Add retry" {
		t.Fatalf("commits: %d %q %v", code, out, err)
	}
	if code, out, _ = runImplementation(root, id, "--commit", body.Commits[0].SHA[:8]); code != 0 || !strings.Contains(out, "retry.go") {
		t.Fatalf("commit diff: %d %q", code, out)
	}
	if code, _, _ = runImplementation(root, id, "--commit", "deadbeef"); code != exitNotFound {
		t.Fatalf("unknown commit exits %d", code)
	}
}

func TestImplementationReasoningExitsThreeUntilAssumptionsAreChecked(t *testing.T) {
	root, id := implementationRun(t)
	code, out, errw := runImplementation(root, id, "--reasoning")
	if code != exitUnchecked || !strings.Contains(out, "D1") || !strings.Contains(errw, "1 assumed decisions are not checked") {
		t.Fatalf("reasoning: %d %q %q", code, out, errw)
	}
	if code, _, _ = runImplementation(root, id, "--check", "D2"); code != 1 {
		t.Fatalf("a settled decision cannot be checked: %d", code)
	}
	if code, _, _ = runImplementation(root, id, "--check", "D1"); code != 0 {
		t.Fatalf("check D1: %d", code)
	}
	if code, _, _ = runImplementation(root, id, "--reasoning"); code != 0 {
		t.Fatalf("after checking, reasoning exits %d", code)
	}
}

func TestImplementationAuditExitsTwelveWhenTheChainIsBroken(t *testing.T) {
	root, id := implementationRun(t)
	dir := filepath.Join(root, id)
	for _, typ := range []string{"started", "sealed", "held"} {
		if err := journal.Append(dir, journal.Span, typ, nil); err != nil {
			t.Fatal(err)
		}
	}
	code, out, _ := runImplementation(root, id, "--audit", "--verify")
	if code != 0 || !strings.Contains(out, "chain: intact, 3 of 3 entries") {
		t.Fatalf("intact: %d %q", code, out)
	}
	path := filepath.Join(dir, journal.File)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), `"sealed"`, `"sealed!"`, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ = runImplementation(root, id, "--audit")
	if code != exitChainBroken || !strings.Contains(out, "chain: broken at entry 3 of 3") {
		t.Fatalf("broken: %d %q", code, out)
	}
}

func TestImplementationUsageErrorsExitTwo(t *testing.T) {
	root, id := implementationRun(t)
	for _, args := range [][]string{{}, {id}, {id, "--files", "--commits"}, {id, "--check"}, {id, "--files", "stray"}} {
		if code, _, _ := runImplementation(root, args...); code != 2 {
			t.Fatalf("%v exits %d, want 2", args, code)
		}
	}
	if code, _, _ := runImplementation(root, "no-such-run", "--files"); code != exitNotFound {
		t.Fatalf("a run that does not exist exits %d", code)
	}
}
