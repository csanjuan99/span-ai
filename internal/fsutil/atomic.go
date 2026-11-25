// Package fsutil holds the small file helpers more than one package needs.
package fsutil

import (
	"errors"
	"os"
	"path/filepath"
)

// WriteAtomic writes data to path through a temporary file in the same folder, so a reader
// never sees half a file. It creates the folder when it is missing and flushes the data to
// disk before the rename. A failure joins the error of removing the temporary file, so a
// leftover is never silent.
func WriteAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	if err := fill(f, data); err != nil {
		return discard(tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return discard(tmp, err)
	}
	return nil
}

// fill writes data to f, flushes it and closes it; the close error is kept even when the
// write failed first.
func fill(f *os.File, data []byte) error {
	_, werr := f.Write(data)
	if werr == nil {
		werr = f.Sync()
	}
	return errors.Join(werr, f.Close())
}

// discard removes the temporary file after a failure and returns the failure with whatever
// the removal reported.
func discard(tmp string, cause error) error {
	if err := os.Remove(tmp); err != nil && !os.IsNotExist(err) {
		return errors.Join(cause, err)
	}
	return cause
}
