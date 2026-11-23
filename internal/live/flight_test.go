package live

import (
	"sync"
	"testing"
)

func TestClaimAllowsOneFlight(t *testing.T) {
	hub := New()
	if !hub.Claim("span-1", "frame") {
		t.Fatal("first claim")
	}
	if hub.Claim("span-1", "frame") {
		t.Fatal("second claim")
	}
	hub.EndFlight("span-1")
	if !hub.Claim("span-1", "build") {
		t.Fatal("claim after the worker finished")
	}
	flight, ok := hub.Flight("span-1")
	if !ok || flight.Stage != "build" {
		t.Fatalf("flight %+v %v", flight, ok)
	}
}

func TestClaimSerializesParallelWorkers(t *testing.T) {
	hub := New()
	var won int
	var mu sync.Mutex
	var group sync.WaitGroup
	group.Add(8)
	for range 8 {
		go func() {
			defer group.Done()
			if hub.Claim("span-1", "frame") {
				mu.Lock()
				won++
				mu.Unlock()
			}
		}()
	}
	group.Wait()
	if won != 1 {
		t.Fatalf("winners %d", won)
	}
}
