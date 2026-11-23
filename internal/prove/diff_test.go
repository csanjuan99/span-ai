package prove

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// listPackage is one package as go list -json prints it.
type listPackage struct {
	ImportPath   string
	Dir          string
	Imports      []string `json:",omitempty"`
	TestImports  []string `json:",omitempty"`
	XTestImports []string `json:",omitempty"`
}

// module is a small graph: b imports a, c's tests import a, d's external tests import b,
// f imports e, and e has no importer outside f.
var module = []listPackage{
	{ImportPath: "m/a", Dir: "/repo/a", Imports: []string{"fmt"}},
	{ImportPath: "m/b", Dir: "/repo/b", Imports: []string{"m/a", "strings"}},
	{ImportPath: "m/c", Dir: "/repo/c", TestImports: []string{"m/a", "testing"}},
	{ImportPath: "m/d", Dir: "/repo/d", XTestImports: []string{"m/b", "m/d"}},
	{ImportPath: "m/e", Dir: "/repo/e"},
	{ImportPath: "m/f", Dir: "/repo/f", Imports: []string{"m/e"}},
}

func listing(t *testing.T, pkgs []listPackage) string {
	t.Helper()
	var b strings.Builder
	for _, pkg := range pkgs {
		raw, err := json.MarshalIndent(pkg, "", "\t")
		if err != nil {
			t.Fatal(err)
		}
		b.Write(raw)
		b.WriteString("\n")
	}
	return b.String()
}

// lister answers go list with a canned listing and records every command.
type lister struct {
	calls []string
	text  string
	exit  int
	err   error
}

func (l *lister) Run(_ context.Context, command, _ string) (Output, error) {
	l.calls = append(l.calls, command)
	if !strings.HasPrefix(command, "go list ") {
		return Output{}, fmt.Errorf("unexpected command %q", command)
	}
	if l.err != nil {
		return Output{}, l.err
	}
	return Output{ExitCode: l.exit, Text: l.text}, nil
}

func TestDiffFocusTestsTheChangedGoPackageAndItsImporters(t *testing.T) {
	cases := []struct {
		name    string
		command string
		changed []string
		want    []string
	}{
		{"importers, test-only and external test edges", "go test ./...", []string{"a/a.go"},
			[]string{"go test m/a m/b m/c m/d"}},
		{"a test file maps to its own package", "go test ./...", []string{"e/e_test.go"},
			[]string{"go test m/e m/f"}},
		{"no importers", "go test ./...", []string{"f/f.go", "README.md", "docs/notes.md"},
			[]string{"go test m/f"}},
		{"coverpkg keeps its pattern", "go test -coverpkg ./... ./...", []string{"f/f.go"},
			[]string{"go test -coverpkg ./... m/f"}},
		{"coverpkg with equals", "go test -coverpkg=./... ./...", []string{"f/f.go"},
			[]string{"go test -coverpkg=./... m/f"}},
		{"flags are kept in order", "go test -race -run TestX -count=1 ./...", []string{"f/f.go"},
			[]string{"go test -race -run TestX -count=1 m/f"}},
		{"a value flag after the pattern", "go test ./... -timeout 5m", []string{"f/f.go"},
			[]string{"go test -timeout 5m m/f"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			check := &lister{text: listing(t, module)}
			changed := append([]string{}, tc.changed...)
			got := DiffFocus(context.Background(), check, "/repo", []string{tc.command}, changed)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
			if !reflect.DeepEqual(changed, tc.changed) {
				t.Fatalf("the input was changed: %q", changed)
			}
			if len(check.calls) != 1 || strings.ContainsAny(check.calls[0], "'\"") {
				t.Fatalf("calls %q", check.calls)
			}
		})
	}
}

func TestDiffFocusTestsTheRootPackage(t *testing.T) {
	root := append([]listPackage{{ImportPath: "m", Dir: "/repo"}}, module...)
	got := DiffFocus(context.Background(), &lister{text: listing(t, root)}, "/repo", []string{"go test ./..."}, []string{"main.go"})
	if !reflect.DeepEqual(got, []string{"go test m"}) {
		t.Fatalf("got %q", got)
	}
	// Every directory is under the root package, so any Markdown file may be read by its tests.
	if got := DiffFocus(context.Background(), &lister{text: listing(t, root)}, "/repo", []string{"go test ./..."}, []string{"main.go", "docs/notes.md"}); got != nil {
		t.Fatalf("narrowed to %q", got)
	}
}

