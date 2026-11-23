package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/agent"
	"span/internal/engine"
	"span/internal/live"
)

func getAgent(t *testing.T, project string) agentPick {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/agent?path="+url.QueryEscape(project), nil)
	rec := httptest.NewRecorder()
	Handler(t.TempDir(), "").ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	var body agentPick
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body
}

func postAgent(t *testing.T, project, id string) *httptest.ResponseRecorder {
	t.Helper()
	form := url.Values{"project": {project}, "agent": {id}}
	req := httptest.NewRequest(http.MethodPost, "/agent", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	Handler(t.TempDir(), "").ServeHTTP(rec, req)
	return rec
}

func TestTheAgentListDescribesEveryCLI(t *testing.T) {
	body := getAgent(t, t.TempDir())
	if len(body.Agents) != 4 {
		t.Fatalf("the page sees %d agents", len(body.Agents))
	}
	want := []string{"cursor", "claude", "codex", "pi"}
	for i, id := range want {
		if body.Agents[i].ID != id {
			t.Fatalf("position %d is %s, want %s", i, body.Agents[i].ID, id)
		}
		if body.Agents[i].Label == "" {
			t.Fatalf("%s has no label for the page", id)
		}
	}
}

func TestAProjectWithoutPolicySaysSo(t *testing.T) {
	body := getAgent(t, t.TempDir())
	if body.Exists {
		t.Fatal("a folder without span.yaml reported a policy")
	}
	if !strings.Contains(body.Reason, "span.yaml") {
		t.Fatalf("reason = %q", body.Reason)
	}
}

func TestChoosingAnAgentWritesThePolicyAndReads(t *testing.T) {
	project := t.TempDir()
	path := filepath.Join(project, "span.yaml")
	if err := os.WriteFile(path, []byte("feature: payments\nagent: any\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := postAgent(t, project, "pi")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	var body agentPick
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Selected != "pi" {
		t.Fatalf("selected = %q", body.Selected)
	}
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), "agent: pi") {
		t.Fatalf("the policy was not written:\n%s", raw)
	}
}

func TestChoosingAnUnknownAgentIsRefused(t *testing.T) {
	project := t.TempDir()
	path := filepath.Join(project, "span.yaml")
	if err := os.WriteFile(path, []byte("feature: payments\nagent: any\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := postAgent(t, project, "gpt")
	if rec.Code < 400 {
		t.Fatalf("status %d, want a refusal", rec.Code)
	}
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), "agent: any") {
		t.Fatalf("the policy changed:\n%s", raw)
	}
}

// span.yaml is optional, so choosing an agent in a project without one creates it with
// that one line. A project path that is not a folder is refused, so nothing is written next
// to the server.
func TestChoosingAnAgentWithoutAPolicyCreatesIt(t *testing.T) {
	project := t.TempDir()
	rec := postAgent(t, project, "pi")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	raw, err := os.ReadFile(filepath.Join(project, "span.yaml"))
	if err != nil || string(raw) != "agent: pi\n" {
		t.Fatalf("span.yaml %q %v", raw, err)
	}
	for _, bad := range []string{"", filepath.Join(project, "missing")} {
		if rec := postAgent(t, bad, "pi"); rec.Code < 400 {
			t.Fatalf("project %q: status %d, want a refusal", bad, rec.Code)
		}
	}
	if _, err := os.Stat(filepath.Join(project, "missing")); !os.IsNotExist(err) {
		t.Fatalf("a missing project was created: %v", err)
	}
}

// piHomeForTests makes Pi the only ready CLI without touching the real home.
func piHomeForTests(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".pi", "agent", "npm", "node_modules", "gentle-pi"), 0o755); err != nil {
		t.Fatal(err)
	}
	old := agent.Home
	agent.Home = func() string { return dir }
	t.Cleanup(func() { agent.Home = old })
}

