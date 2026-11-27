package engine

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// headless is a source whose HEAD cannot be read.
type headless struct{ *fakeGit }

func (headless) Head(context.Context) (string, error) { return "", errors.New("no HEAD") }

func TestAnAbortThatCannotReadHeadKeepsTheConflict(t *testing.T) {
	rig := newConflictRig(t, conflictedApp())
	rig.agents.replies = []ConflictReply{combineReply("medium")}
	rig.resume(t)
	if err := AbortConflict(rig.opt.Root, rig.id); err != nil {
		t.Fatal(err)
	}
	rig.opt.Source = headless{rig.git}

	got := rig.resume(t)

	if !strings.Contains(got.Reason, "HEAD could not be read") || rig.cursor(t).Conflict == nil {
		t.Fatalf("%+v conflict %+v", got, rig.cursor(t).Conflict)
	}
}
