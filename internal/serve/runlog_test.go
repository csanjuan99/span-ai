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
)

func getLog(root, id, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	target := "/spans/" + id + "/log?path=" + url.QueryEscape(path)
	Handler(root, "").ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func writeLog(t *testing.T, root, id, rel, text string) {
	t.Helper()
	full := filepath.Join(root, id, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestACommandLogIsServedWhole(t *testing.T) {
	root, id := featureRun(t, "pay", "")
	writeLog(t, root, id, "iter/1/logs/prove-1.log", "line one\nline two\nline three\n")
	rec := getLog(root, id, "iter/1/logs/prove-1.log")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var body struct {
		Path      string `json:"path"`
		Text      string `json:"text"`
		Lines     int    `json:"lines"`
		Truncated bool   `json:"truncated"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Text != "line one\nline two\nline three\n" || body.Lines != 3 || body.Truncated || body.Path != "iter/1/logs/prove-1.log" {
		t.Fatalf("%+v", body)
	}
}

func TestALogLargerThanTheLimitKeepsTheEndAndSaysSo(t *testing.T) {
	root, id := featureRun(t, "pay", "")
	writeLog(t, root, id, "iter/1/logs/build-0.log", strings.Repeat("x", runLogLimit)+"\nthe end\n")
	rec := getLog(root, id, "iter/1/logs/build-0.log")
	var body struct {
		Text      string `json:"text"`
		Truncated bool   `json:"truncated"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Truncated || !strings.HasSuffix(body.Text, "the end\n") || len(body.Text) > runLogLimit+4 {
		t.Fatalf("truncated=%v len=%d", body.Truncated, len(body.Text))
	}
}

func TestOnlyALogUnderTheRunsIterationsIsRead(t *testing.T) {
	root, id := featureRun(t, "pay", "")
	writeLog(t, root, id, "iter/1/logs/prove-1.log", "ok")
	writeLog(t, root, id, "cursor-copy.txt", "secret")
	for _, path := range []string{
		"", "cursor.json", "../other/iter/1/logs/x.log", "/etc/passwd", "iter/1/logs/../../../cursor.json",
		"iter/1/prove.json", "iter/1/logs/missing.log", `iter\1\logs\prove-1.log`,
	} {
		if rec := getLog(root, id, path); rec.Code != http.StatusNotFound {
			t.Errorf("%q: status %d, want 404", path, rec.Code)
		}
	}
}

func TestALogLinkedOutsideTheRunIsNotFollowed(t *testing.T) {
	root, id := featureRun(t, "pay", "")
	outside := filepath.Join(t.TempDir(), "outside.log")
	if err := os.WriteFile(outside, []byte("private"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, id, "iter", "1", "logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "link.log")); err != nil {
		t.Skip("symlinks are not available")
	}
	if rec := getLog(root, id, "iter/1/logs/link.log"); rec.Code != http.StatusNotFound {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestACutLogKeepsItsLineNumbers(t *testing.T) {
	root, id := featureRun(t, "pay", "")
	head := strings.Repeat("a line of output\n", runLogLimit/len("a line of output\n")+10)
	writeLog(t, root, id, "iter/1/logs/prove-1.log", head+"the end\n")
	view, err := readRunLog(root, id, "iter/1/logs/prove-1.log")
	if err != nil {
		t.Fatal(err)
	}
	if !view.Truncated || view.FirstLine <= 1 || view.FirstLine+view.Lines-1 != view.TotalLines {
		t.Fatalf("first %d lines %d total %d", view.FirstLine, view.Lines, view.TotalLines)
	}
	if view.Bytes != int64(len(head)+len("the end\n")) {
		t.Fatalf("bytes %d", view.Bytes)
	}
}
