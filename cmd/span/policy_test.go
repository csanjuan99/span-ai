package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writePolicy(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "span.yaml")
	if err := os.WriteFile(path, []byte("feature: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAResumedRunReadsThePolicyOfItsProject(t *testing.T) {
	root := t.TempDir()
	want := writePolicy(t, filepath.Join(root, "proj"))
	got := resolvePolicy(filepath.Join(root, "proj", ".span", "runs"), filepath.Join(root, "tree"), filepath.Join(root, "proj"))
	if got != want {
		t.Fatalf("policy %q, want %q", got, want)
	}
}

func TestAnOldRunWithNoProjectFallsBackToTheProjectThatOwnsTheState(t *testing.T) {
	root := t.TempDir()
	tree := filepath.Join(root, "tree")
	if err := os.MkdirAll(tree, 0o755); err != nil {
		t.Fatal(err)
	}
	want := writePolicy(t, filepath.Join(root, "proj"))
	got := resolvePolicy(filepath.Join(root, "proj", ".span", "runs"), tree, "")
	if got != want {
		t.Fatalf("a checkout that switched branches no longer has span.yaml; policy %q, want %q", got, want)
	}
}

func TestACheckoutThatStillHasThePolicyKeepsWinningOverTheStateOwner(t *testing.T) {
	root := t.TempDir()
	tree := filepath.Join(root, "tree")
	want := writePolicy(t, tree)
	writePolicy(t, filepath.Join(root, "proj"))
	got := resolvePolicy(filepath.Join(root, "proj", ".span", "runs"), tree, "")
	if got != want {
		t.Fatalf("policy %q, want %q", got, want)
	}
}

func TestNoPolicyAnywhereLeavesTheDefaultPath(t *testing.T) {
	root := t.TempDir()
	if got := resolvePolicy(filepath.Join(root, "proj", ".span", "runs"), filepath.Join(root, "tree"), ""); got != "" {
		t.Fatalf("policy %q, want empty", got)
	}
}

func TestARunWithNoPolicyAnywhereUsesTheDefaults(t *testing.T) {
	root := t.TempDir()
	tree := filepath.Join(root, "tree")
	if err := os.MkdirAll(tree, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tree, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	spans := filepath.Join(root, "proj", ".span", "runs")
	opt, err := options(tree, spans, resolvePolicy(spans, tree, ""), "", "", false)
	if err != nil {
		t.Fatalf("a missing span.yaml stopped the run: %v", err)
	}
	if opt.Policy.MergeMode() != "hold" || len(opt.Policy.Prove.Fast) != 1 || opt.Policy.Prove.Fast[0] != "go test ./..." {
		t.Fatalf("policy %+v", opt.Policy)
	}
	if _, err := os.Stat(filepath.Join(tree, "span.yaml")); !os.IsNotExist(err) {
		t.Fatalf("a run wrote span.yaml: %v", err)
	}
}

// A --policy path is a choice. A typo in it must not quietly run on the defaults; only the
// implicit <workspace>/span.yaml may be missing.
func TestAnExplicitPolicyThatIsMissingStillFails(t *testing.T) {
	tree := t.TempDir()
	missing := filepath.Join(tree, "missing.yaml")
	if _, err := options(tree, "", missing, "", "", true); err == nil {
		t.Fatal("a missing --policy ran on the defaults")
	}
	if _, err := options(tree, "", filepath.Join(tree, "span.yaml"), "", "", false); err != nil {
		t.Fatalf("the implicit span.yaml is optional: %v", err)
	}
	code, err := prepare(tree, "", missing, "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if !code.Explicit {
		t.Fatal("prepare forgot that --policy was given")
	}
	if code, _ := prepare(tree, "", "", "", "", false); code.Explicit {
		t.Fatal("the default policy path counted as explicit")
	}
}

// A folder of several repositories keeps one agent and model in its span.yaml. A repository
// inside it that has none of its own runs with them; one run in the repository itself, or one
// that chose its own, is not changed by a folder it was not launched from.
func TestAFolderPolicyFillsInWhatARepositoryLeavesUnset(t *testing.T) {
	folder := t.TempDir()
	repo := filepath.Join(folder, "api")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(dir, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "span.yaml"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(folder, "agent: pi\nrules: harness\nmodels:\n  pi: ollama/m\n")
	write(repo, "agent: any\nrules: harness\n")

	one, err := options(repo, "", filepath.Join(repo, "span.yaml"), repo, folder, false)
	if err != nil {
		t.Fatal(err)
	}
	if one.Policy.Agent != "pi" || one.Policy.Models["pi"] != "ollama/m" {
		t.Fatalf("a repository chosen from the folder ran with %q %v", one.Policy.Agent, one.Policy.Models)
	}
	if one.Folder != folder {
		t.Fatalf("the run does not remember its folder: %q", one.Folder)
	}

	direct, err := options(repo, "", filepath.Join(repo, "span.yaml"), repo, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if direct.Policy.Agent != "any" || len(direct.Policy.Models) != 0 {
		t.Fatalf("a run started in the repository took the folder's policy: %q %v", direct.Policy.Agent, direct.Policy.Models)
	}

	write(repo, "agent: claude\nrules: harness\n")
	own, err := options(repo, "", filepath.Join(repo, "span.yaml"), repo, folder, false)
	if err != nil {
		t.Fatal(err)
	}
	if own.Policy.Agent != "claude" {
		t.Fatalf("the repository's own agent was overridden: %q", own.Policy.Agent)
	}
}

func TestAFolderWithoutAPolicyLeavesTheRepositoryAsItIs(t *testing.T) {
	folder := t.TempDir()
	repo := filepath.Join(folder, "api")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "span.yaml"), []byte("agent: any\nrules: harness\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := options(repo, "", filepath.Join(repo, "span.yaml"), repo, folder, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.Policy.Agent != "any" {
		t.Fatalf("agent %q", got.Policy.Agent)
	}
}

func TestLaunchFolderIsRememberedOnlyForARepositoryInsideIt(t *testing.T) {
	folder := t.TempDir()
	if got := launchFolder(folder, folder); got != "" {
		t.Fatalf("a launch in the repository itself remembered %q", got)
	}
	if got := launchFolder(folder, filepath.Join(folder, "api")); got != folder {
		t.Fatalf("a repository inside the folder remembered %q, want %q", got, folder)
	}
}
