package live

import (
	"context"
	"testing"
	"time"
)

func TestAFlightStartedWithFlyCanBeStoppedAtOnce(t *testing.T) {
	hub := New()
	if !hub.Claim("s1", "intake") {
		t.Fatal("claim failed")
	}
	running := make(chan struct{})
	ended := make(chan error, 1)
	hub.Fly("s1", func(ctx context.Context) {
		close(running)
		<-ctx.Done()
		ended <- ctx.Err()
	})
	<-running
	if !hub.Stop("s1") {
		t.Fatal("a flight started with Fly was not stoppable")
	}
	select {
	case err := <-ended:
		if err == nil {
			t.Fatal("the flight ended without a cancelled context")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the flight ignored the stop")
	}
}

func TestStopRightAfterFlyNeverMissesTheFlight(t *testing.T) {
	for range 200 {
		hub := New()
		hub.Claim("s1", "intake")
		done := make(chan struct{})
		hub.Fly("s1", func(ctx context.Context) {
			defer close(done)
			<-ctx.Done()
		})
		if !hub.Stop("s1") {
			t.Fatal("stop raced the registration and found nothing running")
		}
		<-done
	}
}

func TestFlyReleasesTheClaimWhenTheWorkEnds(t *testing.T) {
	hub := New()
	hub.Claim("s1", "intake")
	done := make(chan struct{})
	hub.Fly("s1", func(context.Context) { close(done) })
	<-done
	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, busy := hub.Flight("s1"); !busy {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("the claim was not released")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestStopAllCancelsEveryFlightAndDrainWaitsForThem(t *testing.T) {
	hub := New()
	for _, id := range []string{"a", "b"} {
		hub.Claim(id, "build")
		hub.Fly(id, func(ctx context.Context) { <-ctx.Done() })
	}
	hub.StopAll()
	if !hub.Drain(2 * time.Second) {
		t.Fatal("flights did not drain after StopAll")
	}
	if len(hub.Flights()) != 0 {
		t.Fatalf("flights left: %v", hub.Flights())
	}
}

func TestDrainGivesUpAtTheDeadline(t *testing.T) {
	hub := New()
	hub.Claim("a", "build")
	if hub.Drain(30 * time.Millisecond) {
		t.Fatal("drain reported success with a flight still claimed")
	}
}

func TestNothingCanBeClaimedOnceTheHubIsClosedByStopAll(t *testing.T) {
	hub := New()
	hub.StopAll()
	if hub.Claim("late", "intake") {
		t.Fatal("a flight was admitted after shutdown began; it would outlive the process")
	}
}
