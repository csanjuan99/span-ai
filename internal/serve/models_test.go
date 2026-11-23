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
)

// ready makes every CLI installed and configured with Gentle AI.
func ready(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	for _, spec := range agent.Specs {
		path := filepath.Join(home, filepath.FromSlash(spec.Marker))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("Organic Driven Development\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	origHome, origLook := agent.Home, agent.Look
	agent.Home = func() string { return home }
	agent.Look = func(string) error { return nil }
	t.Cleanup(func() { agent.Home, agent.Look = origHome, origLook })
}

func postModel(t *testing.T, project, cli, model string) *httptest.ResponseRecorder {
	t.Helper()
	form := url.Values{"project": {project}, "cli": {cli}, "model": {model}}
	req := httptest.NewRequest(http.MethodPost, "/agent/model", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	Handler(t.TempDir(), "").ServeHTTP(rec, req)
	return rec
}

func gitProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestChoosingAModelWritesThePolicyAndTheAgentViewReadsItBack(t *testing.T) {
	ready(t)
	project := gitProject(t)
	if rec := postModel(t, project, "claude", "opus"); rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	if got := getAgent(t, project).Models["claude"]; got != "opus" {
		t.Fatalf("model %q", got)
	}
	raw, _ := os.ReadFile(filepath.Join(project, "span.yaml"))
	if string(raw) != "models:\n  claude: opus\n" {
		t.Fatalf("span.yaml %q", raw)
	}
	if rec := postModel(t, project, "claude", ""); rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	if got := getAgent(t, project).Models; len(got) != 0 {
		t.Fatalf("the model stayed: %v", got)
	}
}

func TestAModelThatCouldBeAFlagOrAnUnknownCLIIsRefusedAndNothingIsWritten(t *testing.T) {
	ready(t)
	project := gitProject(t)
	for _, tc := range [][2]string{{"claude", "--approve"}, {"claude", "a b"}, {"gemini", "x"}, {"", "x"}} {
		if rec := postModel(t, project, tc[0], tc[1]); rec.Code == http.StatusOK {
			t.Fatalf("%v was accepted", tc)
		}
	}
	if _, err := os.Stat(filepath.Join(project, "span.yaml")); !os.IsNotExist(err) {
		t.Fatalf("a refused request wrote span.yaml: %v", err)
	}
}

func TestACLIWithoutGentleAICannotGetAModelAndTheReasonSaysWhy(t *testing.T) {
	home := t.TempDir()
	origHome, origLook := agent.Home, agent.Look
	agent.Home = func() string { return home }
	agent.Look = func(string) error { return nil }
	t.Cleanup(func() { agent.Home, agent.Look = origHome, origLook })
	rec := postModel(t, gitProject(t), "claude", "opus")
	if rec.Code == http.StatusOK {
		t.Fatal("a model was set for a CLI without Gentle AI")
	}
	if !strings.Contains(rec.Body.String(), "Gentle AI and its configuration are required") {
		t.Fatalf("body %s", rec.Body.String())
	}
}

func TestTheModelListIsServedPerCLIAndAnUnknownCLIIsRefused(t *testing.T) {
	get := func(cli string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/agent/models?cli="+url.QueryEscape(cli), nil)
		rec := httptest.NewRecorder()
		Handler(t.TempDir(), "").ServeHTTP(rec, req)
		return rec
	}
	rec := get("claude")
	var view modelView
	if err := json.Unmarshal(rec.Body.Bytes(), &view); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("%d %s %v", rec.Code, rec.Body.String(), err)
	}
	if view.Source != "fixed" || len(view.Models) != 3 || !view.Free {
		t.Fatalf("%+v", view)
	}
	codex := get("codex")
	if err := json.Unmarshal(codex.Body.Bytes(), &view); err != nil || view.Models == nil || len(view.Models) != 0 {
		t.Fatalf("an empty list must be [] for the page: %s", codex.Body.String())
	}
	if get("gemini").Code == http.StatusOK {
		t.Fatal("an unknown CLI was listed")
	}
}

func TestAModelCanAlwaysBeClearedEvenWhenTheCLIIsNoLongerReady(t *testing.T) {
	ready(t)
	project := gitProject(t)
	if rec := postModel(t, project, "claude", "opus"); rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	home := t.TempDir()
	agent.Home = func() string { return home }
	if rec := postModel(t, project, "claude", "sonnet"); rec.Code == http.StatusOK {
		t.Fatal("a model was changed for a CLI that is no longer ready")
	}
	if rec := postModel(t, project, "claude", ""); rec.Code != http.StatusOK {
		t.Fatalf("a saved model could not be cleared: %d %s", rec.Code, rec.Body.String())
	}
	if got := getAgent(t, project).Models; len(got) != 0 {
		t.Fatalf("the model stayed: %v", got)
	}
}
