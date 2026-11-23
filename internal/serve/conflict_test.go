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
)

func conflictRun(t *testing.T, state string) (string, string) {
	t.Helper()
	root := t.TempDir()
	cur := engine.Cursor{ID: "span-c", Iteration: 1, Stage: "merge", Hold: true, Reason: engine.ConflictChooseReason}
	if state != "" {
		cur.Conflict = &engine.Conflict{Base: "origin/dev", State: state, Groups: []engine.ConflictGroup{
			{ID: "g1", Kind: "content", Files: []string{"a.go"}, Options: []engine.ConflictOption{{ID: "ours"}, {ID: "theirs"}}},
			{ID: "g2", Kind: "lockfile", Files: []string{"go.sum"}, Options: []engine.ConflictOption{{ID: "regen"}, {ID: "ours"}}},
		}}
	}
	dir := filepath.Join(root, cur.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(cur)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cursor.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return root, cur.ID
}

type wakes struct{ signals []Signal }

func (w *wakes) wake(_ context.Context, _ string, signal Signal) error {
	w.signals = append(w.signals, signal)
	return nil
}

func postForm(t *testing.T, h http.Handler, path string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestConflictChooseWritesTheChoiceAndWakesTheRun(t *testing.T) {
	root, id := conflictRun(t, engine.ConflictChoosing)
	woke := &wakes{}
	h := New(root, "", nil, woke.wake, nil, nil)
	rec := postForm(t, h, "/spans/"+id+"/conflict/choose", url.Values{"group": {"g1", "g2"}, "option": {"theirs", "regen"}})
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	if len(woke.signals) != 1 || !woke.signals[0].Resume {
		t.Fatalf("wakes %+v", woke.signals)
	}
	raw, err := os.ReadFile(filepath.Join(root, id, "conflict-choice.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	if err := json.Unmarshal(raw, &got); err != nil || got["g1"] != "theirs" || got["g2"] != "regen" {
		t.Fatalf("choice %s %v", raw, err)
	}
}

func TestConflictChooseRejectsAnIncompleteOrUnknownChoice(t *testing.T) {
	root, id := conflictRun(t, engine.ConflictChoosing)
	woke := &wakes{}
	h := New(root, "", nil, woke.wake, nil, nil)
	for _, form := range []url.Values{
		{"group": {"g1"}, "option": {"ours"}},
		{"group": {"g1", "g2"}, "option": {"ours"}},
		{"group": {"g1", "g2"}, "option": {"ours", "nope"}},
		{"group": {"g1", "g2", "g9"}, "option": {"ours", "ours", "ours"}},
		{"group": {"g1", "g1"}, "option": {"ours", "theirs"}},
		{},
	} {
		rec := postForm(t, h, "/spans/"+id+"/conflict/choose", form)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%v: status %d %s", form, rec.Code, rec.Body.String())
		}
	}
	if len(woke.signals) != 0 {
		t.Fatal("a rejected choice woke the run")
	}
	if _, err := os.Stat(filepath.Join(root, id, "conflict-choice.json")); err == nil {
		t.Fatal("a rejected choice was written")
	}
}

func TestConflictRoutesAnswer409WithoutAConflictInChoosing(t *testing.T) {
	for _, state := range []string{"", engine.ConflictApplying, engine.ConflictProposing} {
		root, id := conflictRun(t, state)
		woke := &wakes{}
		h := New(root, "", nil, woke.wake, nil, nil)
		for path, form := range map[string]url.Values{
			"choose": {"group": {"g1", "g2"}, "option": {"ours", "ours"}},
			"hint":   {"text": {"keep both"}},
			"abort":  {},
		} {
			rec := postForm(t, h, "/spans/"+id+"/conflict/"+path, form)
			if rec.Code != http.StatusConflict {
				t.Fatalf("state %q %s: status %d", state, path, rec.Code)
			}
		}
		if len(woke.signals) != 0 {
			t.Fatal("woke a run with no conflict to decide")
		}
	}
}

func TestConflictHintAndAbortWriteTheirFiles(t *testing.T) {
	root, id := conflictRun(t, engine.ConflictChoosing)
	woke := &wakes{}
	h := New(root, "", nil, woke.wake, nil, nil)
	if rec := postForm(t, h, "/spans/"+id+"/conflict/hint", url.Values{"text": {"  "}}); rec.Code != http.StatusBadRequest {
		t.Fatalf("empty hint: %d", rec.Code)
	}
	if rec := postForm(t, h, "/spans/"+id+"/conflict/hint", url.Values{"text": {"keep the base's name"}}); rec.Code != http.StatusAccepted {
		t.Fatalf("hint: %d %s", rec.Code, rec.Body.String())
	}
	raw, err := os.ReadFile(filepath.Join(root, id, "conflict-hint"))
	if err != nil || strings.TrimSpace(string(raw)) != "keep the base's name" {
		t.Fatalf("hint %q %v", raw, err)
	}
	if rec := postForm(t, h, "/spans/"+id+"/conflict/abort", nil); rec.Code != http.StatusAccepted {
		t.Fatalf("abort: %d", rec.Code)
	}
	if _, err := os.Stat(filepath.Join(root, id, "conflict-abort")); err != nil {
		t.Fatal(err)
	}
	if len(woke.signals) != 2 {
		t.Fatalf("wakes %d", len(woke.signals))
	}
	failed, failedID := conflictRun(t, engine.ConflictFailed)
	if rec := postForm(t, New(failed, "", nil, woke.wake, nil, nil), "/spans/"+failedID+"/conflict/abort", nil); rec.Code != http.StatusAccepted {
		t.Fatalf("abort of a failed conflict: %d", rec.Code)
	}
}

func TestConflictRoutesRejectAnUnsafeID(t *testing.T) {
	root, _ := conflictRun(t, engine.ConflictChoosing)
	h := New(root, "", nil, (&wakes{}).wake, nil, nil)
	if rec := postForm(t, h, "/spans/..%2Fx/conflict/abort", nil); rec.Code != http.StatusNotFound {
		t.Fatalf("status %d", rec.Code)
	}
}
