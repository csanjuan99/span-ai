package engine

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"span/internal/prove"
)

func files(t *testing.T, root string, names ...string) {
	t.Helper()
	for _, name := range names {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDependenciesAreDetectedFromTheLockfiles(t *testing.T) {
	cases := map[string]struct {
		files []string
		want  []string
	}{
		"nothing to install":    {files: []string{"go.mod"}},
		"npm lockfile":          {files: []string{"package.json", "package-lock.json"}, want: []string{". npm ci"}},
		"pnpm":                  {files: []string{"package.json", "pnpm-lock.yaml"}, want: []string{". pnpm install --frozen-lockfile"}},
		"yarn":                  {files: []string{"package.json", "yarn.lock"}, want: []string{". yarn install --frozen-lockfile"}},
		"bun":                   {files: []string{"package.json", "bun.lock"}, want: []string{". bun install --frozen-lockfile"}},
		"no lockfile":           {files: []string{"package.json"}, want: []string{". npm install --no-package-lock"}},
		"a project in a folder": {files: []string{"go.mod", "web/package.json", "web/package-lock.json"}, want: []string{"web npm ci"}},
		"a workspace installs once": {
			files: []string{"package.json", "pnpm-lock.yaml", "web/package.json"}, want: []string{". pnpm install --frozen-lockfile"},
		},
		"a folder with its own lockfile": {
			files: []string{"package.json", "package-lock.json", "web/package.json", "web/package-lock.json"},
			want:  []string{". npm ci", "web npm ci"},
		},
		"yarn berry":        {files: []string{"package.json", "yarn.lock", ".yarnrc.yml"}, want: []string{". yarn install --immutable"}},
		"already installed": {files: []string{"package.json", "package-lock.json", "node_modules/a/x"}},
		"python and php": {
			files: []string{"uv.lock", "api/composer.json", "api/composer.lock"}, want: []string{". uv sync --frozen", "api composer install"},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			files(t, root, tc.files...)
			var got []string
			for _, step := range detectPrepare(root, false) {
				rel, err := filepath.Rel(root, step.dir)
				if err != nil {
					t.Fatal(err)
				}
				got = append(got, rel+" "+step.command)
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// A worktree installs what it needs with nothing configured, and only then records a baseline.
func TestAWorktreePreparesItselfWithoutConfiguration(t *testing.T) {
	opt, _, check := baselineOpt(t)
	opt.Worktree = true
	opt.Project = t.TempDir()
	files(t, opt.Workspace, "package.json", "package-lock.json")
	check.respond = func(command string, _ int) (prove.Output, error) {
		if command == "npm ci" {
			return prove.Output{}, nil
		}
		return prove.Output{ExitCode: 1, Text: legacyFailure}, nil
	}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if len(check.calls) < 2 || check.calls[0] != "npm ci" {
		t.Fatalf("dependencies were not installed first: %v", check.calls)
	}
	if _, ok := readBaseline(t, opt, got.ID); !ok {
		t.Fatal("no baseline after installing")
	}
}

// An install that was interrupted leaves its folder behind: it is not trusted.
func TestAnInterruptedInstallRunsAgain(t *testing.T) {
	root := t.TempDir()
	files(t, root, "package.json", "package-lock.json", "node_modules/partial/x")
	if got := detectPrepare(root, false); len(got) != 0 {
		t.Fatalf("%+v", got)
	}
	if got := detectPrepare(root, true); len(got) != 1 || got[0].command != "npm ci" {
		t.Fatalf("%+v", got)
	}
}

func TestThePackageManagerFieldChoosesTheToolWithoutALockfile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"packageManager":"pnpm@9.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := detectPrepare(root, false); len(got) != 1 || got[0].command != "pnpm install --no-lockfile" {
		t.Fatalf("%+v", got)
	}
}
