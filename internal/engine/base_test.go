package engine

import (
	"context"
	"strings"
	"testing"

	"span/internal/statement"
)

// knownBases is a source that knows which bases exist, as adapt.Git does. A base in gone
// exists for its first checks only, as a branch deleted on origin mid-run does.
type knownBases struct {
	*fakeSource
	bases  map[string]bool
	gone   map[string]int
	checks *int
}

func (k knownBases) BaseExists(_ context.Context, base string) (bool, error) {
	if k.checks != nil {
		*k.checks++
	}
	if left, ok := k.gone[base]; ok {
		k.gone[base] = left - 1
		return left > 0, nil
	}
	return k.bases[base], nil
}

func TestTheChangeIsOpenedIntoTheRunsBase(t *testing.T) {
	opt, _, _, review, _, _ := baseOpt(t)
	opt.Base = "feat/SAL-3-wallet"
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Advance || review.opens != 1 || review.openedBase != "feat/SAL-3-wallet" {
		t.Fatalf("%+v opens %d base %q", got, review.opens, review.openedBase)
	}
}

func TestTheCursorBaseWinsOverTheOptions(t *testing.T) {
	opt, _, _, review, _, _ := baseOpt(t)
	opt.Branch = "feat/SAL-4-cards"
	opt.Base = "feat/SAL-3-wallet"
	if _, err := Start(context.Background(), opt); err != nil {
		t.Fatal(err)
	}
	if review.openedBase != "feat/SAL-3-wallet" {
		t.Fatalf("opened into %q", review.openedBase)
	}
}

func TestNoBaseHoldsAtOpen(t *testing.T) {
	opt, _, _, review, _, _ := baseOpt(t)
	opt.Base = ""
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold || got.Stage != open || review.opens != 0 || !strings.Contains(got.Reason, "no base branch") {
		t.Fatalf("%+v opens %d", got, review.opens)
	}
}

func TestABaseThatIsTheCheckedOutBranchHoldsBeforeAnyWork(t *testing.T) {
	opt, agents, _, review, _, _ := baseOpt(t)
	opt.Base = "feat/payments" // what fakeSource.Branch reports
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold || got.Stage != intake || review.opens != 0 || len(agents.prompts) != 0 || !strings.Contains(got.Reason, "is the branch itself") {
		t.Fatalf("%+v opens %d prompts %d", got, review.opens, len(agents.prompts))
	}
}

const missingNowhere = "base branch feat/nowhere does not exist on origin; push it or choose another base"

func TestAMissingBaseHoldsBeforeTheBranchIsCut(t *testing.T) {
	opt, agents, source, review, _, _ := baseOpt(t)
	opt.Branch = "feat/SAL-4-cards"
	opt.Base = "feat/nowhere"
	opt.Source = knownBases{fakeSource: source, bases: map[string]bool{"dev": true}}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold || got.Stage != intake || got.Reason != missingNowhere || source.branch != "" || review.opens != 0 || len(agents.prompts) != 0 {
		t.Fatalf("%+v branch %q", got, source.branch)
	}
}

// span run names no branch: the base is still checked before the first stage, so a repo
// without that base fails fast instead of after the build.
func TestAMissingBaseHoldsBeforeTheFirstStageWithoutABranch(t *testing.T) {
	opt, agents, source, review, _, _ := baseOpt(t)
	opt.Base = "feat/nowhere"
	opt.Source = knownBases{fakeSource: source, bases: map[string]bool{"dev": true}}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold || got.Stage != intake || got.Reason != missingNowhere || review.opens != 0 || source.pushes != 0 || len(agents.prompts) != 0 {
		t.Fatalf("%+v opens %d pushes %d prompts %d", got, review.opens, source.pushes, len(agents.prompts))
	}
}

func TestABaseDeletedMidRunHoldsBeforeThePush(t *testing.T) {
	opt, _, source, review, _, _ := baseOpt(t)
	opt.Base = "feat/SAL-3"
	checks := 0
	opt.Source = knownBases{fakeSource: source, gone: map[string]int{"feat/SAL-3": 1}, checks: &checks}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	want := "base branch feat/SAL-3 does not exist on origin; push it or choose another base"
	if got.Verdict != statement.Hold || got.Stage != open || got.Reason != want || review.opens != 0 || source.pushes != 0 || checks != 2 {
		t.Fatalf("%+v opens %d pushes %d checks %d", got, review.opens, source.pushes, checks)
	}
}

func TestAnInvalidBaseInTheOptionsHoldsBeforeAnyGit(t *testing.T) {
	for _, base := range []string{"--upload-pack=touch PWNED1", "HEAD", "-x", "deadbeef"} {
		opt, _, source, review, _, _ := baseOpt(t)
		opt.Base = base
		checks := 0
		opt.Source = knownBases{fakeSource: source, bases: map[string]bool{base: true}, checks: &checks}
		got, err := Start(context.Background(), opt)
		if err != nil {
			t.Fatal(err)
		}
		if got.Verdict != statement.Hold || got.Stage != intake || !strings.Contains(got.Reason, "invalid base branch") || checks != 0 || review.opens != 0 {
			t.Fatalf("%q: %+v checks %d", base, got, checks)
		}
	}
}

func TestAnOpenChangeOnTheWrongBaseHolds(t *testing.T) {
	opt, _, source, review, _, _ := baseOpt(t)
	review.found = true
	review.url = "https://github.com/o/r/pull/7/files"
	review.base = "main"
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	want := "the open change https://github.com/o/r/pull/7/files targets main but this run's base is dev; retarget it (gh pr edit 7 --base dev) or close it, then resume"
	if got.Verdict != statement.Hold || got.Stage != open || got.Reason != want || review.opens != 0 || source.pushes != 0 {
		t.Fatalf("%+v opens %d", got, review.opens)
	}
}

func TestSeveralOpenChangesOnWrongBasesAreAllNamed(t *testing.T) {
	opt, _, _, review, _, _ := baseOpt(t)
	review.found = true
	review.url = "https://github.com/o/r/pull/7"
	review.base = "main"
	review.extra = []Change{{URL: "https://github.com/o/r/pull/8", Base: "release/2.1"}}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	want := "the open changes https://github.com/o/r/pull/7 (targets main), https://github.com/o/r/pull/8 (targets release/2.1) do not target this run's base dev; retarget one (gh pr edit 7 --base dev) and close the others, then resume"
	if got.Verdict != statement.Hold || got.Stage != open || got.Reason != want || review.opens != 0 {
		t.Fatalf("%+v", got)
	}
}

func TestAnyOpenChangeOnTheRightBaseIsReused(t *testing.T) {
	opt, _, source, review, _, _ := baseOpt(t)
	review.found = true
	review.url = "https://github.com/o/r/pull/7"
	review.base = "main"
	review.extra = []Change{{URL: "https://github.com/o/r/pull/8", Base: "dev"}}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Advance || review.opens != 0 || source.pushes != 0 {
		t.Fatalf("%+v opens %d", got, review.opens)
	}
	snap, err := Load(opt.Root, got.ID)
	if err != nil || snap.Cursor.ChangeURL != "https://github.com/o/r/pull/8" {
		t.Fatalf("change url %q, %v", snap.Cursor.ChangeURL, err)
	}
}

func TestAnOpenChangeOnTheRightBaseContinues(t *testing.T) {
	opt, _, _, review, _, _ := baseOpt(t)
	review.found = true
	review.url = "https://github.com/o/r/pull/7"
	review.base = "dev"
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Advance || review.opens != 0 {
		t.Fatalf("%+v opens %d", got, review.opens)
	}
}
