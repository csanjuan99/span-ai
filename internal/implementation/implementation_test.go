package implementation

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/journal"
)

// fixture builds a repository with a branch of two commits and a run root whose cursor
// points at it, with a feature document in the workspace.
func fixture(t *testing.T, feature string) (root, id string) {
	t.Helper()
	ws := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = ws
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	write := func(name, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(ws, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(ws, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git("init", "-q", "-b", "main")
	write("a.go", "one\n")
	git("add", ".")
	git("commit", "-q", "-m", "base")
	git("checkout", "-q", "-b", "feat/webhook-retry")
	write("retry.go", "a\nb\n")
	git("add", ".")
	git("commit", "-q", "-m", "Add retry")
	write("a.go", "one\ntwo\n")
	git("commit", "-q", "-am", "Use retry")
	head := git("rev-parse", "HEAD")
	if feature != "" {
		write("odd/tasks/webhook-retry.md", feature)
	}
	root = t.TempDir()
	id = "r1"
	if err := os.MkdirAll(filepath.Join(root, id), 0o755); err != nil {
		t.Fatal(err)
	}
	cursor := map[string]any{"id": id, "workspace": ws, "base": "main", "gitSHA": head, "branch": "feat/webhook-retry", "feature": "webhook-retry"}
	raw, err := json.Marshal(cursor)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, id, "cursor.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return root, id
}

func TestChangedFilesCarryTheViewedMark(t *testing.T) {
	root, id := fixture(t, "")
	files, err := ChangedFiles(context.Background(), root, id)
	if err != nil || len(files) != 2 {
		t.Fatalf("files = %+v err = %v", files, err)
	}
	if err := SetViewed(context.Background(), root, id, "retry.go", true); err != nil {
		t.Fatal(err)
	}
	files, err = ChangedFiles(context.Background(), root, id)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if f.Viewed != (f.Path == "retry.go") {
			t.Fatalf("viewed marks = %+v", files)
		}
	}
	if err := SetViewed(context.Background(), root, id, "retry.go", false); err != nil {
		t.Fatal(err)
	}
	files, err = ChangedFiles(context.Background(), root, id)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if f.Viewed {
			t.Fatalf("unmarking left %s viewed", f.Path)
		}
	}
}

func TestSetViewedRefusesAFileOutsideTheChange(t *testing.T) {
	root, id := fixture(t, "")
	err := SetViewed(context.Background(), root, id, "missing.go", true)
	var request *RequestError
	if !errors.As(err, &request) || request.Code != "not_found" {
		t.Fatalf("err = %v, want not_found", err)
	}
	if _, err := ChangedFiles(context.Background(), root, "nope"); err == nil {
		t.Fatal("a run that does not exist must fail")
	}
}

func TestCommitDiffTakesAPrefixAndOnlyBranchCommits(t *testing.T) {
	root, id := fixture(t, "")
	commits, err := RunCommits(context.Background(), root, id)
	if err != nil || len(commits) != 2 || commits[0].Subject != "Use retry" {
		t.Fatalf("commits = %+v err = %v", commits, err)
	}
	res, err := RunCommitDiff(context.Background(), root, id, commits[1].SHA[:8])
	if err != nil || !strings.Contains(res.Patch, "retry.go") || strings.Contains(res.Patch, "+two") {
		t.Fatalf("diff of the first commit = %+v err = %v", res, err)
	}
	var request *RequestError
	if _, err := RunCommitDiff(context.Background(), root, id, "deadbeef"); !errors.As(err, &request) || request.Code != "not_found" {
		t.Fatalf("a commit off the branch: %v", err)
	}
	if _, err := RunCommitDiff(context.Background(), root, id, "ab"); !errors.As(err, &request) || request.Code != "invalid" {
		t.Fatalf("a too short sha: %v", err)
	}
}

func TestFileDiffWholeBranchOrOneCommit(t *testing.T) {
	root, id := fixture(t, "")
	whole, err := RunFileDiff(context.Background(), root, id, "a.go", "", false)
	if err != nil || !strings.Contains(whole.Patch, "+two") || whole.Hunks != 1 || whole.Status != "M" {
		t.Fatalf("whole = %+v err = %v", whole, err)
	}
	commits, err := RunCommits(context.Background(), root, id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RunFileDiff(context.Background(), root, id, "a.go", commits[1].SHA, false); err == nil {
		t.Fatal("a.go was not changed in the first commit")
	}
	one, err := RunFileDiff(context.Background(), root, id, "retry.go", commits[1].SHA, false)
	if err != nil || one.Commit != commits[1].SHA || one.Status != "A" {
		t.Fatalf("one = %+v err = %v", one, err)
	}
}

const feature = `# Retry failed webhook deliveries

## Specs
S1 A failed webhook delivery is retried automatically.
S2 Retries use exponential backoff: 1, 2, 4, 8 and 16 seconds.

## Tasks
- [x] Add the retry scheduler 7b02d9e
- [ ] Test timing

## Log
- Decision: Retry on 429, 502, 503 and 504 only because other 4xx codes will not succeed.
- Decision: Delays of 1, 2, 4, 8 and 16 seconds because S2 gives them. [source: S2; files: retry.go]
`

func TestReasoningSplitsDecisionsAndTracksChecks(t *testing.T) {
	root, id := fixture(t, feature)
	view, err := RunReasoning(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if view.Title != "Retry failed webhook deliveries" || len(view.Specs) != 2 || view.Specs[1].ID != "S2" || !strings.HasPrefix(view.Specs[1].Text, "Retries use") {
		t.Fatalf("specs = %+v", view)
	}
	if len(view.Tasks) != 2 || !view.Tasks[0].Done || view.Tasks[0].Commit != "7b02d9e" || view.Tasks[0].Text != "Add the retry scheduler" {
		t.Fatalf("tasks = %+v", view.Tasks)
	}
	d1, d2 := view.Decisions[0], view.Decisions[1]
	if !d1.Assumed || d1.Why != "other 4xx codes will not succeed." || d2.Assumed || d2.Source != "S2" || len(d2.Files) != 1 || d2.Files[0] != "retry.go" {
		t.Fatalf("decisions = %+v", view.Decisions)
	}
	if view.Unchecked != 1 || !view.Recorded {
		t.Fatalf("unchecked = %d recorded = %v", view.Unchecked, view.Recorded)
	}
	if err := CheckDecisions(root, id, []string{"D2"}); err == nil {
		t.Fatal("a decision the story settled has nothing to check")
	}
	if err := CheckDecisions(root, id, []string{"D9"}); err == nil {
		t.Fatal("an unknown decision must be refused")
	}
	if err := CheckDecisions(root, id, []string{"D1"}); err != nil {
		t.Fatal(err)
	}
	view, err = RunReasoning(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if view.Unchecked != 0 || !view.Decisions[0].Checked || view.Decisions[0].CheckedAt == "" {
		t.Fatalf("after check = %+v", view)
	}
	audit, err := RunAudit(root, id)
	if err != nil || len(audit.Entries) != 1 || audit.Entries[0].Actor != journal.Person || !strings.Contains(audit.Entries[0].What, "ids D1") {
		t.Fatalf("the check should be journaled: %+v err = %v", audit, err)
	}
}

func TestReasoningWithoutADocumentIsEmpty(t *testing.T) {
	root, id := fixture(t, "")
	view, err := RunReasoning(root, id)
	if err != nil || len(view.Decisions) != 0 || view.Recorded {
		t.Fatalf("view = %+v err = %v", view, err)
	}
}

func TestAuditNamesTheBrokenLink(t *testing.T) {
	root, id := fixture(t, "")
	dir := filepath.Join(root, id)
	for _, typ := range []string{"started", "sealed", "held"} {
		if err := journal.Append(dir, journal.Span, typ, map[string]string{"stage": "build"}); err != nil {
			t.Fatal(err)
		}
	}
	audit, err := RunAudit(root, id)
	if err != nil || !audit.Intact || len(audit.Entries) != 3 || audit.Head != audit.Entries[2].Hash {
		t.Fatalf("intact audit = %+v err = %v", audit, err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, journal.File))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, journal.File), []byte(strings.Replace(string(raw), `"sealed"`, `"sealed!"`, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	audit, err = RunAudit(root, id)
	if err != nil || audit.Intact || audit.BrokenAt != 3 {
		t.Fatalf("edited audit = %+v err = %v", audit, err)
	}
}
