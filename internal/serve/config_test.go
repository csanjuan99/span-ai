package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func getConfig(t *testing.T, project string) configPick {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/config?path="+url.QueryEscape(project), nil)
	rec := httptest.NewRecorder()
	Handler(t.TempDir(), "").ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	var body configPick
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body
}

func TestAProjectWithoutPolicyReadsAsDefaults(t *testing.T) {
	view := getConfig(t, t.TempDir())
	if view.Exists || view.Folder == "" {
		t.Fatalf("view = %+v", view)
	}
	if view.Stages.Merge != "auto" || view.Repair != 3 {
		t.Fatalf("defaults = %+v", view)
	}
}

func TestTheConfigReadsWhatSpanYamlSays(t *testing.T) {
	project := t.TempDir()
	body := "base: feat/SAL-3\nstages:\n  merge: hold\n  conflict: auto\nprove:\n  focused:\n    - go test {files}\n  fast:\n    - go test ./...\nqa:\n  command: npm test\nrepair:\n  maxAttempts: 5\njira:\n  blockStatus: Blocked\nlimits:\n  stageTimeout: 45m\n"
	if err := os.WriteFile(filepath.Join(project, "span.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	view := getConfig(t, project)
	if !view.Exists || view.Base != "feat/SAL-3" || view.Stages.Merge != "hold" || view.Stages.Conflict != "auto" {
		t.Fatalf("view = %+v", view)
	}
	if len(view.Prove.Fast) != 1 || view.Prove.Fast[0] != "go test ./..." || view.QA != "npm test" {
		t.Fatalf("checks = %+v", view)
	}
	if view.Repair != 5 || view.BlockStatus != "Blocked" || view.StageTimeout != "45m" {
		t.Fatalf("limits = %+v", view)
	}
}

func TestAnUnreadablePolicySaysWhyInsteadOfFailing(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "span.yaml"), []byte("stages: [broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	view := getConfig(t, project)
	if view.Exists || view.Reason == "" {
		t.Fatalf("view = %+v", view)
	}
}

func TestAProjectWithoutAgentsReadsEveryStageOnTheProjectAgent(t *testing.T) {
	view := getConfig(t, t.TempDir())
	if view.Agent != "auto" || view.DoneStatus != "" || len(view.Agents) != 3 {
		t.Fatalf("view = %+v", view)
	}
	for stage, name := range view.Agents {
		if name != "" {
			t.Fatalf("%s = %q", stage, name)
		}
	}
}
