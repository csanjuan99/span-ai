package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"span/internal/live"
)

func TestALaunchOverTheCapCreatesNothing(t *testing.T) {
	dir := gitRepo(t)
	spans := filepath.Join(t.TempDir(), "runs")
	started := stubRun(t)
	hub := live.New()
	hub.SetLimit(1)
	first := story("AET-1 Charge once")
	first.Workspace = dir
	if _, _, err := beginOne(first, spans, hub); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("the first run never flew")
	}
	before, err := os.ReadDir(spans)
	if err != nil {
		t.Fatal(err)
	}
	second := story("AET-2 Refund once")
	second.Workspace = dir
	_, _, err = beginOne(second, spans, hub)
	var full *live.FullError
	if !errors.As(err, &full) {
		t.Fatalf("a launch over the cap was not refused: %v", err)
	}
	after, err := os.ReadDir(spans)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Fatalf("a refused launch created a run: %d -> %d entries", len(before), len(after))
	}
	for id := range hub.Flights() {
		hub.Stop(id)
	}
}
