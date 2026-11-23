package serve

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/engine"
	"span/internal/live"
	"span/internal/setup"
)

func TestPostWritesStageInstructions(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "span-1"), 0o755); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/spans/span-1/instructions/frame", strings.NewReader("Prefer the ledger."))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	Handler(root, "").ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d", rec.Code)
	}
	got, err := os.ReadFile(filepath.Join(root, "span-1", "instructions", "frame.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "Prefer the ledger." {
		t.Fatalf("body %q", got)
	}
}

func TestTokenIsRequiredWhenSet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	Handler(t.TempDir(), "secret").ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestNonLoopbackWithoutTokenRefuses(t *testing.T) {
	err := Listen("0.0.0.0:1", t.TempDir(), "", nil, nil, nil, nil)
	if err != ErrToken {
		t.Fatalf("err %v", err)
	}
}

func TestQuickstartFormWritesPolicyAndBranch(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "package.json"), []byte(`{"name":"app"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(project, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := strings.NewReader("workspace=" + project + "&kind=feat&story=Add+saved+cards")
	req := httptest.NewRequest(http.MethodPost, "/quickstart", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	Handler(t.TempDir(), "").ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "feat/add-saved-cards") {
		t.Fatalf("body %s", rec.Body.String())
	}
	raw, err := os.ReadFile(filepath.Join(project, "span.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "npm test") {
		t.Fatalf("policy %s", raw)
	}
}

func TestQuickstartAcceptsStoryFiles(t *testing.T) {
	project := t.TempDir()
	if err := os.Mkdir(filepath.Join(project, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	if err := form.WriteField("workspace", project); err != nil {
		t.Fatal(err)
	}
	if err := form.WriteField("story", "Add saved cards"); err != nil {
		t.Fatal(err)
	}
	part, err := form.CreateFormFile("file", "notes/board.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("png-bytes")); err != nil {
		t.Fatal(err)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/quickstart", &body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	rec := httptest.NewRecorder()
	var got setup.Request
	New(t.TempDir(), "", func(_ context.Context, req setup.Request) (Outcome, error) {
		got = req
		return Outcome{Text: "accepted\n"}, nil
	}, nil, nil, nil).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	if len(got.Files) != 1 || got.Files[0].Name != "board.png" || string(got.Files[0].Body) != "png-bytes" {
		t.Fatalf("%+v", got.Files)
	}
	if got.Story != "Add saved cards" {
		t.Fatalf("story %q", got.Story)
	}
}

func TestFolderOfReposRequiresASelection(t *testing.T) {
	parent := t.TempDir()
	for _, name := range []string{"salute-infra", "salute-apollo"} {
		if err := os.MkdirAll(filepath.Join(parent, name, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/repos?path="+parent, nil)
	rec := httptest.NewRecorder()
	Handler(t.TempDir(), "").ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "salute-infra") {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	body := strings.NewReader("workspace=" + parent + "&story=Add+saved+cards")
	post := httptest.NewRequest(http.MethodPost, "/quickstart", body)
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	got := httptest.NewRecorder()
	Handler(t.TempDir(), "").ServeHTTP(got, post)
	if got.Code != http.StatusBadRequest || !strings.Contains(got.Body.String(), "selection_required") {
		t.Fatalf("status %d %s", got.Code, got.Body.String())
	}
}

func TestJiraAccountResponseOmitsTheToken(t *testing.T) {
	project := t.TempDir()
	other := t.TempDir()
	body := strings.NewReader("project=" + url.QueryEscape(project) + "&url=https://example.atlassian.net&email=dev@example.com&token=secret-token")
	req := httptest.NewRequest(http.MethodPost, "/jira/account", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	Handler(t.TempDir(), "").ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "secret-token") {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "example.atlassian.net") {
		t.Fatalf("body %s", rec.Body.String())
	}
	otherBody := strings.NewReader("project=" + url.QueryEscape(other) + "&url=https://other.atlassian.net&email=other@example.com&token=other-token")
	otherReq := httptest.NewRequest(http.MethodPost, "/jira/account", otherBody)
	otherReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	otherRec := httptest.NewRecorder()
	Handler(t.TempDir(), "").ServeHTTP(otherRec, otherReq)
	if otherRec.Code != http.StatusOK {
		t.Fatalf("status %d %s", otherRec.Code, otherRec.Body.String())
	}
	out := httptest.NewRequest(http.MethodPost, "/jira/logout", strings.NewReader("project="+url.QueryEscape(project)))
	out.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	Handler(t.TempDir(), "").ServeHTTP(rec, out)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "secret-token") || !strings.Contains(rec.Body.String(), `"configured":false`) {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	still := httptest.NewRequest(http.MethodGet, "/jira?path="+url.QueryEscape(other), nil)
	rec = httptest.NewRecorder()
	Handler(t.TempDir(), "").ServeHTTP(rec, still)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"configured":true`) || strings.Contains(rec.Body.String(), "other-token") {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
}

