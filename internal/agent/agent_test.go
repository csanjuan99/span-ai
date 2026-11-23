package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSelectSkipsCLIWithoutGentleAI(t *testing.T) {
	home := t.TempDir()
	orig := Look
	t.Cleanup(func() { Look = orig })
	Look = func(name string) error {
		if name == "claude" || name == "cursor-agent" {
			return nil
		}
		return os.ErrNotExist
	}
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte("# Claude\n\nOrganic Driven Development is mandatory.\n")
	if err := os.WriteFile(filepath.Join(home, ".claude", "CLAUDE.md"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Select("any", home)
	if err != nil {
		t.Fatal(err)
	}
	if got.Build != "claude" || got.Inspect != "claude" {
		t.Fatalf("%+v", got)
	}
}

func TestSelectUsesASecondAgentForInspect(t *testing.T) {
	home := t.TempDir()
	orig := Look
	t.Cleanup(func() { Look = orig })
	Look = func(string) error { return nil }
	if err := os.MkdirAll(filepath.Join(home, ".cursor", "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".cursor", "rules", "gentle-ai.mdc"), []byte("odd"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".codex", "AGENTS.md"), []byte("Organic Driven Development"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Select("cursor-agent", home)
	if err != nil {
		t.Fatal(err)
	}
	if got.Build != "cursor-agent" || got.Inspect != "codex" {
		t.Fatalf("%+v", got)
	}
}

func TestSelectNamedAgentMustBeEnabled(t *testing.T) {
	orig := Look
	t.Cleanup(func() { Look = orig })
	Look = func(string) error { return os.ErrNotExist }
	if _, err := Select("claude", t.TempDir()); err == nil {
		t.Fatal("expected missing agent")
	}
}

func TestCommandArgsLetsTheHeadlessAgentWrite(t *testing.T) {
	tests := []struct {
		name string
		want []string
	}{
		{name: "cursor-agent", want: []string{"-p", "--trust", "--force"}},
		{name: "claude", want: []string{"-p", "--dangerously-skip-permissions"}},
		{name: "codex", want: []string{"exec", "--sandbox", "workspace-write", "--approve-for-me"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := commandArgs(tt.name)
			if strings.Join(got, " ") != strings.Join(tt.want, " ") {
				t.Fatalf("args %q", got)
			}
		})
	}
}

func TestParseJSONReadsObjectInsideProse(t *testing.T) {
	var doc struct {
		Blocking bool   `json:"blocking"`
		Reason   string `json:"reason"`
	}
	if err := ParseJSON("note {\"blocking\":true,\"reason\":\"missing spec\"} thanks", &doc); err != nil {
		t.Fatal(err)
	}
	if !doc.Blocking || doc.Reason != "missing spec" {
		t.Fatalf("%+v", doc)
	}
}

func TestACancelledAgentReportsTheContextErrorNotABlock(t *testing.T) {
	script := filepath.Join(t.TempDir(), "slow-agent")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nsleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	_, err := RunWriter(ctx, script, t.TempDir(), "prompt", nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want the context error", err)
	}
}
