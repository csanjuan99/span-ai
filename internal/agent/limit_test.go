package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClassifyReadsAQuotaLimit(t *testing.T) {
	for _, text := range []string{
		"Error: Claude AI usage limit reached. Your limit resets at 15:00.",
		"API error: 429 Too Many Requests",
		"openai: rate_limit_exceeded: Please slow down",
		"You have exceeded your current quota, please check your plan",
		"insufficient credits to run this request",
		"weekly limit reached for this account",
	} {
		kind, detail, ok := Classify(text)
		if !ok {
			t.Fatalf("not classified as a limit: %q", text)
		}
		if kind != Quota {
			t.Fatalf("%q read as %s, want %s", text, kind, Quota)
		}
		if detail == "" {
			t.Fatalf("%q kept no line to show", text)
		}
	}
}

func TestClassifyReadsMissingCredentials(t *testing.T) {
	for _, text := range []string{
		"No API key found for openai-codex.",
		"invalid api key provided",
		"authentication_error: token expired",
		"please log in to continue",
		"401 Unauthorized",
	} {
		kind, detail, ok := Classify(text)
		if !ok {
			t.Fatalf("not classified as credentials: %q", text)
		}
		if kind != Credentials {
			t.Fatalf("%q read as %s, want %s", text, kind, Credentials)
		}
		if detail == "" {
			t.Fatalf("%q kept no line to show", text)
		}
	}
}

func TestClassifyLeavesAnOrdinaryFailureAlone(t *testing.T) {
	for _, text := range []string{
		"exit status 1: FAIL src/cart.test.ts > totals",
		"expected 3 assertions, received 1",
		"the build wrote 429 lines to the log",
		"cannot find module './totals'",
	} {
		if kind, detail, ok := Classify(text); ok {
			t.Fatalf("%q was classified as %s (%q)", text, kind, detail)
		}
	}
}

func TestRunReportsABlockedAgent(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	body := "#!/bin/sh\necho 'Claude AI usage limit reached. Your limit resets at 15:00.' >&2\nexit 1\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	_, err := RunWriter(context.Background(), "claude", dir, "prompt", nil)
	if err == nil {
		t.Fatal("a blocked agent was reported as success")
	}
	blocked, ok := AsBlocked(err)
	if !ok {
		t.Fatalf("error was not classified: %v", err)
	}
	if blocked.Kind != Quota || blocked.Agent != "claude" {
		t.Fatalf("%+v", blocked)
	}
	if !strings.Contains(blocked.Detail, "usage limit") {
		t.Fatalf("detail = %q", blocked.Detail)
	}
}

func TestRunKeepsAnOrdinaryFailureAsAnError(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	body := "#!/bin/sh\necho 'cannot find module ./totals' >&2\nexit 1\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	_, err := RunWriter(context.Background(), "claude", dir, "prompt", nil)
	if err == nil {
		t.Fatal("a failing agent was reported as success")
	}
	if _, ok := AsBlocked(err); ok {
		t.Fatalf("an ordinary failure was read as a limit: %v", err)
	}
	if !strings.Contains(err.Error(), "cannot find module") {
		t.Fatalf("the error lost the output: %v", err)
	}
}

func TestAsBlockedReadsAWrappedError(t *testing.T) {
	blocked := &Blocked{Agent: "codex", Kind: Credentials, Detail: "no api key"}
	wrapped := errors.Join(errors.New("stage build"), blocked)
	got, ok := AsBlocked(wrapped)
	if !ok || got.Kind != Credentials {
		t.Fatalf("ok=%v got=%+v", ok, got)
	}
}

func TestClassifyReadsAnExpiredSession(t *testing.T) {
	for _, text := range []string{
		"Failed to authenticate: OAuth session expired and could not be refreshed",
		"oauth session expired",
		"token expired, run /login",
	} {
		kind, _, ok := Classify(text)
		if !ok || kind != Credentials {
			t.Fatalf("%q read as %s ok=%v", text, kind, ok)
		}
	}
}

func TestAReplyOnStdoutIsKeptInTheError(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	body := "#!/bin/sh\necho 'Failed to authenticate: OAuth session expired' \nexit 1\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	_, err := RunWriter(context.Background(), "claude", dir, "prompt", nil)
	if err == nil {
		t.Fatal("a failing agent was reported as success")
	}
	if blocked, ok := AsBlocked(err); ok {
		if blocked.Kind != Credentials {
			t.Fatalf("kind = %s", blocked.Kind)
		}
		return
	}
	if !strings.Contains(err.Error(), "OAuth session expired") {
		t.Fatalf("an stdout-only diagnosis was lost: %q", err.Error())
	}
}
