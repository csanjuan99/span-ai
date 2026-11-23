package serve

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"span/internal/engine"
	"span/internal/live"
	"span/internal/qa"
	"span/internal/setup"
	"span/internal/statement"
	"span/internal/transcript"
)

func TestQuickstartReturnsTheSpanID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/quickstart", strings.NewReader("workspace=.&story=Charge"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	New(t.TempDir(), "", func(context.Context, setup.Request) (Outcome, error) {
		return Outcome{ID: "span-9", Stage: "intake"}, nil
	}, nil, nil, nil).ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	var body struct {
		ID    string `json:"id"`
		Stage string `json:"stage"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.ID != "span-9" || body.Stage != "intake" {
		t.Fatalf("%+v", body)
	}
}

func TestAcceptanceNamesTheCoveredDeclaration(t *testing.T) {
	root := t.TempDir()
	project := t.TempDir()
	source := "package pay\n\nfunc TestAC1_Charges(t *testing.T) {}\n"
	if err := os.WriteFile(filepath.Join(project, "pay_test.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	id := "span-1"
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cursor := engine.Cursor{ID: id, Stage: "qa", Workspace: project}
	raw, _ := json.Marshal(cursor)
	if err := os.WriteFile(filepath.Join(dir, "cursor.json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	brief := "Charge\n\nAcceptance criteria:\nCharges the card.\nKeeps a receipt.\n"
	if err := os.WriteFile(filepath.Join(dir, "brief.md"), []byte(brief), 0o644); err != nil {
		t.Fatal(err)
	}
	lines := qa.Lines(brief)
	gate := qa.Judge(lines, 0, map[string]bool{"AC-1": true}, qa.Outcomes{})
	gateRaw, _ := json.Marshal(gate)
	if err := os.WriteFile(filepath.Join(dir, "qa-gate.json"), append(gateRaw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/spans/"+id+"/acceptance", nil)
	rec := httptest.NewRecorder()
	Handler(root, "").ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	var body acceptanceBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Lines) != 2 || body.Lines[0].File != "pay_test.go" || body.Lines[0].Line != 3 || body.Lines[0].Result != "pass" {
		t.Fatalf("covered %+v", body.Lines)
	}
	if body.Lines[1].ID != "AC-2" || body.Lines[1].Result != "unverified" || body.Lines[1].File != "" {
		t.Fatalf("missing %+v", body.Lines[1])
	}
}

func TestLiveReplaysWithoutTheToken(t *testing.T) {
	root := t.TempDir()
	id := "span-1"
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(engine.Cursor{ID: id, Stage: "build"})
	if err := os.WriteFile(filepath.Join(dir, "cursor.json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	log, err := transcript.Open(filepath.Join(dir, "transcript.log"), []string{"secret-token"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := log.Write([]byte("saw secret-token\n")); err != nil {
		t.Fatal(err)
	}
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	hub := live.New()
	srv := httptest.NewServer(New(root, "", nil, nil, hub, nil))
	defer srv.Close()
	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/spans/"+id+"/live", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.CloseNow()
	var replay engine.Event
	if err := wsjson.Read(ctx, conn, &replay); err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(replay)
	if strings.Contains(string(encoded), "secret-token") || !strings.Contains(string(encoded), "[redacted]") {
		t.Fatalf("replay %s", encoded)
	}
	hub.Publish(id, engine.Event{Type: "statement", Statement: &statement.Statement{Stage: "frame", Reason: "sealed"}})
	var next engine.Event
	if err := wsjson.Read(ctx, conn, &next); err != nil {
		t.Fatal(err)
	}
	if next.Type != "statement" || next.Statement == nil || next.Statement.Reason != "sealed" {
		t.Fatalf("%+v", next)
	}
}

func TestLiveEchoesTheBrowserProtocolWhenNoTokenIsSet(t *testing.T) {
	root := t.TempDir()
	id := "span-1"
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(engine.Cursor{ID: id, Stage: "frame"})
	if err := os.WriteFile(filepath.Join(dir, "cursor.json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(New(root, "", nil, nil, live.New(), nil))
	defer srv.Close()
	conn, _, err := websocket.Dial(context.Background(), "ws"+strings.TrimPrefix(srv.URL, "http")+"/spans/"+id+"/live", &websocket.DialOptions{
		Subprotocols: []string{"span.dGVzdA"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.CloseNow()
	if conn.Subprotocol() != "span.dGVzdA" {
		t.Fatalf("protocol %q", conn.Subprotocol())
	}
}

func TestLiveStillRequiresTheConfiguredToken(t *testing.T) {
	root := t.TempDir()
	id := "span-1"
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(engine.Cursor{ID: id, Stage: "frame"})
	if err := os.WriteFile(filepath.Join(dir, "cursor.json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	const token = "secret"
	srv := httptest.NewServer(New(root, token, nil, nil, live.New(), nil))
	defer srv.Close()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/spans/" + id + "/live"
	wrong, _, err := websocket.Dial(context.Background(), url, &websocket.DialOptions{Subprotocols: []string{"span.dGVzdA"}})
	if err == nil {
		wrong.CloseNow()
		t.Fatal("a different protocol was accepted")
	}
	conn, _, err := websocket.Dial(context.Background(), url, &websocket.DialOptions{Subprotocols: []string{bearerProtocol(token)}})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.CloseNow()
	if conn.Subprotocol() != bearerProtocol(token) {
		t.Fatalf("protocol %q", conn.Subprotocol())
	}
}

func TestImplementationCapsThePatch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	project := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = project
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Span", "GIT_AUTHOR_EMAIL=span@example.test",
			"GIT_COMMITTER_NAME=Span", "GIT_COMMITTER_EMAIL=span@example.test",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	run("init", "-b", "master")
	if err := os.WriteFile(filepath.Join(project, "note.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "note.txt")
	run("commit", "-m", "base")
	base := strings.TrimSpace(gitOut(t, project, "rev-parse", "HEAD"))
	if err := os.WriteFile(filepath.Join(project, "big.txt"), []byte(strings.Repeat("a", 300*1024)), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "big.txt")
	run("commit", "-m", "head")
	head := strings.TrimSpace(gitOut(t, project, "rev-parse", "HEAD"))
	root := t.TempDir()
	id := "span-1"
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(engine.Cursor{ID: id, Workspace: project, Base: base, GitSHA: head})
	if err := os.WriteFile(filepath.Join(dir, "cursor.json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/spans/"+id+"/implementation", nil)
	rec := httptest.NewRecorder()
	Handler(root, "").ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var diff struct {
		Stat      string `json:"stat"`
		Truncated bool   `json:"truncated"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &diff); err != nil {
		t.Fatal(err)
	}
	if !diff.Truncated || !strings.Contains(diff.Stat, "big.txt") {
		t.Fatalf("stat %q truncated %v", diff.Stat, diff.Truncated)
	}
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}
