package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func settingsProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := exec.Command("git", "-C", dir, "init", "-q").Run(); err != nil {
		t.Skip("git unavailable")
	}
	body := "# keep this comment\nbase: dev\nrepair:\n  maxAttempts: 3 # attempts\n"
	if err := os.WriteFile(filepath.Join(dir, "span.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestSettingsSetWritesOnlyTheChangedLine(t *testing.T) {
	dir := settingsProject(t)
	var out, errw bytes.Buffer
	if code := settingsMain([]string{"set", "--project", dir, "--preview", "base=main"}, &out, &errw); code != 0 {
		t.Fatalf("preview exit %d: %s", code, errw.String())
	}
	before, err := os.ReadFile(filepath.Join(dir, "span.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(before), "base: dev") {
		t.Fatalf("preview wrote the file: %s", before)
	}
	out.Reset()
	if code := settingsMain([]string{"set", "--project", dir, "base=main"}, &out, &errw); code != 0 {
		t.Fatalf("save exit %d: %s", code, errw.String())
	}
	raw, err := os.ReadFile(filepath.Join(dir, "span.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "# keep this comment") || !strings.Contains(string(raw), "base: main") || !strings.Contains(string(raw), "maxAttempts: 3 # attempts") {
		t.Fatalf("not line-preserving: %s", raw)
	}
}

func TestSettingsRefusesWhatTheRouteRefuses(t *testing.T) {
	dir := settingsProject(t)
	var out, errw bytes.Buffer
	if code := settingsMain([]string{"set", "--project", dir, "repair=abc"}, &out, &errw); code != 1 {
		t.Fatalf("bad number exit %d", code)
	}
	if code := settingsMain([]string{"set", "--project", dir, "nope=1"}, &out, &errw); code != 1 {
		t.Fatalf("unknown key exit %d", code)
	}
	if code := settingsMain([]string{"set", "--project", dir}, &out, &errw); code != 2 {
		t.Fatalf("no keys exit %d", code)
	}
	if code := settingsMain([]string{"bogus"}, &out, &errw); code != 2 {
		t.Fatalf("unknown verb exit %d", code)
	}
}
