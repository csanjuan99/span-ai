package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/statement"
)

// otherRun records a run that works in the checkout of opt, on the branch it left there.
func otherRun(t *testing.T, opt Options, done bool) {
	t.Helper()
	dir := filepath.Join(opt.Root, "span-other")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cursor := `{"id":"span-other","branch":"feat/other","workspace":"` + opt.Workspace + `"`
	if done {
		cursor += `,"done":true`
	}
	if err := os.WriteFile(filepath.Join(dir, "cursor.json"), []byte(cursor+"}"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A run that waits for a person still owns what it left uncommitted in the checkout, so
// another run does not switch branches over it.
func TestAnInPlaceRunDoesNotSwitchOverAnotherRunsUncommittedWork(t *testing.T) {
	cases := map[string]struct {
		dirty, done, holds bool
	}{
		"uncommitted work of an unfinished run": {dirty: true, holds: true},
		"a clean checkout":                      {},
		"work left by a finished run":           {dirty: true, done: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			opt, _, source, _, _, _ := baseOpt(t)
			opt.Branch, opt.Base = "feat/mine", "dev"
			source.branch, source.dirty = "feat/other", tc.dirty
			otherRun(t, opt, tc.done)
			got, err := Start(context.Background(), opt)
			if err != nil {
				t.Fatal(err)
			}
			held := got.Verdict == statement.Hold && strings.Contains(got.Reason, "span-other")
			if held != tc.holds {
				t.Fatalf("%+v", got)
			}
			if tc.holds && source.branch != "feat/other" {
				t.Fatalf("the checkout was switched to %q", source.branch)
			}
		})
	}
}
