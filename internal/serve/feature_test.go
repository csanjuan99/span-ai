package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/engine"
)

func featureRun(t *testing.T, feature, doc string) (root, id string) {
	t.Helper()
	workspace := t.TempDir()
	if doc != "" {
		tasks := filepath.Join(workspace, "odd", "tasks")
		if err := os.MkdirAll(tasks, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tasks, feature+".md"), []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	root = t.TempDir()
	id = "span-1"
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(engine.Cursor{ID: id, Workspace: workspace, Feature: feature})
	if err := os.WriteFile(filepath.Join(dir, "cursor.json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, id
}

func getFeature(root, id string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	Handler(root, "").ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/spans/"+id+"/feature", nil))
	return rec
}

func TestTheFeatureDocumentIsServedForTheReport(t *testing.T) {
	root, id := featureRun(t, "pay", "# Pay\n\n## Log\n\nDecision: use the stored card\n")
	rec := getFeature(root, id)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var body struct {
		Path      string `json:"path"`
		Markdown  string `json:"markdown"`
		Truncated bool   `json:"truncated"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body.Markdown, "Decision: use the stored card") || body.Path != "odd/tasks/pay.md" || body.Truncated {
		t.Fatalf("%+v", body)
	}
}

func TestAMissingFeatureDocumentIsAnEmptyReportNotAnError(t *testing.T) {
	root, id := featureRun(t, "pay", "")
	rec := getFeature(root, id)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"markdown":""`) {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}

func TestAFeatureNameThatEscapesTheTasksFolderIsNotRead(t *testing.T) {
	root, id := featureRun(t, "../../secret", "")
	rec := getFeature(root, id)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"markdown":""`) {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}

func TestALongFeatureDocumentIsCut(t *testing.T) {
	root, id := featureRun(t, "pay", strings.Repeat("x", featureLimit+10))
	var body struct {
		Markdown  string `json:"markdown"`
		Truncated bool   `json:"truncated"`
	}
	if err := json.Unmarshal(getFeature(root, id).Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Truncated || len(body.Markdown) != featureLimit {
		t.Fatalf("truncated=%v len=%d", body.Truncated, len(body.Markdown))
	}
}
