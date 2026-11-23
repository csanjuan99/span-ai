package serve

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"span/internal/setup"
)

const (
	// bodyWindow is how long an identical submission is treated as the same click. It only
	// has to outlast a double click or a retry on a slow connection; a person who stops a
	// run and sends the story again a minute later wants a new run.
	bodyWindow = 10 * time.Second
	// keyWindow is how long an explicit Idempotency-Key is remembered.
	keyWindow = 10 * time.Minute
)

// entry is one launch in flight or recently done. Later callers with the same key wait
// for it and get its answer instead of starting another run.
type entry struct {
	// fingerprint digests the body that started the launch, so a reused key with another
	// body can be told apart from a retry of the same one.
	fingerprint string
	done        chan struct{}
	out         Outcome
	err         error
	expires     time.Time
}

// recent remembers launches so a double submit starts one run. Two runs for one story
// in one checkout is the failure this exists to prevent.
type recent struct {
	mu    sync.Mutex
	items map[string]*entry
	now   func() time.Time
}

func newRecent() *recent {
	return &recent{items: map[string]*entry{}, now: time.Now}
}

// do runs fn once per key inside the window. A failure or a partial start is not
// remembered, so the person can fix the cause and send the story again. A caller that
// arrives while the first is still starting waits for it, but not past its own request.
func (c *recent) do(ctx context.Context, key, fingerprint string, window time.Duration, fn func() (Outcome, error)) (Outcome, error) {
	c.mu.Lock()
	for k, e := range c.items {
		if !e.expires.IsZero() && c.now().After(e.expires) {
			delete(c.items, k)
		}
	}
	if e, ok := c.items[key]; ok {
		c.mu.Unlock()
		if e.fingerprint != fingerprint {
			return Outcome{}, errKeyReused
		}
		select {
		case <-e.done:
			return e.out, e.err
		case <-ctx.Done():
			return Outcome{}, ctx.Err()
		}
	}
	e := &entry{fingerprint: fingerprint, done: make(chan struct{})}
	c.items[key] = e
	c.mu.Unlock()

	finished := false
	defer func() {
		c.mu.Lock()
		if !finished {
			// fn panicked. Waiters must not hang and the key must not stay stuck.
			e.err = errors.New("the launch did not finish")
		}
		if e.err != nil || e.out.Partial {
			delete(c.items, key)
		} else {
			e.expires = c.now().Add(window)
		}
		c.mu.Unlock()
		close(e.done)
	}()
	e.out, e.err = fn()
	finished = true
	return e.out, e.err
}

// keyReused is the answer to an Idempotency-Key that already started a different story.
// Replaying the old run would hand the caller a run for something they did not send.
type keyReused struct{}

func (keyReused) Error() string {
	return "this Idempotency-Key already started a different story; use a new key to start another run"
}

func (keyReused) SpanCode() string { return "idempotency_mismatch" }

var errKeyReused error = keyReused{}

// launchKey names a submission: the caller's Idempotency-Key when there is one, else a
// digest of what was submitted. The fingerprint is that digest either way.
func launchKey(r *http.Request, req setup.Request) (key string, window time.Duration, fingerprint string) {
	raw, err := json.Marshal(req)
	if err != nil {
		return "", 0, ""
	}
	sum := sha256.Sum256(raw)
	fingerprint = hex.EncodeToString(sum[:])
	if given := strings.TrimSpace(r.Header.Get("Idempotency-Key")); given != "" {
		return "key:" + given, keyWindow, fingerprint
	}
	return "body:" + fingerprint, bodyWindow, fingerprint
}
