package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func str(s string) *string { return &s }

func list(items ...string) *[]string { return &items }

func num(n int) *int { return &n }

func read(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

const sample = `# Hand-written policy.
# base: feat/SAL-3
run:
  weight: ""
agent: any
stages:
  open: auto
  # How a merge conflict is resolved.
  conflict: hold
prove:
  # Scoped to the story.
  focused:
    - go test {files}
  fast:
    - go test ./...
  acceptance: []
qa:
  command: go test ./...
repair:
  maxAttempts: 3
# Where the change goes.
promote:
  environments:
    - name: preview
      command: echo preview
      mode: auto
`

func TestEditChangesOneValueAndKeepsEverythingElse(t *testing.T) {
	path := write(t, sample)
	if _, err := Edit(path, Changes{Merge: str("hold")}); err != nil {
		t.Fatal(err)
	}
	got := read(t, path)
	want := strings.Replace(sample, "  conflict: hold\n", "  conflict: hold\n  merge: hold\n", 1)
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestEditReplacesAnExistingScalarInPlace(t *testing.T) {
	path := write(t, sample)
	if _, err := Edit(path, Changes{Conflict: str("auto"), Repair: num(5), QA: str("npm test")}); err != nil {
		t.Fatal(err)
	}
	got := read(t, path)
	for _, line := range []string{"  conflict: auto", "  maxAttempts: 5", "  command: npm test", "# How a merge conflict is resolved.", "# Hand-written policy.", "# Where the change goes."} {
		if !strings.Contains(got, line) {
			t.Fatalf("missing %q in:\n%s", line, got)
		}
	}
}

func TestEditReplacesAListAndLeavesTheNeighboursAlone(t *testing.T) {
	path := write(t, sample)
	if _, err := Edit(path, Changes{Fast: list("go vet ./...", "go test ./...")}); err != nil {
		t.Fatal(err)
	}
	got := read(t, path)
	want := strings.Replace(sample, "  fast:\n    - go test ./...\n", "  fast:\n    - go vet ./...\n    - go test ./...\n", 1)
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestEditEmptiesAListAndFillsAnEmptyOne(t *testing.T) {
	path := write(t, sample)
	if _, err := Edit(path, Changes{Focused: list(), Acceptance: list("npm run e2e")}); err != nil {
		t.Fatal(err)
	}
	got := read(t, path)
	if !strings.Contains(got, "  focused: []\n") || strings.Contains(got, "go test {files}") {
		t.Fatalf("focused not emptied:\n%s", got)
	}
	if !strings.Contains(got, "  acceptance:\n    - npm run e2e\n") {
		t.Fatalf("acceptance not filled:\n%s", got)
	}
	if !strings.Contains(got, "  fast:\n    - go test ./...\n") {
		t.Fatalf("fast was touched:\n%s", got)
	}
}

func TestEditReplacesAFlowStyleList(t *testing.T) {
	path := write(t, "prove:\n  fast: [go test ./...]\nqa:\n  command: x\n")
	if _, err := Edit(path, Changes{Fast: list("make test")}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, path); got != "prove:\n  fast:\n    - make test\nqa:\n  command: x\n" {
		t.Fatalf("got:\n%s", got)
	}
}

func TestEditCreatesTheSectionWhenItIsMissing(t *testing.T) {
	path := write(t, "agent: any\n")
	if _, err := Edit(path, Changes{Merge: str("hold"), Fast: list("go test ./...")}); err != nil {
		t.Fatal(err)
	}
	got := read(t, path)
	if !strings.HasPrefix(got, "agent: any\n") || !strings.Contains(got, "stages:\n  merge: hold\n") || !strings.Contains(got, "prove:\n  fast:\n    - go test ./...\n") {
		t.Fatalf("got:\n%s", got)
	}
}

func TestEditSetsAndClearsTheBase(t *testing.T) {
	path := write(t, sample)
	if _, err := Edit(path, Changes{Base: str("develop")}); err != nil {
		t.Fatal(err)
	}
	got := read(t, path)
	if !strings.Contains(got, "\nbase: develop\n") || !strings.Contains(got, "# base: feat/SAL-3") {
		t.Fatalf("got:\n%s", got)
	}
	if _, err := Edit(path, Changes{Base: str("")}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, path); strings.Contains(got, "\nbase:") || !strings.Contains(got, "# base: feat/SAL-3") {
		t.Fatalf("base not cleared:\n%s", got)
	}
}

func TestEditClearingTheQACommandDropsTheEmptySection(t *testing.T) {
	path := write(t, "qa:\n  command: go test ./...\nrepair:\n  maxAttempts: 2\n")
	if _, err := Edit(path, Changes{QA: str("")}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, path); got != "repair:\n  maxAttempts: 2\n" {
		t.Fatalf("got:\n%s", got)
	}
}

func TestEditRefusesAnInvalidValueAndWritesNothing(t *testing.T) {
	cases := map[string]Changes{
		"stage mode":    {Merge: str("maybe")},
		"open conflict": {Conflict: str("conditional")},
		"plan":          {Plan: str("sometimes")},
		"repair low":    {Repair: num(0)},
		"repair high":   {Repair: num(9)},
		"base option":   {Base: str("-x")},
		"base head":     {Base: str("HEAD")},
		"prose command": {Fast: list("go test | tee out")},
		"empty command": {Fast: list("  ")},
		"newline":       {Fast: list("go test\nrm -rf /")},
		"prose qa":      {QA: str("make; make")},
	}
	for name, change := range cases {
		path := write(t, sample)
		if _, err := Edit(path, change); err == nil {
			t.Fatalf("%s: want an error", name)
		}
		if read(t, path) != sample {
			t.Fatalf("%s: the file changed", name)
		}
	}
}

func TestEditRefusesASectionWrittenInline(t *testing.T) {
	body := "stages: {open: auto}\n"
	path := write(t, body)
	_, err := Edit(path, Changes{Merge: str("hold")})
	if err == nil || !strings.Contains(err.Error(), "inline") {
		t.Fatalf("err = %v", err)
	}
	if read(t, path) != body {
		t.Fatal("the file changed")
	}
}

func TestEditRefusesAFileThatDoesNotLoad(t *testing.T) {
	body := "stages: [broken\n"
	path := write(t, body)
	if _, err := Edit(path, Changes{Merge: str("hold")}); err == nil {
		t.Fatal("want an error")
	}
	if read(t, path) != body {
		t.Fatal("the file changed")
	}
}

func TestPreviewWritesNothingAndSaysWhatWouldChange(t *testing.T) {
	path := write(t, sample)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := PreviewEdit(path, Changes{Merge: str("hold"), Repair: num(3), Fast: list("make test")})
	if err != nil {
		t.Fatal(err)
	}
	if read(t, path) != sample {
		t.Fatal("preview wrote")
	}
	after, err := os.Stat(path)
	if err != nil || !after.ModTime().Equal(info.ModTime()) {
		t.Fatalf("preview touched the file: %v", err)
	}
	want := []DiffLine{
		{Key: "stages.merge", Before: "auto", After: "hold"},
		{Key: "prove.fast", Before: "go test ./...", After: "make test"},
	}
	if len(preview.Lines) != len(want) {
		t.Fatalf("lines = %+v", preview.Lines)
	}
	for i, line := range want {
		if preview.Lines[i] != line {
			t.Fatalf("line %d = %+v, want %+v", i, preview.Lines[i], line)
		}
	}
}

func TestAnEditThatChangesNothingWritesNothing(t *testing.T) {
	path := write(t, sample)
	preview, err := Edit(path, Changes{Open: str("auto"), Repair: num(3)})
	if err != nil || len(preview.Lines) != 0 {
		t.Fatalf("preview = %+v, err = %v", preview, err)
	}
	missing := filepath.Join(t.TempDir(), Name)
	if _, err := Edit(missing, Changes{Open: str("auto")}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatal("a no-op created the file")
	}
}

func TestEditCreatesAMissingFileWithOnlyWhatWasAsked(t *testing.T) {
	path := filepath.Join(t.TempDir(), Name)
	if _, err := Edit(path, Changes{Base: str("develop")}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, path); got != "base: develop\n" {
		t.Fatalf("got %q", got)
	}
}

func TestEditKeepsTheFileModeAndLeavesNoTempFile(t *testing.T) {
	path := write(t, sample)
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Edit(path, Changes{Merge: str("hold")}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, err = %v", info.Mode(), err)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries = %v, err = %v", entries, err)
	}
}

func TestEditedFileStillLoads(t *testing.T) {
	path := write(t, sample)
	if _, err := Edit(path, Changes{Plan: str("conditional"), Promote: str("hold"), Acceptance: list("npm run e2e")}); err != nil {
		t.Fatal(err)
	}
	file, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if file.PlanMode() != "conditional" || file.Stages.Promote != "hold" || len(file.Prove.Acceptance) != 1 || file.Promote.Environments[0].Name != "preview" {
		t.Fatalf("file = %+v", file)
	}
}
