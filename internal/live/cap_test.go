package live

import (
	"errors"
	"strings"
	"testing"
)

func TestACappedHubRefusesTheClaimPastItsLimitAndFreesTheSlotOnRelease(t *testing.T) {
	hub := New()
	hub.SetLimit(2)
	for _, id := range []string{"a", "b"} {
		if err := hub.Admit(id, "intake"); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
	}
	err := hub.Admit("c", "intake")
	var full *FullError
	if !errors.As(err, &full) || full.Limit != 2 {
		t.Fatalf("a third run was admitted or the error is wrong: %v", err)
	}
	if !errors.Is(err, ErrBusy) {
		t.Fatal("a refused run must read as busy so the page answers 409")
	}
	if hub.Claim("c", "intake") {
		t.Fatal("Claim let a third run in")
	}
	if _, ok := hub.Flight("c"); ok {
		t.Fatal("a refused run left a flight behind")
	}
	hub.EndFlight("a")
	if err := hub.Admit("c", "intake"); err != nil {
		t.Fatalf("the slot was not freed: %v", err)
	}
}

func TestTheRefusalNamesTheLimitAndWhatToDo(t *testing.T) {
	hub := New()
	hub.SetLimit(1)
	if !hub.Claim("a", "build") {
		t.Fatal("claim failed")
	}
	err := hub.Room()
	if err == nil {
		t.Fatal("a full hub reported room")
	}
	for _, want := range []string{"--max-parallel 1", "wait", "raise"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("%q is missing from %q", want, err)
		}
	}
}

func TestWithoutALimitNothingIsRefused(t *testing.T) {
	hub := New()
	for _, id := range []string{"a", "b", "c", "d", "e"} {
		if err := hub.Admit(id, "intake"); err != nil {
			t.Fatal(err)
		}
	}
	if err := hub.Room(); err != nil {
		t.Fatal(err)
	}
}

func TestTheSameRunClaimedTwiceIsBusyNotFull(t *testing.T) {
	hub := New()
	hub.SetLimit(5)
	if err := hub.Admit("a", "intake"); err != nil {
		t.Fatal(err)
	}
	err := hub.Admit("a", "intake")
	var full *FullError
	if !errors.Is(err, ErrBusy) || errors.As(err, &full) {
		t.Fatalf("%v", err)
	}
}
