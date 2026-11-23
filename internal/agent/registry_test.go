package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// look installs only the named binaries for one test and restores the real lookup.
func look(t *testing.T, names ...string) {
	t.Helper()
	old := Look
	Look = func(name string) error {
		for _, want := range names {
			if name == want {
				return nil
			}
		}
		return os.ErrNotExist
	}
	t.Cleanup(func() { Look = old })
}

// home writes the marker files a harness installs, with the Gentle AI phrase.
func home(t *testing.T, markers ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, marker := range markers {
		path := filepath.Join(dir, filepath.FromSlash(marker))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("Organic Driven Development\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestTheFourAgentsAreRegisteredInOrder(t *testing.T) {
	want := []string{"cursor", "claude", "codex", "pi"}
	if len(Specs) != len(want) {
		t.Fatalf("registry has %d agents", len(Specs))
	}
	for i, id := range want {
		if Specs[i].ID != id {
			t.Fatalf("position %d is %s, want %s", i, Specs[i].ID, id)
		}
	}
}

func TestPiRunsOnceAndTrustsTheProject(t *testing.T) {
	spec, ok := Find("pi")
	if !ok {
		t.Fatal("pi is not registered")
	}
	if strings.Join(spec.Args, " ") != "--print --approve" {
		t.Fatalf("pi args = %v", spec.Args)
	}
}

func TestFindAcceptsTheBinaryNameAndRejectsJunk(t *testing.T) {
	if spec, ok := Find("cursor-agent"); !ok || spec.ID != "cursor" {
		t.Fatalf("legacy binary name did not resolve: %+v", spec)
	}
	if _, ok := Find("gpt"); ok {
		t.Fatal("an unknown name resolved")
	}
}

func TestReadyNeedsBothTheBinaryAndGentle(t *testing.T) {
	dir := home(t, ".claude/CLAUDE.md")
	look(t, "claude")

	claude, _ := Find("claude")
	if ok, reason := claude.Ready(dir); !ok || reason != "" {
		t.Fatalf("claude should be ready: %s", reason)
	}

	pi, _ := Find("pi")
	ok, reason := pi.Ready(dir)
	if ok {
		t.Fatal("pi is ready without its binary")
	}
	if !strings.Contains(reason, "not on PATH") {
		t.Fatalf("reason = %q", reason)
	}
}

func TestAnInstalledAgentWithoutGentleNamesTheMissingHalf(t *testing.T) {
	dir := t.TempDir()
	look(t, "codex")
	codex, _ := Find("codex")
	ok, reason := codex.Ready(dir)
	if ok {
		t.Fatal("codex is ready without Gentle AI")
	}
	if !strings.Contains(reason, "does not have Gentle AI") {
		t.Fatalf("reason = %q", reason)
	}
}

func TestAPiMarkerDirectoryCountsAsGentle(t *testing.T) {
	dir := home(t, ".pi/agent/npm/node_modules/gentle-pi/skills/SKILL.md")
	look(t, "pi")
	pi, _ := Find("pi")
	if ok, reason := pi.Ready(dir); !ok {
		t.Fatalf("pi should be ready: %s", reason)
	}
}

func TestSelectPicksTheRequestedAgentAndAnotherForInspect(t *testing.T) {
	dir := home(t, ".pi/agent/npm/node_modules/gentle-pi", ".claude/CLAUDE.md")
	look(t, "pi", "claude")
	sel, err := Select("pi", dir)
	if err != nil {
		t.Fatal(err)
	}
	if sel.Build != "pi" || sel.Inspect != "claude" {
		t.Fatalf("%+v", sel)
	}
}

func TestSelectNamesWhyARequestedAgentCannotRun(t *testing.T) {
	dir := home(t, ".claude/CLAUDE.md")
	look(t, "claude")
	_, err := Select("pi", dir)
	if err == nil {
		t.Fatal("selecting an absent agent succeeded")
	}
	if !strings.Contains(err.Error(), "pi is not on PATH") {
		t.Fatalf("error = %v", err)
	}
}

func TestSelectAnyFollowsTheRegistryOrder(t *testing.T) {
	dir := home(t, ".claude/CLAUDE.md", ".codex/AGENTS.md", ".pi/agent/npm/node_modules/gentle-pi")
	look(t, "claude", "codex", "pi")
	sel, err := Select("any", dir)
	if err != nil {
		t.Fatal(err)
	}
	if sel.Build != "claude" || sel.Inspect != "codex" {
		t.Fatalf("%+v", sel)
	}
}

func TestReportDescribesEveryAgent(t *testing.T) {
	dir := home(t, ".pi/agent/npm/node_modules/gentle-pi")
	look(t, "pi", "codex")
	rows := Report(dir)
	if len(rows) != len(Specs) {
		t.Fatalf("report has %d rows", len(rows))
	}
	byID := map[string]State{}
	for _, row := range rows {
		byID[row.ID] = row
	}
	if !byID["pi"].Ready {
		t.Fatalf("pi = %+v", byID["pi"])
	}
	if byID["codex"].Ready || byID["codex"].Reason == "" {
		t.Fatalf("codex = %+v", byID["codex"])
	}
	if byID["cursor"].Installed {
		t.Fatalf("cursor = %+v", byID["cursor"])
	}
}

func TestKnownAcceptsEveryWayAPolicyCanNameAnAgent(t *testing.T) {
	for _, value := range []string{"", "auto", "any", "cursor", "claude", "codex", "pi", "cursor-agent", "Pi"} {
		if !Known(value) {
			t.Fatalf("%q should be known", value)
		}
	}
	for _, value := range []string{"gpt", "cursor-agent-x", "none"} {
		if Known(value) {
			t.Fatalf("%q should be rejected", value)
		}
	}
}