func TestDiffFocusPassesBuildTagsToGoList(t *testing.T) {
	check := &lister{text: listing(t, module)}
	got := DiffFocus(context.Background(), check, "/repo", []string{"go test -tags integration ./..."}, []string{"f/f.go"})
	if !reflect.DeepEqual(got, []string{"go test -tags integration m/f"}) {
		t.Fatalf("got %q", got)
	}
	if !strings.Contains(check.calls[0], "-tags integration") {
		t.Fatalf("go list ignored the build tags: %q", check.calls[0])
	}
}

func TestDiffFocusKeepsTheWideBandWhenItCannotNarrowGo(t *testing.T) {
	cases := []struct {
		name    string
		command string
		changed []string
		check   *lister
	}{
		{"every package affected", "go test ./...", []string{"a/a.go", "e/e.go"}, nil},
		{"markdown inside a package", "go test ./...", []string{"f/f.go", "f/README.md"}, nil},
		{"markdown below a package", "go test ./...", []string{"f/f.go", "f/golden/want.md"}, nil},
		{"-args", "go test -v ./... -args -x", []string{"f/f.go"}, nil},
		{"-args first", "go test -args -x ./...", []string{"f/f.go"}, nil},
		{"-mod", "go test -mod=vendor ./...", []string{"f/f.go"}, nil},
		{"-mod with a value", "go test -mod vendor ./...", []string{"f/f.go"}, nil},
		{"-modfile", "go test -modfile=alt.mod ./...", []string{"f/f.go"}, nil},
		{"go.mod", "go test ./...", []string{"a/a.go", "go.mod"}, nil},
		{"go.sum", "go test ./...", []string{"go.sum"}, nil},
		{"testdata", "go test ./...", []string{"a/testdata/case.json"}, nil},
		{"a go file under testdata", "go test ./...", []string{"a/testdata/x.go"}, nil},
		{"yaml", "go test ./...", []string{"a/a.go", "config.yaml"}, nil},
		{"embedded asset", "go test ./...", []string{"a/page.html"}, nil},
		{"only docs", "go test ./...", []string{"README.md"}, nil},
		{"nothing changed", "go test ./...", nil, nil},
		{"a directory with no package", "go test ./...", []string{"tools/gen.go"}, nil},
		{"an extra package argument", "go test ./... ./other", []string{"f/f.go"}, nil},
		{"a single package", "go test ./a", []string{"a/a.go"}, nil},
		{"no package argument", "go test", []string{"a/a.go"}, nil},
		{"a value flag missing its value", "go test ./... -run", []string{"f/f.go"}, nil},
		{"not go test", "go vet ./...", []string{"f/f.go"}, nil},
		{"prose", "go test ./... | tee out", []string{"f/f.go"}, nil},
		{"go list failed", "go test ./...", []string{"f/f.go"}, &lister{exit: 1, text: "go: cannot find main module"}},
		{"go list did not start", "go test ./...", []string{"f/f.go"}, &lister{err: fmt.Errorf("executable not found")}},
		{"go list printed nothing", "go test ./...", []string{"f/f.go"}, &lister{}},
		{"go list printed garbage", "go test ./...", []string{"f/f.go"}, &lister{text: "{not json"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			check := tc.check
			if check == nil {
				check = &lister{text: listing(t, module)}
			}
			if got := DiffFocus(context.Background(), check, "/repo", []string{tc.command}, tc.changed); got != nil {
				t.Fatalf("narrowed to %q", got)
			}
		})
	}
}

func TestDiffFocusGivesUpOnTooManyFiles(t *testing.T) {
	changed := make([]string, 201)
	for i := range changed {
		changed[i] = "f/f.go"
	}
	check := &lister{text: listing(t, module)}
	if got := DiffFocus(context.Background(), check, "/repo", []string{"go test ./..."}, changed); got != nil {
		t.Fatalf("narrowed to %q", got)
	}
	if got := DiffFocus(context.Background(), check, "/repo", []string{"vitest run"}, []string{"a.ts"}); got == nil {
		t.Fatal("a small diff was not narrowed")
	}
}

func TestDiffFocusSkipsProseAndEmptyCommands(t *testing.T) {
	check := &lister{text: listing(t, module)}
	got := DiffFocus(context.Background(), check, "/repo", []string{"", "make lint | tee", "go test ./..."}, []string{"f/f.go"})
	if !reflect.DeepEqual(got, []string{"go test m/f"}) {
		t.Fatalf("got %q", got)
	}
}

