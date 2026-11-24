package blocker

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestFromCommit(t *testing.T) {
	first := errors.New("git commit -m feat(a): span run-1: exit status 1: \n\x1b[31mSTATUS: FAILED\x1b[0m\nModal.tsx:10 - rule - issue")
	same := errors.New("git commit -m feat(a): span run-2: exit status 1: \nstatus:   failed\nModal.tsx:10 - rule - issue")
	other := errors.New("git commit -m feat(a): span run-1: exit status 1: \nSTATUS: FAILED\nOther.tsx:1 - rule - issue")

	tests := []struct {
		name      string
		err       error
		prev      *Blocker
		wantCount int
		retryable bool
	}{
		{name: "first rejection", err: first, wantCount: 1, retryable: true},
		{name: "same output again, colors and run id aside", err: same, prev: FromCommit(first, nil), wantCount: 2, retryable: false},
		{name: "different output starts over", err: other, prev: FromCommit(first, nil), wantCount: 1, retryable: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromCommit(tt.err, tt.prev)
			if got.Class != CommitRejected || got.Count != tt.wantCount || got.Retryable() != tt.retryable {
				t.Fatalf("got %+v", got)
			}
			if strings.Contains(got.Evidence, "\x1b") || strings.Contains(got.Evidence, "run-1") {
				t.Fatalf("evidence kept noise: %q", got.Evidence)
			}
		})
	}
}

func TestANilBlockerIsRetryable(t *testing.T) {
	var b *Blocker
	if !b.Retryable() {
		t.Fatal("no blocker means the stage can run")
	}
}

func TestEvidenceKeepsTheTailWithinTheLimit(t *testing.T) {
	long := errors.New(strings.Repeat("noise line\n", 2000) + "the rule that failed")
	got := FromCommit(long, nil)
	if len(got.Evidence) > 4096 || !strings.HasSuffix(got.Evidence, "the rule that failed") {
		t.Fatalf("evidence = %d bytes, tail %q", len(got.Evidence), got.Evidence[max(0, len(got.Evidence)-30):])
	}
}

func TestOnlyARefusedCommitIsAHookRejection(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "the hook refused", err: errors.New("git commit -m feat: span x: exit status 1: STATUS: FAILED"), want: true},
		{name: "git add failed", err: errors.New("git add -A -- .: exit status 128: fatal: index.lock"), want: false},
		{name: "no identity", err: errors.New("git commit -m feat: Author identity unknown"), want: false},
		{name: "cancelled", err: context.Canceled, want: false},
		{name: "timed out", err: fmt.Errorf("git commit -m x: exit status 1: %w", context.DeadlineExceeded), want: false},
		{name: "nil", err: nil, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsHookRejection(tt.err); got != tt.want {
				t.Fatalf("IsHookRejection = %v", got)
			}
		})
	}
}
