package live

import (
	"context"
	"testing"
)

func TestStopCancelsTheFlight(t *testing.T) {
	hub := New()
	ctx, cancel := context.WithCancel(context.Background())
	hub.WatchStop("s1", cancel)
	if !hub.Stop("s1") {
		t.Fatal("stop reported nothing running")
	}
	if ctx.Err() == nil {
		t.Fatal("the context was not cancelled")
	}
	if hub.Stop("s1") {
		t.Fatal("the same flight was stopped twice")
	}
}

func TestStopSaysNothingIsRunningWhenNothingIs(t *testing.T) {
	if New().Stop("absent") {
		t.Fatal("stopping an idle run reported success")
	}
}

func TestAFlightThatEndsForgetsHowToBeStopped(t *testing.T) {
	hub := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hub.WatchStop("s2", cancel)
	if !hub.Claim("s2", "build") {
		t.Fatal("claim failed")
	}
	hub.EndFlight("s2")
	if hub.Stop("s2") {
		t.Fatal("a finished flight was still stoppable")
	}
	if ctx.Err() != nil {
		t.Fatal("a finished flight was cancelled")
	}
}