// spanOnDisk writes the cursor a handler needs before it can answer.
func spanOnDisk(t *testing.T, root, id, project string) {
	t.Helper()
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(engine.Cursor{ID: id, Stage: "prove", Iteration: 2, Hold: true, Workspace: project})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cursor.json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestChoosingAnAgentForAStageWritesTheRunAndReadsBack(t *testing.T) {
	piHomeForTests(t)
	root := t.TempDir()
	id := "2026-01-01-0000-aaaa"
	spanOnDisk(t, root, id, t.TempDir())

	form := url.Values{"stage": {"build"}, "agent": {"pi"}}
	req := httptest.NewRequest(http.MethodPost, "/spans/"+id+"/agent", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	Handler(root, "").ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	var snap engine.Snapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &snap); err != nil {
		t.Fatal(err)
	}
	if snap.StageAgents["build"] != "pi" {
		t.Fatalf("stageAgents = %v", snap.StageAgents)
	}
}

func TestChoosingAnAgentForAStageWithoutAModelIsRefused(t *testing.T) {
	piHomeForTests(t)
	root := t.TempDir()
	id := "2026-01-01-0000-bbbb"
	spanOnDisk(t, root, id, t.TempDir())
	form := url.Values{"stage": {"prove"}, "agent": {"pi"}}
	req := httptest.NewRequest(http.MethodPost, "/spans/"+id+"/agent", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	Handler(root, "").ServeHTTP(rec, req)
	if rec.Code < 400 {
		t.Fatalf("status %d, want a refusal", rec.Code)
	}
	if _, err := os.Stat(filepath.Join(root, id, "agents.json")); !os.IsNotExist(err) {
		t.Fatal("a refused choice was written")
	}
}

func TestPausingARunWritesTheRequest(t *testing.T) {
	root := t.TempDir()
	id := "2026-01-01-0000-cccc"
	spanOnDisk(t, root, id, t.TempDir())
	req := httptest.NewRequest(http.MethodPost, "/spans/"+id+"/pause", nil)
	rec := httptest.NewRecorder()
	Handler(root, "").ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, id, "pause")); err != nil {
		t.Fatal("the pause request was not written")
	}
}

func TestRestartingFromAStageMovesTheCursorAndKeepsTheNote(t *testing.T) {
	root := t.TempDir()
	id := "2026-01-01-0000-rew1"
	spanOnDisk(t, root, id, t.TempDir())
	before, err := engine.Load(root, id)
	if err != nil {
		t.Fatal(err)
	}
	form := url.Values{
		"stage": {"build"},
		"note":  {"Use the handoff code, do not rewrite the module."},
	}
	req := httptest.NewRequest(http.MethodPost, "/spans/"+id+"/rewind", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	Handler(root, "").ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	var snap engine.Snapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &snap); err != nil {
		t.Fatal(err)
	}
	if snap.Cursor.Stage != "build" {
		t.Fatalf("stage = %q", snap.Cursor.Stage)
	}
	if snap.Cursor.Iteration != before.Cursor.Iteration+1 {
		t.Fatalf("iteration = %d, want %d so the earlier attempt stays on disk", snap.Cursor.Iteration, before.Cursor.Iteration+1)
	}
	if snap.Cursor.Hold {
		t.Fatal("the restarted run is still held")
	}
	note, err := os.ReadFile(filepath.Join(root, id, "instructions", "build.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(note), "handoff code") {
		t.Fatalf("note = %q", note)
	}
}

func TestRestartingFromAStageItCannotStartFromIsRefused(t *testing.T) {
	root := t.TempDir()
	id := "2026-01-01-0000-rew2"
	spanOnDisk(t, root, id, t.TempDir())
	form := url.Values{"stage": {"intake"}}
	req := httptest.NewRequest(http.MethodPost, "/spans/"+id+"/rewind", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	Handler(root, "").ServeHTTP(rec, req)
	if rec.Code < 400 {
		t.Fatalf("status %d, want a refusal", rec.Code)
	}
	snap, err := engine.Load(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Cursor.Stage == "intake" {
		t.Fatal("the cursor moved to a stage that cannot start")
	}
}

func TestStoppingSomethingThatIsNotRunningSaysSo(t *testing.T) {
	root := t.TempDir()
	id := "2026-01-01-0000-stop1"
	spanOnDisk(t, root, id, t.TempDir())
	req := httptest.NewRequest(http.MethodPost, "/spans/"+id+"/stop", nil)
	rec := httptest.NewRecorder()
	Handler(root, "").ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want a neutral answer", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "nothing is running") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestPausingWithNothingInFlightDoesNotQueueAFile(t *testing.T) {
	root := t.TempDir()
	id := "2026-01-01-0000-pause1"
	spanOnDisk(t, root, id, t.TempDir())
	req := httptest.NewRequest(http.MethodPost, "/spans/"+id+"/pause", nil)
	rec := httptest.NewRecorder()
	New(root, "", nil, nil, live.New(), nil).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "not running") {
		t.Fatalf("body = %s", rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, id, "pause")); !os.IsNotExist(err) {
		t.Fatal("a pause with nothing in flight queued a file for later")
	}
}
