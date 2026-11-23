package serve

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/engine"
	"span/internal/journal"
)

func journalRun(t *testing.T) (root, id string) {
	t.Helper()
	root = t.TempDir()
	id = "span-1"
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(engine.Cursor{ID: id, Stage: "merge", Hold: true})
	if err := os.WriteFile(filepath.Join(dir, "cursor.json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, id
}

func post(h http.Handler, path string, form url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func wakeOK(context.Context, string, Signal) error { return nil }

func TestAPersonsApprovalIsJournaledWithItsStage(t *testing.T) {
	root, id := journalRun(t)
	h := New(root, "", nil, wakeOK, nil, nil)
	if rec := post(h, "/spans/"+id+"/approve", url.Values{"stage": {"merge"}}); rec.Code != http.StatusAccepted {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	events, err := journal.Read(filepath.Join(root, id))
	if err != nil || len(events) != 1 {
		t.Fatalf("%v %v", events, err)
	}
	if events[0].Actor != journal.Person || events[0].Type != "approve" || events[0].Detail["stage"] != "merge" {
		t.Fatalf("%+v", events[0])
	}
}

func TestARefusedActionLeavesNoLine(t *testing.T) {
	root, id := journalRun(t)
	h := New(root, "", nil, wakeOK, nil, nil)
	if rec := post(h, "/spans/"+id+"/approve", url.Values{"stage": {"not-a-stage"}}); rec.Code < 400 {
		t.Fatalf("expected a refusal, got %d", rec.Code)
	}
	events, err := journal.Read(filepath.Join(root, id))
	if err != nil || len(events) != 0 {
		t.Fatalf("a refused action was journaled: %v %v", events, err)
	}
}

func TestTextAPersonTypedIsJournaledAsItsLengthOnly(t *testing.T) {
	root, id := journalRun(t)
	h := New(root, "", nil, wakeOK, nil, nil)
	secret := "use password hunter2 for the db"
	post(h, "/spans/"+id+"/rewind", url.Values{"stage": {"build"}, "note": {secret}})
	events, err := journal.Read(filepath.Join(root, id))
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		for key, value := range event.Detail {
			if strings.Contains(value, "hunter2") || strings.Contains(key, "hunter2") {
				t.Fatalf("typed text reached the journal: %+v", event)
			}
		}
	}
}

func TestTheEventsEndpointReportsAnIntactChainAndABrokenOne(t *testing.T) {
	root, id := journalRun(t)
	dir := filepath.Join(root, id)
	for _, typ := range []string{"stop", "pause", "resume"} {
		if err := journal.Append(dir, journal.Person, typ, nil); err != nil {
			t.Fatal(err)
		}
	}
	read := func() eventsView {
		rec := httptest.NewRecorder()
		Handler(root, "").ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/spans/"+id+"/events", nil))
		var view eventsView
		if err := json.Unmarshal(rec.Body.Bytes(), &view); err != nil {
			t.Fatal(err)
		}
		return view
	}
	if view := read(); !view.Intact || len(view.Events) != 3 {
		t.Fatalf("%+v", view)
	}
	path := filepath.Join(dir, journal.File)
	raw, _ := os.ReadFile(path)
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), `"type":"pause"`, `"type":"stop"`, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if view := read(); view.Intact || view.BrokenAt != 3 {
		t.Fatalf("an edited line went unnoticed: %+v", view)
	}
}
