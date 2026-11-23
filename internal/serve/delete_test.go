package serve

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/engine"
	"span/internal/live"
)

func deleteReq(handler http.Handler, id string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/spans/"+id+"/delete", nil))
	return rec
}

func TestDeletingARunRemovesItFromTheListAndLeavesTheOthers(t *testing.T) {
	root := t.TempDir()
	writeCursor(t, root, engine.Cursor{ID: "span-1", Stage: "frame"})
	writeCursor(t, root, engine.Cursor{ID: "span-2", Stage: "frame"})
	handler := New(root, "", nil, nil, live.New(), nil)
	rec := deleteReq(handler, "span-1")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"deleted":true`) {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "span-1")); !os.IsNotExist(err) {
		t.Fatal("the run is still there")
	}
	heads := engine.Heads(root)
	if len(heads) != 1 || heads[0].ID != "span-2" {
		t.Fatalf("heads = %+v", heads)
	}
}

func TestARunInFlightHereIsNotDeleted(t *testing.T) {
	root := t.TempDir()
	hub := live.New()
	if !hub.Claim("span-1", "frame") {
		t.Fatal("claim")
	}
	writeCursor(t, root, engine.Cursor{ID: "span-1", Stage: "frame"})
	rec := deleteReq(New(root, "", nil, nil, hub, nil), "span-1")
	if rec.Code != http.StatusConflict {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "span-1", "cursor.json")); err != nil {
		t.Fatal("a refused delete removed the run")
	}
}

func TestDeletingARunThatDoesNotExistIsRefused(t *testing.T) {
	rec := deleteReq(New(t.TempDir(), "", nil, nil, live.New(), nil), "nope")
	if rec.Code == http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}
