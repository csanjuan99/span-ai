package engine

import (
	"context"
	"errors"
	"testing"
	"time"
)

// settleReview refuses Merge as a conflict a set number of times, the way a host does while
// it still computes mergeability for a branch that was just pushed.
type settleReview struct {
	refuse int
	calls  int
	other  error
}

func (s *settleReview) Find(context.Context, string, string) ([]Change, error) { return nil, nil }
func (s *settleReview) Open(context.Context, string, string, string, string, string) (string, error) {
	return "", nil
}
func (s *settleReview) Merged(context.Context, string) (bool, error) { return false, nil }
func (s *settleReview) Merge(context.Context, string) error {
	s.calls++
	if s.calls <= s.refuse {
		return ErrConflict
	}
	return s.other
}

func settling(review Review) *runner {
	return &runner{opt: Options{Review: review, HostSettle: []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond}}, cur: Cursor{ChangeURL: "https://x/pull/1"}}
}

func TestAJustPushedResolutionWaitsForTheHostToCatchUp(t *testing.T) {
	review := &settleReview{refuse: 2}
	if err := settling(review).mergeSettling(context.Background(), true); err != nil {
		t.Fatalf("the merge was not retried until the host agreed: %v", err)
	}
	if review.calls != 3 {
		t.Fatalf("%d merge calls", review.calls)
	}
}

func TestAConflictThatNeverClearsIsStillReported(t *testing.T) {
	review := &settleReview{refuse: 99}
	err := settling(review).mergeSettling(context.Background(), true)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("a real conflict was hidden: %v", err)
	}
	if review.calls != 4 {
		t.Fatalf("%d merge calls, want the first plus three retries", review.calls)
	}
}

func TestWithoutAFreshPushAConflictIsNotRetried(t *testing.T) {
	review := &settleReview{refuse: 99}
	if err := settling(review).mergeSettling(context.Background(), false); !errors.Is(err, ErrConflict) {
		t.Fatalf("%v", err)
	}
	if review.calls != 1 {
		t.Fatalf("a base that moved was retried: %d calls", review.calls)
	}
}

func TestAnotherRefusalIsNotRetried(t *testing.T) {
	review := &settleReview{other: ErrBehind}
	if err := settling(review).mergeSettling(context.Background(), true); !errors.Is(err, ErrBehind) {
		t.Fatalf("%v", err)
	}
	if review.calls != 1 {
		t.Fatalf("%d calls", review.calls)
	}
}

func TestAStopWhileWaitingEndsTheWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	review := &settleReview{refuse: 99}
	run := settling(review)
	run.opt.HostSettle = []time.Duration{time.Hour}
	cancel()
	if err := run.mergeSettling(ctx, true); !errors.Is(err, ErrConflict) {
		t.Fatalf("%v", err)
	}
}
