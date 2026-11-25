package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"span/internal/start"
)

func TestProbeHealthRejectsAMalformedAnswer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte("<html>not json</html>")); err != nil {
			t.Error(err)
		}
	}))
	defer srv.Close()
	if _, _, err := probeHealth(srv.URL, ""); err == nil {
		t.Fatal("a malformed body must be an error, not a connected server")
	}
}

func TestProbeHealthReportsARefusedToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	body, refused, err := probeHealth(srv.URL, "wrong")
	if err != nil || !refused || body.Status != "refused" {
		t.Fatalf("body = %+v refused = %v err = %v", body, refused, err)
	}
}

func startCall(args ...string) (int, string, string) {
	var out, errw bytes.Buffer
	code := startMain(args, &out, &errw)
	return code, out.String(), errw.String()
}

func TestStartPrecheckExitsWithTheRefusalNumberInTextAndInJSON(t *testing.T) {
	notGit := t.TempDir()
	code, _, errw := startCall("precheck", "--workspace", notGit, "--story", "x")
	if code != int(start.NumberNotGit) || errw == "" {
		t.Fatalf("text: a refusal exits with its number: %d %q", code, errw)
	}
	code, out, errw := startCall("precheck", "--workspace", notGit, "--story", "x", "--json")
	var body struct {
		Number int
		Code   string
	}
	if json.Unmarshal([]byte(out), &body) != nil || body.Number != int(start.NumberNotGit) || body.Code != "not_git" || errw != "" {
		t.Fatalf("json: the refusal is the route's body: %q %q", out, errw)
	}
	if code != int(start.NumberNotGit) {
		t.Fatalf("json: a script reads the same exit code as the text mode: %d", code)
	}
}

func TestStartPrecheckExitsZeroWhenNothingBlocks(t *testing.T) {
	dir := gitRepo(t)
	code, out, _ := startCall("precheck", "--workspace", dir, "--story", "AET-1 Charge once", "--json")
	if code != 0 || !strings.Contains(out, `"ok": true`) {
		t.Fatalf("%d %s", code, out)
	}
}
