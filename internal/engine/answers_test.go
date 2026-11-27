package engine

import (
	"os"
	"path/filepath"
	"testing"
)

// Answers are appended. A file that cannot be read must refuse, never be replaced by the
// new answers alone.
func TestAnswersThatCannotBeReadAreNotOverwritten(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "answers.md")
	if err := os.WriteFile(path, []byte("first\nanswer\n"), 0o200); err != nil {
		t.Fatal(err)
	}
	if _, err := os.ReadFile(path); err == nil {
		t.Skip("this user can read a write-only file")
	}
	if err := writeAnswers(dir, []Answer{{Question: "second", Body: "more"}}); err == nil {
		t.Fatal("unreadable answers were accepted")
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != "first\nanswer\n" {
		t.Fatalf("%q %v", raw, err)
	}
}
