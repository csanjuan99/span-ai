package serve

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"span/internal/runlock"
	"span/internal/setup"
)

func countingLaunch(calls *atomic.Int32) Launch {
	return func(context.Context, setup.Request) (Outcome, error) {
		n := calls.Add(1)
		return Outcome{ID: "span-" + string(rune('0'+n)), Stage: "intake"}, nil
	}
}

func submit(h http.Handler, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/quickstart", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestADoubleSubmitStartsOneRun(t *testing.T) {
	var calls atomic.Int32
	h := New(t.TempDir(), "", countingLaunch(&calls), nil, nil, nil)
	first := submit(h, "workspace=.&story=Charge", nil)
	second := submit(h, "workspace=.&story=Charge", nil)
	if calls.Load() != 1 {
		t.Fatalf("launched %d runs for one story", calls.Load())
	}
	if first.Body.String() != second.Body.String() || first.Code != http.StatusAccepted || second.Code != http.StatusAccepted {
		t.Fatalf("first %d %s / second %d %s", first.Code, first.Body, second.Code, second.Body)
	}
}

func TestADifferentStoryIsAnotherRun(t *testing.T) {
	var calls atomic.Int32
	h := New(t.TempDir(), "", countingLaunch(&calls), nil, nil, nil)
	submit(h, "workspace=.&story=Charge", nil)
	submit(h, "workspace=.&story=Refund", nil)
	if calls.Load() != 2 {
		t.Fatalf("launched %d, want 2", calls.Load())
	}
}

func TestAnIdempotencyKeyOutranksTheBody(t *testing.T) {
	var calls atomic.Int32
	h := New(t.TempDir(), "", countingLaunch(&calls), nil, nil, nil)
	key := map[string]string{"Idempotency-Key": "AET-1"}
	submit(h, "workspace=.&story=Charge", key)
	submit(h, "workspace=.&story=Charge+again", key)
	if calls.Load() != 1 {
		t.Fatalf("one key launched %d runs", calls.Load())
	}
	submit(h, "workspace=.&story=Charge", map[string]string{"Idempotency-Key": "AET-2"})
	if calls.Load() != 2 {
		t.Fatalf("a new key must start a run, launched %d", calls.Load())
	}
}

func TestAFailedLaunchCanBeRetried(t *testing.T) {
	var calls atomic.Int32
	h := New(t.TempDir(), "", func(context.Context, setup.Request) (Outcome, error) {
		if calls.Add(1) == 1 {
			return Outcome{}, errors.New("another run is active in this checkout")
		}
		return Outcome{ID: "span-1", Stage: "intake"}, nil
	}, nil, nil, nil)
	if rec := submit(h, "workspace=.&story=Charge", nil); rec.Code == http.StatusAccepted {
		t.Fatalf("a refused launch was accepted: %d", rec.Code)
	}
	if rec := submit(h, "workspace=.&story=Charge", nil); rec.Code != http.StatusAccepted {
		t.Fatalf("the retry was served the cached failure: %d %s", rec.Code, rec.Body)
	}
	if calls.Load() != 2 {
		t.Fatalf("launched %d", calls.Load())
	}
}

func TestParallelIdenticalSubmitsStartOneRun(t *testing.T) {
	var calls atomic.Int32
	release := make(chan struct{})
	h := New(t.TempDir(), "", func(context.Context, setup.Request) (Outcome, error) {
		calls.Add(1)
		<-release
		return Outcome{ID: "span-1", Stage: "intake"}, nil
	}, nil, nil, nil)
	var group sync.WaitGroup
	codes := make([]int, 6)
	for i := range codes {
		group.Add(1)
		go func() {
			defer group.Done()
			codes[i] = submit(h, "workspace=.&story=Charge", nil).Code
		}()
	}
	time.Sleep(100 * time.Millisecond)
	close(release)
	group.Wait()
	if calls.Load() != 1 {
		t.Fatalf("launched %d runs from simultaneous clicks", calls.Load())
	}
	for _, code := range codes {
		if code != http.StatusAccepted {
			t.Fatalf("codes %v", codes)
		}
	}
}

