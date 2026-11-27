package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/jira"
)

// --json keeps its body and exit code, and an account that cannot be read is still said.
func TestJiraStatusJSONReportsAnAccountItCannotRead(t *testing.T) {
	for name, tc := range map[string]struct {
		account string
		said    bool
	}{
		"no account":      {},
		"corrupt account": {account: "{", said: true},
	} {
		t.Run(name, func(t *testing.T) {
			project := t.TempDir()
			if tc.account != "" {
				path, err := jira.Path(project)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(tc.account), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			var out, errw bytes.Buffer
			if code := jiraMain([]string{"status", "--project", project, "--json"}, &out, &errw); code != 0 {
				t.Fatalf("exit %d: %s", code, errw.String())
			}
			if !strings.Contains(out.String(), `"configured":false`) && !strings.Contains(out.String(), `"configured": false`) {
				t.Fatalf("body %s", out.String())
			}
			if said := errw.Len() > 0; said != tc.said {
				t.Fatalf("stderr %q", errw.String())
			}
		})
	}
}
