package adapt

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExecKeepsBothStreamsAndTheExitCode(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "boom")
	body := "#!/bin/sh\necho 'stdout line'\necho 'stderr line' >&2\nexit 3\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Exec{}.Run(context.Background(), script, dir)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got.ExitCode != 3 {
		t.Fatalf("exit = %d, want 3", got.ExitCode)
	}
	for _, want := range []string{"stdout line", "stderr line"} {
		if !strings.Contains(got.Text, want) {
			t.Fatalf("output is missing %q: %q", want, got.Text)
		}
	}
}

func TestExecReportsACommandThatDoesNotStart(t *testing.T) {
	dir := t.TempDir()
	if _, err := (Exec{}).Run(context.Background(), filepath.Join(dir, "absent"), dir); err == nil {
		t.Fatal("a missing program was reported as if it ran")
	}
}

func TestExecKeepsTheEndOfALongOutput(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "loud")
	body := "#!/bin/sh\nfor i in $(seq 1 20000); do echo \"line $i\"; done\nexit 1\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Exec{}.Run(context.Background(), script, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.Text, "line 20000") {
		t.Fatal("the end of the output was dropped")
	}
	if len(got.Text) > OutputCap {
		t.Fatalf("output kept %d bytes, cap is %d", len(got.Text), OutputCap)
	}
	if got.Truncated {
		t.Fatal("an output under the cap was marked truncated")
	}
}

func TestTailBufferSaysWhenItDroppedTheStart(t *testing.T) {
	b := &tailBuffer{max: 8}
	_, _ = b.Write([]byte("12345678"))
	if b.Truncated() {
		t.Fatal("an output at the cap was marked truncated")
	}
	_, _ = b.Write([]byte("9"))
	if !b.Truncated() || b.String() != "23456789" {
		t.Fatalf("truncated=%v text=%q", b.Truncated(), b.String())
	}
	// Past the compaction point, the flag stays set.
	_, _ = b.Write([]byte(strings.Repeat("x", 40)))
	if !b.Truncated() {
		t.Fatal("the flag was lost after compaction")
	}
}

func TestExecMarksAnOutputPastTheCap(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "flood")
	body := "#!/bin/sh\nhead -c 1100000 /dev/zero | tr '\\0' 'a'\necho\necho 'FAIL tail'\nexit 1\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Exec{}.Run(context.Background(), script, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Truncated || !strings.Contains(got.Text, "FAIL tail") {
		t.Fatalf("truncated=%v len=%d", got.Truncated, len(got.Text))
	}
}

func sleeper(t *testing.T) string {
	t.Helper()
	script := filepath.Join(t.TempDir(), "slow")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nsleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return script
}

// A cancelled check did not fail; it did not finish. Reporting it as exit -1 would send
// the run back to build and spend a repair on a Stop or Ctrl-C.
func TestACancelledCheckIsNotAFailedCheck(t *testing.T) {
	script := sleeper(t)
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	got, err := Exec{}.Run(ctx, script, t.TempDir())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v (exit %d), want the context error", err, got.ExitCode)
	}
}

func TestACancelledReleaseReturnsTheContextError(t *testing.T) {
	script := sleeper(t)
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	err := CommandRelease{}.Run(ctx, script, t.TempDir(), "abc123")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want the context error", err)
	}
}