func TestEveryRunStartedIsReported(t *testing.T) {
	h := New(t.TempDir(), "", func(context.Context, setup.Request) (Outcome, error) {
		return Outcome{ID: "span-1", Stage: "intake", Runs: []RunRef{
			{ID: "span-1", Stage: "intake", Repo: "api"},
			{ID: "span-2", Stage: "intake", Repo: "web"},
		}}, nil
	}, nil, nil, nil)
	rec := submit(h, "workspace=.&story=Charge", nil)
	body := rec.Body.String()
	for _, want := range []string{`"id":"span-1"`, `"runs"`, `"span-2"`, `"repo":"web"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %s in %s", want, body)
		}
	}
}

func TestTheDedupeWindowExpires(t *testing.T) {
	now := time.Unix(1000, 0)
	cache := &recent{items: map[string]*entry{}, now: func() time.Time { return now }}
	calls := 0
	run := func() (Outcome, error) { calls++; return Outcome{ID: "x"}, nil }
	cache.do(context.Background(), "k", "", time.Second, run)
	cache.do(context.Background(), "k", "", time.Second, run)
	if calls != 1 {
		t.Fatalf("calls %d inside the window", calls)
	}
	now = now.Add(2 * time.Second)
	cache.do(context.Background(), "k", "", time.Second, run)
	if calls != 2 {
		t.Fatalf("calls %d after the window", calls)
	}
}

func TestAPanickingLaunchDoesNotHangTheDuplicates(t *testing.T) {
	cache := newRecent()
	started := make(chan struct{})
	release := make(chan struct{})
	go func() {
		defer func() { _ = recover() }()
		cache.do(context.Background(), "k", "", time.Minute, func() (Outcome, error) {
			close(started)
			<-release
			panic("boom")
		})
	}()
	<-started
	done := make(chan error, 1)
	go func() {
		_, err := cache.do(context.Background(), "k", "", time.Minute, func() (Outcome, error) { return Outcome{}, nil })
		done <- err
	}()
	time.Sleep(50 * time.Millisecond)
	close(release)
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("a duplicate of a panicked launch reported success")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the duplicate is still waiting on a launch that panicked")
	}
	if out, err := cache.do(context.Background(), "k", "", time.Minute, func() (Outcome, error) { return Outcome{ID: "again"}, nil }); err != nil || out.ID != "again" {
		t.Fatalf("the key stayed stuck after the panic: %+v %v", out, err)
	}
}

func TestAWaiterGivesUpWhenItsRequestIsCancelled(t *testing.T) {
	cache := newRecent()
	started := make(chan struct{})
	release := make(chan struct{})
	defer close(release)
	go cache.do(context.Background(), "k", "", time.Minute, func() (Outcome, error) {
		close(started)
		<-release
		return Outcome{ID: "x"}, nil
	})
	<-started
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := cache.do(ctx, "k", "", time.Minute, nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want the request context error", err)
	}
}

func TestAPartialStartIsNotRemembered(t *testing.T) {
	var calls atomic.Int32
	h := New(t.TempDir(), "", func(context.Context, setup.Request) (Outcome, error) {
		calls.Add(1)
		return Outcome{ID: "span-1", Stage: "intake", Partial: true, Text: "repo web did not start\n",
			Runs: []RunRef{{ID: "span-1", Stage: "intake", Repo: "api"}}}, nil
	}, nil, nil, nil)
	first := submit(h, "workspace=.&story=Charge", nil)
	submit(h, "workspace=.&story=Charge", nil)
	if calls.Load() != 2 {
		t.Fatalf("a partial start was served from the cache; launched %d", calls.Load())
	}
	if !strings.Contains(first.Body.String(), `"partial":true`) {
		t.Fatalf("the response does not say it is partial: %s", first.Body)
	}
}

func TestABusyCheckoutAnswersConflict(t *testing.T) {
	h := New(t.TempDir(), "", func(context.Context, setup.Request) (Outcome, error) {
		return Outcome{}, fmt.Errorf("refused: %w", &runlock.BusyError{Scope: runlock.Workspace, Holder: "pid 9 run span-3"})
	}, nil, nil, nil)
	rec := submit(h, "workspace=.&story=Charge", nil)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), `"busy"`) {
		t.Fatalf("status %d %s", rec.Code, rec.Body)
	}
}

func TestAKeyReusedWithAnotherStoryIsRefused(t *testing.T) {
	var calls atomic.Int32
	h := New(t.TempDir(), "", countingLaunch(&calls), nil, nil, nil)
	key := map[string]string{"Idempotency-Key": "AET-1"}
	first := submit(h, "workspace=.&story=Charge", key)
	second := submit(h, "workspace=.&story=Refund", key)
	if second.Code != http.StatusUnprocessableEntity || !strings.Contains(second.Body.String(), "idempotency_mismatch") {
		t.Fatalf("status %d %s", second.Code, second.Body)
	}
	if calls.Load() != 1 {
		t.Fatalf("launched %d", calls.Load())
	}
	again := submit(h, "workspace=.&story=Charge", key)
	if again.Code != http.StatusAccepted || again.Body.String() != first.Body.String() {
		t.Fatalf("the same body with the same key must still replay: %d %s", again.Code, again.Body)
	}
}
