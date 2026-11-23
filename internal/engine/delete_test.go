package engine

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"span/internal/runlock"
)

func TestDeleteRemovesTheRunRecordAndNothingElse(t *testing.T) {
	opt, _, _, _, _, _ := baseOpt(t)
	id, err := Open(opt)
	if err != nil {
		t.Fatal(err)
	}
	other, err := Open(opt)
	if err != nil {
		t.Fatal(err)
	}
	if err := Delete(opt.Root, id); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(opt.Root, id)); !os.IsNotExist(err) {
		t.Fatalf("the run is still on disk: %v", err)
	}
	if _, err := os.Stat(filepath.Join(opt.Root, other)); err != nil {
		t.Fatalf("another run was touched: %v", err)
	}
	if _, err := os.Stat(opt.Workspace); err != nil {
		t.Fatalf("the checkout was touched: %v", err)
	}
}

func TestADrivenRunCannotBeDeleted(t *testing.T) {
	opt, _, _, _, _, _ := baseOpt(t)
	id, err := Open(opt)
	if err != nil {
		t.Fatal(err)
	}
	held, err := runlock.Acquire(runlock.Run, filepath.Join(opt.Root, id, "lock"), "run "+id)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Release()
	var busy *runlock.BusyError
	if err := Delete(opt.Root, id); !errors.As(err, &busy) {
		t.Fatalf("err = %v, want a BusyError", err)
	}
	if _, err := os.Stat(filepath.Join(opt.Root, id)); err != nil {
		t.Fatalf("a refused delete removed the run: %v", err)
	}
}

func TestDeleteRefusesWhatIsNotARun(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "keep.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"", ".", "..", "../x", "a/b", "missing", "keep.txt"} {
		if err := Delete(root, id); err == nil {
			t.Fatalf("Delete(%q) succeeded", id)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "keep.txt")); err != nil {
		t.Fatalf("a file outside any run was removed: %v", err)
	}
}
