// Package live fans run events out to the page.
package live

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"span/internal/engine"
)

// ErrBusy means this process already has a worker for that span.
var ErrBusy = errors.New("an agent is already running")

// FullError means the process is already driving as many runs as --max-parallel allows.
// It reads as ErrBusy, so every place that maps a busy run to a conflict maps this too.
type FullError struct {
	Limit int
}

// Error says what is full, which flag sets it, and what to do about it.
func (e *FullError) Error() string {
	return fmt.Sprintf("span serve is at its limit of %d runs at once (--max-parallel %d): wait for a run to finish, or raise --max-parallel", e.Limit, e.Limit)
}

// Unwrap lets errors.Is(err, ErrBusy) hold.
func (e *FullError) Unwrap() error { return ErrBusy }

// SpanCode is the error code the HTTP layer answers with.
func (e *FullError) SpanCode() string { return "busy" }

// Flight is one engine worker owned by this process.
type Flight struct {
	Stage string
	Since time.Time
}

// Hub keeps one buffer per subscriber. A subscriber whose buffer is full is cut: its
// channel closes so the socket ends and the page reconnects and replays, instead of
// silently missing a frame it can never get back.
type Hub struct {
	mu      sync.Mutex
	subs    map[string]map[chan []byte]struct{}
	seqs    map[string]uint64
	flights map[string]Flight
	stops   map[string]context.CancelFunc
	closed  bool
	// limit is how many flights may be claimed at once. Zero is no limit.
	limit int
}

// New returns an empty hub.
func New() *Hub {
	return &Hub{subs: map[string]map[chan []byte]struct{}{}, seqs: map[string]uint64{}}
}

// WatchStop registers how to cancel the flight of one run. Whoever launched it owns the
// context; the hub only holds the handle.
func (h *Hub) WatchStop(id string, cancel context.CancelFunc) {
	if h == nil || cancel == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.stops == nil {
		h.stops = map[string]context.CancelFunc{}
	}
	h.stops[id] = cancel
}

// Fly runs work in a goroutine that Stop can cancel. The caller has already claimed id.
// The stop handle is registered before the goroutine starts, so a Stop that arrives
// the moment after Fly returns still finds the flight; registering it inside the
// goroutine would leave a window where the stop button reports "nothing is running".
// The claim is released when work returns.
func (h *Hub) Fly(id string, work func(context.Context)) {
	ctx, cancel := context.WithCancel(context.Background())
	h.WatchStop(id, cancel)
	go func() {
		defer h.EndFlight(id)
		defer cancel()
		work(ctx)
	}()
}

// StopAll cancels every flight in this process and closes the hub: Claim refuses from
// here on. Without that, a request that lands while the flights drain would start an
// agent that the exiting process abandons.
func (h *Hub) StopAll() {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.closed = true
	cancels := make([]context.CancelFunc, 0, len(h.stops))
	for id, cancel := range h.stops {
		cancels = append(cancels, cancel)
		delete(h.stops, id)
	}
	h.mu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
}

// Drain waits until no flight is claimed and reports whether that happened before the
// deadline. A cancelled flight stays claimed until its worker returns.
func (h *Hub) Drain(wait time.Duration) bool {
	deadline := time.Now().Add(wait)
	for {
		if len(h.Flights()) == 0 {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// Stop cancels the flight of one run and reports whether something was running.
func (h *Hub) Stop(id string) bool {
	if h == nil {
		return false
	}
	h.mu.Lock()
	cancel, ok := h.stops[id]
	delete(h.stops, id)
	h.mu.Unlock()
	if !ok {
		return false
	}
	cancel()
	return true
}

// Publish stamps the next seq of the run on a copy of event and sends it. The stamp
// and the sends share one lock so every subscriber sees the frames in seq order. A
// marshal failure does not fail the run; a full subscriber is cut, not skipped.
func (h *Hub) Publish(id string, event engine.Event) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.seqs == nil {
		h.seqs = map[string]uint64{}
	}
	h.seqs[id]++
	event.Seq = h.seqs[id]
	raw, err := json.Marshal(event)
	if err != nil {
		return
	}
	for ch := range h.subs[id] {
		select {
		case ch <- raw:
		default:
			delete(h.subs[id], ch)
			close(ch)
		}
	}
}

// Seq reports the last seq stamped for id, zero before the first Publish. The live
// socket reads it after subscribing so its replay names where the stream resumes.
func (h *Hub) Seq(id string) uint64 {
	if h == nil {
		return 0
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.seqs[id]
}

// SetLimit caps how many runs this process drives at once. Zero or less removes the cap.
func (h *Hub) SetLimit(limit int) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.limit = max(limit, 0)
}

// Room reports whether another run could be admitted now, as a *FullError when the cap is
// reached. A launch asks before it creates anything; Admit is still the atomic check.
func (h *Hub) Room() error {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.limit > 0 && len(h.flights) >= h.limit {
		return &FullError{Limit: h.limit}
	}
	return nil
}

// Admit reserves id for one worker and says why when it cannot: ErrBusy for a run that is
// already flying, a *FullError past the cap. A closed hub refuses with ErrBusy.
func (h *Hub) Admit(id, stage string) error {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return ErrBusy
	}
	if h.flights == nil {
		h.flights = map[string]Flight{}
	}
	if _, ok := h.flights[id]; ok {
		return ErrBusy
	}
	if h.limit > 0 && len(h.flights) >= h.limit {
		return &FullError{Limit: h.limit}
	}
	h.flights[id] = Flight{Stage: stage, Since: time.Now()}
	return nil
}

// Claim reserves id for one worker. A second claim fails until EndFlight.
func (h *Hub) Claim(id, stage string) bool {
	return h.Admit(id, stage) == nil
}

// EndFlight releases id. A nil hub does nothing.
func (h *Hub) EndFlight(id string) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.flights, id)
	delete(h.stops, id)
}

// Flight reports the worker reserved for id.
func (h *Hub) Flight(id string) (Flight, bool) {
	if h == nil {
		return Flight{}, false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	flight, ok := h.flights[id]
	return flight, ok
}

// Flights returns a copy of the workers reserved in this process.
func (h *Hub) Flights() map[string]Flight {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make(map[string]Flight, len(h.flights))
	for id, flight := range h.flights {
		out[id] = flight
	}
	return out
}

// Subscribe returns the frames for id and a function that detaches. The channel closes
// when the hub cuts a subscriber that fell behind; detaching after that is a no-op.
func (h *Hub) Subscribe(id string) (<-chan []byte, func()) {
	ch := make(chan []byte, 32)
	h.mu.Lock()
	if h.subs[id] == nil {
		h.subs[id] = map[chan []byte]struct{}{}
	}
	h.subs[id][ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.subs[id], ch)
		h.mu.Unlock()
	}
}
