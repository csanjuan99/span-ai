package policy

import (
	"strings"
	"testing"
)

func TestCommitAllowIsReadFromTheYAML(t *testing.T) {
	got, err := Parse([]byte("commit:\n  allow:\n    - fixtures/\n    - .env.test\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Commit.Allow) != 2 || got.Commit.Allow[0] != "fixtures/" {
		t.Fatalf("%+v", got.Commit)
	}
}

func TestCommitAllowRefusesPathsThatLeaveTheRepository(t *testing.T) {
	for _, bad := range []string{"../secret", "/etc/passwd", "a/../../b", ""} {
		_, err := Parse([]byte("commit:\n  allow:\n    - \"" + bad + "\"\n"))
		if err == nil || !strings.Contains(err.Error(), "commit.allow") {
			t.Fatalf("%q was accepted: %v", bad, err)
		}
	}
}
