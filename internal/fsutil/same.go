package fsutil

import (
	"os"
	"path/filepath"
)

// SamePath reports whether two paths name one file or folder, however each is spelled:
// through a symlink or, on a filesystem that ignores case, with other capitals.
func SamePath(a, b string) bool {
	if filepath.Clean(a) == filepath.Clean(b) {
		return true
	}
	first, err := os.Stat(a)
	if err != nil {
		return false
	}
	second, err := os.Stat(b)
	return err == nil && os.SameFile(first, second)
}
