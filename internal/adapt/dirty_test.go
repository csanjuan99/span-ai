package adapt

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func repo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "span@example.test"},
		{"config", "user.name", "Span"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	return dir
}

func commitAll(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "-A"}, {"commit", "-q", "-m", "base"}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
}

func TestDirtyIgnoresSpansOwnStateDirectory(t *testing.T) {
	dir := repo(t)
	commitAll(t, dir, "README.md", "base\n")

	if dirty, err := (Git{Dir: dir}).Dirty(context.Background()); err != nil || dirty {
		t.Fatalf("a clean tree reported dirty=%v err=%v", dirty, err)
	}

	state := filepath.Join(dir, ".span", "runs", "2026-01-01-0000-aaaa")
	if err := os.MkdirAll(state, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(state, "cursor.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dirty, err := (Git{Dir: dir}).Dirty(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if dirty {
		t.Fatal("Span's own state made the workspace look changed")
	}
}

func TestDirtyStillSeesARealChange(t *testing.T) {
	dir := repo(t)
	commitAll(t, dir, "README.md", "base\n")
	if err := os.WriteFile(filepath.Join(dir, "src.ts"), []byte("export const x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dirty, err := (Git{Dir: dir}).Dirty(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !dirty {
		t.Fatal("a real change was not seen")
	}
}
