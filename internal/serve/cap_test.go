package serve

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"span/internal/live"
	"span/internal/setup"
)

func TestARunRefusedForTheCapAnswersConflictWithTheReason(t *testing.T) {
	hub := live.New()
	hub.SetLimit(1)
	if !hub.Claim("busy-run", "build") {
		t.Fatal("claim failed")
	}
	launch := func(context.Context, setup.Request) (Outcome, error) { return Outcome{}, hub.Room() }
	h := New(t.TempDir(), "", launch, nil, hub, nil)
	rec := post(h, "/quickstart", url.Values{"story": {"AET-9 Anything"}, "path": {t.TempDir()}})
	if rec.Code != http.StatusConflict {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["error"] != "busy" || !strings.Contains(body["message"], "--max-parallel 1") {
		t.Fatalf("%v", body)
	}
}

func TestAWakeRefusedForTheCapAnswersConflict(t *testing.T) {
	root, id := journalRun(t)
	hub := live.New()
	hub.SetLimit(1)
	if !hub.Claim("other", "build") {
		t.Fatal("claim failed")
	}
	wake := func(context.Context, string, Signal) error { return hub.Admit(id, "merge") }
	h := New(root, "", nil, wake, hub, nil)
	rec := post(h, "/spans/"+id+"/resume", url.Values{})
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "--max-parallel 1") {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
}
