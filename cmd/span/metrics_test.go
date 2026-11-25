package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"span/internal/engine"
)

func TestMetricsPrintsWhatTheRouteAnswers(t *testing.T) {
	root := opsRun(t, engine.Cursor{Stage: "prove", Hold: true})
	var out, errw bytes.Buffer
	if code := metricsMain([]string{"--json", "--spans", root}, &out, &errw); code != 0 || errw.Len() != 0 {
		t.Fatalf("exit %d: %s", code, errw.String())
	}
	var got engine.MetricsReport
	if err := json.Unmarshal(out.Bytes(), &got); err != nil || got.Runs.Total != 1 || got.Runs.Held != 1 {
		t.Fatalf("the report is the route's body: %s (%v)", out.String(), err)
	}
	out.Reset()
	if code := metricsMain([]string{"--spans", root}, &out, &errw); code != 0 || !strings.Contains(out.String(), "1 runs") || !strings.Contains(out.String(), "held 1") {
		t.Fatalf("text %d: %q", code, out.String())
	}
	if code := metricsMain([]string{"extra"}, &out, &errw); code != 2 {
		t.Fatalf("an extra argument is a usage error, got %d", code)
	}
	empty := t.TempDir()
	out.Reset()
	if code := metricsMain([]string{"--json", "--spans", empty}, &out, &errw); code != 0 || !json.Valid(out.Bytes()) {
		t.Fatalf("a folder without runs is an empty report, not an error: %d %q", code, out.String())
	}
}

func TestStartBranchesListsWhatTheRouteListsAndRefusesANonFolder(t *testing.T) {
	dir := gitRepo(t)
	code, out, errw := startCall("branches", "--workspace", dir, "--json")
	var got struct {
		Branches []string
		Default  string
	}
	if code != 0 || json.Unmarshal([]byte(out), &got) != nil || len(got.Branches) == 0 || errw != "" {
		t.Fatalf("%d %s %s", code, out, errw)
	}
	if code, _, errw = startCall("branches", "--workspace", dir+"/nope", "--json"); code != 1 || errw == "" {
		t.Fatalf("a folder that is not there exits 1: %d %q", code, errw)
	}
}
