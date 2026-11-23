package handoff

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteKeepsTheBaseNameAndDropsTheDirectory(t *testing.T) {
	dir := t.TempDir()
	err := Write(dir, []File{{Name: "notes/board.png", Body: []byte("png")}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "board.png"))
	if err != nil || string(got) != "png" {
		t.Fatalf("%q %v", got, err)
	}
	paths := Paths("span-1", []string{"board.png"})
	if len(paths) != 1 || paths[0] != ".span/runs/span-1/handoffs/board.png" {
		t.Fatalf("%v", paths)
	}
}

func TestWriteRejectsAnEmptyName(t *testing.T) {
	if err := Write(t.TempDir(), []File{{Name: "../.."}}); err == nil {
		t.Fatal("accepted an empty name")
	}
}
