package engine

import (
	"context"
	"strings"
	"testing"

	"span/internal/statement"
)

func atMerge(t *testing.T, review *fakeReview, opt Options, base string) string {
	t.Helper()
	review.found = true
	review.url = "https://example.test/pull/9"
	return writeCursor(t, opt, Cursor{
		ID: "span-1", Iteration: 1, Stage: merge, GitSHA: "abc123", ChangeURL: review.url, Base: base,
	})
}

func TestAChangeMergedIntoAnotherBaseHoldsTheRun(t *testing.T) {
	opt, _, _, review, _, _ := baseOpt(t)
	review.merged, review.base = true, "main"
	id := atMerge(t, review, opt, "dev")
	got, err := Resume(context.Background(), opt, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold || !strings.Contains(got.Reason, "merged into main, not into this run's base dev") {
		t.Fatalf("a wrong-base merge closed the run: %+v", got)
	}
}

func TestAnOpenChangeOnAnotherBaseIsNotMerged(t *testing.T) {
	opt, _, _, review, _, _ := baseOpt(t)
	review.base = "main"
	id := atMerge(t, review, opt, "dev")
	got, err := Resume(context.Background(), opt, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold || review.merges != 0 || !strings.Contains(got.Reason, "gh pr edit") {
		t.Fatalf("merged into the wrong base: %+v merges %d", got, review.merges)
	}
}

func TestApprovingTheOtherBaseLetsTheRunGoOn(t *testing.T) {
	opt, _, _, review, _, _ := baseOpt(t)
	review.merged, review.base = true, "main"
	id := atMerge(t, review, opt, "dev")
	if _, err := Resume(context.Background(), opt, id); err != nil {
		t.Fatal(err)
	}
	if err := Approve(opt.Root, id, "merge", ""); err != nil {
		t.Fatal(err)
	}
	got, err := Resume(context.Background(), opt, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Advance {
		t.Fatalf("an approval did not release it: %+v", got)
	}
}

func TestAChangeOnTheRunBaseMergesAsBefore(t *testing.T) {
	opt, _, _, review, _, _ := baseOpt(t)
	review.base = "dev"
	id := atMerge(t, review, opt, "dev")
	got, err := Resume(context.Background(), opt, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Advance || review.merges != 1 {
		t.Fatalf("%+v merges %d", got, review.merges)
	}
}
