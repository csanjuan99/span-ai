package serve

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/engine"
	"span/internal/live"
	"span/internal/runlock"
)

func TestRelaunchStopsTheFlightAndGivesEveryAgentStageTheChosenCLI(t *testing.T) {
	ready(t)
	root := t.TempDir()
	writeCursor(t, root, engine.Cursor{ID: "span-1", Stage: "build"})
	hub := live.New()
	if !hub.Claim("span-1", "build") {
		t.Fatal("claim")
	}
	stopped := make(chan struct{})
	hub.Fly("span-1", func(ctx context.Context) {
		<-ctx.Done()
		close(stopped)
	})
	woken := false
	handler := New(root, "", nil, func(context.Context, string, Signal) error {
		select {
		case <-stopped:
			woken = true
		default:
			t.Error("woken while the old flight still ran")
		}
		return nil
	}, hub, nil)
	form := url.Values{"agent": {"claude"}}
	req := httptest.NewRequest(http.MethodPost, "/spans/span-1/relaunch", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted || !woken {
		t.Fatalf("status %d woken %v %s", rec.Code, woken, rec.Body.String())
	}
	got := engine.ReadStageAgents(filepath.Join(root, "span-1"))
	for _, stage := range engine.AgentStages {
		if got[stage] != "claude" {
			t.Fatalf("agents = %v", got)
		}
	}
}

func TestARelaunchRefusedForLackOfRoomDoesNotStopTheRun(t *testing.T) {
	root := t.TempDir()
	writeCursor(t, root, engine.Cursor{ID: "span-1", Stage: "build"})
	// Another process drives span-1, and this server is already at its limit.
	held, err := runlock.Acquire(runlock.Run, filepath.Join(root, "span-1", "lock"), "run span-1")
	if err != nil {
		t.Fatal(err)
	}
	defer held.Release()
	hub := live.New()
	hub.SetLimit(1)
	if !hub.Claim("span-2", "build") {
		t.Fatal("claim")
	}
	handler := New(root, "", nil, func(context.Context, string, Signal) error {
		t.Error("woken past the limit")
		return nil
	}, hub, nil)
	req := httptest.NewRequest(http.MethodPost, "/spans/span-1/relaunch", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "span-1", "stop")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a refused relaunch asked the run to stop: %v", err)
	}
}
