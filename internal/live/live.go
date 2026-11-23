// Package live fans run events out to the page.
package live

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"span/internal/engine"
)

// ErrBusy means this process already has a worker for that span.
var ErrBusy = errors.New("an agent is already running")

// Flight is one engine worker owned by this process.
type Flight struct {
	Stage string
	Since time.Time
}

// Hub keeps one buffer per subscriber. A full buffer drops the frame.
type Hub struct {
	mu      sync.Mutex
	subs    map[string]map[chan []byte]struct{}
	flights map[string]Flight
}

// New returns an empty hub.
func New() *Hub {
	return &Hub{subs: map[string]map[chan []byte]struct{}{}}
}

// Publish sends one event. A marshal or send failure does not fail the run.
func (h *Hub) Publish(id string, event engine.Event) {
	if h == nil {
		return
	}
	raw, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[id] {
		select {
		case ch <- raw:
		default:
		}
	}
}

// Claim reserves id for one worker. A second claim fails until EndFlight.
func (h *Hub) Claim(id, stage string) bool {
	if h == nil {
		return true
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.flights == nil {
		h.flights = map[string]Flight{}
	}
	if _, ok := h.flights[id]; ok {
		return false
	}
	h.flights[id] = Flight{Stage: stage, Since: time.Now()}
	return true
}

// EndFlight releases id. A nil hub does nothing.
func (h *Hub) EndFlight(id string) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.flights, id)
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

// Subscribe returns the frames for id and a function that detaches.
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
