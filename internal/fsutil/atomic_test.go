package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAtomicCreatesTheFolderAndReplacesTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a", "b.json")
	for _, want := range []string{"one", "two"} {
		if err := WriteAtomic(path, []byte(want)); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Fatalf("got %q, %v; want %q", got, err, want)
		}
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 {
		t.Fatalf("a temporary file was left behind: %v, %v", entries, err)
	}
}
