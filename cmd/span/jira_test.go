package main

import (
	"bytes"
	"encoding/json"
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
	if code != 1 || json.Unmarshal([]byte(out), &view) != nil || view.Configured || !strings.Contains(errw, "not_configured") {
		t.Fatalf("unconfigured status %d %s %s", code, out, errw)
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
