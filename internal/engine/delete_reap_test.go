//go:build !windows

package engine

import (
	"path/filepath"
	"testing"
	"time"
)

func TestDeletingAnInterruptedRunEndsTheAgentItLeft(t *testing.T) {
	shortHalt(t)
	opt, _, _, _, _, _ := baseOpt(t)
	id, err := Open(opt)
	if err != nil {
		t.Fatal(err)
	}
	ended := startOrphan(t, filepath.Join(opt.Root, id))

	if err := Delete(opt.Root, id); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ended:
	case <-time.After(5 * time.Second):
		t.Fatal("the orphan survived the delete of its run")
	}
}
