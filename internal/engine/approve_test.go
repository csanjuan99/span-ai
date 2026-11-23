package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"span/internal/statement"
)

func heldAtMerge(t *testing.T) (Options, *fakeReview, string) {
	t.Helper()
	opt, _, _, review, _, _ := baseOpt(t)
	opt.Policy.Stages.Merge = "hold"
	review.found = true
	review.url = "https://example.test/pull/9"
	id := writeCursor(t, opt, Cursor{
		ID: "span-1", Iteration: 1, Stage: merge, GitSHA: "abc123", ChangeURL: review.url,
		Hold: true, Reason: "merge is hold",
	})
	return opt, review, id
}

func TestAHeldMergeStaysHeldUntilAPersonApprovesIt(t *testing.T) {
	opt, review, id := heldAtMerge(t)
	got, err := Resume(context.Background(), opt, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold || review.merges != 0 {
		t.Fatalf("a retry is not an approval: %+v merges %d", got, review.merges)
	}
}

func TestApprovingAHeldMergeLetsItMergeOnce(t *testing.T) {
	opt, review, id := heldAtMerge(t)
	if err := Approve(opt.Root, id, "merge", ""); err != nil {
		t.Fatal(err)
	}
	got, err := Resume(context.Background(), opt, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Advance || review.merges != 1 {
		t.Fatalf("approved merge: %+v merges %d", got, review.merges)
	}
}

func TestAnApprovalIsSpentWhenTheStageAdvances(t *testing.T) {
	opt, review, id := heldAtMerge(t)
	if err := Approve(opt.Root, id, "merge", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(opt.Root, id, "approve-merge")); err != nil {
		t.Fatalf("the approval is not on disk before the stage runs: %v", err)
	}
	got, err := Resume(context.Background(), opt, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Advance || review.merges != 1 {
		t.Fatalf("%+v merges %d", got, review.merges)
	}
	if _, err := os.Stat(filepath.Join(opt.Root, id, "approve-merge")); !os.IsNotExist(err) {
		t.Fatalf("a spent approval stays on disk: %v", err)
	}
}

func TestAnApprovalOnlyCoversTheHeadThePersonSaw(t *testing.T) {
	opt, review, id := heldAtMerge(t)
	if err := Approve(opt.Root, id, "merge", ""); err != nil {
		t.Fatal(err)
	}
	snap, err := Load(opt.Root, id)
	if err != nil {
		t.Fatal(err)
	}
	snap.Cursor.GitSHA = "moved-on"
	writeCursor(t, opt, snap.Cursor)
	got, err := Resume(context.Background(), opt, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold || review.merges != 0 {
		t.Fatalf("a new head merged on an old approval: %+v merges %d", got, review.merges)
	}
}

func TestOnlyAStageHeldByItsModeCanBeApproved(t *testing.T) {
	opt, _, id := heldAtMerge(t)
	for _, c := range []struct{ stage, env string }{
		{"build", ""}, {"open", ""}, {"merge/../x", ""}, {"promote", "../x"}, {"", ""},
	} {
		if err := Approve(opt.Root, id, c.stage, c.env); err == nil {
			t.Fatalf("approved %q %q", c.stage, c.env)
		}
	}
	if err := Approve(opt.Root, "missing", "merge", ""); err == nil {
		t.Fatal("approved a run that does not exist")
	}
}

func TestAnApprovalForOpenAndPromoteWorksToo(t *testing.T) {
	opt, _, _, review, release, _ := baseOpt(t)
	opt.Policy.Stages.Open = "hold"
	opt.Policy.Promote.Environments[0].Mode = "hold"
	id := writeCursor(t, opt, Cursor{ID: "span-2", Iteration: 1, Stage: open, GitSHA: "abc123", Hold: true, Reason: "open is hold"})
	if err := Approve(opt.Root, id, "open", ""); err != nil {
		t.Fatal(err)
	}
	got, err := Resume(context.Background(), opt, id)
	if err != nil {
		t.Fatal(err)
	}
	if review.opens != 1 {
		t.Fatalf("approved open: %+v opens %d", got, review.opens)
	}
	if got.Stage != promote || got.Verdict != statement.Hold || len(release.calls) != 0 {
		t.Fatalf("promote still waits for its own approval: %+v calls %#v", got, release.calls)
	}
}
