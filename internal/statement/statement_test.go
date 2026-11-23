package statement

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSealChangesWhenReasonChanges(t *testing.T) {
	base := Statement{SpanID: "s", Stage: "frame", Iteration: 1, Verdict: Advance, Reason: "ready", Time: time.Unix(0, 0).UTC()}
	other := base
	other.Reason = "not ready"
	if err := base.Seal(); err != nil {
		t.Fatal(err)
	}
	if err := other.Seal(); err != nil {
		t.Fatal(err)
	}
	if base.Digest == "" || base.Digest == other.Digest {
		t.Fatalf("digests = %q %q", base.Digest, other.Digest)
	}
}

func TestWriteRoundTripKeepsParent(t *testing.T) {
	dir := t.TempDir()
	s := Statement{
		SpanID: "s", Stage: "prove", Iteration: 2, Verdict: Hold,
		ParentDigest: "abc", Reason: "command missing", Time: time.Unix(10, 0).UTC(),
	}
	if err := Write(dir, &s); err != nil {
		t.Fatal(err)
	}
	got, ok, err := Read(dir, "prove")
	if err != nil || !ok {
		t.Fatalf("read ok=%v err=%v", ok, err)
	}
	if got.ParentDigest != "abc" || got.Digest == "" {
		t.Fatalf("got %+v", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "prove.md")); err != nil {
		t.Fatal(err)
	}
}

func TestArchiveRemovesCurrentStage(t *testing.T) {
	dir := t.TempDir()
	s := Statement{SpanID: "s", Stage: "build", Iteration: 1, Verdict: Advance, Reason: "ok", Time: time.Unix(0, 0).UTC()}
	if err := Write(dir, &s); err != nil {
		t.Fatal(err)
	}
	if err := Archive(dir, "build", "block-1"); err != nil {
		t.Fatal(err)
	}
	_, ok, err := Read(dir, "build")
	if err != nil || ok {
		t.Fatalf("current still present ok=%v err=%v", ok, err)
	}
}
