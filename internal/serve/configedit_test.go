package serve

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

const editBody = "# Hand-written.\nstages:\n  merge: auto\nprove:\n  fast:\n    - go test ./...\nrepair:\n  maxAttempts: 3\n"

func editProject(t *testing.T, body string) (string, string) {
	t.Helper()
	project := t.TempDir()
	path := filepath.Join(project, "span.yaml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return project, path
}

func postConfig(t *testing.T, route, project string, fields url.Values) (int, configEdit, string) {
	t.Helper()
	form := url.Values{"project": {project}}
	for key, values := range fields {
		form[key] = values
	}
	rec := postForm(t, Handler(t.TempDir(), ""), route, form)
	var out configEdit
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
	}
	return rec.Code, out, rec.Body.String()
}

func fileOf(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestPreviewListsTheChangesAndWritesNothing(t *testing.T) {
	project, path := editProject(t, editBody)
	code, out, body := postConfig(t, "/config/preview", project, url.Values{"merge": {"hold"}, "fast": {"go vet ./...\n\ngo test ./..."}, "repair": {"3"}})
	if code != http.StatusOK {
		t.Fatalf("status %d %s", code, body)
	}
	if len(out.Lines) != 2 || out.Lines[0].Key != "stages.merge" || out.Lines[0].After != "hold" || out.Lines[1].Key != "prove.fast" || out.Lines[1].After != "go vet ./..., go test ./..." {
		t.Fatalf("lines = %+v", out.Lines)
	}
	if fileOf(t, path) != editBody {
		t.Fatal("preview wrote")
	}
}

func TestSaveWritesTheFileAndReturnsTheNewSettings(t *testing.T) {
	project, path := editProject(t, editBody)
	code, out, body := postConfig(t, "/config", project, url.Values{"merge": {"hold"}, "qa": {"npm test"}, "base": {"develop"}, "repair": {"5"}})
	if code != http.StatusOK {
		t.Fatalf("status %d %s", code, body)
	}
	if out.Config == nil || out.Config.Stages.Merge != "hold" || out.Config.QA != "npm test" || out.Config.Base != "develop" || out.Config.Repair != 5 {
		t.Fatalf("config = %+v", out.Config)
	}
	if len(out.Lines) != 4 {
		t.Fatalf("lines = %+v", out.Lines)
	}
	got := fileOf(t, path)
	if got[:len("# Hand-written.")] != "# Hand-written." {
		t.Fatalf("comment lost:\n%s", got)
	}
}

func TestSaveRefusesAnInvalidValueAndLeavesTheFile(t *testing.T) {
	cases := map[string]url.Values{
		"mode":   {"merge": {"maybe"}},
		"repair": {"repair": {"12"}},
		"number": {"repair": {"many"}},
		"prose":  {"fast": {"go test | tee"}},
		"base":   {"base": {"HEAD"}},
	}
	for name, fields := range cases {
		project, path := editProject(t, editBody)
		code, _, body := postConfig(t, "/config", project, fields)
		if code != http.StatusBadRequest {
			t.Fatalf("%s: status %d %s", name, code, body)
		}
		if fileOf(t, path) != editBody {
			t.Fatalf("%s: the file changed", name)
		}
	}
}

func TestSaveRefusesAKeyThatIsNotEditable(t *testing.T) {
	project, path := editProject(t, editBody)
	code, _, body := postConfig(t, "/config", project, url.Values{"agent": {"pi"}, "harness": {"x"}})
	if code != http.StatusBadRequest {
		t.Fatalf("status %d %s", code, body)
	}
	if fileOf(t, path) != editBody {
		t.Fatal("the file changed")
	}
}

func TestSaveWontCreateSpanYamlOutsideAProject(t *testing.T) {
	project := t.TempDir()
	code, _, _ := postConfig(t, "/config", project, url.Values{"merge": {"hold"}})
	if code != http.StatusBadRequest {
		t.Fatalf("status %d", code)
	}
	if _, err := os.Stat(filepath.Join(project, "span.yaml")); !os.IsNotExist(err) {
		t.Fatal("span.yaml was created")
	}
}

func TestSaveCreatesSpanYamlInAGitProject(t *testing.T) {
	project := t.TempDir()
	if err := os.Mkdir(filepath.Join(project, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	code, out, body := postConfig(t, "/config", project, url.Values{"merge": {"hold"}})
	if code != http.StatusOK || out.Config == nil || !out.Config.Exists {
		t.Fatalf("status %d %s", code, body)
	}
	if got := fileOf(t, filepath.Join(project, "span.yaml")); got != "stages:\n  merge: hold\n" {
		t.Fatalf("got %q", got)
	}
}

func TestSaveWithNothingChangedWritesNothing(t *testing.T) {
	project, path := editProject(t, editBody)
	code, out, _ := postConfig(t, "/config", project, url.Values{"merge": {"auto"}})
	if code != http.StatusOK || len(out.Lines) != 0 {
		t.Fatalf("status %d lines %+v", code, out.Lines)
	}
	if fileOf(t, path) != editBody {
		t.Fatal("the file changed")
	}
}
