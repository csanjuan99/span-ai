package adapt

import (
	"context"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestArgvSplitsLikeAPersonWouldReadIt(t *testing.T) {
	cases := []struct {
		in   string
		env  []string
		argv []string
	}{
		{"go test ./...", nil, []string{"go", "test", "./..."}},
		{"pytest -k \"a and b\"", nil, []string{"pytest", "-k", "a and b"}},
		{"pytest -k 'a or b' -x", nil, []string{"pytest", "-k", "a or b", "-x"}},
		{"CI=1 pnpm test", []string{"CI=1"}, []string{"pnpm", "test"}},
		{"CI=1 NODE_ENV=test npm --prefix web test", []string{"CI=1", "NODE_ENV=test"}, []string{"npm", "--prefix", "web", "test"}},
		{"FOO=\"a b\" run", []string{"FOO=a b"}, []string{"run"}},
		{"echo \"say \\\"hi\\\"\"", nil, []string{"echo", "say \"hi\""}},
		{"run C:\\tools\\x.exe --flag", nil, []string{"run", "C:\\tools\\x.exe", "--flag"}},
		{"run \"\" x", nil, []string{"run", "", "x"}},
		{"  spaced   out  ", nil, []string{"spaced", "out"}},
		{"make A=1", nil, []string{"make", "A=1"}},
	}
	for _, tc := range cases {
		env, argv, err := Argv(tc.in)
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if !reflect.DeepEqual(env, tc.env) || !reflect.DeepEqual(argv, tc.argv) {
			t.Fatalf("%q -> env %q argv %q", tc.in, env, argv)
		}
	}
}

func TestArgvRefusesWhatItCannotRunAsOneProgram(t *testing.T) {
	for _, bad := range []string{"", "   ", "CI=1", "pytest -k \"unterminated", "run 'x"} {
		if _, _, err := Argv(bad); err == nil {
			t.Fatalf("%q was accepted", bad)
		}
	}
}

func TestExecRunsAQuotedArgumentAsOneArgument(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses printf")
	}
	out, err := (Exec{}).Run(context.Background(), "printf '%s|' \"a b\" c", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if out.Text != "a b|c|" {
		t.Fatalf("%q", out.Text)
	}
}

func TestExecPassesALeadingAssignmentToTheCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses env")
	}
	out, err := (Exec{}).Run(context.Background(), "SPAN_PROBE=yes env", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.Text, "SPAN_PROBE=yes") {
		t.Fatalf("the variable did not reach the command: %q", out.Text)
	}
}
