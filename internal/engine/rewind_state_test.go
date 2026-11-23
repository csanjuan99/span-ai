package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func rewound(t *testing.T, cur Cursor, to string) Cursor {
	t.Helper()
	root := t.TempDir()
	cur.ID = "2026-01-01-0000-rew2"
	dir := filepath.Join(root, cur.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeCursorFile(dir, cur); err != nil {
		t.Fatal(err)
	}
	if err := Rewind(root, cur.ID, to); err != nil {
		t.Fatal(err)
	}
	got, err := readCursorFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestRewindingBeforeOpenDropsTheChangeTheOldAttemptOpened(t *testing.T) {
	for _, to := range []string{build, proveSt, qaSt, inspect} {
		got := rewound(t, Cursor{
			Stage: merge, Iteration: 1, GitSHA: "abc", ChangeURL: "https://example.test/pull/9",
			Pending: &Pending{Kind: open, Branch: "feat/x", SHA: "abc"},
		}, to)
		if got.ChangeURL != "" || got.Pending != nil {
			t.Fatalf("restart at %s kept the old change: %+v", to, got)
		}
	}
}

func TestRewindingAtOrAfterOpenKeepsTheChange(t *testing.T) {
	for _, to := range []string{open, merge, promote} {
		got := rewound(t, Cursor{Stage: note, Iteration: 1, GitSHA: "abc", ChangeURL: "https://example.test/pull/9"}, to)
		if got.ChangeURL == "" {
			t.Fatalf("restart at %s lost the change url", to)
		}
	}
}

func TestARestartedRunReadsTheCommitTheBranchIsOnNow(t *testing.T) {
	opt, _, source, _, _, _ := baseOpt(t)
	source.sha = "after-hand-edit"
	id := writeCursor(t, opt, Cursor{ID: "span-1", Iteration: 1, Stage: proveSt, GitSHA: "before", Hold: true, Reason: "x"})
	if err := Rewind(opt.Root, id, proveSt); err != nil {
		t.Fatal(err)
	}
	if _, err := Resume(context.Background(), opt, id); err != nil {
		t.Fatal(err)
	}
	got, err := readCursorFile(filepath.Join(opt.Root, id))
	if err != nil {
		t.Fatal(err)
	}
	if got.GitSHA != "after-hand-edit" {
		t.Fatalf("the run kept the commit from before the restart: %q", got.GitSHA)
	}
	if got.Rewound {
		t.Fatal("the restart marker was not cleared")
	}
}
