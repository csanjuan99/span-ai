package main

import (
	"bytes"
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"testing"

	"span/internal/handoff"
	"span/internal/launch"
	"span/internal/setup"
)

func repoDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitOut(t, dir, "init", "-q")
	return dir
}

// Attached files hang on a story. Both launch paths refuse them without one, with the same words.
func TestFilesWithoutAStoryAreRefusedBeforeAnythingStarts(t *testing.T) {
	dir := repoDir(t)
	req := setup.Request{Workspace: dir, Files: []handoff.File{{Name: "a.md", Body: []byte("x")}}}
	if _, err := resolveTargets(context.Background(), req); !errors.Is(err, errFilesNeedStory) {
		t.Fatalf("err %v", err)
	}
	out, code := runQuickstart(context.Background(), req)
	if code != 1 || !strings.Contains(out, "invalid: files need a story") {
		t.Fatalf("%d %q", code, out)
	}
}

func TestEachTargetGetsItsOwnPathStoryAndTheFolderItWasLaunchedFrom(t *testing.T) {
	folder := t.TempDir()
	repo := repoDir(t)
	got, err := setup.ForTarget(
		setup.Request{Workspace: folder, Story: "all", Issue: "X-1", Base: "dev"},
		launch.Target{Name: "api", Path: repo, Story: "its story", Issue: "X-2"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Workspace != repo || got.Story != "its story" || got.Issue != "X-2" {
		t.Fatalf("%+v", got)
	}
	if got.Folder != folder {
		t.Fatalf("folder %q: the repository must remember the folder it was launched from", got.Folder)
	}
}

// A run that was stopped ends on a cancelled context. That is not a failure to report.
func TestAStoppedRunIsNotLoggedAsAFailure(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)
	reportRun("r1", context.Canceled)
	reportRun("r1", nil)
	if buf.Len() != 0 {
		t.Fatalf("logged %q", buf.String())
	}
	reportRun("r2", errors.New("boom"))
	if !strings.Contains(buf.String(), "run r2: boom") {
		t.Fatalf("a real failure was dropped: %q", buf.String())
	}
}
