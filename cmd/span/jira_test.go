package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"span/internal/jira"
)

func jiraCall(args ...string) (int, string, string) {
	var out, errw bytes.Buffer
	code := jiraMain(args, &out, &errw)
	return code, out.String(), errw.String()
}

func TestJiraJSONMatchesTheAccountRouteAndFlagsFollowTheKey(t *testing.T) {
	dir := t.TempDir()
	code, out, errw := jiraCall("status", "--project", dir, "--json")
	var view jira.Public
	if code != 0 || json.Unmarshal([]byte(out), &view) != nil || view.Configured || errw != "" {
		t.Fatalf("unconfigured status --json answers the route's body and exits 0: %d %s %s", code, out, errw)
	}
	if code, _, errw = jiraCall("status", "--project", dir); code != 1 || !strings.Contains(errw, "not_configured") {
		t.Fatalf("unconfigured status in text keeps exit 1: %d %s", code, errw)
	}
	if code, _, _ = jiraCall("status", "--project", dir, "extra"); code != 2 {
		t.Fatalf("an extra argument is a usage error, got %d", code)
	}
	code, out, errw = jiraCall("configure", "--project", dir, "--url", "not a url", "--token", "x", "--json")
	var failed map[string]string
	if code != 1 || json.Unmarshal([]byte(out), &failed) != nil || failed["error"] != "invalid" || !strings.Contains(failed["message"], "jira site url is missing") {
		t.Fatalf("bad url %d %s %s", code, out, errw)
	}
	code, out, errw = jiraCall("configure", "--project", dir, "--url", "https://example.atlassian.net", "--email", "a@b.c", "--token", "sekret", "--json")
	if code != 0 || json.Unmarshal([]byte(out), &view) != nil || !view.Configured || view.Email != "a@b.c" || view.BaseURL != "https://example.atlassian.net" || view.Auth != "basic" || strings.Contains(out, "sekret") {
		t.Fatalf("configure %d %s %s", code, out, errw)
	}
	code, out, errw = jiraCall("issue", "not-a-key", "--project", dir, "--json")
	if code != 1 || json.Unmarshal([]byte(out), &failed) != nil || failed["error"] != "invalid" {
		t.Fatalf("flag after the key: %d %s %s", code, out, errw)
	}
	code, out, errw = jiraCall("logout", "--json", "--project", dir)
	if code != 0 || json.Unmarshal([]byte(out), &view) != nil || view.Configured {
		t.Fatalf("logout %d %s %s", code, out, errw)
	}
}

func fakeJiraSite(t *testing.T, status int) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/search/jql" {
			http.NotFound(w, r)
			return
		}
		if status != http.StatusOK {
			http.Error(w, "no", status)
			return
		}
		fmt.Fprint(w, `{"issues":[]}`)
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func TestJiraCheckAsksTheSiteWithoutSavingAndExitsByTheStatus(t *testing.T) {
	dir := t.TempDir()
	site := fakeJiraSite(t, http.StatusOK)
	code, out, errw := jiraCall("check", "--project", dir, "--url", site, "--email", "a@b.c", "--token", "t", "--json")
	var got struct{ Status, Message string }
	if code != 0 || json.Unmarshal([]byte(out), &got) != nil || got.Status != "ok" || errw != "" {
		t.Fatalf("an accepted account exits 0 with status ok: %d %s %s", code, out, errw)
	}
	if code, out, _ = jiraCall("status", "--project", dir, "--json"); code != 0 || strings.Contains(out, `"configured": true`) {
		t.Fatalf("a check saves nothing: %d %s", code, out)
	}
	code, out, _ = jiraCall("check", "--project", dir, "--url", fakeJiraSite(t, http.StatusUnauthorized), "--token", "bad", "--json")
	if code != 1 || json.Unmarshal([]byte(out), &got) != nil || got.Status != "unauthorized" {
		t.Fatalf("a refused token exits 1 with status unauthorized: %d %s", code, out)
	}
	code, out, _ = jiraCall("check", "--project", dir, "--url", site, "--json")
	if code != 1 || json.Unmarshal([]byte(out), &got) != nil || got.Status != "invalid" {
		t.Fatalf("no token is invalid: %d %s", code, out)
	}
	if code, _, _ = jiraCall("check", "--project", dir, "extra"); code != 2 {
		t.Fatalf("an extra argument is a usage error, got %d", code)
	}
}

func TestJiraCheckWithoutATokenTestsTheSavedOne(t *testing.T) {
	dir := t.TempDir()
	site := fakeJiraSite(t, http.StatusOK)
	if code, out, errw := jiraCall("configure", "--project", dir, "--url", site, "--token", "saved", "--json"); code != 0 {
		t.Fatalf("configure %d %s %s", code, out, errw)
	}
	code, out, _ := jiraCall("check", "--project", dir, "--url", site, "--json")
	if code != 0 || !strings.Contains(out, `"ok"`) {
		t.Fatalf("the saved token is tested: %d %s", code, out)
	}
}