func TestDiffFocusRewritesVitestAndJest(t *testing.T) {
	changed := []string{"src/b.tsx", "src/a.ts", "docs/notes.md"}
	cases := map[string]string{
		"vitest":                      "vitest related --run src/a.ts src/b.tsx",
		"vitest run":                  "vitest related --run src/a.ts src/b.tsx",
		"vitest run --coverage":       "vitest related --run --coverage src/a.ts src/b.tsx",
		"vitest --run":                "vitest related --run src/a.ts src/b.tsx",
		"npx vitest run":              "npx vitest related --run src/a.ts src/b.tsx",
		"pnpm vitest run":             "pnpm vitest related --run src/a.ts src/b.tsx",
		"pnpm exec vitest run":        "pnpm exec vitest related --run src/a.ts src/b.tsx",
		"yarn vitest":                 "yarn vitest related --run src/a.ts src/b.tsx",
		"jest":                        "jest --findRelatedTests src/a.ts src/b.tsx",
		"npx jest --ci":               "npx jest --ci --findRelatedTests src/a.ts src/b.tsx",
		"pnpm jest":                   "pnpm jest --findRelatedTests src/a.ts src/b.tsx",
		"yarn jest --reporter=silent": "yarn jest --reporter=silent --findRelatedTests src/a.ts src/b.tsx",
	}
	for command, want := range cases {
		check := &lister{}
		got := DiffFocus(context.Background(), check, "/repo", []string{command}, changed)
		if !reflect.DeepEqual(got, []string{want}) {
			t.Fatalf("%q derived %q, want %q", command, got, want)
		}
		if len(check.calls) != 0 {
			t.Fatalf("%q ran %q", command, check.calls)
		}
	}
	if changed[0] != "src/b.tsx" {
		t.Fatalf("the input was sorted in place: %q", changed)
	}
}

func TestDiffFocusKeepsTheWideBandWhenItCannotNarrowJavaScript(t *testing.T) {
	cases := []struct {
		command string
		changed []string
	}{
		{"npm test", []string{"src/a.ts"}},
		{"pnpm test", []string{"src/a.ts"}},
		{"npm run test", []string{"src/a.ts"}},
		{"vitest", []string{"src/a.ts", "package.json"}},
		{"vitest", []string{"src/a.ts", "pnpm-lock.yaml"}},
		{"vitest", []string{"src/a.ts", "vitest.config.ts"}},
		{"vitest", []string{"src/a.ts", "vite.config.mjs"}},
		{"vitest", []string{"src/a.ts", "vitest.workspace.ts"}},
		{"vitest", []string{"src/a.ts", "vitest.config.mts"}},
		{"jest", []string{"src/a.ts", "jest.config.js"}},
		{"vitest", []string{"src/a.ts", "src/setupTests.ts"}},
		{"jest", []string{"src/a.ts", "src/__mocks__/api.ts"}},
		{"vitest", []string{"src/a.ts.orig"}},
		{"vitest", []string{"src/a.ts", "tsconfig.json"}},
		{"jest", []string{"src/a.ts", "src/a.css"}},
		{"vitest", []string{"README.md"}},
		{"vitest", nil},
		{"vitest watch", []string{"src/a.ts"}},
		{"vitest run src/a.test.ts", []string{"src/a.ts"}},
		{"vitest run --reporter dot", []string{"src/a.ts"}},
		{"jest --findRelatedTests src/x.ts", []string{"src/a.ts"}},
		{"jest src", []string{"src/a.ts"}},
	}
	for _, tc := range cases {
		if got := DiffFocus(context.Background(), &lister{}, "/repo", []string{tc.command}, tc.changed); got != nil {
			t.Fatalf("%q with %q narrowed to %q", tc.command, tc.changed, got)
		}
	}
}

func TestDiffFocusIsDeterministic(t *testing.T) {
	first := DiffFocus(context.Background(), &lister{text: listing(t, module)}, "/repo", []string{"go test ./..."}, []string{"b/b.go", "a/a.go"})
	second := DiffFocus(context.Background(), &lister{text: listing(t, module)}, "/repo", []string{"go test ./..."}, []string{"a/a.go", "b/b.go"})
	if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(first, []string{"go test m/a m/b m/c m/d"}) {
		t.Fatalf("first %q second %q", first, second)
	}
}
