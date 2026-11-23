package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), Name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSetAgentReplacesOneLineAndKeepsTheComments(t *testing.T) {
	path := write(t, `# Hand-written policy.
feature: payments
# Pick the CLI. any follows the registry order.
agent: any
prove:
  fast:
    - go test ./...
`)
	if err := SetAgent(path, "pi"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, want := range []string{"# Hand-written policy.", "# Pick the CLI. any follows the registry order.", "agent: pi", "feature: payments", "- go test ./..."} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in:\n%s", want, body)
		}
	}
	if strings.Count(body, "agent:") != 1 {
		t.Fatalf("the file has more than one agent line:\n%s", body)
	}
}

func TestSetAgentInsertsTheKeyWhenItIsMissing(t *testing.T) {
	path := write(t, "feature: payments\nprove:\n  fast:\n    - go test ./...\n")
	if err := SetAgent(path, "claude"); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	body := string(raw)
	if !strings.Contains(body, "feature: payments\nagent: claude") {
		t.Fatalf("the key was not inserted after feature:\n%s", body)
	}
}

func TestSetAgentRejectsAnUnknownAgentAndLeavesTheFileAlone(t *testing.T) {
	original := "feature: payments\nagent: any\n"
	path := write(t, original)
	if err := SetAgent(path, "gpt"); err == nil {
		t.Fatal("an unknown agent was written")
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != original {
		t.Fatalf("the file changed:\n%s", raw)
	}
}

func TestSetAgentRefusesToWriteAFileThatWouldNotLoad(t *testing.T) {
	original := "agent: any\nstages:\n  merge: whenever\n"
	path := write(t, original)
	if err := SetAgent(path, "pi"); err == nil {
		t.Fatal("an invalid policy was written")
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != original {
		t.Fatalf("the file changed:\n%s", raw)
	}
}

func TestValidateRejectsAnUnknownAgent(t *testing.T) {
	file := File{Feature: "payments", Agent: "gpt"}
	err := file.Validate()
	if err == nil {
		t.Fatal("an unknown agent passed validation")
	}
	for _, want := range []string{"cursor", "claude", "codex", "pi"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("the message does not name %s: %v", want, err)
		}
	}
}

func TestValidateAcceptsTheLegacyBinaryName(t *testing.T) {
	if err := (File{Feature: "payments", Agent: "cursor-agent"}).Validate(); err != nil {
		t.Fatalf("an existing span.yaml broke: %v", err)
	}
}
