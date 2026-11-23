package statement

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestTailKeepsTheEndOfTheOutput(t *testing.T) {
	long := strings.Repeat("a", TailBytes*2) + "\nFAIL src/cart.test.ts > totals\n"
	got := Tail(long)
	if !strings.Contains(got, "FAIL src/cart.test.ts > totals") {
		t.Fatalf("tail did not keep the end: %q", got[len(got)/2:])
	}
	if len(got) > TailBytes+8 {
		t.Fatalf("tail is %d bytes, want at most %d", len(got), TailBytes+8)
	}
}

func TestTailLeavesShortOutputAlone(t *testing.T) {
	if got := Tail("exit 1\n"); got != "exit 1\n" {
		t.Fatalf("got %q", got)
	}
}

func TestSignatureIgnoresOrderAndRepeat(t *testing.T) {
	one := Signature("AC-301", "AC-101", "npm test exited 1")
	two := Signature("npm test exited 1", "AC-101", "AC-301", "AC-101")
	if one != two {
		t.Fatalf("signatures differ: %q %q", one, two)
	}
	other := Signature("AC-101", "AC-301", "expected 2 to be 3")
	if one == other {
		t.Fatal("a different failure kept the same signature")
	}
}

func TestMarkdownShowsTheCommandOutput(t *testing.T) {
	dir := t.TempDir()
	s := Statement{
		SpanID: "s", Stage: "prove", Iteration: 2, Verdict: Hold, Reason: "npm test exited 1",
		Commands: []Command{{Command: "npm test", ExitCode: 1, Band: "fast",
			Output: "FAIL src/cart.test.ts > totals\n", LogPath: "iter/2/logs/prove-1.log"}},
		Time: time.Unix(0, 0).UTC(),
	}
	if err := Write(dir, &s); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "prove.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, want := range []string{"FAIL src/cart.test.ts > totals", "iter/2/logs/prove-1.log"} {
		if !strings.Contains(body, want) {
			t.Fatalf("report is missing %q:\n%s", want, body)
		}
	}
}

func oldShapeStatement() Statement {
	return Statement{SpanID: "s", Stage: "prove", Iteration: 1, Verdict: Hold, Reason: "go test ./... exited 1",
		Commands: []Command{{Command: "go test ./...", ExitCode: 1, Band: "fast", Output: "FAIL x", Fingerprint: "abc"}},
		Time:     time.Unix(0, 0).UTC()}
}

func TestADigestWithoutPreexistingIsUnchanged(t *testing.T) {
	// Pinned from the statement code before the preexisting field existed.
	s := oldShapeStatement()
	if err := s.Seal(); err != nil {
		t.Fatal(err)
	}
	if want := "fd7064c5c61b5fab67ad0444034e488e8026d0af83b9c1c051829dbd6014cf02"; s.Digest != want {
		t.Fatalf("digest = %s, want %s", s.Digest, want)
	}
}

func TestPreexistingRoundTripsAndIsShown(t *testing.T) {
	dir := t.TempDir()
	s := oldShapeStatement()
	s.Verdict = Advance
	s.Commands[0].Preexisting = true
	if err := Write(dir, &s); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "prove.json"))
	if err != nil || !strings.Contains(string(raw), `"preexisting": true`) {
		t.Fatalf("json %s err %v", raw, err)
	}
	got, ok, err := Read(dir, "prove")
	if err != nil || !ok || !got.Commands[0].Preexisting {
		t.Fatalf("ok=%v err=%v %+v", ok, err, got.Commands)
	}
	md, _ := os.ReadFile(filepath.Join(dir, "prove.md"))
	if !strings.Contains(string(md), "already failing on the base commit") {
		t.Fatalf("report %s", md)
	}
	plain := oldShapeStatement()
	if err := Write(t.TempDir(), &plain); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(mustJSON(t, plain)), "preexisting") {
		t.Fatal("a false preexisting was written")
	}
}

func mustJSON(t *testing.T, s Statement) []byte {
	t.Helper()
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