func TestAnswerContinuesAWaitingRun(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "span-1")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(engine.Cursor{ID: "span-1", Stage: "frame", Hold: true, Reason: "open questions: Which currency?", Questions: []string{"Which currency?"}})
	if err := os.WriteFile(filepath.Join(dir, "cursor.json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	var got Signal
	handler := New(root, "", nil, func(_ context.Context, id string, signal Signal) error {
		if id != "span-1" {
			t.Fatalf("id %s", id)
		}
		got = signal
		return nil
	}, nil, nil)
	body := `{"answers":[{"question":"Which currency?","body":"USD"}]}`
	req := httptest.NewRequest(http.MethodPost, "/spans/span-1/answer", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted || len(got.Answers) != 1 || got.Answers[0].Body != "USD" {
		t.Fatalf("status %d signal %+v body %s", rec.Code, got, rec.Body.String())
	}
	stopped := httptest.NewRequest(http.MethodPost, "/spans/span-1/resume", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, stopped)
	if rec.Code != http.StatusConflict {
		t.Fatalf("resume %d %s", rec.Code, rec.Body.String())
	}
}

func writeCursor(t *testing.T, root string, cur engine.Cursor) {
	t.Helper()
	dir := filepath.Join(root, cur.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(cur)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cursor.json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRelaunchStartsARunThatLostItsAgent(t *testing.T) {
	root := t.TempDir()
	writeCursor(t, root, engine.Cursor{ID: "span-1", Stage: "frame", Reason: "agent reply has no JSON object"})
	var got Signal
	handler := New(root, "", nil, func(_ context.Context, id string, signal Signal) error {
		if id != "span-1" || !signal.Resume {
			t.Fatalf("id %s signal %+v", id, signal)
		}
		got = signal
		return nil
	}, live.New(), nil)
	req := httptest.NewRequest(http.MethodPost, "/spans/span-1/relaunch", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted || !got.Resume {
		t.Fatalf("status %d signal %+v body %s", rec.Code, got, rec.Body.String())
	}
	status := httptest.NewRequest(http.MethodGet, "/spans/span-1/agent", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, status)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"relaunch":true`) || !strings.Contains(rec.Body.String(), `"summary":"not running"`) {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
}

func TestRelaunchRefusesALiveStoppedOrWaitingRun(t *testing.T) {
	root := t.TempDir()
	hub := live.New()
	if !hub.Claim("span-1", "frame") {
		t.Fatal("claim")
	}
	writeCursor(t, root, engine.Cursor{ID: "span-1", Stage: "frame"})
	writeCursor(t, root, engine.Cursor{ID: "span-2", Stage: "frame", Stop: true, Reason: "out of scope"})
	writeCursor(t, root, engine.Cursor{ID: "span-3", Stage: "frame", Hold: true, Reason: "open questions: Which currency?", Questions: []string{"Which currency?"}})
	called := false
	handler := New(root, "", nil, func(context.Context, string, Signal) error {
		called = true
		return nil
	}, hub, nil)
	for _, id := range []string{"span-1", "span-2", "span-3"} {
		req := httptest.NewRequest(http.MethodPost, "/spans/"+id+"/relaunch", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusConflict || called {
			t.Fatalf("%s status %d called %v body %s", id, rec.Code, called, rec.Body.String())
		}
	}
	liveReq := httptest.NewRequest(http.MethodGet, "/spans/span-1/agent", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, liveReq)
	if !strings.Contains(rec.Body.String(), `"running":true`) || !strings.Contains(rec.Body.String(), `"summary":"working"`) {
		t.Fatalf("body %s", rec.Body.String())
	}
}
